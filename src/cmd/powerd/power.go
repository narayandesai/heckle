package main

import (
	"fmt"
	"bytes"
	"http"
	"json"
	"flag"
	"os"
	"io/ioutil"
	"strings"
	"net"
	"sync"
	daemon "flunky/daemon"
)


type outletNode struct {
	Address string
	Outlet  string
}

type States struct {
	State  bool
	Reboot bool
}

type outletDB struct {
	outlets map[string]States
}


var resources map[string]outletNode
var powerDaemon *daemon.Daemon
var fileDir string
var outletStatus *outletDB
var m sync.Mutex

func returnState(info bool) (ret string) {
	if info {
		ret = "on"
	} else {
		ret = "off"
	}
	return
}

func (outletStatus *outletDB) checkValid(node string, op string) bool {
	if op == "on" && outletStatus.outlets[node].State {
		return false
	}
	if op == "off" && !outletStatus.outlets[node].State {
		return false
	}
	if op == "reboot" && outletStatus.outlets[node].Reboot {
		return false
	}
	return false
}

//There is alot of room for error if it comes back empty and if the data is not formatted correctly
//because of the index function
func (outletStatus *outletDB) returnStatus(status string, nodes []string) {
	powerDaemon.DaemonLog.LogDebug("Function: returnStatus")

	if len(nodes) == 0 {
		nodes = make([]string, 0)
		for key, _ := range resources {
			nodes = append(nodes, key)
		}
	}

	powerDaemon.DaemonLog.LogDebug(fmt.Sprintf("Functon: returnStatus -- Reading status %s", nodes))
	for _, node := range nodes {
		dex := strings.Index(status, resources[node].Outlet)
		first := status[dex:]

		dex = strings.Index(first, "\n")
		second := first[:dex]

		dex = strings.Index(second, "On")
		if dex < 0 {
			dex = strings.Index(second, "Off")
			if dex < 0 {
				powerDaemon.DaemonLog.LogError("Node has no status", os.NewError("Empty update"))
				return
			}
		}
		third := second[dex:]
		dex = strings.Index(third, " ")
		state := strings.TrimSpace((third[:dex]))

		m.Lock()
		powerDaemon.DaemonLog.LogDebug(fmt.Sprintf("Function :returnStatus -- Updating outletSatus %s", outletStatus))
		key := outletStatus.outlets[node]
		if state == "On" {
			key.State = true
		} else {
			key.State = false
		}
		reboot := strings.TrimSpace((third[dex:]))
		if reboot == "Reboot" {
			key.Reboot = true
			powerDaemon.DaemonLog.Log(fmt.Sprintf("%s has a pending reboot", node))
		} else {
			if key.Reboot {
				powerDaemon.DaemonLog.Log(fmt.Sprintf("%s's reboot complete the node is %s", node, returnState(key.State)))
			}
			key.Reboot = false
		}
		outletStatus.outlets[node] = key
		powerDaemon.DaemonLog.LogDebug(fmt.Sprintf("Function: Return status -- Finished update %s", outletStatus))
		m.Unlock()
	}

	return
}


