package heckleFuncs

import (
     "strings"
     "os"
     "fmt"
     "json"
     "io/ioutil"
     "syscall"
     "encoding/base64"
     "./heckleTypes"
)

func PrintError(errorMsg string, error os.Error) {
     //This function prints the error passed if error is not nil.
     if error != nil {
          fmt.Fprintf(os.Stderr, "%s\n", errorMsg)
     }
}
     
func Authenticate(tmpAuth string, path string) (username string, authed bool, admin bool) {
     tmpAuthArray := strings.Split(tmpAuth, " ")
     
     authValues , error := base64.StdEncoding.DecodeString(tmpAuthArray[1])
     PrintError("ERROR: Failed to decode encoded auth settings in http request.", error)
     
     authValuesArray := strings.Split(string(authValues), ":")
     username = authValuesArray[0]
     password := authValuesArray[1]
     
     var auth  map[string]heckleTypes.UserNode
     
     authFile, error := os.Open(path + "UserDatabase")
     PrintError("ERROR: Unable to open UserDatabase for reading.", error)
     
     intError := syscall.Flock(authFile.Fd(), 2) //2 is exclusive lock
     if intError != 0 {
          PrintError("ERROR: Unable to lock UserDatabase for reading.", os.NewError("Flock Syscall Failed"))
     }
     
     someBytes, error := ioutil.ReadAll(authFile)
     PrintError("ERROR: Unable to read from file UserDatabase.", error)
     
     intError = syscall.Flock(authFile.Fd(), 8) //8 is unlock
     if intError != 0 {
          PrintError("ERROR: Unable to unlock UserDatabase for reading.", os.NewError("Flock Syscall Failed"))
     }
     
     error = authFile.Close()
     PrintError("ERROR: Failed to close UserDatabase.", error)
     
     error = json.Unmarshal(someBytes, &auth)
     PrintError("ERROR: Failed to unmarshal data read from UserDatabase file.", error)
     
     authed = (password == auth[username].Password)
     admin = auth[username].Admin
     
     return
}

/*func AddUser(writer http.ResponseWriter, request *http.Request) {
     //add comment
     var auth map[string]heckleTypes.UserNode
     var addUserMsg map[string]heckleTypes.UserNode
     request.ProtoMinor = 0

     _, authed, admin := Authenticate(request.Header.Get("Authorization"))
     
     if !authed {
          PrintError("ERROR: Username password combo invalid.", os.NewError("Access Denied"))
          return
     }
     
     if !admin {
          PrintError("ERROR: Need to be an admin to access add user command.", os.NewError("Access Denied"))
          return
     }
          
     someBytes, error := ioutil.ReadAll(request.Body)
     PrintError("ERROR: Unable to read all from allocate list POST.", error)
     
     error = request.Body.Close()
     PrintError("ERROR: Failed to close allocation list request body.", error)
     
     error = json.Unmarshal(someBytes, &addUserMsg)
     PrintError("ERROR: Unable to unmarshal allocation list.", error)
     
     authFile, error := os.Open("UserDatabase")
     PrintError("ERROR: Unable to open UserDatabase for reading.", error)
     
     intError := syscall.Flock(authFile.Fd(), 2) //2 is exclusive lock
     if intError != 0 {
          PrintError("ERROR: Unable to lock UserDatabase for reading.", os.NewError("Flock Syscall Failed"))
     }
     
     someBytes, error = ioutil.ReadAll(authFile)
     PrintError("ERROR: Unable to read from file UserDatabase.", error)
     
     error = json.Unmarshal(someBytes, &auth)
     PrintError("ERROR: Failed to unmarshal data read from UserDatabase file.", error)

     for key, value := range addUserMsg {
          auth[key] = value
     }
     
     someBytes, error = json.Marshal(auth)
     PrintError("ERROR: Failed to marshal new database info.", error)
     
     error = authFile.Truncate(0)
     PrintError("ERROR: Failed to truncate database file.", error)
     
     _, error = authFile.WriteAt(someBytes, 0)
     PrintError("ERROR: Failed to write new database info to auth file.", error)
     
     intError = syscall.Flock(authFile.Fd(), 2) //2 is exclusive lock
     if intError != 0 {
          PrintError("ERROR: Unable to lock UserDatabase for reading.", os.NewError("Flock Syscall Failed"))
     }
     
     error = authFile.Close()
     PrintError("ERROR: Failed to close UserDatabase.", error)
     
}*/