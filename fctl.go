   package main

import (
	"bytes"
    "flag"   
    "fmt"
	"json"
    "os"
    "./flunky"
)

var Usage = func() {
    fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
    flag.PrintDefaults()
}

var server string
var verbose bool
var help bool
var image string
var wait bool
var timeout int
var extra string

func init() {
    flag.BoolVar(&help, "h", false, "print usage")
    flag.BoolVar(&verbose, "v", false, "print debug information")
    flag.StringVar(&server, "S", "http://localhost:8081", "server base URL")
	flag.StringVar(&image, "i", "", "image")
	flag.BoolVar(&wait, "w", false, "Wait for build completion")
	flag.StringVar(&extra, "e", "", "Extradata for allocation")
	flag.IntVar(&timeout, "t", -1, "Allocation timeout")
}

type ctlmsg struct {
	Address string
	Image string
	Extra map[string]string
}

func main() {
    flag.Parse()
    if help {
        Usage()
        os.Exit(0)
       }

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

	addresses := flag.Args()
	for a := range addresses {
		cm.Address = addresses[a]
		js, _ := json.Marshal(cm)		
		buf := bytes.NewBufferString(fmt.Sprintf("%s", js))
		_ = bs.Post("/ctl", buf)
	}
}
