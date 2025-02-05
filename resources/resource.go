package resources

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/readseeker"
	"github.com/ccammack/cannon/util"
)

type Resource struct {
	file          string // {input}
	hash          string
	tmpOutputFile string // {output}
	srcFile       string // serve this file for html src attributes
	html          string
	stdout        string // {stdout}
	stderr        string // {stderr}
	reader        *readseeker.ReadSeeker
	progress      []string
}

func NewResource(tempDir string, file string, hash string) *Resource {
	return &Resource{
		file:          file,
		hash:          hash,
		tmpOutputFile: createPreviewFile(tempDir),
		srcFile:       file,
	}
}

func (res *Resource) Open() {
	// find the first matching configuration rule
	_, rules := matchConversionRules(res)
	if len(rules) == 0 {
		// no matching rule found
		res.progress = append(res.progress, "No matching rules found")
		res.serveRaw()
	} else {
		// apply the first matching rule
		rule := rules[0]
		res.progress = append(res.progress, fmt.Sprintf("Apply rule[%d]: %v", rule.idx, rule))

		if !res.serveInput(rule) && !res.serveCommand(rule) && !res.serveRaw() {
			log.Printf("Error serving resource: %v", res)
			res.progress = append(res.progress, fmt.Sprintf("Error serving resource: %v", res))
		}
	}

	// give it a reader; some converted files will fail because they are still open
	// TODO: figure out how to wait for the output file to be closed before creating the readseeker
	res.reader = readseeker.New(res.srcFile)

	// log progress
	for _, line := range res.progress {
		log.Println(line)
	}

	// work complete
	// time.Sleep(5000 * time.Millisecond)
}

func (res *Resource) Close() {
	// cancel reader
	if res.reader != nil {
		res.reader.Cancel()
	}
}

func summarize(line string) string {
	length := 80
	half := int(float64((length - 1) / 2))
	v := []rune(strings.ReplaceAll(line, "\n", ""))
	if length >= len(v) {
		return line
	}
	output := string(v[:half]) + " [...] " + string(v[len(v)-half:])
	return output
}

func (resource *Resource) serveRaw() bool {
	// max display length for unknown file types
	const maxLength = 4096

	// TODO: consider serving binary files by length and text files by line count
	// right now, a really wide csv might only display the first line
	// and a really narrow csv will display too many lines
	// add a maxLines config value or calculate it from maxLength
	// automatically wrap binary files to fit the browser
	// maybe use the curernt size of the browser window to calculate maxLines

	length, err := util.GetFileLength(resource.file)
	if err != nil {
		log.Printf("Error getting length of %s: %v", resource.file, err)
	}

	bytes, count, err := util.GetFileBytes(resource.file, util.Min(maxLength, length))
	if err != nil {
		log.Printf("Error reading file %s: %v", resource.file, err)
	}

	if count == 0 {
		log.Printf("Error reading empty file %s", resource.file)
	}

	s := string(bytes)
	if length >= maxLength {
		s += "\n\n[...]"
	}

	// display the first part of the raw file
	resource.html = "<xmp>" + s + "</xmp>"
	resource.progress = append(resource.progress, fmt.Sprintf("Serve raw: %s", summarize(resource.html)))

	return true
}

func (resource *Resource) serveInput(rule ConversionRule) bool {
	// serve the command if available
	if len(rule.cmd) != 0 {
		return false
	}

	// serve raw if missing html
	if len(rule.html) == 0 {
		return false
	}

	// replace placeholders
	resource.html = strings.ReplaceAll(rule.html, "{url}", "/src/"+resource.hash)
	resource.progress = append(resource.progress, fmt.Sprintf("Serve selected: %s", summarize(resource.html)))

	return true
}

func (resource *Resource) serveCommand(rule ConversionRule) bool {
	// serve raw if missing command
	if len(rule.cmd) == 0 {
		return false
	}

	// run the command and wait
	exit := runAndWait(resource, rule)
	if exit != 0 {
		// serve raw on command failure
		resource.progress = append(resource.progress, fmt.Sprintf("Command failed with status code: %d", exit))
		return false
	}

	// use the *src: value provided or guess the output file by matching the wildcard "{output}*"
	if rule.src != "" {
		resource.srcFile = config.ReplacePlaceholder(rule.src, "{output}", resource.tmpOutputFile)
	} else {
		resource.srcFile = findMatchingOutputFile(resource.tmpOutputFile)
	}

	// replace html placeholders
	html := rule.html
	html = config.ReplaceEnvPlaceholders(html)
	html = config.ReplacePlaceholder(html, "{output}", resource.tmpOutputFile)
	html = config.ReplacePlaceholder(html, "{url}", "/src/"+resource.hash)
	html = config.ReplacePlaceholder(html, "{stdout}", resource.stdout)
	html = config.ReplacePlaceholder(html, "{stderr}", resource.stderr)

	// replace {content} with the contents of the resource.outputExt file
	if strings.Contains(html, "{content}") {
		b, err := os.ReadFile(resource.srcFile)
		if err != nil {
			html = config.ReplacePlaceholder(html, "{content}", string(b))
		}
	}

	// save output html
	resource.html = html
	resource.progress = append(resource.progress, fmt.Sprintf("Serve output: %s", summarize(resource.html)))
	return true
}
