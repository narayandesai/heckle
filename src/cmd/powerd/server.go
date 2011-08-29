package main

import (
	"fmt"
	"http"
	"io/ioutil"
	"json"
	"os"
	"strings"
	daemon "flunky/daemon"
)

var powerDaemon *daemon.Daemon

type ControllerMuxServer struct {
	cm ControllerMux
}

func NewControllerMuxServer() (cms ControllerMuxServer) {
	cms.cm = NewControllerMux()
	return
}

func (cms ControllerMuxServer) wrapStatus(w http.ResponseWriter, req *http.Request) {
	powerDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0

	err := powerDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		powerDaemon.DaemonLog.LogError("Access not permitted.", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := ioutil.ReadAll(req.Body)
	err = req.Body.Close()
	if err != nil {
		fmt.Println(err.String())
	}

	var nodes []string

	err = json.Unmarshal(body, &nodes)
	if err != nil{
		fmt.Println(err.String())
    }

	status, err := cms.cm.Status(nodes)

	if err != nil {
		powerDaemon.DaemonLog.LogError(err.String(), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonStat, err := json.Marshal(status)
	powerDaemon.DaemonLog.LogError("Unable to marshal outlet status response.", err)
	
	_, err = w.Write(jsonStat)
	powerDaemon.DaemonLog.LogError("Unable to write outlet status response.", err)

	return
}

func (cms ControllerMuxServer) wrapControl(w http.ResponseWriter, req *http.Request) {
	powerDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0

	err := powerDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		powerDaemon.DaemonLog.LogError("Access not permitted.", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	cmd := strings.Split(req.RawURL, "/")[2]

	switch cmd {
	case "on", "off", "reboot":
		break
	default:
		powerDaemon.DaemonLog.LogError(fmt.Sprintf("%s command not supported", cmd), 
			os.NewError("dummy"))
		w.WriteHeader(http.StatusNotFound)
		return
		break
	}

	body, err := ioutil.ReadAll(req.Body)
	err = req.Body.Close()
	if err != nil {
		fmt.Println(err.String())
	}

	var nodes []string

	err = json.Unmarshal(body, &nodes)
	if err != nil{
		fmt.Println(err.String())
    }

	powerDaemon.DaemonLog.Log(fmt.Sprintf("%s: %s", cmd, nodes))

	switch cmd {
	case "on":
		err = cms.cm.On(nodes)
	case "off":
		err = cms.cm.Off(nodes)
	case "reboot":
		err = cms.cm.Reboot(nodes)
	}

	if err != nil {
		powerDaemon.DaemonLog.LogError(err.String(), err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	return
}
func main() {
	var err os.Error

	powerDaemon, err = daemon.New("power")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cms := NewControllerMuxServer()
	err  = cms.cm.LoadSentryFromFile("/etc/heckle/power-sentry.db")
	if (err != nil) {
		fmt.Println("Failed to load database")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(cms.cm.Controllers)

	http.HandleFunc("/status", func (w http.ResponseWriter, req *http.Request) {cms.wrapStatus(w, req)})
	http.HandleFunc("/command/", func (w http.ResponseWriter, req *http.Request) {cms.wrapControl(w, req)})
	powerDaemon.DaemonLog.Log(fmt.Sprintf("%s started on %s", powerDaemon.Name, powerDaemon.URL))
	err = powerDaemon.ListenAndServe()
	if err != nil {
		os.Exit(1)
	}
}
