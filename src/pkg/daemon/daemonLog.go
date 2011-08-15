package daemon

import (
	"log"
	"os"
	"time"
	"http"
	"flag"
)

type DaemonLogger struct {
	stdoutLog *log.Logger
	fileLog   *log.Logger
	error     int
	name      string
}

var debug bool

func init() {
	flag.BoolVar(&debug, "d", false, "Log debug information")
}

func NewDaemonLogger(logFilePath string, daemonName string) *DaemonLogger {
	daemonLogger := new(DaemonLogger)
	daemonLogger.name = daemonName
	daemonLogger.stdoutLog = log.New(os.Stdout, "", 0)
	logFile, _ := os.OpenFile(logFilePath+daemonName+".log", os.O_WRONLY|os.O_CREATE, 0666)
	daemonLogger.fileLog = log.New(logFile, "", 0)
	daemonLogger.error = 0
	return daemonLogger
}

func (daemonLogger *DaemonLogger) Log(message string) {
	currentTime := time.LocalTime()
	formatTime := currentTime.Format("Jan _2 15:04:05")
	name, _ := os.Hostname()
	pid := os.Getpid()
	daemonLogger.stdoutLog.Printf("%s %s %s[%d]: %s", formatTime, name, daemonLogger.name, pid, message)
	daemonLogger.fileLog.Printf("%s %s %s[%d]: %s", formatTime, name, daemonLogger.name, pid, message)
}

func (daemonLogger *DaemonLogger) LogError(message string, error os.Error) {
	currentTime := time.LocalTime()
	formatTime := currentTime.Format("Jan _2 15:04:05")
	name, _ := os.Hostname()
	pid := os.Getpid()
	if error != nil {
		daemonLogger.stdoutLog.Printf("%s %s %s[%d]: ERROR %s", formatTime, name, daemonLogger.name, pid, message)
		daemonLogger.fileLog.Printf("%s %s %s[%d]: ERROR %s", formatTime, name, daemonLogger.name, pid, message)
		daemonLogger.error++
	}

}

func (daemonLogger *DaemonLogger) LogHttp(request *http.Request) {
	currentTime := time.LocalTime()
	formatTime := currentTime.Format("Jan _2 15:04:05")
	name, _ := os.Hostname()
	pid := os.Getpid()
	daemonLogger.stdoutLog.Printf("%s %s %s[%d]: %s %s Bytes Recieved: %d", formatTime, name, daemonLogger.name, pid, request.Method, request.RawURL, request.ContentLength)
	daemonLogger.fileLog.Printf("%s %s %s[%d]: %s %s Bytes Recieved: %d", formatTime, name, daemonLogger.name, pid, request.Method, request.RawURL, request.ContentLength)
}

func (daemonLogger *DaemonLogger) DebugHttp(request *http.Request) {
	if debug {
		currentTime := time.LocalTime()
		formatTime := currentTime.Format("Jan _2 15:04:05")
		name, _ := os.Hostname()
		pid := os.Getpid()
		daemonLogger.stdoutLog.Printf("%s %s %s[%d]: %s %s Bytes Recieved: %d", formatTime, name, daemonLogger.name, pid, request.Method, request.RawURL, request.ContentLength)
		daemonLogger.fileLog.Printf("%s %s %s[%d]: %s %s Bytes Recieved: %d", formatTime, name, daemonLogger.name, pid, request.Method, request.RawURL, request.ContentLength)
	}
}

func (daemonLogger *DaemonLogger) LogDebug(message string) {
	currentTime := time.LocalTime()
	formatTime := currentTime.Format("Jan _2 15:04:05")
	name, _ := os.Hostname()
	pid := os.Getpid()
	if debug {
		daemonLogger.stdoutLog.Printf("%s %s %s[%d]: DEBUG %s", formatTime, name, daemonLogger.name, pid, message)
		daemonLogger.fileLog.Printf("%s %s %s[%d]: ERROR %s", formatTime, name, daemonLogger.name, pid, message)
	}
}

func (daemonLogger *DaemonLogger) ReturnError() int {
	return daemonLogger.error
}
