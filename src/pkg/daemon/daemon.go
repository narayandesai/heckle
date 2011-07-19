package daemon

type Daemon struct {
     name      string
     DaemonLog DaemonLogger
     AuthN     AuthInfo
     Cfg       ConfigInfo
}

func New(name string, cfgFile string, authPath string) *Daemon {
     daemon := new(Daemon)
     daemon.Name = name
     daemon.AuthN = NewAuthInfo(path)
     daemon.Cfg = NewConfigInfo(cfgFile)
     daemon.DaemonLog = NewDaemonLogger(Cfg.Data["logfile"])
     
     return daemon
}