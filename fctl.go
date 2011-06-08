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

type status struct {
	Address string
	Status string
}

type readyBailNode struct {
	ready, Bail bool
}

func main() {
    flag.Parse()
    if help || minutesTimeout < 1 {
        Usage()
	os.Exit(0)
       }

	secondsTimeout := minutesTimeout * 60
	cancelTime := time.Seconds() + secondsTimeout
	bailOut, ready := false, false
	addresses := flag.Args()
	readyBail := make([len(addresses)]readyBailNode)

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

	for time.Seconds() < cancelTime && !bailOut && !ready {
		for pos, value := range addresses {
			cm := new(ctlmsg)
			cm.Address = value
			js, _ := json.Marshal(cm)
			buf := bytes.NewBufferString(string(js))
			ret, _ := bs.Post("/status", buf)
	
			fmt.Fprintf(os.Stderr, "%s\n", ret)
		}
		time.Sleep(10000000000)
	}
}
