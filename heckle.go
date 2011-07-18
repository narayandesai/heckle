package main

import (
     "fmt"
     "json"
     "os"
     "time"
     "io/ioutil"
     "strings"
     "http"
     "bytes"
     "sync"
     "runtime"
     "./flunky"
     "./heckleTypes"
     "./heckleFuncs"
)

var currentRequests           map[string]*heckleTypes.CurrentRequestsNode
var cfgOptions                map[string]string
var resources                 map[string]*heckleTypes.ResourceInfo
var allocationNumber          uint64
var heckleToAllocateChan      chan heckleTypes.Listmsg
var allocateToPollingChan     chan []string
var pollingToHeckleChan       chan map[string]*heckleTypes.StatusMessage
var pollingCancelChan         chan []string
var currentRequestsLock       sync.Mutex
var resourcesLock             sync.Mutex
var allocationNumberLock      sync.Mutex

func init() {
     //This populates the cfgOptions map with the json cfg file.
     cfgOptions = make(map[string]string)
     heckleToAllocateChan = make(chan heckleTypes.Listmsg)
     allocateToPollingChan = make(chan []string)
     pollingToHeckleChan = make (chan map[string]*heckleTypes.StatusMessage)
     pollingCancelChan = make(chan []string)
     
     currentRequests = make(map[string]*heckleTypes.CurrentRequestsNode)

     allocationNumber = 1
     getResources()

     cfgFile, error := os.Open("heckle.cfg")
     heckleFuncs.PrintError("ERROR: Unable to open heckle.cfg for reading.", error)
     
     someBytes, error := ioutil.ReadAll(cfgFile)
     heckleFuncs.PrintError("ERROR: Unable to read from file heckle.cfg.", error)
     
     error = cfgFile.Close()
     heckleFuncs.PrintError("ERROR: Failed to close heckle.cfg.", error)
     
     error = json.Unmarshal(someBytes, &cfgOptions)
     heckleFuncs.PrintError("ERROR: Failed to unmarshal data read from heckle cfg file.", error)
}

func resetResources(resourceNames []string) {
     //This is a function to accomodate reseting the json database.
     //it just creates a new resource map with all the resourceNames
     //entries in it.
     resourcesLock.Lock()
     resources = make(map[string]*heckleTypes.ResourceInfo)
     
     for _, value := range resourceNames {
          resources[value] = &heckleTypes.ResourceInfo{false, 0, 0, "None", "None", "", 0}
     }
     resourcesLock.Unlock()
}

func updateDatabase() {
     //This updates the json database file with the information in the
     //resource map.
     //databaseFile, error := os.OpenFile("ResourceDatabase", os.O_RDWR | os.O_TRUNC, 0777)
     databaseFile, error := os.Create("ResourceDatabase")
     heckleFuncs.PrintError("ERROR: Unable to open resource database file for writing.", error)
     
     resourcesLock.Lock()
     js, error := json.Marshal(resources)
     heckleFuncs.PrintError("ERROR: Failed to marshal resources for resources database file.", error)
     resourcesLock.Unlock()
     
     _, error = databaseFile.Write(js)
     heckleFuncs.PrintError("ERROR: Failed to write resources to resources database file.", error)
     
     error = databaseFile.Close()
     heckleFuncs.PrintError("ERROR: Failed to close resources database file.", error)
}

func resetResourceDatabase() {
     //This function is intended for admins to reset the json database file.
     resourceFile, error := os.Open("resources")
     heckleFuncs.PrintError("ERROR: Failed to open resources file.", error)
     
     someBytes, error := ioutil.ReadAll(resourceFile)
     heckleFuncs.PrintError("ERROR: Failed to read resources file.", error)
     
     error = resourceFile.Close()
     heckleFuncs.PrintError("ERROR: Failed to close resource file.", error)
     
     resourceNames := strings.Split(string(someBytes), "\n")
     resourceNames = resourceNames[:len(resourceNames)-1]
     
     resetResources(resourceNames)
     updateDatabase()
}

