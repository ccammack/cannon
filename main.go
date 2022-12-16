
package main

import (
    "fmt"
    "log"
    "net/http"
)

const jsFile = `alert('Hello World!');`

func main() {
    http.HandleFunc("/file.js", JsHandler)

    log.Fatal(http.ListenAndServe(":5000", nil))
}

func JsHandler(w http.ResponseWriter, r *http.Request) {
    // Getting the headers so we can set the correct mime type
    headers := w.Header()
    headers["Content-Type"] = []string{"application/javascript"}
    fmt.Fprint(w, jsFile)
}
