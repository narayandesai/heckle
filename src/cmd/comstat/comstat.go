//BUG (Mike Guantonio): Conversion between hours and minutes does not work
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"io/ioutil"
	"json"
	"time"
	"tabwriter"
	fclient "flunky/client"
	fnet "flunky/net"
)

var help bool
var fileDir string
var error bool
var log bool
var stat bool
var dump bool

type Status struct {
	StartTime    int64
	UpTime       int64
	LastActivity int64
	Errors       int
}

func init() {
	flag.BoolVar(&help, "h", false, "Print help message")
	flag.BoolVar(&error, "e", false, "Print error logs")
	flag.BoolVar(&log, "l", false, "Print log file for specified host")
	flag.BoolVar(&stat, "s", false, "Print daemon status")
	flag.BoolVar(&dump, "d", false, "Dump daemon data")
}

func logs(name string) string {
	var errorLog string
	logName := "/var/log/" + name + ".log"
	_, err := os.Stat(logName)
	if err != nil {
		fclient.PrintError(err.String(), err)
		os.Exit(1)
	}
	file, err := ioutil.ReadFile(logName)
	if log {
		return string(file)
	}
	if error {
		contents := strings.Split(string(file), "\n")
		for _, data := range contents {
			dex := strings.Index(data, "ERROR")
			if dex > 0 {
				errorLog = errorLog + data + "\n"
			}
		}
		return (errorLog)
	}
	return ""
}

func printStatus(status map[string]Status) {
	tabWrite := tabwriter.NewWriter(os.Stdout, 1, 4, 0, '\t', 0)
	header := "NAME\t START\t LAST_ACTIVITY\t UPTIME\t ERRORS\n"
	tabWrite.Write([]byte(header))

        for name, statusType := range(status){
	    startTime := time.SecondsToLocalTime(statusType.StartTime).Format("01-02 15:04:05")
	    upTime := time.SecondsToLocalTime(statusType.UpTime).Format("04:05")
	    lastActivity := time.SecondsToLocalTime(statusType.LastActivity).Format("01-02 15:04:05")
	    line := fmt.Sprintf("%s\t %s\t         %s\t %s\t %d\n", name, startTime, lastActivity, upTime, statusType.Errors)
	    tabWrite.Write([]byte(line))
	}
	tabWrite.Flush()
	return
}

func main() {
	flag.Parse()
	clientList := make(map[string]*fnet.BuildServer)

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
	}

	components := flag.Args()
	if len(components) == 0 {
		fclient.PrintError("No clients specified", os.NewError("no client"))
		os.Exit(1)
	}

	for _, name := range components {
		comm, err := fclient.NewClient()
		if err != nil {
			fclient.PrintError("Failed to setup communication", err)
			os.Exit(1)
		}
		client, err := comm.SetupClient(name)
		if err != nil {
			fclient.PrintError(fmt.Sprintf("Failed to lookup component %s", name), err)
			os.Exit(1)
		}
		clientList[name] = client
	}

	if stat {
	    var statusType Status
	    respList := make(map[string]Status)
	    for name, client := range(clientList){
		resp, err := client.Get("daemon")
		if err != nil {
			fclient.PrintError("Could not get daemon status", err)
			os.Exit(1)
		}
		err = json.Unmarshal(resp, &statusType)
                if err != nil {
                    fclient.PrintError(err.String(), err)
                }
		respList[name] = statusType
	     }
	    printStatus(respList)
	}

	if dump {
	    for name, client := range(clientList){
		resp, err := client.Get("dump")
		if err != nil {
			fclient.PrintError(fmt.Sprintf("Failed to contact component %s", name), err)
			os.Exit(1)
		}
		fmt.Println(string(resp))
		}
	}

	if log {
	    for name, _ := range(clientList){
		logging := logs(name)
		fmt.Println(logging)
            }
	}

	if error {
	    for name, _ := range(clientList){
		errorLogging := logs(name)
		if len(errorLogging) <= 0 {
			fmt.Println(fmt.Sprintf("No errors exsist in the logs for %s", name))
		} else {
			fmt.Println(errorLogging)
		}
	     }
	}

}
