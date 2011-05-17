package main

import (
    "flag"   
    "fmt"
    "os"
    "./flunky"
)

var Usage = func() {
    fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
    flag.PrintDefaults()
}

var server string
var verbose bool
var help bool

func init() {
     flag.BoolVar(&help, "h", false, "print usage")
     flag.BoolVar(&verbose, "v", false, "print debug information")
     flag.StringVar(&server, "S", "http://localhost:8081", "server base URL")
}

func main() {
    flag.Parse()
    if help {
        Usage()
        os.Exit(0)
       }
    if verbose {
        fmt.Fprintf(os.Stderr, "Server is %s\n", server)
        }       

    bs := flunky.NewBuildServer(server)
    data, _ := bs.Run("foo")

    fmt.Fprintf(os.Stderr, "response is :%s:\n", data)
    //bc := simpleclient.NewBuildClient(server)
    //    host := (*bc).GetHostname()
    //data := (*bc).Get("dags")
    //os.Stdout.WriteString(data + "\n")
}
