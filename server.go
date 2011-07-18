//The flunky master package provides a back end for the Heckle system 
// to interface with. The Flunky Master system provides tools to 
// render build environments as well as stores a minimal amount 
// of information of the build environment status. Flunky Master
// will also propogate different information back to requesting
// clients.
// BUG(Mike Guantonio): ipaddress resolution may be out of range and raises a painc if sent
// to the system. There needs to be painc error handling in order to fix this. 
//BUG(Mike Guantonio): Currently the Flunky Master system does not support Http Auth.
//BUG(Mike Guantonio): There is not reporting to std out for the HTTTP status of a message.
//BUG(Mike Guantonio): Errors currently print as ascii characters and not integers. 
//BUG(Mike Guantonio): Http errors are not handled. 
package server

//Duties to finish
//3. Implement go routines
//4. Clean code up.
//4b. Use the defer
//7. Comment the code
//8. Conform to the golang style doc
//9. Interfaces for each class
//11. Add error returns to all functions
//12. Implement sync package and select statuments
//13. Find a way to make fm not a global var.
//15. Write documentation for new system. 
//21. Overload the http handler func in order to to make fm not local. 
//22. Find out if GET and POST are really important.
//23. Change the data.errors to and itoa function.
//24. Write exception if a file becomes damaged for reading in data.json.
//25. Create a flunky master config file. 
//26. Add information to handle a heclke allocation (#) which can conatin information about 
// all nodes for that build request. 
// 27. Future: add information for the build number that is internal to the system for log messages. 


import (
	"http"
	"os"
	"io/ioutil"
	"json"
	"time"
	"bytes"
	"strings"
	"log"
	"github.com/ziutek/kasia.go"
	"runtime"
	"net"
	//"encoding/base64"
	"sync"
)

var fm Flunkym
var m sync.Mutex
var auth map[string]userNode


//User node is a data type that holds a user's authtcation credintials. 
type userNode struct {
	Password string
	Admin    bool
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
	static         string
	dynamic        string
	dataFile       string
	staticdataPath string
	log            string
	image          string
}

//infoMsg store information for any information that is passed into Flunky Master
// from a clients.
type infoMsg struct {
	Time    int64
	Message string
	MsgType string
}

//RetType allow for information to be sent back to the requesting client
// in the buildvars function. 
type RetType struct {
	Status       string
	LastActivity int64
	Info         []infoMsg
}

//DataStore is the main user database for all compute nodes that have
// connected to the Flunky Master system for build orders. 
type DataStore struct {
	Allocate  int64
	Counts    map[string]int
	Errors    int
	Activity  int64
	Info      []infoMsg
	Image     string
	Extra     map[string]string
	Addresses []string
}

//ctrlmsg is the message that is sent to Flunky Master in order to assert
// as setup of the compute node to be built. 
type ctlmsg struct {
	Addresses []string
	Time      int64
	Image     string
	Extra     map[string]string
}

//Flunkym is the main data type that will allow the user to interface and add
// onto the system. This data type has all of the interfaces that are needed
// for all functions. 
type Flunkym struct {
	path   PathType
	data   map[string]DataStore
	static map[string]string
	log    *log.Logger
}

func decode(tmpAuth string) (username string, password string) {
	/*tmpAuthArray := strings.Split(tmpAuth, " ", 0)
	  authValues , error := base64.StdEncoding.DecodeString(tmpAuthArray[1])
	  CheckError(error,"Could not decode message")

	  authValuesArray := strings.Split(string(authValues), ":", 0)
	  username = authValuesArray[0]
	  password = authValuesArray[1]*/

	return
}

func (fm *Flunkym) init() {
	auth = make(map[string]userNode)
	logger := CreateLog()
	fm.log = logger
	root, _ := os.Getwd()
	fm.SetPath(root)
	fm.Load()
	fm.Assert_setup("ubuntu-maverick-amd64", "127.0.0.1")
	authFile, error := os.Open("UserDatabase")
	CheckError(error, "Cannot open user database")
	someBytes, error := ioutil.ReadAll(authFile)
	CheckError(error, "ERROR: Unable to read from file UserDatabase.")
	error = json.Unmarshal(someBytes, &auth)
	CheckError(error, "ERROR: Failed to unmarshal data read from UserDatabase file.")
	return
}


func CheckError(err os.Error, info string) bool {
	var correct bool
	correct = true

	if err != nil {
	        m.Lock()
		fm.log.Printf("%s - ERROR: %s", time.LocalTime(), err)
		m.Unlock()
		correct = false
	}
	return correct
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
	key := data[address]
	key.Data = orders
	key.Counts = fm.data[address].Counts
	data[address] = key
	return data
}

