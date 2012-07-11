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

func (msg *infoMsg) Format(client string) string {
	strval := fmt.Sprintf("%s: %s: %s: %s", time.Now().Format(time.UnixDate), client, msg.MsgType, msg.Message)
	return strval
}

type statusMessage struct {
	Status       string
	LastActivity int64
	Info         []infoMsg
}

type nodeStatus struct {
        Status string
	Failed bool
        NumPrinted int
}

func (nS *nodeStatus) UpdateFromStatus(node string, status statusMessage) {
       if (nS.Status == "Cancel") {
                return
       }

       for i, value := range status.Info {
                if (i > nS.NumPrinted) {
		   fmt.Fprintf(os.Stdout, "%s\n", value.Format(node))
                   nS.NumPrinted = i
                }
       }

       if (nS.Status != status.Status) {
                fmt.Fprintf(os.Stdout, "%s: %s: Status: %s\n", time.Now().Format(time.UnixDate), node, status.Status)
                nS.Status = status.Status
       }

       if (time.Now().Unix() - status.LastActivity) > 300 {
                /* node activity watchdog, different than timeouts below */
                nS.Failed = true
		fmt.Fprintf(os.Stderr, "%s: %s: Watchdog timeout. Node build failed", time.Now().Format(time.UnixDate), node)
       }
}

func determineDone(statusMap map[string]nodeStatus) bool {
	done := true
	for k, _ := range statusMap {
		done = done && (statusMap[k].Status == "Ready" || statusMap[k].Status == "Cancel" )
	}
	return done
}

func pollForMessages(cancelTime time.Time, addresses []string, bs *fnet.BuildServer) {
	done := false
	readyBail := make(map[string]nodeStatus, len(addresses))
	for _, value := range addresses {
		readyBail[value] = nodeStatus{"", false, -1}
	}

	statRequest := new(ctlmsg)
	statRequest.Addresses = addresses
	statRequest.Time = time.Now().Unix()

	statmap := make(map[string]statusMessage, 50)

	for time.Since(cancelTime).Seconds() < 0 && !done {
		ret, _ := bs.PostServer("/status", statRequest)
		json.Unmarshal(ret, &statmap)

		for _, address := range addresses {
			ns := readyBail[address]
                        ns.UpdateFromStatus(address, statmap[address])
			readyBail[address] = ns
		}
		done = determineDone(readyBail)
		if (done == false) {
             		time.Sleep(time.Second)
                }
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

	secondsTimeout := minutesTimeout * 60 * 1000000000
	cancelTime := time.Now().Add(time.Duration(secondsTimeout))
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
	_, err = bs.PostServer("/ctl", cm)
	if err != nil {
		fclient.PrintError("Failed to allocate node", err)
		os.Exit(1)
	}

	pollForMessages(cancelTime, addresses, bs)
	fmt.Fprintf(os.Stdout, "Done allocating your nodes. Report failed builds to your system administrator.\n")
}
