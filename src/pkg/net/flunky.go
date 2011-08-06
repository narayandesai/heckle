package flunky

import (
	"exec"
	"fmt"
	"http"
	"io"
	"io/ioutil"
	"json"
	"os"
	"strings"
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

	response, err := server.client.Get(fullpath)

	if err != nil {
		err = os.NewError(fmt.Sprintf("err is %s\n", err))
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
		os.NewError(fmt.Sprintf("File fetch of %s failed\n", path))
		return
	}

	runpath := os.TempDir() + path + fmt.Sprintf("%s", os.Getpid())

	server.DebugLog(fmt.Sprintf("runpath is %s", runpath))

	newbin, err := os.Create(runpath)
	if err != nil {
		err = os.NewError(fmt.Sprintf("Failed to create file %s\n", runpath))
		return
	}
	_, err = newbin.Write(data)
	if err != nil {
		err = os.NewError(fmt.Sprintf("Failed to write data\n"))
		return
	}
	err = newbin.Chmod(0777)
	if err != nil {
		err = os.NewError(fmt.Sprintf("Failed to chmod %s\n", runpath))
		return
	}

	newbin.Close()

	server.DebugLog(fmt.Sprintf("wrote executable to %s", runpath))

	fcmd := exec.Command(runpath)
	fcmd.Stdout = os.Stdout
	fcmd.Stderr = os.Stderr

	err = fcmd.Run()

	if err != nil {
		err = os.NewError(fmt.Sprintf("%s\n", err))
	}

	server.DebugLog(fmt.Sprintf("Exit status:%d", status))

	err = os.Remove(runpath)
	return
}

func (server *BuildServer) Post(path string, data io.Reader) (body []byte, err os.Error) {
	response, err := server.client.Post(server.URL+path, "text/plain", data)
	if err != nil {
		err = os.NewError(fmt.Sprintf("Post failed: %s\n", err))
		return
	}
	if response.StatusCode != 200 {
		err = os.NewError(string(response.StatusCode))
		return
	}
	server.DebugLog(fmt.Sprintf("POST response statuscode:%d", response.StatusCode))

	body, _ = ioutil.ReadAll(response.Body)
	response.Body.Close()

	return
}

type Communication struct {
	Locations map[string]string
	User      string
	Password  string
}

func NewCommunication(path string, user string, password string) (comm Communication, err os.Error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	comm.User = user
	comm.Password = password
	comm.Locations = make(map[string]string, 10)
	json.Unmarshal(data, &comm.Locations)
	return
}

func (comm *Communication) SetupClient(component string) (hclient *BuildServer, err os.Error) {
        location, ok := comm.Locations[component]
	if !ok {
		err = os.NewError("Compomnent Lookup Failure")
		return
	}

	parts := strings.Split(location, "://")
	url := fmt.Sprintf("%s://%s:%s@%s", parts[0], comm.User, comm.Password, parts[1])
	hclient = NewBuildServer(url, false)
	return
}
