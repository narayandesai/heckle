package flunky

import (
	"exec"
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

func (server *BuildServer) Run(path string) (status int, err os.Error) {
	status = 255
	data, err := server.Get(path)

	runpath := os.TempDir() + path + fmt.Sprintf("%s", os.Getpid())

	if server.debug {
		fmt.Fprintf(os.Stderr, "runpath is %s\n", runpath)
	}

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

	if server.debug {
		fmt.Fprintf(os.Stderr, "wrote executable to %s\n", runpath)
	}

	cmd, err := exec.Run(runpath, []string{runpath}, os.Environ(), "/", exec.PassThrough, exec.PassThrough, exec.PassThrough)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	wmsg, err := cmd.Wait(0)
	status = wmsg.ExitStatus()

	if server.debug {
		fmt.Fprintf(os.Stderr, "status is %d\n", status)
	}

	err = os.Remove(runpath)
	return
};