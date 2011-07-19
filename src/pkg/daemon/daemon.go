package daemon

type Daemon struct {
     Name      string
     DaemonLog *DaemonLogger
     AuthN     *Authinfo
     Cfg       *ConfigInfo
}

func New(name string, cfgFile string) *Daemon {
     daemon := new(Daemon)
     daemon.Name = name
     daemon.Cfg = NewConfigInfo(cfgFile)
     daemon.AuthN = NewAuthInfo(daemon.Cfg.Data["authpath"])
     daemon.DaemonLog = NewDaemonLogger(daemon.Cfg.Data["logfile"], daemon.Name)
     
     return daemon
}