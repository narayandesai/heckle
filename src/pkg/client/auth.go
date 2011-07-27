package client

import (
	"io/ioutil"
	"json"
	"os"
	fnet "flunky/net"
)

type Auth struct {
	User string
	Password string
}

func GetUserAuth() (user string, password string, err os.Error) {
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

func NewClient() (comm fnet.Communication, err os.Error) {
	user, password, err := GetUserAuth()
	if err != nil {
		return
	}

	comm, err = fnet.NewCommunication("/etc/heckle/components.conf", user, password)
	return
}