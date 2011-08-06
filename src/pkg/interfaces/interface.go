package interfaces

type Ctlmsg struct {
     Addresses []string
     Time      int64
     Image     string
     AllocNum  uint64
     Extra     map[string]string
}

type Listmsg struct {
     Addresses           []string
     Image               string
     ActivityTimeout     int64
     AllocNum            uint64
}

type Nummsg struct {
     NumNodes            int
     Image               string
     ActivityTimeout     int64
}

type StatusMessage struct {
     Status         string
     LastActivity   int64
     Info           []InfoMsg
}

type InfoMsg struct {
     Time    int64
     Message string
     MsgType string
}

