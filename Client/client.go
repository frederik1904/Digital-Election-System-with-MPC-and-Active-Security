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
	"net/rpc"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ClientHolder struct {
	client    *resty.Client
	rpcClient *rpc.Client
	ipString  string
}

type client struct {
	ipList                []IpStruct
	clients               []*ClientHolder
	createLock            sync.RWMutex
	voteAmount, voteValue int
	ss                    *n.ActiveSecretSharing
	primeP, primeQ        *big.Int
	logger                n.Logger
	sessionId             uuid.UUID
	arth                  f.ArithmeticLogic
	lock                  sync.RWMutex
	idsSent               map[uuid.UUID]bool
}

type intermediary struct {
	enc n.EncryptedShareListMarshalFriendly
	val int
}

type IpStruct struct {
	Ip        string
	Port      string
	RPCPort   string
	client    *resty.Client
	PublicKey rsa.PublicKey
	lock      sync.RWMutex
}

type ClientInfo struct {
	Primep, Primeq string
	IpList         []IpStruct
	SessionId      uuid.UUID
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

	split := strings.Split(connectIp, ":")
	fmt.Println(split)
	c.ipList = []IpStruct{{
		Ip:        split[0],
		Port:      split[1],
		client:    nil,
		PublicKey: rsa.PublicKey{},
	}}

	c.idsSent = make(map[uuid.UUID]bool)

	c.getClientInfo()

	n.Debug(2, "Got publickeys: %v", c.ipList)
	c.arth = n.NewArithmetic(c.primeP, c.primeQ)
	c.ss = n.NewActiveSecretSharing(len(c.ipList), c.primeP.String(), c.primeQ.String())

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

	c.primeP, _ = new(big.Int).SetString(info.Primep, 10)
	c.primeQ, _ = new(big.Int).SetString(info.Primeq, 10)
	c.ipList = info.IpList
	c.sessionId = info.SessionId
}

func (c *client) getCurrentVote() {
	var secret []f.Share

	for _, v := range c.ipList {
		resp, err := c.createClient(v).R().Get(string(n.GetCurrentShare))
		n.DebugRestCall(resp, err)

		var marshalShare f.MarshalFriendlyShare
		json.Unmarshal(resp.Body(), &marshalShare)
		secret = append(secret, marshalShare.TransformToShare())
	}

	n.Debug(0, "Vote we sent is currently at: %v\n", c.voteValue)
	n.Debug(0, "Vote is currently at: %v\n", c.ss.Reconstruct(secret, 2))
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

func (c *client) createRPCClient(ip IpStruct) *rpc.Client {
	c.createLock.Lock()
	defer c.createLock.Unlock()
	exist := false
	str := fmt.Sprintf("%s:%s", ip.Ip, ip.RPCPort)
	var holder *ClientHolder

	for _, v := range c.clients {
		if strings.Compare(str, v.ipString) == 0 {
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
		c.clients = append(c.clients, &ClientHolder{
			rpcClient: client,
			ipString:  str,
		})
	}

	return client
}

func (c *client) vote(value int, shareToSend n.EncryptedShareListMarshalFriendly) {
	fmt.Println(c.sessionId.String(), shareToSend.Id, "VOTE", -1, time.Now())
	c.logger.LOG(c.sessionId.String(), shareToSend.Id.String(), "VOTE", -1, time.Now())

	cpy := make([]IpStruct, len(c.ipList))
	copy(cpy, c.ipList)
	rand.Shuffle(len(cpy), func(i, j int) {
		cpy[i], cpy[j] = cpy[j], cpy[i]
	})

	for _, v := range cpy {
		// When using RPC someone can hijack our ID, due to plain text. Reason for leaving create client.
		_, err := c.createClient(v).R().SetBody(shareToSend).Post(string(n.SendShare))
		for err != nil {
			time.Sleep(1000)
			_, err = c.createClient(v).R().SetBody(shareToSend).Post(string(n.SendShare))
		}
		//c.createRPCClient(v).Go("NetworkHTTPS.ReceiveServerShare", shareToSend, nil, nil)

	}

	n.Debug(2, "Vote has been sent to all servers with value: %d, current vote tally: %d with %d votes\n",
		value, c.voteValue, c.voteAmount)
}

func (c *client) generateShares(value int) n.EncryptedShareListMarshalFriendly {
	shares, proof, zk := c.ss.SecretGen(big.NewInt(int64(value)))

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

		hash := crypto.SHA3_512.New()
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
		Proof:     proof,
		Knowledge: zk,
	}.TransformToMarshalFriendly()
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
	fmt.Println("Write vote_multiple x or vm x to vote x times at radome (write instant after x to make votes and send them all at once")
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
			c.vote(val, c.generateShares(val))
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
			instant := false
			if len(inputArr) > 2 {
				for i := 1; i < len(inputArr); i++ {
					if inputArr[i] == "instant" || inputArr[i] == "i" {
						instant = true
					}
				}
			}
			if instant {
				var shares []intermediary
				var chans []chan bool
				var lock sync.RWMutex
				for i := 0; i < val; i++ {
					ch := make(chan bool)
					go func(cha chan bool) {
						j := rand.Intn(2)
						tmp := c.generateShares(j)
						lock.Lock()
						for c.idsSent[tmp.Id] {
							fmt.Println("Generating new id to vote")
							tmp = c.generateShares(j)
						}
						c.idsSent[tmp.Id] = true
						shares = append(shares, intermediary{tmp, i})

						lock.Unlock()
						cha <- true
					}(ch)
					chans = append(chans, ch)
				}
				for i := 0; i < len(chans); i++ {
					<-chans[i]
				}

				for i := 0; i < len(shares); i++ {
					go func(chn chan bool, shares n.EncryptedShareListMarshalFriendly) {
						//time.Sleep(time.Duration(100))
						c.vote(0, shares)
						//chn <- true
					}(chans[i], shares[i].enc)
				}

			} else {
				for i := 0; i < val; i++ {
					tmp := rand.Intn(2)
					go c.vote(tmp, c.generateShares(tmp))
				}
			}
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
