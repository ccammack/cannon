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
	"golang.org/x/exp/maps"
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

func FormatPageContent() map[string]template.HTML {
	// set default values
	data := map[string]template.HTML{}

	lock.Lock()
	defer lock.Unlock()

	if resource != nil && resource.Ready {
		// serve the converted output file (or error text on failure)
		maps.Copy(data, map[string]template.HTML{
			"title": template.HTML(filepath.Base(resource.file)),
			"html":  template.HTML(resource.html),
		})
		// } else if resource != nil {
		// 	// serve a spinner while waiting for the next resource
		// 	// https://codepen.io/nikhil8krishnan/pen/rVoXJa
		// 	maps.Copy(data, map[string]template.HTML{
		// 		"title": template.HTML(filepath.Base("Loading...")),
		// 		"html":  template.HTML(SpinnerTemplate),
		// 	})
	} else {
		// serve default values until the first resource is added
		html := "<p>Waiting for file...</p>"
		maps.Copy(data, map[string]template.HTML{
			"title": "Cannon preview",
			"html":  template.HTML(html),
		})
	}

	return data
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	// handle route /
	if r.Header.Get("Upgrade") == "websocket" {
		err := connections.New(w, r)
		if err != nil {
			http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		}
	} else {
		data := FormatPageContent()

		// generate complete html page from template
		t, err := template.New("page").Parse(PageTemplate)
		util.CheckPanicOld(err)
		err = t.Execute(w, data)
		util.CheckPanicOld(err)

		// accept := r.Header.Get("Accept")
		// if accept == "" || strings.Contains(accept, "text/html") {
		// 	// html
		// 	err = t.Execute(w, data)
		// 	util.CheckPanicOld(err)
		// } else {
		// 	// json
		// 	data["page"] = template.HTML(t.Tree.Root.String())
		// 	util.RespondJson(w, data)
		// }
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
