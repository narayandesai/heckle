package main

//On all code there should be diagLogic to see if a node is unavialble.
//Something to keep in mind... Do we have to worry about post and get
//Find a way to print ready only once. with the server? 
//No way to know if the server actually rebooted. All I can do is wait for info/error
// and hope that it worked...
//Incorporate web 404 and 500 errors... Stuff like that. 
//Make it a go routine to handle each individual node and not iterate through the list
//once a node is done it can return and therefore move on with the process. 
//Write file will take care of creating a file if it does not exissit....
//Need to provide checkes that the node has not been imaged or rebotted first. 
//Otherwise I could call a /diag or a /alloc and get nothing essentially...

//Ideas for the go routines:
// For loop to specify the range of addresses
// kill the loop when all have been go'ed
// Then collect each ones status. Put a mutex lock on any shared data
// and then see where that leads me. 
// Might be nice to include a start and end time in diag logs...
// Or a This is how long this took...
//Commang line does not work 
//Each node is to be sent to a function and each time should be marshalled to a list
// if posssible. 
//Check if a failute or a pass when things happen when doing a full gamnut run. 
//All calls need to return an address list that has or has not been changed. 
//Odd error that returns one more than what is needed for printing.
//Generate error report for all faield nodes function. 

/*  if err := viewTemplate.Execute(w, record); err != nil {
    http.Error(w, err.String(), 500)*/

import (
	"http"
	"io/ioutil"
	"json"
	"time"
	"bytes"
	"strings"
	"flag"
	"os"
	"log"
)

/*var Usage = func() {
    defaultUsage(commandLine)
}*/
var diagLog *log.Logger
var help bool
var nodes string
var server string
var power string
var image string

func init() {
	flag.BoolVar(&help, "h", false, "Print Usages")
	flag.StringVar(&nodes, "n", " ", "Node to be Checked") //right now need 2 on command line to run one...
	flag.StringVar(&power, "p", " ", "Power Cycle")
	flag.StringVar(&image, "i", "ubuntu-Rescue", "Set node(s) as rescue image")
	flag.StringVar(&server, "S", "http://localhost:8080", "Base Server Address")
	flag.StringVar(&power, "P", "http://localhost:8085", "Power Server Address")

	diagLog = log.New(os.Stdout, "Diagnostic: ", 0)
}

type CtrlMsg struct {
	Addresses []string
	Time      int64
	Image     string
	Extra     map[string]string
}

type NodeStatus struct {
	Name  string
	Time  int64
	Stat  bool
	ready bool
}

type infoMsg struct {
	Time    int64
	Message string
	MsgType string
}

type statusMessage struct {
	Status       string
	LastActivity int64
	Info         []infoMsg
}

func Get(serverurl string) *http.Response {
	var worker http.Client
	file,  err := worker.Get(serverurl)
	CheckError(err, "Cannot perform GET")
	return file
}

func CheckMethod(req *http.Request) bool {
	var exsist bool
	k := strings.Split(req.UserAgent(), "/")
	l := k[0]
	if l == "curl" {
		exsist = true
	} else {
		exsist = false
	}
	return exsist
}


func ControlMsg(nodes []string, times int64) (*bytes.Buffer, *CtrlMsg) {
	req := new(CtrlMsg)
	req.Addresses = nodes
	req.Time = times
	req.Image = "ubuntu-Rescue"
	resp, err := json.Marshal(req)
	CheckError(err, "Cannot Send ctrl msg")
	buf := bytes.NewBufferString(string(resp))
	return buf, req
}

func SendCtrl(addressList []string) bool {
	exsist := false
	nanoBase := 1000000000
	interval := int64(5) * int64(nanoBase)
	buf, _ := ControlMsg(addressList, int64(0))
	timeoutOffset := 5 * 60
	start := time.Seconds()
	end := start + int64(timeoutOffset)
	diagLog.Printf("%s - INFO: Sending control messages for %s", time.LocalTime(), addressList)
	for start < end {
		_, err := Post((server + "/ctl"), buf)

		if err == nil {
			exsist = true
			break
		}
		diagLog.Printf("%s - ERROR: %s", time.LocalTime(), err)
		time.Sleep(interval)
		start = time.Seconds()
	}
	return exsist
}

func FillNodes(addresses []string) map[string]NodeStatus {
	nodes := make(map[string]NodeStatus)
	for _, address := range addresses {
		key := nodes[address]
		key.Name = address
		key.Time = time.Seconds()
		key.Stat = false
		nodes[address] = key
	}
	return nodes
}

func Post(path string, buf *bytes.Buffer) (body []byte, err os.Error) {
	var worker http.Client
	gen, err := worker.Post(path, "text/plain", buf)
	if err != nil {
		return
	}
	body, _ = ioutil.ReadAll(gen.Body)
	gen.Body.Close()
	return
}

func PrepareCurl(ret []byte) []byte {
	woo := strings.Split(string(ret), "[")
	koo := strings.Split(woo[1], "]")
	maw := "[" + koo[0] + "]"
	baz := []byte(maw)
	return baz
}

