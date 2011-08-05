//The flunky master package provides a back end for the Heckle system 
// to interface with. The Flunky Master system provides tools to 
// render build environments as well as stores a minimal amount 
// of information of the build environment status. Flunky Master
// will also propogate different information back to requesting
// clients.

// BUG(Mike Guantonio): ipaddress resolution may be out of range and raises a painc if sent
// to the system. There needs to be painc error handling in order to fix this. 
// BUG(Mike Guantonio): There is not reporting to std out for the HTTTP status of a message.
// BUG(Mike Guantonio): Errors currently print as ascii characters and not integers.  
// BUG(Mike Guantonio): Render static and dynamic do not allow for dynamic file names
// BUG(Miek Guantonio): Authorization information is not pulled from flunky.
package main

//Duties to finish
//3. Implement go routines
//7. Comment the code
//11. Add error returns to all functions
//12. Implement select statuments
//15. Write documentation for new system. 
//21. Overload the http handler func in order to to make fm not local. 
//22. Find out if GET and POST are really important.
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
	"encoding/base64"
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
	//AllocNum string
}

//Flunkym is the main data type that will allow the user to interface and add
// onto the system. This data type has all of the interfaces that are needed
// for all functions. 
type Flunkym struct {
	path   PathType
	data   map[string]DataStore
	static map[string]string
}

func (fm *Flunkym) init() {
        var err os.Error
        flag.BoolVar(&help, "h", false, "Print usage message")
        flag.Parse()
        if help {
	    Usage()
        }
	fmDaemon, err = daemon.New("flunkymaster")
	if err != nil {
	  fmt.Println("Could not create daemon")
          os.Exit(1)
        }
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
	orders["Errors"] = string(fm.data[address].Errors) //itoa function needed
	orders["Username"] = fm.data[address].Username
	orders["Password"] = fm.data[address].Password
	key := data[address]
	key.Data = orders
	key.Counts = fm.data[address].Counts
	data[address] = key
	return data
}

//Mutex lock needed
func (fm *Flunkym) Assert_setup(image string, ip string) {
	info := make([]interfaces.InfoMsg, 0)
	image_dir := fm.path.image + "/" + image
	_, err := os.Stat(image_dir)
	fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not find %s", image), err)
	usr := CreateCredin(8)
	pass := CreateCredin(8)
	newsetup := make(map[string]DataStore)
	counts := make(map[string]int)
	newsetup[ip] = DataStore{time.Seconds(), counts, 0, time.Seconds(), info, image, nil, "", ""}
	newsetup[ip].Counts["bootconfig"] = 0
	key := newsetup[ip]
	key.Username = usr
	key.Password = pass
	newsetup[ip] = key
	//newsetup[ip].AllocateNum = msg.AllocateNum)
	fm.data[ip] = newsetup[ip]
	fm.Store()
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("Allocated %s as %s", ip, image))
	return
}

func (fm *Flunkym) Load() {
	_, err := os.Stat(fm.path.dataFile)
	if err != nil {
		data := make(map[string]DataStore)
		fm.data = data
		fmDaemon.DaemonLog.Log("No previous data exsists. Data created")
	} else {
		fmDaemon.DaemonLog.LogDebug("Loading previous fm data")
		file, err := ioutil.ReadFile(fm.path.dataFile)
		fmDaemon.DaemonLog.LogError(fmt.Sprintf("Cannot read %s", fm.path.dataFile), err)

		if len(file) <= 0 {
			fmDaemon.DaemonLog.LogError(fmt.Sprintf("%s is an empty file. Creating new %s", fm.path.dataFile, fm.path.dataFile), os.NewError("Empty Json"))
			data := make(map[string]DataStore)
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
	fmDaemon.DaemonLog.LogError("Could not staticBuildVars.Json", err)
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
	path.dataFile = path.root + "/" + fmDaemon.Cfg.Data["backupFile"]
	path.staticdataPath = path.root + "/staticVars.json"
	path.image = path.root + "/images"
	fm.path = *path
	return
}

//Mutex
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
	fmDaemon.DaemonLog.Log(fmt.Sprintf("%s Rendered %s", address, loc))
	return dynamic
}

func (fm *Flunkym) RenderImage(toRender string, address string) (buf []byte) {
	key := fm.data[address]
	l := bytes.NewBuffer(buf)
	bvar := build_vars(address, toRender)
	request := fm.path.image + "/" + key.Image + "/" + toRender
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

func (fm *Flunkym)DecodeRequest(req *http.Request, address string) (username string, authed bool, admin bool){
    if _, ok := req.Header["Authorization"]; !ok {
        fmDaemon.DaemonLog.LogError("Request header did not contain authorization information", os.NewError("Http Auth missing"))
       return
     }
     header := req.Header.Get("Authorization")
     tmpCredins := strings.Split(header, " ")

     credins, error := base64.URLEncoding.DecodeString(tmpCredins[1])
     fmDaemon.DaemonLog.LogError("Failed to decode encoded auth setting in http request", error)
      
      userCredin := strings.Split(string(credins), ":")
      username = userCredin[0]
      password := userCredin[1]
      authed, admin = fm.AuthFlunky(username, password, address)
      return
}

func (fm *Flunkym)AuthFlunky(user string, password string, address string)(valid bool, admin bool){
   if user !=  fm.data[address].Username {
       return false, false
   }  
   valid = (password == fm.data[address].Password)
   admin = true
   return
}

func DumpCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.LogHttp(req)
	req.ProtoMinor = 0
        err := fmDaemon.AuthN.HTTPAuthenticate(req, false)
	if err != nil{
            fmDaemon.DaemonLog.LogError("Permission denied", err)
	    w.WriteHeader(http.StatusUnauthorized)
	    return
        }
	m.Lock()
	tmp, err := json.Marshal(fm.data)
	m.Unlock()
	fmDaemon.DaemonLog.LogError("Cannot Marshal fm.data", err)
	_, err = w.Write(tmp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	fmDaemon.DaemonLog.LogDebug("Data Dump")
}

func StaticCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.LogHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
        //Flunky auth needed
	tmp := fm.RenderGetStatic(req.RawURL, address)
	w.Write(tmp)
}

func DynamicCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.LogHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
        //Flunky Auth needed
	tmp := fm.RenderGetDynamic(req.RawURL, address)
	status := strings.TrimSpace(string(tmp))
	w.Write([]byte(status))

}

func BootconfigCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.LogHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	//Flunky auth needed
	tmp := fm.RenderImage("bootconfig", address) // allow for "name", "data[image]
	_, err := w.Write(tmp)
	fmDaemon.DaemonLog.LogError("Will not write status", err)
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("%s Rendered bootconfig", address))
}

func InstallCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.LogHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	//Flunky auth needed
	tmp := fm.RenderImage("install", address)
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("%s Rendered install", address))
	status := strings.TrimSpace(string(tmp))
	w.Write([]byte(status))
}

//Mutex needed
func InfoCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.LogHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	//Flunky auth needed
	var tmp DataStore
	body, _ := fmDaemon.ReadRequest(req)
	fmDaemon.DaemonLog.LogDebug("Received Info")
	var msg interfaces.InfoMsg
	err := json.Unmarshal(body, &msg)
	if err != nil{
	   fmDaemon.DaemonLog.LogError("Could not unmarshall data", err)
	   w.WriteHeader(http.StatusInternalServerError)
        }else{
	tmp = fm.data[address]
	tmp.Activity = time.Seconds()
	msg.Time = time.Seconds()
	msg.MsgType = "Info"
	tmp.Info = append(tmp.Info, msg)
	m.Lock()
	fm.data[address] = tmp
	m.Unlock()
	fm.Store()
	}
}

//Mutex needed
func ErrorCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.LogHttp(req)
	req.ProtoMinor = 0
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":")
	address := addTmp[0]
	//Flunky auth needed
	var tmp DataStore
	body, _ := fmDaemon.ReadRequest(req)
	m.Lock()
	fmDaemon.DaemonLog.LogDebug("Recieved error")
	m.Unlock()
	var msg interfaces.InfoMsg
	err := json.Unmarshal(body, &msg)
	fmDaemon.DaemonLog.LogError("Cannot unmarsahll data", err)
	tmp = fm.data[address]
	tmp.Activity = time.Seconds()
	tmp.Errors += 1
	msg.Time = time.Seconds()
	msg.MsgType = "Error"
	tmp.Info = append(tmp.Info, msg)
	m.Lock()
	fm.data[address] = tmp
	m.Unlock()
	fm.Store()
}

func CtrlCall(w http.ResponseWriter, req *http.Request) {
	fmDaemon.DaemonLog.LogHttp(req)
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

	body, _ := fmDaemon.ReadRequest(req)
	temper, err := net.LookupIP(address)
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("Could not find %s in host tables", address))
	iaddr := temper[0].String()
	var msg interfaces.Ctlmsg
	
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("Received ctrl message from %s", iaddr))
	
	err = json.Unmarshal(body, &msg)
	fmDaemon.DaemonLog.LogError("Could not unmarshall data", err)
	if len(msg.Addresses) == 0 {
		fmDaemon.DaemonLog.Log(fmt.Sprintf("Recieved empty update from %s. No action taken", address))
	} else {
		for _, addr := range msg.Addresses {
			temper, err := net.LookupIP(addr)
			fmDaemon.DaemonLog.LogError(fmt.Sprintf("Could not find %s in host table", addr), err)
			iaddr := temper[0].String()
			fmDaemon.DaemonLog.Log(fmt.Sprintf("Allocating %s as %s", addr, msg.Image))
			fm.Assert_setup(msg.Image, iaddr)
		}

		fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("Added %s to flunkyMaster", msg.Addresses))
	}
}

//Mutex needed
func StatusCall(w http.ResponseWriter, req *http.Request) {
	var msg interfaces.Ctlmsg
	cstatus := make(map[string]interfaces.StatusMessage)
	fmDaemon.DaemonLog.LogHttp(req)
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
	body, _ := fmDaemon.ReadRequest(req)
	err = json.Unmarshal(body, &msg)
	fmDaemon.DaemonLog.LogError("Could not unmarshall message", err)
       
	fmDaemon.DaemonLog.LogDebug(fmt.Sprintf("Recieved request for status from %s", address))
	for _, addr := range msg.Addresses {
		temper, err := net.LookupIP(addr)
		iaddr := temper[0].String()
		fmDaemon.DaemonLog.LogError("Could not find the ip addess in host tables", err)

		tmp := fm.data[iaddr]
		key := cstatus[addr]
		tmpl := fm.RenderImage("status", iaddr)
		status := strings.TrimSpace(string(tmpl))
		key.Status = string(status)
		key.LastActivity = time.Seconds()

		for _, info := range tmp.Info {
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

/*func makeHandler(fn  func(w http.ResponseWriter, r *http.Request, f chan Flunkym))http.HandlerFunc{
     return func (w http.ResponseWriter, r *http.Request){
     tmp := <- f
     fn(w, r, tmp)
     }
}*/


func main() {
	//runtime.GOMAXPROCS(4)
	fm.init()
	fm.Store()
	
	http.Handle("/dump", http.HandlerFunc(DumpCall))
	http.Handle("/static/", http.HandlerFunc(StaticCall))
	http.Handle("/dynamic", http.HandlerFunc(DynamicCall))
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
