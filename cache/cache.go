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

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"
)

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
	setResource(hash, resource)
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

func Update(w *http.ResponseWriter, r *http.Request) {
	// select a new file to display

	// extract params from the request body
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	util.CheckPanicOld(err)

	// set the current file to display
	file := params["file"]
	hash := makeHash(file)
	preview := createResource(file, hash)
	if preview == "" {
		// TODO: handle this case better
		log.Panicf("error trying to create resource")
	}

	// perform file conversion concurrently to complete the resource
	go convertFile(file, hash, preview)

	setCurrentHash(hash)

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
	body := formatCurrentResourceData()
	util.RespondJson(w, body)
}
