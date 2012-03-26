package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"syscall"
)

type ConfigInfo struct {
	path      string
	lock      sync.RWMutex
	Data      map[string]string
	tmstamp   int64
	daemonLog *DaemonLogger
}

func (config *ConfigInfo) load() (err error) {
	configFile, err := os.Open(config.path)
	config.daemonLog.LogError(fmt.Sprintf("Cannot open %s for reading", config.path), err)
	if err != nil {
		os.Exit(1)
	}
	intError := syscall.Flock(configFile.Fd(), 2)
	config.daemonLog.LogError("Error: Cannot read file for configurations", err)

	configContents, err := ioutil.ReadAll(configFile)
	config.daemonLog.LogError(fmt.Sprintf("Cannot read data from %s", config.path), err)

	intError = syscall.Flock(configFile.Fd(), 8)
	if intError != 0 {
		config.daemonLog.LogError("Cannot unlock the config file for reading.", errors.New("Flock sys call Failed"))
	}

	fi, err := configFile.Stat()
	config.daemonLog.LogError(fmt.Sprintf("Stat of %s failed", err), err)
	err = configFile.Close()
	config.daemonLog.LogError("Cannot close file", err)

	config.lock.Lock()
	defer config.lock.Unlock()

	err = json.Unmarshal(configContents, &config.Data)
	config.daemonLog.LogError("Cannot unmarshall config.Data", err)
	config.tmstamp = fi.Mtime_ns
	return
}

func NewConfigInfo(path string, daemonLog *DaemonLogger) *ConfigInfo {
	config := new(ConfigInfo)
	config.path = path
	config.Data = make(map[string]string)
	config.daemonLog = daemonLog
	config.load()
	return config
}
