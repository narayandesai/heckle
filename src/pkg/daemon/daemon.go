package daemon

import (
	"reflect"
	"time"
	"flag"
	"fmt"
	"http"
	"os"
	"io/ioutil"
	"strings"
	"json"
	fnet "flunky/net"
	iface "flunky/interfaces"
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
		fmt.Println(err.String())
	}

	switch reflect.TypeOf(theType) {
	case reflect.TypeOf(make([]string, 0)):
	       	tmp = theType.([]string)
		tmp = append(tmp.([]string), "Foo")//no empty list
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
	if err != nil{
	   fmt.Println(err.String())
        }
	
	retType = tmp
	return
}

func (daemon *Daemon) ReadRequest(req *http.Request) (body []byte, err os.Error) {
	body, err = ioutil.ReadAll(req.Body)
	err = req.Body.Close()
	fmt.Println(string(body))
	return
}

func (daemon *Daemon) GetPort() (port string, err os.Error) {
	parts := strings.Split(daemon.URL, ":")
	if len(parts) > 0 {
		port = parts[len(parts)-1]
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
	err = http.ListenAndServe(":"+port, nil)
	daemon.DaemonLog.LogError("Failed to listen on http socket.", err)
	daemon.DaemonLog.Log(fmt.Sprintf("%s started on %s", daemon.Name, daemon.URL))
	return
}

func (daemon *Daemon) UpdateActivity() {
	daemon.stat.LastActivity = time.Seconds()
}

func (daemon *Daemon) ReturnStatus() Status {
	daemon.stat.UpTime = time.Seconds() - daemon.stat.StartTime
	daemon.stat.Errors = daemon.DaemonLog.ReturnError()
	return daemon.stat
}

func New(name string) (daemon *Daemon, err os.Error) {
	daemon = new(Daemon)
	daemon.Name = name
	daemon.stat.StartTime = time.Seconds()

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
		err = os.NewError("Component lookup failure")
	}

	daemon.URL = location
	return
}
