/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

// TODO: golang serve file from memory
// TODO: golang lru cache
// https://www.alexedwards.net/blog/golang-response-snippets

package cache

import (
	"cannon/util"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"sync"
)

// https://vivek-syngh.medium.com/http-response-in-golang-4ca1b3688d6
// https://programmer.help/blogs/golang-json-encoding-decoding-and-text-html-templates.html
// https://stackoverflow.com/questions/38436854/golang-use-json-in-template-directly
// https://gist.github.com/alex-leonhardt/8ed3f78545706d89d466434fb6870023
// https://gist.github.com/Integralist/d47c2e8c6064ec065108ad59df6e1fb9
// https://go.dev/blog/json
// https://www.sohamkamani.com/golang/json/
// https://stackoverflow.com/questions/30537035/golang-json-rawmessage-literal
// https://go.dev/play/p/C1tXFi23Bw
// https://appdividend.com/2022/06/22/golang-serialize-json-string/
// https://www.socketloop.com/tutorials/golang-marshal-and-unmarshal-json-rawmessage-struct-example
// https://noamt.medium.com/using-gos-json-rawmessage-a2371a1c11b7
// https://stackoverflow.com/questions/23255456/whats-the-proper-way-to-convert-a-json-rawmessage-to-a-struct
// https://jhall.io/pdf/Advanced%20JSON%20handling%20in%20Go.pdf
// https://codewithyury.com/how-to-correctly-serialize-json-string-in-golang/
// https://www.digitalocean.com/community/tutorials/how-to-use-json-in-go
// https://gobyexample.com/json
// https://yourbasic.org/golang/json-example/

type Resource struct {
	filenameIn     string
	filenameInHash string
	filenameOut    string
	html           string
	htmlHash       string
}

type Resources struct {
	currentHash    string
	resourceLookup map[string]Resource
}

var (
	resources     *Resources
	resoursesLock = new(sync.RWMutex)
)

func makeHash(s string) string {
	// TODO: is sha1 a good choice here?
	hash := sha1.New()
	hash.Write([]byte(s))
	hashstr := hex.EncodeToString(hash.Sum(nil))
	return hashstr
}

func init() {
	resources = new(Resources)
	resources.resourceLookup = make(map[string]Resource)

	// add default resource
	hash := "0"
	html := "<p>Waiting...</p>"
	resource := Resource{
		"Waiting...",
		hash,
		"",
		html,
		makeHash(html),
	}
	resources.resourceLookup[hash] = resource
	resources.currentHash = hash
}

func getResources() *Resources {
	resoursesLock.RLock()
	defer resoursesLock.RUnlock()
	return resources
}

func convertFile(file string, hash string) {
	// TODO: move file conversion into a coroutine
	// TODO: iterate config rules and run the matching one

	html := "<img src='##document.location.href##file?hash=" + hash + "'>"
	//html := file

	resource := Resource{
		file,
		hash,
		file,
		html,
		makeHash(html),
	}
	resources := getResources()
	resources.resourceLookup[hash] = resource
}

func setCurrentResource(file string) {
	hash := makeHash(file)
	resources := getResources()
	_, ok := resources.resourceLookup[hash]
	if !ok {
		// add a new null entry
		resources.resourceLookup[hash] = Resource{
			file,
			hash,
			"",
			"",
			"",
		}

		// perform file conversion and then fill out the resource
		go convertFile(file, hash)
	}
	resources.currentHash = hash
}

const PageTemplate = `
<!doctype html>
<html>
	<head>
		<title>{{.filename}}</title>
		<script>
			// let filehash = "{{.filehash}}";
			// let htmlhash = "{{.htmlhash}}";
			let filehash = "0";
			let htmlhash = "0";

			//const html = "{{.html}}"

			function replaceContent(html) {
				console.log("replaceContent")
				const container = document.getElementById("{{.containerid}}");
				console.log(container)
				if (container) {
					const inner = html.replace("##document.location.href##", document.location.href)
					container.innerHTML = inner;
				}
			}

			window.onload = function(e) {
				// copy server address from document.location.href
				console.log("onload");

				//replaceContent(html)

				const statusurl = document.location.href + "status"
				//const statusurl = "https://jsonplaceholder.typicode.com/comments?postId=1"

				setTimeout(function status() {
					// ask the server for updates and reload if needed
					fetch(statusurl)
					.then((response) => response.json())
					.then((data) => {
						console.log(data);

						// if (filehash != data.filehash) {
						// 	console.log("filehash")
						// 	filehash = data.filehash
						// 	//location.reload();
						// } else if (data.htmlhash != htmlhash) {
						// 	console.log("htmlhash")
						// 	replaceContent(data.html)
						// 	htmlhash = data.htmlhash;
						// }

						if (htmlhash != data.htmlhash) {
							filehash = data.filehash
							htmlhash = data.htmlhash;
							document.title = data.filename
							replaceContent(data.html)
						}

						setTimeout(status, {{.intervalms}});
					});
				}, {{.intervalms}});
			}
		</script>
	</head>
	<body>
		<div id="{{.containerid}}"></div>
	</body>
</html>
`

