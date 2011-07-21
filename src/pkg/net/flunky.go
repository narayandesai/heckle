package flunky

import (
	"exec"
	"fmt"
	"http"
	"io"
	"io/ioutil"
	"os"
)

type BuildServer struct {
	URL    string
	client http.Client
	debug  bool
}

func NewBuildServer(serverURL string, debug bool) *BuildServer {
	var client http.Client
	return &BuildServer{serverURL, client, debug}
}

func (server *BuildServer) DebugLog(message string) {
	if server.debug {
		fmt.Println(message)
	}
}

func (server *BuildServer) Get(path string) (body []byte, err os.Error) {
     tmpRequest, err := http.NewRequest("GET", server.URL + "/" + path, nil)
     tmpRequest.Header.Set("Content-Type", "text/plain")
     
     response, err := server.client.Do(tmpRequest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err is %s\n", err)
		return
	}

	body, _ = ioutil.ReadAll(response.Body)
	response.Body.Close()

	if response.StatusCode != 200 {
		err = os.NewError("Fetch Failed")
		return
	}

	return
}

func (server *BuildServer) Run(path string) (status int, err os.Error) {
	status = 255

     tmpRequest, err := http.NewRequest("GET", path, nil)
     tmpRequest.Header.Set("Content-Type", "text/plain")
     
     response, err := server.client.Do(tmpRequest)
	if err != nil {
		return
	}

     data, err := ioutil.ReadAll(response.Body)
     if err != nil {
          return
     }
     
	newbin, err := ioutil.TempFile("", "bin")
	if err != nil {
		return
	}

	runpath := newbin.Name()
	server.DebugLog(fmt.Sprintf("runpath is %s", runpath))

	_, err = newbin.Write(data)
	if err != nil {
		return
	}
	err = newbin.Chmod(0777)
	if err != nil {
		return
	}

	newbin.Close()

	server.DebugLog(fmt.Sprintf("wrote executable to %s", runpath))

     err = exec.Command(runpath).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	err = os.Remove(runpath)
	return
}

func (server *BuildServer) Post(path string, data io.Reader) (body []byte, err os.Error) {
	//response, err := server.client.Post(server.URL+path, "text/plain", data)
     
     tmpRequest, err := http.NewRequest("POST", server.URL+path, data)
     tmpRequest.Header.Set("Content-Type", "text/plain")
     
     response, err := server.client.Do(tmpRequest)
     
	if err != nil {
		fmt.Fprintf(os.Stderr, "Post failed: %s\n", err)
		return
	}

	if response.StatusCode != 200 {
		server.DebugLog(fmt.Sprintf("POST response statuscode:%d", response.StatusCode))
		err = os.NewError("Fetch Failed")
		return
	}

	body, _ = ioutil.ReadAll(response.Body)
	response.Body.Close()

	return
}
