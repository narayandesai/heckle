package main

import (
	"flag"
	"fmt"
	"os"
	"json"
	"io/ioutil"
	inet "flunky/net"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(0)
}

var help bool
var fileDir string

func init() {
	fileDir = "../../../etc/Heckle/"
	flag.BoolVar(&help, "h", false, "Print help message")
}

func ReadConfig(fname string) (comData map[string]string) {
	var contents []byte
	_, err := os.Stat(fname)
	if err != nil {
		fmt.Println(fname + " does not exsist")
	} else {
		contents, err = ioutil.ReadFile(fname)
		if err != nil {
			fmt.Println("Cannot read file")
		}
	}

	if len(contents) > 0 {
		err := json.Unmarshal(contents, &comData)
		if err != nil {
			fmt.Println("Unable to unmarshal data")
		}
	} else {
		fmt.Println("Invalid json. Could not read data")
	}
	return

}

func GetDump(daemonName string) {
	config := ReadConfig(fileDir)
	server, ok := config[daemonName]

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

	if len(os.Args) <= 1 {
		fmt.Println("No arguements provided")
		Usage()
		os.Exit(1)
	} else {
		for _, name := range flag.Args() {
			GetDump(name)
		}
	}
}
