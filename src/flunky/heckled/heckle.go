package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	//"bytes"
	daemon "flunky/daemon"
	iface "flunky/interfaces"
	fnet "flunky/net"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
)

type States struct {
	State  bool
	Reboot bool
}

type resourceInfo struct {
	Allocated                        bool
	TimeAllocated, AllocationEndTime time.Time
	Owner, Image, Comments           string
	AllocationNumber                 uint64
}

type currentRequestsNode struct {
	User             string
	Image            string
	Status           string
	AllocationNumber uint64
	ActivityTimeout  int64
	TryOnFail        bool
	LastActivity     int64
	Info             []iface.InfoMsg
}

func (resource *resourceInfo) Reset() {
	resource.Allocated = false
	resource.TimeAllocated = time.Now()
	resource.AllocationEndTime = time.Now()
	resource.Owner = "None"
	resource.Image = "None"
	resource.Comments = ""
	resource.AllocationNumber = 0
}

func (resource *resourceInfo) Allocate(owner string, image string, allocationNum uint64) {
	resource.Allocated = true
	resource.Owner = owner
	resource.Image = image
	resource.TimeAllocated = time.Now()
	resource.AllocationEndTime = time.Now().Add(time.Duration(604800))
	resource.AllocationNumber = allocationNum
}

func (resource *resourceInfo) Broken() {
	resource.Allocated = true
	resource.TimeAllocated = time.Now()
	resource.AllocationEndTime = time.Now().Add(time.Duration(9223372036854775807))
	resource.Owner = "System Admin"
	resource.Image = "brokenNode-headAche-amd64"
	resource.Comments = "Installation failed or there was a timeout."
	resource.AllocationNumber = 0
}

var currentRequests map[string]*currentRequestsNode
var resources map[string]*resourceInfo
var allocationNumber uint64
var heckleToAllocateChan chan iface.Listmsg
var allocateToPollingChan chan []string
var pollingToHeckleChan chan map[string]*iface.StatusMessage
var pollingCancelChan chan []string
var heckleDaemon *daemon.Daemon
var currentRequestsLock sync.Mutex
var resourcesLock sync.Mutex
var allocationNumberLock sync.Mutex
var fmComm, pComm fnet.Communication
var fs, ps *fnet.BuildServer
var remove map[string]bool
var toDelete map[string]bool

func init() {
	//new comments here
	var err error
	flag.Parse()

	heckleDaemon, err = daemon.New("heckle")
	heckleDaemon.DaemonLog.LogError("Failed to create new heckle daemon.", err)
	heckleDaemon.DaemonLog.LogDebug("Initializing variables and setting up daemon.")

	user, pass, _ := heckleDaemon.AuthN.GetUserAuth()
	err = heckleDaemon.AuthN.Authenticate(user, pass, true)
	if err != nil {
		fmt.Println(fmt.Sprintf("You do not have proper permissions to start %s daemon.", heckleDaemon.Name))
		os.Exit(1)
	}

	fmComm, err = fnet.NewCommunication(daemon.FileDir+"components.conf", heckleDaemon.Cfg.Data["username"], heckleDaemon.Cfg.Data["password"])
	heckleDaemon.DaemonLog.LogError("Failed to make new communication structure in heckle for flunkymaster.", err)

	pComm, err = fnet.NewCommunication(daemon.FileDir+"components.conf", heckleDaemon.Cfg.Data["username"], heckleDaemon.Cfg.Data["password"])
	heckleDaemon.DaemonLog.LogError("Failed to make new communication structure in heckle for power.", err)

	fs, err = fmComm.SetupClient("flunky")
	heckleDaemon.DaemonLog.LogError("Failed to setup heckle to flunkymaster communication.", err)

	ps, err = pComm.SetupClient("power")
	heckleDaemon.DaemonLog.LogError("Failed to setup heckle to power communication.", err)

	heckleToAllocateChan = make(chan iface.Listmsg)
	allocateToPollingChan = make(chan []string)
	pollingToHeckleChan = make(chan map[string]*iface.StatusMessage)
	pollingCancelChan = make(chan []string)

	currentRequests = make(map[string]*currentRequestsNode)
	remove = make(map[string]bool)
	toDelete = make(map[string]bool)
	allocationNumber = 1
	getResources()
}

