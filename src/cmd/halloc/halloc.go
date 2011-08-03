package main

import (
	"flag"
	"json"
	"os"
	"bytes"
	"fmt"
	"time"
	"strconv"
	fnet "flunky/net"
	iface "flunky/interfaces"
	hclient "flunky/client"
)

var help, status bool
var server, image, fileDir string
var allocationList []string
var numNodes, timeIncrease int
var allocationNumber, freeAlloc uint64
var hallocC fnet.Communication
var bs *fnet.BuildServer

func init() {
	var error os.Error
	flag.BoolVar(&help, "h", false, "Print usage of command.")
	flag.BoolVar(&status, "s", false, "Print status of used nodes.")
	flag.IntVar(&numNodes, "n", 0, "Request an arbitrary number of nodes.")
	flag.IntVar(&timeIncrease, "t", 0, "Increase current allocation by this many hours.")
	flag.Uint64Var(&freeAlloc, "f", 0, "Free a reserved allocation number preemptively.")
	flag.StringVar(&image, "i", "ubuntu-maverick-amd64", "Image to be loaded on to the nodes.")

	flag.Parse()

	if hallocC, error = hclient.NewClient(); error != nil {
		fmt.Fprintf(os.Stderr, "Failed to get new client from client package in halloc.\n")
		os.Exit(1)
	}

	allocationNumber = uint64(0)
	allocationList = flag.Args()
}

func printError(message string, err os.Error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", message)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
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

	someBytes, error := json.Marshal(nm)
	printError("Failed to marshal nummsg in requestNumber function.", error)

	buf := bytes.NewBufferString(string(someBytes))
	someBytes, error = bs.Post("/number", buf)
	printError("Failed to post the request for number of nodes to heckle.", error)

	if len(someBytes) == 0 {
		allocationFail("number")
	}

	error = json.Unmarshal(someBytes, &tmpAllocationNumber)
	printError("Failed to unmarshal allocation number from http response in request number.", error)

	fmt.Fprintf(os.Stdout, "Allocation number is %s.", strconv.Uitoa64(tmpAllocationNumber))

	return
}

func requestList() (tmpAllocationNumber uint64) {
	nm := iface.Listmsg{allocationList, image, 300}

	someBytes, error := json.Marshal(nm)
	printError("Failed to marshal nummsg in requestList function.", error)

	buf := bytes.NewBufferString(string(someBytes))
	someBytes, error = bs.Post("/list", buf)
	printError("Failed to post the request for list of nodes to heckle.", error)

	if len(someBytes) == 0 {
		allocationFail("list")
	}

	error = json.Unmarshal(someBytes, &tmpAllocationNumber)
	printError("Failed to unmarshal allocation number from http response in request list.", error)

	fmt.Fprintf(os.Stdout, "Allocation number is %s.\n", strconv.Uitoa64(tmpAllocationNumber))

	return
}

func requestTimeIncrease() {
	tmpTimeMsg := int64(timeIncrease * 3600)

	someBytes, error := json.Marshal(tmpTimeMsg)
	printError("Failed to marshal time increase in requestTimeIncrease function.", error)

	buf := bytes.NewBufferString(string(someBytes))
	someBytes, error = bs.Post("/increaseTime", buf)
	printError("Failed to post the request for time increase to heckle.", error)

	return
}

func ConvertTime(tm int64) string {
	return time.SecondsToLocalTime(tm).Format("15:04:05")
}

func pollForStatus() {
	statMap := make(map[string]*iface.StatusMessage)
	pollStatus := make(map[string]string)
	for {
		time.Sleep(10000000000)
		someBytes, error := json.Marshal(allocationNumber)
		printError("Failed to marshal allocation number for status poll.", error)

		buf := bytes.NewBufferString(string(someBytes))
		someBytes, error = bs.Post("/status", buf)
		printError("Failed to post for status of nodes to heckle.", error)

		error = json.Unmarshal(someBytes, &statMap)
		printError("Failed to unmarshal status info from http response in status polling.", error)

		done := false
		for key, value := range statMap {
			pollStatus[key] = value.Status
			done = true
			for i := range value.Info {
				if len(value.Info) != 0 {

					fmt.Fprintf(os.Stdout, "NODE: %s\tSTATUS: %s\tLAST ACTIVITY: %s\tMESSAGE: %s : %s\n", key, value.Status, ConvertTime(value.LastActivity), ConvertTime(value.Info[i].Time), value.Info[i].Message)
				}
			}
			done = done && (pollStatus[key] == "Ready")
			if pollStatus[key] == "Cancel" {
				pollStatus[key] = "", false
			}

		}

		if done {
			fmt.Fprintf(os.Stdout, "Done allocating nodes.  Your allocation number is %d.  Please report failures to your system administrator.\n", allocationNumber)
			os.Exit(0)
		}
	}
}

func freeAllocation() {
	someBytes, error := json.Marshal(freeAlloc)
	printError("Failed to marshal allocation number for status poll.", error)
	buf := bytes.NewBufferString(string(someBytes))
	someBytes, error = bs.Post("/freeAllocation", buf)
	printError("Failed to post for status of nodes to heckle.", error)
}

func nodeStatus() {
	buf := bytes.NewBufferString("")
	someBytes, error := bs.Post("/nodeStatus", buf)
	printError("Failed to post the request for node status to heckle.", error)

	fmt.Fprintf(os.Stdout, "%s", string(someBytes))

	return
}

func main() {
	var error os.Error
	if len(allocationList) != 0 && numNodes != 0 {
		printError("Cannot use node list, and number of nodes option at the same time.", os.NewError("Flag contradiction"))
		os.Exit(1)
	} else if (len(allocationList) == 0 && numNodes == 0 && timeIncrease == 0 && freeAlloc == 0 && !status) || help {
		usage()
		os.Exit(0)
	}

	if bs, error = hallocC.SetupClient("heckle"); error != nil {
		printError("Failed to setup client in halloc.", os.NewError("Client Setup Failed"))
		os.Exit(1)
	}

	if status {
		nodeStatus()
		os.Exit(0)
	}

	if timeIncrease != 0 {
		requestTimeIncrease()
	}

	if freeAlloc != 0 {
		freeAllocation()
	}

	if numNodes != 0 {
		allocationNumber = requestNumber()
		pollForStatus()
	} else if len(allocationList) != 0 {
		allocationNumber = requestList()
		pollForStatus()
	}
}
