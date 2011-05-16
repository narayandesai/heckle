package simpleclient

import (
    "io/ioutil"
    "http"
    "json"
    "strings"
    "fmt"
)

type BuildClient struct {
    hostname string
    client http.Client
}

type Message struct {
    StartOn string
    Supplies string
    Requires []string
    Script string
}

func (t *Message) String() string {
    requires := "Requires:"
    for i := range t.Requires {
        requires = strings.Join([]string{requires, t.Requires[i]}, "\n\t") 
    }
    return fmt.Sprintf("%s\n%s\n%s\n%s", t.StartOn, t.Supplies, requires, t.Script)
}

// TODO: read build_server URL from some type of conf file
const (
    BUILDSERVER = "http://localhost:8080"
)

/*
 BuildClient class methods
 */
// class init
func NewBuildClient(hostname string) *BuildClient {
    var client http.Client
    return &BuildClient{hostname, client}
}

// get hostname
func (t *BuildClient) GetHostname() (string) {
    return t.hostname
}

// HTTP get
func (t *BuildClient) Get(sub_url string) (string) {
    c := t.client
    r, _, err := c.Get(BUILDSERVER + "/" + sub_url)
    if err != nil {
        return "ERROR"
    }
    body, _ := ioutil.ReadAll(r.Body)
    r.Body.Close()
    var n Message
    err2 := json.Unmarshal(body, &n)
    if err2 != nil {
        return "Unmarshal Error!"
    }
    return n.String() 
}


