/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

// TODO: golang serve file from memory
// TODO: golang lru cache
// https://www.alexedwards.net/blog/golang-response-snippets

// TODO: respond using JSON tamplates

package cache

import (
	"cannon/util"
	"html/template"
	"net/http"
	"os"
)

func init() {
}

const PageTemplate = `
<!doctype html>
<html>
  <head>
    <title>This is the title of the webpage!</title>
  </head>
  <body>
    <p>This is an example paragraph. Anything in the <strong>body</strong> tag will appear on the page, just like this <strong>p</strong> tag and its contents.</p>
  </body>
</html>
`

func Page(w *http.ResponseWriter) {
	// write the current page to either w or stdout
	t := template.New("page")
	t, err := t.Parse(PageTemplate)
	if err != nil {
		panic(err)
	}
	if w != nil {
		t.Execute(*w, nil)
	} else {
		t.Execute(os.Stdout, nil)
	}
}

func Update(w *http.ResponseWriter, r *http.Request) {
	body := map[string]string{
		"state": "updated",
	}
	util.RespondJson(w, body)
}

func Status(w *http.ResponseWriter) {
	body := map[string]string{
		"file": "file goes here",
	}
	util.RespondJson(w, body)
}
