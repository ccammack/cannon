/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

// TODO: golang serve file from memory
// TODO: golang lru cache
// https://www.alexedwards.net/blog/golang-response-snippets

package cache

import (
	"cannon/config"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.org/x/exp/maps"
)

// https://vivek-syngh.medium.com/http-response-in-golang-4ca1b3688d6
// https://programmer.help/blogs/golang-json-encoding-decoding-and-text-html-templates.html
// https://stackoverflow.com/questions/38436854/golang-use-json-in-template-directly
// https://gist.github.com/alex-leonhardt/8ed3f78545706d89d466434fb6870023
// https://gist.github.com/Integralist/d47c2e8c6064ec065108ad59df6e1fb9
// https://go.dev/blog/json
// https://www.sohamkamani.com/golang/json/
// https://stackoverflow.com/questions/30537035/golang-json-rawmessage-literal
// https://go.dev/play/p/C1tXFi23Bw
// https://appdividend.com/2022/06/22/golang-serialize-json-string/
// https://www.socketloop.com/tutorials/golang-marshal-and-unmarshal-json-rawmessage-struct-example
// https://noamt.medium.com/using-gos-json-rawmessage-a2371a1c11b7
// https://stackoverflow.com/questions/23255456/whats-the-proper-way-to-convert-a-json-rawmessage-to-a-struct
// https://jhall.io/pdf/Advanced%20JSON%20handling%20in%20Go.pdf
// https://codewithyury.com/how-to-correctly-serialize-json-string-in-golang/
// https://www.digitalocean.com/community/tutorials/how-to-use-json-in-go
// https://gobyexample.com/json
// https://yourbasic.org/golang/json-example/

type Resource struct {
	ready          bool
	filenameIn     string
	filenameInHash string
	filenameOut    string
	html           string
	htmlHash       string
}

type Resources struct {
	currentHash    string
	resourceLookup map[string]Resource
}

var (
	resources     *Resources
	resoursesLock = new(sync.RWMutex)
	tempDir       string
)

func Exit() {
	if len(tempDir) > 0 {
		os.RemoveAll(tempDir)
	}
}

func makeHash(s string) string {
	// TODO: is sha1 a good choice here?
	hash := sha1.New()
	hash.Write([]byte(s))
	hashstr := hex.EncodeToString(hash.Sum(nil))
	return hashstr
}

func init() {
	resources = new(Resources)
	resources.resourceLookup = make(map[string]Resource)

	// add a default resource
	// hash := "0"
	// html := "<p>Waiting for file...</p>"
	// resource := Resource{
	// 	"Cannon preview",
	// 	hash,
	// 	"",
	// 	html,
	// 	makeHash(html),
	// }
	// resources.resourceLookup[hash] = resource
	// resources.currentHash = hash

	config.RegisterCallback(func(event string) {
		if event == "reload" {
			resources := getResources()
			resources.resourceLookup = make(map[string]Resource)
		}
	})
}

func getResources() *Resources {
	resoursesLock.RLock()
	defer resoursesLock.RUnlock()
	return resources
}

func convertFile(file string, hash string) {
	// TODO: move file conversion into a coroutine
	// TODO: iterate config rules and run the matching one

	// create a temp directory the first time someone asks for a file
	if len(tempDir) == 0 {
		dir, err := ioutil.TempDir("", "cannon")
		if err != nil {
			panic(err)
		}
		tempDir = dir
	}

	// create a temp output file
	filePtr, err := ioutil.TempFile(tempDir, "preview")
	if err != nil {
		panic(err)
	}
	defer filePtr.Close()

	// simulate file conversion
	source, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer source.Close()
	destination, err := os.Create(filePtr.Name())
	if err != nil {
		panic(err)
	}
	defer destination.Close()

	// simulate conversion delay
	time.Sleep(10 * time.Second)

	html := "<img src='{document.location.href}file?hash=" + hash + "'>"

	resource := Resource{
		true, // set ready=true when when finished
		file,
		hash,
		filePtr.Name(),
		html,
		makeHash(html),
	}
	resources := getResources()
	resources.resourceLookup[hash] = resource
}