func resetResources(resourceNames []string) {
	//This is a function to accomodate reseting the json database.
	//it just creates a new resource map with all the resourceNames
	//entries in it.
	resourcesLock.Lock()
	resources = make(map[string]*resourceInfo)

	for _, value := range resourceNames {
		resources[value] = &resourceInfo{false, time.Now(), time.Now(), "None", "None", "", 0}
	}
	resourcesLock.Unlock()
}

func updateDatabase(term bool) {
	//This updates the json database file with the information in the
	//resource map.
	heckleDaemon.DaemonLog.LogDebug("Updating persistant json resource database file.")
	databaseFile, error := os.OpenFile(daemon.FileDir+"resources.db", os.O_RDWR, 0777)
	heckleDaemon.DaemonLog.LogError("Unable to open resource database file for reading and writing.", error)

	err := syscall.Flock(int(databaseFile.Fd()), 2) //2 is exclusive lock
	if (err != nil) {
		heckleDaemon.DaemonLog.LogError("Unable to lock resource database file.", errors.New("Flock Syscall Failed"))
	}

	error = databaseFile.Truncate(0)
	heckleDaemon.DaemonLog.LogError("Failed to truncate file.", error)

	resourcesLock.Lock()
	js, error := json.Marshal(resources)
	heckleDaemon.DaemonLog.LogError("Failed to marshal resources for resources database file.", error)
	resourcesLock.Unlock()

	_, error = databaseFile.Write(js)
	heckleDaemon.DaemonLog.LogError("Failed to write resources to resources database file.", error)

	if term {
		heckleDaemon.DaemonLog.LogDebug("Exiting gracefully, bye bye.")
		os.Exit(0)
	}

	err = syscall.Flock(int(databaseFile.Fd()), 8) //8 is unlock
	if (err != nil) {
		heckleDaemon.DaemonLog.LogError("Unable to unlock resource database for reading.", errors.New("Flock Syscall Failed"))
	}

	error = databaseFile.Close()
	heckleDaemon.DaemonLog.LogError("Failed to close resources database file.", error)
}

/*func resetResourceDatabase() {
     //This function is intended for admins to reset the json database file.
     resourceFile, error := os.Open("resources")
     heckleDaemon.DaemonLog.LogError("Failed to open resources file.", error)

     someBytes, error := ioutil.ReadAll(resourceFile)
     heckleDaemon.DaemonLog.LogError("Failed to read resources file.", error)

     error = resourceFile.Close()
     heckleDaemon.DaemonLog.LogError("Failed to close resource file.", error)

     resourceNames := strings.Split(string(someBytes), "\n")
     resourceNames = resourceNames[:len(resourceNames)-1]

     resetResources(resourceNames)
     updateDatabase(false)
}*/

func getResources() {
	//This function populated the resources map from the json database file.
	heckleDaemon.DaemonLog.LogDebug("Initializing resource data from resource database file.")

	databaseFile, error := os.Open(daemon.FileDir + "resources.db")
	heckleDaemon.DaemonLog.LogError("Failed to open resource database file for reading.", error)

	someBytes, error := ioutil.ReadAll(databaseFile)
	heckleDaemon.DaemonLog.LogError("Failed to read from resource database file.", error)

	error = databaseFile.Close()
	heckleDaemon.DaemonLog.LogError("Failed to close resource database file.", error)

	resourcesLock.Lock()
	resources = make(map[string]*resourceInfo)
	error = json.Unmarshal(someBytes, &resources)
	heckleDaemon.DaemonLog.LogError("Failed to unmarshal data read from resource database file.", error)
	resourcesLock.Unlock()
}

func getNumNodes(numNodes int, owner string, image string, allocationNum uint64) []string {
	//This function is for the http allocate number of nodes function.
	//It will create a list of that many free nodes or as many free
	//nodes as it can, upate the resource map accordingly, and return
	//the list.
	heckleDaemon.DaemonLog.Log("Finding " + strconv.Itoa(numNodes) + " nodes to allocate.")
	tmpNodeList := []string{}
	index := 0

	resourcesLock.Lock()
	defer resourcesLock.Unlock()

	for key, value := range resources {
		if index < numNodes && !value.Allocated {
			tmpNodeList = append(tmpNodeList, key)
			index++
		}
	}

	if index != numNodes {
		heckleDaemon.DaemonLog.LogError("Not enough open nodes to allocate, cancelling allocation", errors.New("Not Enough Nodes"))
		return []string{}
	} else {
		for _, value := range tmpNodeList {
			resources[value].Allocate(owner, image, allocationNum)
		}
	}
	return tmpNodeList
}

