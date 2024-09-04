package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ccammack/cannon/cancelread"
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

func GetMimeType(file string) string {
	_, command := config.Mime().Strings()
	if len(command) > 0 {
		cmd, args := util.FormatCommand(command, map[string]string{"{input}": file})
		out, _ := exec.Command(cmd, args...).CombinedOutput()
		return strings.TrimSuffix(string(out), "\n")
	}
	return ""
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

func findMatchingOutputFile(output string) string {
	// find newly created files that match output*
	// TODO: add a vars block to the YAML and remove this function
	matches, err := filepath.Glob(output + "*")
	if err != nil {
		log.Printf("error matching filename %s: %v", output, err)
	}
	if len(matches) > 2 {
		log.Printf("error matched too many files for %s: %v", output, matches)
	}
	for _, match := range matches {
		if len(match) > len(output) {
			output = match
		}
	}
	return output
}

func runAndWait(resource *Resource, rule conversionRule) int {
	cmd, args := util.FormatCommand(rule.cmd, map[string]string{
		"{input}":  resource.input,
		"{output}": resource.output,
	})

	// timeout
	_, timeout := config.Timeout().Int()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	// prepare command
	var outb, errb bytes.Buffer
	command := exec.CommandContext(ctx, cmd, args...)
	command.Stdout = &outb
	command.Stderr = &errb

	// run command
	err := command.Run()
	resource.stdout = outb.String()
	resource.stderr = errb.String()

	// fail if the command takes too long
	if ctx.Err() == context.DeadlineExceeded {
		return 255
	}

	// collect and return exit code
	exit := 0
	if err != nil {
		// there was an error
		exit = 255

		// extract the actual error
		if exitError, ok := err.(*exec.ExitError); ok {
			exit = exitError.ExitCode()
		}
	}

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
	reader    *cancelread.Reader
}

// max display length for unknown file types
const maxLength = 4096

func serveRaw(resource *Resource) bool {
	length, err := util.GetFileLength(resource.input)
	if err != nil {
		log.Printf("error getting length of %s: %v", resource.input, err)
	}

	bytes, count, err := util.GetFileBytes(resource.input, util.Min(maxLength, length))
	if err != nil {
		log.Printf("error reading file %s: %v", resource.input, err)
	}

	if count == 0 {
		log.Printf("error reading empty file %s", resource.input)
	}

	s := string(bytes)
	if length >= maxLength {
		s += "\n\n[...]"
	}

	// display the first part of the raw file
	resource.html = "<xmp>" + s + "</xmp>"
	resource.htmlHash = util.MakeHash(resource.html)

	return true
}

func serveInput(resource *Resource, rule conversionRule) bool {
	// serve the command if available
	if len(rule.cmd) != 0 {
		return false
	}

	// serve raw if missing html
	if len(rule.html) == 0 {
		return false
	}

	// replace placeholders
	resource.html = strings.ReplaceAll(rule.html, "{url}", "{document.location.href}"+resource.inputHash)
	resource.htmlHash = util.MakeHash(resource.html)

	return true
}

func serveCommand(resource *Resource, rule conversionRule) bool {
	// serve raw if missing command
	if len(rule.cmd) == 0 {
		return false
	}

	// run the command and wait
	exit := runAndWait(resource, rule)
	if exit != 0 {
		// serve raw on command failure
		return false
	}

	// generate output filename
	// TODO: allow the user to define their own vars in config.yml
	resource.outputExt = findMatchingOutputFile(resource.output)

	// replace html placeholders
	html := rule.html
	html = strings.ReplaceAll(html, "{output}", resource.output)
	html = strings.ReplaceAll(html, "{outputext}", resource.outputExt)
	html = strings.ReplaceAll(html, "{url}", "{document.location.href}"+resource.inputHash)
	html = strings.ReplaceAll(html, "{stdout}", resource.stdout)
	html = strings.ReplaceAll(html, "{stderr}", resource.stderr)

	// replace {content} with the contents of {outputext}
	b, err := os.ReadFile(resource.outputExt)
	if err != nil {
		html = strings.ReplaceAll(html, "{content}", string(b))
	}

	// save output html
	resource.html = html
	resource.htmlHash = util.MakeHash(resource.html)

	return true
}

func createPreviewFile() string {
	// create a temp directory on the first call
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if len(cache.tempDir) == 0 {
		dir, err := os.MkdirTemp("", "cannon")
		util.CheckPanicOld(err)
		cache.tempDir = dir
	}

	// create a temp file to hold the output preview file
	fp, err := os.CreateTemp(cache.tempDir, "preview")
	util.CheckPanicOld(err)
	defer fp.Close()
	return fp.Name()
}

func convert(input string, ch chan *Resource) {
	// TODO: use the file.ToLower() rather than hashing
	inputHash := util.MakeHash(input)

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
	resource = &Resource{input, inputHash, createPreviewFile(), input, "", "", "", "", nil}

	// find the first matching configuration rule
	_, rules := matchConversionRules(input)
	if len(rules) == 0 {
		// no matching rule found
		serveRaw(resource)
	} else {
		// apply the first matching rule
		rule := rules[0]

		if !serveInput(resource, rule) && !serveCommand(resource, rule) && !serveRaw(resource) {
			log.Printf("error serving resource: %v", resource)
		}
	}

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

func Reset(w *http.ResponseWriter, r *http.Request) {
	// look up the current resource if it exists
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	resource, ok := cache.lookup[cache.currentHash]
	if ok && resource.reader != nil {
		resource.reader.Cancel()
		resource.reader = nil
	}
}

func File(w *http.ResponseWriter, r *http.Request) {
	// serve the requested file by hash
	path := strings.ReplaceAll(r.URL.Path, "{document.location.href}", "")
	hash := strings.ReplaceAll(path, "/", "")

	// look up the current resource if it exists
	// TODO: this smells because it creates a new cache entry for the streaming file
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	resource, ok := cache.lookup[hash]

	if ok {
		// serve the output file with extension
		// http.ServeFile(*w, r, resource.outputExt)

		if resource.reader == nil {
			resource.reader = cancelread.New(resource.outputExt)
		}

		if resource.reader != nil {
			http.ServeContent(*w, r, filepath.Base(resource.reader.Path), resource.reader.Info.ModTime(), resource.reader)
			return
		}
	}

	// serve 404
	http.Error(*w, "Resource Not Found", http.StatusNotFound)
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
