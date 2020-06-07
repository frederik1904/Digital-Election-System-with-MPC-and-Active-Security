package Client

import (
	f "../server/framework"
	n "../server/src"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	cr "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty"
	"github.com/google/uuid"
	"io"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ClientHolder struct {
	client   *resty.Client
	ipString string
}

type client struct {
	ipList     []IpStruct
	clients    []*ClientHolder
	createLock sync.RWMutex
	voteAmount, voteValue  int
	ss         *n.PassiveSecretSharing
	prime      *big.Int
	logger     n.Logger
	sessionId  uuid.UUID
	lock       sync.RWMutex
	idsSent map[uuid.UUID]bool
}

type IpStruct struct {
	Ip        string
	Port      string
	client    *resty.Client
	PublicKey rsa.PublicKey
}

type ClientInfo struct {
	Prime     string
	IpList    []IpStruct
	SessionId uuid.UUID
}

type intermediary struct {
	enc n.EncryptedShareListMarshalFriendly
	val int
}

func MakeClient(Logger n.Logger) {
	var connectIp string
	if n.DEBUG == 0 {
		connectIp = n.GetInput("What is the ip:port of the server:")
	} else {
		connectIp = fmt.Sprintf("%s:%s", n.MasterIp, n.MasterPort)
	}
	c := client{
		ipList:     nil,
		voteAmount: 0,
		voteValue:  0,
		logger:     Logger,
	}
	c.idsSent = make(map[uuid.UUID]bool)
	split := strings.Split(connectIp, ":")
	fmt.Println(split)
	c.ipList = []IpStruct{{
		Ip:        split[0],
		Port:      split[1],
		client:    nil,
		PublicKey: rsa.PublicKey{},
	}}

	c.getClientInfo()

	n.Debug(2, "Got publickeys: %v", c.ipList)

	c.ss = n.NewSecretSharing(len(c.ipList), c.prime)
	c.startRunningLoop()
}

func (c *client) getClientInfo() {
	fmt.Println(c.ipList[0])
	resp, err := c.createClient(c.ipList[0]).R().Get(string(n.ClientJoinNetwork))
	n.DebugRestCall(resp, err)

	var info ClientInfo
	err = json.Unmarshal(resp.Body(), &info)
	if err != nil {
		n.Debug(2, "Got err %v", err)
	}

	c.prime, _ = big.NewInt(0).SetString(info.Prime, 10)
	c.ipList = info.IpList
	c.sessionId = info.SessionId
}

func (c *client) getCurrentVote() {
	secret := []f.Share{}
	for _, v := range c.ipList {
		resp, err := c.createClient(v).R().Get(string(n.GetCurrentShare))
		for err != nil {
			time.Sleep(100)
			resp, err = c.createClient(v).R().Get(string(n.GetCurrentShare))
		}
		n.DebugRestCall(resp, err)

		var s f.Share
		json.Unmarshal(resp.Body(), &s)

		secret = append(secret, s)
	}

	fmt.Printf("Vote is currently at: %v\n", c.ss.Reconstruct(secret, 2))
}

func (c *client) getPublicKey(ip IpStruct) {
	resp, err := c.createClient(ip).R().Get(string(n.GetPublicKey))
	n.DebugRestCall(resp, err)

	json.Unmarshal(resp.Body(), &ip.PublicKey)
}

func (c *client) getIpLists() {
	cl := c.createClient(c.ipList[0])

	resp, err := cl.
		R().
		Get(string(n.GetAllParticipants))
	n.DebugRestCall(resp, err)

	var ips []IpStruct
	err = json.Unmarshal(resp.Body(), &ips)

	n.Debug(2, "Got ip's: %v\n", ips)
	c.ipList = ips
}

func (c *client) getPrime() {
	resp, err := c.createClient(c.ipList[0]).R().Get(string(n.GetPrime))
	n.DebugRestCall(resp, err)
	json.Unmarshal(resp.Body(), c.prime)
}

func (c *client) createClient(ip IpStruct) *resty.Client {
	c.createLock.Lock()
	defer c.createLock.Unlock()
	exist := false
	str := fmt.Sprintf("%s:%s", ip.Ip, ip.Port)
	var holder *ClientHolder
	for _, v := range c.clients {
		if v.ipString == str {
			holder = v
			exist = true
			break
		}
	}
	if exist {
		return holder.client
	}

	client := resty.New()
	client.SetTLSClientConfig(
		&tls.Config{InsecureSkipVerify: true}).
		SetHostURL(fmt.Sprintf("https://%s:%s", ip.Ip, ip.Port))

	c.clients = append(c.clients, &ClientHolder{
		client:   client,
		ipString: str,
	})

	return client
}

func (c *client) _vote(value int) {
	secrets := c.ss.SecretGen(big.NewInt(int64(value)))
	var encShares []n.EncryptedShare
	
	for i, v := range secrets {
		m, _ := json.Marshal(v)
		msg, _ := rsa.EncryptPKCS1v15(cr.Reader, &c.ipList[i].PublicKey, m)
		encShares = append(encShares, n.EncryptedShare{
			Point:      i,
			CipherText: msg,
		})
	}
	shareToSend := n.EncryptedShareList{
		Sender:    -1,
		Sign:      nil,
		Id:        secrets[0].Id,
		EncShares: encShares,
	}
	go c.logger.LOG(c.sessionId.String(), secrets[0].Id.String(), "VOTE", -1, time.Now())
	for _, v := range c.ipList {
		resp, err := c.createClient(v).R().SetBody(shareToSend).Post(string(n.SendShare))
		for err != nil {
			time.Sleep(1000)
			resp, err = c.createClient(v).R().SetBody(shareToSend).Post(string(n.SendShare))
		}
		n.DebugRestCall(resp, err)
	}

	c.voteAmount++
	c.voteValue += value

	fmt.Printf("Vote has been sent to all servers with value: %d, current vote tally: %d with %d votes\n",
		value, c.voteValue, c.voteAmount)
	fmt.Printf("Sent the shares: \n%v\n", secrets)
}

func (c *client) vote(value int, shareToSend n.EncryptedShareListMarshalFriendly) {
	fmt.Println(c.sessionId.String(), shareToSend.Id, "VOTE", -1, time.Now())
	c.logger.LOG(c.sessionId.String(), shareToSend.Id.String(), "VOTE", -1, time.Now())
	for _, v := range c.ipList {
		// When using RPC someone can hijack our ID, due to plain text. Reason for leaving create client.
		_, err := c.createClient(v).R().SetBody(shareToSend).Post(string(n.SendShare))
		for err != nil {
			time.Sleep(1000)
			_, err = c.createClient(v).R().SetBody(shareToSend).Post(string(n.SendShare))
		}
		//c.createRPCClient(v).Go("NetworkHTTPS.ReceiveServerShare", shareToSend, nil, nil)

	}

	n.Debug(0, "Vote has been sent to all servers with value: %d, current vote tally: %d with %d votes\n",
		value, c.voteValue, c.voteAmount)
}

func (c *client) generateShares(value int) (n.EncryptedShareListMarshalFriendly, []f.Share) {
	shares:= c.ss.SecretGen(big.NewInt(int64(value)))

	c.lock.Lock()
	for c.idsSent[shares[0].Id] {
		shares = c.ss.SecretGen(big.NewInt(int64(value)))
	}
	c.idsSent[shares[0].Id] = true
	c.lock.Unlock()

	var encShares []n.EncryptedShare

	for i, v := range shares {
		m, _ := json.Marshal(v.TransformForNetwork())
		key := GenerateRandomKey(32)
		ci, err := aes.NewCipher(key)
		if err != nil {
			panic(err.Error())
		}

		gcm, err := cipher.NewGCM(ci)
		nonce := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(cr.Reader, nonce); err != nil {
			panic(err.Error())
		}

		CipherText := append(nonce, gcm.Seal(nil, nonce, m, nil)...)

		hash := crypto.SHA1.New()
		enc, err := rsa.EncryptOAEP(hash, cr.Reader, &c.ipList[i].PublicKey, key, nil)
		if err != nil {
			panic(err.Error())
		}

		encShares = append(encShares, n.EncryptedShare{
			Point:             i,
			CipherText:        CipherText,
			EncryptedShareKey: enc,
		})
	}
	c.incVoteAmount(value)

	return n.EncryptedShareList{
		Id:        shares[0].Id,
		EncShares: encShares,
	}.TransformToMarshalFriendly(), shares
}

func GenerateRandomKey(bytesize int) []byte {
	checkBytesize(bytesize)
	key := make([]byte, bytesize)
	rand.Read(key)
	return key
}

func checkBytesize(bytesize int) {
	availsizes := []int{16, 24, 32, 64, 128, 256, 512}
	for _, size := range availsizes {
		if bytesize == size {
			return
		}
	}
	panic("Please provide keysize of either 16, 24, 32, or 64 bytes")
}

func (c *client) incVoteAmount(value int, negative ...bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if value == 0 || value == 1 {
		if len(negative) == 0 {
			c.voteAmount++
			c.voteValue += value
		} else {
			c.voteValue -= value
			c.voteAmount--
		}

	}
}

func (c *client) printHelp() {
	fmt.Println("Write \"vote x\" to cast a vote or (\"v x\")")
	fmt.Println("Write vote_value or vv to get current vote value")
	fmt.Println("Write ip to list all connections")
	fmt.Println("Write vote_multiple x or vm x to vote x times at radome")
	fmt.Println("Write exit to stop the client")
}

func (c *client) startRunningLoop() {
	for {
		input := n.GetInput("What would you like to do? (press h for help)")
		inputArr := strings.Split(input, " ")
		if len(inputArr) == 0 {
			fmt.Println("No input given, press h for help.")
			continue
		}
		switch strings.ToLower(inputArr[0]) {
		case "h":
			c.printHelp()
		case "vote", "v":
			if len(inputArr) < 2 {
				fmt.Println("No value for vote given.")
				continue
			}
			val, err := strconv.Atoi(inputArr[1])
			if err != nil {
				fmt.Println("Could not convert value to an integer.")
				continue
			}
			enc, _ := c.generateShares(val)
			c.vote(val, enc)
		case "vote_value", "vv":
			c.getCurrentVote()
		case "ip":
			c.printIps()
		case "vm", "vote_multiple":
			if len(inputArr) < 2 {
				fmt.Println("No value for vote given.")
				continue
			}
			val, err := strconv.Atoi(inputArr[1])
			if err != nil {
				fmt.Println("Could not convert value to an integer.")
				continue
			}
			var s [][]f.Share
			for i := 0; i < val; i++ {
				tmp := rand.Intn(2)
				enc, shares := c.generateShares(tmp)
				s = append(s, shares)
				go c.vote(tmp, enc)
			}

			startShares := s[0]
			t, _:= new(big.Int).SetString(n.Prime, 10)
			arth := n.NewSimpleArithmetic(t)

			for i, v := range s {
				if i != 0 {
					for j, x := range v {
						startShares[j] = arth.Add(startShares[j], x)
					}
				}
			}
			fmt.Println(c.ss.Reconstruct(startShares, 2))
		case "exit":
			break
		}
	}
}

func (c *client) printIps() {
	for _, v := range c.ipList {
		fmt.Printf("IP %s:%s\n", v.Ip, v.Port)
	}
}