func CreateLog() *log.Logger {
	/*var file *os.File
	        _, err := os.Stat(fm.path.log)
		if err != nil {
			file, err =os.Create("flunky.log")
			CheckError(err, "Could not create file")
		} else {
			file, err = os.OpenFile(fm.path.log, os.O_RDWR, 0666)
			CheckError(err, "Could not locate file")
		}
		logger := log.New(file, "Flunky Master: ", 0)*/
	logger := log.New(os.Stdout, "Flunky Master: ", 0)
	return logger

}


//Mutex lock needed
func (fm *Flunkym) Assert_setup(image string, ip string) {
	var addresses []string
	info := make([]infoMsg, 0)
	image_dir := fm.path.image + "/" + image
	_, err := os.Stat(image_dir)
	CheckError(err, "Could not find specified Image")

	newsetup := make(map[string]DataStore)
	counts := make(map[string]int)
	newsetup[ip] = DataStore{time.Seconds(), counts, 0, time.Seconds(), info, image, nil, addresses}
	newsetup[ip].Counts["bootconfig"] = 0
	//newsetup[ip].AllocateNum = msg.AllocateNum
	m.Lock()
	fm.data[ip] = newsetup[ip]
	m.Unlock()
	fm.Store()
	m.Lock()
	fm.log.Printf("%s - INFO: Allocated %s as %s", time.LocalTime(), ip, image)
	m.Unlock()
	return
}

func (fm *Flunkym) Load() {
	_, err := os.Stat(fm.path.dataFile)
	if err != nil {
		data := make(map[string]DataStore)
		fm.data = data
		fm.log.Printf("%s - INFO: No previous data exsists. Data created", time.LocalTime())
	} else {
		fm.log.Printf("%s - INFO: Loading previous fm data", time.LocalTime())
		file, err := ioutil.ReadFile(fm.path.dataFile)
		CheckError(err, "Could not locate file")

		err = json.Unmarshal(file, &fm.data)
		CheckError(err, "Could not unmarshall json")
		fm.log.Printf("%s - INFO: Data Loaded", time.LocalTime())
	}

	file, err := ioutil.ReadFile(fm.path.staticdataPath)
	CheckError(err, "Could not Read File")

	err = json.Unmarshal(file, &fm.static)
	CheckError(err, "Could not unmarshall Json")

	return
}

//Mutex lock needed
func (fm *Flunkym) Store() {
        m.Lock()
	backup, err := json.Marshal(fm.data)
	CheckError(err, "Could not marshall Data")

	err = ioutil.WriteFile(fm.path.dataFile, backup, 0666)
	CheckError(err, "Could not write file")
        m.Unlock()
	return
}

func (fm *Flunkym) SetPath(root string) {
	fm.log.Printf("%s - INFO: Setting up path variables", time.LocalTime())
	path := new(PathType)
	path.root = root + "/repository"
	path.static = path.root + "/static"
	path.dynamic = path.root + "/dynamic"
	path.dataFile = path.root + "/data.json"
	path.staticdataPath = path.root + "/staticVars.json"
	path.log = path.root + "/flunky.log"
	path.image = path.root + "/images"
	fm.path = *path
	fm.log.Printf("%s - INFO: Path variables Created", time.LocalTime())
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
        m.Lock()
	fm.log.Printf("%s - INFO: Rendering %s for %s", time.LocalTime(), loc, address)
	m.Unlock()
	fname := fm.path.static + "/" + loc
	_, e := os.Stat(fname)
	CheckError(e, "Could not find the specified file")

	fm.Increment_Count(address, loc)
	contents, er := ioutil.ReadFile(fname)
	CheckError(er, "Could not read file")
	m.Lock()
	fm.log.Printf("%s - INFO: Rendered %s for %s", time.LocalTime(), loc, address)
	m.Unlock()
	return contents
}

func (fm *Flunkym) RenderGetDynamic(loc string, address string) []byte {
        m.Lock()
	fm.log.Printf("%s - INFO: Rendering %s for %s", time.LocalTime(), loc, address)
	m.Unlock()
	var tmp []byte
	dynamic_buf := bytes.NewBuffer(tmp)
	bvar := build_vars(address, loc)
	fname := fm.path.dynamic + "/" + loc
	_, e := os.Stat(fname)
	CheckError(e, "Could not find specified file")

	ans, err := ioutil.ReadFile(fname)
	CheckError(err, "Cannot read file")

	tmpl, err := kasia.Parse(string(ans))
	CheckError(err, "Cannot Parse Template")

	err = tmpl.Run(dynamic_buf, bvar[address])
	CheckError(err, "Cannot render template")

	fm.Increment_Count(address, loc)

	dynamic := dynamic_buf.Bytes()
	m.Lock()
	fm.log.Printf("%s - INFO: %s Rendered for %s", time.LocalTime(), loc, address)
	m.Unlock()
	return dynamic
}