func ReadInfo(address string, status map[string]statusMessage) bool {
	ok := true
	var addressList []string
	addressList = append(addressList, address)
	for _, info := range status[address].Info {

		if info.MsgType == "Error" {
			diagLog.Printf("%s - ERROR: %s - ERRORMSG: %s", time.LocalTime(), address, info.Message)
			diagLog.Printf("%s - ERROR: Cannot Allocate %s. Failure.", time.LocalTime(), address)
			CheckPower(addressList, "off")
			ok = false
			return ok
		} else if info.MsgType == "Info" {
			diagLog.Printf("%s - INFO: %s - %s", time.LocalTime(), address, info.Message)
		}
	}

	return ok
}

func Delete(mainList []string, toDelete []string) []string {
	var newList []string
	var pos int
	num := len(mainList)
	for _, add := range toDelete {
		for i := 0; i < num; i++ {

			if add == mainList[i] {
				pos = i
				break
			}
		}
		if pos > 0 && pos < len(mainList)-1 {
			low := mainList[:pos]
			high := mainList[pos+1:]
			newList = append(newList, low...)
			newList = append(newList, high...)
			mainList = nil
			mainList = append(mainList, newList...)
		} else if pos == 0 {
			mainList = mainList[pos+1:]
		} else if pos == len(mainList)-1 {
			mainList = mainList[:pos]
		}

	}
	return mainList
}

func PowerCycle(w http.ResponseWriter, req *http.Request) {
	var nodes []string
	curl := CheckMethod(req)
	addresses, err := ioutil.ReadAll(req.Body)
	CheckError(err, "Could not read body")

	if curl {
		address := PrepareCurl(addresses)
		err = json.Unmarshal(address, &nodes)
		CheckError(err, "Unable to unmarshall request")
	} else {
		err = json.Unmarshal(addresses, &nodes)
	}

	status := CheckPower(nodes, "reboot")
	if status {
		diagLog.Printf("%s - INFO: Power Cycle Request Filled", time.LocalTime())
	}
	//return a node status structure to the caller to see that everying went kind of ok.  

}

func ImageNodes(w http.ResponseWriter, req *http.Request) {
	var nodes []string
	curl := CheckMethod(req)
	addresses, err := ioutil.ReadAll(req.Body)
	CheckError(err, "Could not read function body")

	if curl {
		address := PrepareCurl(addresses)
		err = json.Unmarshal(address, &nodes)
		CheckError(err, "Unable to unmarshall request")
	} else {
		err = json.Unmarshal(addresses, &nodes)
	}

	nodeStat := FillNodes(nodes)
	SendCtrl(nodes)
	status, _ := CheckBuild(nodes, nodeStat)
	if status {
		diagLog.Printf("%s - INFO: %s are allocated and available", time.LocalTime(), nodes)
	}
	//Return info/error message back to caller in form of nodestat
}


func DiagnoseNodes(w http.ResponseWriter, req *http.Request) {
	var nodes []string
	curl := CheckMethod(req)

	addresses, err := ioutil.ReadAll(req.Body)
	CheckError(err, "Could not read function body")

	if curl {
		address := PrepareCurl(addresses)
		err = json.Unmarshal(address, &nodes)
		CheckError(err, "Unable to unmarshall request")
	} else {
		err = json.Unmarshal(addresses, &nodes)
	}
	nodeStat := FillNodes(nodes)
	stat := CheckNodes(nodes, nodeStat)
	if stat {
		diagLog.Printf("%s - INFO: Diagnosis complete for %s", time.LocalTime(), nodes)
	}
	//Return some type of info and error message back to the caller. 
}

func CheckWeb(err string, info string) bool {
	ok := true
	if err != "200 OK" {
		diagLog.Printf("%s - ERROR: %s", time.LocalTime(), err)
		ok = false
	}
	return ok
}

func CheckError(err os.Error, info string) bool {
	ok := true
	if err != nil {
		diagLog.Printf("%s - ERROR: %s", time.LocalTime(), err)
		ok = false
	}
	return ok
}

func CheckNodes(addressList []string, nodeStat map[string]NodeStatus) bool {
	ready := false
	status := SendCtrl(addressList)
	if status {
		diagLog.Printf("%s - INFO: Control messages sent for %s", time.LocalTime(), addressList)
	} else {
		ready = false
		return ready
	}
	diagLog.Printf("%s - INFO: Rebooting Nodes now...", time.LocalTime())
	status = CheckPower(addressList, "reboot")
	if status {
		diagLog.Printf("%s - INFO: %s have been power cycled", time.LocalTime(), addressList)
	} else {
		ready = false
		return ready
	}
	status, addressList = CheckBuild(addressList, nodeStat)
	if status {
		diagLog.Printf("%s - INFO: %s are allocated and available", time.LocalTime(), addressList)
	} else {
		ready = false
		return ready
	}
	diagLog.Printf("%s - INFO: Build completed. Rebooting %s", time.LocalTime(), addressList)
	status = CheckPower(addressList, "reboot")
	if status {
		diagLog.Printf("%s - INFO: %s have been power cycled", time.LocalTime(), addressList)
		ready = true
	} else {
		ready = false
		return ready
	}

	ready = true
	return ready
}


