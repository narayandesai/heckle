package flunky

import (
	"fmt"
	"http"
	"io/ioutil"
	"os"
)

type BuildServer struct {
	URL string
	client http.Client
	debug bool
}

func NewBuildServer(serverURL string) *BuildServer {
	var client http.Client
	return &BuildServer{serverURL, client, true}
}

func (server *BuildServer) Get(path string) (body []byte, err os.Error) {

	fullpath := server.URL + "/" + path

	response, _, err := server.client.Get(fullpath)

	if err != nil {
		fmt.Fprintf(os.Stderr, "err is %s\n", err)
		return
	}

	body, _ = ioutil.ReadAll(response.Body)
    response.Body.Close()

	return
}