func (fm *Flunkym) RenderImage(toRender string, address string) []byte {
        m.Lock()
	fm.log.Printf("%s - INFO: Rendering %s to %s", time.LocalTime(), toRender, address)
	m.Unlock()
	var buf []byte
	key := fm.data[address]
	l := bytes.NewBuffer(buf)
	bvar := build_vars(address, toRender)
	request := fm.path.image + "/" + key.Image + "/" + toRender
	_, e := os.Stat(request)
	CheckError(e, "Could not find file specified")
	ans, err := ioutil.ReadFile(request)
	CheckError(e, "Could not Read file")

	tmpl, err := kasia.Parse(string(ans))
	CheckError(err, "Could not parse file to be rendered")

	err = tmpl.Run(l, bvar[address])
	CheckError(err, "Could not render template")

	fm.Increment_Count(address, toRender)
	v := l.Bytes()
	m.Lock()
	fm.log.Printf("%s - INFO: %s Rendered to %s", time.LocalTime(), toRender, address)
	m.Unlock()
	return v
}


func DumpCall(w http.ResponseWriter, req *http.Request) {
	req.ProtoMinor = 0
	username, password := decode(req.Header.Get("Authorization"))
	if password != auth[username].Password {
		CheckError(os.NewError("Access Denied"), "Username password combo invalid.")
		return
	}
	m.Lock()
	tmp, err := json.Marshal(fm.data)
	m.Unlock()
	CheckError(err, "Cannot Marshal data")
	_, err = w.Write(tmp)
	if err != nil {
		http.Error(w, "Cannot write to socket", 500)
	}
	m.Lock()
	fm.log.Printf("%s - INFO: Data dump processed", time.LocalTime())
	m.Unlock()
}

func StaticCall(w http.ResponseWriter, req *http.Request) {
	req.ProtoMinor = 0
	username, password := decode(req.Header.Get("Authorization"))
	if password != auth[username].Password {
		CheckError(os.NewError("Access Denied"), "Username password combo invalid.")
		return
	}
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":", 2)
	address := addTmp[0]
	tmp := fm.RenderGetStatic("foo", address) //allow for random type names
	w.Write(tmp)
}

func DynamicCall(w http.ResponseWriter, req *http.Request) {
	req.ProtoMinor = 0
	username, password := decode(req.Header.Get("Authorization"))
	if password != auth[username].Password {
		CheckError(os.NewError("Access Denied"), "Username password combo invalid.")
		return
	}
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":", 2)
	address := addTmp[0]
	tmp := fm.RenderGetDynamic("test", address)
	w.Write(tmp)

}

func BootconfigCall(w http.ResponseWriter, req *http.Request) {
	req.ProtoMinor = 0
	username, password := decode(req.Header.Get("Authorization"))
	if password != auth[username].Password {
		CheckError(os.NewError("Access Denied"), "Username password combo invalid.")
		return
	}
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":", 2)
	address := addTmp[0]
	m.Lock()
	fm.log.Printf("%s - INFO: Creating bootconfig image for %s", time.LocalTime(), address)
	m.Unlock()
	tmp := fm.RenderImage("bootconfig", address) // allow for "name", "data[image]
	w.Write(tmp)
	m.Lock()
	fm.log.Printf("%s - INFO: bootconfig image Rendered for %s", time.LocalTime(), address)
	m.Unlock()
}

func InstallCall(w http.ResponseWriter, req *http.Request) {
	req.ProtoMinor = 0
	username, password := decode(req.Header.Get("Authorization"))
	if password != auth[username].Password {
		CheckError(os.NewError("Access Denied"), "Username password combo invalid.")
		return
	}
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":", 2)
	address := addTmp[0]
	m.Lock()
	fm.log.Printf("%s - INFO: Creating install script for %s", time.LocalTime(), address)
	m.Unlock()
	tmp := fm.RenderImage("install", address)
	m.Lock()
	fm.log.Printf("%s - INFO: install rendered for %s", time.LocalTime(), address)
	m.Unlock()
	w.Write(tmp)
}

