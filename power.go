package main

import (
     "http"
     "json"
     //"sync"
     "exec"
     "os"
     "io/ioutil"
     "strings"
     "encoding/base64"
     "./heckleTypes"
     "./heckleFuncs"
)



var resources  map[string]heckleTypes.OutletNode
var powerCFG   map[string]string
var auth       map[string]heckleTypes.UserNode
//var resourcesLock   sync.Mutex   shouldn't need a lock, never changing data.

func init() {
     powerDBFile, error := os.Open("powerCont.db")
     heckleFuncs.PrintError("ERROR: Unable to open powerCont.db for reading.", error)
     
     someBytes, error := ioutil.ReadAll(powerDBFile)
     heckleFuncs.PrintError("ERROR: Unable to read from file powerCont.db.", error)
     
     error = powerDBFile.Close()
     heckleFuncs.PrintError("ERROR: Failed to close powerCont.db.", error)
     
     error = json.Unmarshal(someBytes, &resources)
     heckleFuncs.PrintError("ERROR: Failed to unmarshal data read from powerCont.db file.", error)
     
     powerCFGFile, error := os.Open("powerCont.cfg")
     heckleFuncs.PrintError("ERROR: Unable to open powerCont.cfg for reading.", error)
     
     someBytes, error = ioutil.ReadAll(powerCFGFile)
     heckleFuncs.PrintError("ERROR: Unable to read from file powerCont.cfg.", error)
     
     error = powerCFGFile.Close()
     heckleFuncs.PrintError("ERROR: Failed to close powerCont.cfg.", error)
     
     error = json.Unmarshal(someBytes, &powerCFG)
     heckleFuncs.PrintError("ERROR: Failed to unmarshal data read from powerCont.cfg file.", error)
     
     authFile, error := os.Open("PowerUserDatabase")
     heckleFuncs.PrintError("ERROR: Unable to open PowerUserDatabase for reading.", error)
     
     someBytes, error = ioutil.ReadAll(authFile)
     heckleFuncs.PrintError("ERROR: Unable to read from file PowerUserDatabase.", error)
     
     error = authFile.Close()
     heckleFuncs.PrintError("ERROR: Failed to close PowerUserDatabase.", error)
     
     error = json.Unmarshal(someBytes, &auth)
     heckleFuncs.PrintError("ERROR: Failed to unmarshal data read from PowerUserDatabase file.", error)
}

func decode(tmpAuth string) (username string, password string) {
     tmpAuthArray := strings.Split(tmpAuth, " ")
     
     authValues , error := base64.StdEncoding.DecodeString(tmpAuthArray[1])
     heckleFuncs.PrintError("ERROR: Failed to decode encoded auth settings in http request.", error)
     
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
     
     _, authed, admin := heckleFuncs.Authenticate(request.Header.Get("Authorization"), powerCFG["userDatabasePath"])
     
     if !authed {
          heckleFuncs.PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !admin {
          heckleFuncs.PrintError("ERROR: No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleFuncs.PrintError("ERROR: Unable to read all from reboot POST.", error)
     
     error = request.Body.Close()
     heckleFuncs.PrintError("ERROR: Failed to close reboot request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     heckleFuncs.PrintError("ERROR: Unable to unmarshal nodes to be rebooted.", error)
     
     for _, value := range nodes {
          go func(value string) {
               error = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "reboot", resources[value].Outlet).Run()
               heckleFuncs.PrintError("ERROR: Failed to run powerCont.sh in rebootList.", error)
          }(value)
     }
}

func offList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     var nodes []string
     request.ProtoMinor = 0
     
     _, authed, admin := heckleFuncs.Authenticate(request.Header.Get("Authorization"), powerCFG["userDatabasePath"])
     
     if !authed {
          heckleFuncs.PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !admin {
          heckleFuncs.PrintError("ERROR: No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleFuncs.PrintError("ERROR: Unable to read all from off POST.", error)
     
     error = request.Body.Close()
     heckleFuncs.PrintError("ERROR: Failed to close off request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     heckleFuncs.PrintError("ERROR: Unable to unmarshal nodes to be turned off.", error)
     
     for _, value := range nodes {
          go func(value string) {
               error = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "off", resources[value].Outlet).Run()
               heckleFuncs.PrintError("ERROR: Failed to run powerCont.sh in offList.", error)
          }(value)
     }
}

func statusList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     var nodes []string
     outletStatus := make(map[string]string)
     request.ProtoMinor = 0
     
     _, authed, admin := heckleFuncs.Authenticate(request.Header.Get("Authorization"), powerCFG["userDatabasePath"])
     
     if !authed {
          heckleFuncs.PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !admin {
          heckleFuncs.PrintError("ERROR: No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleFuncs.PrintError("ERROR: Unable to read all from off POST.", error)
     
     error = request.Body.Close()
     heckleFuncs.PrintError("ERROR: Failed to close off request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     heckleFuncs.PrintError("ERROR: Unable to unmarshal nodes to be turned off.", error)
     
     for _, value := range nodes {
          _, ok := outletStatus[value]
          
          if !ok {
               someBytes, error = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "status").Output()
               heckleFuncs.PrintError("ERROR: Failed to execute powerCont.sh and get out put in power status request.", error)

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
     heckleFuncs.PrintError("ERROR: Unable to marshal outlet status response.", error)
     
     _, error = writer.Write(jsonStat)
     heckleFuncs.PrintError("ERROR: Unable to write outlet status response.", error)
}

func main() {
     http.HandleFunc("/reboot", rebootList)
     http.HandleFunc("/off", offList)
     http.HandleFunc("/status", statusList)
     
     error := http.ListenAndServe(":" + powerCFG["powerPort"], nil)
     heckleFuncs.PrintError("ERROR: Failed to listen on http socket.", error)
}