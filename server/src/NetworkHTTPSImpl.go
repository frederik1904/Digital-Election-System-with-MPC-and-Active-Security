package src

import (
	f "../framework"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty"
	"github.com/google/uuid"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
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
	receivedShares                                   map[uuid.UUID]*EncryptedShareSeenFrom
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
	RPCPort   string
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
	rpcClient *rpc.Client
	ipPort string
}


type ChallengeVoteStruct struct {
	Id        uuid.UUID
	SecretKey []byte
	Hash      [32]byte
	Sender    int
}

type EncryptedShareSeenFrom struct {
	sharesReceived     map[[32]byte]*SignedEncryptedShare
	signaturesSeenFrom map[int]bool
	delivered          bool
	invalidated        bool
	VerificationSend bool
	challenges         []ChallengeVoteStruct
	lock               sync.Mutex
}

type Signature struct {
	Sender int
	Sign   []byte
}

type EncryptedShareListMarshalFriendly struct {
	Id        uuid.UUID
	Proof     []string
	EncShares []EncryptedShare
}

func (e EncryptedShareList) TransformToMarshalFriendly() EncryptedShareListMarshalFriendly {
	var strings []string

	return EncryptedShareListMarshalFriendly{
		Id:        e.Id,
		Proof:     strings,
		EncShares: e.EncShares,
	}
}

func (e EncryptedShareListMarshalFriendly) TransformEncryptedShareList() EncryptedShareList {
	var proof []big.Int

	for _, v := range e.Proof {
		res, _ := new(big.Int).SetString(v, 10)
		proof = append(proof, *res)
	}

	return EncryptedShareList{
		Id:        e.Id,
		EncShares: e.EncShares,
	}
}


type EncryptedShareList struct {
	Sender    int
	Sign      []byte
	Id        uuid.UUID
	EncShares []EncryptedShare
}

type SignedEncryptedShare struct {
	Signatures      []Signature
	EncryptedShares EncryptedShareListMarshalFriendly
}