//Mutex needed
func InfoCall(w http.ResponseWriter, req *http.Request) {
	req.ProtoMinor = 0
	username, password := decode(req.Header.Get("Authorization"))
	if password != auth[username].Password {
		CheckError(os.NewError("Access Denied"), "Username password combo invalid.")
		return
	}
	var tmp DataStore
	body, _ := ioutil.ReadAll(req.Body)
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":", 2)
	address := addTmp[0]
	m.Lock()
	fm.log.Printf("%s - INFO: Recived Info", time.LocalTime())
	m.Unlock()
	var msg infoMsg
	err := json.Unmarshal(body, &msg)
	CheckError(err, "Cannot unmarshal message")
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


func ErrorCall(w http.ResponseWriter, req *http.Request) {
	req.ProtoMinor = 0
	username, password := decode(req.Header.Get("Authorization"))
	if password != auth[username].Password {
		CheckError(os.NewError("Access Denied"), "Username password combo invalid.")
		return
	}
	var tmp DataStore
	body, _ := ioutil.ReadAll(req.Body)
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":", 2)
	address := addTmp[0]
	m.Lock()
	fm.log.Printf("%s - INFO: Recieved error!", time.LocalTime())
	m.Unlock()
	var msg infoMsg
	err := json.Unmarshal(body, &msg)
	CheckError(err, "Cannot Unmarsal message")
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

//Need go routine for each address
func CtrlCall(w http.ResponseWriter, req *http.Request) {
	req.ProtoMinor = 0
	username, password := decode(req.Header.Get("Authorization"))
	if password != auth[username].Password {
		CheckError(os.NewError("Access Denied"), "Username password combo invalid.")
		return
	}
	body, _ := ioutil.ReadAll(req.Body)
	add := req.RemoteAddr
	addTmp := strings.Split(add, ":", 2)
	address := addTmp[0]
	temper, err := net.LookupIP(address)
	CheckError(err, "Coud not find address")
	iaddr := temper[0].String()
	var msg DataStore
	m.Lock()
	fm.log.Printf("%s - INFO: Recived ctrl message from %s", time.LocalTime(), iaddr)
	m.Unlock()
	err = json.Unmarshal(body, &msg)
	CheckError(err, "Cannot Unmarshall message")
	for _, addr := range msg.Addresses {
		temper, err := net.LookupIP(addr)
		CheckError(err, "Coud not find address")
		iaddr := temper[0].String()
		m.Lock()
		fm.log.Printf("%s - INFO: Allocating %s as %s", time.LocalTime(), iaddr, msg.Image)
		m.Unlock()
		fm.Assert_setup(msg.Image, iaddr)
	}
	m.Lock()
	fm.log.Printf("%s - INFO: Added nodes to flunkyMaster", time.LocalTime())
	m.Unlock()
}

//Go routine needed for each address
func StatusCall(w http.ResponseWriter, req *http.Request) {
	req.ProtoMinor = 0
	username, password := decode(req.Header.Get("Authorization"))
	if password != auth[username].Password {
		CheckError(os.NewError("Access Denied"), "Username password combo invalid.")
		return
	}
	body, _ := ioutil.ReadAll(req.Body)
	var msg ctlmsg
	err := json.Unmarshal(body, &msg)
	CheckError(err, "Could not unmarshal message")

	cstatus := make(map[string]RetType)
	for _, addr := range msg.Addresses {
		temper, err := net.LookupIP(addr)
		iaddr := temper[0].String()
		CheckError(err, "Could not find adderss")
		m.Lock()
		fm.log.Printf("%s - INFO: Recieved request for status from %s", time.LocalTime(), iaddr)
		m.Unlock()
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
	CheckError(err, "Could not Marsal status")
	w.Write(ret)
}

func main() {

	//Config file: BuildServer Ip:
	//Server IP:
	//Name of log file
	// name of backup file
	runtime.GOMAXPROCS(4)
	fm.init()
	fm.Store()
	fm.log.Printf("%s - INFO: Starting server...", time.LocalTime())

	http.Handle("/dump", http.HandlerFunc(DumpCall))
	http.Handle("/static", http.HandlerFunc(StaticCall))
	http.Handle("/dynamic", http.HandlerFunc(DynamicCall))
	http.Handle("/bootconfig", http.HandlerFunc(BootconfigCall))
	http.Handle("/install", http.HandlerFunc(InstallCall))
	http.Handle("/info", http.HandlerFunc(InfoCall))
	http.Handle("/error", http.HandlerFunc(ErrorCall))
	http.Handle("/ctl", http.HandlerFunc(CtrlCall))
	http.Handle("/status", http.HandlerFunc(StatusCall))

	err := http.ListenAndServe("localhost:8080", nil)
	CheckError(err, "ListenandServe error : "+err.String())
	fm.log.Printf("Server exited gracefully")

}
