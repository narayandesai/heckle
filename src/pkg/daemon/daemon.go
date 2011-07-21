package daemon

type Daemon struct {
     Name      string
     DaemonLog *DaemonLogger
     AuthN     *Authinfo
     Cfg       *ConfigInfo
}

func New(name string, fileDir string) *Daemon {
     daemon := new(Daemon)
     daemon.Name = name
     daemon.DaemonLog = NewDaemonLogger(fileDir, daemon.Name)
     daemon.Cfg = NewConfigInfo(fileDir + name + ".cfg", daemon.DaemonLog)
     daemon.AuthN = NewAuthInfo(daemon.Cfg.Data["authpath"], daemon.DaemonLog)
     
     return daemon
}