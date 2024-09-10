package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"
	"golang.org/x/exp/maps"
)

type Cache struct {
	lock     sync.RWMutex
	tempDir  string
	currHash string
	currRes  *Resource
	lookup   map[string]*Resource
}

var cache = Cache{lookup: make(map[string]*Resource)}

func init() {
	// reset the resource map on config file changes
	config.RegisterCallback(func(event string) {
		if event == "reload" {
			// clean the cache
			cache.lock.Lock()
			cache.tempDir = ""
			cache.currHash = ""
			cache.currRes = nil
			for _, v := range cache.lookup {
				if v.reader != nil {
					v.reader.Cancel()
					v.reader = nil
				}
			}
			cache.lock.Unlock()

			// replace the cache
			cache = Cache{lookup: make(map[string]*Resource)}
		}
	})
}

func Exit() {
	// clean up temp files
	if len(cache.tempDir) > 0 {
		os.RemoveAll(cache.tempDir)
	}
}

func convert(file string, hash string, ch chan *Resource) {
	resource := newResource(file, hash)

	// find the first matching configuration rule
	_, rules := matchConversionRules(file)
	if len(rules) == 0 {
		// no matching rule found
		serveRaw(resource)
	} else {
		// apply the first matching rule
		rule := rules[0]

		if !serveInput(resource, rule) && !serveCommand(resource, rule) && !serveRaw(resource) {
			log.Printf("Error serving resource: %v", resource)
		}
	}

	ch <- resource
}

func FormatPageContent() map[string]template.HTML {
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
				log.Printf("Error converting file: %v", file)
			} else {
				cache.lock.Lock()
				defer cache.lock.Unlock()

				// if the resource was also added concurrently, replace the old
				res, ok := cache.lookup[resource.inputHash]
				if ok && res.reader != nil {
					res.reader.Cancel()
					res.reader = nil
				}

				// store the new resource in the lookup for next time
				cache.lookup[resource.inputHash] = resource

				// update currRes
				cache.currRes = resource
			}
		}()
	}
}

func Update(w http.ResponseWriter, r *http.Request) {
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
		body["message"] = template.HTML(fmt.Sprintf("Error reading file or hash: %s %s", file, hash))
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

func Close(w http.ResponseWriter, r *http.Request) {
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
		body["message"] = template.HTML("Error finding reader")
	}
	util.RespondJson(w, body)
}

func File(w http.ResponseWriter, r *http.Request) {
	// serve the requested file by hash
	s := strings.Replace(r.URL.Path, "{document.location.href}", "", 1)
	hash := strings.Replace(s, "/file/", "", 1)

	cache.lock.Lock()
	defer cache.lock.Unlock()
	resource, ok := cache.lookup[hash]
	if !ok {
		// serve 404
		http.Error(w, "Resource Not Found", http.StatusNotFound)
	} else if resource.stream {
		// stream native audio/video files
		// fmt.Println("streaming")

		// setting Transfer-Encoding makes the stream performance acceptable, but spams the log with errors:
		// 		http: WriteHeader called with both Transfer-Encoding of "chunked" and a Content-Length of 782506548
		// removing Content-Length doesn't work because http.ServeContent() adds it back again:
		//		w.Header().Del("Content-Length")
		w.Header().Set("Transfer-Encoding", "chunked")

		// consider hijacking the writer's output and removing the Transfer-Encoding on the way out?
		// consider using ServeFileFS with wrappers for fs.FS and io.Seeker?
		// 		https://github.com/golang/go/issues/51971
		//		https://pkg.go.dev/net/http#ServeFileFS
		http.ServeContent(w, r, filepath.Base(resource.reader.Path), resource.reader.Info.ModTime(), resource.reader)

		// rw := http.ResponseWriter(w)
		// rw.Header().Set("Content-Length", "-1")
		// rw.Header().Del("Content-Length")
		// rw.Header().Set("Transfer-Encoding", "chunked")
		// http.ServeContent(rw, r, filepath.Base(resource.reader.Path), resource.reader.Info.ModTime(), resource.reader)
		// ServeHTTP(resource, rw, r)

	} else {
		// serve everything else in one go
		// fmt.Println("one file")

		// TODO: lomg-term, stop using ServeFile in favor of something that allows the server to close the file during transfer
		http.ServeFile(w, r, resource.outputExt)
	}
}

func ServeHTTP(resource *Resource, w http.ResponseWriter, r *http.Request) {
	// defer sfh.file.Close()

	fmt.Println("ServeHTTP() @ top")

	fmt.Printf("%v\n", r)

	sfh := resource.reader

	var buffer = make([]byte, int(sfh.Info.Size()))
	_, err := sfh.File.Read(buffer)
	if err != nil {
		fmt.Println("ServeHTTP() @ 1")
		log.Println(errors.New("error with writing file: " + "\nerror message: " + err.Error() + "\n"))
	}

	fmt.Println("ServeHTTP() @ 2", len(buffer))

	sfh.File.Seek(0, 0)
	contentType := http.DetectContentType(buffer)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-type", contentType)

	fmt.Println("ServeHTTP() @ 3")

	fmt.Println("ServeHTTP() @ 4", len(buffer))

	for _, b := range buffer {
		select {
		case <-sfh.Ctx.Done():
			fmt.Println("Transfer cancelled")
			http.Error(w, "Transfer cancelled", http.StatusGone)
			return
		default:
			a := []byte{b}
			w.Write(a)
		}
	}

	fmt.Println("ServeHTTP() @ 5")

	fmt.Println("ServeHTTP() @ bottom")
}
