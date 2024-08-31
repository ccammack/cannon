package cache

// type Resource struct {
// 	ready         bool
// 	inputName     string
// 	inputNameHash string
// 	// stdout         string
// 	// stderr         string
// 	combinedOutput string
// 	outputName     string
// 	html           string
// 	htmlHash       string
// }

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

// func getCreateResource(file string, hash string) Resource {
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
// 	return Resource{}
// }
