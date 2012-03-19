package daemon

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

type DaemonLogger struct {
	stdoutLog *log.Logger
	fileLog   *log.Logger
	error     int
}

var debug bool

func init() {
	flag.BoolVar(&debug, "d", false, "Log debug information")
}

func NewDaemonLogger(logFilePath string, daemonName string) *DaemonLogger {
	daemonLogger := new(DaemonLogger)
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	header := fmt.Sprintf("%s %s[%d]: ", hostname, daemonName, pid)
	daemonLogger.stdoutLog = log.New(os.Stdout, header, log.LstdFlags)
	logFile, err := os.OpenFile(logFilePath+daemonName+".log", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("Failed to open logfile: ", err)
	} else {
		logFile.Seek(0, 2)
		daemonLogger.fileLog = log.New(logFile, header, log.LstdFlags)
	}
	daemonLogger.error = 0
	return daemonLogger
}

func (dLog DaemonLogger) PrintAll(message string) {
	dLog.stdoutLog.Print(message)
	if dLog.fileLog != nil {
		dLog.fileLog.Print(message)
	}
}

func (dLog *DaemonLogger) Log(message string) {
	dLog.PrintAll(message)
}

func (dLog *DaemonLogger) LogError(message string, error error) {
	if error != nil {
		dLog.PrintAll(message)
		dLog.error++
	}
}

func (dLog *DaemonLogger) LogHttp(request *http.Request) {
	msg := fmt.Sprintf("%s %s Bytes Recieved: %d", request.Method, request.RawURL, request.ContentLength)
	dLog.PrintAll(msg)
}

func (dLog *DaemonLogger) DebugHttp(request *http.Request) {
	if debug {
		msg := fmt.Sprintf("%s %s Bytes Recieved: %d", request.Method, request.RawURL, request.ContentLength)
		dLog.PrintAll(msg)
	}
}

func (dLog *DaemonLogger) LogDebug(message string) {
	if debug {
		dLog.PrintAll(message)
	}
}

func (daemonLogger *DaemonLogger) ReturnError() int {
	return daemonLogger.error
}