type EncryptedShare struct {
	Point      int
	CipherText []byte
	EncryptedShareKey []byte
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

	n.receivedShares = make(map[uuid.UUID]*EncryptedShareSeenFrom)
	n.receivedSecretsForValidation = make(map[uuid.UUID][]f.Share)

	//TODO: Error handling
	n.keyPair, _ = rsa.GenerateKey(rand.Reader, 2048)
	listener, p := n.createListener() // Adds local address to localPeer
	go rpc.Accept(listener)
	rpc.Register(n)
	inn := GetInput("Are you creating a network? (Y/n)")
	if strings.Compare(strings.ToLower(strings.TrimSpace(inn)), "n") == 0 {
		var masterIpPort, port string
		portValue, _ := rand.Int(rand.Reader, big.NewInt(1000))
		port = fmt.Sprintf("%d", portValue.Add(portValue, big.NewInt(5000)))

		if DEBUG == 0 {
			masterIpPort = GetInput("Please write the ip and port of a server on the network to join: (ip:port)")
		} else {
			masterIpPort = fmt.Sprintf("%s:%s", MasterIp, MasterPort)
		}

		n.ownIp = IpStruct{
			Ip:        localIp,
			Port:      port,
			RPCPort:   p,
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
			RPCPort: p,
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

func (n *NetworkHTTPS) createListener() (*net.TCPListener, string) {
	var ln *net.TCPListener

	addr, _ := net.ResolveTCPAddr("tcp", ":")
	ln, _ = net.ListenTCP("tcp", addr)
	_, port, _ := net.SplitHostPort(ln.Addr().String())

	return ln, port
}

func (n *NetworkHTTPS) RunNetwork(port string) {
	http.HandleFunc(string(JoinNetwork), n.joinNetwork)
	http.HandleFunc(string(NewParticipantInNetwork), n.newParticipantInNetwork)
	http.HandleFunc(string(SignalStartPhaseTwo), n.receiveStartPhaseTwoSignal)
	http.HandleFunc(string(GetAllParticipants), n.getAllParticipants)
	http.HandleFunc(string(GetPublicKey), n.getPublicKey)
	http.HandleFunc(string(SendShare), n.receiveClientShare)
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
	panic("implement me")
}

func (n *NetworkHTTPS) VerificationFlood(share f.Share) {
	go n.logger.LOG(n.sessionId.String(), share.Id.String(), "VerificationFlood",
		n.serverId, time.Now())
	fmt.Printf("Observer sent share: %v\n", share.Id)
	for _, v := range n.ipList {
		go n.floodSingleClient(
			v,
			ReceiveVerificationSecret,
			share,
			POST, true)
	}
}

// Server functions

// Sets the return value method to json and enables cors to be by any ip address and not only its own
func setHeader(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Content-Type", "application/json")
}

func (n *NetworkHTTPS) ReciveVerificationSecret(share f.Share, reply *bool) error {
	go n.logger.LOG(n.sessionId.String(), share.Id.String(), "reciveVerificationSecret",
		n.serverId, time.Now())

	n.receiverIdsLock.Lock()
	defer n.receiverIdsLock.Unlock()
	sharesSeenFrom, exists := n.receivedShares[share.Id]

	if exists && sharesSeenFrom.delivered {
		return nil
	}

	shares, exist := n.receivedSecretsForValidation[share.Id]

	if !exist {
		n.receivedSecretsForValidation[share.Id] = []f.Share{share}
	} else {
		for _, v := range shares {
			if v.Point == share.Point {
				exist = false
				break
			}
		}
		if exist {
			shares = append(n.receivedSecretsForValidation[share.Id], share)
			n.receivedSecretsForValidation[share.Id] = shares
			if len(shares) >= n.calculateMinimumNeededParties() {
				go n.logger.LOG(n.sessionId.String(), share.Id.String(), "DELIVER_VOTE", n.serverId, time.Now())
				n.receivedShares[share.Id].delivered = true
				var s f.Share
				for _, v := range sharesSeenFrom.sharesReceived {
					if len(v.Signatures) >= n.t {
						s, _, _ = n._DecryptVerifyShare(*v)
					}
				}

				n.observer.VerificationSecretArrived(share.Id, shares, s)
			}
			for _, v := range n.ipList {
				go n.floodSingleClient(
					v,
					ReceiveVerificationSecret,
					share,
					POST, true)
			}
		}
	}
	return nil
}

func (n *NetworkHTTPS) sendCurrentShare(w http.ResponseWriter, r *http.Request) {
	setHeader(&w)
	js, _ := json.Marshal(n.observer.GetCurrentVote())

	for i,v := range n.receivedShares {
			fmt.Println(i,v.delivered,v.invalidated, len(v.signaturesSeenFrom))
	}

	fmt.Println(len(n.receivedShares))

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

// Receives a share form the server, and checks it against the client share to verify.
func (n *NetworkHTTPS) ReceiveServerShare(serverSignedEncryptionShare SignedEncryptedShare, reply *bool) error {
	n.receiverIdsLock.Lock()
	shareId := serverSignedEncryptionShare.EncryptedShares.Id
	encryptedShareSeenFrom, shareSeen := n.receivedShares[shareId]

	if shareSeen {
		encryptedShareSeenFrom.lock.Lock()
		n.receiverIdsLock.Unlock()

		if encryptedShareSeenFrom.invalidated {
			go n.logger.LOG(n.sessionId.String(), shareId.String(), "ReceiveServerShare(INVALIDATED)", n.serverId, time.Now())
			encryptedShareSeenFrom.lock.Unlock()
			Debug(2, "The share with id: %v was already invalidated\n")
			return nil
		}

		if encryptedShareSeenFrom.delivered  || encryptedShareSeenFrom.VerificationSend {
			go n.logger.LOG(n.sessionId.String(), shareId.String(), "ReceiveServerShare(GOTTAGOFAST)", n.serverId, time.Now())
			encryptedShareSeenFrom.lock.Unlock()
			Debug(2, "The share with id: %v was already delivered\n")
			return nil
		}
	}
	go n.logger.LOG(n.sessionId.String(), shareId.String(), "ReceiveServerShare", n.serverId, time.Now())

	// Create hash for comparison with other shares
	shareHash := createShareHash(serverSignedEncryptionShare)

	// Check if share has been seen before

	if !shareSeen {
		Debug(2, "First time seeing id: %v", serverSignedEncryptionShare.EncryptedShares.Id)
		// Add new share seen from with share server sent added.
		encryptedShareSeenFrom = &EncryptedShareSeenFrom{
			sharesReceived:     map[[32]byte]*SignedEncryptedShare{},
			signaturesSeenFrom: map[int]bool{},
		}
		n.receivedShares[shareId] = encryptedShareSeenFrom
		encryptedShareSeenFrom.lock.Lock()
		n.receiverIdsLock.Unlock()
	}

	signEncShare, found := encryptedShareSeenFrom.sharesReceived[shareHash]

	if !found {
		Debug(2, "First time seeing hash: %v", shareHash)
		signEncShare = &SignedEncryptedShare{
			Signatures:      []Signature{},
			EncryptedShares: serverSignedEncryptionShare.EncryptedShares,
		}
	}

	newSignatures := findNewSignatures(serverSignedEncryptionShare.Signatures, signEncShare.Signatures, encryptedShareSeenFrom.signaturesSeenFrom)

	// Create variable to store new found signatures
	var newVerifiedSignatures bool = false
	Debug(2, "New signatures from: ")
	for _, v := range newSignatures {
		Debug(2, "%v - ", v.Sender)
		if encryptedShareSeenFrom.signaturesSeenFrom[v.Sender] {
			Debug(2, "Already verified signature from sender %v\n", v.Sender)
			continue
		}
		verifyError := rsa.VerifyPKCS1v15(&n.ipList[v.Sender].PublicKey, crypto.SHA256, shareHash[:], v.Sign[:])
		if verifyError != nil {
			Debug(0, "New server share received with unverifiable signature\n%v\n", verifyError)
		} else {
			Debug(2, "Verified share from sender\n")
			signEncShare.Signatures = append(signEncShare.Signatures, v)
			newVerifiedSignatures = true
			encryptedShareSeenFrom.signaturesSeenFrom[v.Sender] = true
		}
	}
	encryptedShareSeenFrom.sharesReceived[shareHash] = signEncShare

	n._CheckSignatures(encryptedShareSeenFrom)
	Debug(2, "Got new signatures: %v\n", newVerifiedSignatures)
	encryptedShareSeenFrom.lock.Unlock()
	if newVerifiedSignatures {
		for _, v := range n.ipList {
			n.floodSingleClient(v, ReceiveServerShare, signEncShare, "", true)
		}
	}

	return nil
}

func (n *NetworkHTTPS) _CheckSignatures(encryptedShareSeenFrom *EncryptedShareSeenFrom) {
	if encryptedShareSeenFrom.delivered {
		return
	}
	for _, v := range encryptedShareSeenFrom.sharesReceived {
		if len(v.Signatures) >= len(n.ipList)-n.t {
			// Verify received share. If fail, bail out and return error on RPC
			share, err, _ := n._DecryptVerifyShare(*v)
			if err != nil {
				if len(v.Signatures) >= len(n.ipList) - n.t {
					encryptedShareSeenFrom.invalidated = true
				}
				encryptedShareSeenFrom.lock.Unlock()
				Debug(2, "Could not verify share from server (ReceiveServerShare) with error: %v\n", err)
				return //errors.New(fmt.Sprintf("Could not very share received from server (ReceiveServerShare) with error: %v", err))
			}

			Debug(0, "I delivered, %v\n", share.Id)
			Debug(2, "EcnSharesSeenFrom: %v\n", encryptedShareSeenFrom.sharesReceived)
			if !encryptedShareSeenFrom.invalidated {
				encryptedShareSeenFrom.VerificationSend = true
				n.observer.NewShareArrived(share)
			}
			break
		}
	}
}

func (n *NetworkHTTPS) receiveClientShare(w http.ResponseWriter, r *http.Request) {
	setHeader(&w)

	var encShareList EncryptedShareListMarshalFriendly
	err := json.NewDecoder(r.Body).Decode(&encShareList)
	if err != nil {
		panic(err)
	}

	clientSignedEncryptionShare := SignedEncryptedShare{
		Signatures:      []Signature{},
		EncryptedShares: encShareList,
	}

	shareId := clientSignedEncryptionShare.EncryptedShares.Id
	n.receiverIdsLock.Lock()
	encryptedShareSeenFrom, found := n.receivedShares[shareId]
	if found {
		encryptedShareSeenFrom.lock.Lock()
		n.receiverIdsLock.Unlock()

		if encryptedShareSeenFrom.invalidated {
			go n.logger.LOG(n.sessionId.String(), shareId.String(), "receiveClientShare(INVALIDATED)", n.serverId, time.Now())
			encryptedShareSeenFrom.lock.Unlock()
			Debug(2, "The share with id: %v was already invalidated\n")
			return
		}

		if encryptedShareSeenFrom.delivered || encryptedShareSeenFrom.VerificationSend {
			encryptedShareSeenFrom.lock.Unlock()
			go n.logger.LOG(n.sessionId.String(), shareId.String(), "receiveClientShare(GOTTAGOFAST)", n.serverId, time.Now())
			Debug(2, "The share with id: %v was already delivered\n")
			return
		}
	}
	go n.logger.LOG(n.sessionId.String(), shareId.String(), "receiveClientShare", n.serverId, time.Now())

	// Client share hashed for comparison
	shareHash := createShareHash(clientSignedEncryptionShare)

	// Create own signature
	signature, _ := rsa.SignPKCS1v15(rand.Reader, n.keyPair, crypto.SHA256, shareHash[:])
	ownSign := Signature{Sender: n.serverId, Sign: signature}

	clientSignedEncryptionShare.Signatures = append(clientSignedEncryptionShare.Signatures, ownSign)

	// Check if share with Id has been seen from servers and add signatures from correct server share hash

	if found {
		Share, exist := encryptedShareSeenFrom.sharesReceived[shareHash]
		if exist {
			Share.Signatures = append(Share.Signatures, ownSign)
			clientSignedEncryptionShare = *Share
		} else {
			encryptedShareSeenFrom.sharesReceived[shareHash] = &clientSignedEncryptionShare
		}
	} else {
		// Else, add new share seen from with share client sent.
		encryptedShareSeenFrom = &EncryptedShareSeenFrom{
			sharesReceived:     map[[32]byte]*SignedEncryptedShare{shareHash: &clientSignedEncryptionShare},
			signaturesSeenFrom: map[int]bool{n.serverId: true},
		}
		n.receivedShares[shareId] = encryptedShareSeenFrom
		encryptedShareSeenFrom.lock.Lock()
		n.receiverIdsLock.Unlock()
	}

	// Verify received share. If fail, bail out
	_, err, _ = n._DecryptVerifyShare(clientSignedEncryptionShare)
	if err != nil {
		if !found {
			encryptedShareSeenFrom.signaturesSeenFrom[n.serverId] = false
		}

		encryptedShareSeenFrom.lock.Unlock()
		Debug(2, "Could not verify share received from client (receiveClientShare)\n")
		w.WriteHeader(400)
		return
	}

	// Set our server id to have added a signature
	encryptedShareSeenFrom.signaturesSeenFrom[n.serverId] = true
	encryptedShareSeenFrom.sharesReceived[shareHash] = &clientSignedEncryptionShare
	n._CheckSignatures(encryptedShareSeenFrom)
	encryptedShareSeenFrom.lock.Unlock()

	for _, v := range n.ipList {
		go n.floodSingleClient(v, ReceiveServerShare, clientSignedEncryptionShare, "", true)
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
func (n *NetworkHTTPS) floodSingleClient(ip IpStruct, endPoint ApiEndpoints, payload interface{}, ty RequestType, rpc ...bool) ([]byte, error) {
	if len(rpc) == 0 {
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

	n.createRPCClient(ip).Go(string(endPoint), payload, nil, nil)

	return []byte{}, nil
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

func (n *NetworkHTTPS) decryptShare(encShare EncryptedShare) (f.Share, []byte) {
	key, err := rsa.DecryptOAEP(crypto.SHA1.New(), rand.Reader, n.keyPair, encShare.EncryptedShareKey, nil)
	if err != nil {
		panic(err)
	}
	gcm := _GenerateGCM(key)
	nonceSize := gcm.NonceSize()
	if len(encShare.CipherText) < nonceSize {
		panic("NonceSize is larger than ciphertext")
	}
	nonce, CipherText := encShare.CipherText[:nonceSize], encShare.CipherText[nonceSize:]

	msg, err := gcm.Open(nil, nonce, CipherText, nil)
	if err != nil {
		panic(err)
	}
	var MShare f.MarshalFriendlyShare

	if err := json.Unmarshal(msg, &MShare); err != nil {
		Debug(2, "ERROR: %v", err)
	}
	res := MShare.TransformToShare()

	return res, key
}

// Calculates the minimum needed votes before we can proceed with verification and sending secrets
func (n *NetworkHTTPS) calculateMinimumNeededParties() int {
	return 2*n.t + 1
}

func (n *NetworkHTTPS) createRPCClient(ip IpStruct) *rpc.Client {
	n.ipListLock.Lock()
	defer n.ipListLock.Unlock()
	exist := false
	str := fmt.Sprintf("%s:%s", ip.Ip, ip.RPCPort)
	var holder *ClientHolder

	for _, v := range n.clients {
		if strings.Compare(str, v.ipPort) == 0 {
			exist = true
			holder = v
		}
	}
	if exist && holder.rpcClient != nil {
		return holder.rpcClient
	}

	client, _ := rpc.Dial("tcp", str)
	if exist {
		holder.rpcClient = client
	} else {
		n.clients = append(n.clients, &ClientHolder{
			rpcClient: client,
			ipPort:    str,
		})
	}

	return client
}

func createShareHash(clientSignedEncryptionShare SignedEncryptedShare) [32]byte {
	clientMarshal, err := json.Marshal(clientSignedEncryptionShare.EncryptedShares)
	if err != nil {
		panic(fmt.Sprintf("NetworkHTTPSImpl._computeReceived marshal (clientSignedEncryptionShare shareSeen=true) error: %v", err))
	}
	clientHash := sha256.Sum256(clientMarshal)
	return clientHash
}

func (n *NetworkHTTPS) _DecryptVerifyShare(s SignedEncryptedShare) (f.Share, error, []byte) {
	var share f.Share
	var key []byte
	encShareList := s.EncryptedShares.TransformEncryptedShareList()

	for _, v := range encShareList.EncShares {
		if v.Point == n.serverId {
			share, _ = n.decryptShare(v)
			break
		}
	}

	return share, nil, key
}

func findNewSignatures(receivedSet, localSet []Signature, seenMap map[int]bool) []Signature {
	var signs []Signature
	for _, v := range receivedSet {
		//If we have already verified a signature from the given sender we wont look at any other signatures from that server
		if seenMap[v.Sender] {
			continue
		}
		found := false
		for _, j := range localSet {
			if v.Sender == j.Sender {
				found = true
				break
			}
		}

		if !found {
			signs = append(signs, v)
		}
	}
	return signs
}

func _GenerateGCM(key []byte) cipher.AEAD {
	ci, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}

	gcm, err := cipher.NewGCM(ci)

	return gcm
}