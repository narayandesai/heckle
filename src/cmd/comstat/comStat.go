package main

import (
     "flag"
     "fmt"
     "os"
     "json"
     inet "flunky/net"
     daemon "flunky/daemon"      

)

var Usage = func(){
    fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
    flag.PrintDefaults()
    os.Exit(0)
}

var dump bool
var help bool
var fileDir string
var comStatDaemon *daemon.Daemon

func init(){
   fileDir = "../../../etc/ComStat/"
   comStatDaemon = daemon.New("ComStat", fileDir)
   flag.BoolVar(&dump, "d", false, "Print state information")
   flag.BoolVar(&help, "h", false, "Print help message")
}

func GetDump(daemonName string) {
    data := make(map[string]string) 
    server := comStatDaemon.Cfg.Data[daemonName]
    
    query := inet.NewBuildServer(server, true)
    
    resp, err:= query.Get("dump")
    if err != nil {
       comStatDaemon.DaemonLog.LogError("Cannot read data from daemonName", err)
     }else{
        err := json.Unmarshal(resp, &data)
       if err != nil {
	   comStatDaemon.DaemonLog.LogError("Cannot unmarshall data for daemonLog", err)
	   }else{
	     fmt.Println(data)
    }
    return
}
}

func main() {
   flag.Parse()

   if(help){
     Usage()
     os.Exit(0)
  }
  if dump {
     daemonName := flag.Args()
     for _, name := range(daemonName){   
         GetDump(name)
    }
  }

}
