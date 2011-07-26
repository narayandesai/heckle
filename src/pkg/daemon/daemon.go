package daemon

var FileDir string

func init() {
     flag.StringVar(&FileDir, "F", "/etc/heckle/", "Directory where daemon files can be found.")
}

type Daemon struct {
     Name      string
     DaemonLog *DaemonLogger
     AuthN     *Authinfo
     Cfg       *ConfigInfo
}

func New(name string) *Daemon {
     daemon := new(Daemon)
     daemon.Name = name
     daemon.DaemonLog = NewDaemonLogger(FileDir, daemon.Name)
     daemon.Cfg = NewConfigInfo(FileDir + name + ".conf", daemon.DaemonLog)
     daemon.AuthN = NewAuthInfo(FileDir + "users.db", daemon.DaemonLog)
     
     return daemon
}