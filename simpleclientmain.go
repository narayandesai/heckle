package main

import (
    "flag"   
    "fmt"
    "os"
    "./simpleclient"
)

var Usage = func() {
    fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
    //PrintDefaults()
}

var server string

func init() {
     flag.StringVar(&server, "S", "http://localhost:8081", "server base URL")
}

func main() {
    flag.Parse()
    fmt.Fprintf(os.Stderr, "Server is %s\n", server)
    bc := simpleclient.NewBuildClient(server)
    host := (*bc).GetHostname()
    os.Stdout.WriteString(host + "\n")
    data := (*bc).Get("dags")
    os.Stdout.WriteString(data + "\n")
}