func Page(w *http.ResponseWriter) {
	resources := getResources()
	resource, ok := resources.resourceLookup[resources.currentHash]
	if !ok {
		panic("Resource lookup failed in cache.go!")
	}

	data := map[string]string{
		"filename":    resource.filenameIn,
		"filehash":    resource.filenameInHash,
		"html":        resource.html,
		"htmlhash":    resource.htmlHash,
		"intervalms":  "100",
		"containerid": "container",
	}

	// write the current page to either w or stdout
	t := template.New("page")
	t, err := t.Parse(PageTemplate)
	if err != nil {
		panic(err)
	}
	if w != nil {
		t.Execute(*w, data)
	} else {
		t.Execute(os.Stdout, data)
	}
}

func DumpRequest(r *http.Request) {
	// TODO: save this info in reference.org
	res, error := httputil.DumpRequest(r, true)
	if error != nil {
		log.Fatal(error)
	}
	fmt.Print(string(res))
	util.Append(string(res))
}

func Update(w *http.ResponseWriter, r *http.Request) {
	DumpRequest(r)

	// extract params from the request body
	type Params struct {
		File string `json:"file"`
	}
	var params Params
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		panic(err)
	}
	// fmt.Print(params.File)
	// util.Append(params.File)

	// set the current file to display
	setCurrentResource(params.File)

	// body := map[string]string{
	// 	"state": "updated",
	// }
	// util.RespondJson(w, body)

	type UpdateMessage struct {
		State string `json:"state"`
	}

	body := UpdateMessage{
		State: "updated",
	}
	if w != nil {
		(*w).Header().Set("Content-Type", "application/json")
		(*w).WriteHeader(http.StatusOK)
		json.NewEncoder(*w).Encode(body)
	} else {
		json.NewEncoder(os.Stdout).Encode(body)
	}
}

func File(w *http.ResponseWriter, r *http.Request) {
	DumpRequest(r)

	// TODO: match /file/wdn2oiuhfiu2ncoine
	// https://gist.github.com/reagent/043da4661d2984e9ecb1ccb5343bf438
	// https://www.honeybadger.io/blog/go-web-services/

	hash := r.URL.Query().Get("hash")

	resources := getResources()
	resource, ok := resources.resourceLookup[hash]
	if !ok {
		panic("Resource lookup failed in cache.go!")
	}

	// serve the console stdout+stderr until the process completes, then serve the output file
	http.ServeFile(*w, r, resource.filenameOut)

	// if hash == "wdn2oiuhfiu2ncoine" {
	// 	http.ServeFile(*w, r, "/home/ccammack/Downloads/FmFAbozXoAEMm3l.jpeg")
	// } else {
	// 	http.ServeFile(*w, r, "/home/ccammack/Downloads/FixL3ExWQAkNoiS.png")
	// }

	//http.ServeFile(*w, r, "/home/ccammack/Downloads/American P-51 Fighters Attack Tokyo, Incredible Remastered HD Footage [SAPqr3YCNmA].webm")
}

func Status(w *http.ResponseWriter) {
	resources := getResources()
	resource, ok := resources.resourceLookup[resources.currentHash]
	if !ok {
		panic("Resource lookup failed in cache.go!")
	}

	// uri := "https://<server>:<port>"
	// if r != nil {
	// 	uri, err := url.QueryUnescape(r.URL.Query().Get("uri"))
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }

	// goal
	// util.RespondJson(w, `{"file": "the current very successful file will also go here"}`)

	// works
	// body := map[string]string{
	// 	"file": "file goes here",
	// }
	// util.RespondJson(w, body)

	// works
	type StatusMessage struct {
		Filename string `json:"filename"`
		Filehash string `json:"filehash"`
		Html     string `json:"html"`
		Htmlhash string `json:"htmlhash"`
	}
	body := StatusMessage{
		Filename: resource.filenameIn,
		Filehash: resource.filenameInHash,
		Html:     resource.html,
		Htmlhash: resource.htmlHash,
	}

	// works
	// var body map[string]interface{}
	// err := json.Unmarshal([]byte(`{"file": "the current file will also go here"}`), &body)
	// if err != nil {
	// 	panic(err)
	// }

	// works
	// var body json.RawMessage
	// err := body.UnmarshalJSON([]byte(`{"file": "the current file will go here"}`))
	// if err != nil {
	// 	panic(err)
	// }

	if w != nil {
		(*w).Header().Set("Content-Type", "application/json")
		(*w).WriteHeader(http.StatusOK)
		json.NewEncoder(*w).Encode(body)
	} else {
		json.NewEncoder(os.Stdout).Encode(body)
	}
}
