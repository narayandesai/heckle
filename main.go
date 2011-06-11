package main

import (
	"bytes"
	"flag"   
	"fmt"
	"os"
	"json"
	"strings"
	"io/ioutil"
	"./flunky"
)

var Usage = func() {
    fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
    flag.PrintDefaults()
}

var server string
var verbose bool
var help bool
var info string
var error string
var get string
var exec string

type infoErrorMsg struct {
	Message string
}

func init() {
	flag.BoolVar(&help, "h", false, "print usage")
	flag.BoolVar(&verbose, "v", false, "print debug information")
	flag.StringVar(&server, "S", "http://localhost:8080", "server base URL")
	flag.StringVar(&info, "i", "", "log info message")
	flag.StringVar(&error, "e", "", "log error message")
	flag.StringVar(&get, "g", "", "fetch and print endpoint")
	flag.StringVar(&exec, "x", "", "fetch rendered template and run it.")
}

func printError(errorMsg string, error os.Error) {
	if error != nil {
		fmt.Fprintf(os.Stderr, "%s\n", errorMsg)
	}
}

func parseCmdLine() {
	cmdLineFile, error := os.Open("/proc/cmdline")
	printError("ERROR:  Failed to open /proc/cmdline for reading.", error)

	cmdLineBytes, error := ioutil.ReadAll(cmdLineFile)
	printError("ERROR:  Failed to read all from /proc/cmdline.", error)

	error = cmdLineFile.Close()
	printError("ERROR:  Failed to close /proc/cmdline.", error)

	cmdLineOptions := strings.Split(string(cmdLineBytes), " ", -1)
	
	for _, value := range cmdLineOptions {
		cmdLineOption := strings.Split(value, "=", -1)

		if cmdLineOption[0] == "flunky" {
			server = cmdLineOption[1]
		}
	}
}

func main() {
    flag.Parse()
    if help {
        Usage()
        os.Exit(0)
       }

	if flag.Arg(0) == "/opt/bootlocal.sh" {
		parseCmdLine()
		exec = "install"
	}

	bs := flunky.NewBuildServer(server, verbose)

	bs.DebugLog(fmt.Sprintf("Server is %s", server))

	if get != "" {
		data, err := bs.Get(get)
		if err != nil {
			os.Exit(255)
		}
		fmt.Fprintf(os.Stdout, "%s", string(data))
	} else if exec != "" {
		status, err := bs.Run(exec)
		if err != nil {
			os.Exit(255)
		}
		os.Exit(status)
	} else if info != "" {
		im := new(infoErrorMsg)
		im.Message = info
		js, _ := json.Marshal(im)
		buf := bytes.NewBufferString(string(js))
		_, err := bs.Post("/info", buf)

		if err != nil {
			os.Exit(255)
		}
	} else if error != "" {
		em := new(infoErrorMsg)
		em.Message = error
		js, _ := json.Marshal(em)
		buf := bytes.NewBufferString(string(js))
		_, err := bs.Post("/error", buf)
		if err != nil {
			os.Exit(255)
		}
	}
}