func (outletStatus outletDB) dialServer(cmd string) (string, os.Error) {
	powerDaemon.DaemonLog.LogDebug(fmt.Sprintf("Function dialServer: -- command %s", cmd))

	byt := make([]byte, 82920)
	finalBuf := bytes.NewBuffer(make([]byte, 82920))
	cmdList := []string{"admn", "admn", cmd}

	//Set up negoations to the telnet server. Default is accept everything.
	powerDaemon.DaemonLog.LogDebug("Function dialServer: -- Set up telnet")
	k, err := net.Dial("tcp", "radix-pwr11:23")
	if err != nil {
		err = os.NewError("Cannot contact radix-pwr11 server")
		return "", err
	}
	ret := []byte{255, 251, 253}
	for i := 0; i < 5; i++ {
		_, err = k.Write(ret)
		if err != nil {
			err = os.NewError("Cannot write login info to socket")
		}
		k.Read(byt)
	}
	powerDaemon.DaemonLog.LogDebug("Function dialSever : -- setup complete")

	powerDaemon.DaemonLog.LogDebug(fmt.Sprintf("Function dialServer: -- Sending commands to server %s", cmdList))

	//All three for loops just send commands to the terminal
	for _, cmd := range cmdList {
		for {
			n, err := k.Read(byt)
			if err != nil {
				err = os.NewError("Cannot read from socket for terminal")
				return "", err
			}
			m := strings.Index(string(byt[:n]), ":")
			if m > 0 {
				_, err = k.Write([]byte(cmd + "\n"))
				if err != nil {
					err = os.NewError("Cannot write to socket")
					return "", err
				}
				break
			}
		}
		if cmd == "status" {
			break
		}
	}

	//See if the command is successful and then read the rest of the output.
	//keep in mind that it wont always be successful and therefore not read properly
	for {
		//k.SetReadTimeout(1000000*5) //if it can't be read break
		n, _ := k.Read(byt)
		m := strings.Index(string(byt[:n]), "successful")
		if m > 0 {
			powerDaemon.DaemonLog.LogDebug(fmt.Sprintf("Function dialServer: -- finalbuf\n %s", finalBuf.String()))
			break
		}
		finalBuf.Write(byt[:n])

	}
	if len(finalBuf.String()) <= 0 {
		err = os.NewError("Was not successful")
		return "", err
	}
	//Strip off the headers
	final := finalBuf.String()
	dex := strings.Index(final, "State")
	newFinal := final[dex:]
	dex = strings.Index(newFinal, "\n")
	powerDaemon.DaemonLog.LogDebug(fmt.Sprintf("Function dialServer: -- ripped off headers and closing connection %s", newFinal[dex:]))

	//close connection and return
	err = k.Close()
	if err != nil {
		err = os.NewError("Cannot close socket")
		return "", err
	}
	powerDaemon.DaemonLog.LogDebug("Function dialServer : -- Returning from function")
	return strings.TrimSpace(newFinal[dex:]), err
}

func printCmd(nodes []string, cmd string) {
	switch cmd {
	case "on":
		powerDaemon.DaemonLog.Log(fmt.Sprintf("Power outlet for %s is %s", nodes, cmd))
		break
	case "off":
		powerDaemon.DaemonLog.Log(fmt.Sprintf("Power outlet for %s is %s", nodes, cmd))
		break
	case "reboot":
		powerDaemon.DaemonLog.Log(fmt.Sprintf("Power outlet for %s is %sing", nodes, cmd))
		break
	}
	return
}

