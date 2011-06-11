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
	Address string
	Image string
	Extra map[string]string
}

type infoMsg struct {
	time uint64
	Message string
	MsgType string
}

type statusMessage struct {
	Address string
	Status string
	Info []infoMsg
}

type readyBailNode struct {
	Ready, Bail, Printed bool
}

func interpretPoll(status string) (ready bool, bail bool) {
	//fmt.Fprintf(os.Stdout, "Status = %s\n", status)
	if status == "Ready" {
		ready = true
	} else if status == "Cancel" {
		bail = true
	}
	return ready, bail
}

func determineDone(readyBail []readyBailNode) bool {
	done := true
	for i := 0 ; i < len(readyBail) ; i++ {
		done = done && (readyBail[i].Ready || readyBail[i].Bail)
	}
	return done
}

func pollForStatusMessage(pos int, node string, readyBail []readyBailNode, bs *flunky.BuildServer) {
	cm := new(ctlmsg)
	cm.Address = node
	js, _ := json.Marshal(cm)
	buf := bytes.NewBufferString(string(js))
	ret, _ := bs.Post("/status", buf)
	fmt.Fprintf(os.Stdout, "%s\n", ret)
	tmpStatusMessage := new(statusMessage)
	json.Unmarshal(ret, tmpStatusMessage)

	readyBail[pos].Ready, readyBail[pos].Bail = interpretPoll(tmpStatusMessage.Status)
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
	if readyBail.Ready && !readyBail.Printed {
		fmt.Fprintf(os.Stdout, "NODE: %s STATUS: Ready ", node)
		readyBail.Printed = true
	} else if readyBail.Bail && !readyBail.Printed {
		fmt.Fprintf(os.Stdout, "NODE: %s STATUS: Failed ", node)
		readyBail.Printed = true
	} else {
		fmt.Fprintf(os.Stdout, "NODE: %s STATUS: Building ", node)
	}

	for _, value := range tmpStatusMessage.Info {
		if value.MsgType == "Info" {
			fmt.Fprintf(os.Stdout, "INFO: ")
		} else if value.MsgType == "Error" {
			fmt.Fprintf(os.Stdout, "ERROR: ")
		}
		fmt.Fprintf(os.Stdout, "%s", value.Message)
	}
	fmt.Fprintf(os.Stdout, "\n")
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
	// FIXME: need to add in extradata

	for _, value := range addresses {
		cm.Address = value
		js, _ := json.Marshal(cm)		
		buf := bytes.NewBufferString(string(js))
		_,_ = bs.Post("/ctl", buf)
	}

	pollForMessages(cancelTime, addresses, readyBail, bs)
	fmt.Fprintf(os.Stdout, "Done allocating your nodes. Report failed builds to your system administrator.\n")
}
