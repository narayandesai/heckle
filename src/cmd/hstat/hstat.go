package main

import (
	"fmt"
	"flag"
	"os"
	"bytes"
	"strings"
	//"time"
	fnet "flunky/net"
	cli "flunky/client"
)

var help, status bool
var bs *fnet.BuildServer
var hstat fnet.Communication

func init() {
	var err os.Error
	if hstat, err = cli.NewClient(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get new client.\n")
		os.Exit(1)
	}

	if bs, err = hstat.SetupClient("heckle"); err != nil {
		printError("Failed to setup client in halloc.", os.NewError("Client Setup Failed"))
		os.Exit(1)
	}

	flag.BoolVar(&help, "h", false, "Print usage of command.")

}

func printError(message string, err os.Error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", message)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func printStatus(info string){
    fmt.Println("ID\t USER\t\t NODES\t\t OWNED UNTIL\t\t\t\t IMAGE")
    id := parse("ALLOCATION", info)
    user := parse("OWNER", info)
    image := parse("IMAGE", info)
    nodes := parse("NODE", info)
    timeEnd   := parse("END:", info)
    fmt.Println(fmt.Sprintf("%s\t %s\t %s\t\t %s\t\t %s", id, user,nodes, timeEnd, image))    
}


func parse(word string, words string) string{
    dex := strings.Index(words, word)
    newWords := words[dex:]

    dex = strings.Index(newWords, "\t")
    finalWord := newWords[:dex]

    dex = strings.Index(finalWord, " ")
    ret := finalWord[dex:]

    return strings.TrimSpace(ret) 
    
}
func getStatus()(someBytes []byte, err os.Error) {
	buf := bytes.NewBufferString("")
	someBytes, error := bs.Post("/nodeStatus", buf)
	printError("Failed to post the request for node status to heckle.", error)
	return
}

func main() {
	flag.Parse()

	if help {
		usage()
		os.Exit(0)
	}
        someBytes, _ := getStatus()

	printStatus(string(someBytes))
}