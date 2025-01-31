package resources

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ccammack/cannon/cache"
	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/connections"
	"github.com/ccammack/cannon/util"
)

var (
	resourceCache        = cache.New()
	tempDir       string = ""
	currHash      string = ""
	// currFile      string = ""
)

func deleteTempData() {
	// clear the resource cache
	resourceCache.Clear()

	// delete temp files
	if len(tempDir) > 0 {
		os.RemoveAll(tempDir)
	}
}

func init() {
	tempName := "cannon"
	tempDir = util.CreateTempDir(tempName)

	// react to config file changes
	config.RegisterCallback(func(event string) {
		if event == "reload" {
			deleteTempData()
		}
	})
}

func Shutdown() {
	// tell the client to disconnect
	connections.Broadcast(map[string]interface{}{
		"action": "shutdown",
	})

	deleteTempData()
}

func prepareTemplateVars(hash string) map[string]interface{} {
	// set default values
	_, style := config.Style().String()
	data := map[string]interface{}{
		"style": template.CSS(style),
	}

	status, result := resourceCache.Get(hash)
	if status == cache.StatusReady {
		// serve the converted output file (or error text on failure)
		res := result.(*Resource)
		data["title"] = template.HTMLEscapeString(filepath.Base(res.file))
		data["hash"] = template.HTML(res.hash)
		data["html"] = template.HTML(res.html)
	} else {
		// serve default values until the first resource is added
		data["title"] = template.HTMLEscapeString("Cannon preview")
		data["hash"] = template.HTML("")
		data["html"] = template.HTML("<p>Waiting for file...</p>")
	}

	return data
}

func BroadcastCurrent() {
	// send the current resource to the clients
	status, _ := resourceCache.Get(currHash)
	connections.Broadcast(map[string]interface{}{
		"action": "update",
		"hash":   currHash,
		"ready":  (status == cache.StatusReady),
	})
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	// handle route /
	if r.Header.Get("Upgrade") == "websocket" {
		// handle socket connection requests
		err := connections.New(w, r)
		if err != nil {
			http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		}
	} else {
		// handle normal page generation
		templ := template.Must(template.New("page").Parse(PageTemplate))
		vars := prepareTemplateVars(currHash)
		err := templ.Execute(w, vars)
		if err != nil {
			log.Printf("error generating page: %v", err)
		}
	}
}

func HandleDisplay(w http.ResponseWriter, r *http.Request) {
	// select a new file to display
	body := map[string]interface{}{}

	// extract params from the request body
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Panicf("error decoding json payload: %v", err)
	}

	// TODO: consider using file.ToLower() as the key rather than hashing
	file := params["file"]
	hash := params["hash"]

	if file != "" && hash != "" {
		// create a new resource
		res := NewResource(tempDir, file, hash)
		resourceCache.Put(hash, res)
		currHash = hash
		body["status"] = template.HTML("success")
	} else {
		// this is reached sometimes after deleting a file with lf
		body["status"] = template.HTML("error")
		body["message"] = template.HTML(fmt.Sprintf("Error reading file or hash: %s %s", file, hash))
	}

	// respond
	util.RespondJson(w, body)
}

func HandleClose(w http.ResponseWriter, r *http.Request) {
	// close the specified resource
	body := map[string]interface{}{}

	// extract params from the request body
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Panicf("error decoding json payload: %v", err)
	}

	hash := params["hash"]

	if hash != "" {
		// close the resource
		resourceCache.Evict(hash)
		body["status"] = template.HTML("success")
	} else {
		// not sure if this ever happens
		body["status"] = template.HTML("error")
		body["message"] = template.HTML(fmt.Sprintf("Error reading hash: %s", hash))
	}

	util.RespondJson(w, body)
}

func HandleSrc(w http.ResponseWriter, r *http.Request) {
	status, result := resourceCache.Get(currHash)
	res := result.(*Resource)
	if status == cache.StatusReady {
		reader := res.reader
		http.ServeContent(w, r, filepath.Base(reader.Info.Name()), reader.Info.ModTime(), reader)
	} else {
		http.Error(w, "http.StatusServiceUnavailable", http.StatusServiceUnavailable)
	}
}
