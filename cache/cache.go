package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	tempDir string

	currHash string
	currRes  *Resource

	lock   sync.RWMutex
	lookup map[string]*Resource
}{lookup: make(map[string]*Resource)}

func reloadCallback(event string) {
	if event == "reload" {
		cache.currRes = nil
		cache.tempDir = ""
		for _, v := range cache.lookup {
			if v.reader != nil {
				v.reader.Cancel()
				v.reader = nil
			}
		}
	}
}

func init() {
	// reset the resource map on config file changes
	config.RegisterCallback(reloadCallback)
}

func Exit() {
	// clean up temp files
	if len(cache.tempDir) > 0 {
		os.RemoveAll(cache.tempDir)
	}
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
	resource.reader = cancelread.New(resource.outputExt)

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
	resource.html = strings.ReplaceAll(rule.html, "{url}", "{document.location.href}"+"file/"+resource.inputHash)
	resource.htmlHash = util.MakeHash(resource.html)
	resource.reader = cancelread.New(resource.outputExt)

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
	html = strings.ReplaceAll(html, "{url}", "{document.location.href}"+"file/"+resource.inputHash)
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
	resource.reader = cancelread.New(resource.outputExt)

	return true
}

func createPreviewFile() string {
	// create a temp directory on the first call
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

func convert(input string, inputHash string, ch chan *Resource) {
	resource := &Resource{input, inputHash, createPreviewFile(), input, "", "", "", "", nil}

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

func FormatCurrentResourceData() map[string]template.HTML {
	// set default values
	_, interval := config.Interval().String()
	data := map[string]template.HTML{
		"interval": template.HTML(interval),
	}

	if cache.currRes != nil {
		// serve the converted output file (or error text on failure)
		maps.Copy(data, map[string]template.HTML{
			"title":    template.HTML(filepath.Base(cache.currRes.input)),
			"html":     template.HTML(cache.currRes.html),
			"htmlhash": template.HTML(cache.currRes.htmlHash),
		})
	} else if len(cache.lookup) > 0 {
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

func updateCurrentHash(hash string) {
	// update currHash to be the selected item's hash
	// update curRes if the resource already exists
	// currRes will remain nil until the resource exists
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if hash != cache.currHash {
		cache.currHash = hash
		resource, ok := cache.lookup[cache.currHash]
		if ok {
			cache.currRes = resource
		} else {
			cache.currRes = nil
		}
	}
}

func updateCurrentResource(file string, hash string) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if cache.currRes == nil {
		// create a new resource
		ch := make(chan *Resource)
		go func() {
			convert(file, hash, ch)
		}()

		// wait for convert() to finish
		go func() {
			resource := <-ch
			if resource == nil {
				log.Printf("error converting file: %v", file)
			}

			cache.lock.Lock()
			defer cache.lock.Unlock()

			// sanity check that the resource is not already in the lookup
			if _, ok := cache.lookup[cache.currHash]; ok {
				log.Printf("error creating resource: %v", file)
			}

			// store the new resource in the lookup for next time
			cache.lookup[cache.currHash] = resource

			// update currRes
			cache.currRes = resource
		}()
	}
}

func Update(w *http.ResponseWriter, r *http.Request) {
	// select a new file to display
	body := map[string]template.HTML{}

	// extract params from the request body
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	util.CheckPanicOld(err)

	// TODO: consider using file.ToLower() as the key rather than hashing
	file := params["file"]
	hash := params["hash"]

	if file == "" || hash == "" {
		body["status"] = template.HTML("error")
		body["message"] = template.HTML(fmt.Sprintf("error reading file or hash: %s %s", file, hash))
	} else {
		body["status"] = template.HTML("success")

		// switch to the new item
		updateCurrentHash(hash)

		// switch to the new resource
		updateCurrentResource(file, hash)
	}

	// respond
	util.RespondJson(w, body)
}

func Reset(w *http.ResponseWriter, r *http.Request) {
	// extract params from the request body
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	util.CheckPanicOld(err)

	// set the current input to display
	hash := params["hash"]

	cache.lock.Lock()
	defer cache.lock.Unlock()
	body := map[string]template.HTML{}
	resource, ok := cache.lookup[hash]
	if ok && resource.reader != nil {
		resource.reader.Cancel()
		resource.reader = nil
		delete(cache.lookup, hash)
		if cache.currRes == resource {
			cache.currRes = nil
			cache.currHash = ""
		}
		body["status"] = template.HTML("success")
	} else {
		body["status"] = template.HTML("error")
		body["message"] = template.HTML("error finding reader")
	}
	util.RespondJson(w, body)
}

func File(w *http.ResponseWriter, r *http.Request) {
	// serve the requested file by hash
	s := strings.Replace(r.URL.Path, "{document.location.href}", "", 1)
	hash := strings.Replace(s, "/file/", "", 1)

	cache.lock.Lock()
	defer cache.lock.Unlock()
	resource, ok := cache.lookup[hash]
	if !ok || resource == nil || resource.reader == nil {
		// log.Printf("404 from read: %v", r.URL.Path)

		// serve 404
		http.Error(*w, "Resource Not Found", http.StatusNotFound)
	} else {
		// log.Printf("read is serving: %v", r.URL.Path)
		// http.ServeFile(*w, r, resource.outputExt)
		http.ServeContent(*w, r, filepath.Base(resource.reader.Path), resource.reader.Info.ModTime(), resource.reader)
	}
}
