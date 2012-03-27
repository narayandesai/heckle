package daemon

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
)

type Auth struct {
	User     string
	Password string
}

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

func (auth *Authinfo) Load() (err error) {
	if auth.path == "" {
		err = errors.New("Authorization file does not exsist")
		return
	}
	authFile, err := os.Open(auth.path)
	emsg := fmt.Sprintf("Unable to open %s for reading.", auth.path)
	if err != nil {
		err = errors.New(emsg)
		os.Exit(1)
	}

	err = syscall.Flock(int(authFile.Fd()), 2) //2 is exclusive lock
	if (err != nil) {
		emsg = fmt.Sprintf("Unable to lock %s for reading.", auth.path)
		err = errors.New(emsg)
	}

	someBytes, err := ioutil.ReadAll(authFile)
	auth.daemonLog.LogError("Unable to read from file UserDatabase.", err)

	err = syscall.Flock(int(authFile.Fd()), 8) //8 is unlock
	if (err != nil) {
		err = errors.New("Flock Syscall Failed")
	}
	fi, err := authFile.Stat()
	auth.daemonLog.LogError("Failed to stat file", err)

	err = authFile.Close()

	auth.lock.Lock()
	defer auth.lock.Unlock()

	err = json.Unmarshal(someBytes, &auth.Users)
	auth.daemonLog.LogError("Failed to unmarshal data read from UserDatabase file.", err)
	auth.dbstamp = fi.ModTime().Unix()
	return
}

func (auth *Authinfo) GetHTTPAuthenticateInfo(req *http.Request) (user string, valid bool, admin bool) {
	if _, ok := req.Header["Authorization"]; !ok {
		auth.daemonLog.LogError("Request header did not contain Authorization information.", errors.New("HTTP Auth Missing"))
		return
	}

	header := req.Header.Get("Authorization")
	tmpAuthArray := strings.Split(header, " ")

	authValues, error := base64.URLEncoding.DecodeString(tmpAuthArray[1])
	auth.daemonLog.LogError("Failed to decode encoded auth settings in http request.", error)

	authValuesArray := strings.Split(string(authValues), ":")
	user = authValuesArray[0]
	password := authValuesArray[1]
	valid, admin = auth.GetAuthenticateCred(user, password)
	return
}

func (auth *Authinfo) GetAuthenticateCred(user string, password string) (valid bool, admin bool) {
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

func (auth *Authinfo) Authenticate(user string, password string, isAdmin bool) (err error) {
	auth.lock.RLock()
	defer auth.lock.RUnlock()

	_, ok := auth.Users[user]
	if !ok {
		err = errors.New(fmt.Sprintf("User does not exsist"))
		return
	}

	valid := (password == auth.Users[user].Password)
	if !valid {
		err = errors.New(fmt.Sprintf("Invalid Password"))
		return
	}

	if isAdmin {
		admin := auth.Users[user].Admin
		if !admin {
			err = errors.New(fmt.Sprintf("Authorization denied, not administrator"))
			return
		}
	}
	return
}

func (auth *Authinfo) HTTPAuthenticate(req *http.Request, isAdmin bool) (err error) {
	if _, ok := req.Header["Authorization"]; !ok {
		err = errors.New("Request header did not contain Authorization information.")
		return
	}
	header := req.Header.Get("Authorization")
	tmpAuthArray := strings.Split(header, " ")

	authValues, err := base64.URLEncoding.DecodeString(tmpAuthArray[1])
	if err != nil {
		auth.daemonLog.LogError("Failed to decode encoded auth settings in http request.", err)
	}

	authValuesArray := strings.Split(string(authValues), ":")
	user := authValuesArray[0]
	password := authValuesArray[1]
	err = auth.Authenticate(user, password, isAdmin)
	return
}

func (auth *Authinfo) NewUser(user string, password string, admin bool) {
	auth.lock.Lock()
	defer auth.lock.Unlock()
	return
}

func (auth *Authinfo) DelUser(user string) (err error) {
	auth.lock.Lock()
	defer auth.lock.Unlock()
	return
}

func (auth *Authinfo) Store() (err error) {

	authFile, err := os.Open(auth.path)
	auth.daemonLog.LogError(fmt.Sprintf("Unable to open %s for reading.", auth.path), err)

	err = syscall.Flock(int(authFile.Fd()), 2) //2 is exclusive lock
	if (err != nil) {
		err = errors.New(fmt.Sprintf("Failed to lock %s for reading.", auth.path))
	}

	auth.lock.RLock()
	defer auth.lock.RUnlock()
	data, err := json.Marshal(auth.Users)

	_, err = authFile.Write(data)
	auth.daemonLog.LogError("Unable to write to file .", err)

	err = syscall.Flock(int(authFile.Fd()), 8) //8 is unlock
	if (err != nil) {
		err = errors.New(fmt.Sprintf("Unable to unlock %s for reading.", auth.path))
	}

	fi, err := authFile.Stat()
	auth.dbstamp = fi.ModTime().Unix()
	authFile.Close()
	return
}

func (auth *Authinfo) GetUserAuth() (user string, password string, err error) {
	var authdata Auth
	homedir := os.Getenv("HOME")

	rawdata, err := ioutil.ReadFile(homedir + "/.hauth")
	if err != nil {
		return
	}
	json.Unmarshal(rawdata, &authdata)
	user = authdata.User
	password = authdata.Password
	return

}
