package cache

import (
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
	"strings"
	"sync"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"

	"golang.org/x/exp/maps"
)

const SpinnerTemplate = `
	<svg version="1.1" id="spinner" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px"
		viewBox="0 0 100 100" enable-background="new 0 0 0 0" xml:space="preserve">
		<path fill="#888" d="M73,50c0-12.7-10.3-23-23-23S27,37.3,27,50 M30.9,50c0-10.5,8.5-19.1,19.1-19.1S69.1,39.5,69.1,50">
			<animateTransform
				attributeName="transform"
				attributeType="XML"
				type="rotate"
				dur="1s"
				from="0 50 50"
				to="360 50 50"
				repeatCount="indefinite" />
		</path>
	</svg>
`

const PageTemplate = `
<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8" />
		<!-- https://stackoverflow.com/a/62438464 - https://heroicons.com/ - https://fffuel.co/eeencode/ -->
		<link rel="icon" href="data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIGZpbGw9Im5vbmUiIHZpZXdCb3g9IjAgMCAyNCAyNCIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZT0iY3VycmVudENvbG9yIiBjbGFzcz0idy02IGgtNiI+PHBhdGggc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIiBkPSJNMi4yNSAxMi43NVYxMkEyLjI1IDIuMjUgMCAwMTQuNSA5Ljc1aDE1QTIuMjUgMi4yNSAwIDAxMjEuNzUgMTJ2Ljc1bS04LjY5LTYuNDRsLTIuMTItMi4xMmExLjUgMS41IDAgMDAtMS4wNjEtLjQ0SDQuNUEyLjI1IDIuMjUgMCAwMDIuMjUgNnYxMmEyLjI1IDIuMjUgMCAwMDIuMjUgMi4yNWgxNUEyLjI1IDIuMjUgMCAwMDIxLjc1IDE4VjlhMi4yNSAyLjI1IDAgMDAtMi4yNS0yLjI1aC01LjM3OWExLjUgMS41IDAgMDEtMS4wNi0uNDR6IiAvPjwvc3ZnPg==" type="image/svg+xml" />
		<title>
			{{.title}}
		</title>
		<style>
			div {
				width:95vw;
				height:95vh;
			}
			img {
				max-width: 100%;
				height:auto;
				max-height: 100%;
			}
			video {
				max-width: 100%;
				height: auto;
				max-height: 100%;
			}
			iframe {
				position: absolute;
				top: 0;
				left: 0;
				width: 100%;
				height: 100%;
				border: 0;
			}
			object {
				max-width: 100%;
				height: auto;
				max-height: 100%;
			}
			#spinner {
				width: 40px;
				height: 40px;
				margin: 20px;
				display:inline-block;
			}
		</style>
		<script>
			let htmlhash = "{{.htmlhash}}";
			window.onload = function(e) {
				// replace placeholder media address with document.location.href
				const container = document.getElementById("container");
				if (container) {
					const inner = container.innerHTML.replace("{document.location.href}", document.location.href);
					container.innerHTML = inner;
				}

				// poll server /status using address from document.location.href
				const statusurl = document.location.href + "status";
				setTimeout(function status() {
					// ask the server for updates and reload if needed
					fetch(statusurl)
					.then((response) => response.json())
					.then((data) => {
						if (htmlhash != data.htmlhash) {
							if (data.html.includes("<video") || data.html.includes("<audio")) {
								// to ensure proper cleanup, reload the page if the incoming element is a video or audio
								location.reload()
							} else {
								// otherwise, just replace the content for faster response
								htmlhash = data.htmlhash
								document.title = data.title

								// replace placeholder media address with document.location.href
								const container = document.getElementById("container");
								if (container) {
									const inner = data.html.replace("{document.location.href}", document.location.href);
									container.innerHTML = inner;
								}
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
		<div id="container">{{.html}}</div>
	</body>
</html>
`

// max display length for unknown file types
const byteLength = 4096

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
		util.CheckPanicOld(err)
		resources.tempDir = dir
	}

	// create a temp file to hold the output preview file
	fp, err := ioutil.TempFile(resources.tempDir, "preview")
	util.CheckPanicOld(err)
	defer fp.Close()
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

func GetMimeType(file string) string {
	_, command := config.Mime().Strings()
	if len(command) > 0 {
		cmd, args := util.FormatCommand(command, map[string]string{"{input}": file})
		out, _ := exec.Command(cmd, args...).CombinedOutput()
		return strings.TrimSuffix(string(out), "\n")
	}
	return ""
}

