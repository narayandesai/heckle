package main

import (
    "flag"
    "json"
    "os"
    "bytes"
    "fmt"
    "time"
    "strconv"
    fnet "flunky/net"
    iface "flunky/interfaces"
    daemon "flunky/daemon"
)

var help, status                        bool
var server, image, fileDir              string
var allocationList                      []string
var numNodes, timeIncrease              int
var allocationNumber, freeAlloc         uint64
var bs                                  *fnet.BuildServer
var testHeckleD                         *daemon.Daemon

func init() {
     flag.BoolVar(&help, "h", false, "Print usage of command.")
     flag.BoolVar(&status, "s", false, "Print status of used nodes.")
     flag.IntVar(&numNodes, "n", 0, "Request an arbitrary number of nodes.")
     flag.IntVar(&timeIncrease, "t", 0, "Increase current allocation by this many hours.")
     flag.Uint64Var(&freeAlloc, "f", 0, "Free a reserved allocation number preemptively.")
     flag.StringVar(&image, "i", "ubuntu-maverick-amd64", "Image to be loaded on to the nodes.")
     flag.StringVar(&fileDir, "F", "../../../etc/TestHeckle/", "Directory where client files can be found.")
     
     flag.Parse()
     
     testHeckleD = daemon.New("TestHeckle", fileDir)
     testHeckleD.DaemonLog.Log("Parsed command line arguements and set up logging.")
     
     allocationNumber = uint64(0)
     allocationList = flag.Args()
}

func usage() {
     testHeckleD.DaemonLog.Log("Printing usage.")
     fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
     flag.PrintDefaults()
}

func requestNumber() (tmpAllocationNumber uint64) {
     testHeckleD.DaemonLog.Log("Creating iface.Nummsg.")
     nm := iface.Nummsg{numNodes, image, 300}
     
     testHeckleD.DaemonLog.Log("Attempting to marshal iface.Nummsg.")
     someBytes, error := json.Marshal(nm)
     testHeckleD.DaemonLog.LogError("Failed to marshal nummsg in requestNumber function.", error)
     
     testHeckleD.DaemonLog.Log("Creating a buffer type of the marshaled data.")
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/number", buf)
     testHeckleD.DaemonLog.LogError("Failed to post the request for number of nodes to heckle.", error)
     
     testHeckleD.DaemonLog.Log("Attempting to unmarshal allocation number.")
     error = json.Unmarshal(someBytes, &tmpAllocationNumber)
     testHeckleD.DaemonLog.LogError("Failed to unmarshal allocation number from http response in request number.", error)
     
     testHeckleD.DaemonLog.Log("Allocation number is " + strconv.Uitoa64(tmpAllocationNumber) + ".")
     
     return
}

func requestList() (tmpAllocationNumber uint64) {
     testHeckleD.DaemonLog.Log("Creating iface.Listmsg.")
     nm := iface.Listmsg{allocationList, image, 300}
     
     testHeckleD.DaemonLog.Log("Attempting to marshal iface.Listmsg.")
     someBytes, error := json.Marshal(nm)
     testHeckleD.DaemonLog.LogError("Failed to marshal nummsg in requestList function.", error)
     
     testHeckleD.DaemonLog.Log("Creating a buffer type of the marshaled data.")
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/list", buf)
     testHeckleD.DaemonLog.LogError("Failed to post the request for list of nodes to heckle.", error)
     
     testHeckleD.DaemonLog.Log("Attempting to unmarshal allocation number.")
     error = json.Unmarshal(someBytes, &tmpAllocationNumber)
     testHeckleD.DaemonLog.LogError("Failed to unmarshal allocation number from http response in request list.", error)
     
     testHeckleD.DaemonLog.Log("Allocation number is " + strconv.Uitoa64(tmpAllocationNumber) + ".")
     
     return
}

func requestTimeIncrease() {
     testHeckleD.DaemonLog.Log("Turning time increase into seconds.")
     tmpTimeMsg := int64(timeIncrease * 3600)

     testHeckleD.DaemonLog.Log("Attempting to marshal time increase in seconds.")
     someBytes, error := json.Marshal(tmpTimeMsg)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to marshal time increase in requestTimeIncrease function.", error)
     
     testHeckleD.DaemonLog.Log("Creating a buffer type and posting time increasein seconds to heckle.")
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/increaseTime", buf)
     testHeckleD.DaemonLog.LogError("Failed to post the request for time increase to heckle.", error)
     
     return
}

