package daemon

type Daemon struct {
     Name      string
     DaemonLog *DaemonLogger
     AuthN     *Authinfo
     Cfg       *ConfigInfo
}

func New(name string, cfgFile string, authPath string) *Daemon {
     daemon := new(Daemon)
     daemon.Name = name
     daemon.AuthN = NewAuthInfo(authPath)
     daemon.Cfg = NewConfigInfo(cfgFile)
     daemon.DaemonLog = NewDaemonLogger(daemon.Cfg.Data["logfile"], daemon.Name)
     
     return daemon
}