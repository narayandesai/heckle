//Do i speak directly to the power controller on the server
//Or the power daemon that we wrote for heckle.
package main

import (
       "strings"
       "fmt"
       "net"
       "bytes"
       "flag"
)

type outletNode struct {
        Address string
        Outlet  string
}

type States struct{
     State bool
     Reboot bool
}

var outletStatus map[string]States

func returnState(info bool)(ret string){
     if info{
       ret = "on"
     }else{
       ret = "off"
    }
     return
}

func returnStatus(status string, nodes []string){

        for _, node := range nodes {
                dex := strings.Index(status, resources[node].Outlet)
                first := status[dex:]

                dex = strings.Index(first, "\n")
                second := first[:dex]

                dex = strings.Index(second, "On")
                if dex < 0 {
                        dex = strings.Index(second, "Off")
                        if dex < 0 {
                                fmt.Println("Node has no status")
                                return
                        }
                }
                third := second[dex:]
                dex = strings.Index(third, " ")

                state := strings.TrimSpace((third[:dex]))
                key := outletStatus[node]
                if state == "On"{
                    key.State = true
                }else{
                    key.State = false
                }
                outletStatus[node] = key

                reboot := strings.TrimSpace((third[dex:]))
                if reboot == "Reboot" {
                    key.Reboot = true
                    fmt.Println(fmt.Sprintf("%s has a pending reboot", node))
                }else{
                    if key.Reboot {
                        fmt.Println(fmt.Sprintf("%s's reboot complete the node is %s", node, returnState(key.State)))
                    }
  key.Reboot = false
                }
                 outletStatus[node] = key


        }
        return
}

func dialServer(cmd string) string {
        byt := make([]byte, 82920)
        //buf := bytes.NewBufferString("admn\n")
        finalBuf := bytes.NewBuffer(make([]byte, 82920))

        cmdList := []string{"admn", "admn", cmd}

        //Set up negoations to the telnet server. Default is accept everything.
        k, _ := net.Dial("tcp", "radix-pwr11:23")
        ret := []byte{255, 251, 253}
        for i := 0; i < 5; i++ {
                k.Write(ret)
                k.Read(byt)
        }

        //All three for loops just send commands to the terminal
        for _, cmd := range(cmdList){
            for {
                        n, _ := k.Read(byt)
                        m := strings.Index(string(byt[:n]), ":")
                        if m > 0 {
                           k.Write([]byte(cmd + "\n"))
                           break
                           }
               }
               if cmd == "status"{break}
        }
 //See if the command is successful and then read the rest of the output.
        for {
                n, _ := k.Read(byt)
                m := strings.Index(string(byt[:n]), "successful")
                if m > 0 {
                        break
                }
                finalBuf.Write(byt[:n])

        }

        //Strip off the headers
        final := finalBuf.String()
        dex := strings.Index(final, "State")
        newFinal := final[dex:]
        dex = strings.Index(newFinal, "\n")
        //close connection and return
        k.Close()
        return strings.TrimSpace(newFinal[dex:])
}

func main() {
     flag.Parse()
}