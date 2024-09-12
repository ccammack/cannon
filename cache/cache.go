package cache

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/connections"

	"github.com/ccammack/cannon/util"
)

var (
	tempDir  string
	lock     sync.RWMutex
	resource *Resource
)

func init() {
	// create a temp directory on init
	dir, err := os.MkdirTemp("", "cannon")
	if err != nil {
		log.Panicf("error creating temp dir: %v", err)
	}
	tempDir = dir

	// react to config file changes
	config.RegisterCallback(func(event string) {
		if event == "reload" {
			lock.Lock()
			defer lock.Unlock()
			resource = nil
		}
	})
}

func Exit() {
	// clean up temp files
	if len(tempDir) > 0 {
		os.RemoveAll(tempDir)
	}
}

func Shutdown() {
	// tell the client to shutdown
	connections.Broadcast(map[string]template.HTML{
		"action": "shutdown",
	})
}

func prepareTemplateVars() map[string]interface{} {
	// set default values
	_, style := config.Style().String()
	data := map[string]interface{}{
		"style": template.CSS(style),
	}

	lock.Lock()
	defer lock.Unlock()

	if resource != nil && resource.Ready {
		// serve the converted output file (or error text on failure)
		data["title"] = template.HTMLEscapeString(filepath.Base(resource.file))
		data["html"] = template.HTML(resource.html)
	} else {
		// serve default values until the first resource is added
		data["title"] = template.HTMLEscapeString("Cannon preview")
		data["html"] = template.HTML("<p>Waiting for file...</p>")
	}

	return data
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
		vars := prepareTemplateVars()
		err := templ.Execute(w, vars)
		if err != nil {
			log.Printf("error generating page: %v", err)
		}
	}
}

// func HandleStatus(w http.ResponseWriter, r *http.Request) {
// 	// handle route /status
// 	data := map[string]template.HTML{
// 		"status": "success",
// 	}
// 	maps.Copy(data, FormatPageContent())
// 	util.RespondJson(w, data)
// }

func HandleUpdate(w http.ResponseWriter, r *http.Request) {
	// select a new file to display
	body := map[string]template.HTML{}

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
		lock.Lock()
		defer lock.Unlock()
		if resource != nil {
			resource.Close()
		}

		// create a resource and call back when finished
		resource = newResource(file, hash, func(res *Resource) {
			// if the new resource is still selected resource
			if res == resource {
				// tell the client to reload the page
				connections.Broadcast(map[string]template.HTML{
					"action": "reload",
				})
			}
		})

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
	// extract params from the request body
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Panicf("error decoding json payload: %v", err)
	}
	lock.Lock()
	defer lock.Unlock()
	if resource != nil {
		resource.Close()
	}
	resource = nil
	body := map[string]template.HTML{}
	body["status"] = template.HTML("success")
	util.RespondJson(w, body)
}

func HandleFile(w http.ResponseWriter, r *http.Request) {
	http.ServeContent(w, r, filepath.Base(resource.reader.Info.Name()), resource.reader.Info.ModTime(), resource.reader)
}