func setCurrentResource(file string) {
	hash := makeHash(file)
	resources := getResources()
	_, ok := resources.resourceLookup[hash]
	if !ok {
		// add a new null entry
		resources.resourceLookup[hash] = Resource{
			false,
			file,
			hash,
			"",
			"",
			"",
		}

		// perform file conversion and then fill out the resource
		go convertFile(file, hash)
	}
	resources.currentHash = hash
}

func getCurrentResourceData() map[string]string {
	// default values
	data := map[string]string{
		"interval": strconv.Itoa(config.GetConfig().Settings.Interval),
	}

	resources := getResources()
	if len(resources.resourceLookup) == 0 {
		// serve default values until the first file is selected
		html := "<p>Waiting for file...</p>"
		maps.Copy(data, map[string]string{
			"title":    "Cannon preview",
			"filehash": "0",
			"html":     html,
			"htmlhash": makeHash(html),
		})
		// data = map[string]string{
		// 	"title":    "Cannon preview",
		// 	"filehash": "0",
		// 	"html":     html,
		// 	"htmlhash": makeHash(html),
		// }
	} else {
		resource, ok := resources.resourceLookup[resources.currentHash]
		if !ok {
			panic("Resource lookup failed in cache.go!")
		}

		if !resource.ready {
			// serve stdout+stderr until the conversion sets the output filename
			html := "<p>Loading " + resource.filenameIn + "...</p>"
			maps.Copy(data, map[string]string{
				"title":    filepath.Base(resource.filenameIn),
				"filehash": resource.filenameInHash,
				"html":     html,
				"htmlhash": makeHash(html),
			})
			// data = map[string]string{
			// 	"title":    filepath.Base(resource.filenameIn),
			// 	"filehash": resource.filenameInHash,
			// 	"html":     html,
			// 	"htmlhash": makeHash(html),
			// }
		} else {
			// serve the converted output file
			maps.Copy(data, map[string]string{
				"title":    filepath.Base(resource.filenameIn),
				"filehash": resource.filenameInHash,
				"html":     resource.html,
				"htmlhash": resource.htmlHash,
			})
			// data = map[string]string{
			// 	"title":    filepath.Base(resource.filenameIn),
			// 	"filehash": resource.filenameInHash,
			// 	"html":     resource.html,
			// 	"htmlhash": resource.htmlHash,
			// }
		}
	}

	return data
}

const PageTemplate = `
<!doctype html>
<html>
	<head>
		<title>{{.title}}</title>
		<script>
			let filehash = "0";
			let htmlhash = "0";
			window.onload = function(e) {
				// copy server address from document.location.href
				const statusurl = document.location.href + "status";
				setTimeout(function status() {
					// ask the server for updates and reload if needed
					fetch(statusurl)
					.then((response) => response.json())
					.then((data) => {
						if ((filehash != data.filehash) || (htmlhash != data.htmlhash)) {
							filehash = data.filehash;
							htmlhash = data.htmlhash;
							document.title = data.title;
							const container = document.getElementById("container");
							if (container) {
								// copy server address from document.location.href
								const inner = data.html.replace("{document.location.href}", document.location.href);
								container.innerHTML = inner;
							}
						}
						setTimeout(status, {{.interval}});
					})
					.catch(err => {
						// Failed to load resource: net::ERR_CONNECTION_REFUSED
						document.title = "Cannon preview";
						const container = document.getElementById("container");
						if (container) {
							const inner = "<p>Disconnected from server: " + statusurl + "</p>";
							container.innerHTML = inner;
						}
					});
				}, {{.interval}});
			}
		</script>
	</head>
	<body>
		<div id="container"></div>
	</body>
</html>
`

func Page(w *http.ResponseWriter) {
	// emit html for the current page
	data := getCurrentResourceData()

	// resources := getResources()
	// resource, ok := resources.resourceLookup[resources.currentHash]
	// if !ok {
	// 	panic("Resource lookup failed in cache.go!")
	// }
	// data := map[string]string{
	// 	"title":       filepath.Base(resource.filenameIn),
	// 	"filehash":    resource.filenameInHash,
	// 	"html":        resource.html,
	// 	"htmlhash":    resource.htmlHash,
	// 	"interval":  "100",
	// }

	// write the current page to either w or stdout
	t := template.New("page")
	t, err := t.Parse(PageTemplate)
	if err != nil {
		panic(err)
	}
	if w != nil {
		t.Execute(*w, data)
	} else {
		t.Execute(os.Stdout, data)
	}
}

