package server

// https://gist.github.com/rgl/0351b6d9362abb32d6b55f86bd17ab65

import (
    "log"
    "fmt"
    "context"
    "net/http"
    "html"
)

var (
	server *http.Server
)

func Start() {
	if (server == nil) {
		// add routes
		mux := http.NewServeMux()
		mux.HandleFunc("/", statusHandler)
		mux.HandleFunc("/stop", stopHandler)
		mux.HandleFunc("/hello", helloHandler)

		// start server
		server = &http.Server {
			Addr:           ":8080",
			Handler:	mux,
		}
		log.Fatal(server.ListenAndServe(), nil)
	}
}

func Stop() {
	if (server != nil) {
		go func() {
			if err := server.Shutdown(context.Background()); err != nil {
				log.Fatal(err)
			}
		}()
	}
}

func Toggle() {
	if (server == nil) {
		Start()
	} else {
		Stop()
	}
}

func helloHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello\n")
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "status, %q", html.EscapeString(r.URL.Path))
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Stopped, %s!", r.URL.Path[1:])
    Stop()
}

