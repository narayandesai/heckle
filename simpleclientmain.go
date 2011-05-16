package simpleclient

import (
    "os"
    "./simpleclient"
)

func main() {
    bc := simpleclient.NewBuildClient("fred")
    host := (*bc).GetHostname()
    os.Stdout.WriteString(host + "\n")
    data := (*bc).Get("dags")
    os.Stdout.WriteString(data + "\n")
}