func checkNodeList(nodeList []string, owner string, image string, allocationNum uint64) (listOk bool) {
	//This function is for the http allocate list function.  It checks
	//the list requested and allocated as many nodes as are available
	//that are in that list.  It updates the resource map accordingly,
	//and returns the new list.
	listOk = true
	heckleDaemon.DaemonLog.LogDebug(fmt.Sprintf("Checking to see if %s can be allocated.", nodeList))

	resourcesLock.Lock()
	defer resourcesLock.Unlock()
	for _, value := range nodeList {
		if val, ok := resources[value]; !ok || val.Allocated {
			listOk = false
		}
	}

	if !listOk {
		heckleDaemon.DaemonLog.LogError("Some of the nodes asked for are allocated, cancelling allocation.", errors.New("List Nodes Taken"))
	} else {
		for _, value := range nodeList {
			resources[value].Allocate(owner, image, allocationNum)
		}
	}
	return
}

func allocateList(writer http.ResponseWriter, req *http.Request) {
	//This is an http handler function to deal with allocation list requests.
	//It grabs the list from the message, makes a new lsit of all available
	//nodes within that original list.  Gets an allocation number, and adds
	//them to the current requests map.

	heckleDaemon.DaemonLog.DebugHttp(req)
	heckleDaemon.DaemonLog.LogDebug("Allocating a list of nodes.")

	req.ProtoMinor = 0
	err := heckleDaemon.AuthN.HTTPAuthenticate(req, false)
	if err != nil {
		heckleDaemon.DaemonLog.LogError("Permission denied", err)
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}
	username, _, _ := heckleDaemon.AuthN.GetHTTPAuthenticateInfo(req)
	heckleDaemon.UpdateActivity()

	jsonType := heckleDaemon.ProcessJson(req, new(iface.Listmsg))
	listMsg := jsonType.(*iface.Listmsg)

	allocationNumberLock.Lock()
	tmpAllocationNumber := allocationNumber

	allocationListOk := checkNodeList(listMsg.Addresses, username, listMsg.Image, tmpAllocationNumber)
	if !allocationListOk {
		allocationNumberLock.Unlock()
		return
	}
	allocationNumber++
	allocationNumberLock.Unlock()

	heckleToAllocateChan <- iface.Listmsg{listMsg.Addresses, listMsg.Image, 0, int(tmpAllocationNumber)}

	currentRequestsLock.Lock()
	for _, value := range listMsg.Addresses {
		currentRequests[value] = &currentRequestsNode{username, listMsg.Image, "Building", tmpAllocationNumber, listMsg.ActivityTimeout, false, 0, []iface.InfoMsg{}}
		remove[value] = false
	}
	currentRequestsLock.Unlock()
	heckleDaemon.DaemonLog.Log(fmt.Sprintf("Added allocation #%d nodes: %s to Heckle", tmpAllocationNumber, listMsg.Addresses))
	js, _ := json.Marshal(tmpAllocationNumber)
	writer.Write(js)

	updateDatabase(false)
}

func allocateNumber(writer http.ResponseWriter, req *http.Request) {
	//This is just an http function that deals with allocation number requests.
	//It grabs the number, gets a list of that number or less of nodes, gets
	//an allocation number, and adds them to the current requests map.
	heckleDaemon.DaemonLog.DebugHttp(req)

	req.ProtoMinor = 0
	err := heckleDaemon.AuthN.HTTPAuthenticate(req, false)
	if err != nil {
		heckleDaemon.DaemonLog.LogError("Permission denied", err)
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}
	heckleDaemon.UpdateActivity()
	username, _, _ := heckleDaemon.AuthN.GetHTTPAuthenticateInfo(req)

	jsonType := heckleDaemon.ProcessJson(req, new(iface.Nummsg))
	numMsg := jsonType.(*iface.Nummsg)

	heckleDaemon.DaemonLog.Log(fmt.Sprintf("Allocating %d nodes.", numMsg))
	allocationNumberLock.Lock()
	tmpAllocationNumber := allocationNumber

	allocationList := getNumNodes(numMsg.NumNodes, username, numMsg.Image, tmpAllocationNumber)
	if len(allocationList) == 0 {
		allocationNumberLock.Unlock()
		return
	}
	allocationNumber++
	allocationNumberLock.Unlock()

	heckleToAllocateChan <- iface.Listmsg{allocationList, numMsg.Image, 0, int(tmpAllocationNumber)}

	currentRequestsLock.Lock()
	for _, value := range allocationList {
		currentRequests[value] = &currentRequestsNode{username, numMsg.Image, "Building", tmpAllocationNumber, numMsg.ActivityTimeout, true, 0, []iface.InfoMsg{}}
	}
	currentRequestsLock.Unlock()

	js, _ := json.Marshal(tmpAllocationNumber)
	writer.Write(js)

	updateDatabase(false)
}

