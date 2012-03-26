package daemon

import (
	"encoding/json"
	"errors"
	"flag"
	iface "flunky/interfaces"
	fnet "flunky/net"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
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
	User      string
	Password  string
	stat      Status
	Comm      fnet.Communication
}

type Status struct {
	StartTime    int64
	UpTime       int64
	LastActivity int64
	Errors       int
}

func (daemon *Daemon) ProcessJson(req *http.Request, theType interface{}) (retType interface{}) {
	var tmp interface{}

	body, err := ioutil.ReadAll(req.Body)
	err = req.Body.Close()
	if err != nil {
		fmt.Println(err.Error())
	}

	switch reflect.TypeOf(theType) {
	case reflect.TypeOf(make([]string, 0)):
		tmp = theType.([]string)
		tmp = append(tmp.([]string), "Foo") //no empty list
		break
	case reflect.TypeOf(new(iface.InfoMsg)):
		tmp = theType.(*iface.InfoMsg)
		break
	case reflect.TypeOf(new(iface.Ctlmsg)):
		tmp = theType.(*iface.Ctlmsg)
		break
	case reflect.TypeOf(new(iface.Listmsg)):
		tmp = theType.(*iface.Listmsg)
		break
	case reflect.TypeOf(new(iface.Nummsg)):
		tmp = theType.(*iface.Nummsg)
		break
	case reflect.TypeOf(new(uint64)):
		tmp = theType.(*uint64)
		break
	case reflect.TypeOf(new(int64)):
		tmp = theType.(*int64)
		break
	case reflect.TypeOf(new(string)):
		tmp = theType.(*string)
		break
	}

	err = json.Unmarshal(body, &tmp)
	if err != nil {
		fmt.Println(err.Error())
	}

	retType = tmp
	return
}

func (daemon *Daemon) ReadRequest(req *http.Request) (body []byte, err error) {
	body, err = ioutil.ReadAll(req.Body)
	err = req.Body.Close()
	fmt.Println(string(body))
	return
}

func (daemon *Daemon) GetPort() (port string, err error) {
	parts := strings.Split(daemon.URL, ":")
	if len(parts) > 0 {
		port = parts[len(parts)-1]
		return
	}
	err = errors.New("Failed to parse URL")
	return
}

func (daemon *Daemon) ListenAndServe() (err error) {
	port, err := daemon.GetPort()
	if err != nil {
		fmt.Println("Port configuration error")
		os.Exit(1)
	}
	daemon.DaemonLog.Log(fmt.Sprintf("%s starting on %s", daemon.Name, daemon.URL))
	err = http.ListenAndServe(":"+port, nil)
	daemon.DaemonLog.LogError("Failed to listen on http socket.", err)
	return
}

func (daemon *Daemon) UpdateActivity() {
	daemon.stat.LastActivity = time.Now()
}

func (daemon *Daemon) ReturnStatus() Status {
	daemon.stat.UpTime = time.Now().Sub(daemon.stat.StartTime)
	daemon.stat.Errors = daemon.DaemonLog.ReturnError()
	return daemon.stat
}

func New(name string) (daemon *Daemon, err error) {
	daemon = new(Daemon)
	daemon.Name = name
	daemon.stat.StartTime = time.Now()

	daemon.DaemonLog = NewDaemonLogger("/var/log/", daemon.Name)
	daemon.Cfg = NewConfigInfo(FileDir+name+".conf", daemon.DaemonLog)
	daemon.AuthN = NewAuthInfo(FileDir+"users.db", daemon.DaemonLog)
	if user, ok := daemon.Cfg.Data["user"]; ok {
		daemon.User = user
	}

	if password, ok := daemon.Cfg.Data["password"]; ok {
		daemon.Password = password
	}

	daemon.Comm, err = fnet.NewCommunication(FileDir+"components.conf", daemon.User, daemon.Password)
	if err != nil {
		return
	}

	location, ok := daemon.Comm.Locations[name]
	if !ok {
		err = errors.New("Component lookup failure")
	}

	daemon.URL = location
	return
}
