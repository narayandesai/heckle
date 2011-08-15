package main

import (
	"fmt"
	"flag"
	"os"
	"bytes"
	"strings"
	"time"
	"tabwriter"
	fnet "flunky/net"
	cli "flunky/client"
)

var help, status bool
var bs *fnet.BuildServer
var hstat fnet.Communication
var user string
var node string
var alloc string
var image string

func init() {
	var err os.Error
	if hstat, err = cli.NewClient(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get new client.\n")
		os.Exit(1)
	}

	if bs, err = hstat.SetupClient("heckle"); err != nil {
		cli.PrintError("Failed to setup client in halloc.", os.NewError("Client Setup Failed"))
		os.Exit(1)
	}

	flag.BoolVar(&help, "h", false, "Print usage of command.")
	flag.StringVar(&user, "u", " ", "Find user")
	flag.StringVar(&node, "n", " ", "Find nodes")
	flag.StringVar(&alloc, "a", " ", "Find alloc")
	flag.StringVar(&image, "i", " ", "Find image")
}

func FindDuration(end string) string {
	endTime, _ := time.Parse(time.UnixDate, end)
	ret := endTime.Format("01-02-2006 15:04")
	return ret
}

func printStatus(info string) {
	var output string
	statusList := strings.Split(info, "\n")
	tabWrite := tabwriter.NewWriter(os.Stdout, 1, 4, 0, '\t', 0)
	header := fmt.Sprintf("ID\t USER\t NODES\t IMAGE\t RESERVATION ENDING\n")
	tabWrite.Write([]byte(header))

	for _, status := range statusList {
		if len(status) > 0 {
			id := parse("ALLOCATION", status)
			user := parse("OWNER", status)
			image := parse("IMAGE", status)
			nodes := parse("NODE", status)
			timeEnd := parse("END:", status)
			duration := FindDuration(timeEnd)

			output = fmt.Sprintf("%s\t %s\t %s\t %s\t\t\t %s\n", id, user, nodes, image, duration)
			tabWrite.Write([]byte(output))
		}
	}
	tabWrite.Flush()
}


func parse(word string, words string) string {
	dex := strings.Index(words, word)
	newWords := words[dex:]

	dex = strings.Index(newWords, "\t")
	finalWord := newWords[:dex]

	dex = strings.Index(finalWord, " ")
	ret := finalWord[dex:]

	return strings.TrimSpace(ret)
}

func getStatus() (someBytes []byte, err os.Error) {
	buf := bytes.NewBufferString("")
	someBytes, err = bs.Post("/nodeStatus", buf)
	return
}

func findValues(searchTerm string, masterList string, userTerm string) (ret string) {
	var tmp string
	buf := bytes.NewBufferString(tmp)

	valueList := strings.Split(masterList, "\n")
	for _, value := range valueList {
		if len(value) > 0 {
			user := parse(searchTerm, value)
			if user == userTerm {
				buf.WriteString(value)
				buf.WriteString("\n")

			}
		}
	}
	ret = buf.String()
	return
}

func main() {
	flag.Parse()

	if help {
		cli.Usage()
		os.Exit(0)
	}

	someBytes, err := getStatus()
	if err != nil {
		cli.PrintError("Cannot find the status of nodes in heckle.", err)
		os.Exit(1)
	}

	if len(someBytes) <= 0 {
		cli.PrintError("Empty update", os.NewError("no data"))
		os.Exit(1)
	}

	if user != " " {
		validList := findValues("OWNER", string(someBytes), user)
		if len(validList) > 0 {
			printStatus(validList)
		} else {
			cli.PrintError("User does not exist in the system", os.NewError("Unknown user"))
			os.Exit(1)
		}

	}

	if node != " " {
		validList := findValues("NODE", string(someBytes), node)
		if len(validList) > 0 {
			printStatus(validList)
		} else {
			cli.PrintError("Node is not allocated or does not exsist", os.NewError("Missing"))
			os.Exit(1)
		}
	}

	if alloc != " " {
		validList := findValues("ALLOCATION", string(someBytes), alloc)
		if len(validList) > 0 {
			printStatus(validList)
		} else {
			cli.PrintError("Allocation number does not exsist", os.NewError("Allocation does not exsist"))
			os.Exit(1)
		}
	}

	if image != " " {
		validList := findValues("IMAGE", string(someBytes), image)
		if len(validList) > 0 {
			printStatus(validList)
		} else {
			cli.PrintError("Image does not exist.", os.NewError("No image"))
			os.Exit(1)
		}
	}

	if len(os.Args) < 2 {
		printStatus(string(someBytes))
	}

}
