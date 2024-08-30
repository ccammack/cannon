package cache

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"
)

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

func init() {
	// reset the resource map on config file changes
	config.RegisterCallback(reloadCallback)
}

func Exit() {
	// clean up
	if len(resources.tempDir) > 0 {
		os.RemoveAll(resources.tempDir)
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

func createResource(file string, hash string) string {
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

		return preview
	}

	return ""
}
