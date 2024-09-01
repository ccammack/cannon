package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
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

var cache = struct {
	lock        sync.RWMutex
	currentHash string
	lookup      map[string]*Resource
	tempDir     string
}{lookup: make(map[string]*Resource)}

func reloadCallback(event string) {
	if event == "reload" {
		cache.lock.Lock()
		cache.currentHash = ""
		cache.lookup = make(map[string]*Resource)
		cache.tempDir = ""
		cache.lock.Unlock()
	}
}

func init() {
	// reset the resource map on config file changes
	config.RegisterCallback(reloadCallback)
}

func Exit() {
	// clean up
	cache.lock.Lock()
	if len(cache.tempDir) > 0 {
		os.RemoveAll(cache.tempDir)
	}
	cache.lock.Unlock()
}

type conversionRule struct {
	idx       int
	matchExt  bool
	Ext       []string
	matchMime bool
	Mime      []string
	cmd       []string
	html      string
}

func matchConversionRules(file string) (string, []conversionRule) {
	extension := strings.ToLower(strings.TrimLeft(path.Ext(file), "."))
	mimetype := strings.ToLower(GetMimeType(file))

	matches := []conversionRule{}

	rulesk, rulesv := config.Rules()
	for idx, rule := range rulesv {
		// TODO: add support for glob patterns
		_, exts := rule.Ext.Strings()
		matchExt := len(extension) > 0 && len(exts) > 0 && util.Find(exts, extension) < len(exts)

		// TODO: add support for glob patterns
		_, mimes := rule.Mime.Strings()
		matchMime := len(mimetype) > 0 && len(mimes) > 0 && util.Find(mimes, mimetype) < len(mimes)

		if matchExt || matchMime {
			_, cmd := rule.Cmd.Strings()
			_, html := rule.Html.String()

			matches = append(matches, conversionRule{idx, matchExt, exts, matchMime, mimes, cmd, html})
		}
	}

	return rulesk, matches
}

func generateOutputFilename(input string, entries []string) string {
	for _, entry := range entries {
		if strings.Contains(entry, "{output}") {
			return strings.Replace(entry, "{output}", input, -1)
		}
	}
	return input
}

func runAndWait(resource *Resource, rule conversionRule) int {
	cmd, args := util.FormatCommand(rule.cmd,
		map[string]string{
			"{input}":  resource.input,
			"{output}": resource.output,
		})

	command := exec.Command(cmd, args...)
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	stdoutWriter := io.MultiWriter(os.Stdout, &stdoutBuffer)
	stderrWriter := io.MultiWriter(os.Stderr, &stderrBuffer)
	command.Stdout = stdoutWriter
	command.Stderr = stderrWriter

	exit := 0
	err := command.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exit = exitError.ExitCode()
		}
	}

	resource.stdout = stdoutBuffer.String()
	resource.stderr = stderrBuffer.String()

	return exit
}

type Resource struct {
	input     string // {input}
	inputHash string
	output    string // {output}
	outputExt string // {outputExt}
	html      string
	htmlHash  string
	stdout    string // {stdout}
	stderr    string // {stderr}
}

func serveRaw(resource *Resource, rule conversionRule) bool {
	// display the first part of the raw file
	resource.html = formatRawFileElement(resource.input)
	resource.htmlHash = makeHash(resource.html)

	return true
}

func serveInput(resource *Resource, rule conversionRule) bool {
	if len(rule.html) == 0 {
		return false
	}

	// serve the original input file
	resource.html = strings.ReplaceAll(rule.html, "{url}", "{document.location.href}"+resource.inputHash)
	resource.htmlHash = makeHash(resource.html)

	return true
}

func serveCommand(resource *Resource, rule conversionRule) bool {
	if len(rule.cmd) == 0 {
		return false
	}

	// run the command and wait for it to complete
	exit := runAndWait(resource, rule)
	if exit != 0 {
		return false
	}

	// if the conversion succeeds...

	// generate output filename
	resource.outputExt = generateOutputFilename(resource.output, rule.cmd)

	// replace html placeholders
	html := rule.html
	html = strings.ReplaceAll(html, "{output}", resource.output)
	html = strings.ReplaceAll(html, "{file}", resource.outputExt)
	html = strings.ReplaceAll(html, "{url}", "{document.location.href}"+resource.inputHash)
	html = strings.ReplaceAll(html, "{stdout}", resource.stdout)
	html = strings.ReplaceAll(html, "{stderr}", resource.stderr)

	// replace {content} with the contents of {file}
	b, err := os.ReadFile(resource.outputExt)
	if err == nil {
		log.Printf("error reading content of %s: % v", resource.outputExt, err)
	} else {
		html = strings.ReplaceAll(html, "{content}", string(b))
	}

	// save output html
	resource.html = html
	resource.htmlHash = makeHash(resource.html)

	return true
}

