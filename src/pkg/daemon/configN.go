//Shouldn't have to care about authentication
//
package daemon

import (
	"sync"
	"os"
	"fmt"
	"syscall"
	"io/ioutil"
	"json"
)

type ConfigInfo struct {
	path    string
	lock    sync.RWMutex
	Data    map[string]string
	tmstamp int64
}

func (config *ConfigInfo) load() (err os.Error) {
	configFile, err := os.Open(config.path)
	errmsg := fmt.Sprintf("ERROR: Cannot open %s for reading", config.path)
	PrintError(errmsg, err)

	intError := syscall.Flock(configFile.Fd(), 2)
	errmsg = fmt.Sprintf("Error: Cannot read file for configurations")
	PrintError(errmsg, err)

	configContents, err := ioutil.ReadAll(configFile)
	errmsg = fmt.Sprintf("ERROR: Cannot read data from %s", config.path)
	PrintError(errmsg, err)

	intError = syscall.Flock(configFile.Fd(), 8)
	if intError != 0 {
		PrintError("Cannot unlock the config file for reading.", os.NewError("Flock sys call Failed"))
	}

	fi, err := configFile.Stat()
	errmsg = fmt.Sprintf("ERROR: Stat of %s failed", err)
	PrintError(errmsg, err)
	err = configFile.Close()
	PrintError("Cannot close file", err)

	config.lock.Lock()
	defer config.lock.Unlock()

	err = json.Unmarshal(configContents, &config.Data)
	PrintError("Cannot unmarshall config.Data", err)
	config.tmstamp = fi.Mtime_ns
	return

}


func NewConfigInfo(path string) *ConfigInfo {
	config := new(ConfigInfo)
	config.path = path
	config.Data = make(map[string]string)
     config.load()
     
	return config
}
