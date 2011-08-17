//The flunky master package provides a back end for the Heckle system 
// to interface with. The Flunky Master system provides tools to 
// render build environments as well as stores a minimal amount 
// of information of the build environment status. Flunky Master
// will also propogate different information back to requesting
// clients.

// BUG(Mike Guantonio): ipaddress resolution may be out of range and raises a painc if sent
// to the system. There needs to be painc error handling in order to fix this.  
// BUG(Mike Guantonio): Render static and dynamic do not allow for dynamic file names
package main

//Duties to finish
//7. Comment the code
//11. Add error returns to all functions
//15. Write documentation for new system.  
//23. Change the data.errors to and itoa function. 
//26. Add information to handle a heclke allocation (#) which can conatin information about 
// all nodes for that build request. 

import (
	"http"
	"os"
	"io/ioutil"
	"json"
	"time"
	"bytes"
	"flag"
	"strings"
	"github.com/ziutek/kasia.go"
	//"runtime"
	"net"
	"sync"
	"fmt"
	"flunky/daemon"
	"flunky/interfaces"
	"rand"
	//"encoding/base64"
)

var fm Flunkym
var m sync.Mutex
var fmDaemon *daemon.Daemon
var random *rand.Rand
var help bool

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(0)
}

//Bvar stores the build information for a node requesting a render.
type Bvar struct {
	Data   map[string]string
	Counts map[string]int
}

//PathType allows the user to set up the reference to where the data is 
// stored within the system. 
type PathType struct {
	root           string
	dataFile       string
	static         string
	dynamic        string
	staticdataPath string
	image          string
}

//DataStore is the main user database for all compute nodes that have
// connected to the Flunky Master system for build orders. 
type DataStore struct {
	Allocate int64
	Counts   map[string]int
	Errors   int
	Activity int64
	Info     []interfaces.InfoMsg
	Image    string
	Extra    map[string]string
	Username string
	Password string
	AllocNum uint64
}

//Flunkym is the main data type that will allow the user to interface and add
// onto the system. This data type has all of the interfaces that are needed
// for all functions. 
type Flunkym struct {
	path   *PathType
	data   map[string]*DataStore
	static map[string]string
}

func (fm *Flunkym) init() {
	var err os.Error
	flag.BoolVar(&help, "h", false, "Print usage message")
	flag.Parse()
	if help {
		Usage()
	}
	fmDaemon, err = daemon.New("flunky")
	if err != nil {
		fmt.Println("Could not create daemon")
		os.Exit(1)
	}
	user, pass, _ := fmDaemon.AuthN.GetUserAuth()
	err = fmDaemon.AuthN.Authenticate(user, pass, true)
	if err != nil {
		fmt.Println(fmt.Sprintf("You do not have proper permissions to start %s daemon.", fmDaemon.Name))
		os.Exit(1)
	}
	fm.data = make(map[string]*DataStore)
	fm.SetPath(fmDaemon)
	src := rand.NewSource(time.Seconds())
	random = rand.New(src)
	random.Seed(time.Seconds())
	fm.Load()
	return
}

func CreateCredin(len int) string {
	var rawCredin string
	var genNum int
	for i := 0; i < len; i++ {
		for {
			randNum := random.Intn(256)
			if (randNum > 47 && randNum < 58) || (randNum > 64 && randNum < 91) || (randNum > 96 && randNum < 123) {
				genNum = randNum
				break
			}
		}
		rawCredin = rawCredin + string(byte(genNum))
	}
	return rawCredin
}

func build_vars(address string, path string) map[string]Bvar {
	orders := make(map[string]string)
	data := make(map[string]Bvar)
	orders = fm.static
	orders["Address"] = address
	orders["Path"] = path
	orders["Count"] = "2"
	orders["IMAGE"] = fm.data[address].Image
	orders["Image"] = fm.data[address].Image
	orders["Username"] = fm.data[address].Username
	orders["Password"] = fm.data[address].Password
	key := data[address]
	key.Data = orders
	key.Counts = fm.data[address].Counts
	key.Counts["Errors"] = fm.data[address].Errors
	data[address] = key
	return data
}

