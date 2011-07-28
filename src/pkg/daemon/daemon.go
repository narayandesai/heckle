package daemon

import (
	"flag"
	"fmt"
	"http"
	"os"
	"strings"
	fnet "flunky/net"
)

var FileDir string

func init() {
	flag.StringVar(&FileDir, "F", "/etc/heckle/", "Directory where daemon files can be found.")
}

type Daemon struct {
	Name      string
	DaemonLog *DaemonLogger
	AuthN     *Authinfo
	Cfg       *ConfigInfo
	URL       string
	User        string
	Password string
	Comm     fnet.Communication
}

func (daemon *Daemon) GetPort() (port string, err os.Error) {
	parts := strings.Split(daemon.URL, ":")
	if len(parts) > 0 {
		port = parts[len(parts)]
		return
	}
	err = os.NewError("Failed to parse URL")
	return
}

func (daemon *Daemon) ListenAndServe() (err os.Error) {
	port, err := daemon.GetPort()
	if err != nil {
		fmt.Println("Port configuration error")
		os.Exit(1)
	}
    err = http.ListenAndServe(port, nil)
    daemon.DaemonLog.LogError("Failed to listen on http socket.", err)
	return
}

func New(name string) (daemon *Daemon, err os.Error) {
	daemon.Name = name
	daemon.DaemonLog = NewDaemonLogger(FileDir, daemon.Name)
	daemon.Cfg = NewConfigInfo(FileDir+name+".conf", daemon.DaemonLog)
	daemon.AuthN = NewAuthInfo("/etc/heckle/users.db", daemon.DaemonLog)
	if user, ok := daemon.Cfg.Data["user"]; ok {
		daemon.User = user
	}

	if password, ok := daemon.Cfg.Data["password"]; ok {
		daemon.Password = password
	}

	daemon.Comm, err = fnet.NewCommunication("/etc/heckle/components.conf", daemon.User, daemon.Password)
	if err != nil {
		return
	}
	
	location, ok := daemon.Comm.Locations[name]
	if !ok {
		err = os.NewError("Component lookup failure")
	}

	daemon.URL = location
	return 
}
