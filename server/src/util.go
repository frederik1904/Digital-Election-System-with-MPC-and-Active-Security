package src

import (
	f "../framework"
	"bufio"
	"crypto/rand"
	"fmt"
	"github.com/go-resty/resty"
	b "math/big"
	"net"
	"os"
	"strings"
	"sync"
)

const DEBUG = 0
const LOG = false

func Debug(level int, format string, a ...interface{}) {
	if DEBUG >= level {
		var tmp []interface{}
		for _,v := range a{
			switch v.(type) {
			case f.Share:
				x := v.(f.Share)
				tmp = append(tmp, fmt.Sprintf("S: %v\t T: %v \t ID: %v \t Point: %v \n ", x.S.String(), x.T.String(), x.Id, x.Point))
			default:
				tmp = append(tmp, v)
			}
		}

		fmt.Printf(format, tmp...)
	}
}

func GetLocalIp() string {
	conn, _ := net.Dial("udp", "8.8.8.8:80")
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func PrintShares(shares []f.Share,) string {
	strS := "S: ["
	for _, share := range shares {
		strS += fmt.Sprintf("%v, ", &share.S)
	}
	strS += "]"
	strT := "T: ["
	for _, share := range shares {
		strT += fmt.Sprintf("%v, ", &share.T)
	}
	strT += "]"
/*
	strProof := "Proof: ["
	for _, value := range shares[0].Proof {
		strProof += fmt.Sprintf("%v, ", &value)
	}
	strProof += "]"
*/
	return strS + "\n" + strT + "\n"
}

func DebugRestCall(resp *resty.Response, err error) {
	if DEBUG >= 2 {
		// Explore response object
		fmt.Println("Response Info:")
		fmt.Println("Error      :", err)
		fmt.Println("Status Code:", resp.StatusCode())
		fmt.Println("Status     :", resp.Status())
		fmt.Println("Time       :", resp.Time())
		fmt.Println("Received At:", resp.ReceivedAt())
		fmt.Println("Body       :\n", resp)
		fmt.Println()

		/*
			// Explore trace info
			fmt.Println("Request Trace Info:")
			ti := resp.Request.TraceInfo()
			fmt.Println("DNSLookup    :", ti.DNSLookup)
			fmt.Println("ConnTime     :", ti.ConnTime)
			fmt.Println("TLSHandshake :", ti.TLSHandshake)
			fmt.Println("ServerTime   :", ti.ServerTime)
			fmt.Println("ResponseTime :", ti.ResponseTime)
			fmt.Println("TotalTime    :", ti.TotalTime)
			fmt.Println("IsConnReused :", ti.IsConnReused)
			fmt.Println("IsConnWasIdle:", ti.IsConnWasIdle)
			fmt.Println("ConnIdleTime :", ti.ConnIdleTime)
		*/
	}
}

func GetInput(text string) string {
	fmt.Println(text)
	t, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSuffix(t, "\n")
}

func CalculateT(parties int) int {
	if parties < 3 {
		panic(fmt.Sprintf("Too few parties expected 3 or more but got %d", parties))
	}
	// t < n/3
	return (parties - 1) / 3
}

func MaxPrimeFactors(_n *b.Int) *b.Int {
	max := b.NewInt(-1)
	n := b.NewInt(0)
	n = n.Sub(_n, n)
	// Get the number of 2s that divide n
	for b.NewInt(0).Cmp(new(b.Int).Mod(n, b.NewInt(2))) == 0 {
		max = b.NewInt(2)
		n.Div(n, b.NewInt(2))
	}

	// n must be odd at this point. so we can skip one element
	// (note i = i + 2)
	for i := b.NewInt(3); i.Cmp(new(b.Int).Sqrt(n)) <= 0 ; i.Add(i, b.NewInt(2)) {
		// while i divides n, append i and divide n
		for i.Cmp(new(b.Int).Mod(n, b.NewInt(2))) == 0 {
			max = i
			n.Div(n, i)
		}
	}

	// This condition is to handle the case when n is a prime number
	// greater than 2
	if n.Cmp(b.NewInt(2)) > 0 {
		max = n
	}

	return max
}

type primeFactor struct {
	found bool
	lock sync.RWMutex
	p,q *b.Int
}

var pf primeFactor

func FrederiksMaxPrimeFactor(bitsize, threads int) (*b.Int, *b.Int) {
	var chans []chan bool
	pf = primeFactor{
		found: false,
		lock:  sync.RWMutex{},
		p:     nil,
		q:     nil,
	}
	for i := 0; i < threads; i++ {
		chans = append(chans, make(chan bool))
		go _frederiksMaxPrimeFactor(chans[i], bitsize)
	}
	for i := 0; i < threads; i++ {
		<- chans[i]
	}
	return pf.p, pf.q
}

func _frederiksMaxPrimeFactor(c chan bool, bitsize int) {
	p, _ := rand.Prime(rand.Reader, bitsize)
	q := new(b.Int)
	count := 1
	for {
		pf.lock.RLock()
		if !q.Rsh(q.Sub(p, b.NewInt(1)), 1).ProbablyPrime(20) && !pf.found {
			pf.lock.RUnlock()
			p, _ = rand.Prime(rand.Reader, bitsize)
			count++
		} else {
			pf.lock.RUnlock()
			break
		}
	}
	if q.Rsh(q.Sub(p, b.NewInt(1)), 1).ProbablyPrime(20) {
		pf.lock.Lock()
		pf.found, pf.p, pf.q = true, p, q
		pf.lock.Unlock()
	}
	c <- true
}
