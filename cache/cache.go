package cache

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"
)

var cache = struct {
	lock    sync.RWMutex
	current string
	lookup  map[string]*Resource
	tempDir string
}{lookup: make(map[string]*Resource)}

func reloadCallback(event string) {
	if event == "reload" {
		cache.lock.Lock()
		cache.current = ""
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

///

///

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

type conversionRule struct {
	idx       int
	matchExt  bool
	Ext       []string
	matchMime bool
	Mime      []string
	cmd       []string
	html      string
}

func matchConversionRules2(file string) (string, []conversionRule) {
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

func runAndWait2(input string, output string, match string, command []string) (string, int, error) {
	cmd, args := util.FormatCommand(command, map[string]string{"{input}": input, "{output}": output})
	out, err := exec.Command(cmd, args...).CombinedOutput()
	exit := 0
	if exitError, ok := err.(*exec.ExitError); ok {
		exit = exitError.ExitCode()
	}
	// combined := fmt.Sprintf("  Match: %v\n", match)
	combined := fmt.Sprintf("Command: %v\n", command)
	combined += fmt.Sprintf("    Run: %s %s\n\n", cmd, strings.Trim(fmt.Sprintf("%v", args), "[]"))
	combined += string(out)
	return combined, exit, err
}

func serveInput(key string, resource Resource, input string, output string, hash string, rule conversionRule) bool {
	// if the rule doesn't provide a command, serve the original input file
	if len(rule.html) == 0 {
		// if the rule doesn't provide a tag, display the first part of the raw file
		resource.html = formatRawFileElement(input)
		resource.htmlHash = makeHash(resource.html)
		resource.ready = true
	} else {
		// otherwise, use the provided tag
		resource.outputName = resource.inputName
		resource.html = strings.Replace(rule.html, "{url}", "{document.location.href}"+hash, 1)
		resource.htmlHash = makeHash(resource.html)
		resource.ready = true
	}

	return true
}

func generateOutputFilename(input string, entries []string) string {
	for _, entry := range entries {
		if strings.Contains(entry, "{output}") {
			return strings.Replace(entry, "{output}", input, -1)
		}
	}

	return ""
}

func serveCommand(key string, resource Resource, input string, output string, hash string, rule conversionRule) bool {
	if len(rule.cmd) == 0 {
		return false
	}

	// run the command and wait for it to complete
	combined, exit, err := runAndWait(input, output, "", rule.cmd)
	resource.combinedOutput = combined
	if exit != 0 || err != nil {
		return false
	}

	// generate output filename
	// TODO: complain if not found
	file := generateOutputFilename(output, rule.cmd)

	// if the conversion succeeds...
	html := rule.html
	html = strings.ReplaceAll(html, "{output}", output)
	html = strings.ReplaceAll(html, "{file}", file)
	html = strings.ReplaceAll(html, "{url}", "{document.location.href}"+hash)

	// TODO; support more patterns
	// html = strings.ReplaceAll(html, "{stdout}", stdout)
	// html = strings.ReplaceAll(html, "{stderr}", stderr)
	// html = strings.ReplaceAll(html, "{content}", content)

	resource.html = html
	resource.htmlHash = makeHash(resource.html)
	resource.ready = true

	return true
}

func convertFile2(input string, hash string) {
	// run conversion rules on the input file to produce output
	resource := getCreateResource(input, hash)
	// if !ok {
	// 	panic("Resource lookup failed in cache.go!")
	// }

	output := resource.inputName

	// find the first matching configuration rule
	key, rules := matchConversionRules2(input)
	if len(rules) == 0 {
		// no matching rule found, so display the first part of the raw file
		resource.html = formatRawFileElement(input)
		resource.htmlHash = makeHash(resource.html)
		resource.ready = true
		return
	}

	// apply the first matching rule
	rule := rules[0]

	if serveCommand(key, resource, input, output, hash, rule) ||
		serveInput(key, resource, input, output, hash, rule) {
		// serve_raw(key, idx, resource, input, hash, rule)
		fmt.Println("sucess")
	}

	// update the resource
	// setResource(hash, resource)
}

func convertFile(input string, hash string) {
	// run conversion rules on the input file to produce output
	resource := getCreateResource(input, hash)
	output := resource.outputName

	// find the first matching configuration rule
	match, command, tag, found := matchConfigRules(input)
	if !found {
		// no matching rule found, so display the first part of the raw file
		resource.html = formatRawFileElement(input)
		resource.htmlHash = makeHash(resource.html)
		resource.ready = true
	} else {
		if len(command) == 0 {
			// if the rule doesn't provide a command, serve the original input file
			if len(tag) == 0 {
				// if the rule doesn't provide a tag, display the first part of the raw file
				resource.html = formatRawFileElement(input)
				resource.htmlHash = makeHash(resource.html)
				resource.ready = true
			} else {
				// otherwise, use the provided tag
				resource.outputName = resource.inputName
				resource.html = strings.Replace(tag, "{url}", "{document.location.href}"+hash, 1)
				resource.htmlHash = makeHash(resource.html)
				resource.ready = true
			}
		} else {
			// run the matching command and wait for it to complete
			combined, exit, err := runAndWait(input, output, match, command)
			resource.combinedOutput = combined
			if exit != 0 || err != nil {
				// if the conversion fails, serve the combined stdout+err text from the console
				resource.html = formatRawStringElement(resource.combinedOutput)
				resource.htmlHash = makeHash(resource.html)
				resource.ready = true
			} else {
				hasOutputPlaceholder := util.Find(command, "{output}") < len(command)
				if !hasOutputPlaceholder {
					// if the rule ran but did not provide an {output}, serve the combined stdout+err
					if len(tag) == 0 {
						// if the rule doesn't provide a tag, display the first part of the raw file
						resource.html = formatRawStringElement(resource.combinedOutput)
						resource.htmlHash = makeHash(resource.html)
						resource.ready = true
					} else {
						// otherwise, use the provided tag
						resource.outputName = resource.inputName
						resource.html = strings.Replace(tag, "{url}", resource.combinedOutput, 1)
						resource.htmlHash = makeHash(resource.html)
						resource.ready = true
					}
				} else {
					// if the rule provided an {output}
					if len(tag) == 0 {
						// if the rule doesn't provide a tag, display the first part of the raw file
						resource.html = formatRawFileElement(getFileWithExtension(output))
						resource.htmlHash = makeHash(resource.html)
						resource.ready = true
					} else {
						// if the file conversion succeeds, serve the converted output file
						resource.html = strings.Replace(tag, "{url}", "{document.location.href}"+hash, 1)
						resource.htmlHash = makeHash(resource.html)
						resource.ready = true
					}
				}
			}
		}
	}

	// update the resource
	// setResource(hash, resource)
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

func convert(input string, ch chan *Resource) {
	// TODO: use the file.ToLower() rather than hashing
	hash := makeHash(input)

	cache.lock.Lock()
	cache.current = hash
	resource, ok := cache.lookup[hash]
	cache.lock.Unlock()
	if ok {
		ch <- resource
	}

	time.Sleep(5 * time.Second)
	ch <- nil

	// resource := Resource{
	// 	false,
	// 	file,
	// 	hash,
	// 	"",
	// 	preview,
	// 	"",
	// 	""
	// }
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

	// generate page from template
	// t, err := template.New("page").Parse(PageTemplate)
	// util.CheckPanicOld(err)
	// t.Execute(*w, data)

	// wait for convert()
	resource := <-ch
	if resource == nil {
		log.Printf("error converting file: %v", input)
		return
	}
	cache.lock.Lock()
	cache.lookup[resource.inputNameHash] = resource
	cache.lock.Unlock()

	// res, ok := getResource(hash)
	// fmt.Println(res)
	// fmt.Println(ok)

	// TODO: reuse existing preview files if possible
	// preview := createResource(file, hash)

	// if preview == "" {
	// 	// TODO: handle this case better
	// 	log.Panicf("error trying to create resource")
	// }

	// run conversion rules on the input file to produce output
	// resource, ok := getResource(hash)
	// if !ok {
	// 	panic("Resource lookup failed in cache.go!")
	// }

	// perform file conversion concurrently to complete the resource
	// go convertFile(file, hash)
	// convertFile(file, hash)

	// go convertFile2(file, hash)
	// convertFile2(file, hash)

	// update the resource
	// setResource(hash, resource)
	// setCurrentHash(hash)
}

func File(w *http.ResponseWriter, r *http.Request) {
	// // serve the requested file by hash
	// path := strings.ReplaceAll(r.URL.Path, "{document.location.href}", "")
	// hash := strings.ReplaceAll(path, "/", "")
	// resource, ok := getResource(hash)
	// if !ok {
	// 	// resource is not ready yet
	// 	http.ServeFile(*w, r, "")
	// }

	// // serve the output file with extension if it exists rather than the original placeholder temp file
	// file := resource.outputName
	// if resource.outputName != resource.inputName {
	// 	file = getFileWithExtension(file)
	// }
	// http.ServeFile(*w, r, file)
}

func Status(w *http.ResponseWriter) {
	// respond with current state info
	body := formatCurrentResourceData()
	util.RespondJson(w, body)
}
