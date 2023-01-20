/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

package cache

import (
	"cannon/config"
	"cannon/util"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/exp/maps"
)

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

type Resource struct {
	ready          bool
	inputName      string
	inputNameHash  string
	combinedOutput string
	outputName     string
	html           string
	htmlHash       string
}

var resources = struct {
	lock    sync.RWMutex
	current string
	lookup  map[string]Resource
	tempDir string
}{lookup: make(map[string]Resource)}

func reloadCallback(event string) {
	if event == "reload" {
		resources.lock.Lock()
		resources.current = ""
		resources.lookup = make(map[string]Resource)
		resources.tempDir = ""
		resources.lock.Unlock()
	}
}

func getResource(hash string) (Resource, bool) {
	resources.lock.Lock()
	resource, ok := resources.lookup[hash]
	resources.lock.Unlock()
	return resource, ok
}

func setResource(hash string, resource Resource) {
	resources.lock.Lock()
	resources.lookup[hash] = resource
	resources.lock.Unlock()
}

func getCurrentHash() string {
	resources.lock.Lock()
	defer resources.lock.Unlock()
	return resources.current
}

func setCurrentHash(hash string) {
	resources.lock.Lock()
	resources.current = hash
	resources.lock.Unlock()
}

func createPreviewFile() string {
	// create a temp directory on the first call
	resources.lock.Lock()
	defer resources.lock.Unlock()
	if len(resources.tempDir) == 0 {
		dir, err := ioutil.TempDir("", "cannon")
		if err != nil {
			panic(err)
		}
		resources.tempDir = dir
	}

	// create a temp file to hold the output preview file
	fp, err := ioutil.TempFile(resources.tempDir, "preview")
	defer fp.Close()
	if err != nil {
		panic(err)
	}
	return fp.Name()
}

func Exit() {
	// clean up
	if len(resources.tempDir) > 0 {
		os.RemoveAll(resources.tempDir)
	}
}

func makeHash(s string) string {
	hash := md5.New()
	hash.Write([]byte(s))
	hashstr := hex.EncodeToString(hash.Sum(nil))
	return hashstr
}

func init() {
	// reset the resource map on config file changes
	config.RegisterCallback(reloadCallback)
}

func matchConfigRules(file string) ([]string, string) {
	extension := strings.TrimLeft(path.Ext(file), ".")
	// mime := // TODO: mime type goes here

	cfg := config.GetConfig()
	rules := cfg.FileConversionRules
	for _, rule := range rules {
		if rule.Type == "extension" {
			if util.Find(rule.Matches, extension) < len(rule.Matches) {
				return config.GetPlatformCommand(rule.Command), rule.Tag
			}
		}
		// TODO: add mime type
		// TODO: add generic binary
		// TODO: add generic text
	}

	// no match found
	return []string{}, ""
}

func copy(input string, output string) {
	// copy input file contents to output file
	data, err := ioutil.ReadFile(input)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(output, data, 0644)
	if err != nil {
		panic(err)
	}
}

func getLargestFile(pattern string) string {
	// find the file matching pattern with the largest size
	matches, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}
	largest := ""
	size := int64(0)
	for _, match := range matches {
		fi, err := os.Stat(match)
		if err != nil {
			panic(err)
		}
		if fi.Size() > size {
			largest = match
			size = fi.Size()
		}
	}
	return largest
}

func convertFile(input string, hash string, output string) {
	// run conversion rules on the input file to produce output
	resource, ok := getResource(hash)
	if !ok {
		panic("Resource lookup failed in cache.go!")
	}

	// find the first matching configuration rule
	conversion, tag := matchConfigRules(input)

	// run the matching command and wait for it to complete
	if len(conversion) > 0 {
		resource.combinedOutput += fmt.Sprintf("Config: %v\n\n", conversion)
		command, args := util.FormatCommand(conversion, map[string]string{"{input}": input, "{output}": output})
		resource.combinedOutput += fmt.Sprintf("   Run: %s %s\n\n", command, strings.Trim(fmt.Sprintf("%v", args), "[]"))
		out, err := exec.Command(command, args...).CombinedOutput()
		resource.combinedOutput += string(out)
		if err != nil {
			resource.html = "<pre>" + resource.combinedOutput + "</pre>"
			resource.htmlHash = makeHash(resource.html)
			resource.ready = true
		} else {
			// if the rule creates an output file with extension, copy it over the one without
			largest := getLargestFile(output + "*")
			if largest != output {
				copy(largest, output)
			}

			resource.html = strings.Replace(tag, "{src}", "{document.location.href}file?hash="+hash, 1)
			resource.htmlHash = makeHash(resource.html)
			resource.ready = true
		}
	} else {
		// if the rule doesn't contain a command, copy the input file into the temp folder and serve the copy
		copy(input, output)

		resource.html = strings.Replace(tag, "{src}", "{document.location.href}file?hash="+hash, 1)
		resource.htmlHash = makeHash(resource.html)
		resource.ready = true
	}

	// update the resource
	setResource(hash, resource)
}

