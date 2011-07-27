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
	"http"
)

type UserNode struct {
	Password string
	Admin    bool
}

type Authinfo struct {
	path      string
	Users     map[string]UserNode
	lock      sync.RWMutex
	dbstamp   int64
	daemonLog *DaemonLogger
}

func NewAuthInfo(path string, daemonLog *DaemonLogger) *Authinfo {
	auth := new(Authinfo)
	auth.path = path
	auth.Users = make(map[string]UserNode, 20)
	auth.daemonLog = daemonLog
	auth.Load()
	return auth
}

func (auth *Authinfo) Load() (err os.Error) {
	if auth.path == "" {
		auth.daemonLog.LogError("No auth file specified.", os.NewError(" Auth file does not exsist"))
		return
	}
	authFile, err := os.Open(auth.path)
	emsg := fmt.Sprintf("ERROR: Unable to open %s for reading.", auth.path)
	auth.daemonLog.LogError(emsg, err)
	if err != nil {
		os.Exit(1)
	}

	intError := syscall.Flock(authFile.Fd(), 2) //2 is exclusive lock
	if intError != 0 {
		emsg = fmt.Sprintf("ERROR: Unable to lock %s for reading.", auth.path)
		auth.daemonLog.LogError(emsg, os.NewError("Flock Syscall Failed"))
	}

	someBytes, err := ioutil.ReadAll(authFile)
	auth.daemonLog.LogError("ERROR: Unable to read from file UserDatabase.", err)

	intError = syscall.Flock(authFile.Fd(), 8) //8 is unlock
	if intError != 0 {
		auth.daemonLog.LogError("ERROR: Unable to unlock UserDatabase for reading.", os.NewError("Flock Syscall Failed"))
	}
	fi, err := authFile.Stat()

	auth.daemonLog.LogError("ERROR: Failed to stat file", err)

	err = authFile.Close()

	auth.lock.Lock()
	defer auth.lock.Unlock()

	err = json.Unmarshal(someBytes, &auth.Users)
	auth.daemonLog.LogError("ERROR: Failed to unmarshal data read from UserDatabase file.", err)
	auth.dbstamp = fi.Mtime_ns
	return
}

func (auth *Authinfo) HTTPAuthenticate(req *http.Request) (user string, valid bool, admin bool) {
	if _, ok := req.Header["Authorization"]; !ok {
		auth.daemonLog.LogError("Request header did not contain Authorization information.", os.NewError("HTTP Auth Missing"))
		return
	}

	header := req.Header.Get("Authorization")
	tmpAuthArray := strings.Split(header, " ")

	authValues, error := base64.URLEncoding.DecodeString(tmpAuthArray[1])
	auth.daemonLog.LogError("ERROR: Failed to decode encoded auth settings in http request.", error)

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
	auth.daemonLog.LogError(fmt.Sprintf("ERROR: Unable to open %s for reading.", auth.path), err)

	intError := syscall.Flock(authFile.Fd(), 2) //2 is exclusive lock
	if intError != 0 {
		auth.daemonLog.LogError(fmt.Sprintf("ERROR: Unable to lock %s for writing.", auth.path), os.NewError("Flock Syscall Failed"))
	}

	auth.lock.RLock()
	defer auth.lock.RUnlock()
	data, err := json.Marshal(auth.Users)

	_, err = authFile.Write(data)
	auth.daemonLog.LogError("ERROR: Unable to write to file .", err)

	intError = syscall.Flock(authFile.Fd(), 8) //8 is unlock
	if intError != 0 {
		auth.daemonLog.LogError("ERROR: Unable to unlock UserDatabase for reading.", os.NewError("Flock Syscall Failed"))
	}

	fi, err := authFile.Stat()
	auth.dbstamp = fi.Mtime_ns
	authFile.Close()
	return
}
