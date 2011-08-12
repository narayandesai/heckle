package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"io/ioutil"
	fclient "flunky/client"
)

var help bool
var fileDir string
var error bool
var log bool

func init() {
	flag.BoolVar(&help, "h", false, "Print help message")
	flag.BoolVar(&error, "e", false, "Print error logs")
	flag.BoolVar(&log, "l", false, "Print log file for specified host")
}

func prettyPrint(data string) {
     prettyData := strings.Split(data, ",")
     for _, pretty := range(prettyData){
        fmt.Println(pretty)
     }
     return

}

func logs(name string) string {
   var errorLog string
   logName := "/var/log/" + name + ".log"
   _,err :=  os.Stat(logName)
   if err != nil{
       fclient.PrintError(err.String(), err)
       os.Exit(1)
   }
   file, err := ioutil.ReadFile(logName)
   if log{
      return string(file)
   }
   if error {
      contents := strings.Split(string(file), "\n")
      for _, data := range(contents){
      	   dex := strings.Index(data, "ERROR")
       	   if dex > 0{
              errorLog = errorLog + data + "\n"
       	    }
       }
   return(errorLog)
   }
   return ""
}

func main() {
	flag.Parse()

	if help {
	        fmt.Println(fmt.Sprintf("Command syntax -- %s daemon1 daemon2 ...daemonN", os.Args[0]))
		fclient.Usage()
		os.Exit(0)
	}

	if len(os.Args) <= 1 {
		fclient.PrintError("No arguments provided", os.NewError("no args"))
		fmt.Println(fmt.Sprintf("Command syntax -- %s daemon1 daemon2 ...daemonN", os.Args[0]))
		fclient.Usage()
		os.Exit(1)
	} else {
	        if log{
		   for _, name := range(flag.Args()){
		       logging := logs(name)
		       fmt.Println(logging)
                   }
		   os.Exit(0)
                 }
   
                 if error {
		    for _, name := range(flag.Args()){
		        errorLogging := logs(name)
			if len(errorLogging) <=0{
	                     fmt.Println(fmt.Sprintf("No errors exsist in the logs for %s", name))
                        }else{
				fmt.Println(errorLogging)
                        }
                     }
		     os.Exit(0)
                  }
		    
		comm, err := fclient.NewClient()
		if err != nil {
			fclient.PrintError("Failed to setup communication", err)
			os.Exit(1)
		}

		for _, name := range flag.Args() {
			client, err := comm.SetupClient(name)
			if err != nil {
				fclient.PrintError(fmt.Sprintf("Failed to lookup component %s", name), err)
				os.Exit(1)
			}
			resp, err := client.Get("dump")
			if err != nil {
				fclient.PrintError(fmt.Sprintf("Failed to contact component %s", name), err)
			        os.Exit(1)
			}
			//prettyPrint(string(resp))
			fmt.Println(string(resp))
		}
	}
}
