package main

import (
	"encoding/json"
	"flag"
	cli "flunky/client"
	"fmt"
	"os"
	"sort"
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

func format(outletStatus map[string]States) {
	bins := make(map[string][]string, 5)
	for node, state := range outletStatus {
		translator := map[bool]string{true: "on", false: "off"}
		ostatus := translator[state.State]
		if state.Reboot {
			ostatus = ostatus + "*"
		}
		value, ok := bins[ostatus]
		if ok {
			bins[ostatus] = append(value, node)
		} else {
			bins[ostatus] = []string{node}
		}
	}

	fmt.Println("STATUS\t\tNODES")

	for state, nodes := range bins {
		sort.Strings(nodes)
		fmt.Println(fmt.Sprintf("%s\t\t%s", state, nodes))
	}

	return
}

func commPower(nodes []string, cmd string) (outletStatus map[string]States, err error) {
	var powerCommand string
	if cmd == "status" {
		powerCommand = "/status"
	} else {
		powerCommand = "/command/" + cmd
	}

	powerServ, err := cli.NewClient()
	if err != nil {
		cli.PrintError("Unable to set up communication to power", err)
		return
	}

	client, err := powerServ.SetupClient("power")
	if err != nil {
		cli.PrintError("Unable to lookup power", err)
		return
	}

	statusRet, err := client.PostServer(powerCommand, nodes)

	if err != nil {
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
		if err != nil {
			os.Exit(1)
		}
		format(status)
	} else {
		_, err := commPower(node, command)
		if err != nil {
			cli.PrintError(fmt.Sprintf("Failed to %s nodes %s", command, node), err)
			os.Exit(1)
		}
	}
}
