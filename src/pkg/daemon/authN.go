package daemon

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"json"
	"os"
	"strings"
	"sync"
	"syscall"
)

func PrintError(errorMsg string, error os.Error) {
	//This function prints the error passed if error is not nil.
	if error != nil {
		fmt.Fprintf(os.Stderr, "%s\n", errorMsg)
	}
}

type UserNode struct {
	Password string
	Admin    bool
}

type Authinfo struct {
	path    string
	Users   map[string]UserNode
	lock    sync.RWMutex
	dbstamp int64
}

func NewAuthInfo(path string) *Authinfo {
	auth := new(Authinfo)
	auth.path = path
	auth.Users = make(map[string]UserNode, 20)
	return auth
}

func (auth *Authinfo) Load() (err os.Error) {
	authFile, err := os.Open(auth.path)
	emsg := fmt.Sprintf("ERROR: Unable to open %s for reading.", auth.path)
	PrintError(emsg, err)

	intError := syscall.Flock(authFile.Fd(), 2) //2 is exclusive lock
	if intError != 0 {
		emsg = fmt.Sprintf("ERROR: Unable to lock %s for reading.", auth.path)
		PrintError(emsg, os.NewError("Flock Syscall Failed"))
	}

	someBytes, err := ioutil.ReadAll(authFile)
	PrintError("ERROR: Unable to read from file UserDatabase.", err)

	intError = syscall.Flock(authFile.Fd(), 8) //8 is unlock
	if intError != 0 {
		PrintError("ERROR: Unable to unlock UserDatabase for reading.", os.NewError("Flock Syscall Failed"))
	}
	fi, err := authFile.Stat()

	PrintError("ERROR: Failed to stat file", err)

	err = authFile.Close()

	auth.lock.Lock()
	defer auth.lock.Unlock()

	err = json.Unmarshal(someBytes, &auth.Users)
	PrintError("ERROR: Failed to unmarshal data read from UserDatabase file.", err)
	auth.dbstamp = fi.Mtime_ns
	return
}

func (auth *Authinfo) HTTPAuthenticate(header string) (user string, valid bool, admin bool) {
	tmpAuthArray := strings.Split(header, " ")

	authValues, error := base64.StdEncoding.DecodeString(tmpAuthArray[1])
	PrintError("ERROR: Failed to decode encoded auth settings in http request.", error)

	authValuesArray := strings.Split(string(authValues), ":")
	user = authValuesArray[0]
	password := authValuesArray[1]
	valid, admin = auth.Authenticate(user, password)
	return
}

func (auth *Authinfo) Authenticate(user string, password string) (valid bool, admin bool) {
	auth.lock.RLock()
	defer auth.lock.RUnlock()
	_, ok := auth.Users[user]

	if !ok {
		return false, false
	}

	valid = (password == auth.Users[user].Password)
	admin = auth.Users[user].Admin
	return
}

func (auth *Authinfo) NewUser(user string, password string, admin bool) {
	auth.lock.Lock()
	defer auth.lock.Unlock()
	return
}

func (auth *Authinfo) DelUser(user string) (err os.Error) {
	auth.lock.Lock()
	defer auth.lock.Unlock()
	return
}

func (auth *Authinfo) Store() (err os.Error) {

	authFile, err := os.Open(auth.path)
	PrintError(fmt.Sprintf("ERROR: Unable to open %s for reading.", auth.path), err)

	intError := syscall.Flock(authFile.Fd(), 2) //2 is exclusive lock
	if intError != 0 {
		PrintError(fmt.Sprintf("ERROR: Unable to lock %s for writing.", auth.path), os.NewError("Flock Syscall Failed"))
	}

	auth.lock.RLock()
	defer auth.lock.RUnlock()
	data, err := json.Marshal(auth.Users)

	_, err = authFile.Write(data)
	PrintError("ERROR: Unable to write to file .", err)

	intError = syscall.Flock(authFile.Fd(), 8) //8 is unlock
	if intError != 0 {
		PrintError("ERROR: Unable to unlock UserDatabase for reading.", os.NewError("Flock Syscall Failed"))
	}

	fi, err := authFile.Stat()
	auth.dbstamp = fi.Mtime_ns
	authFile.Close()
	return
}
