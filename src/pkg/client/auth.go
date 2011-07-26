package client

import (
	"io/ioutil"
	"json"
	"os"
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
