package cache

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/connections"
	"github.com/ccammack/cannon/util"
)

func Shutdown() {
	// tell the client to disconnect
	connections.Broadcast(map[string]interface{}{
		"action": "shutdown",
	})

	// close all resources
	closeAll()
}

func prepareTemplateVars() map[string]interface{} {
	// set default values
	_, style := config.Style().String()
	data := map[string]interface{}{
		"style": template.CSS(style),
	}

	res, ok := currResource()
	if ok {
		// serve the converted output file (or error text on failure)
		data["title"] = template.HTMLEscapeString(filepath.Base(res.file))
		data["html"] = template.HTML(res.html)
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
		ch := make(chan *Resource)
		setCurrentResource(file, hash, ch)

		go func() {
			// wait for the resource to be become ready
			ready := <-ch
			curr, ok := currResource()
			if ok && curr == ready {
				// request a reload if the current resource is the one that just became ready
				connections.Broadcast(map[string]template.HTML{"action": "reload"})
			} else {
			}
		}()

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
		close(hash)

		body["status"] = template.HTML("success")
	} else {
		// not sure if this ever happens
		body["status"] = template.HTML("error")
		body["message"] = template.HTML(fmt.Sprintf("Error reading hash: %s", hash))
	}

	util.RespondJson(w, body)
}

func HandleFile(w http.ResponseWriter, r *http.Request) {
	reader, ok := currReader()
	if ok {
		http.ServeContent(w, r, filepath.Base(reader.Info.Name()), reader.Info.ModTime(), reader)
	} else {
		http.Error(w, "http.StatusServiceUnavailable", http.StatusServiceUnavailable)
	}
}