func isBinaryFile(file string) ([]byte, int, bool) {
	// treat the file as binary if it contains a NUL anywhere in the first byteLength bytes
	fp, err := os.Open(file)
	util.CheckPanicOld(err)
	fs, err := fp.Stat()
	util.CheckPanicOld(err)
	b := make([]byte, util.Min(byteLength, fs.Size()))
	n, err := fp.Read(b)
	util.CheckPanicOld(err)
	for i := 0; i < n; i++ {
		if b[i] == '\x00' {
			return b, int(fs.Size()), true
		}
	}
	return b, int(fs.Size()), false
}

func matchConfigRules(file string) (string, []string, string, bool) {
	// TODO: this should return an array of matches and so the caller can run all of them until something succeeds

	extension := strings.ToLower(strings.TrimLeft(path.Ext(file), "."))
	mimetype := strings.ToLower(GetMimeType(file))
	_, rules := config.Rules()

	for _, rule := range rules {
		match := ""
		_, exts := rule.Ext.Strings()
		_, mimes := rule.Mime.Strings()
		if len(extension) > 0 && len(exts) > 0 && util.Find(exts, extension) < len(exts) {
			match = fmt.Sprintf("ext: %v", exts)
		} else if len(mimetype) > 0 && len(mimes) > 0 && util.Find(mimes, mimetype) < len(mimes) {
			match = fmt.Sprintf("mime: %v", mimes)
		}
		if len(match) > 80 {
			match = match[:util.Min(len(match), 80)] + "...]"
		}
		if match != "" {
			_, cmds := rule.Cmd.Strings()
			_, html := rule.Html.String()
			return match, cmds, html, true
		}
	}

	return "", []string{}, "", false
}

func getFileWithExtension(file string) string {
	// find the file matching pattern with the longest name
	matches, err := filepath.Glob(file + "*")
	util.CheckPanicOld(err)
	for _, match := range matches {
		if len(match) > len(file) {
			file = match
		}
	}
	return file
}

func emitRawFileElement(file string) string {
	bytes, size, _ := isBinaryFile(file)
	s := string(bytes)
	html := "<xmp>" + s + "\n\n"
	if size > byteLength {
		html += "[...]"
	}
	html += "</xmp>"
	return html
}

func emitRawStringElement(raw string) string {
	size := len(raw)
	bytes := raw[0:util.Min(size, byteLength)]
	s := string(bytes)
	html := "<xmp>" + s + "\n\n"
	if size > byteLength {
		html += "[...]"
	}
	html += "</xmp>"
	return html
}

func runAndWait(input string, output string, match string, command []string) (string, int, error) {
	cmd, args := util.FormatCommand(command, map[string]string{"{input}": input, "{output}": output})
	out, err := exec.Command(cmd, args...).CombinedOutput()
	exit := 0
	if exitError, ok := err.(*exec.ExitError); ok {
		exit = exitError.ExitCode()
	}
	combined := fmt.Sprintf("  Match: %v\n", match)
	combined += fmt.Sprintf("Command: %v\n", command)
	combined += fmt.Sprintf("    Run: %s %s\n\n", cmd, strings.Trim(fmt.Sprintf("%v", args), "[]"))
	combined += string(out)
	return combined, exit, err
}