func getResources() {
     //This function populated the resources map from the json database file.
     databaseFile, error := os.Open("ResourceDatabase")
     heckleFuncs.PrintError("ERROR: Failed to open resource database file for reading.", error)
     
     someBytes, error := ioutil.ReadAll(databaseFile)
     heckleFuncs.PrintError("ERROR: Failed to read from resource database file.", error)
     
     error = databaseFile.Close()
     heckleFuncs.PrintError("ERROR: Failed to close resource database file.", error)
     
     resourcesLock.Lock()
     resources = make(map[string]*heckleTypes.ResourceInfo)
     error = json.Unmarshal(someBytes, &resources)
     heckleFuncs.PrintError("ERROR: Failed to unmarshal data read from resource database file.", error)
     resourcesLock.Unlock()
}

func printResources(onlyAllocated bool) {
     //This functionw ill print all resources.  If true is passed in
     //it only shows allocated nodes.
     resourcesLock.Lock()
     for key, value := range resources {
          if onlyAllocated {
               if value.Allocated {
                    printResource(key, value)
               }
          } else {
               printResource(key, value)
          }
     }
     resourcesLock.Unlock()
}

func printResource(node string, resource *heckleTypes.ResourceInfo) {
     //This function will print an individual resource from the resource map.
     fmt.Fprintf(os.Stdout, "NODE: %s\tALLOCATED: ", node)
     if resource.Allocated {
          fmt.Fprintf(os.Stdout, "Yes\t")
     } else {
          fmt.Fprintf(os.Stdout, "No\t")
     }
     fmt.Fprintf(os.Stdout, "OWNER: %s\nIMAGE: %s\nTIME ALLOCATED: %s\nALLOCATION END: %s\nCOMMENTS: %s\n\n", resource.Owner, resource.Image, time.SecondsToLocalTime(resource.TimeAllocated).Format(time.UnixDate), time.SecondsToLocalTime(resource.AllocationEndTime).Format(time.UnixDate), resource.Comments)
}

func getFreeNodes(numNodes int, owner string, image string, allocationNum uint64) []string {
     //This function is for the http allocate number of nodes function.
     //It will create a list of that many free nodes or as many free
     //nodes as it can, upate the resource map accordingly, and return
     //the list.
     tmpNodeList := []string{}
     index := 0
     
     resourcesLock.Lock()
     for key, value := range resources {
          if index < numNodes && !value.Allocated {
               tmpNodeList = append(tmpNodeList, key)
               value.Allocate(owner, image, allocationNum)
               index++
          }
     }
     resourcesLock.Unlock()
   
     if index != numNodes {
          fmt.Fprintf(os.Stderr, "ERROR: Not enough open nodes to allocate, allocating %d nodes instead.\n", (index+1))
     }
     
     return tmpNodeList
}

func checkNodeList(nodeList []string, owner string, image string, allocationNum uint64) []string {
     //This function is for the http allocate list function.  It checks
     //the list requested and allocated as many nodes as are available
     //that are in that list.  It updates the resource map accordingly,
     //and returns the new list.
     tmpList := []string{}
     resourcesLock.Lock()
     for _, value := range nodeList {
          if !resources[value].Allocated {
               tmpList = append(tmpList, value)
          }
     }
     
     for _, value := range tmpList {
          resources[value].Allocate(owner, image, allocationNum)
     }
     resourcesLock.Unlock()
     
     if len(tmpList) != len(nodeList) {
          fmt.Fprintf(os.Stderr, "Some of the nodes asked for are allocated, allocating %d nodes.\n", len(tmpList))
     }
     
     return tmpList
}

