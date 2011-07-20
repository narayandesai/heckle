package daemon

import (
     "log"
     "os"
     "time"
     "http"
)

type DaemonLogger struct {
     stdoutLog *log.Logger
     fileLog   *log.Logger
}

func NewDaemonLogger(logFileName string, daemonName string) *DaemonLogger {
     daemonLogger := new(DaemonLogger)
     daemonLogger.stdoutLog = log.New(os.Stdout, daemonName + ": ", 0)
     logFile, _ := os.OpenFile(logFileName, os.O_WRONLY | os.O_CREATE, 0666)
     daemonLogger.fileLog = log.New(logFile, daemonName + ": ", 0)
     
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

func (daemonLogger *DaemonLogger)LogHttp(request *http.Request){
     //if req.Status != "200 OK"{
     	  daemonLogger.stdoutLog.Printf("%s - %s: %s %s", time.LocalTime(), request.Method, request.RawURL, request.Proto)
	  daemonLogger.fileLog.Printf("%s - %s: %s %s", time.LocalTime(), request.Method, request.RawURL, request.Proto)
    /* }else{
	  daemonLogger.stdoutLog.Printf("%s -- %s %s %s", time.LocalTime(), request.Method, request.RawURL, request.Proto, request.Status)
	  daemonLogger.fileLog.Printf("%s -- %s %s %s", time.LocalTime(), request.Method, request.RawURL, request.Proto, request.Status)
     }*/
}
	   