func createPreviewFile() string {
	// create a temp directory on the first call
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if len(cache.tempDir) == 0 {
		dir, err := ioutil.TempDir("", "cannon")
		util.CheckPanicOld(err)
		cache.tempDir = dir
	}

	// create a temp file to hold the output preview file
	fp, err := ioutil.TempFile(cache.tempDir, "preview")
	util.CheckPanicOld(err)
	defer fp.Close()
	return fp.Name()
}

func convert(input string, ch chan *Resource) {
	// TODO: use the file.ToLower() rather than hashing
	inputHash := makeHash(input)

	// find and return the resource if it already exists
	cache.lock.Lock()
	cache.currentHash = inputHash
	resource, ok := cache.lookup[inputHash]
	cache.lock.Unlock()
	if ok {
		ch <- resource
		return
	}

	// create a new resource
	resource = &Resource{input, inputHash, createPreviewFile(), input, "", "", "", ""}

	// find the first matching configuration rule
	_, rules := matchConversionRules(input)
	if len(rules) == 0 {
		// no matching rule found, so display the first part of the raw file
		resource.html = formatRawFileElement(input)
		resource.htmlHash = makeHash(resource.html)

		return
	}

	// apply the first matching rule
	rule := rules[0]
	fmt.Println(rule)

	b := serveCommand(resource, rule) || serveInput(resource, rule) || serveRaw(resource, rule)
	if !b {
		log.Printf("error generating output")
	}

	// time.Sleep(30 * time.Second)
	ch <- resource
}

func formatCurrentResourceData() map[string]template.HTML {
	// set default values
	_, interval := config.Interval().String()
	data := map[string]template.HTML{
		"interval": template.HTML(interval),
	}

	// look up the current resource if it exists
	cache.lock.RLock()
	currentHash := cache.currentHash
	resource, ok := cache.lookup[currentHash]
	cache.lock.RUnlock()

	if ok {
		// serve the converted output file (or error text on failure)
		maps.Copy(data, map[string]template.HTML{
			"title":    template.HTML(filepath.Base(resource.input)),
			"html":     template.HTML(resource.html),
			"htmlhash": template.HTML(resource.htmlHash),
		})
	} else if currentHash != "" {
		// serve a spinner until the file has finished
		// https://codepen.io/nikhil8krishnan/pen/rVoXJa
		maps.Copy(data, map[string]template.HTML{
			"title":    template.HTML(filepath.Base("Loading...")),
			"html":     template.HTML(SpinnerTemplate),
			"htmlhash": template.HTML(makeHash(SpinnerTemplate)),
		})
	} else {
		// serve default values until the first resource is added
		html := "<p>Waiting for file...</p>"
		maps.Copy(data, map[string]template.HTML{
			"title":    "Cannon preview",
			"html":     template.HTML(html),
			"htmlhash": template.HTML(makeHash(html)),
		})
	}

	return data
}

func Update(w *http.ResponseWriter, r *http.Request) {
	// select a new file to display

	// extract params from the request body
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	util.CheckPanicOld(err)

	// set the current input to display
	input := params["file"]

	// convert file to html-native
	ch := make(chan *Resource)
	go func() {
		convert(input, ch)
	}()

	// respond with { state: updated }
	body := map[string]template.HTML{
		"state": "updated",
	}
	util.RespondJson(w, body)

	// wait for convert() to finish
	resource := <-ch
	if resource == nil {
		log.Printf("error converting file: %v", input)
		return
	}
	cache.lock.Lock()
	cache.lookup[resource.inputHash] = resource
	cache.lock.Unlock()
}

func File(w *http.ResponseWriter, r *http.Request) {
	// serve the requested file by hash
	path := strings.ReplaceAll(r.URL.Path, "{document.location.href}", "")
	hash := strings.ReplaceAll(path, "/", "")

	// look up the current resource if it exists
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	resource, ok := cache.lookup[hash]

	if ok {
		// serve the output file with extenstion
		http.ServeFile(*w, r, resource.outputExt)
	} else {
		// server an empty file
		http.ServeFile(*w, r, "")
	}
}

func Page(w *http.ResponseWriter) {
	// emit html for the current page
	data := formatCurrentResourceData()

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

func Status(w *http.ResponseWriter) {
	// respond with current state info
	body := formatCurrentResourceData()
	util.RespondJson(w, body)
}