//Mutex lock needed
func (fm *Flunkym) Assert_setup(image string, ip string, alloc uint64) {
	info := make([]interfaces.InfoMsg, 0)
	image_dir := fm.path.image + "/" + image
	_, err := os.Stat(image_dir)
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not find %s", image), err)
	usr := CreateCredin(8)
	pass := CreateCredin(8)
	fm.data[ip] = new(DataStore)
	fm.data[ip].Counts = make(map[string]int)
	fm.data[ip].Extra = make(map[string]string)
	fm.data[ip].Info = info
	fm.data[ip].Image = image
        fm.data[ip].Counts["bootconfig"] = 0
        fm.data[ip].Username = usr
	fm.data[ip].Password = pass
	fm.data[ip].AllocNum = alloc
	fm.Store()
	return
}

func (fm *Flunkym) Load() {
	_, err := os.Stat(fm.path.dataFile)
	if err != nil {
		data := make(map[string]*DataStore)
		fm.data = data
		fmDaemon.DaemonLog.Log("No previous data exsists. Data created")
	} else {
		fmDaemon.DaemonLog.LogDebug("Loading previous fm data")
		file, err := ioutil.ReadFile(fm.path.dataFile)
		fmDaemon.DaemonLog.LogError(fmt.Sprintf("Cannot read %s", fm.path.dataFile), err)

		if len(file) <= 0 {
			fmDaemon.DaemonLog.LogError(fmt.Sprintf("%s is an empty file. Creating new %s", fm.path.dataFile, fm.path.dataFile), os.NewError("Empty Json"))
			data := make(map[string]*DataStore)
			fm.data = data
		} else {
			err = json.Unmarshal(file, &fm.data)
			fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not unmarshall fm.data"), err)
			fmDaemon.DaemonLog.LogDebug("Data Loaded")
		}
	}
	file, err := ioutil.ReadFile(fm.path.staticdataPath)
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not read %s", fm.path.staticdataPath), err)

	err = json.Unmarshal(file, &fm.static)
	fmDaemon.DaemonLog.LogError("Could not read staticBuildVars.Json", err)
	return
}

//Mutex lock needed
func (fm *Flunkym) Store() {
	_, err := os.Stat(fm.path.dataFile)
	if err != nil {
		_, err := os.Create(fm.path.dataFile)
		fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not create %s", fm.path.dataFile), err)
	}

	backup, err := json.Marshal(fm.data)
	fmDaemon.DaemonLog.LogError("Could not marshall Data", err)

	err = ioutil.WriteFile(fm.path.dataFile, backup, 0666)
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not write fm.data to %s", fm.path.dataFile), err)
	return
}

func (fm *Flunkym) SetPath(fmDaemon *daemon.Daemon) {
	path := new(PathType)
	root := fmDaemon.Cfg.Data["repoPath"]
	path.root = root
	path.dataFile = daemon.FileDir + fmDaemon.Cfg.Data["backupFile"]
	path.staticdataPath = daemon.FileDir + "staticVars.json"
	path.image = path.root + "/images"
	fm.path = path
	return
}

func (fm *Flunkym) Increment_Count(address string, path string) {
	key := fm.data[address]
	key.Counts[path] += 1
	m.Lock()
	fm.data[address] = key
	m.Unlock()
	fm.Store()
	return
}

func (fm *Flunkym) RenderGetStatic(loc string, address string) []byte {
	fname := fm.path.root + loc
	_, err := os.Stat(fname)
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not find %s", fname), err)

	fm.Increment_Count(address, loc)
	contents, err := ioutil.ReadFile(fname)
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not read %s", fname), err)
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("%s Rendered %s", address, loc))
	return contents
}

func (fm *Flunkym) RenderGetDynamic(loc string, address string) []byte {
	var tmp []byte
	dynamic_buf := bytes.NewBuffer(tmp)
	bvar := build_vars(address, loc)
	fname := fm.path.root + loc
	_, err := os.Stat(fname)
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not find %s", fname), err)

	ans, err := ioutil.ReadFile(fname)
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not read %s", fname), err)

	tmpl, err := kasia.Parse(string(ans))
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Cannot Parse Template for %s", fname), err)

	err = tmpl.Run(dynamic_buf, bvar[address])
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Cannot execute template render for %s", fname), err)

	fm.Increment_Count(address, loc)

	dynamic := dynamic_buf.Bytes()
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("%s Rendered %s", address, loc))
	return dynamic
}

