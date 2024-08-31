package cache

import (
	"io/ioutil"

	"github.com/ccammack/cannon/util"
)

type Resource struct {
	ready         bool
	inputName     string
	inputNameHash string
	// stdout         string
	// stderr         string
	combinedOutput string
	outputName     string
	html           string
	htmlHash       string
}

// func getResource(hash string) (Resource, bool) {
// 	cache.lock.Lock()
// 	resource, ok := cache.lookup[hash]
// 	cache.lock.Unlock()
// 	return resource, ok
// }

// func setResource(hash string, resource Resource) {
// 	cache.lock.Lock()
// 	cache.lookup[hash] = resource
// 	cache.lock.Unlock()
// }

// func getCurrentHash() string {
// 	cache.lock.Lock()
// 	defer cache.lock.Unlock()
// 	return cache.current
// }

// func setCurrentHash(hash string) {
// 	cache.lock.Lock()
// 	cache.current = hash
// 	cache.lock.Unlock()
// }

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

// func createResource(file string, hash string) string {
// 	// create a new resource for the file if it doesn't already exist
// 	_, ok := getResource(hash)
// 	if !ok {
// 		preview := createPreviewFile()

// 		// add a new entry for the resource
// 		setResource(hash, Resource{
// 			false,
// 			file,
// 			hash,
// 			"",
// 			preview,
// 			"",
// 			"",
// 		})

// 		return preview
// 	}

// 	return ""
// }

func getCreateResource(file string, hash string) Resource {
	// create a new resource for the file if it doesn't already exist
	// resource, ok := getResource(hash)
	// if !ok {
	// 	preview := createPreviewFile()

	// 	// add a new entry for the resource
	// 	setResource(hash, Resource{
	// 		false,
	// 		file,
	// 		hash,
	// 		"",
	// 		preview,
	// 		"",
	// 		"",
	// 	})

	// 	resource, ok = getResource(hash)
	// 	if !ok {
	// 		log.Panic()
	// 	}
	// }

	// return resource
	return Resource{}
}
