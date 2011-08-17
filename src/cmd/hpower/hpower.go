package main

import (
	"fmt"
	"flag"
	"os"
	"json"
	"tabwriter"
	cli "flunky/client"
)

var help bool
var command string

func init() {
	flag.BoolVar(&help, "h", false, "Print help message")
	flag.StringVar(&command, "c", " ", "Run command on power server")
}

type outletNode struct {
	Address string
	Outlet  string
}

type States struct {
	State  bool
	Reboot bool
}

func returnState(info bool, value string) (ret string) {
	switch value {
	case "state":
		if info {
			ret = "on"
		} else {
			ret = "off"
		}
		break
	case "reboot":
		if info {
			ret = "is rebooting"
		} else {
			ret = "no pending reboot"
		}
		break
	default:
		cli.PrintError("Unsupported op", os.NewError("no op"))
		ret = ""
		break
	}
	return
}

func format(outletStatus map[string]States) (stats []string) {
	for key, node := range outletStatus {
		status := fmt.Sprintf("%s\t %s\t %s\n", key, returnState(node.State, "state"), returnState(node.Reboot, "reboot"))
		stats = append(stats, status)
	}
	return
}

func print(statusList []string) {
	heading := "NODE ID\t OUTLET STATUS\t\t\t CONTROL STATE\n"
	tabWrite := tabwriter.NewWriter(os.Stdout, 1, 4, 0, '\t', 0)
	tabWrite.Write([]byte(heading))

	for _, stat := range statusList {
		tabWrite.Write([]byte(stat))
	}
	tabWrite.Flush()
	return
}

func printCommand(nodes []string, cmd string) {
	fmt.Println(fmt.Sprintf("Nodes %s have been %s", nodes, cmd))
	return
}

func commPower(nodes []string, cmd string) (outletStatus map[string]States) {
	var powerCommand string
	if cmd == "status" {
		powerCommand = "/status"
	} else {
		powerCommand = "/command/" + cmd
	}

	powerServ, err := cli.NewClient()
	if err != nil {
		cli.PrintError("Unable to set up communication to power", err)
		os.Exit(1)
	}

	client, err := powerServ.SetupClient("power")
	if err != nil {
		cli.PrintError("Unable to lookup power", err)
		os.Exit(1)
	}

	statusRet, err := client.PostServer(powerCommand, nodes)
        if err != nil{
	    cli.PrintError("Unable to post", err)
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
		cli.Usage()
		os.Exit(0)
	}

	node := flag.Args()
	switch command {
	case "on", "off", "reboot", "status":
		break
	default:
		cli.PrintError("Unsupported operation.", os.NewError("Unsuppored op"))
		os.Exit(1)
		break
	}

	if command == "status" {
		outlets := commPower(node, command)
		statusList := format(outlets)
		print(statusList)
	} else {
		outlet := commPower(node, command)
		if outlet == nil {
			printCommand(node, command)
		}
	}
}
