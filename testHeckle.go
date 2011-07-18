package main

import (
     "flag"
     "json"
     "os"
     "bytes"
     "fmt"
     "io/ioutil"
     "time"
     "./flunky"
     "./heckleTypes"
     "./heckleFuncs"
     )

var cfgOptions                          map[string]string
var help                                bool
var server, image                       string
var allocationList                      []string
var numNodes, timeIncrease              int
var allocationNumber, freeAlloc         uint64
var bs                                  *flunky.BuildServer

func init() {
     flag.BoolVar(&help, "h", false, "Print usage of command.")
     flag.IntVar(&numNodes, "n", 0, "Request an arbitrary number of nodes.")
     flag.IntVar(&timeIncrease, "t", 0, "Increase current allocation by this many hours.")
     flag.Uint64Var(&freeAlloc, "f", 0, "Free a reserved allocation number preemptively.")
     flag.StringVar(&image, "i", "ubuntu-maverick-amd64", "Image to be loaded on to the nodes.")
     
     flag.Parse()
     
     cfgOptions = make(map[string]string)
     allocationNumber = uint64(0)
     allocationList = flag.Args()
     
     cfgFile, error := os.Open("testHeckle.cfg")
     heckleFuncs.PrintError("ERROR: Unable to open testHeckle.cfg for reading.", error)
     
     someBytes, error := ioutil.ReadAll(cfgFile)
     heckleFuncs.PrintError("ERROR: Unable to read from file testHeckle.cfg", error)
     
     error = cfgFile.Close()
     heckleFuncs.PrintError("ERROR: Failed to close testHeckle.cfg.", error)
     
     error = json.Unmarshal(someBytes, &cfgOptions)
     heckleFuncs.PrintError("ERROR: Failed to unmarshal data read from testHeckle cfg file.", error)
}

func usage() {
     fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
     flag.PrintDefaults()
}

func requestNumber() (tmpAllocationNumber uint64) {
     nm := heckleTypes.Nummsg{numNodes, image, 300}
     
     someBytes, error := json.Marshal(nm)
     heckleFuncs.PrintError("ERROR: Failed to marshal nummsg in requestNumber function.", error)
     
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/number", buf)
     heckleFuncs.PrintError("ERROR: Failed to post the request for number of nodes to heckle.", error)
     
     error = json.Unmarshal(someBytes, &tmpAllocationNumber)
     heckleFuncs.PrintError("ERROR: Failed to unmarshal allocation number from http response in request number.", error)
     
     return
}

func requestList() (tmpAllocationNumber uint64) {
     nm := heckleTypes.Listmsg{allocationList, image, 300}
     
     someBytes, error := json.Marshal(nm)
     heckleFuncs.PrintError("ERROR: Failed to marshal nummsg in requestList function.", error)
     
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/list", buf)
     heckleFuncs.PrintError("ERROR: Failed to post the request for list of nodes to heckle.", error)
     
     error = json.Unmarshal(someBytes, &tmpAllocationNumber)
     heckleFuncs.PrintError("ERROR: Failed to unmarshal allocation number from http response in request list.", error)
     
     return
}

func requestTimeIncrease() {
     tmpTimeMsg := int64(timeIncrease * 3600)

     someBytes, error := json.Marshal(tmpTimeMsg)
     heckleFuncs.PrintError("ERROR: Failed to marshal time increase in requestTimeIncrease function.", error)
     
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/increaseTime", buf)
     heckleFuncs.PrintError("ERROR: Failed to post the request for time increase to heckle.", error)
     
     return
}

func pollForStatus() {
     statMap := make(map[string]*heckleTypes.StatusMessage)
     for {
          time.Sleep(10000000000)
          someBytes, error := json.Marshal(allocationNumber)
          heckleFuncs.PrintError("ERROR: Failed to marshal allocation number for status poll.", error)
     
          buf := bytes.NewBufferString(string(someBytes))
          someBytes, error = bs.Post("/status", buf)
          heckleFuncs.PrintError("ERROR: Failed to post for status of nodes to heckle.", error)

          error = json.Unmarshal(someBytes, &statMap)
          heckleFuncs.PrintError("ERROR: Failed to unmarshal status info from http response in status polling.", error)
          
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
     heckleFuncs.PrintError("ERROR: Failed to marshal allocation number for status poll.", error)
     
     buf := bytes.NewBufferString(string(someBytes))
     someBytes, error = bs.Post("/freeAllocation", buf)
     heckleFuncs.PrintError("ERROR: Failed to post for status of nodes to heckle.", error)
}

func main() {  
     if len(allocationList) != 0 && numNodes != 0 {
          fmt.Fprintf(os.Stderr, "ERROR: Cannot use node list, and number of nodes option at the same time.\n")
          os.Exit(1)
     } else if (len(allocationList) == 0 && numNodes == 0 && timeIncrease == 0 && freeAlloc == 0) || help {
          usage()
          os.Exit(0)
     }
     
     bs = flunky.NewBuildServer(cfgOptions["heckleServer"], false, cfgOptions["Username"], cfgOptions["Password"])
     
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