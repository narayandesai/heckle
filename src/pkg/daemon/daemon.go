package daemon

type Daemon struct {
     Name      string
     DaemonLog *DaemonLogger
     AuthN     *Authinfo
     Cfg       *ConfigInfo
}

func New(name string) *Daemon {
     daemon := new(Daemon)
     daemon.Name = name
     daemon.DaemonLog = NewDaemonLogger("../../../etc/" + name + "/", daemon.Name)
     daemon.Cfg = NewConfigInfo("../../../etc/" + name + "/" + name + ".cfg", daemon.DaemonLog)
     daemon.AuthN = NewAuthInfo(daemon.Cfg.Data["authpath"], daemon.DaemonLog)
     
     return daemon
}