package main

import (
	"io/ioutil"
	"json"
	"os"
)

type PortStatus struct {
	State bool
	Reboot bool
}

type Controller interface {
	Status()  (map[string]PortStatus, os.Error)
	On(string) (os.Error)
	Off(string) (os.Error)
	Reboot(string) (os.Error)
	Controls(string) (bool)
}

type ControllerMux struct {
	Controllers []Controller
}

func NewControllerMux() (cm ControllerMux) {
	cm = ControllerMux{}
	cm.Controllers = make([]Controller, 0)
	return
}

type PortSplit struct {
	Device Controller
	Hostlist []string
}

func (mux ControllerMux) splitNodeList(nodelist []string) (split []PortSplit, err os.Error) {
	for i := range mux.Controllers {
		ctrl := mux.Controllers[i]
		pl := PortSplit{ctrl, []string{}}
		for i := range nodelist {
			found := ctrl.Controls(nodelist[i])
			if found {
				pl.Hostlist = append(pl.Hostlist, nodelist[i])
			}
		}
		if ((len(pl.Hostlist) > 0) || (len(nodelist) == 0 )){
			split = append(split, pl)
		}
	}
	foundnodes := 0
	for i := range split {
		foundnodes += len(split[i].Hostlist)
	}

	if foundnodes != len(nodelist) {
		err = os.NewError("Could not locate all nodes")
	}
	return
}

func (mux ControllerMux) Status(nodelist []string) (status map[string]PortStatus, err os.Error) {
	status = make(map[string]PortStatus, 8)
	split, err := mux.splitNodeList(nodelist)
	if (err != nil) {
		return
	}

	for i := range split {
		ctrl := split[i].Device
		var data map[string]PortStatus
		data, err = ctrl.Status()
		if (err != nil) {
			return
		}
		if (len(nodelist) == 0) {
			for key, value := range data {
				status[key] = value
			}
			continue
		} else {
			for node := range nodelist {
				nstatus, found := data[nodelist[node]]
				if found {
					status[nodelist[node]] = nstatus
				}
			}
		}
	}
	return 
}

type ControllerAction func (ctrl Controller, node string) (err os.Error)

func (mux ControllerMux) splitAction(nodelist []string, action ControllerAction) (err os.Error) {
	split, err := mux.splitNodeList(nodelist)
	if (err != nil) {
		return
	}

	for i := range split {
		ctrl := split[i].Device
		for j := range split[i].Hostlist {
			err = action(ctrl, split[i].Hostlist[j])
			if (err != nil) {
				return
			}
		}
	}
	return
}

func (mux ControllerMux) On(nodelist []string) (err os.Error) {
	err = mux.splitAction(nodelist, 
		func (ctrl Controller, node string) (err os.Error){
		porterr := ctrl.On(node)
		if (porterr != nil) {
			err = porterr
		}
		return err
	})
	return err
}

func (mux ControllerMux) Off(nodelist []string) (err os.Error) {
	err = mux.splitAction(nodelist, 
		func (ctrl Controller, node string) (err os.Error){
		porterr := ctrl.Off(node)
		if (porterr != nil) {
			err = porterr
		}
		return err
	})
	return err
}

func (mux ControllerMux) Reboot(nodelist []string) (err os.Error) {
	err = mux.splitAction(nodelist, 
		func (ctrl Controller, node string) (err os.Error){
		porterr := ctrl.Reboot(node)
		if (porterr != nil) {
			err = porterr
		}
		return err
	})
	return err
}

func (mux *ControllerMux) LoadSentryFromFile(filename string) (err os.Error) {
	data, err := ioutil.ReadFile(filename)
	if (err != nil) { 
		return
	}
	controllers := []Sentry{}
	err = json.Unmarshal(data, &controllers)
	if (err != nil) {
		return
	}
	for i := range controllers {
		mux.Controllers = append(mux.Controllers, controllers[i])
	}
	return
}

func (mux *ControllerMux) LoadIpmiFromFile(filename string) (err os.Error) {
	data, err := ioutil.ReadFile(filename)
	if (err != nil) { 
		return
	}
	controllers := []Ipmi{}
	err = json.Unmarshal(data, &controllers)
	if (err != nil) {
		return
	}
	for i := range controllers {
		mux.Controllers = append(mux.Controllers, controllers[i])
	}
	return
}