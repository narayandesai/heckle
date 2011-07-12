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
     "encoding/base64"
)

type outletNode struct{
     Address   string
     Outlet    string
}

type userNode struct {
     Password  string
     Admin     bool
}

var resources  map[string]outletNode
var powerCFG   map[string]string
var auth       map[string]userNode
//var resourcesLock   sync.Mutex   shouldn't need a lock, never changing data.

func init() {
     powerDBFile, error := os.Open("powerCont.db")
     printError("ERROR: Unable to open powerCont.db for reading.", error)
     
     someBytes, error := ioutil.ReadAll(powerDBFile)
     printError("ERROR: Unable to read from file powerCont.db.", error)
     
     error = powerDBFile.Close()
     printError("ERROR: Failed to close powerCont.db.", error)
     
     error = json.Unmarshal(someBytes, &resources)
     printError("ERROR: Failed to unmarshal data read from powerCont.db file.", error)
     
     powerCFGFile, error := os.Open("powerCont.cfg")
     printError("ERROR: Unable to open powerCont.cfg for reading.", error)
     
     someBytes, error = ioutil.ReadAll(powerCFGFile)
     printError("ERROR: Unable to read from file powerCont.cfg.", error)
     
     error = powerCFGFile.Close()
     printError("ERROR: Failed to close powerCont.cfg.", error)
     
     error = json.Unmarshal(someBytes, &powerCFG)
     printError("ERROR: Failed to unmarshal data read from powerCont.cfg file.", error)
     
     authFile, error := os.Open("PowerUserDatabase")
     printError("ERROR: Unable to open PowerUserDatabase for reading.", error)
     
     someBytes, error = ioutil.ReadAll(authFile)
     printError("ERROR: Unable to read from file PowerUserDatabase.", error)
     
     error = authFile.Close()
     printError("ERROR: Failed to close PowerUserDatabase.", error)
     
     error = json.Unmarshal(someBytes, &auth)
     printError("ERROR: Failed to unmarshal data read from PowerUserDatabase file.", error)
}

func printError(errorMsg string, error os.Error) {
     //This function prints the error passed if error is not nil.
     if error != nil {
          fmt.Fprintf(os.Stderr, "%s\n", errorMsg)
     }
}

func decode(tmpAuth string) (username string, password string) {
     tmpAuthArray := strings.Split(tmpAuth, " ")
     
     authValues , error := base64.StdEncoding.DecodeString(tmpAuthArray[1])
     printError("ERROR: Failed to decode encoded auth settings in http request.", error)
     
     authValuesArray := strings.Split(string(authValues), ":")
     username = authValuesArray[0]
     password = authValuesArray[1]
     
     return
}

func rebootList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     var nodes []string
     request.ProtoMinor = 0
     
     username, password := decode(request.Header.Get("Authorization"))
     
     if password != auth[username].Password {
          printError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !auth[username].Admin {
          printError("ERROR: No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     printError("ERROR: Unable to read all from reboot POST.", error)
     
     error = request.Body.Close()
     printError("ERROR: Failed to close reboot request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     printError("ERROR: Unable to unmarshal nodes to be rebooted.", error)
     
     for _, value := range nodes {
          go func(value string) {
               error = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "reboot", resources[value].Outlet).Run()
               printError("ERROR: Failed to run powerCont.sh in rebootList.", error)
          }(value)
     }
}

func offList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     var nodes []string
     request.ProtoMinor = 0
     
     username, password := decode(request.Header.Get("Authorization"))
     
     if password != auth[username].Password {
          printError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !auth[username].Admin {
          printError("ERROR: No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     printError("ERROR: Unable to read all from off POST.", error)
     
     error = request.Body.Close()
     printError("ERROR: Failed to close off request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     printError("ERROR: Unable to unmarshal nodes to be turned off.", error)
     
     for _, value := range nodes {
          go func(value string) {
               error = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "off", resources[value].Outlet).Run()
               printError("ERROR: Failed to run powerCont.sh in offList.", error)
          }(value)
     }
}

func statusList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     var nodes []string
     outletStatus := make(map[string]string)
     request.ProtoMinor = 0
     
     username, password := decode(request.Header.Get("Authorization"))
     
     if password != auth[username].Password {
          printError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !auth[username].Admin {
          printError("ERROR: No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     printError("ERROR: Unable to read all from off POST.", error)
     
     error = request.Body.Close()
     printError("ERROR: Failed to close off request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     printError("ERROR: Unable to unmarshal nodes to be turned off.", error)
     
     for _, value := range nodes {
          _, ok := outletStatus[value]
          
          if !ok {
               someBytes, error = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "status").Output()
               printError("ERROR: Failed to execute powerCont.sh and get out put in power status request.", error)

               tmpStatusLines := strings.Split(string(someBytes), "\n")

               for i := 18 ; i < 42 ; i++ {
                    tmpStatusFields := strings.Split(tmpStatusLines[i], " ")

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
     
     error := http.ListenAndServe(":" + powerCFG["powerPort"], nil)
     printError("ERROR: Failed to listen on http socket.", error)
}