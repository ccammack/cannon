package cache

import (
	"log"
	"os"
	"strings"
	"sync"

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
	ready         bool
}

var resourceManager = struct {
	lock    sync.RWMutex
	tempDir string
	cache   map[string]*Resource
	current *Resource
}{cache: make(map[string]*Resource)}

func init() {
	resourceManager.lock.Lock()
	defer resourceManager.lock.Unlock()
	resourceManager.tempDir = util.CreateTempDir("cannon")

	// react to config file changes
	config.RegisterCallback(func(event string) {
		if event == "reload" {
			closeAll()
		}

		resourceManager.tempDir = util.CreateTempDir("cannon")
	})
}

func setCurrentResource(file string, hash string, ch chan *Resource) {
	resourceManager.lock.Lock()
	defer resourceManager.lock.Unlock()

	res, ok := resourceManager.cache[hash]
	if ok {
		resourceManager.current = res
		go func() {
			ch <- res
		}()
	} else {
		res := &Resource{
			file:          file,
			hash:          hash,
			tmpOutputFile: createPreviewFile(resourceManager.tempDir),
			srcFile:       file,
		}
		resourceManager.cache[hash] = res
		resourceManager.current = res

		go func() {
			// find the first matching configuration rule
			_, rules := matchConversionRules(res.file)
			if len(rules) == 0 {
				// no matching rule found
				res.serveRaw()
			} else {
				// apply the first matching rule
				rule := rules[0]
				if !res.serveInput(rule) && !res.serveCommand(rule) && !res.serveRaw() {
					log.Printf("Error serving resource: %v", res)
				}
			}

			// give it a reader; some converted files will fail because they are still open
			// TODO: figure out how to wait for the output file to be closed before creating the readseeker
			res.reader = readseeker.New(res.srcFile)

			// work complete
			res.ready = true
			ch <- res
		}()
	}
}

func close(hash string) {
	resourceManager.lock.Lock()
	defer resourceManager.lock.Unlock()
	res, ok := resourceManager.cache[hash]
	if ok {
		// close the cached resource
		if res.reader != nil {
			res.reader.Cancel()
		}
		delete(resourceManager.cache, hash)
	}

	// also nil the current resource if it matches
	res = resourceManager.current
	if res != nil && res.hash == hash {
		resourceManager.current = nil
	}
}

func closeAll() {
	for hash := range resourceManager.cache {
		close(hash)
	}

	if len(resourceManager.cache) != 0 {
		log.Println("error in resources.closeAll()")
	}

	// delete temp files
	if len(resourceManager.tempDir) > 0 {
		os.RemoveAll(resourceManager.tempDir)
	}
}

func currResource() (*Resource, bool) {
	// return the current resource if it exists and is ready for display
	resourceManager.lock.Lock()
	defer resourceManager.lock.Unlock()
	res := resourceManager.current
	if res != nil && res.ready {
		return res, true
	}
	return nil, false
}

func currReader() (*readseeker.ReadSeeker, bool) {
	// return the current reader if it exists and is ready for reading
	res, ok := currResource()
	if ok && res.reader != nil {
		return res.reader, true
	}
	return nil, false
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
	html = config.ReplacePlaceholder(html, "{url}", "{document.location.href}"+"file/"+resource.hash)
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

	return true
}
