package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	fclient "flunky/client"
)

var help bool
var fileDir string

func init() {
	flag.BoolVar(&help, "h", false, "Print help message")
}

func prettyPrint(data string) {
     prettyData := strings.Split(data, ",")
     for _, pretty := range(prettyData){
        fmt.Println(pretty)
     }
     return

}

func main() {
	flag.Parse()

	if help {
	        fmt.Println(fmt.Sprintf("Command syntax -- %s daemon1 daemon2 ...daemonN", os.Args[0]))
		fclient.Usage()
		os.Exit(0)
	}

	if len(os.Args) <= 1 {
		fclient.PrintError("No arguments provided", os.NewError("no args"))
		fmt.Println(fmt.Sprintf("Command syntax -- %s daemon1 daemon2 ...daemonN", os.Args[0]))
		fclient.Usage()
		os.Exit(1)
	} else {
		comm, err := fclient.NewClient()
		if err != nil {
			fclient.PrintError("Failed to setup communication", err)
			os.Exit(1)
		}

		for _, name := range flag.Args() {
			client, err := comm.SetupClient(name)
			if err != nil {
				fclient.PrintError(fmt.Sprintf("Failed to lookup component %s", name), err)
				os.Exit(1)
			}
			resp, err := client.Get("dump")
			if err != nil {
				fclient.PrintError(fmt.Sprintf("Failed to contact component %s", name), err)
			        os.Exit(1)
			}
			//prettyPrint(string(resp))
			fmt.Println(string(resp))
		}
	}
}
