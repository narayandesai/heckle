package main

import(
	"fmt"
	"flag"
	"os"
	"json"
	"bytes"
	"strconv"
	fnet "flunky/net"
	cli "flunky/client"
)

var help bool

func Usage() {
     fmt.Fprintf(os.Stderr, "Usage of %s: hfree allocNum\n", os.Args[0])
     flag.PrintDefaults()
}


func init(){
      flag.BoolVar(&help, "h", false, "Print usage of command.")
}

func printError(err os.Error){
   if err != nil{
       fmt.Println(err)
   }

}

func freeAlloc(alloc int64)(ret string, err os.Error){
   bs := new(fnet.BuildServer)
   allocfree, err := cli.NewClient()
   printError(err)
   bs, err = allocfree.SetupClient("heckle")
   printError(err)
   
   msg, err := json.Marshal(alloc)
   printError(err)
   buf := bytes.NewBufferString(string(msg))
   ans, err := bs.Post("/freeAllocation", buf)
   printError( err)
   ret = string(ans)
   return
}

func main(){
    flag.Parse()

    if help{
        Usage()
	os.Exit(0)
    } 

    if len(flag.Args())<= 0{
	Usage()
        os.Exit(0)
   }

    allocations := flag.Args()
    alloc, _ := strconv.Atoi64(allocations[0])
    ret, err := freeAlloc(alloc) 
    printError(err)
    if err != nil{
        os.Exit(1)
    }
    if ret != "not present" {
        fmt.Println(fmt.Sprintf("Freed allocation #%d.", alloc))
    }else{
        fmt.Println(fmt.Sprintf("Unable to free allocation #%d. Allocation does not exsist.", alloc))
	os.Exit(1)
    }    

}
