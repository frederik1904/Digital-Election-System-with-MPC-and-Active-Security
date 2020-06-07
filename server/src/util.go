package src

import (
	"bufio"
	"fmt"
	externalip "github.com/glendc/go-external-ip"
	"github.com/go-resty/resty"
	"os"
	"strings"
)

const DEBUG = 0
const LOG = false

func Debug(level int, format string, a ...interface{}) {
	if DEBUG >= level {
		fmt.Printf(format, a...)
	}
}

func GetLocalIp() string {
	if DEBUG > 0 {
		return "127.0.0.1"
	}
	consensus := externalip.DefaultConsensus(nil, nil)
	ip, err := consensus.ExternalIP()
	if err != nil {
		panic("I CANNOT FIND MY OWN IP (GetLocalIp)")
	}
	return ip.String()
}

func DebugRestCall(resp *resty.Response, err error) {
	if DEBUG > 2 {
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

	// t < n/2
	return (parties - 1) / 2
}

func fuckMagnus(s string, i ...interface{}) {
	Debug(1, s, i...)
}