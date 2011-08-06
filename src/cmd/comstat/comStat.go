package main

import (
	"flag"
	"fmt"
	"os"
	fclient "flunky/client"
)

var Usage = func() {
    fmt.Println(fmt.Sprintf("Command syntax -- %s daemon1 daemon2 ...daemonN", os.Args[0]))
	fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(0)
}

var help bool
var fileDir string

func init() {
	flag.BoolVar(&help, "h", false, "Print help message")
}


func main() {
	flag.Parse()

	if help {
		Usage()
		os.Exit(0)
	}

	if len(os.Args) <= 1 {
		fmt.Println("No arguments provided")
		Usage()
		os.Exit(1)
	} else {
		comm, err := fclient.NewClient()
		if err != nil {
			fmt.Println("Failed to setup communication")
			os.Exit(1)
		}

		for _, name := range flag.Args() {
			client, err := comm.SetupClient(name)
			if err != nil {
				fmt.Println(fmt.Sprintf("Failed to lookup component %s", name))
				os.Exit(1)
			}
			resp, err := client.Get("dump")
			if err != nil {
				fmt.Println(fmt.Sprintf("Failed to contact component %s", name))
			        os.Exit(1)
			}
			fmt.Println(string(resp))
		}
	}
}
