package client

import (
	"encoding/json"
	fnet "flunky/net"
	"io/ioutil"
	"os"
)

type Auth struct {
	User     string
	Password string
}

func GetUserAuth() (user string, password string, err error) {
	homedir := os.Getenv("HOME")

	var authdata Auth

	rawdata, err := ioutil.ReadFile(homedir + "/.hauth")
	if err != nil {
		return
	}
	json.Unmarshal(rawdata, &authdata)
	user = authdata.User
	password = authdata.Password
	return
}

func NewClient() (comm fnet.Communication, err error) {
	user, password, err := GetUserAuth()
	if err != nil {
		return
	}

	comm, err = fnet.NewCommunication("/etc/heckle/components.conf", user, password)
	return
}
