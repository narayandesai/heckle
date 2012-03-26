package main

import (
	//"bytes"
	"encoding/json"
	"errors"
	"flag"
	fclient "flunky/client"
	fnet "flunky/net"
	"fmt"
	"os"
	"time"
)

var server string
var verbose bool
var help bool
var image string
var wait bool
var minutesTimeout int64
var extra string
var username, password string

func init() {
	flag.BoolVar(&help, "h", false, "print usage")
	flag.BoolVar(&verbose, "v", false, "print debug information")
	flag.StringVar(&image, "i", "", "image")
	flag.BoolVar(&wait, "w", false, "Wait for build completion")
	flag.StringVar(&extra, "e", "", "Extradata for allocation")
	flag.Int64Var(&minutesTimeout, "t", 45, "Allocation timeout in minutes")
}

type ctlmsg struct {
	Addresses []string
	Time      int64
	Image     string
	Extra     map[string]string
}

type infoMsg struct {
	Time    int64
	Message string
	MsgType string
}

type statusMessage struct {
	Status       string
	LastActivity int64
	Info         []infoMsg
}

type readyBailNode struct {
	Ready, Bail, Printed bool
}

func (msg *infoMsg) Format(client string) string {
	strval := fmt.Sprintf("%s: %s: Node: %s: %s", msg.MsgType, time.Unix(msg.Time, 0).Format(time.UnixDate), client, msg.Message)
	return strval
}

func (rbn *readyBailNode) InterpretPoll(status string, lastActivity int64) {
	if status == "Ready" {
		rbn.Ready = true
	} else if status == "Cancel" || time.Now().Sub(lastActivity) > 300 {
		rbn.Bail = true
	}
}

func determineDone(readyBail map[string]readyBailNode) bool {
	done := true
	for k, _ := range readyBail {
		done = done && (readyBail[k].Ready || readyBail[k].Bail)
	}
	return done
}

func pollForMessages(cancelTime int64, addresses []string, bs *fnet.BuildServer) {
	done := false
	readyBail := make(map[string]readyBailNode, len(addresses))
	for _, value := range addresses {
		readyBail[value] = readyBailNode{false, false, false}
	}

	statRequest := new(ctlmsg)
	statRequest.Addresses = addresses
	statRequest.Time = time.Now()

	statmap := make(map[string]statusMessage, 50)

	for time.Now() < cancelTime && !done {
		time.Sleep(10000000000)
		/*sRjs, _ := json.Marshal(statRequest)
		statRequest.Time = time.Seconds()
		reqbuf := bytes.NewBufferString(string(sRjs))*/

		ret, _ := bs.PostServer("/status", statRequest)
		json.Unmarshal(ret, &statmap)

		for _, address := range addresses {
			rbn := readyBail[address]
			rbn.InterpretPoll(statmap[address].Status, statmap[address].LastActivity)
			readyBail[address] = rbn
			printStatusMessage(address, rbn, statmap[address])
		}
		done = determineDone(readyBail)
	}
}

func printStatusMessage(node string, readyBail readyBailNode, tmpStatusMessage statusMessage) {
	for _, value := range tmpStatusMessage.Info {
		fmt.Fprintf(os.Stdout, "%s\n", value.Format(node))
	}
	if readyBail.Ready && !readyBail.Printed {
		fmt.Fprintf(os.Stdout, "NODE: %s TIME: %s STATUS: Ready \n", node, time.Now().Format(time.UnixDate))
		readyBail.Printed = true
	} else if readyBail.Bail && !readyBail.Printed {
		fmt.Fprintf(os.Stdout, "NODE: %s TIME: %s STATUS: Failed \n", node, time.Now().Format(time.UnixDate))
		readyBail.Printed = true
	} else {
		fmt.Fprintf(os.Stdout, "NODE: %s TIME: %s STATUS: Building \n", node, time.Now().Format(time.UnixDate))
	}
}

func main() {
	flag.Parse()
	if help || minutesTimeout < 1 {
		fclient.Usage()
		os.Exit(0)
	}

	comm, err := fclient.NewClient()
	if err != nil {
		fclient.PrintError("Failed to setup communcation", err)
		os.Exit(1)
	}

	secondsTimeout := minutesTimeout * 60
	cancelTime := time.Now() + secondsTimeout
	addresses := flag.Args()

	bs, err := comm.SetupClient("flunky")

	bs.DebugLog(fmt.Sprintf("Allocating hosts: %s", flag.Args()))

	if image == "" {
		fclient.PrintError("-i option is required\n", errors.New("wrong arg"))
		fclient.Usage()
		os.Exit(1)
	}

	cm := new(ctlmsg)
	cm.Image = image
	cm.Addresses = addresses
	// FIXME: need to add in extradata
	/*js, _ := json.Marshal(cm)
	buf := bytes.NewBufferString(string(js))*/
	_, err = bs.PostServer("/ctl", cm)
	if err != nil {
		fclient.PrintError("Failed to allocate node", err)
		os.Exit(1)
	}

	pollForMessages(cancelTime, addresses, bs)
	fmt.Fprintf(os.Stdout, "Done allocating your nodes. Report failed builds to your system administrator.\n")
}