func (fm *Flunkym) RenderImage(toRender string, address string) (buf []byte) {
	l := bytes.NewBuffer(buf)
	bvar := build_vars(address, toRender)

	request := fm.path.image + "/" + fm.data[address].Image + "/" + toRender
	_, err := os.Stat(request)

	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Cannot find %s", request), err)
	ans, err := ioutil.ReadFile(request)
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Cannot read %s", request), err)

	tmpl, err := kasia.Parse(string(ans))
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Cannot parse template for %s", request), err)

	err = tmpl.Run(l, bvar[address])
	fmDaemon.DaemonLog.LogError("Cannot render template", err)

	fm.Increment_Count(address, toRender)
	v := l.Bytes()
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("%s: Rendered %s", address, toRender))
	return v
}

func (fm *Flunkym) DecodeRequest(req *http.Request, address string) (username string, authed bool, admin bool) {
        fmt.Println("IN decode request")
	header := req.Header.Get("Authorization")
	fmt.Println("Header", header)
	return
}

func DumpCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	err := fmDaemon.AuthN.HTTPAuthenticate(req, false)
	if err != nil {
		fmDaemon.DaemonLog.LogError("Permission denied", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	fmDaemon.UpdateActivity()
	m.Lock()
	tmp, err := json.Marshal(fm.data)
	m.Unlock()
	fmDaemon.DaemonLog.LogError("Cannot Marshal fm.data", err)
	_, err = w.Write(tmp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	fmDaemon.DaemonLog.Log(fmt.Sprintf("Processed data dump for %s", req.RemoteAddr))
}

func StaticCall(w http.ResponseWriter, req *http.Request) {
	var msg interfaces.InfoMsg
	fmDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	//Flunky auth needed
	fmDaemon.UpdateActivity()
	cmd := strings.Split(req.RawURL, "/")
	staticTemp := fm.RenderGetStatic(req.RawURL, address)
	w.Write(staticTemp)
	host, _ := net.LookupAddr(address)

	fm.data[address].Activity = time.Seconds()
	msg.Time = time.Seconds()
	msg.MsgType = "Info"
	msg.Message = fmt.Sprintf("%s is loading %s", host[:1], cmd)
	fm.data[address].Info = append(fm.data[address].Info, msg)
	fm.Store()
	fmDaemon.DaemonLog.Log(fmt.Sprintf("%s Rendered %s", cmd[1:], address))
}

func DynamicCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	cmd := strings.Split(req.RawURL, "/")
	//Flunky Auth needed
	fmDaemon.UpdateActivity()
	tmp := fm.RenderGetDynamic(req.RawURL, address)
	status := strings.TrimSpace(string(tmp))
	w.Write([]byte(status))
	fmDaemon.DaemonLog.Log(fmt.Sprintf("%s Rendered %s", cmd[1:], address))
}

func BootconfigCall(w http.ResponseWriter, req *http.Request) {
	var msg interfaces.InfoMsg
	fmDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	fmDaemon.UpdateActivity()
	cmd := strings.Split(req.RawURL, "/")
	imageTemp := fm.RenderImage(strings.TrimSpace(cmd[1]), address)
	_, err := w.Write(imageTemp)

	
	fm.data[address].Activity = time.Seconds()
	msg.Time = time.Seconds()
	msg.MsgType = "Info"
	host, _ := net.LookupAddr(address)
	msg.Message = fmt.Sprintf("%s is booting up", host[:1])
	fm.data[address].Info = append(fm.data[address].Info, msg)
	fm.Store()
	fmDaemon.DaemonLog.LogError("Will not write status", err)
	fmDaemon.DaemonLog.Log(fmt.Sprintf("%s in allocation #%d has booted.", address, fm.data[address].AllocNum))
}

func InstallCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	cmd := strings.Split(req.RawURL, "/")
	//Flunky auth needed
	fmDaemon.UpdateActivity()
	tmp := fm.RenderImage(strings.TrimSpace(cmd[1]), address)
	status := strings.TrimSpace(string(tmp))
	w.Write([]byte(status))
}

//Mutex needed
func InfoCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	fmDaemon.UpdateActivity()

	jsonType := fmDaemon.ProcessJson(req, new(interfaces.InfoMsg))
	msg := jsonType.(*interfaces.InfoMsg)
	fm.data[address].Activity = time.Seconds()
	msg.Time = time.Seconds()
	msg.MsgType = "Info"
	fm.data[address].Info = append(fm.data[address].Info, *msg)
	fm.Store()
	fmDaemon.DaemonLog.Log(fmt.Sprintf("Info recieved from %s: %s.", address, msg.Message))
	
}