func Update(w *http.ResponseWriter, r *http.Request) {
	// update the current file to display

	// works
	// extract params from the request body
	// type Params struct {
	// 	File string `json:"file"`
	// }
	// var params Params
	// err := json.NewDecoder(r.Body).Decode(&params)
	// if err != nil {
	// 	panic(err)
	// }

	// extract params from the request body
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		panic(err)
	}
	// fmt.Print(params)
	// util.Append(fmt.Sprint(params))
	// util.Append(params["file"])

	// set the current file to display
	//setCurrentResource(params.file)
	setCurrentResource(params["file"])

	// works
	// type UpdateMessage struct {
	// 	State string `json:"state"`
	// }
	// body := UpdateMessage{
	// 	State: "updated",
	// }

	// write {state:updated} to either w or stdout
	body := map[string]string{
		"state": "updated",
	}

	if w != nil {
		(*w).Header().Set("Content-Type", "application/json")
		(*w).WriteHeader(http.StatusOK)
		json.NewEncoder(*w).Encode(body)
	} else {
		json.NewEncoder(os.Stdout).Encode(body)
	}
}

func File(w *http.ResponseWriter, r *http.Request) {
	// serve the requested file by hash
	hash := r.URL.Query().Get("hash")
	resources := getResources()
	resource, ok := resources.resourceLookup[hash]
	if !ok {
		panic("Resource lookup failed in cache.go!")
	}
	http.ServeFile(*w, r, resource.filenameOut)
}

func Status(w *http.ResponseWriter) {
	body := getCurrentResourceData()

	// body := map[string]string{}

	// resources := getResources()
	// if len(resources.resourceLookup) == 0 {
	// 	// serve default values until the first file is selected
	// 	html := "<p>Now We Are Because Waiting for file...</p>"
	// 	body = map[string]string{
	// 		"title":    "Cannon preview",
	// 		"filehash": "0",
	// 		"html":     html,
	// 		"htmlhash": makeHash(html),
	// 	}
	// } else {
	// 	resource, ok := resources.resourceLookup[resources.currentHash]
	// 	if !ok {
	// 		panic("Resource lookup failed in cache.go!")
	// 	}

	// 	if len(resource.filenameOut) == 0 {
	// 		// serve stdout+stderr until the conversion sets the output filename
	// 		html := "<p>Loading " + resource.filenameIn + "...</p>"
	// 		body = map[string]string{
	// 			"title":    filepath.Base(resource.filenameIn),
	// 			"filehash": resource.filenameInHash,
	// 			"html":     html,
	// 			"htmlhash": makeHash(html),
	// 		}
	// 	} else {
	// 		// serve the converted output file
	// 		body = map[string]string{
	// 			"title":    filepath.Base(resource.filenameIn),
	// 			"filehash": resource.filenameInHash,
	// 			"html":     resource.html,
	// 			"htmlhash": resource.htmlHash,
	// 		}
	// 	}
	// }

	// works
	// body := map[string]string{
	// 	"file": "file goes here",
	// }
	// util.RespondJson(w, body)

	// works
	// type StatusMessage struct {
	// 	Title    string `json:"title"`
	// 	Filehash string `json:"filehash"`
	// 	Html     string `json:"html"`
	// 	Htmlhash string `json:"htmlhash"`
	// }
	// body := StatusMessage{
	// 	Title:    filepath.Base(resource.filenameIn),
	// 	Filehash: resource.filenameInHash,
	// 	Html:     resource.html,
	// 	Htmlhash: resource.htmlHash,
	// }

	// works
	// var body map[string]interface{}
	// err := json.Unmarshal([]byte(`{"file": "the current file will also go here"}`), &body)
	// if err != nil {
	// 	panic(err)
	// }

	// works
	// var body json.RawMessage
	// err := body.UnmarshalJSON([]byte(`{"file": "the current file will go here"}`))
	// if err != nil {
	// 	panic(err)
	// }

	if w != nil {
		(*w).Header().Set("Content-Type", "application/json")
		(*w).WriteHeader(http.StatusOK)
		json.NewEncoder(*w).Encode(body)
	} else {
		json.NewEncoder(os.Stdout).Encode(body)
	}
}