func convertFile(input string, hash string, output string) {
	// run conversion rules on the input file to produce output
	resource, ok := getResource(hash)
	if !ok {
		panic("Resource lookup failed in cache.go!")
	}

	// find the first matching configuration rule
	match, command, tag, found := matchConfigRules(input)
	if !found {
		// no matching rule found, so display the first part of the raw file
		resource.html = emitRawFileElement(input)
		resource.htmlHash = makeHash(resource.html)
		resource.ready = true
	} else {
		if len(command) == 0 {
			// if the rule doesn't provide a command, serve the original input file
			if len(tag) == 0 {
				// if the rule doesn't provide a tag, display the first part of the raw file
				resource.html = emitRawFileElement(input)
				resource.htmlHash = makeHash(resource.html)
				resource.ready = true
			} else {
				// otherwise, use the provided tag
				resource.outputName = resource.inputName
				resource.html = strings.Replace(tag, "{src}", "{document.location.href}"+hash, 1)
				resource.htmlHash = makeHash(resource.html)
				resource.ready = true
			}
		} else {
			// run the matching command and wait for it to complete
			combined, exit, err := runAndWait(input, output, match, command)
			resource.combinedOutput = combined
			if exit != 0 || err != nil {
				// if the conversion fails, serve the combined stdout+err text from the console
				resource.html = emitRawStringElement(resource.combinedOutput)
				resource.htmlHash = makeHash(resource.html)
				resource.ready = true
			} else {
				hasOutputPlaceholder := util.Find(command, "{output}") < len(command)
				if !hasOutputPlaceholder {
					// if the rule ran but did not provide an {output}, serve the combined stdout+err
					if len(tag) == 0 {
						// if the rule doesn't provide a tag, display the first part of the raw file
						resource.html = emitRawStringElement(resource.combinedOutput)
						resource.htmlHash = makeHash(resource.html)
						resource.ready = true
					} else {
						// otherwise, use the provided tag
						resource.outputName = resource.inputName
						resource.html = strings.Replace(tag, "{src}", resource.combinedOutput, 1)
						resource.htmlHash = makeHash(resource.html)
						resource.ready = true
					}
				} else {
					// if the rule provided an {output}
					if len(tag) == 0 {
						// if the rule doesn't provide a tag, display the first part of the raw file
						resource.html = emitRawFileElement(getFileWithExtension(output))
						resource.htmlHash = makeHash(resource.html)
						resource.ready = true
					} else {
						// if the file conversion succeeds, serve the converted output file
						resource.html = strings.Replace(tag, "{src}", "{document.location.href}"+hash, 1)
						resource.htmlHash = makeHash(resource.html)
						resource.ready = true
					}
				}
			}
		}
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

	precache := 0 // config.GetConfig().Settings.Precache
	if precache == 0 {
		return
	}

	// precache the files around the "current" one
	files, err := ioutil.ReadDir(filepath.Dir(file))
	util.CheckPanicOld(err)
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

func getCurrentResourceData() map[string]template.HTML {
	// return the current resource for display

	// set default values
	_, interval := config.Interval().String()

	data := map[string]template.HTML{
		"interval": template.HTML(interval),
	}

	// look up the current resource if it exists
	resource, ok := getResource(getCurrentHash())
	if !ok {
		// serve default values until the first resource is added
		html := "<p>Waiting for file...</p>"
		maps.Copy(data, map[string]template.HTML{
			"title":    "Cannon preview",
			"html":     template.HTML(html),
			"htmlhash": template.HTML(makeHash(html)),
		})
	} else {
		if !resource.ready {
			// serve a spinner until ready is true - https://codepen.io/nikhil8krishnan/pen/rVoXJa
			maps.Copy(data, map[string]template.HTML{
				"title":    template.HTML(filepath.Base(resource.inputName)),
				"html":     template.HTML(SpinnerTemplate),
				"htmlhash": template.HTML(makeHash(SpinnerTemplate)),
			})
		} else {
			// serve the converted output file (or error text on failure)
			maps.Copy(data, map[string]template.HTML{
				"title":    template.HTML(filepath.Base(resource.inputName)),
				"html":     template.HTML(resource.html),
				"htmlhash": template.HTML(resource.htmlHash),
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
	util.CheckPanicOld(err)

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
	util.CheckPanicOld(err)

	// set the current file to display
	file := params["file"]
	hash := makeHash(file)
	createResource(file, hash)
	setCurrentHash(hash)

	// precache nearby files
	precacheNearbyFiles(file)

	// respond with { state: updated }
	body := map[string]template.HTML{
		"state": "updated",
	}
	util.RespondJson(w, body)
}

func File(w *http.ResponseWriter, r *http.Request) {
	// serve the requested file by hash
	path := strings.ReplaceAll(r.URL.Path, "{document.location.href}", "")
	hash := strings.ReplaceAll(path, "/", "")
	resource, ok := getResource(hash)
	if !ok {
		// resource is not ready yet
		http.ServeFile(*w, r, "")
	}

	// serve the output file with extension if it exists rather than the original placeholder temp file
	file := resource.outputName
	if resource.outputName != resource.inputName {
		file = getFileWithExtension(file)
	}
	http.ServeFile(*w, r, file)
}

func Status(w *http.ResponseWriter) {
	// respond with current state info
	body := getCurrentResourceData()
	util.RespondJson(w, body)
}
