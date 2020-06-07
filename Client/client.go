package Client

import (
	f "../server/framework"
	n "../server/src"
	cr "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty"
	"github.com/google/uuid"
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

func (c *client) vote(value int) {
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
	go c.logger.LOG(c.sessionId.String(), secrets[0].Id.String(), "VOTE", time.Now(), -1)
	for _, v := range c.ipList {
		resp, err := c.createClient(v).R().SetBody(shareToSend).Post(string(n.SendShare))
		for err != nil {
			time.Sleep(100)
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
			c.vote(val)
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
			for i := 0; i < val; i++ {
				go c.vote(rand.Intn(2))
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
