package client

import (
	"flag"
	"fmt"
	"os"
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()

}

func PrintError(message string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", message)
	}
	return
}
