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
}

var debug bool

func init() {
	flag.BoolVar(&debug, "d", false, "Log debug information")
}

func NewDaemonLogger(logFilePath string, daemonName string) *DaemonLogger {
	daemonLogger := new(DaemonLogger)
	daemonLogger.stdoutLog = log.New(os.Stdout, daemonName+": ", 0)
	logFile, _ := os.OpenFile(logFilePath+daemonName+".log", os.O_WRONLY|os.O_CREATE, 0666)
	daemonLogger.fileLog = log.New(logFile, daemonName+":", 0)
	/*if debug {
	   debugFile, _ := os.OpenFile(logFilePath + daemonName + ".log", os.O_WRONLY | os.O_CREATE, 0666)
	   daemonLogger.debug.file = log.New(debugFile, daemonName + ":", 0)
	   daemonLogger.debug.stdout = log.New(os.Stdout, daemonName + ": ", 0)
	}*/
	return daemonLogger
}

func (daemonLogger *DaemonLogger) Log(message string) {
	daemonLogger.stdoutLog.Printf("%s - INFO: %s", time.LocalTime(), message)
	daemonLogger.fileLog.Printf("%s - INFO: %s", time.LocalTime(), message)
}

func (daemonLogger *DaemonLogger) LogError(message string, error os.Error) {
	if error != nil {
		daemonLogger.stdoutLog.Printf("%s - ERROR: %s", time.LocalTime(), message)
		daemonLogger.fileLog.Printf("%s - ERROR: %s", time.LocalTime(), message)
	}
}

func (daemonLogger *DaemonLogger) LogHttp(request *http.Request) {
	daemonLogger.stdoutLog.Printf("%s - %s: %s %s Bytes Recieved: %d", time.LocalTime(), request.Method, request.RawURL, request.Proto, request.ContentLength)
	daemonLogger.fileLog.Printf("%s - %s: %s %s Bytes Recieved: %d", time.LocalTime(), request.Method, request.RawURL, request.Proto, request.ContentLength)
}

func (daemonLogger *DaemonLogger) LogDebug(message string) {
	if debug {
		daemonLogger.stdoutLog.Printf("%s - DEBUG: %s", time.LocalTime(), message)
		daemonLogger.fileLog.Printf("%s - DEBUG: %s", time.LocalTime(), message)
	}
}