func pollForStatus() {
     testHeckleD.DaemonLog.Log("Creating map of status messages for polling.")
     statMap := make(map[string]*iface.StatusMessage)
     for {
          testHeckleD.DaemonLog.Log("Sleeping for 10 seconds.")
          time.Sleep(10000000000)
          testHeckleD.DaemonLog.Log("Marshaling allocation number.")
          someBytes, error := json.Marshal(allocationNumber)
          testHeckleD.DaemonLog.LogError("Failed to marshal allocation number for status poll.", error)
     
          testHeckleD.DaemonLog.Log("Creating a buffer type of the marshaled data and posting for status.")
          buf := bytes.NewBufferString(string(someBytes))
          someBytes, error = bs.Post("/status", buf)
          testHeckleD.DaemonLog.LogError("Failed to post for status of nodes to heckle.", error)

          testHeckleD.DaemonLog.Log("Unmarshaling response.")
          error = json.Unmarshal(someBytes, &statMap)
          testHeckleD.DaemonLog.LogError("Failed to unmarshal status info from http response in status polling.", error)
          
          testHeckleD.DaemonLog.Log("Printing out any new status messages.")
          done := false
          for key, value := range statMap {
               if len(value.Info) != 0 {
                    done = true
                    for i := range value.Info {
                         testHeckleD.DaemonLog.Log(fmt.Sprintf("NODE: %s\tSTATUS: %s\tLAST ACTIVITY: %d:%d:%d\tMESSAGE: %d:%d:%d : %s : %s\n", key, value.Status, time.SecondsToLocalTime(value.LastActivity).Hour, time.SecondsToLocalTime(value.LastActivity).Minute, time.SecondsToLocalTime(value.LastActivity).Second, time.SecondsToLocalTime(value.Info[i].Time).Hour, time.SecondsToLocalTime(value.Info[i].Time).Minute, time.SecondsToLocalTime(value.Info[i].Time).Second, value.Info[i].Message, value.Info[i].MsgType))
                    }
                    done = done && (value.Status == "Ready")
                    /*if value.Status == "Cancel" {
                         statMap[key] = nil, false
                    }*/
               }
          }
          fmt.Fprintf(os.Stdout, "\n")
          
          if done {
               testHeckleD.DaemonLog.Log(fmt.Sprintf("Done allocating nodes.  Your allocation number is %d.  Please report failures to system admin.\n", allocationNumber))
               os.Exit(0)
          }
     }
}

func freeAllocation() {
     testHeckleD.DaemonLog.Log("Marshaling allocation number to free.")
     someBytes, error := json.Marshal(freeAlloc)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to marshal allocation number for status poll.", error)
     testHeckleD.DaemonLog.Log("Creating buffer type and posting to free allocation.")
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/freeAllocation", buf)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to post for status of nodes to heckle.", error)
}

func nodeStatus() {
     testHeckleD.DaemonLog.Log("Posting to heckle for node status.")
     buf := bytes.NewBufferString("")
     someBytes, error := bs.Post("/nodeStatus", buf)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to post the request for node status to heckle.", error)
     
     fmt.Fprintf(os.Stdout, "%s", string(someBytes))
     
     return
}

func main() {  
     testHeckleD.DaemonLog.Log("Checking for flag miss matches (-n and a list).")
     if len(allocationList) != 0 && numNodes != 0 {
          testHeckleD.DaemonLog.LogError("ERROR: Cannot use node list, and number of nodes option at the same time.", os.NewError("Flag contradiction"))
          os.Exit(1)
     } else if (len(allocationList) == 0 && numNodes == 0 && timeIncrease == 0 && freeAlloc == 0 && !status) || help {
          usage()
          os.Exit(0)
     }
     
     testHeckleD.DaemonLog.Log("Setting up flunky build server.")
     bs = fnet.NewBuildServer("http://" + testHeckleD.Cfg.Data["username"] + ":" + testHeckleD.Cfg.Data["password"] + "@" + testHeckleD.Cfg.Data["heckleServer"], false)
     
     if status {
          nodeStatus()
          os.Exit(0)
     }
     
     if timeIncrease != 0 {
          requestTimeIncrease()
     }
     
     if freeAlloc != 0 {
          freeAllocation()
     }
     
     if numNodes != 0 {
          allocationNumber = requestNumber()
          pollForStatus()
     } else if len(allocationList) != 0 {
          allocationNumber = requestList()
          pollForStatus()
     } 
}