func createResource(file string, hash string) {
	// create a new resource for the file if it doesn't already exist
	_, ok := getResource(hash)
	if !ok {
		preview := createPreviewFile()

		// add a new entry for the resource
		setResource(hash, Resource{
			false,
			file,
			hash,
			"",
			preview,
			"",
			"",
		})

		// perform file conversion concurrently to complete the resource
		go convertFile(file, hash, preview)
	}
}

func precacheNearbyFiles(file string) {
	// TODO: need to sort the files to match their display order in lf and others

	precache := config.GetConfig().Settings.Precache
	if precache == 0 {
		return
	}

	// precache the files around the "current" one
	files, err := ioutil.ReadDir(filepath.Dir(file))
	if err != nil {
		panic(err)
	}
	sorted := []string{}
	for _, file := range files {
		if !file.IsDir() {
			sorted = append(sorted, file.Name())
		}
	}

	// find current item
	index := util.Find(sorted, filepath.Base(file))

	// precache files after
	for i, count := index+1, 0; i < len(sorted) && count < precache; i, count = i+1, count+1 {
		after := filepath.Dir(file) + "/" + sorted[i]
		createResource(after, makeHash(after))
	}

	// precache files before
	for i, count := index-1, 0; i >= 0 && count < precache; i, count = i-1, count+1 {
		before := filepath.Dir(file) + "/" + sorted[i]
		createResource(before, makeHash(before))
	}
}

func getCurrentResourceData() map[string]string {
	// return the current resource for display

	// set default values
	data := map[string]string{
		"interval": strconv.Itoa(config.GetConfig().Settings.Interval),
	}

	// look up the current resource if it exists
	resource, ok := getResource(getCurrentHash())
	if !ok {
		// serve default values until the first resource is added
		html := "<p>Waiting for file...</p>"
		maps.Copy(data, map[string]string{
			"title":    "Cannon preview",
			"filehash": "",
			"html":     html,
			"htmlhash": makeHash(html),
		})
	} else {
		if !resource.ready {
			// serve the file conversion's combined stdout+stderr until ready is true
			html := "<p>Loading " + resource.inputName + "...</p>"
			maps.Copy(data, map[string]string{
				"title":    filepath.Base(resource.inputName),
				"filehash": resource.inputNameHash,
				"html":     html,
				"htmlhash": makeHash(html),
			})
		} else {
			// serve the converted output file
			maps.Copy(data, map[string]string{
				"title":    filepath.Base(resource.inputName),
				"filehash": resource.inputNameHash,
				"html":     resource.html,
				"htmlhash": resource.htmlHash,
			})
		}
	}

	return data
}

func Page(w *http.ResponseWriter) {
	// emit html for the current page
	data := getCurrentResourceData()

	// generate page from template
	t, err := template.New("page").Parse(PageTemplate)
	if err != nil {
		panic(err)
	}

	// respond with current page html
	if w != nil {
		t.Execute(*w, data)
	} else {
		t.Execute(os.Stdout, data)
	}
}

func Update(w *http.ResponseWriter, r *http.Request) {
	// select a new file to display

	// extract params from the request body
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		panic(err)
	}

	// set the current file to display
	file := params["file"]
	hash := makeHash(file)
	createResource(file, hash)
	setCurrentHash(hash)

	// precache nearby files
	precacheNearbyFiles(file)

	// respond with { state: updated }
	body := map[string]string{
		"state": "updated",
	}
	util.RespondJson(w, body)
}

func File(w *http.ResponseWriter, r *http.Request) {
	// serve the requested file by hash
	hash := r.URL.Query().Get("hash")
	resource, ok := getResource(hash)
	if !ok {
		panic("Resource lookup failed in cache.go!")
	}
	http.ServeFile(*w, r, resource.outputName)
}

func Status(w *http.ResponseWriter) {
	// respond with current state info
	body := getCurrentResourceData()
	util.RespondJson(w, body)
}
