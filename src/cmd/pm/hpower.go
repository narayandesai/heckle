package main

import (
	"fmt"
	"flag"
	"os"
	"json"
	"tabwriter"
	cli "flunky/client"
)

var help, query, cycle, turnon, turnoff, reboot bool

func init() {
	flag.BoolVar(&help, "h", false, "Print help message")
	flag.BoolVar(&turnoff, "0", false, "Turn off nodes")
	flag.BoolVar(&turnon, "1", false, "Turn on nodes")
	flag.BoolVar(&cycle, "c", false, "Reboot nodes")
	flag.BoolVar(&query, "q", false, "Query status of nodes")
}

type outletNode struct {
	Address string
	Outlet  string
}

type States struct {
	State  bool
	Reboot bool
}

func (entry States) formatState(name string) (result string) {
	translator := map[bool]string{true:"on", false:"off"}
	ostatus := translator[entry.State]
	if (entry.Reboot) {
		ostatus = ostatus + "*"
	}
	result = fmt.Sprintf("%s\t\t%s\n", name, ostatus)
	return
}

func format(outletStatus map[string]States) {
	heading := "NODE\t\tSTATUS\n"
	tabWrite := tabwriter.NewWriter(os.Stdout, 1, 4, 0, '\t', 0)
	tabWrite.Write([]byte(heading))

	for node, state := range outletStatus {
		tabWrite.Write([]byte(state.formatState(node)))
	}

	tabWrite.Flush()
	return
}

func commPower(nodes []string, cmd string) (outletStatus map[string]States, err os.Error) {
	var powerCommand string
	if cmd == "status" {
		powerCommand = "/status"
	} else {
		powerCommand = "/command/" + cmd
	}

	powerServ, err := cli.NewClient()
	if (err != nil) {
		cli.PrintError("Unable to set up communication to power", err)
		return
	}

	client, err := powerServ.SetupClient("power")
	if (err != nil) {
		cli.PrintError("Unable to lookup power", err)
		return
	}

	statusRet, err := client.PostServer(powerCommand, nodes)

    if (err != nil) {
	    cli.PrintError("Unable to post", err)
		return
    }

	switch cmd {
	case "on", "off", "reboot":
		outletStatus = nil
		break
	case "status":
		err = json.Unmarshal(statusRet, &outletStatus)
		if err != nil {
			cli.PrintError("Unable to unmarsahll status", err)
		}
		break
	}
	return

}

func main() {
	flag.Parse()
	if help {
		fmt.Println("Usage of pm: pm [-0] [-1] [-c] [-q] <optional node list>")
		os.Exit(0)
	}

	var command string

	node := flag.Args()

	switch {
	case cycle == true: 
		command = "reboot"
	case turnon == true:
		command = "on"
	case turnoff == true:
		command = "off"
	case query == true:
		command = "status"
	default:
		fmt.Println("One of -0 -1 -c or -q is required")
		os.Exit(1)
	}

	if command == "status" {
		status, err := commPower(node, command)
		if (err != nil) {
			os.Exit(1)
		}
		format(status)
	} else {
		_, err := commPower(node, command)
		if (err != nil) {
			cli.PrintError(fmt.Sprintf("Failed to %s nodes %s", command, node), err)
			os.Exit(1)
		}
	}
}
