package src

import (
	f "../framework"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty"
	"github.com/google/uuid"
	"log"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type NetworkHTTPS struct {
	ipList                                           []IpStruct
	clients                                          []*ClientHolder
	ownIp                                            IpStruct
	ipListLock                                       sync.RWMutex
	stateWait                                        chan phaseTwoStartStruct
	observer                                         f.NetworkObserver
	receivedIds                                      map[uuid.UUID]bool
	receivedSecretsForValidation                     map[uuid.UUID][]f.Share
	receiverIdsLock, receiverSecretForValidationLock sync.RWMutex
	keyPair                                          *rsa.PrivateKey
	t, serverId                                      int
	prime                                            *big.Int
	sessionId                                        uuid.UUID
	logger                                           Logger
}

type IpStruct struct {
	Ip        string
	Port      string
	client    *resty.Client
	PublicKey rsa.PublicKey
}

type secretToFlood struct {
	Id         uuid.UUID
	Ip         IpStruct
	Point      int64
	SecretHash [32]byte
	Sign       []byte
}

type phaseTwoStartStruct struct {
	Parties   int
	Prime     string
	Secret    f.MarshalFriendlyShare
	SessionId uuid.UUID
}

type ClientInfo struct {
	Prime     string
	IpList    []IpStruct
	SessionId uuid.UUID
}

type ClientHolder struct {
	client *resty.Client
	ipPort string
}

type EncryptedShareList struct {
	Sender    int
	Sign      []byte
	Id        uuid.UUID
	EncShares []EncryptedShare
}

type EncryptedShare struct {
	Point      int
	CipherText []byte
}

type EncShareListHash struct {
	Id        uuid.UUID
	EncShares []EncryptedShare
}

func (e EncShareListHash) marshalEncShare() []byte {
	res, _ := json.Marshal(e)
	return res
}

const certificate_path = "server/src/"

// Starter functions
func (n *NetworkHTTPS) StartNetwork(log interface{}) (int, f.Share, *big.Int, uuid.UUID, int64) {
	localIp := GetLocalIp()
	var phaseTwoStruct phaseTwoStartStruct
	var secrets []f.Share
	n.stateWait = make(chan phaseTwoStartStruct)
	n.logger = log.(Logger)

	n.receivedIds = make(map[uuid.UUID]bool)
	n.receivedSecretsForValidation = make(map[uuid.UUID][]f.Share)

	//TODO: Error handling
	n.keyPair, _ = rsa.GenerateKey(rand.Reader, 2048)
	inn := GetInput("Are you creating a network? (Y/n)")
	if strings.Compare(strings.ToLower(strings.TrimSpace(inn)), "n") == 0 {
		var masterIpPort, port string
		if DEBUG == 0 {
			masterIpPort = GetInput("Please write the ip and port of a server on the network to join: (ip:port)")
			port = GetInput("Please write the port you want to use:")

		} else {
			masterIpPort = fmt.Sprintf("%s:%s", MasterIp, MasterPort)
			portValue, _ := rand.Int(rand.Reader, big.NewInt(1000))
			port = fmt.Sprintf("%d", portValue.Add(portValue, big.NewInt(5000)))
		}

		n.ownIp = IpStruct{
			Ip:        localIp,
			Port:      port,
			PublicKey: n.keyPair.PublicKey,
		}

		n.ipList = append(n.ipList, n.ownIp)
		go n.RunNetwork(port)
		ip := strings.Split(strings.TrimSpace(masterIpPort), ":")

		n.ParticipateInNetwork(IpStruct{
			Ip:     ip[0],
			Port:   ip[1],
			client: nil,
		})

	} else {
		fmt.Printf("Opening network on %s:%s", localIp, MasterPort)

		n.ownIp = IpStruct{
			Ip:        localIp,
			Port:      MasterPort,
			PublicKey: n.keyPair.PublicKey,
		}

		n.ipList = append(n.ipList, n.ownIp)
		go n.RunNetwork(MasterPort)
		GetInput("\nPress any key to continue to phase 2.")

		sessionId := uuid.New()
		n.sessionId = sessionId

		prime, _ := big.NewInt(0).SetString(Prime, 10)

		Debug(1, "Prime was initialized to: %d\n", prime)

		ss := NewSecretSharing(len(n.ipList), prime)
		secrets = ss.SecretGen(big.NewInt(-1))
		n.sortIpStruct()
		n.SendStartPhaseTwoSignal(len(n.ipList), prime, secrets, sessionId)
	}
	phaseTwoStruct = <-n.stateWait
	fmt.Print(phaseTwoStruct)
	n.t = CalculateT(len(n.ipList))
	n.prime, _ = big.NewInt(0).SetString(phaseTwoStruct.Prime, 10)
	n.sessionId = phaseTwoStruct.SessionId

	serverId := -1

	for i, v := range n.ipList {
		if v.Port == n.ownIp.Port && v.Ip == n.ownIp.Ip {
			serverId = i
		}
	}

	if serverId == -1 {
		panic("Could not find our self in the ipList")
	}
	n.serverId = serverId

	return len(n.ipList), phaseTwoStruct.Secret.TransformToShare(), n.prime, n.sessionId, int64(n.serverId)
}

func (n *NetworkHTTPS) RunNetwork(port string) {
	http.HandleFunc(string(JoinNetwork), n.joinNetwork)
	http.HandleFunc(string(NewParticipantInNetwork), n.newParticipantInNetwork)
	http.HandleFunc(string(SignalStartPhaseTwo), n.receiveStartPhaseTwoSignal)
	http.HandleFunc(string(GetAllParticipants), n.getAllParticipants)
	http.HandleFunc(string(GetPublicKey), n.getPublicKey)
	http.HandleFunc(string(SendShare), n.receiveShare)
	http.HandleFunc(string(ClientJoinNetwork), n.clientJoinNetwork)
	http.HandleFunc(string(GetCurrentShare), n.sendCurrentShare)

	err := http.ListenAndServeTLS(
		fmt.Sprintf(":%s", port),
		certificate_path+"server.crt", certificate_path+"server.key",
		nil)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// Observer pattern
func (n *NetworkHTTPS) AddObserver(networkObserver f.NetworkObserver) {
	n.observer = networkObserver
}

func (n *NetworkHTTPS) RemoveObserver(networkObserver f.NetworkObserver) {
	n.observer = nil
}

func (n *NetworkHTTPS) Flood(share f.Share) {
	panic("Implement me")
}

func (n *NetworkHTTPS) FloodSecretToShare(share secretToFlood) {
	go n.logger.LOG(n.sessionId.String(), share.Id.String(), "FloodSecretToShare",
		time.Now(), n.serverId)

	for _, v := range n.ipList {
		go n.floodSingleClient(
			v,
			FloodSecret,
			share,
			POST)
	}
}

func (n *NetworkHTTPS) VerificationFlood(share f.Share) {
	fmt.Printf("Observer sent share: %v\n", share)
	for _, v := range n.ipList {
		go n.floodSingleClient(
			v,
			ReceiveVerificationSecret,
			share,
			POST)
	}
}

// Server functions

// Sets the return value method to json and enables cors to be by any ip address and not only its own
func setHeader(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Content-Type", "application/json")
}

func (n *NetworkHTTPS) sendCurrentShare(w http.ResponseWriter, r *http.Request) {
	setHeader(&w)
	js, _ := json.Marshal(n.observer.GetCurrentVote())
	w.Write(js)
}

func (n *NetworkHTTPS) clientJoinNetwork(w http.ResponseWriter, r *http.Request) {
	setHeader(&w)
	js, _ := json.Marshal(ClientInfo{
		Prime:     n.prime.String(),
		IpList:    n.ipList,
		SessionId: n.sessionId,
	})
	w.Write(js)
}

// API call that floods new participants in the network so that all people knows the ip of each others,
// if the ip is not know it is sent to all the other known participants, else it is ignored.
func (n *NetworkHTTPS) newParticipantInNetwork(w http.ResponseWriter, r *http.Request) {
	exists := false
	var ip IpStruct
	err := json.NewDecoder(r.Body).Decode(&ip)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	setHeader(&w)

	for _, v := range n.ipList {
		if ip.Ip == v.Ip && ip.Port == v.Port {
			exists = true
			break
		}
	}

	if !exists {
		Debug(3, "The participant with ip %s:%s did not exist, adding it and flood commense\n", ip.Ip, ip.Port)
		n.ipList = append(n.ipList, ip)
		n.FloodNewParticipantInNetwork(ip)
	} else {
		Debug(3, "We already have the participant's ip (%s:%s) in our list\n", ip.Ip, ip.Port)
	}

}

// A server endpoint to call when you want to join the network as a server, then calls all other servers in the network
// and tells them that it exists.
func (n *NetworkHTTPS) joinNetwork(w http.ResponseWriter, r *http.Request) {
	//Check body input
	var ip IpStruct
	err := json.NewDecoder(r.Body).Decode(&ip)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	Debug(2, "Got connection from %s:%s\n", ip.Ip, ip.Port)

	//Process response
	setHeader(&w)
	js, err := json.Marshal(n.ipList)
	w.Write(js)

	//Update local state
	n.ipList = append(n.ipList, ip)
	Debug(3, "New IpList is %v\n", n.ipList)

	go n.FloodNewParticipantInNetwork(ip)
}

// A server endpoint to start on phase two, that being starting to receive votes, this is also the function that
// returns (by the stateWait channel) the prime, participant amount and the minus one secret
func (n *NetworkHTTPS) receiveStartPhaseTwoSignal(w http.ResponseWriter, r *http.Request) {
	var phaseTwo phaseTwoStartStruct
	err := json.NewDecoder(r.Body).Decode(&phaseTwo)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	n.sortIpStruct()
	n.stateWait <- phaseTwo
}

func (n *NetworkHTTPS) receiveShare(w http.ResponseWriter, r *http.Request) {
	var encShares EncryptedShareList
	var share f.Share

	// Decrpypt the specific share that we have the private key for.
	json.NewDecoder(r.Body).Decode(&encShares)
	for _, v := range encShares.EncShares {
		if v.Point == n.serverId {
			share = n.decryptShare(v)
		}
	}

	// Take locks because saving sharesSeenFrom
	n.receiverIdsLock.Lock()
	defer n.receiverIdsLock.Unlock()
	_, exist := n.receivedIds[share.Id]
	if !exist {
		// Log info to database
		go n.logger.LOG(n.sessionId.String(), share.Id.String(), "DELIVER_VOTE",
			time.Now(), n.serverId)
		n.receivedIds[share.Id] = true
		n.observer.NewSecretArrived(share)
		for _, v := range n.ipList {
			go n.floodSingleClient(v, SendShare, encShares, POST)
		}
	} else {
		go n.logger.LOG(n.sessionId.String(), share.Id.String(), "RECEIVE_SHARE",
			time.Now(), n.serverId)
	}
}

func (n *NetworkHTTPS) getAllParticipants(w http.ResponseWriter, r *http.Request) {
	setHeader(&w)
	js, err := json.Marshal(n.ipList)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(js)
}

func (n *NetworkHTTPS) getPublicKey(w http.ResponseWriter, r *http.Request) {
	setHeader(&w)
	js, err := json.Marshal(n.keyPair.PublicKey)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(js)
}

// Client functions
func (n *NetworkHTTPS) ParticipateInNetwork(ip IpStruct) {
	Debug(1, "Joining network %s:%s\n", ip.Ip, ip.Port)
	resp, err := n.floodSingleClient(ip, JoinNetwork, n.ipList[0], POST)

	var ips []IpStruct
	err = json.Unmarshal(resp, &ips)

	if err != nil {
		panic("Could not unmarshal the list of ip addresses")
	}
	n.ipList = append(n.ipList, ips...)

	for _, v := range n.ipList {
		go n.GetPublicKey(v)
	}
}

func (n *NetworkHTTPS) FloodNewParticipantInNetwork(ip IpStruct) {
	Debug(3, "‰v, %v\n", NewParticipantInNetwork, n.ipList)
	for _, v := range n.ipList {
		go n.floodSingleClient(v, NewParticipantInNetwork, ip, POST)
	}
}

func (n *NetworkHTTPS) SendStartPhaseTwoSignal(parties int, prime *big.Int, secrets []f.Share, id uuid.UUID) {
	for i, v := range n.ipList {
		go n.floodSingleClient(
			v,
			SignalStartPhaseTwo,
			phaseTwoStartStruct{
				Parties:   parties,
				Prime:     prime.String(),
				Secret:    secrets[i].TransformForNetwork(),
				SessionId: id,
			},
			POST)
	}
}

//TODO: SEND PUBLIC KEY HERE WHEN GETTING IP, MAKE IT WITH A HANDSHAKE.
func (n *NetworkHTTPS) GetPublicKey(ip IpStruct) {
	resp, err := n.createClient(ip).
		R().
		Get(string(GetPublicKey))

	json.Unmarshal(resp.Body(), &ip.PublicKey)

	DebugRestCall(resp, err)
}

func (n *NetworkHTTPS) ChangeNetworkState(state f.NetworkStates) {
	panic("implement me")
}

//UTIL
func (n *NetworkHTTPS) floodSingleClient(ip IpStruct, endPoint ApiEndpoints, payload interface{}, ty RequestType) ([]byte, error) {
	var resp *resty.Response
	var err error
	request := n.createClient(ip).R()

	switch ty {
	case GET:
		resp, err = request.
			Get(string(endPoint))
	case POST:
		resp, err = request.
			SetBody(payload).
			Post(string(endPoint))
	}
	DebugRestCall(resp, err)
	return resp.Body(), err
}

func (n *NetworkHTTPS) createClient(ip IpStruct) *resty.Client {
	n.ipListLock.Lock()
	defer n.ipListLock.Unlock()
	exist := false
	str := fmt.Sprintf("%s:%s", ip.Ip, ip.Port)
	var holder *ClientHolder

	for _, v := range n.clients {
		if strings.Compare(str, v.ipPort) == 0 {
			exist = true
			holder = v
		}
	}
	if exist {
		return holder.client
	}

	client := resty.New()
	client.SetTLSClientConfig(
		&tls.Config{InsecureSkipVerify: true}).
		SetHostURL(fmt.Sprintf("https://%s:%s", ip.Ip, ip.Port))

	n.clients = append(n.clients, &ClientHolder{
		client: client,
		ipPort: str,
	})

	return client
}

func (n *NetworkHTTPS) sortIpStruct() {
	sort.Slice(n.ipList, func(i, j int) bool {
		a, b := n.ipList[i], n.ipList[j]
		if a.Ip == b.Ip {
			return a.Port <= b.Port
		}
		return a.Ip < b.Ip
	})
}

func (n *NetworkHTTPS) decryptShare(encShare EncryptedShare) f.Share {
	msg, err := rsa.DecryptPKCS1v15(rand.Reader, n.keyPair, encShare.CipherText)
	if err != nil {
		panic(fmt.Sprintf("Could not decrypt the share with point %v, own point: %v", encShare.Point, n.serverId))
	}
	var share f.Share
	if err := json.Unmarshal(msg, &share); err != nil {
		fuckMagnus("ERROR: %v", err)
	}

	return share
}

// Calculates the minimum needed votes before we can proceed with verification and sending secrets
func (n *NetworkHTTPS) calculateMinimumNeededParties() int {
	return 2*n.t + 1
}
