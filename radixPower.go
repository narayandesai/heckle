package main

import (
     "http"
     "json"
     //"sync"
     "fmt"
     "exec"
     "os"
     "io/ioutil"
     "strings"
)

type outletNode struct{
     Address   string
     Outlet    string
}

var resources       map[string]outletNode
var radixOPcfg      map[string]string
//var resourcesLock   sync.Mutex   shouldn't need a lock, never changing data.

func init() {
     radixDBFile, error := os.Open("radixOP.db")
     printError("ERROR: Unable to open radixOP.db for reading.", error)
     
     someBytes, error := ioutil.ReadAll(radixDBFile)
     printError("ERROR: Unable to read from file radixOP.db.", error)
     
     error = radixDBFile.Close()
     printError("ERROR: Failed to close radixOP.db.", error)
     
     error = json.Unmarshal(someBytes, &resources)
     printError("ERROR: Failed to unmarshal data read from radixOP.db file.", error)
     
     radixCFGFile, error := os.Open("radixOP.cfg")
     printError("ERROR: Unable to open radixOP.cfg for reading.", error)
     
     someBytes, error = ioutil.ReadAll(radixCFGFile)
     printError("ERROR: Unable to read from file radixOP.cfg.", error)
     
     error = radixCFGFile.Close()
     printError("ERROR: Failed to close radixOP.cfg.", error)
     
     error = json.Unmarshal(someBytes, &radixOPcfg)
     printError("ERROR: Failed to unmarshal data read from radixOP.cfg file.", error)
}

func printError(errorMsg string, error os.Error) {
     //This function prints the error passed if error is not nil.
     if error != nil {
          fmt.Fprintf(os.Stderr, "%s\n", errorMsg)
     }
}

func rebootList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     var nodes []string
     request.ProtoMinor = 0
     
     someBytes, error := ioutil.ReadAll(request.Body)
     printError("ERROR: Unable to read all from reboot POST.", error)
     
     error = request.Body.Close()
     printError("ERROR: Failed to close reboot request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     printError("ERROR: Unable to unmarshal nodes to be rebooted.", error)
     
     for _, value := range nodes {
          go func() {
               tmpCmd, error := exec.Run("radixOP.sh", []string{"radixOP.sh", resources[value].Address, "admn", "admn", "reboot", resources[value].Outlet}, os.Environ(), "", exec.PassThrough, exec.DevNull, exec.PassThrough)
               printError("ERROR: Failed to run radixOP.sh.", error)
               
               _, error = tmpCmd.Wait(0)
               printError("ERROR: Failed to wait for radixOP.sh to finish.", error)
               
               error = tmpCmd.Close()
               printError("ERROR: Failed to close radixOP.sh cmd.", error)
          }()
     }
}

func offList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     var nodes []string
     request.ProtoMinor = 0
     
     someBytes, error := ioutil.ReadAll(request.Body)
     printError("ERROR: Unable to read all from off POST.", error)
     
     error = request.Body.Close()
     printError("ERROR: Failed to close off request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     printError("ERROR: Unable to unmarshal nodes to be turned off.", error)
     
     for _, value := range nodes {
          go func() {
               tmpCmd, error := exec.Run("radixOP.sh", []string{"radixOP.sh", resources[value].Address, "admn", "admn", "off", resources[value].Outlet}, os.Environ(), "", exec.PassThrough, exec.DevNull, exec.PassThrough)
               printError("ERROR: Failed to run radixOP.sh.", error)
               
               _, error = tmpCmd.Wait(0)
               printError("ERROR: Failed to wait for radixOP.sh to finish.", error)
               
               error = tmpCmd.Close()
               printError("ERROR: Failed to close radixOP.sh cmd.", error)
          }()
     }
}

func statusList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     var nodes []string
     outletStatus := make(map[string]string)
     request.ProtoMinor = 0
     
     someBytes, error := ioutil.ReadAll(request.Body)
     printError("ERROR: Unable to read all from off POST.", error)
     
     error = request.Body.Close()
     printError("ERROR: Failed to close off request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     printError("ERROR: Unable to unmarshal nodes to be turned off.", error)
     
     for _, value := range nodes {
          _, ok := outletStatus[value]
          
          if !ok {
               tmpCmd, error := exec.Run("radixOP.sh", []string{"radixOP.sh", resources[value].Address, "admn", "admn", "status"}, os.Environ(), "", exec.PassThrough, exec.Pipe, exec.PassThrough)
               printError("ERROR: Failed to run radixOP.sh.", error)

               _, error = tmpCmd.Wait(0)
               printError("ERROR: Failed to wait for radixOP.sh to finish.", error)

               someBytes, error = ioutil.ReadAll(tmpCmd.Stdout)
               printError("ERROR: Failed to read all from outlet status pipe.", error)

               error = tmpCmd.Close()
               printError("ERROR: Failed to close radixOP.sh cmd.", error)

               tmpStatusLines := strings.Split(string(someBytes), "\n", -1)

               for i := 18 ; i < 42 ; i++ {
                    tmpStatusFields := strings.Split(tmpStatusLines[i], " ", -1)

                    for _, value2 := range nodes {
                         if resources[value2].Address == resources[value].Address && resources[value2].Outlet == tmpStatusFields[3] {
                              outletStatus[value2] = tmpStatusFields[13]
                         }
                    }
               }
          }
     }

     jsonStat, error := json.Marshal(outletStatus)
     printError("ERROR: Unable to marshal outlet status response.", error)
     
     _, error = writer.Write(jsonStat)
     printError("ERROR: Unable to write outlet status response.", error)
}

func main() {
     http.HandleFunc("/reboot", rebootList)
     http.HandleFunc("/off", offList)
     http.HandleFunc("/status", statusList)
     //http.HandleFunc("/status", statusList)
     
     error := http.ListenAndServe(":" + radixOPcfg["radixOPPort"], nil)
     printError("ERROR: Failed to listen on http socket.", error)
}