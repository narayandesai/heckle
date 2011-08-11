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
	daemon "flunky/daemon"
)


type outletNode struct {
	Address string
	Outlet  string
}

type States struct{
     State bool
     Reboot bool
}

type outletDB struct {
     outlets map[string]States
}

var resources map[string]outletNode
var powerDaemon *daemon.Daemon
var fileDir string
var outletStatus *outletDB

func returnState(info bool)(ret string){
     if info{
       ret = "on"
     }else{
       ret = "off"
    }
     return
}

func (outletStatus *outletDB) checkValid(node string, op string) bool{
     if op == "on" && outletStatus.outlets[node].State{
         return false
     }
     if op == "off" && !outletStatus.outlets[node].State{
         return false
     }
     if op == "reboot" && outletStatus.outlets[node].Reboot{
         return false
     }
     return false
}

//Add in error handling for the function
func (outletStatus *outletDB) returnStatus(status string, nodes []string){

        if nodes[0] == "*"{
	    nodes = nodes[1:]
	    for key, _ := range(resources){
	    	nodes = append(nodes, key)
            }
        }

	for _, node := range nodes {
		dex := strings.Index(status, resources[node].Outlet)
		first := status[dex:]

		dex = strings.Index(first, "\n")
		second := first[:dex]

		dex = strings.Index(second, "On")
		if dex < 0 {
			dex = strings.Index(second, "Off")
			if dex < 0 {
				fmt.Println("Node has no status")
				return
			}
		}
		third := second[dex:]
		dex = strings.Index(third, " ")

		state := strings.TrimSpace((third[:dex]))
		key := outletStatus.outlets[node]
		if state == "On"{
		    key.State = true
                }else{
		    key.State = false
 		}
		outletStatus.outlets[node] = key
		
		reboot := strings.TrimSpace((third[dex:]))
		if reboot == "Reboot" {
		    key.Reboot = true
		    powerDaemon.DaemonLog.Log(fmt.Sprintf("%s has a pending reboot", node))
                }else{
		    if key.Reboot {
		        powerDaemon.DaemonLog.Log(fmt.Sprintf("%s's reboot complete the node is %s", node, returnState(key.State)))
	            }
		        key.Reboot = false
		}
		 outletStatus.outlets[node] = key
		   

	}
	return
}

//Add in error call back
func (outletStatus outletDB) dialServer(cmd string) string {
	byt := make([]byte, 82920)
	//buf := bytes.NewBufferString("admn\n")
	finalBuf := bytes.NewBuffer(make([]byte, 82920))
        
        cmdList := []string{"admn", "admn", cmd}

	//Set up negoations to the telnet server. Default is accept everything.
	k, _ := net.Dial("tcp", "radix-pwr11:23")
	ret := []byte{255, 251, 253}
	for i := 0; i < 5; i++ {
		k.Write(ret)
		k.Read(byt)
	}

	//All three for loops just send commands to the terminal
        for _, cmd := range(cmdList){
	    for {
			n, _ := k.Read(byt)
			m := strings.Index(string(byt[:n]), ":")
			if m > 0 {
			   k.Write([]byte(cmd + "\n"))
			   break
			   }
	       }
	       if cmd == "status"{break}
	}

	//See if the command is successful and then read the rest of the output.
	for {
		n, _ := k.Read(byt)
		m := strings.Index(string(byt[:n]), "successful")
		if m > 0 {
			break
		}
		finalBuf.Write(byt[:n])

	}

	//Strip off the headers
	final := finalBuf.String()
	dex := strings.Index(final, "State")
	newFinal := final[dex:]
	dex = strings.Index(newFinal, "\n")

	//close connection and return
	k.Close()
	return strings.TrimSpace(newFinal[dex:])
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
	powerDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	err := powerDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		powerDaemon.DaemonLog.LogError("Unauthorized request for dump.", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	tmp, err := json.Marshal(outletStatus)
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
	case "on" , "off", "reboot":
		break
	default:
		powerDaemon.DaemonLog.LogError(fmt.Sprintf("%s command not supported", cmd), os.NewError("unsupported"))
		w.WriteHeader(http.StatusNotFound)
		return
		break
	}
	body, err := powerDaemon.ReadRequest(req)
	powerDaemon.DaemonLog.LogError("Unable to read request", err)

	err = json.Unmarshal(body, &nodes)
	powerDaemon.DaemonLog.LogError(fmt.Sprintf("Unable to unmarshal nodes for %s command.", cmd), err)
	fmt.Println(nodes)
	for _, node := range(nodes){
	    if outletStatus.checkValid(node, cmd){
               outletStatus.dialServer(cmd + " " + resources[node].Outlet +"\n")
	    }else{
	       powerDaemon.DaemonLog.LogError(fmt.Sprintf("%s is already %s", node, cmd), os.NewError("Pre-exsisting state"))
	    }
	}
	printCmd(nodes, cmd)

        ret := outletStatus.dialServer("status") //not optimal
	outletStatus.returnStatus(ret, nodes)
	buf, err := json.Marshal(outletStatus)
	if err != nil{
	   fmt.Println(err)
	}
        _, err = w.Write(buf)
	if err != nil {fmt.Println(err)}
}


func statusList(w http.ResponseWriter, req *http.Request) {
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
	powerDaemon.DaemonLog.LogError("Could not read request", err)

	err = json.Unmarshal(body, &nodes)
	powerDaemon.DaemonLog.LogError("Unable to unmarshal nodes to be turned off.", err)

	status := outletStatus.dialServer(cmd)
	outletStatus.returnStatus(status, nodes)

	jsonStat, err := json.Marshal(outletStatus)
	powerDaemon.DaemonLog.LogError("Unable to marshal outlet status response.", err)

	_, err = w.Write(jsonStat)
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
	user, pass, _ := powerDaemon.AuthN.GetUserAuth()
	err = powerDaemon.AuthN.Authenticate(user, pass, true)
	if err != nil {
		fmt.Println(fmt.Sprintf("You dont have permissions to start %s daemon.", powerDaemon.Name))
		os.Exit(1)
	}

        outletStatus = new(outletDB)
	powerDB, err := ioutil.ReadFile(daemon.FileDir + "power.db")
	powerDaemon.DaemonLog.LogError("Unable to open power.db for reading.", err)

	err = json.Unmarshal(powerDB, &resources)
	powerDaemon.DaemonLog.LogError("Failed to unmarshal data read from power.db file.", err)

	for key, node := range(resources){
	fmt.Println(key, node)
	}

	http.HandleFunc("/dump", DumpCall)
	http.HandleFunc("/command/", command)
	http.HandleFunc("/status", statusList)
	powerDaemon.DaemonLog.Log(fmt.Sprintf("%s started on %s", powerDaemon.Name, powerDaemon.URL))
	err = powerDaemon.ListenAndServe()
	if err != nil {
		os.Exit(1)
	}
}