func allocate() {
	//This is the allocate thread.  It set up a client for ctl messages to
	//flunky master.  On each iteration it grabs new nodes from heckle to be
	//allocated and send them off to flunkymaster.
	heckleDaemon.DaemonLog.LogDebug("Starting allocation go routine.")

	for i := range heckleToAllocateChan {
		cm := new(iface.Ctlmsg)
		cm.Image = i.Image
		cm.Addresses = i.Addresses
		cm.AllocNum = uint64(i.AllocNum)
		// FIXME: need to add in extradata

		/*js, _ := json.Marshal(cm)
		buf := bytes.NewBufferString(string(js))*/

		_, err := fs.PostServer("/ctl", cm)
		heckleDaemon.DaemonLog.LogError("Failed to post for allocation of nodes.", err)

		if err == nil {
			allocateToPollingChan <- i.Addresses

			_, err = ps.PostServer("/command/reboot", i.Addresses)
			heckleDaemon.DaemonLog.LogError("Failed to post for reboot of nodes in allocation go routine.", err)
		}
	}
	close(allocateToPollingChan)
}

func addToPollList(pollAddressesLock *sync.Mutex, pollAddresses *[]string) {
	for i := range allocateToPollingChan {
		heckleDaemon.DaemonLog.LogDebug(fmt.Sprintf("Adding %s to polling list.", i))
		pollAddressesLock.Lock()
		*pollAddresses = append(*pollAddresses, i...)
		pollAddressesLock.Unlock()
	}
}

func deleteFromPollList(pollAddressesLock *sync.Mutex, pollAddresses *[]string) {
	for i := range pollingCancelChan {
		heckleDaemon.DaemonLog.LogDebug(fmt.Sprintf("Removing %s from polling list.", i))
		pollAddressesLock.Lock()
		for _, value := range i {
			for pos2, value2 := range *pollAddresses {
				if value == value2 {
					*pollAddresses = append((*pollAddresses)[:pos2], (*pollAddresses)[pos2+1:]...)
				}
			}
		}
		pollAddressesLock.Unlock()
	}
}

func polling() {
	//This function sets up an http client for polling flunky master.  Every
	//iteration through the loop it grabs all new addresses from the allocation
	//thread, polls on all addresses in that list, send those messages to heckle,
	//grabs nodes for cancelation and removes them from the list.
	heckleDaemon.DaemonLog.LogDebug("Starting polling go routine.")
	pollAddresses := []string{}
	pollingOutletStatus := make(map[string]string)
	var pollAddressesLock sync.Mutex
	pollTime := time.Now()

	go addToPollList(&pollAddressesLock, &pollAddresses)
	go deleteFromPollList(&pollAddressesLock, &pollAddresses)

	for ; ; time.Sleep(10 * time.Second) {
		heckleDaemon.DaemonLog.LogDebug("Polling for messages from flunky master and power daemons.")
		statRequest := new(iface.Ctlmsg)
		pollAddressesLock.Lock()
		statRequest.Addresses = pollAddresses
		pollAddressesLock.Unlock()
		if len(statRequest.Addresses) != 0 {
			statRequest.Time = pollTime.Unix()

			var statmap map[string]*iface.StatusMessage
			/*sRjs, _ := json.Marshal(statRequest)
			reqbuf := bytes.NewBufferString(string(sRjs))*/

			ret, _ := fs.PostServer("/status", statRequest)
			pollTime = time.Now()
			json.Unmarshal(ret, &statmap)

			outletStatus := make(map[string]States)

			ret, _ = ps.PostServer("/status", pollAddresses)
			json.Unmarshal(ret, &outletStatus)

			var pstat string
			//This garbage needs to change!
			for key, value := range statmap {
				if outletStatus[key].Reboot {
					pstat = "currently rebooting"
				} else if outletStatus[key].State {
					pstat = "On"
				} else {
					pstat = "Off"
				}
				if _, ok := pollingOutletStatus[key]; !ok {
					value.Info = append(value.Info, iface.InfoMsg{time.Now().Unix(), "Power outlet for " + key + " is " + pstat + ".", "Info"})
					pollingOutletStatus[key] = pstat
				} else if pollingOutletStatus[key] != pstat {
					value.Info = append(value.Info, iface.InfoMsg{time.Now().Unix(), "Power outlet for " + key + " is " + pstat + ".", "Info"})
					pollingOutletStatus[key] = pstat
				}
			}
			heckleDaemon.DaemonLog.LogDebug("Sending status messages to main routine.")
			pollingToHeckleChan <- statmap
		}
	}
}