//Mutex needed
func ErrorCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	//Flunky auth needed
	fmDaemon.UpdateActivity()
	jsonType := fmDaemon.ProcessJson(req, new(interfaces.InfoMsg))
	msg := jsonType.(*interfaces.InfoMsg)

	fm.data[address].Activity = time.Seconds()
	fm.data[address].Errors += 1
	msg.Time = time.Seconds()
	msg.MsgType = "Error"
	fm.data[address].Info = append(fm.data[address].Info, *msg)
	fm.Store()
	fmDaemon.DaemonLog.Log(fmt.Sprintf("Error recieved from %s: %s. Error count is %d", address, msg.Message, fm.data[address].Errors))
}

func CtrlCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	err := fmDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		fmDaemon.DaemonLog.LogError("Could not authenticate", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	temper, err := net.LookupIP(address)
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("Could not find %s in host tables", address))
	iaddr := temper[0].String()

	fmDaemon.UpdateActivity()
	jsonType := fmDaemon.ProcessJson(req, new(interfaces.Ctlmsg))
	msg := jsonType.(*interfaces.Ctlmsg)

	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("Received ctrl message from %s", iaddr))

	if len(msg.Addresses) == 0 {
		fmDaemon.DaemonLog.Log(fmt.Sprintf("Recieved empty update from %s. No action taken", address))
	} else {
		for _, addr := range msg.Addresses {
		go func (addr string){
			temper, err := net.LookupIP(addr)
			fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not find %s in host table", addr), err)
			iaddr := temper[0].String()
			fm.Assert_setup(msg.Image, iaddr, msg.AllocNum)
			fmDaemon.DaemonLog.Log(fmt.Sprintf("Allocating %s as %s for allocation #%d", addr, msg.Image, msg.AllocNum))
                }(addr)
		}

		fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("Added %s to flunkyMaster", msg.Addresses))
	}
}

func StatusCall(w http.ResponseWriter, req *http.Request) {
	cstatus := make(map[string]interfaces.StatusMessage)
	fmDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	err := fmDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		fmDaemon.DaemonLog.LogError("No access granted", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
        jsonType := fmDaemon.ProcessJson(req, new(interfaces.Ctlmsg))
	msg := jsonType.(*interfaces.Ctlmsg)

	cmd := strings.Split(req.RawURL, "/")
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("Recieved request for status from %s", address))
	for _, addr := range msg.Addresses {
		temper, err := net.LookupIP(addr)
		iaddr := temper[0].String()
		fmDaemon.DaemonLog.LogError("Could not find the ip addess in host tables", err)

		
		key := cstatus[addr]
		tmpl := fm.RenderImage(strings.TrimSpace(cmd[1]), iaddr)
		status := strings.TrimSpace(string(tmpl))
		key.Status = string(status)
		key.LastActivity = time.Seconds()

		for _, info := range fm.data[iaddr].Info {
			if info.Time > msg.Time {
				key.Info = append(key.Info, info)
			}
		}
		cstatus[addr] = key
	}
	ret, err := json.Marshal(cstatus)
	fmDaemon.DaemonLog.LogError("Could not Marsal status", err)
	w.Write(ret)
}

func daemonCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0

	err := fmDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		fmDaemon.DaemonLog.LogError("Access not permitted.", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	fmDaemon.UpdateActivity()
	stat := fmDaemon.ReturnStatus()

	status, err := json.Marshal(stat)
	if err != nil {
		fmDaemon.DaemonLog.LogError(err.String(), err)
	}
	w.Write(status)
	return
}

func main() {
	//runtime.GOMAXPROCS(4)
	fm.init()
	fm.Store()

	http.Handle("/daemon", http.HandlerFunc(daemonCall))
	http.Handle("/dump", http.HandlerFunc(DumpCall))
	http.Handle("/static/", http.HandlerFunc(StaticCall))
	http.Handle("/dynamic/", http.HandlerFunc(DynamicCall))
	http.Handle("/bootconfig", http.HandlerFunc(BootconfigCall))
	http.Handle("/install", http.HandlerFunc(InstallCall))
	http.Handle("/info", http.HandlerFunc(InfoCall))
	http.Handle("/error", http.HandlerFunc(ErrorCall))
	http.Handle("/ctl", http.HandlerFunc(CtrlCall))
	http.Handle("/status", http.HandlerFunc(StatusCall))

	fmDaemon.DaemonLog.Log(fmt.Sprintf("%s started on %s", fmDaemon.Name, fmDaemon.URL))
	err := fmDaemon.ListenAndServe()
	if err != nil {
		fmDaemon.DaemonLog.Log("Server exited gracefully. Cannot Listen on port")
	}

}
