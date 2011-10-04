package main

import (
	"exec"
	/* "fmt" */
	"os"
	"strings"
	)

type Ipmi struct {
	Username string
	Password string
	Nodes map[string]string
}

func (i Ipmi) RunCmd(icmd string, nodes []string) (result string, err os.Error) {
	nodestr := strings.Join(nodes, ",")
	cmd := exec.Command("/usr/sbin/ipmipower", icmd, "-p", i.Password, "-u", i.Username, "-h", nodestr)
	/* ipmipower hangs up if stdin closes */
	_, _ = cmd.StdinPipe()

	bresult, err := cmd.Output()
	result = string(bresult)
	return
}

func (i Ipmi) Status() (map[string]PortStatus, os.Error) {
	ps := make(map[string]PortStatus, 10)
	mgts := make([]string, 20)
	
	for _, m := range i.Nodes {
		mgts = append(mgts, m)
	}

	output, _ := i.RunCmd("-s", mgts)
	lines := strings.Split(string(output), "\n")
	for index := range lines {
		fields := strings.Split(lines[index], ": ")
		if (len(fields) != 2) {
			continue
		}
		address := fields[0]
		status := fields[1]
		nstat := PortStatus{status == "on", false}
		for name, val := range i.Nodes {
			if val == address {
				ps[name] = nstat
			}
		}
	}

	return ps, nil
}

func (i Ipmi) On(node string) (err os.Error) {
	mname, ok := i.Nodes[node]
	if (!ok) {
		return os.NewError("Failed to resolve client")
	}
	ndata := []string{mname}
	_, err = i.RunCmd("--on", ndata)
	return 
}

func (i Ipmi) Off(node string) (err os.Error) {
	mname, ok := i.Nodes[node]
	if (!ok) {
		return os.NewError("Failed to resolve client")
	}
	ndata := []string{mname}
	_, err = i.RunCmd("--off", ndata)
	return
}

func (i Ipmi) Reboot(node string) (err os.Error) {
	mname, ok := i.Nodes[node]
	if (!ok) {
		return os.NewError("Failed to resolve client")
	}
	ndata := []string{mname}
	_, err = i.RunCmd("--reboot", ndata)
	return
}

func (i Ipmi) Controls(node string) (ok bool) {
	_, ok = i.Nodes[node]
	return
}

