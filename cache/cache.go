/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

// TODO: golang serve file from memory
// TODO: golang lru cache
// https://www.alexedwards.net/blog/golang-response-snippets

package cache

import (
	"cannon/config"
	"cannon/util"
	"crypto/sha1"
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
		resources.lock.RLock()
		defer resources.lock.RUnlock()
		resources.lookup = make(map[string]Resource)
	}
}

func getResourceCount() int {
	resources.lock.RLock()
	defer resources.lock.RUnlock()
	return len(resources.lookup)
}

func getResource(hash string) (Resource, bool) {
	resources.lock.RLock()
	defer resources.lock.RUnlock()
	resource, ok := resources.lookup[hash]
	return resource, ok
}

func setResource(hash string, resource Resource) {
	resources.lock.RLock()
	resources.lookup[hash] = resource
	resources.lock.RUnlock()
}

func setCurrentResource(hash string) {
	resources.lock.RLock()
	resources.current = hash
	resources.lock.RUnlock()
}

func getCurrentResource() Resource {
	resources.lock.RLock()
	resource, ok := resources.lookup[resources.current]
	if !ok {
		panic("Resource lookup failed in cache.go!")
	}
	resources.lock.RUnlock()
	return resource
}

func getTempDir() string {
	resources.lock.RLock()
	defer resources.lock.RUnlock()
	return resources.tempDir
}

func setTempDir(dir string) {
	resources.lock.RLock()
	defer resources.lock.RUnlock()
	resources.tempDir = dir
}

func Exit() {
	if len(resources.tempDir) > 0 {
		os.RemoveAll(resources.tempDir)
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
	// reset the resource map on config file changes
	config.RegisterCallback(reloadCallback)
}

func matchConfigRules(file string) ([]string, string) {
	extension := strings.TrimLeft(path.Ext(file), ".")
	// mime := // TODO: mime type goes here

	config := config.GetConfig()
	rules := config.FileConversionRules
	for _, rule := range rules {
		if rule.Type == "extension" {
			if util.Find(rule.Matches, extension) < len(rule.Matches) {
				return rule.Command, rule.Tag
			}
		}
	}

	// no match found
	return []string{}, ""
}

func copy(input string, output string) {
	// run default copy command
	data, err := ioutil.ReadFile(input)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(output, data, 0644)
	if err != nil {
		panic(err)
	}
}

func convertFile(input string, hash string, stdout string, output string) {

	// if strings.TrimLeft(path.Ext(input), ".") == "pdf" {
	// 	fmt.Println("gotcha")
	// }

	// copy the current resource
	resource, ok := getResource(hash)
	if !ok {
		panic("Resource lookup failed in cache.go!")
	}

	// find matching configuration rule
	commandArr, tag := matchConfigRules(input)

	// run the matching command and wait for it to complete
	if len(commandArr) > 0 {
		resource.combinedOutput += fmt.Sprintf("Config: %v\n\n", commandArr)

		// run the requested command
		command := commandArr[0]
		rest := commandArr[1:]
		args := []string{}
		for _, arg := range rest {
			arg := strings.Replace(arg, "{input}", input, 1)
			arg = strings.Replace(arg, "{output}", output, 1)
			args = append(args, arg)
		}

		resource.combinedOutput += fmt.Sprintf("   Run: %s %s\n\n", command, strings.Trim(fmt.Sprintf("%v", args), "[]"))
		out, err := exec.Command(command, args...).CombinedOutput()
		resource.combinedOutput += string(out)
		if err != nil {
			resource.html = "<pre>" + resource.combinedOutput + "</pre>"
			resource.htmlHash = makeHash(resource.html)
			resource.ready = true
		} else {
			// if the rules create an output file with extension, copy it over the one without
			matches, err := filepath.Glob(output + "*")
			if err != nil {
				panic(err)
			}
			source := ""
			size := int64(0)
			for _, match := range matches {
				fi, err := os.Stat(match)
				if err != nil {
					panic(err)
				}
				if fi.Size() > size {
					source = match
					size = fi.Size()
				}
			}
			if source != output {
				copy(source, output)
			}

			resource.html = strings.Replace(tag, "{src}", "{document.location.href}file?hash="+hash, 1)
			resource.htmlHash = makeHash(resource.html)
			resource.ready = true
		}
	} else {
		copy(input, output)

		resource.html = strings.Replace(tag, "{src}", "{document.location.href}file?hash="+hash, 1)
		resource.htmlHash = makeHash(resource.html)
		resource.ready = true
	}

	// update the resource
	setResource(hash, resource)
}

func createResource(file string, hash string) {
	// create a new resource if a matching one doesn't already exist
	_, ok := getResource(hash)
	if !ok {
		// create a temp directory the first time someone needs it
		if len(getTempDir()) == 0 {
			dir, err := ioutil.TempDir("", "cannon")
			if err != nil {
				panic(err)
			}
			setTempDir(dir)
		}

		// create a temp file to hold stdout for the file conversion
		stdoutFilePtr, err := ioutil.TempFile(getTempDir(), "stdout")
		if err != nil {
			panic(err)
		}
		defer stdoutFilePtr.Close()

		// create a temp file to hold the final output file
		prevewFilePtr, err := ioutil.TempFile(getTempDir(), "preview")
		if err != nil {
			panic(err)
		}
		defer prevewFilePtr.Close()

		// add a new entry for the resource
		setResource(hash, Resource{
			false,
			file,
			hash,
			"",
			prevewFilePtr.Name(),
			"",
			"",
		})

		// perform file conversion concurrently to complete the resource
		go convertFile(file, hash, stdoutFilePtr.Name(), prevewFilePtr.Name())
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

	if getResourceCount() == 0 {
		// serve default values until the first resource is added
		html := "<p>Waiting for file...</p>"
		maps.Copy(data, map[string]string{
			"title":    "Cannon preview",
			"filehash": "0",
			"html":     html,
			"htmlhash": makeHash(html),
		})
	} else {
		resource := getCurrentResource()

		if !resource.ready {
			// serve the file conversion's stdout+stderr until ready is true
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
	// update the current file to display

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
	setCurrentResource(hash)

	// precache nearby files
	precacheNearbyFiles(params["file"])

	// respond with { state: updated }
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
	resource, ok := getResource(hash)
	if !ok {
		panic("Resource lookup failed in cache.go!")
	}
	http.ServeFile(*w, r, resource.outputName)
}

func Status(w *http.ResponseWriter) {
	// respond with current state info
	body := getCurrentResourceData()

	if w != nil {
		(*w).Header().Set("Content-Type", "application/json")
		(*w).WriteHeader(http.StatusOK)
		json.NewEncoder(*w).Encode(body)
	} else {
		json.NewEncoder(os.Stdout).Encode(body)
	}
}