func DumpCall(w http.ResponseWriter, req *http.Request) {
	var empty []string
	powerDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	err := powerDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		powerDaemon.DaemonLog.LogError("Unauthorized request for dump.", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	powerDaemon.UpdateActivity()
	ret, _ := outletStatus.dialServer("status")
	outletStatus.returnStatus(ret, empty)
	tmp, err := json.Marshal(outletStatus.outlets)
	powerDaemon.DaemonLog.LogError("Cannot Marshal power resources", err)
	_, err = w.Write(tmp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	powerDaemon.DaemonLog.Log("Serviced request for data dump")
}


func command(w http.ResponseWriter, req *http.Request) {
	var nodes []string
	req.ProtoMinor = 0
	powerDaemon.DaemonLog.DebugHttp(req)
	err := powerDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		powerDaemon.DaemonLog.LogError("Access not permitted.", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	dex := strings.Split(req.RawURL, "/")
	cmd := dex[2]
	switch cmd {
	case "on", "off", "reboot":
		break
	default:
		powerDaemon.DaemonLog.LogError(fmt.Sprintf("%s command not supported", cmd), os.NewError("unsupported"))
		w.WriteHeader(http.StatusNotFound)
		return
		break
	}
	powerDaemon.UpdateActivity()
	body, err := powerDaemon.ReadRequest(req)
	powerDaemon.DaemonLog.LogError("Unable to read request", err)

	err = json.Unmarshal(body, &nodes)
	powerDaemon.DaemonLog.LogError(fmt.Sprintf("Unable to unmarshal nodes for %s command.", cmd), err)

	for _, node := range nodes {
		go func(node string) {
			_, err = outletStatus.dialServer(strings.TrimSpace(cmd + " " + resources[node].Outlet))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}(node)
	}
	printCmd(nodes, cmd)
}

func statusList(w http.ResponseWriter, req *http.Request) {
	retStatus := make(map[string]States)
	powerDaemon.DaemonLog.DebugHttp(req)
	powerDaemon.DaemonLog.LogDebug("Retreiving status for list given by client.")
	var nodes []string
	req.ProtoMinor = 0

	err := powerDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		powerDaemon.DaemonLog.LogError("Access not permitted.", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	dex := strings.Split(req.RawURL, "/")
	cmd := dex[1]
	body, err := powerDaemon.ReadRequest(req)
	powerDaemon.UpdateActivity()
	powerDaemon.DaemonLog.LogError("Could not read request", err)

	err = json.Unmarshal(body, &nodes)
	powerDaemon.DaemonLog.LogError("Unable to unmarshal nodes to be turned off.", err)

	status, err := outletStatus.dialServer(cmd)
	if err != nil {
		powerDaemon.DaemonLog.LogError(err.String(), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	outletStatus.returnStatus(status, nodes)

	var jsonStat []byte
	if len(nodes) == 0 {
		jsonStat, err = json.Marshal(outletStatus.outlets)
		powerDaemon.DaemonLog.LogError("Unable to marshal outlet status response.", err)
	} else {
		for _, node := range nodes {
			retStatus[node] = outletStatus.outlets[node]
		}
		jsonStat, err = json.Marshal(retStatus)
		powerDaemon.DaemonLog.LogError("Unable to marshal outlet status response.", err)
	}
	
	_, err = w.Write(jsonStat)
	powerDaemon.DaemonLog.LogError("Unable to write outlet status response.", err)

	return
}

func daemonCall(w http.ResponseWriter, req *http.Request) {
	powerDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0

	err := powerDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		powerDaemon.DaemonLog.LogError("Access not permitted.", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	powerDaemon.UpdateActivity()
	stat := powerDaemon.ReturnStatus()

	status, err := json.Marshal(stat)
	if err != nil {
		powerDaemon.DaemonLog.LogError(err.String(), err)
	}
	w.Write(status)
	return

}

func main() {

	flag.Parse()
	var err os.Error

	powerDaemon, err = daemon.New("power")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	user, pass, _ := powerDaemon.AuthN.GetUserAuth()
	err = powerDaemon.AuthN.Authenticate(user, pass, true)
	if err != nil {
		fmt.Println(fmt.Sprintf("You dont have permissions to start %s daemon.", powerDaemon.Name))
		os.Exit(1)
	}

	outletStatus = new(outletDB)
	outletStatus.outlets = make(map[string]States)

	powerDB, err := ioutil.ReadFile(daemon.FileDir + "power.db")
	powerDaemon.DaemonLog.LogError("Unable to open power.db for reading.", err)

	err = json.Unmarshal(powerDB, &resources)
	powerDaemon.DaemonLog.LogError("Failed to unmarshal data read from power.db file.", err)

	http.HandleFunc("/daemon", daemonCall)
	http.HandleFunc("/dump", DumpCall)
	http.HandleFunc("/command/", command)
	http.HandleFunc("/status", statusList)
	powerDaemon.DaemonLog.Log(fmt.Sprintf("%s started on %s", powerDaemon.Name, powerDaemon.URL))
	err = powerDaemon.ListenAndServe()
	if err != nil {
		os.Exit(1)
	}
}
