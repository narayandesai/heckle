package main

import (
	"encoding/json"
	"errors"
	daemon "flunky/daemon"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
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
		fmt.Println(err.Error())
	}

	var nodes []string

	err = json.Unmarshal(body, &nodes)
	if err != nil {
		fmt.Println(err.Error())
	}

	status, err := cms.cm.Status(nodes)

	if err != nil {
		powerDaemon.DaemonLog.LogError(err.Error(), err)
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
			errors.New("dummy"))
		w.WriteHeader(http.StatusNotFound)
		return
		break
	}

	body, err := ioutil.ReadAll(req.Body)
	err = req.Body.Close()
	if err != nil {
		fmt.Println(err.Error())
	}

	var nodes []string

	err = json.Unmarshal(body, &nodes)
	if err != nil {
		fmt.Println(err.Error())
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
		powerDaemon.DaemonLog.LogError(err.Error(), err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	return
}
func main() {
	var err error

	powerDaemon, err = daemon.New("power")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cms := NewControllerMuxServer()
	err = cms.cm.LoadSentryFromFile("/etc/heckle/power-sentry.db")
	if err != nil {
		fmt.Println("Failed to load sentry database")
		fmt.Println(err)
		os.Exit(1)
	}

	err = cms.cm.LoadIpmiFromFile("/etc/heckle/power-ipmi.db")
	if err != nil {
		fmt.Println("Failed to load ipmi database")
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc("/status", func(w http.ResponseWriter, req *http.Request) { cms.wrapStatus(w, req) })
	http.HandleFunc("/command/", func(w http.ResponseWriter, req *http.Request) { cms.wrapControl(w, req) })
	err = powerDaemon.ListenAndServe()
	if err != nil {
		os.Exit(1)
	}
}
