package main

import (
     "json"
     "flag"
     "os"
     "time"
     "io/ioutil"
     "http"
     "bytes"
     "sync"
     "runtime"
     "os/signal"
     "syscall"
     "strconv"
     fnet "flunky/net"
     iface "flunky/interfaces"
     daemon "flunky/daemon"
)

type resourceInfo struct {
     Allocated                          bool
     TimeAllocated, AllocationEndTime   int64
     Owner, Image, Comments             string
     AllocationNumber                   uint64
}

type currentRequestsNode struct {
     User                string
     Image               string
     Status              string
     AllocationNumber    uint64
     ActivityTimeout     int64
     TryOnFail           bool
     LastActivity        int64
     Info                []iface.InfoMsg
}

func (resource *resourceInfo) Reset() {
     resource.Allocated = false
     resource.TimeAllocated = 0
     resource.AllocationEndTime = 0
     resource.Owner = "None"
     resource.Image = "None"
     resource.Comments = ""
     resource.AllocationNumber = 0
}

func (resource *resourceInfo) Allocate(owner string, image string, allocationNum uint64) {
     resource.Allocated = true
     resource.Owner = owner
     resource.Image = image
     resource.TimeAllocated = time.Seconds()
     resource.AllocationEndTime = time.Seconds() + 604800
     resource.AllocationNumber = allocationNum
}

func (resource *resourceInfo) Broken() {
     resource.Allocated = true
     resource.TimeAllocated = time.Seconds()
     resource.AllocationEndTime = 9223372036854775807
     resource.Owner = "System Admin"
     resource.Image = "brokenNode-headAche-amd64"
     resource.Comments = "Installation failed or there was a timeout."
     resource.AllocationNumber = 0
}

var currentRequests           map[string]*currentRequestsNode
var resources                 map[string]*resourceInfo
var allocationNumber          uint64
var heckleToAllocateChan      chan iface.Listmsg
var allocateToPollingChan     chan []string
var pollingToHeckleChan       chan map[string]*iface.StatusMessage
var pollingCancelChan         chan []string
var heckleDaemon              *daemon.Daemon
var currentRequestsLock       sync.Mutex
var resourcesLock             sync.Mutex
var allocationNumberLock      sync.Mutex
var fmComm, pComm             fnet.Communication
var fs, ps                    *fnet.BuildServer

func init() {
     //new comments here
     var err os.Error
     flag.Parse()
     
     heckleDaemon, err = daemon.New("heckle")
     heckleDaemon.DaemonLog.LogError("Failed to create new heckle daemon.", err)

     heckleDaemon.DaemonLog.Log("Initializing variables and setting up daemon.")
    
     fmComm, err = fnet.NewCommunication(daemon.FileDir + "components.conf", heckleDaemon.Cfg.Data["username"], heckleDaemon.Cfg.Data["password"])
     heckleDaemon.DaemonLog.LogError("Failed to make new communication structure in heckle for flunkymaster.", err)

     pComm, err = fnet.NewCommunication(daemon.FileDir + "components.conf", heckleDaemon.Cfg.Data["username"], heckleDaemon.Cfg.Data["password"])
     heckleDaemon.DaemonLog.LogError("Failed to make new communication structure in heckle for power.", err)

     fs, err = fmComm.SetupClient("flunkymaster")
     heckleDaemon.DaemonLog.LogError("Failed to setup heckle to flunkymaster communication.", err)

     ps, err = pComm.SetupClient("power")
     heckleDaemon.DaemonLog.LogError("Failed to setup heckle to power communication.", err)
     
     heckleToAllocateChan = make(chan iface.Listmsg)
     allocateToPollingChan = make(chan []string)
     pollingToHeckleChan = make (chan map[string]*iface.StatusMessage)
     pollingCancelChan = make(chan []string)
     
     currentRequests = make(map[string]*currentRequestsNode)

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
          resources[value] = &resourceInfo{false, 0, 0, "None", "None", "", 0}
     }
     resourcesLock.Unlock()
}

