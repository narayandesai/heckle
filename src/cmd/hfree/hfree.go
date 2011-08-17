package main

import (
	"fmt"
	"flag"
	"os"
	//"json"
	//"bytes"
	"strconv"
	fnet "flunky/net"
	cli "flunky/client"
)

var help bool

func init() {
	flag.BoolVar(&help, "h", false, "Print usage of command.")
}

func freeAlloc(alloc int64) (err os.Error) {
	bs := new(fnet.BuildServer)
	allocfree, err := cli.NewClient()
	cli.PrintError("Failed to create a new client", err)
	bs, err = allocfree.SetupClient("heckle")
	cli.PrintError("Falied to setup heckle as a client", err)
	
	_, err = bs.PostServer("/freeAllocation", alloc)
	if err != nil{
	   return
        }

	return
}

func main() {
	flag.Parse()

	if help {
		cli.Usage()
		os.Exit(0)
	}

	if len(flag.Args()) <= 0 {
		cli.Usage()
		os.Exit(0)
	}

	allocations := flag.Args()
	alloc, _ := strconv.Atoi64(allocations[0])

	err := freeAlloc(alloc)
	if err != nil {
		cli.PrintError(fmt.Sprintf("Unable to free allocation #%d. Allocation does not exsist.", alloc), err)
		os.Exit(1)
	} else {
		fmt.Println(fmt.Sprintf("Freed allocation #%d.", alloc))
	}

}
