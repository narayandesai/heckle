package main

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	//"bytes"
	hclient "flunky/client"
	iface "flunky/interfaces"
	fnet "flunky/net"
	"fmt"
	"strconv"
	"time"
)

var help bool
var server, image, fileDir string
var allocationList []string
var numNodes, timeIncrease int
var allocationNumber, freeAlloc uint64
var hallocC fnet.Communication
var bs *fnet.BuildServer

func init() {
	var error error
	flag.BoolVar(&help, "h", false, "Print usage of command.")
	flag.IntVar(&numNodes, "n", 0, "Request an arbitrary number of nodes.")
	flag.IntVar(&timeIncrease, "t", 0, "Increase current allocation by this many hours.")
	flag.StringVar(&image, "i", "ubuntu-maverick-amd64", "Image to be loaded on to the nodes.")

	flag.Parse()

	if hallocC, error = hclient.NewClient(); error != nil {
		fmt.Fprintf(os.Stderr, "Failed to get new client from client package in halloc.\n")
		os.Exit(1)
	}

	allocationNumber = uint64(0)
	allocationList = flag.Args()
}

func allocationFail(allocType string) {
	switch allocType {
	case "number":
		fmt.Fprintf(os.Stderr, "Not enough nodes to satisfy request number.\n")
		os.Exit(1)
	case "list":
		fmt.Fprintf(os.Stderr, "Some of the nodes in the list provided don't exist are are allocated.\n")
		os.Exit(1)
	}

}

func requestNumber() (tmpAllocationNumber uint64) {
	nm := iface.Nummsg{numNodes, image, 300}

	someBytes, error := bs.PostServer("/number", nm)
	hclient.PrintError("Failed to post the request for number of nodes to heckle.", error)

	if len(someBytes) == 0 {
		allocationFail("number")
	}

	error = json.Unmarshal(someBytes, &tmpAllocationNumber)
	hclient.PrintError("Failed to unmarshal allocation number from http response in request number.", error)

	fmt.Fprintf(os.Stdout, "Your allocation number is %d.", tmpAllocationNumber)

	return
}

func requestList() (tmpAllocationNumber uint64) {
	nm := iface.Listmsg{allocationList, image, 300, 0}

	someBytes, error := bs.PostServer("/list", nm)
	hclient.PrintError("Failed to post the request for list of nodes to heckle.", error)

	if len(someBytes) == 0 {
		allocationFail("list")
	}

	error = json.Unmarshal(someBytes, &tmpAllocationNumber)
	hclient.PrintError("Failed to unmarshal allocation number from http response in request list.", error)

	fmt.Fprintf(os.Stdout, "Your allocation number is %s.\n", strconv.FormatUint(tmpAllocationNumber, 10))

	return
}

func requestTimeIncrease() {
	tmpTimeMsg := int64(timeIncrease * 3600)

	_, error := bs.PostServer("/increaseTime", tmpTimeMsg)
	hclient.PrintError("Failed to post the request for time increase to heckle.", error)

	return
}

func ConvertTime(tm int64) string {
	return time.Unix(tm, 0).Format("15:04:05")
}

func printStatus(key string, value *iface.StatusMessage, i int) {
	currentTime := time.Unix(value.LastActivity, 0).Format("Jan _2 15:04:05")
	processName := os.Args[0]
	pid := os.Getpid()
	fmt.Println(fmt.Sprintf("%s %s %s[%d]: %s", currentTime, key, processName, pid, value.Info[i].Message))
	/*fmt.Fprintf(os.Stdout, "NODE: %s\tSTATUS: %s\tLAST ACTIVITY: %s\tMESSAGE: %s : %s\n", key, value.Status, ConvertTime(value.LastActivity), ConvertTime(value.Info[i].Time), value.Info[i].Message)*/
	return
}

func pollForStatus() {
	start := time.Now()
	statMap := make(map[string]*iface.StatusMessage)
	pollStatus := make(map[string]string)
	for {
		time.Sleep(10000000000)

		someBytes, error := bs.PostServer("/status", allocationNumber)
		hclient.PrintError("Failed to post for status of nodes to heckle.", error)

		error = json.Unmarshal(someBytes, &statMap)
		hclient.PrintError("Failed to unmarshal status info from http response in status polling.", error)

		done := false
		for key, value := range statMap {
			pollStatus[key] = value.Status
			done = true
			for i := range value.Info {
				if len(value.Info) != 0 {
					printStatus(key, value, i)
				}
			}
			done = done && (pollStatus[key] == "Ready")
			if pollStatus[key] == "Cancel" {
				delete(pollStatus, key)
			}

		}
		//Get list of nodes for message
		if done {
			end := time.Now()
			final := end.Sub(start).String()
			fmt.Println(fmt.Sprintf("Allocation #%d complete.  The build process for allocation %d took: %s", allocationNumber, allocationNumber, final))
			os.Exit(0)
		}
	}
}

func main() {
	var error error
	if len(allocationList) != 0 && numNodes != 0 {
		hclient.PrintError("Cannot use node list, and number of nodes option at the same time.", errors.New("Flag contradiction"))
		os.Exit(1)
	} else if (len(allocationList) == 0 && numNodes == 0 && timeIncrease == 0 && freeAlloc == 0) || help {
		hclient.Usage()
		os.Exit(0)
	}

	if bs, error = hallocC.SetupClient("heckle"); error != nil {
		hclient.PrintError("Failed to setup client in halloc.", errors.New("Client Setup Failed"))
		os.Exit(1)
	}

	if timeIncrease != 0 {
		requestTimeIncrease()
	}

	if numNodes != 0 {
		allocationNumber = requestNumber()
		pollForStatus()
	} else if len(allocationList) != 0 {
		allocationNumber = requestList()
		pollForStatus()
	}
}
