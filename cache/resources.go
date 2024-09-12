package cache

import (
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
	serveFileExt  string // {outputExt}
	html          string
	htmlHash      string
	stdout        string // {stdout}
	stderr        string // {stderr}
	reader        *readseeker.ReadSeeker
	Ready         bool
}

func newResource(file string, hash string, ready func(res *Resource)) *Resource {
	resource := &Resource{
		file:          file,
		hash:          hash,
		tmpOutputFile: createPreviewFile(tempDir),
		serveFileExt:  file,
		reader:        nil,
	}

	go func() {
		// find the first matching configuration rule
		_, rules := matchConversionRules(resource.file)
		if len(rules) == 0 {
			// no matching rule found
			resource.serveRaw()
		} else {
			// apply the first matching rule
			rule := rules[0]
			if !resource.serveInput(rule) && !resource.serveCommand(rule) && !resource.serveRaw() {
				log.Printf("Error serving resource: %v", resource)
			}
		}

		// give it a reader; some converted files will fail because they are still open
		// TODO: figure out how to wait for the output file to be closed before creating the readseeker
		resource.reader = readseeker.New(resource.serveFileExt)

		// work complete
		resource.Ready = true
		ready(resource)
	}()

	return resource
}

func (resource *Resource) Close() {
	if resource.reader != nil {
		resource.reader.Cancel()
	}
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
	resource.htmlHash = util.MakeHash(resource.html)

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
	resource.html = strings.ReplaceAll(rule.html, "{url}", "{document.location.href}"+"file/"+resource.hash)
	resource.htmlHash = util.MakeHash(resource.html)

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
		return false
	}

	// generate output filename
	// TODO: allow the user to define their own vars in config.yml
	resource.serveFileExt = findMatchingOutputFile(resource.tmpOutputFile)

	// replace html placeholders
	html := rule.html
	html = config.ReplaceEnvPlaceholders(html)
	html = config.ReplacePlaceholder(html, "{output}", resource.tmpOutputFile)
	html = config.ReplacePlaceholder(html, "{outputext}", resource.serveFileExt)
	html = config.ReplacePlaceholder(html, "{url}", "{document.location.href}"+"file/"+resource.hash)
	html = config.ReplacePlaceholder(html, "{stdout}", resource.stdout)
	html = config.ReplacePlaceholder(html, "{stderr}", resource.stderr)

	// replace {content} with the contents of the resource.outputExt file
	if strings.Contains(html, "{content}") {
		b, err := os.ReadFile(resource.serveFileExt)
		if err != nil {
			html = config.ReplacePlaceholder(html, "{content}", string(b))
		}
	}

	// save output html
	resource.html = html
	resource.htmlHash = util.MakeHash(resource.html)

	return true
}
