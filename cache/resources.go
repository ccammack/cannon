package cache

import (
	"log"
	"os"
	"strings"

	"github.com/ccammack/cannon/cancelread"
	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"
)

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
	mime      string
	stream    bool
}

func newResource(file string, hash string) *Resource {
	mime := GetMimeType(file)
	stream := strings.HasPrefix(mime, "audio/") || strings.HasPrefix(mime, "video/")
	resource := &Resource{file, hash, createPreviewFile(), file, "", "", "", "", nil, mime, stream}
	return resource
}

func addReader(resource *Resource) {
	// serving streams requires a reader; nothing else should have one
	if resource.stream {
		resource.reader = cancelread.New(resource.outputExt)
	} else if resource.reader != nil {
		resource.reader.Cancel()
		resource.reader = nil
	}
}

func serveRaw(resource *Resource) bool {
	// max display length for unknown file types
	const maxLength = 4096

	// TODO: consider serving binary files by length and text files by line count
	// right now, a really wide csv might only display the first line
	// and a really narrow csv will display too many lines
	// add a maxLines config value or calculate it from maxLength
	// automatically wrap binary files to fit the browser
	// maybe use the curernt size of the browser window to calculate maxLines

	length, err := util.GetFileLength(resource.input)
	if err != nil {
		log.Printf("Error getting length of %s: %v", resource.input, err)
	}

	bytes, count, err := util.GetFileBytes(resource.input, util.Min(maxLength, length))
	if err != nil {
		log.Printf("Error reading file %s: %v", resource.input, err)
	}

	if count == 0 {
		log.Printf("Error reading empty file %s", resource.input)
	}

	s := string(bytes)
	if length >= maxLength {
		s += "\n\n[...]"
	}

	// display the first part of the raw file
	resource.html = "<xmp>" + s + "</xmp>"
	resource.htmlHash = util.MakeHash(resource.html)
	addReader(resource)

	return true
}

func serveInput(resource *Resource, rule ConversionRule) bool {
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
	addReader(resource)

	return true
}

func serveCommand(resource *Resource, rule ConversionRule) bool {
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
	html = config.ReplaceEnvPlaceholders(html)
	html = config.ReplacePlaceholder(html, "{output}", resource.output)
	html = config.ReplacePlaceholder(html, "{outputext}", resource.outputExt)
	html = config.ReplacePlaceholder(html, "{url}", "{document.location.href}"+"file/"+resource.inputHash)
	html = config.ReplacePlaceholder(html, "{stdout}", resource.stdout)
	html = config.ReplacePlaceholder(html, "{stderr}", resource.stderr)

	// replace {content} with the contents of the resource.outputExt file
	if strings.Contains(html, "{content}") {
		b, err := os.ReadFile(resource.outputExt)
		if err != nil {
			html = config.ReplacePlaceholder(html, "{content}", string(b))
		}
	}

	// save output html
	resource.html = html
	resource.htmlHash = util.MakeHash(resource.html)
	addReader(resource)

	return true
}
