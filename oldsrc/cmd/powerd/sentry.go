package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
)

type Sentry struct {
	Name     string
	Address  string
	User     string
	Password string
	Outlets  map[string]string
}

func (controller Sentry) doCmd(cmd string) (data string, err error) {
	//powerDaemon.DaemonLog.LogDebug(fmt.Sprintf("Function dialServer: -- command %s", cmd))
	buf := make([]byte, 82920)
	response := bytes.NewBuffer(make([]byte, 82920))
	cmdList := []string{controller.User, controller.Password, cmd}

	//connect to telnet server. Default is accept everything.
	address := fmt.Sprintf("%s:23", controller.Address)
	conn, err := net.Dial("tcp", address)
	conn.SetReadTimeout(100000000)
	conn.SetWriteTimeout(100000000)
	if err != nil {
		err = errors.New(fmt.Sprintf("Cannot connect to %s", address))
		return
	}
	ret := []byte{255, 251, 253}
	for i := 0; i < 5; i++ {
		_, err = conn.Write(ret)
		if err != nil {
			err = errors.New("Cannot write login info to socket")
			return data, err
		}
		conn.Read(buf)
	}

	for _, cmd := range cmdList {
		for {
			mybuf := make([]byte, 1024)
			n, err := conn.Read(mybuf)
			if err != nil {
				err = errors.New("Cannot read from socket for terminal")
				return data, err
			}
			m := strings.Index(string(mybuf[:n]), ":")
			if m > 0 {
				_, err = conn.Write([]byte(cmd + "\n"))
				if err != nil {
					err = errors.New("Cannot write to socket")
					return data, err
				}
				break
			}
		}
	}

	//See if the command is successful and then read the rest of the output.
	//keep in mind that it wont always be successful and therefore not read properly
	for {
		n, _ := conn.Read(buf)
		m := strings.Index(string(buf[:n]), "successful")
		if m > 0 {
			//powerDaemon.DaemonLog.LogDebug(fmt.Sprintf("Function dialServer: -- finalbuf\n %s", finalBuf.String()))
			break
		}
		response.Write(buf[:n])

	}

	conn.Write([]byte("quit\n"))
	err = conn.Close()
	if err != nil {
		err = errors.New("Cannot close socket")
		return data, err
	}

	if len(response.String()) <= 0 {
		err = errors.New("Empty return")
		return data, err
	}

	//Strip off the headers
	final := response.String()
	dex := strings.Index(final, "State")
	newFinal := final[dex:]
	dex = strings.Index(newFinal, "\n")
	data = strings.TrimSpace(newFinal[dex:])
	return data, err
}

func (controller Sentry) retryDoCmd(cmd string, retries int) (data string, err error) {
	for i := 0; i < retries; i++ {
		data, err = controller.doCmd(cmd)
		if err != nil {
			continue
		}
		if i > 0 {
			fmt.Println("Needed to retry")
		}
		return data, err
	}
	return data, err
}

func (controller Sentry) Status() (status map[string]PortStatus, err error) {
	status = make(map[string]PortStatus, 8)
	data, err := controller.retryDoCmd("status", 3)
	if err != nil {
		return
	}

	for _, item := range strings.Split(data, "\n") {
		fields := strings.Fields(item)
		if len(fields) == 0 {
			fmt.Println(item)
		}
		port := fields[0]
		for ohost, oport := range controller.Outlets {
			if port == oport {
				status[ohost] = PortStatus{fields[2] == "On", fields[2] != fields[3]}
			}
		}
	}

	return
}

func (controller Sentry) On(hostname string) (err error) {
	outlet, found := controller.Outlets[hostname]
	if !found {
		err = errors.New(fmt.Sprintf("No outlet for host %s", hostname))
		return
	}
	_, err = controller.retryDoCmd(fmt.Sprintf("on %s", outlet), 3)
	return
}

func (controller Sentry) Off(hostname string) (err error) {
	outlet, found := controller.Outlets[hostname]
	if !found {
		err = errors.New(fmt.Sprintf("No outlet for host %s", hostname))
		return
	}
	_, err = controller.retryDoCmd(fmt.Sprintf("off %s", outlet), 3)
	return
}

func (controller Sentry) Reboot(hostname string) (err error) {
	outlet, found := controller.Outlets[hostname]
	if !found {
		err = errors.New(fmt.Sprintf("No outlet for host %s", hostname))
		return
	}
	_, err = controller.retryDoCmd(fmt.Sprintf("reboot %s", outlet), 3)
	return
}

func (controller Sentry) Controls(hostname string) (status bool) {
	_, status = controller.Outlets[hostname]
	return
}
