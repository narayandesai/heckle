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

	fullpath := server.URL + "/" + path

	response, _, err := server.client.Get(fullpath)

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
	data, err := server.Get(path)

	if err != nil {
		fmt.Fprintf(os.Stderr, "File fetch of %s failed\n", path)
		return
	}

	runpath := os.TempDir() + path + fmt.Sprintf("%s", os.Getpid())

	server.DebugLog(fmt.Sprintf("runpath is %s", runpath))

	newbin, err := os.Create(runpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create file %s\n", runpath)
		return
	}
	_, err = newbin.Write(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write data\n")
		return
	}
	err = newbin.Chmod(0777)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to chmod %s\n", runpath)
		return
	}

	newbin.Close()

	server.DebugLog(fmt.Sprintf("wrote executable to %s", runpath))

	cmd, err := exec.Run(runpath, []string{runpath}, os.Environ(), "/", exec.PassThrough, exec.PassThrough, exec.PassThrough)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	wmsg, err := cmd.Wait(0)
	status = wmsg.ExitStatus()

	server.DebugLog(fmt.Sprintf("Exit status:%d", status))

	err = os.Remove(runpath)
	return
}

func (server *BuildServer) Post(path string, data io.Reader) (body []byte, err os.Error) {
	response, err := server.client.Post(server.URL+path, "text/plain", data)
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
