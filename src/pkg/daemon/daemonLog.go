package daemon

import (
     "log"
     "os"
)

type DaemonLogger struct {
     stdoutLog log.Logger
     fileLog   log.Logger
}

func NewDaemonLogger(logFileName string, daemonName string) *DaemonLogger {
     daemonLogger := new(DaemonLogger)
     daemonLogger.stdoutLog = log.New(os.Stdout, daemonName + ": ", 0)
     logFile, _ := os.OpenFile(logFileName, os.O_WRONLY | os.O_CREAT, 0666)
     daemonLogger.fileLog = log.New(logFile, daemonName + ": ", 0)
     
     return daemonLogger
}

func (daemonLogger *DaemonLogger) Log(message string) {
     daemonLogger.stdoutLog.Print(message)
     daemonLogger.fileLog.Print(message)
}