func allocateList(writer http.ResponseWriter, request *http.Request) {
     //This is an http handler function to deal with allocation list requests.
     //It grabs the list from the message, makes a new lsit of all available
     //nodes within that original list.  Gets an allocation number, and adds
     //them to the current requests map.
     listMsg := new(heckleTypes.Listmsg)
     request.ProtoMinor = 0

     username, authed, _ := heckleFuncs.Authenticate(request.Header.Get("Authorization"), cfgOptions["userDatabasePath"])
     
     if !authed {
          heckleFuncs.PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleFuncs.PrintError("ERROR: Unable to read all from allocate list POST.", error)
     
     error = request.Body.Close()
     heckleFuncs.PrintError("ERROR: Failed to close allocation list request body.", error)
     
     error = json.Unmarshal(someBytes, &listMsg)
     heckleFuncs.PrintError("ERROR: Unable to unmarshal allocation list.", error)
     
     allocationNumberLock.Lock()
     tmpAllocationNumber := allocationNumber
     allocationNumber++
     allocationNumberLock.Unlock()
     
     allocationList := checkNodeList(listMsg.Addresses, username, listMsg.Image, tmpAllocationNumber)
     heckleToAllocateChan<- heckleTypes.Listmsg{allocationList, listMsg.Image, 0}
     
     currentRequestsLock.Lock()
     for _, value := range allocationList {
          currentRequests[value] = &heckleTypes.CurrentRequestsNode{username, listMsg.Image, "Building", tmpAllocationNumber, listMsg.ActivityTimeout, false, 0, []heckleTypes.InfoMsg{}}
     }
     currentRequestsLock.Unlock()
     
     js, _ := json.Marshal(tmpAllocationNumber)
     writer.Write(js)
     
     updateDatabase()
}

func allocateNumber(writer http.ResponseWriter, request *http.Request) {
     //This is just an http function that deals with allocation number requests.
     //It grabs the number, gets a list of that number or less of nodes, gets
     //an allocation number, and adds them to the current requests map.
     numMsg := new(heckleTypes.Nummsg)
     request.ProtoMinor = 0
     
     username, authed, _ := heckleFuncs.Authenticate(request.Header.Get("Authorization"), cfgOptions["userDatabasePath"])
     
     if !authed {
          heckleFuncs.PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleFuncs.PrintError("ERROR: Unable to read all from allocate list POST.", error)
     
     error = request.Body.Close()
     heckleFuncs.PrintError("ERROR: Failed to close allocation number request body.", error)
     
     error = json.Unmarshal(someBytes, &numMsg)
     heckleFuncs.PrintError("ERROR: Unable to unmarshal allocation list.", error)
     
     allocationNumberLock.Lock()
     tmpAllocationNumber := allocationNumber
     allocationNumber++
     allocationNumberLock.Unlock()
     
     allocationList := getFreeNodes(numMsg.NumNodes, username, numMsg.Image, tmpAllocationNumber)
     heckleToAllocateChan<- heckleTypes.Listmsg{allocationList, numMsg.Image, 0}
     
     currentRequestsLock.Lock()
     for _, value := range allocationList {
          currentRequests[value] = &heckleTypes.CurrentRequestsNode{username, numMsg.Image, "Building", tmpAllocationNumber, numMsg.ActivityTimeout, true, 0, []heckleTypes.InfoMsg{}}
     }
     currentRequestsLock.Unlock()
     
     js, _ := json.Marshal(tmpAllocationNumber)
     writer.Write(js)
     
     updateDatabase()
}

func allocate() {
     //This is the allocate thread.  It set up a client for ctl messages to
     //flunky master.  On each iteration it grabs new nodes from heckle to be
     //allocated and send them off to flunkymaster.
     fs := flunky.NewBuildServer(cfgOptions["allocationServer"], false, "heckle", cfgOptions["heckle"])
     rs := flunky.NewBuildServer(cfgOptions["powerServer"], false, "heckle", cfgOptions["heckle"])
     
     for i := range heckleToAllocateChan {
          cm := new(heckleTypes.Ctlmsg)
          cm.Image = i.Image
          cm.Addresses = i.Addresses
          // FIXME: need to add in extradata
          js, _ := json.Marshal(cm)
          buf := bytes.NewBufferString(string(js))
          _, err := fs.Post("/ctl", buf)
          heckleFuncs.PrintError("ERROR: Failed to post for allocation of nodes.", err)
          
          if err == nil {
               allocateToPollingChan<-i.Addresses
               
               js, _ = json.Marshal(i.Addresses)
               buf = bytes.NewBufferString(string(js))
               _, err = rs.Post("/reboot", buf)
               heckleFuncs.PrintError("ERROR: Failed to post for reboot of nodes in allocation go routine.", err)
          }
     }
}

func addToPollList (pollAddressesLock *sync.Mutex, pollAddresses *[]string) {
     for i := range allocateToPollingChan {
          pollAddressesLock.Lock()
          *pollAddresses = append(*pollAddresses, i...)
          pollAddressesLock.Unlock()
     }
}

func deleteFromPollList (pollAddressesLock *sync.Mutex, pollAddresses *[]string) {
     for i := range pollingCancelChan {
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
     pollAddresses := []string{}
     var pollAddressesLock sync.Mutex
     bs := flunky.NewBuildServer(cfgOptions["pollingServer"], false, "heckle", cfgOptions["heckle"])
     rs := flunky.NewBuildServer(cfgOptions["powerServer"], false, "heckle", cfgOptions["heckle"])
     pollTime := time.Seconds()
     
     go addToPollList(&pollAddressesLock, &pollAddresses)
     go deleteFromPollList(&pollAddressesLock, &pollAddresses)
     
     for ;  ; time.Sleep(10000000000){
          statRequest := new(heckleTypes.Ctlmsg)
          pollAddressesLock.Lock()
          statRequest.Addresses = pollAddresses
          pollAddressesLock.Unlock()
          statRequest.Time = pollTime

          var statmap map[string]*heckleTypes.StatusMessage
               
          sRjs, _ := json.Marshal(statRequest)
          reqbuf := bytes.NewBufferString(string(sRjs))
          ret, _ := bs.Post("/status", reqbuf)
          pollTime = time.Seconds()
          json.Unmarshal(ret, &statmap)
          
          outletStatus := make(map[string]string)
          sRjs, _ = json.Marshal(pollAddresses)
          reqbuf = bytes.NewBufferString(string(sRjs))
          ret, _ = rs.Post("/status", reqbuf)
          json.Unmarshal(ret, &outletStatus)

          for key, value := range statmap {
               value.Info = append(value.Info, heckleTypes.InfoMsg{time.Seconds(), "Power outlet for this node is " + outletStatus[key] + ".", "Info"})
          }
          
          pollingToHeckleChan<- statmap
     }
}

func findNewNode(owner string, image string, activityTimeout int64, tmpAllocationNumber uint64) {
     //This function finds a single node for someone whose node got canceled and requested
     //a number of nodes.  It then sends this node to the allocation thread and tosses it
     //on the current requests map.
     allocationList := getFreeNodes(1, owner, image, tmpAllocationNumber)
     heckleToAllocateChan<- heckleTypes.Listmsg{allocationList, image, 0}
     
     currentRequestsLock.Lock()
     for _, value := range allocationList {
          currentRequests[value] = &heckleTypes.CurrentRequestsNode{owner, image, "Building", tmpAllocationNumber, activityTimeout, true, 0, []heckleTypes.InfoMsg{}}
     }
     currentRequestsLock.Unlock()
     
     updateDatabase()
}

func status(writer http.ResponseWriter, request *http.Request) {
     //This is an http handler function to deal with allocation status requests.
     //if the host has ownership of the allocation number we send back a map
     //of node names and a status message type.
     allocationStatus := make(map[string]*heckleTypes.StatusMessage)
     allocationNumber := uint64(0)
     request.ProtoMinor = 0
     
     username, authed, admin := heckleFuncs.Authenticate(request.Header.Get("Authorization"), cfgOptions["userDatabasePath"])
     
     if !authed {
          heckleFuncs.PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleFuncs.PrintError("ERROR: Unable to read all from allocation status POST.", error)
     
     error = request.Body.Close()
     heckleFuncs.PrintError("ERROR: Failed to close allocation status request body.", error)
     
     error = json.Unmarshal(someBytes, &allocationNumber)
     heckleFuncs.PrintError("ERROR: Unable to unmarshal allocation number for status request.", error)
     
     currentRequestsLock.Lock()
     for key, value := range currentRequests {
          if allocationNumber == value.AllocationNumber {
               if value.User == username || admin{
                    sm := &heckleTypes.StatusMessage{value.Status, value.LastActivity, value.Info}
                    allocationStatus[key] = sm
                    value.Info = []heckleTypes.InfoMsg{}
               } else {
                    heckleFuncs.PrintError("ERROR: Cannot request status of allocations that do not beling to you.", os.NewError("Access Denied"))
                    currentRequestsLock.Unlock()
                    return
               }
          }
     }
     currentRequestsLock.Unlock()
     
     jsonStat, error := json.Marshal(allocationStatus)
     heckleFuncs.PrintError("ERROR: Unable to marshal allocation status response.", error)
     
     _, error = writer.Write(jsonStat)
     heckleFuncs.PrintError("ERROR: Unable to write allocation status response.", error)
}

func freeAllocation(writer http.ResponseWriter, request *http.Request) {
     //This function allows a user, if it owns the allocation, to free an allocation
     //number and all associated nodes.  It resets the resource map and current
     //requests map.
     rs := flunky.NewBuildServer(cfgOptions["powerServer"], false, "heckle", cfgOptions["heckle"])
     allocationNumber := uint64(0)
     request.ProtoMinor = 0
     
     username, authed, admin := heckleFuncs.Authenticate(request.Header.Get("Authorization"), cfgOptions["userDatabasePath"])
     
     if !authed {
          heckleFuncs.PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleFuncs.PrintError("ERROR: Unable to read all from allocation status POST.", error)
     
     error = request.Body.Close()
     heckleFuncs.PrintError("ERROR: Failed to close free allocation request body.", error)
     
     error = json.Unmarshal(someBytes, &allocationNumber)
     heckleFuncs.PrintError("ERROR: Unable to unmarshal allocation number for freeing.", error)
     
     powerDown := []string{}
     
     currentRequestsLock.Lock()
     resourcesLock.Lock()
     for key, value := range resources {
          if allocationNumber == value.AllocationNumber {
               if username == value.Owner || admin {
                    value.Reset()
                    powerDown = append(powerDown, key)
                    currentRequests[key] = nil, false
               } else {
                    heckleFuncs.PrintError("ERROR: Cannot free allocations that do not belong to you.", os.NewError("Access Denied"))
                    currentRequestsLock.Unlock()
                    resourcesLock.Unlock()
                    return
               }
          }
     }
     currentRequestsLock.Unlock()
     resourcesLock.Unlock()
     
     js, _ := json.Marshal(powerDown)
     buf := bytes.NewBufferString(string(js))
     _, err := rs.Post("/off", buf)
     heckleFuncs.PrintError("ERROR: Failed to post for reboot of nodes in free allocation number.", err)
     
     updateDatabase()
}

func increaseTime(writer http.ResponseWriter, request *http.Request) {
     //This function allows a user, if it owns the allocation, to free an allocation
     //number and all associated nodes.  It resets the resource map and current
     //requests map.
     timeIncrease := int64(0)
     request.ProtoMinor = 0
     
     username, authed, _ := heckleFuncs.Authenticate(request.Header.Get("Authorization"), cfgOptions["userDatabasePath"])
     
     if !authed {
          heckleFuncs.PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleFuncs.PrintError("ERROR: Unable to read all from increase time POST.", error)
     
     error = request.Body.Close()
     heckleFuncs.PrintError("ERROR: Failed to close free increase time request body.", error)
     
     error = json.Unmarshal(someBytes, &timeIncrease)
     heckleFuncs.PrintError("ERROR: Unable to unmarshal time increase in related handler func.", error)
     
     resourcesLock.Lock()
     for _, value := range resources {
          if value.Owner == username {
               value.AllocationEndTime = value.AllocationEndTime + timeIncrease
          }
     }
     resourcesLock.Unlock()
     
     updateDatabase()
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
          rs := flunky.NewBuildServer(cfgOptions["powerServer"], false, "heckle", cfgOptions["heckle"])
          
          js, _ := json.Marshal(powerDown)
          buf := bytes.NewBufferString(string(js))
          _, err := rs.Post("/off", buf)
          heckleFuncs.PrintError("ERROR: Failed to post for reboot of nodes in allocation time outs.", err)
          
          updateDatabase()
     }
}

func freeNode(writer http.ResponseWriter, request *http.Request) {
     //This will free a requested node if the user is the owner of the node.  It removes
     //the node from current resources if it exists and also resets it in resources map.
     rs := flunky.NewBuildServer(cfgOptions["powerServer"], false, "heckle", cfgOptions["heckle"])
     var node string
     request.ProtoMinor = 0
     
     username, authed, _ := heckleFuncs.Authenticate(request.Header.Get("Authorization"), cfgOptions["userDatabasePath"])
     
     if !authed {
          heckleFuncs.PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleFuncs.PrintError("ERROR: Unable to read all from allocation status POST.", error)
     
     error = request.Body.Close()
     heckleFuncs.PrintError("ERROR: Failed to close free node request body.", error)
     
     error = json.Unmarshal(someBytes, &node)
     heckleFuncs.PrintError("ERROR: Unable to unmarshal node to be unallocated.", error)
     
     currentRequestsLock.Lock()
     resourcesLock.Lock()
     
     if resources[node].Owner != username {
          heckleFuncs.PrintError("ERROR: Access denied, cannot free nodes that do not belong to you.", os.NewError("Access Denied"))
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
     _, err := rs.Post("/off", buf)
     heckleFuncs.PrintError("ERROR: Failed to post for reboot of nodes in free node.", err)
     
     updateDatabase()
}

func listenAndServeWrapper() {
     //This branches off another thread to loop through listening and serving http requests.
     error := http.ListenAndServe(":" + cfgOptions["hecklePort"], nil)
     heckleFuncs.PrintError("ERROR: Failed to listen on http socket.", error)
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
     resourcesLock.Lock()
     resources[node].Broken()
     resourcesLock.Unlock()
     
     rs := flunky.NewBuildServer(cfgOptions["powerServer"], false, "heckle", cfgOptions["heckle"])
     
     js, _ := json.Marshal([]string{node})
     buf := bytes.NewBufferString(string(js))
     _, err := rs.Post("/off", buf)
     heckleFuncs.PrintError("ERROR: Failed to post for reboot of nodes in free node.", err)
     
     //pass node off to diagnosing process
     updateDatabase()
}

func interpretPollMessages() {    
     for i := range pollingToHeckleChan {
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
     rs := flunky.NewBuildServer(cfgOptions["powerServer"], false, "heckle", cfgOptions["heckle"])
     request.ProtoMinor = 0
     
     _, authed, admin := heckleFuncs.Authenticate(request.Header.Get("Authorization"), cfgOptions["userDatabasePath"])
     
     if !authed {
          heckleFuncs.PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if admin {
          heckleFuncs.PrintError("ERROR: No access to admin command.", os.NewError("Access Denied"))
          return
     }
     
     someBytes, error := ioutil.ReadAll(request.Body)
     heckleFuncs.PrintError("ERROR: Unable to read all from outlet status POST.", error)
     
     error = request.Body.Close()
     heckleFuncs.PrintError("ERROR: Failed to close outlet status request body.", error)

     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = rs.Post("/status", buf)
     heckleFuncs.PrintError("ERROR: Failed to post for status of outlets to radixPower.go.", error)

     _, error = writer.Write(someBytes)
     heckleFuncs.PrintError("ERROR: Unable to write outlet status response in heckle.", error)
}

func main() {
     http.HandleFunc("/list", allocateList)
     http.HandleFunc("/number", allocateNumber)
     http.HandleFunc("/status", status)
     http.HandleFunc("/freeAllocation", freeAllocation)
     http.HandleFunc("/freeNode", freeNode)
     http.HandleFunc("/increaseTime", increaseTime)
     http.HandleFunc("/outletStatus", outletStatus)
     
     go allocate()
     go polling()
     go interpretPollMessages()
     go listenAndServeWrapper()
     
     for {
          runtime.Gosched()
          allocationTimeouts()
          freeCurrentRequests()
          //fmt.Fprintf(os.Stdout, "Go routines, %d.\n", runtime.Goroutines())
     }
}