func CheckStatus(address string, status map[string]statusMessage, buildStatus string) NodeStatus {
	var key NodeStatus
	var readyList []string
	if status[address].Status == buildStatus {
		key.Stat = true
		key.ready = true
		readyList = append(readyList, address)
		diagLog.Printf("%s - INFO: %s - STATUS: %s", time.LocalTime(), address, status[address].Status)
	} else if status[address].Status != buildStatus || key.ready == false {
		diagLog.Printf("%s - INFO: %s - STATUS: %s", time.LocalTime(), address, status[address].Status)
	}

	return key
}


//Return nodes status and see which nodes failed and which nodes are still ok haha.
//We know we want an address list to be changed only when a node goes down. 
func CheckBuild(addressList []string, nodeStat map[string]NodeStatus) (bool, []string) {
	var cnt int
	var kill []string
	var readyList []string
	newList := addressList

	ok := false

	status := make(map[string]statusMessage)
	timeoutOffset := 45 * 60
	nanoBase := 1000000000
	interval := int64(5) * int64(nanoBase)
	times := time.Seconds()
	timeout := times + int64(timeoutOffset)
	msgTime := int64(0)

	diagLog.Printf("%s - INFO: Waiting for information from nodes in %s", time.LocalTime(), newList)
	for times < timeout {
		time.Sleep(interval)
		buf, _ := ControlMsg(newList, msgTime)
		ret, err := Post((server + "/status"), buf)
		CheckError(err, "Could not find server")
		if err == nil {
			json.Unmarshal(ret, &status)

			for _, address := range newList {
				nodeStat[address] = CheckStatus(address, status, "Ready")
				working := ReadInfo(address, status)
				if !working {
					kill = append(kill, address)
					diagLog.Printf("%s - ERROR: %s has been shut down due to error", time.LocalTime(), address)
				}

			}

			if len(kill) != 0 {
				addressList = Delete(addressList, kill)
				newList = addressList
			}
			kill = nil

			msgTime = time.Seconds()
			times = time.Seconds()
			cnt = 0

			//Last function needs to be cleaned better.
			for _, addy := range newList {
				if nodeStat[addy].Stat == false {
					cnt += 1
				} else if nodeStat[addy].Stat == true {
					readyList = append(readyList, addy)
				}
			}
			newList = Delete(newList, readyList)
			readyList = nil
			if cnt == 0 {
				ok = true
				break
			}
		}
	}
	return ok, addressList
}

//Could use error checking from the power daemon as well. c
func CheckPower(addressList []string, cmd string) bool {
	ready := false
	exsist := false
	nanoBase := 1000000000
	interval := int64(5) * int64(nanoBase)
	timeoutOffset := 5 * 60
	resp, err := json.Marshal(addressList)
	CheckError(err, "Unable to marshal data")

	buf := bytes.NewBufferString(string(resp))

	server := power + "/" + cmd
	start := time.Seconds()
	end := start + int64(timeoutOffset)

	for start < end {
		_, err = Post(server, buf)
		if err == nil {
			exsist = true
			break
		}
		diagLog.Printf("%s - ERROR: %s", time.LocalTime(), err)
		time.Sleep(interval)
		start = time.Seconds()
	}

	if exsist {
		diagLog.Printf("%s - INFO: %s %s request sent", time.LocalTime(), addressList, cmd)
		ready = true
	} else {
		ready = false
		diagLog.Printf("%s - ERROR: Function CheckPower() cannot be contacted", time.LocalTime())
	}

	return ready
}


func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		diagLog.Printf("%s - INFO: Starting Server....", time.LocalTime())
		http.Handle("/diag", http.HandlerFunc(DiagnoseNodes)) //Currenlty only allocates and reboots nodes.
		http.Handle("/power", http.HandlerFunc(PowerCycle))
		http.Handle("/image", http.HandlerFunc(ImageNodes))
		err := http.ListenAndServe("localhost:8082", nil)
		CheckError(err, "Cannot use server")
	} else {
		if help {
			flag.PrintDefaults()
		} else {
			var addressList []string
			if nodes != "empty" {
				if len(flag.Args()) >= 0 {
					addressList = flag.Args()
					addressList = append(addressList, nodes)
				} else {
					addressList = append(addressList, nodes)
				}
			}
			if nodes != " " {
				nodeStat := FillNodes(addressList)
				status := CheckNodes(addressList, nodeStat)
				if !status {
					diagLog.Printf("%s - ERROR: Cannot run the CheckNodes() function", time.LocalTime())
				}
			} else if image != " " {
				nodeStat := FillNodes(addressList)
				status, _ := CheckBuild(addressList, nodeStat)
				if !status {
					diagLog.Printf("%s - ERROR: Cannot run the ImageNodes() function", time.LocalTime())
				}

			} else if power != " " {
				status := CheckPower(addressList, "reboot")
				if !status {
					diagLog.Printf("%s - ERROR: Cannot run the PowerCycle() function", time.LocalTime())
				}

			}
		}

	}
}
