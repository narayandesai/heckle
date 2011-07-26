package main

import (
	"flag"
	"fmt"
	"os"
	inet "flunky/net"
	daemon "flunky/daemon"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(0)
}

var help bool
var fileDir string
var comStatDaemon *daemon.Daemon

func init() {
	flag.BoolVar(&help, "h", false, "Print help message")
}

func GetDump(daemonName string) {

	server, ok := comStatDaemon.Cfg.Data[daemonName]

	if !ok {
		fmt.Println(fmt.Sprintf("Cannot find URL for component %s", daemonName))
		os.Exit(1)
	}

	query := inet.NewBuildServer("http://"+server, true)

	resp, err := query.Get("dump")
	if err != nil {
		fmt.Println(fmt.Sprintf("Cannot read data from %s", daemonName), err)
	} else {
		fmt.Println(string(resp))
		return
	}
}

func main() {
	flag.Parse()

	if help {
		Usage()
		os.Exit(0)
	}

	for _, name := range flag.Args() {
		GetDump(name)
	}
}
