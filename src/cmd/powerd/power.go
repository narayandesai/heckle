package main

import (
	"fmt"
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


type outletNode struct {
	Address string
	Outlet  string
}

var resources map[string]outletNode
var powerDaemon *daemon.Daemon
var fileDir string

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

func rebootList(writer http.ResponseWriter, req *http.Request) {
	powerDaemon.DaemonLog.LogHttp(req)
	powerDaemon.DaemonLog.Log("Rebooting list given by client.")
	var nodes []string
	req.ProtoMinor = 0
	_, authed, admin := powerDaemon.AuthN.HTTPAuthenticate(req)

	if !authed {
		powerDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
		return
	}

	if !admin {
		powerDaemon.DaemonLog.LogError("No access to admin command.", os.NewError("Access Denied"))
		return
	}

	body, err := powerDaemon.ReadRequest(req)
	powerDaemon.DaemonLog.LogError("Unable to ready request", err)

	err = json.Unmarshal(body, &nodes)
	powerDaemon.DaemonLog.LogError("Unable to unmarshal nodes to be rebooted.", err)

	for _, value := range nodes {
		if _, ok := resources[value]; ok {
			go func(value string) {
				err = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "reboot", resources[value].Outlet).Run()
				powerDaemon.DaemonLog.LogError("Failed to run powerCont.sh in rebootList.", err)
			}(value)
		}
	}
}

func offList(writer http.ResponseWriter, req *http.Request) {
	powerDaemon.DaemonLog.LogHttp(req)
	powerDaemon.DaemonLog.Log(fmt.Sprintf("Proceeding to %s nodes given by client.", req.RawURL))
	var nodes []string
	req.ProtoMinor = 0

	_, authed, admin := powerDaemon.AuthN.HTTPAuthenticate(req)

	if !authed {
		powerDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
		return
	}

	if !admin {
		powerDaemon.DaemonLog.LogError("No access to admin command.", os.NewError("Access Denied"))
		return
	}

	body, err := powerDaemon.ReadRequest(req)
	powerDaemon.DaemonLog.LogError("Unable to ready request", err)

	err = json.Unmarshal(body, &nodes)
	powerDaemon.DaemonLog.LogError("Unable to unmarshal nodes to be turned off.", err)

	for _, value := range nodes {
		if _, ok := resources[value]; ok {
			go func(value string) {
				err = exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "off", resources[value].Outlet).Run()
				powerDaemon.DaemonLog.LogError("Failed to run powerCont.sh in offList.", err)
			}(value)
		}
	}
}

func statusList(writer http.ResponseWriter, req *http.Request) {
	powerDaemon.DaemonLog.LogHttp(req)
	powerDaemon.DaemonLog.Log("Retreiving status for list given by client.")
	var nodes []string
	outletStatus := make(map[string]string)
	req.ProtoMinor = 0

	_, authed, admin := powerDaemon.AuthN.HTTPAuthenticate(req)

	if !authed {
		powerDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
		return
	}

	if !admin {
		powerDaemon.DaemonLog.LogError("No access to admin command.", os.NewError("Access Denied"))
		return
	}

	body, err := powerDaemon.ReadRequest(req)
	powerDaemon.DaemonLog.LogError("Could not ready request", err)

	err = json.Unmarshal(body, &nodes)
	powerDaemon.DaemonLog.LogError("Unable to unmarshal nodes to be turned off.", err)

	for _, value := range nodes {
		_, ok := outletStatus[value]
		_, ok2 := resources[value]

		if !ok && ok2 {
			someBytes, err := exec.Command("./powerCont.sh", resources[value].Address, "admn", "admn", "status").Output()
			powerDaemon.DaemonLog.LogError("Failed to execute powerCont.sh and get out put in power status request.", err)

			tmpStatusLines := strings.Split(string(someBytes), "\n")

			for i := 18; i < 42; i++ {
				tmpStatusFields := strings.Split(tmpStatusLines[i], " ")

				for _, value2 := range nodes {
					if _, ok3 := resources[value2]; ok3 && ok2 {
						if resources[value2].Address == resources[value].Address && resources[value2].Outlet == tmpStatusFields[3] {
							outletStatus[value2] = tmpStatusFields[13]
						}
					}
				}
			}
		}
	}

	jsonStat, err := json.Marshal(outletStatus)
	powerDaemon.DaemonLog.LogError("Unable to marshal outlet status response.", err)

	_, err = writer.Write(jsonStat)
	powerDaemon.DaemonLog.LogError("Unable to write outlet status response.", err)
}

func main() {
	flag.Parse()
	var err os.Error

	powerDaemon, err = daemon.New("power")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	powerDB, err := ioutil.ReadFile(daemon.FileDir + "power.db")
	powerDaemon.DaemonLog.LogError("Unable to open power.db for reading.", err)

	err = json.Unmarshal(powerDB, &resources)
	powerDaemon.DaemonLog.LogError("Failed to unmarshal data read from power.db file.", err)

	http.HandleFunc("/dump", DumpCall)
	http.HandleFunc("/reboot", rebootList)
	http.HandleFunc("/off", offList)
	http.HandleFunc("/status", statusList)
	powerDaemon.DaemonLog.Log(fmt.Sprintf("%s started on %s", powerDaemon.Name, powerDaemon.URL))
	err = powerDaemon.ListenAndServe()
	if err != nil {
		os.Exit(1)
	}
}
