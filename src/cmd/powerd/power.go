package main

import (
     "http"
     "json"
     "flag"
     "exec"
     "os"
     "io/ioutil"
     "strings"
	//iface "flunky/interfaces"
	daemon "flunky/daemon"
)


type outletNode struct{
     Address   string
     Outlet    string
}

var resources       map[string]outletNode
var powerDaemon     *daemon.Daemon
var fileDir         string
//var resourcesLock   sync.Mutex   shouldn't need a lock, never changing data.

func init() {
     flag.Parse()
     
     powerDaemon = daemon.New("power")
     
     powerDaemon.DaemonLog.Log("Initializting data for daemon setup.")
     
     powerDBFile, error := os.Open(daemon.FileDir + "power.db")
     powerDaemon.DaemonLog.LogError("Unable to open power.db for reading.", error)
     
     someBytes, error := ioutil.ReadAll(powerDBFile)
     powerDaemon.DaemonLog.LogError("Unable to read from file power.db.", error)
     
     error = powerDBFile.Close()
     powerDaemon.DaemonLog.LogError("Failed to close power.db.", error)
     
     error = json.Unmarshal(someBytes, &resources)
     powerDaemon.DaemonLog.LogError("Failed to unmarshal data read from power.db file.", error)
}


func DumpCall(w http.ResponseWriter, req *http.Request) {
        powerDaemon.DaemonLog.LogHttp(req)
        req.ProtoMinor = 0
       /* username, authed, _ := powerDaemon.AuthN.HTTPAuthenticate(req)
        if !authed {
                powerDaemon.DaemonLog.LogError(fmt.Sprintf("User Authentications for %s failed", username), os.NewError("Access Denied"))
                return
        }*/
        tmp, err := json.Marshal(resources)
        powerDaemon.DaemonLog.LogError("Cannot Marshal power resources", err)
        _, err = w.Write(tmp)
        if err != nil {
                http.Error(w, "Cannot write to socket", 500)
        }
}

func rebootList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     powerDaemon.DaemonLog.Log("Rebooting list given by client.")
     var nodes []string
     request.ProtoMinor = 0
     
     _, authed, admin := powerDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          powerDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !admin {
          powerDaemon.DaemonLog.LogError("No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     powerDaemon.DaemonLog.LogError("Unable to read all from reboot POST.", error)
     
     error = request.Body.Close()
     powerDaemon.DaemonLog.LogError("Failed to close reboot request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     powerDaemon.DaemonLog.LogError("Unable to unmarshal nodes to be rebooted.", error)
     
     for _, value := range nodes {
          if _, ok := resources[value] ; ok {
               go func(value string) {
                    error = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "reboot", resources[value].Outlet).Run()
                    powerDaemon.DaemonLog.LogError("Failed to run powerCont.sh in rebootList.", error)
               }(value)
          }
     }
}

func offList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     powerDaemon.DaemonLog.Log("Turning off list of nodes given by client.")
     var nodes []string
     request.ProtoMinor = 0
     
     _, authed, admin := powerDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          powerDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !admin {
          powerDaemon.DaemonLog.LogError("No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     powerDaemon.DaemonLog.LogError("Unable to read all from off POST.", error)
     
     error = request.Body.Close()
     powerDaemon.DaemonLog.LogError("Failed to close off request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     powerDaemon.DaemonLog.LogError("Unable to unmarshal nodes to be turned off.", error)
     
     for _, value := range nodes {
          if _, ok := resources[value] ; ok {
               go func(value string) {
                    error = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "off", resources[value].Outlet).Run()
                    powerDaemon.DaemonLog.LogError("Failed to run powerCont.sh in offList.", error)
               }(value)
          }
     }
}

func statusList(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     powerDaemon.DaemonLog.Log("Retreiving status for list given by client.")
     var nodes []string
     outletStatus := make(map[string]string)
     request.ProtoMinor = 0
     
     _, authed, admin := powerDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          powerDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !admin {
          powerDaemon.DaemonLog.LogError("No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     powerDaemon.DaemonLog.LogError("Unable to read all from off POST.", error)
     
     error = request.Body.Close()
     powerDaemon.DaemonLog.LogError("Failed to close off request body.", error)
     
     error = json.Unmarshal(someBytes, &nodes)
     powerDaemon.DaemonLog.LogError("Unable to unmarshal nodes to be turned off.", error)
     
     for _, value := range nodes {
          _, ok := outletStatus[value]
          _, ok2 := resources[value]
          
          if !ok && ok2 {
               someBytes, error = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "status").Output()
               powerDaemon.DaemonLog.LogError("Failed to execute powerCont.sh and get out put in power status request.", error)

               tmpStatusLines := strings.Split(string(someBytes), "\n")

               for i := 18 ; i < 42 ; i++ {
                    tmpStatusFields := strings.Split(tmpStatusLines[i], " ")

                    for _, value2 := range nodes {
                         if _, ok3 := resources[value2] ; ok3 && ok2 {
                              if resources[value2].Address == resources[value].Address && resources[value2].Outlet == tmpStatusFields[3] {
                                   outletStatus[value2] = tmpStatusFields[13]
                              }
                         }
                    }
               }
          }
     }

     jsonStat, error := json.Marshal(outletStatus)
     powerDaemon.DaemonLog.LogError("Unable to marshal outlet status response.", error)
     
     _, error = writer.Write(jsonStat)
     powerDaemon.DaemonLog.LogError("Unable to write outlet status response.", error)
}

func main() {
     http.HandleFunc("/dump", DumpCall)
     http.HandleFunc("/reboot", rebootList)
     http.HandleFunc("/off", offList)
     http.HandleFunc("/status", statusList)
     
     error := http.ListenAndServe(":" + powerDaemon.Cfg.Data["powerPort"], nil)
     powerDaemon.DaemonLog.LogError("Failed to listen on http socket.", error)
}