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
	"github.com/ccammack/cannon/util"
	"golang.org/x/exp/maps"
)

var (
	tempDir string
	lock    sync.RWMutex
	currRes *Resource
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
			currRes = nil
		}
	})
}

func Exit() {
	// clean up temp files
	if len(tempDir) > 0 {
		os.RemoveAll(tempDir)
	}
}

func FormatPageContent() map[string]template.HTML {
	// set default values
	_, interval := config.Interval().String()
	data := map[string]template.HTML{
		"interval": template.HTML(interval),
	}

	lock.Lock()
	defer lock.Unlock()

	if currRes != nil && currRes.Ready {
		// serve the converted output file (or error text on failure)
		maps.Copy(data, map[string]template.HTML{
			"title":    template.HTML(filepath.Base(currRes.input)),
			"html":     template.HTML(currRes.html),
			"htmlhash": template.HTML(currRes.htmlHash),
		})
	} else if currRes != nil {
		// serve a spinner while waiting for the next resource
		// https://codepen.io/nikhil8krishnan/pen/rVoXJa
		maps.Copy(data, map[string]template.HTML{
			"title":    template.HTML(filepath.Base("Loading...")),
			"html":     template.HTML(SpinnerTemplate),
			"htmlhash": template.HTML(util.MakeHash(SpinnerTemplate)),
		})
	} else {
		// serve default values until the first resource is added
		html := "<p>Waiting for file...</p>"
		maps.Copy(data, map[string]template.HTML{
			"title":    "Cannon preview",
			"html":     template.HTML(html),
			"htmlhash": template.HTML(util.MakeHash(html)),
		})
	}

	return data
}

func Update(w http.ResponseWriter, r *http.Request) {
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
		if currRes != nil && currRes.reader != nil {
			currRes.reader.Cancel()
		}
		currRes = newResource(file, hash)
		body["status"] = template.HTML("success")
	} else {
		// this is reached sometimes after deleting a file with lf
		body["status"] = template.HTML("error")
		body["message"] = template.HTML(fmt.Sprintf("Error reading file or hash: %s %s", file, hash))
	}

	// respond
	util.RespondJson(w, body)
}

func Close(w http.ResponseWriter, r *http.Request) {
	// extract params from the request body
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Panicf("error decoding json payload: %v", err)
	}
	lock.Lock()
	defer lock.Unlock()
	if currRes != nil && currRes.reader != nil {
		currRes.reader.Cancel()
	}
	currRes = nil
	body := map[string]template.HTML{}
	body["status"] = template.HTML("success")
	util.RespondJson(w, body)
}

func File(w http.ResponseWriter, r *http.Request) {
	http.ServeContent(w, r, filepath.Base(currRes.reader.Info.Name()), currRes.reader.Info.ModTime(), currRes.reader)
}