func findNewNode(owner string, image string, activityTimeout int64, tmpAllocationNumber uint64) {
	//This function finds a single node for someone whose node got canceled and requested
	//a number of nodes.  It then sends this node to the allocation thread and tosses it
	//on the current requests map.

	heckleDaemon.DaemonLog.Log(fmt.Sprintf("User %s, your node is either offline or allocated. Finding a replacement node for allocation #%d.", owner, tmpAllocationNumber))
	allocationList := getNumNodes(1, owner, image, tmpAllocationNumber)
	heckleToAllocateChan <- iface.Listmsg{allocationList, image, 0, int(tmpAllocationNumber)}

	currentRequestsLock.Lock()
	for _, value := range allocationList {
		currentRequests[value] = &currentRequestsNode{owner, image, "Building", tmpAllocationNumber, activityTimeout, true, 0, []iface.InfoMsg{}}
	}
	currentRequestsLock.Unlock()

	updateDatabase(false)
}

func DumpCall(w http.ResponseWriter, req *http.Request) {
	heckleDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0
	err := heckleDaemon.AuthN.HTTPAuthenticate(req, false)
	if err != nil {
		heckleDaemon.DaemonLog.LogError("Permission Denied", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	heckleDaemon.UpdateActivity()
	resourcesLock.Lock()
	tmp, err := json.Marshal(resources)
	resourcesLock.Unlock()
	heckleDaemon.DaemonLog.LogError("Cannot Marshal heckle data", err)
	_, err = w.Write(tmp)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}
	heckleDaemon.DaemonLog.Log("Dump request processed")
}

func status(writer http.ResponseWriter, req *http.Request) {
	//This is an http handler function to deal with allocation status requests.
	//if the host has ownership of the allocation number we send back a map
	//of node names and a status message type.

	heckleDaemon.DaemonLog.DebugHttp(req)
	heckleDaemon.DaemonLog.LogDebug("Sending allocation status to client.")
	allocationStatus := make(map[string]*iface.StatusMessage)
	req.ProtoMinor = 0
	err := heckleDaemon.AuthN.HTTPAuthenticate(req, false)
	if err != nil {
		heckleDaemon.DaemonLog.LogError("Permission Denied", err)
		writer.WriteHeader(http.StatusUnauthorized)
	}
	username, _, admin := heckleDaemon.AuthN.GetHTTPAuthenticateInfo(req)

	jsonType := heckleDaemon.ProcessJson(req, new(uint64))
	allocationNumber := jsonType.(*uint64)

	currentRequestsLock.Lock()
	for key, value := range currentRequests {
		if *allocationNumber == value.AllocationNumber {
			if value.User == username || admin {
				sm := &iface.StatusMessage{value.Status, value.LastActivity, value.Info}
				allocationStatus[key] = sm
				value.Info = []iface.InfoMsg{}
				if value.Status == "Ready" {
					remove[key] = true
				}
			} else {
				heckleDaemon.DaemonLog.LogError("Cannot request status of allocations that do not beling to you.", errors.New("Access Denied"))
				writer.WriteHeader(http.StatusUnauthorized)
				currentRequestsLock.Unlock()
				return
			}
		}
	}
	currentRequestsLock.Unlock()

	jsonStat, err := json.Marshal(allocationStatus)
	heckleDaemon.DaemonLog.LogError("Unable to marshal allocation status response.", err)

	_, err = writer.Write(jsonStat)
	heckleDaemon.DaemonLog.LogError("Unable to write allocation status response.", err)
}

func freeAllocation(writer http.ResponseWriter, req *http.Request) {
	//This function allows a user, if it owns the allocation, to free an allocation
	//number and all associated nodes.  It resets the resource map and current
	//requests map.
	heckleDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0

	err := heckleDaemon.AuthN.HTTPAuthenticate(req, false)
	if err != nil {
		heckleDaemon.DaemonLog.LogError("Permission Denied", err)
		writer.WriteHeader(http.StatusUnauthorized)
	}
	heckleDaemon.UpdateActivity()
	username, _, admin := heckleDaemon.AuthN.GetHTTPAuthenticateInfo(req)

	jsonType := heckleDaemon.ProcessJson(req, new(uint64))
	allocationNumber := jsonType.(*uint64)

	if *allocationNumber <= 0 {
		heckleDaemon.DaemonLog.LogError(fmt.Sprintf("Allocation #%d does not exsist", *allocationNumber), errors.New("0 used"))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	powerDown := []string{}

	currentRequestsLock.Lock()
	resourcesLock.Lock()
	found := false
	for key, value := range resources {
		if *allocationNumber == value.AllocationNumber {
			if username == value.Owner || admin {
				value.Reset()
				powerDown = append(powerDown, key)
				delete(currentRequests, key)
				found = true
			} else {
				heckleDaemon.DaemonLog.LogError("Cannot free allocations that do not belong to you.", errors.New("Access Denied"))
				currentRequestsLock.Unlock()
				resourcesLock.Unlock()
				return
			}
		}
	}
	currentRequestsLock.Unlock()
	resourcesLock.Unlock()

	if !found {
		heckleDaemon.DaemonLog.LogError(fmt.Sprintf("Allocation #%d does not exist.", allocationNumber), errors.New("Wrong Number"))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	pollingCancelChan <- powerDown //Needed because polling will continue to poll if allocation is freed during allocation. 

	_, err = ps.PostServer("/command/off", powerDown)
	heckleDaemon.DaemonLog.LogError("Failed to post for reboot of nodes in free allocation number.", err)
	heckleDaemon.DaemonLog.Log(fmt.Sprintf("Allocation #%d nodes %s have been freed.", *allocationNumber, powerDown))
	updateDatabase(false)
}

func increaseTime(writer http.ResponseWriter, req *http.Request) {
	//This function allows a user, if it owns the allocation, to free an allocation
	//number and all associated nodes.  It resets the resource map and current
	//requests map.
	var end time.Time
	heckleDaemon.DaemonLog.DebugHttp(req)
	heckleDaemon.DaemonLog.LogDebug("Increasing allocation time on an allocation number.")

	req.ProtoMinor = 0
	err := heckleDaemon.AuthN.HTTPAuthenticate(req, false)
	if err != nil {
		heckleDaemon.DaemonLog.LogError("Permission Denied", err)
		writer.WriteHeader(http.StatusUnauthorized)
	}
	heckleDaemon.UpdateActivity()
	username, _, _ := heckleDaemon.AuthN.GetHTTPAuthenticateInfo(req)

	jsonType := heckleDaemon.ProcessJson(req, new(int64))
	timeIncrease := jsonType.(int64)

	resourcesLock.Lock()
	for _, value := range resources {
		if value.Owner == username {
			value.AllocationEndTime = value.AllocationEndTime.Add(time.Duration(timeIncrease))
			end = value.AllocationEndTime
		}
	}
	resourcesLock.Unlock()
	heckleDaemon.DaemonLog.Log(fmt.Sprintf("Increased timeout by %d for %s. Allocation will end at %d", timeIncrease, username, end.Unix()))
	updateDatabase(false)
}

func allocationTimeouts() {
	//This function deals with allocation timeouts.  If a node has timed out in
	//its allocation, we reset the resource map.  Also removes it from current
	//requests if it exists.

	found := false
	powerDown := []string{}

	currentRequestsLock.Lock()
	resourcesLock.Lock()

	for key, value := range resources {
		if value.Allocated && time.Since(value.AllocationEndTime).Seconds() < 0 {
			found = true
			powerDown = append(powerDown, key)
			delete(currentRequests, key)
			value.Reset()
		}
	}

	currentRequestsLock.Unlock()
	resourcesLock.Unlock()

	if found {
		heckleDaemon.DaemonLog.LogDebug("Found allocation time outs, deallocating them.")

		_, err := ps.PostServer("/command/off", powerDown)
		heckleDaemon.DaemonLog.LogError("Failed to post for reboot of nodes in allocation time outs.", err)
		updateDatabase(false)
	}
}

func freeNode(writer http.ResponseWriter, req *http.Request) {
	//This will free a requested node if the user is the owner of the node.  It removes
	//the node from current resources if it exists and also resets it in resources map.

	heckleDaemon.DaemonLog.LogDebug("Freeing a specific node given by client.")

	req.ProtoMinor = 0
	heckleDaemon.DaemonLog.DebugHttp(req)

	err := heckleDaemon.AuthN.HTTPAuthenticate(req, false)
	if err != nil {
		heckleDaemon.DaemonLog.LogError("Permission Denied", err)
		writer.WriteHeader(http.StatusUnauthorized)
	}
	heckleDaemon.UpdateActivity()
	username, _, _ := heckleDaemon.AuthN.GetHTTPAuthenticateInfo(req)

	jsonType := heckleDaemon.ProcessJson(req, new(string))
	node := jsonType.(*string)

	currentRequestsLock.Lock()
	resourcesLock.Lock()

	if val, ok := resources[*node]; !ok || val.Owner != username {
		heckleDaemon.DaemonLog.LogError("Access denied, cannot free nodes that do not belong to you.", errors.New("Access Denied"))
		writer.WriteHeader(http.StatusUnauthorized)
		currentRequestsLock.Unlock()
		resourcesLock.Unlock()
		return
	}

	resources[*node].Reset()
	delete(currentRequests, *node)

	currentRequestsLock.Unlock()
	resourcesLock.Unlock()

	_, err = ps.PostServer("/command/off", ([]string{*node}))
	heckleDaemon.DaemonLog.LogError("Failed to post for reboot of nodes in free node.", err)
	heckleDaemon.DaemonLog.Log(fmt.Sprintf("Freed %s in allocation #%d for %s.", node, resources[*node].AllocationNumber, username))
	updateDatabase(false)
}

func listenAndServeWrapper() {
	//This branches off another thread to loop through listening and serving http requests.
	heckleDaemon.DaemonLog.LogDebug("Starting HTTP listen go routine.")
	heckleDaemon.DaemonLog.Log(fmt.Sprintf("Started %s on %s", heckleDaemon.Name, heckleDaemon.URL))
	error := http.ListenAndServe(":"+heckleDaemon.Cfg.Data["hecklePort"], nil)
	heckleDaemon.DaemonLog.LogError("Failed to listen on http socket.", error)

	updateDatabase(true)
}

func freeCurrentRequests() {
	//If a node is ready or canceled and there are no more info messages remove it from being tracked in
	//current requests.  No need to update resources structure because this is done in other places.

	currentRequestsLock.Lock()
	for key, value := range currentRequests {
		if (value.Status == "Cancel" || toDelete[key] == true) && (len(value.Info) == 0) {
			//if (value.Status == "Cancel" || value.Status == "Ready") && (len(value.Info) == 0) {
			delete(currentRequests, key)
		}
	}
	currentRequestsLock.Unlock()
}

func dealWithBrokenNode(node string) {
	heckleDaemon.DaemonLog.LogDebug("Sending broken node to diagnostic daemon.")
	resourcesLock.Lock()
	resources[node].Broken()
	resourcesLock.Unlock()

	_, err := ps.PostServer("/command/off", ([]string{node}))
	heckleDaemon.DaemonLog.LogError("Failed to post for reboot of nodes in free node.", err)

	//pass node off to diagnosing process
	updateDatabase(false)
}

func interpretPollMessages() {
	heckleDaemon.DaemonLog.LogDebug("Starting message interpreter go routine.")
	for i := range pollingToHeckleChan {
		heckleDaemon.DaemonLog.LogDebug(fmt.Sprintf("New Poll message: %s", i))
		nodesToRemove := []string{}

		for key, value := range i {
			currentRequestsLock.Lock()
			currentRequests[key].Status = value.Status
			currentRequests[key].LastActivity = value.LastActivity

			if len(value.Info) != 0 {
				currentRequests[key].Info = append(currentRequests[key].Info, value.Info...)
			}

			if value.Status == "Cancel" || time.Since(time.Unix(value.LastActivity, 0)) >= time.Duration(currentRequests[key].ActivityTimeout) {
				nodesToRemove = append(nodesToRemove, key)
				go dealWithBrokenNode(key)

				if currentRequests[key].TryOnFail {
					findNewNode(currentRequests[key].User, currentRequests[key].Image, currentRequests[key].ActivityTimeout, currentRequests[key].AllocationNumber)
				}
			}
			if value.Status == "Ready" {
				if remove[key] == true {
					nodesToRemove = append(nodesToRemove, key)
					toDelete[key] = true
				}
			}
			currentRequestsLock.Unlock()
		}
		pollingCancelChan <- nodesToRemove
	}
}

func outletStatus(writer http.ResponseWriter, req *http.Request) {
	heckleDaemon.DaemonLog.LogDebug("Executing power command.")
	req.ProtoMinor = 0
	heckleDaemon.DaemonLog.DebugHttp(req)

	err := heckleDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		heckleDaemon.DaemonLog.LogError("Permission Denied", err)
		writer.WriteHeader(http.StatusUnauthorized)
	}
	heckleDaemon.UpdateActivity()
	body, _ := heckleDaemon.ReadRequest(req)

	someBytes, err := ps.PostServer("/status", body)
	heckleDaemon.DaemonLog.LogError("Failed to post for status of outlets to Power.go.", err)

	_, err = writer.Write(someBytes)
	heckleDaemon.DaemonLog.LogError("Unable to write outlet status response in heckle.", err)
}

func nodeStatus(writer http.ResponseWriter, req *http.Request) {
	heckleDaemon.DaemonLog.LogDebug("Sending back node status.")
	response := ""
	req.ProtoMinor = 0
	heckleDaemon.DaemonLog.DebugHttp(req)

	err := heckleDaemon.AuthN.HTTPAuthenticate(req, false)
	if err != nil {
		heckleDaemon.DaemonLog.LogError("Permission Denied", err)
		writer.WriteHeader(http.StatusUnauthorized)
	}
	heckleDaemon.UpdateActivity()
	resourcesLock.Lock()
	//This need to go away as well. Why are we writing out the response...?
	for key, value := range resources {
		if value.Allocated {
			response = response + "NODE: " + key + "\tALLOCATED: yes\tALLOCATION: " + strconv.FormatUint(value.AllocationNumber, 10) + "\tOWNER: " + value.Owner + "\tIMAGE: " + value.Image + "\tALLOCATION START: " + value.TimeAllocated.Format(time.UnixDate) + "\tALLOCATION END: " + value.AllocationEndTime.Format(time.UnixDate) + "\tCOMMENTS: " + value.Comments + "\n\n"

		}
	}
	resourcesLock.Unlock()

	_, error := writer.Write([]byte(response))
	heckleDaemon.DaemonLog.LogError("Unable to write node status response in heckle.", error)
}

func daemonCall(w http.ResponseWriter, req *http.Request) {
	heckleDaemon.DaemonLog.DebugHttp(req)
	req.ProtoMinor = 0

	err := heckleDaemon.AuthN.HTTPAuthenticate(req, true)
	if err != nil {
		heckleDaemon.DaemonLog.LogError("Access not permitted.", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	heckleDaemon.UpdateActivity()
	stat := heckleDaemon.ReturnStatus()

	status, err := json.Marshal(stat)
	if err != nil {
		heckleDaemon.DaemonLog.LogError(err.Error(), err)
	}
	w.Write(status)
	return

}

func main() {
	http.HandleFunc("/daemon", daemonCall)
	http.HandleFunc("/dump", DumpCall)
	http.HandleFunc("/list", allocateList)
	http.HandleFunc("/number", allocateNumber)
	http.HandleFunc("/status", status)
	http.HandleFunc("/freeAllocation", freeAllocation)
	http.HandleFunc("/freeNode", freeNode)
	http.HandleFunc("/increaseTime", increaseTime)
	http.HandleFunc("/outletStatus", outletStatus)
	http.HandleFunc("/nodeStatus", nodeStatus)

	go allocate()
	go polling()
	go interpretPollMessages()
	go listenAndServeWrapper()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		select {
		case <-interrupt: 
			updateDatabase(true)
			fmt.Println("Shutting down")
			os.Exit(1)
		default:
			runtime.Gosched()
			allocationTimeouts()
			freeCurrentRequests()
			//fmt.Fprintf(os.Stdout, "Go routines, %d.\n", runtime.Goroutines())
		}
	}
}