func updateDatabase(term bool) {
     //This updates the json database file with the information in the
     //resource map.
     heckleDaemon.DaemonLog.Log("Updating persistant json resource database file.")
     databaseFile, error := os.OpenFile(daemon.FileDir + "resources.db", os.O_RDWR, 0777)
     //databaseFile, error := os.Create(daemon.FileDir + "ResourceDatabase")
     heckleDaemon.DaemonLog.LogError("Unable to open resource database file for reading and writing.", error)
     
     intError := syscall.Flock(databaseFile.Fd(), 2) //2 is exclusive lock
     if intError != 0 {
          heckleDaemon.DaemonLog.LogError("Unable to lock resource database file.", os.NewError("Flock Syscall Failed"))
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
          heckleDaemon.DaemonLog.Log("Exiting gracefully, bye bye.")
          os.Exit(0)
     }
     
     intError = syscall.Flock(databaseFile.Fd(), 8) //8 is unlock
     if intError != 0 {
          heckleDaemon.DaemonLog.LogError("Unable to unlock resource database for reading.", os.NewError("Flock Syscall Failed"))
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
     heckleDaemon.DaemonLog.Log("Initializing resource data from resource database file.")
     
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
          heckleDaemon.DaemonLog.LogError("Not enough open nodes to allocate, cancelling allocation", os.NewError("Not Enough Nodes"))
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
     heckleDaemon.DaemonLog.Log("Checking node list to see if all can be allocated.")

     resourcesLock.Lock()
     defer resourcesLock.Unlock()
     for _, value := range nodeList {
          if val, ok := resources[value] ; !ok || val.Allocated {
               listOk = false
          }
     }
     
     if !listOk {
          heckleDaemon.DaemonLog.LogError("Some of the nodes asked for are allocated, cancelling allocation.", os.NewError("List Nodes Taken"))
     } else {
          for _, value := range nodeList {
               resources[value].Allocate(owner, image, allocationNum)
          }
     }
     return
}

func allocateList(writer http.ResponseWriter, request *http.Request) {
     //This is an http handler function to deal with allocation list requests.
     //It grabs the list from the message, makes a new lsit of all available
     //nodes within that original list.  Gets an allocation number, and adds
     //them to the current requests map.
     heckleDaemon.DaemonLog.Log("Allocating a list of nodes.")
     listMsg := new(iface.Listmsg)
     request.ProtoMinor = 0

     username, authed, _ := heckleDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          heckleDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleDaemon.DaemonLog.LogError("Unable to read all from allocate list POST.", error)
     
     error = request.Body.Close()
     heckleDaemon.DaemonLog.LogError("Failed to close allocation list request body.", error)
     
     error = json.Unmarshal(someBytes, &listMsg)
     heckleDaemon.DaemonLog.LogError("Unable to unmarshal allocation list.", error)
     
     allocationNumberLock.Lock()
     tmpAllocationNumber := allocationNumber
     
     allocationListOk := checkNodeList(listMsg.Addresses, username, listMsg.Image, tmpAllocationNumber)
     if !allocationListOk {
          allocationNumberLock.Unlock()
          return
     }
     allocationNumber++
     allocationNumberLock.Unlock()
     
     heckleToAllocateChan<- iface.Listmsg{listMsg.Addresses, listMsg.Image, 0}
     
     currentRequestsLock.Lock()
     for _, value := range listMsg.Addresses {
          currentRequests[value] = &currentRequestsNode{username, listMsg.Image, "Building", tmpAllocationNumber, listMsg.ActivityTimeout, false, 0, []iface.InfoMsg{}}
     }
     currentRequestsLock.Unlock()
     
     js, _ := json.Marshal(tmpAllocationNumber)
     writer.Write(js)
     
     updateDatabase(false)
}

func allocateNumber(writer http.ResponseWriter, request *http.Request) {
     //This is just an http function that deals with allocation number requests.
     //It grabs the number, gets a list of that number or less of nodes, gets
     //an allocation number, and adds them to the current requests map.
     heckleDaemon.DaemonLog.Log("Allocating a number of nodes.")
     numMsg := new(iface.Nummsg)
     request.ProtoMinor = 0
     
     username, authed, _ := heckleDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          heckleDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleDaemon.DaemonLog.LogError("Unable to read all from allocate list POST.", error)
     
     error = request.Body.Close()
     heckleDaemon.DaemonLog.LogError("Failed to close allocation number request body.", error)
     
     error = json.Unmarshal(someBytes, &numMsg)
     heckleDaemon.DaemonLog.LogError("Unable to unmarshal allocation list.", error)
     
     allocationNumberLock.Lock()
     tmpAllocationNumber := allocationNumber
     
     allocationList := getNumNodes(numMsg.NumNodes, username, numMsg.Image, tmpAllocationNumber)
     if len(allocationList) == 0 {
          allocationNumberLock.Unlock()
          return
     }
     allocationNumber++
     allocationNumberLock.Unlock()
     
     heckleToAllocateChan<- iface.Listmsg{allocationList, numMsg.Image, 0}
     
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
     heckleDaemon.DaemonLog.Log("Starting allocation go routine.")
     
     /*fmComm, error := fnet.NewCommunication(heckleDaemon.FileDir, heckleheckleDaemon.Cfg.Data["username"], Daemon.Cfg.Data["password"])
     heckleDaemon.DaemonLog.LogError("Failed to make new communication structure in heckle for flunkymaster.", error)

     pComm, error := fnet.NewCommunication(heckleDaemon.FileDir, heckleheckleDaemon.Cfg.Data["username"], Daemon.Cfg.Data["password"])
     heckleDaemon.DaemonLog.LogError("Failed to make new communication structure in heckle for power.", error)

     fs, error := fmComm.SetupClient("flunky")
     heckleDaemon.DaemonLog.LogError("Failed to setup heckle to flunkymaster communication.", error)

     ps, error := pComm.SetupClient("power")
     heckleDaemon.DaemonLog.LogError("Failed to setup heckle to power communication.", error)*/
     
     for i := range heckleToAllocateChan {
          cm := new(iface.Ctlmsg)
          cm.Image = i.Image
          cm.Addresses = i.Addresses
          // FIXME: need to add in extradata
          js, _ := json.Marshal(cm)
          buf := bytes.NewBufferString(string(js))
          _, err := fs.Post("/ctl", buf)
          heckleDaemon.DaemonLog.LogError("Failed to post for allocation of nodes.", err)
          
          if err == nil {
               allocateToPollingChan<-i.Addresses
               
               js, _ = json.Marshal(i.Addresses)
               buf = bytes.NewBufferString(string(js))
               _, err = ps.Post("/reboot", buf)
               heckleDaemon.DaemonLog.LogError("Failed to post for reboot of nodes in allocation go routine.", err)
          }
     }
     close(allocateToPollingChan)
}

func addToPollList (pollAddressesLock *sync.Mutex, pollAddresses *[]string) {
     for i := range allocateToPollingChan {
          heckleDaemon.DaemonLog.Log("Adding nodes to polling list.")
          pollAddressesLock.Lock()
          *pollAddresses = append(*pollAddresses, i...)
          pollAddressesLock.Unlock()
     }
}

func deleteFromPollList (pollAddressesLock *sync.Mutex, pollAddresses *[]string) {
     for i := range pollingCancelChan {
          heckleDaemon.DaemonLog.Log("Removing nodes from polling list.")
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
     heckleDaemon.DaemonLog.Log("Starting polling go routine.")
     pollAddresses := []string{}
     pollingOutletStatus := make(map[string]string)
     var pollAddressesLock sync.Mutex
     //bs := fnet.NewBuildServer("http://" + heckleDaemon.Cfg.Data["username"] + ":" + heckleDaemon.Cfg.Data["password"] + "@" + heckleDaemon.Cfg.Data["allocationServer"], false)
     //rs := fnet.NewBuildServer("http://" + heckleDaemon.Cfg.Data["username"] + ":" + heckleDaemon.Cfg.Data["password"] + "@" + heckleDaemon.Cfg.Data["powerServer"], false)
     pollTime := time.Seconds()
     
     go addToPollList(&pollAddressesLock, &pollAddresses)
     go deleteFromPollList(&pollAddressesLock, &pollAddresses)
     
     for ;  ; time.Sleep(10000000000){
          heckleDaemon.DaemonLog.Log("Polling for messages from flunky master and power daemons.")
          statRequest := new(iface.Ctlmsg)
          pollAddressesLock.Lock()
          statRequest.Addresses = pollAddresses
          pollAddressesLock.Unlock()
          if len(statRequest.Addresses) != 0 {
               statRequest.Time = pollTime

               var statmap map[string]*iface.StatusMessage     
               sRjs, _ := json.Marshal(statRequest)
               reqbuf := bytes.NewBufferString(string(sRjs))
               ret, _ := fs.Post("/status", reqbuf)
               pollTime = time.Seconds()
               json.Unmarshal(ret, &statmap)
               
               outletStatus := make(map[string]string)
               sRjs, _ = json.Marshal(pollAddresses)
               reqbuf = bytes.NewBufferString(string(sRjs))
               ret, _ = ps.Post("/status", reqbuf)
               json.Unmarshal(ret, &outletStatus)

               for key, value := range statmap {
                    if _, ok := pollingOutletStatus[key] ; !ok {
                         value.Info = append(value.Info, iface.InfoMsg{time.Seconds(), "Power outlet for this node is " + outletStatus[key] + ".", "Info"})
                         pollingOutletStatus[key] = outletStatus[key]
                    } else if pollingOutletStatus[key] != outletStatus[key] {
                         value.Info = append(value.Info, iface.InfoMsg{time.Seconds(), "Power outlet for this node is " + outletStatus[key] + ".", "Info"})
                         pollingOutletStatus[key] = outletStatus[key]
                    }
               }
               heckleDaemon.DaemonLog.Log("Sending status messages to main routine.")
               pollingToHeckleChan<- statmap
          }
     }
}

func findNewNode(owner string, image string, activityTimeout int64, tmpAllocationNumber uint64) {
     //This function finds a single node for someone whose node got canceled and requested
     //a number of nodes.  It then sends this node to the allocation thread and tosses it
     //on the current requests map.
     heckleDaemon.DaemonLog.Log("Finding a replacement node for a node that has failed or is already allocated.")
     allocationList := getNumNodes(1, owner, image, tmpAllocationNumber)
     heckleToAllocateChan<- iface.Listmsg{allocationList, image, 0}
     
     currentRequestsLock.Lock()
     for _, value := range allocationList {
          currentRequests[value] = &currentRequestsNode{owner, image, "Building", tmpAllocationNumber, activityTimeout, true, 0, []iface.InfoMsg{}}
     }
     currentRequestsLock.Unlock()
     
     updateDatabase(false)
}

func DumpCall(w http.ResponseWriter, req *http.Request) {
        heckleDaemon.DaemonLog.LogHttp(req)
        req.ProtoMinor = 0
        /*username, authed, _ := heckleDaemon.AuthN.HTTPAuthenticate(req)
        if !authed {
                heckleDaemon.DaemonLog.LogError(fmt.Sprintf("User Authentications for %s failed", username), os.NewError("Access Denied"))
                return
        }*/
        resourcesLock.Lock()
        tmp, err := json.Marshal(resources)
        resourcesLock.Unlock()
        heckleDaemon.DaemonLog.LogError("Cannot Marshal heckle data", err)
        _, err = w.Write(tmp)
        if err != nil {
                http.Error(w, "Cannot write to socket", 500)
        }
}

func status(writer http.ResponseWriter, request *http.Request) {
     //This is an http handler function to deal with allocation status requests.
     //if the host has ownership of the allocation number we send back a map
     //of node names and a status message type.
     heckleDaemon.DaemonLog.Log("Sending allocation status to client.")
     allocationStatus := make(map[string]*iface.StatusMessage)
     allocationNumber := uint64(0)
     request.ProtoMinor = 0
     
     username, authed, admin := heckleDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          heckleDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleDaemon.DaemonLog.LogError("Unable to read all from allocation status POST.", error)
     
     error = request.Body.Close()
     heckleDaemon.DaemonLog.LogError("Failed to close allocation status request body.", error)
     
     error = json.Unmarshal(someBytes, &allocationNumber)
     heckleDaemon.DaemonLog.LogError("Unable to unmarshal allocation number for status request.", error)
     
     currentRequestsLock.Lock()
     for key, value := range currentRequests {
          if allocationNumber == value.AllocationNumber {
               if value.User == username || admin{
                    sm := &iface.StatusMessage{value.Status, value.LastActivity, value.Info}
                    allocationStatus[key] = sm
                    value.Info = []iface.InfoMsg{}
               } else {
                    heckleDaemon.DaemonLog.LogError("Cannot request status of allocations that do not beling to you.", os.NewError("Access Denied"))
                    currentRequestsLock.Unlock()
                    return
               }
          }
     }
     currentRequestsLock.Unlock()
     
     jsonStat, error := json.Marshal(allocationStatus)
     heckleDaemon.DaemonLog.LogError("Unable to marshal allocation status response.", error)
     
     _, error = writer.Write(jsonStat)
     heckleDaemon.DaemonLog.LogError("Unable to write allocation status response.", error)
}

func freeAllocation(writer http.ResponseWriter, request *http.Request) {
     //This function allows a user, if it owns the allocation, to free an allocation
     //number and all associated nodes.  It resets the resource map and current
     //requests map.
     heckleDaemon.DaemonLog.Log("Freeing allocation number given by client.")
     //rs := fnet.NewBuildServer("http://" + heckleDaemon.Cfg.Data["username"] + ":" + heckleDaemon.Cfg.Data["password"] + "@" + heckleDaemon.Cfg.Data["powerServer"], false)
     allocationNumber := uint64(0)
     request.ProtoMinor = 0
     
     username, authed, admin := heckleDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          heckleDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleDaemon.DaemonLog.LogError("Unable to read all from allocation status POST.", error)
     
     error = request.Body.Close()
     heckleDaemon.DaemonLog.LogError("Failed to close free allocation request body.", error)
     
     error = json.Unmarshal(someBytes, &allocationNumber)
     heckleDaemon.DaemonLog.LogError("Unable to unmarshal allocation number for freeing.", error)
     
     powerDown := []string{}
     
     currentRequestsLock.Lock()
     resourcesLock.Lock()
     found := false
     for key, value := range resources {
          if allocationNumber == value.AllocationNumber {
               if username == value.Owner || admin {
                    value.Reset()
                    powerDown = append(powerDown, key)
                    currentRequests[key] = nil, false
                    found = true
               } else {
                    heckleDaemon.DaemonLog.LogError("Cannot free allocations that do not belong to you.", os.NewError("Access Denied"))
                    currentRequestsLock.Unlock()
                    resourcesLock.Unlock()
                    return
               }
          }
     }
     currentRequestsLock.Unlock()
     resourcesLock.Unlock()
     
     if !found {
          heckleDaemon.DaemonLog.LogError("Allocation number does not exist.", os.NewError("Wrong Number"))
          return
     }
     
     pollingCancelChan<- powerDown //Needed because polling will continue to poll if allocation is freed during allocation. 
     
     js, _ := json.Marshal(powerDown)
     buf := bytes.NewBufferString(string(js))
     _, err := ps.Post("/off", buf)
     heckleDaemon.DaemonLog.LogError("Failed to post for reboot of nodes in free allocation number.", err)
     
     updateDatabase(false)
}

func increaseTime(writer http.ResponseWriter, request *http.Request) {
     //This function allows a user, if it owns the allocation, to free an allocation
     //number and all associated nodes.  It resets the resource map and current
     //requests map.
     heckleDaemon.DaemonLog.Log("Increasing allocation time on an allocation number.")
     timeIncrease := int64(0)
     request.ProtoMinor = 0
     
     username, authed, _ := heckleDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          heckleDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleDaemon.DaemonLog.LogError("Unable to read all from increase time POST.", error)
     
     error = request.Body.Close()
     heckleDaemon.DaemonLog.LogError("Failed to close free increase time request body.", error)
     
     error = json.Unmarshal(someBytes, &timeIncrease)
     heckleDaemon.DaemonLog.LogError("Unable to unmarshal time increase in related handler func.", error)
     
     resourcesLock.Lock()
     for _, value := range resources {
          if value.Owner == username {
               value.AllocationEndTime = value.AllocationEndTime + timeIncrease
          }
     }
     resourcesLock.Unlock()
     
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
          if value.Allocated && value.AllocationEndTime <= time.Seconds() {
               found = true
               powerDown = append(powerDown, key)
               currentRequests[key] = nil, false
               value.Reset()                    
          }
     }
     
     currentRequestsLock.Unlock()
     resourcesLock.Unlock()
     
     if found {
          heckleDaemon.DaemonLog.Log("Found allocation time outs, deallocating them.")
          //rs := fnet.NewBuildServer("http://" + heckleDaemon.Cfg.Data["username"] + ":" + heckleDaemon.Cfg.Data["password"] + "@" + heckleDaemon.Cfg.Data["powerServer"], false)
          
          js, _ := json.Marshal(powerDown)
          buf := bytes.NewBufferString(string(js))
          _, err := ps.Post("/off", buf)
          heckleDaemon.DaemonLog.LogError("Failed to post for reboot of nodes in allocation time outs.", err)
          
          updateDatabase(false)
     }
}

func freeNode(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     heckleDaemon.DaemonLog.Log("Freeing a specific node given by client.")
     //rs := fnet.NewBuildServer("http://" + heckleDaemon.Cfg.Data["username"] + ":" + heckleDaemon.Cfg.Data["password"] + "@" + heckleDaemon.Cfg.Data["powerServer"], false)
     var node string
     request.ProtoMinor = 0
     
     username, authed, _ := heckleDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          heckleDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleDaemon.DaemonLog.LogError("Unable to read all from allocation status POST.", error)
     
     error = request.Body.Close()
     heckleDaemon.DaemonLog.LogError("Failed to close free node request body.", error)
     
     error = json.Unmarshal(someBytes, &node)
     heckleDaemon.DaemonLog.LogError("Unable to unmarshal node to be unallocated.", error)
     
     currentRequestsLock.Lock()
     resourcesLock.Lock()
     
     if val, ok := resources[node] ; !ok || val.Owner != username {
          heckleDaemon.DaemonLog.LogError("Access denied, cannot free nodes that do not belong to you.", os.NewError("Access Denied"))
          currentRequestsLock.Unlock()
          resourcesLock.Unlock()
          return
     }
     
     resources[node].Reset()  
     currentRequests[node] = nil, false
     
     currentRequestsLock.Unlock()
     resourcesLock.Unlock()
     
     js, _ := json.Marshal([]string{node})
     buf := bytes.NewBufferString(string(js))
     _, err := ps.Post("/off", buf)
     heckleDaemon.DaemonLog.LogError("Failed to post for reboot of nodes in free node.", err)
     
     updateDatabase(false)
}

func listenAndServeWrapper() {
     //This branches off another thread to loop through listening and serving http requests.
     heckleDaemon.DaemonLog.Log("Starting HTTP listen go routine.")
     error := http.ListenAndServe(":" + heckleDaemon.Cfg.Data["hecklePort"], nil)
     heckleDaemon.DaemonLog.LogError("Failed to listen on http socket.", error)
     
     updateDatabase(true)
}

func freeCurrentRequests() {
     //If a node is ready or canceled and there are no more info messages remove it from being tracked in
     //current requests.  No need to update resources structure because this is done in other places.
     currentRequestsLock.Lock()
     for key, value := range currentRequests {
          if (value.Status == "Cancel" || value.Status == "Ready") && (len(value.Info) == 0) {
               currentRequests[key] = nil, false
          }
     }
     currentRequestsLock.Unlock()
}

func dealWithBrokenNode(node string) {
     heckleDaemon.DaemonLog.Log("Sending broken node to diagnostic daemon.")
     resourcesLock.Lock()
     resources[node].Broken()
     resourcesLock.Unlock()
     
     //rs := fnet.NewBuildServer("http://" + heckleDaemon.Cfg.Data["username"] + ":" + heckleDaemon.Cfg.Data["password"] + "@" + heckleDaemon.Cfg.Data["powerServer"], false)
     
     js, _ := json.Marshal([]string{node})
     buf := bytes.NewBufferString(string(js))
     _, err := ps.Post("/off", buf)
     heckleDaemon.DaemonLog.LogError("Failed to post for reboot of nodes in free node.", err)
     
     //pass node off to diagnosing process
     updateDatabase(false)
}

func interpretPollMessages() {
     heckleDaemon.DaemonLog.Log("Starting message interpreter go routine.")
     for i := range pollingToHeckleChan {
          heckleDaemon.DaemonLog.Log("Interpreting poll new poll messages.")
          nodesToRemove := []string{}
          for key, value := range i {
               currentRequestsLock.Lock()
               currentRequests[key].Status = value.Status
               currentRequests[key].LastActivity = value.LastActivity

               if len(value.Info) != 0 {
                    currentRequests[key].Info = append(currentRequests[key].Info, value.Info...)
               }    
                    
               if value.Status == "Cancel" || time.Seconds() - value.LastActivity >= currentRequests[key].ActivityTimeout {
                    nodesToRemove = append(nodesToRemove, key)
                    go dealWithBrokenNode(key)

                    if currentRequests[key].TryOnFail {
                         findNewNode(currentRequests[key].User, currentRequests[key].Image, currentRequests[key].ActivityTimeout, currentRequests[key].AllocationNumber)
                    }
               }
               if value.Status == "Ready" {
                    nodesToRemove = append(nodesToRemove, key)
               }
               currentRequestsLock.Unlock()
          }
          pollingCancelChan<- nodesToRemove
     }    
}

func outletStatus(writer http.ResponseWriter, request *http.Request) {
     heckleDaemon.DaemonLog.Log("Executing power command.")
     //rs := fnet.NewBuildServer("http://" + heckleDaemon.Cfg.Data["username"] + ":" + heckleDaemon.Cfg.Data["password"] + "@" + heckleDaemon.Cfg.Data["powerServer"], false)
     request.ProtoMinor = 0
     
     _, authed, admin := heckleDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          heckleDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !admin {
          heckleDaemon.DaemonLog.LogError("No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleDaemon.DaemonLog.LogError("Unable to read all from outlet status POST.", error)
     
     error = request.Body.Close()
     heckleDaemon.DaemonLog.LogError("Failed to close outlet status request body.", error)

     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = ps.Post("/status", buf)
     heckleDaemon.DaemonLog.LogError("Failed to post for status of outlets to radixPower.go.", error)

     _, error = writer.Write(someBytes)
     heckleDaemon.DaemonLog.LogError("Unable to write outlet status response in heckle.", error)
}

func nodeStatus(writer http.ResponseWriter, request *http.Request) {
     heckleDaemon.DaemonLog.Log("Sending back node status.")
     response := ""
     request.ProtoMinor = 0
     
     _, authed, _ := heckleDaemon.AuthN.HTTPAuthenticate(request)
     
     if !authed {
          heckleDaemon.DaemonLog.LogError("Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     resourcesLock.Lock()
     for key, value := range resources {
          if value.Allocated {
               response = response + "NODE: " + key + "\tALLOCATED: yes\tALLOCATION NUM: " + strconv.Uitoa64(value.AllocationNumber) + "\tOWNER: " + value.Owner + "\tIMAGE: " + value.Image + "\tTIME ALLOCATED: " + time.SecondsToLocalTime(value.TimeAllocated).Format(time.UnixDate) + "\tALLOCATION END: " + time.SecondsToLocalTime(value.AllocationEndTime).Format(time.UnixDate) + "\tCOMMENTS: " + value.Comments + "\n\n"

          }
     }
     resourcesLock.Unlock()

     _, error := writer.Write([]byte(response))
     heckleDaemon.DaemonLog.LogError("Unable to write node status response in heckle.", error)
}

func main() {
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
     
     for {
          select {
               case sig := <-signal.Incoming:
                    if sig.(os.UnixSignal) == syscall.SIGTERM || sig.(os.UnixSignal) == syscall.SIGINT || sig.(os.UnixSignal) == syscall.SIGQUIT || sig.(os.UnixSignal) == syscall.SIGTSTP {
                         updateDatabase(true)
                    }
               default:
                    runtime.Gosched()
                    allocationTimeouts()
                    freeCurrentRequests()
                    //fmt.Fprintf(os.Stdout, "Go routines, %d.\n", runtime.Goroutines())
          }
     }
}
