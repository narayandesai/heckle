package main

import (
    "flag"
    "json"
    "os"
    "bytes"
    "fmt"
    "time"
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
     
     flag.Parse()
     
     fileDir = "../../../etc/TestHeckle/"
     testHeckleD = daemon.New("TestHeckle", fileDir)
     
     allocationNumber = uint64(0)
     allocationList = flag.Args()
     
     
}

func usage() {
     fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
     flag.PrintDefaults()
}

func requestNumber() (tmpAllocationNumber uint64) {
     nm := iface.Nummsg{numNodes, image, 300}
     
     someBytes, error := json.Marshal(nm)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to marshal nummsg in requestNumber function.", error)
     
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/number", buf)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to post the request for number of nodes to heckle.", error)
     
     error = json.Unmarshal(someBytes, &tmpAllocationNumber)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to unmarshal allocation number from http response in request number.", error)
     
     return
}

func requestList() (tmpAllocationNumber uint64) {
     nm := iface.Listmsg{allocationList, image, 300}
     
     someBytes, error := json.Marshal(nm)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to marshal nummsg in requestList function.", error)
     
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/list", buf)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to post the request for list of nodes to heckle.", error)
     
     error = json.Unmarshal(someBytes, &tmpAllocationNumber)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to unmarshal allocation number from http response in request list.", error)
     
     return
}

func requestTimeIncrease() {
     tmpTimeMsg := int64(timeIncrease * 3600)

     someBytes, error := json.Marshal(tmpTimeMsg)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to marshal time increase in requestTimeIncrease function.", error)
     
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/increaseTime", buf)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to post the request for time increase to heckle.", error)
     
     return
}

func pollForStatus() {
     statMap := make(map[string]*iface.StatusMessage)
     for {
          time.Sleep(10000000000)
          someBytes, error := json.Marshal(allocationNumber)
          testHeckleD.DaemonLog.LogError("ERROR: Failed to marshal allocation number for status poll.", error)
     
          buf := bytes.NewBufferString(string(someBytes))
          someBytes, error = bs.Post("/status", buf)
          testHeckleD.DaemonLog.LogError("ERROR: Failed to post for status of nodes to heckle.", error)

          error = json.Unmarshal(someBytes, &statMap)
          testHeckleD.DaemonLog.LogError("ERROR: Failed to unmarshal status info from http response in status polling.", error)
          
          done := true
          for key, value := range statMap {
               fmt.Fprintf(os.Stdout, "NODE: %s\tSTATUS: %s\tLAST ACTIVITY: %d:%d:%d\n", key, value.Status, time.SecondsToLocalTime(value.LastActivity).Hour, time.SecondsToLocalTime(value.LastActivity).Minute, time.SecondsToLocalTime(value.LastActivity).Second)
               for i := range value.Info {
                    fmt.Fprintf(os.Stdout, "\t%d:%d:%d : %s : %s\n", time.SecondsToLocalTime(value.Info[i].Time).Hour, time.SecondsToLocalTime(value.Info[i].Time).Minute, time.SecondsToLocalTime(value.Info[i].Time).Second, value.Info[i].Message, value.Info[i].MsgType)
               }
               done = done && (value.Status == "Ready")
               /*if value.Status == "Cancel" {
                    statMap[key] = nil, false
               }*/
          }
          fmt.Fprintf(os.Stdout, "\n")
          
          if done {
               fmt.Fprintf(os.Stdout, "Done allocating nodes.  Your allocation number is %d.  Please report failures to system admin.\n", allocationNumber)
               os.Exit(0)
          }
     }
}

func freeAllocation() {
     someBytes, error := json.Marshal(freeAlloc)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to marshal allocation number for status poll.", error)
     
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/freeAllocation", buf)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to post for status of nodes to heckle.", error)
}

func nodeStatus() {
     buf := bytes.NewBufferString("")
     someBytes, error := bs.Post("/nodeStatus", buf)
     testHeckleD.DaemonLog.LogError("ERROR: Failed to post the request for node status to heckle.", error)
     
     fmt.Fprintf(os.Stdout, "%s", string(someBytes))
     
     return
}

func main() {  
     if len(allocationList) != 0 && numNodes != 0 {
          fmt.Fprintf(os.Stderr, "ERROR: Cannot use node list, and number of nodes option at the same time.\n")
          os.Exit(1)
     } else if (len(allocationList) == 0 && numNodes == 0 && timeIncrease == 0 && freeAlloc == 0 && !status) || help {
          usage()
          os.Exit(0)
     }
     
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