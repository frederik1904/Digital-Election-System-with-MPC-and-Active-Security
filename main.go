package main

import (
	"./Client"
	"./server/src"
	"fmt"
	"strings"
)

func main() {

	log := src.MakeLogger()

	input := src.GetInput("Want to start a Server og client? (S/c):")
	if strings.Compare(input, "c") == 0 {
		Client.MakeClient(log)
	} else {
		mpcController := src.NewActiveMPCController(log)
		fmt.Println(mpcController.GetState())
	}

	for {
		input = src.GetInput("Write exit to close the server")
		if strings.Compare(input, "exit") == 0 {
			break
		}
	}
	/*
		inn, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.Compare(strings.TrimSpace(inn), "c") == 0 {
			n.GetVote()
		} else {

		}
	*/
}
