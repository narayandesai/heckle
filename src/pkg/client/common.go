package client

import (
	"flag"
	"os"
	"fmt"
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()

}

func PrintError(message string, err os.Error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", message)
	}
	return
}
