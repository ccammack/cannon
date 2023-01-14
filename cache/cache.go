/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

// TODO: golang serve file from memory
// TODO: golang lru cache
// https://www.alexedwards.net/blog/golang-response-snippets

package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"net/http"
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
	Input  string
	Hash   string
	Output string
	Tag    string
}

type Resources struct {
	current string
	lookup  map[string]Resource
}

var (
	resources     *Resources
	resoursesLock = new(sync.RWMutex)
)

func init() {
	resources = new(Resources)
	resources.lookup = make(map[string]Resource)
}

func getResources() *Resources {
	resoursesLock.RLock()
	defer resoursesLock.RUnlock()
	return resources
}

func convertFile(file string, hash string) {
	// TODO: move file conversion into a coroutine
	// TODO: iterate config rules and run the matching one
	resource := Resource{
		file,
		hash,
		"output",
		"tag",
	}
	resources := getResources()
	resources.lookup[hash] = resource
}

func setCurrentResource(file string) {
	hash := makeHash(file)
	resources := getResources()
	resource, ok := resources.lookup[hash]
	if !ok || (len(resource.Output) == 0) || (len(resource.Tag) == 0) {
		// add a new null entry
		resources.lookup[hash] = Resource{
			file,
			hash,
			"",
			"",
		}

		// perform file conversion and then fill out the resource
		go convertFile(file, hash)
	}
	resources.current = hash
}

type PageData struct {
	// id
	// name
	// hash
	// tag
	// status url

	Id    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

const PageTemplate = `
<!doctype html>
<html>
	<head>
		<title>{{.name}}</title>
		<script>
			const hash = "{{.hash}}";
			const timer = {{.timer}};
			let url = "";

			window.onload = function(e) {
				// copy server address from document.location.href
				console.log("onload");
				url = document.location.href + "status";
				const media = document.getElementById("media");
				console.log(media)
				if (media) {
					console.log(media.tagName);
					media.src = document.location.href + "file?hash=" + hash;
				}

				setTimeout(function status() {
					// ask the server for updates and reload if needed
					console.log("status");
					fetch(url)
					.then((response) => response.json())
					.then((data) => {
						console.log(data);
						if (data.hash != hash) {
							location.reload();
						}
						setTimeout(status, timer);
					});
				}, timer);
			}
		</script>
	</head>
	<body>
		<img id="{{.id}}" src="{{.src}}">
	</body>
</html>
`

// <img id="media" src="http://localhost:8888/file/wdn2oiuhfiu2ncoine">
// <video id="media" src="http://localhost:8888/file/wdn2oiuhfiu2ncoine"></video>

func Page(w *http.ResponseWriter) {
	data := map[string]string{
		"name":  "Bob's File Goes Here",
		"hash":  "w3ij2ofi3eoc34fh43kvn3o4n",
		"timer": "5000",
		"id":    "media",
		"src":   "https://upload.wikimedia.org/wikipedia/commons/1/1a/Donkey_in_Clovelly%2C_North_Devon%2C_England.jpg",
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

func makeHash(s string) string {
	// TODO: is sha1 a good choice here?
	hash := sha1.New()
	hash.Write([]byte(s))
	hashstr := hex.EncodeToString(hash.Sum(nil))
	return hashstr
}

func Update(w *http.ResponseWriter, r *http.Request) {
	// TODO: save this info in reference.org
	// res, error := httputil.DumpRequest(r, true)
	// if error != nil {
	// 	log.Fatal(error)
	// }
	// fmt.Print(string(res))
	// util.Append(string(res))

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
	// TODO: match /file/wdn2oiuhfiu2ncoine
	// https://gist.github.com/reagent/043da4661d2984e9ecb1ccb5343bf438
	// https://www.honeybadger.io/blog/go-web-services/

	hash := r.URL.Query().Get("hash")
	if hash == "wdn2oiuhfiu2ncoine" {
		http.ServeFile(*w, r, "/home/ccammack/Downloads/FmFAbozXoAEMm3l.jpeg")
	} else {
		http.ServeFile(*w, r, "/home/ccammack/Downloads/FixL3ExWQAkNoiS.png")
	}

	//http.ServeFile(*w, r, "/home/ccammack/Downloads/American P-51 Fighters Attack Tokyo, Incredible Remastered HD Footage [SAPqr3YCNmA].webm")
}

func Status(w *http.ResponseWriter) {
	// goal
	// util.RespondJson(w, `{"file": "the current very successful file will also go here"}`)

	// works
	// body := map[string]string{
	// 	"file": "file goes here",
	// }
	// util.RespondJson(w, body)

	// works
	type StatusMessage struct {
		Hash string `json:"hash"`
	}
	body := StatusMessage{
		Hash: "wdn2oiuhfiu2ncoine",
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
