package main

import (
	"bytes"
	"flag"
	"fmt"
	"json"
	"os"
	"time"
	"./flunky"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(0)
}

var server string
var verbose bool
var help bool
var image string
var wait bool
var minutesTimeout int64
var extra string

func init() {
	flag.BoolVar(&help, "h", false, "print usage")
	flag.BoolVar(&verbose, "v", false, "print debug information")
	flag.StringVar(&server, "S", "http://localhost:8080", "server base URL")
	flag.StringVar(&image, "i", "", "image")
	flag.BoolVar(&wait, "w", false, "Wait for build completion")
	flag.StringVar(&extra, "e", "", "Extradata for allocation")
	flag.Int64Var(&minutesTimeout, "t", 45, "Allocation timeout in minutes")
}

type ctlmsg struct {
	Addresses []string
	Image     string
	Extra     map[string]string
}

type sctlmsg struct {
	Address string
	Image   string
	Extra   map[string]string
}

type infoMsg struct {
	Time    int64
	Message string
	MsgType string
}

func (msg *infoMsg) Format (client string) (string) {
	strval := fmt.Sprintf("%s: %s: Node: %s: %s", msg.MsgType, time.SecondsToUTC(msg.Time).Format(time.UnixDate), client, msg.Message)
	return strval
}

type statusMessage struct {
	Status string
	Info   []infoMsg
}

type readyBailNode struct {
	Ready, Bail, Printed bool
}

func (status *readyBailNode) interpretPoll(status string) {
	if status == "Ready" {
		status.Ready = true
	} else if status == "Cancel" {
		status.Bail = true
	}
}

func determineDone(readyBail []readyBailNode) bool {
	done := true
	for i := 0; i < len(readyBail); i++ {
		done = done && (readyBail[i].Ready || readyBail[i].Bail)
	}
	return done
}

func pollForStatusMessage(pos int, node string, readyBail []readyBailNode, bs *flunky.BuildServer) {
	cm := new(sctlmsg)
	cm.Address = node
	js, _ := json.Marshal(cm)
	buf := bytes.NewBufferString(string(js))
	ret, _ := bs.Post("/status", buf)
	tmpStatusMessage := new(statusMessage)
	json.Unmarshal(ret, tmpStatusMessage)

	readyBail[pos].interpretPoll(tmpStatusMessage.Status)
	printStatusMessage(node, &readyBail[pos], tmpStatusMessage)
}

func pollForMessages(cancelTime int64, addresses []string, readyBail []readyBailNode, bs *flunky.BuildServer) {
	done := false
	for time.Seconds() < cancelTime && !done {
		time.Sleep(10000000000)
		for pos, value := range addresses {
			pollForStatusMessage(pos, value, readyBail, bs)
		}
		done = determineDone(readyBail)
	}
}

func printStatusMessage(node string, readyBail *readyBailNode, tmpStatusMessage *statusMessage) {
	for _, value := range tmpStatusMessage.Info {
		fmt.Fprintf(os.Stdout, "%s\n", value.Format(node))
	}

	if readyBail.Ready && !readyBail.Printed {
		fmt.Fprintf(os.Stdout, "NODE: %s TIME: %s STATUS: Ready \n", node, time.UTC().Format(time.UnixDate))
		readyBail.Printed = true
	} else if readyBail.Bail && !readyBail.Printed {
		fmt.Fprintf(os.Stdout, "NODE: %s TIME: %s STATUS: Failed \n", node, time.UTC().Format(time.UnixDate))
		readyBail.Printed = true
	} else {
		fmt.Fprintf(os.Stdout, "NODE: %s TIME: %s STATUS: Building \n", node, time.UTC().Format(time.UnixDate))
	}
}

func main() {
	flag.Parse()
	if help || minutesTimeout < 1 {
		Usage()
		os.Exit(0)
	}

	secondsTimeout := minutesTimeout * 60
	cancelTime := time.Seconds() + secondsTimeout
	addresses := flag.Args()
	readyBail := make([]readyBailNode, len(addresses))

	bs := flunky.NewBuildServer(server, verbose)

	bs.DebugLog(fmt.Sprintf("Server is %s", server))

	bs.DebugLog(fmt.Sprintf("Allocating hosts: %s", flag.Args()))

	if image == "" {
		fmt.Fprintf(os.Stderr, "-i option is required\n")
		Usage()
		os.Exit(1)
	}

	cm := new(ctlmsg)
	cm.Image = image
	cm.Addresses = addresses
	// FIXME: need to add in extradata
	js, _ := json.Marshal(cm)
	buf := bytes.NewBufferString(string(js))
	_, _ = bs.Post("/ctl", buf)

	pollForMessages(cancelTime, addresses, readyBail, bs)
	fmt.Fprintf(os.Stdout, "Done allocating your nodes. Report failed builds to your system administrator.\n")
}
