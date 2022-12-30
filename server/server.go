/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

package server

// https://gist.github.com/rgl/0351b6d9362abb32d6b55f86bd17ab65

import (
	"cannon/config"
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
)

var (
	server *http.Server
)

func Start() {

	// server will always be nil because this is being called from the command line
	// call /status to see if the server is running on the expected port

	if server == nil {
		// add routes
		mux := http.NewServeMux()
		mux.HandleFunc("/status", statusHandler)
		mux.HandleFunc("/stop", stopHandler)

		// read config
		port := config.GetConfig().Settings.Port
		fmt.Println(port)

		// start server
		server = &http.Server{
			Addr:    fmt.Sprintf(":%v", port),
			Handler: mux,
		}
		log.Fatal(server.ListenAndServe(), nil)
	}
}

func Stop() {
	if server != nil {
		go func() {
			if err := server.Shutdown(context.Background()); err != nil {
				log.Fatal(err)
			}
		}()
	}
}

func Toggle() {
	if server == nil {
		Start()
	} else {
		Stop()
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "status, %q", html.EscapeString(r.URL.Path))
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Stopped, %s!", r.URL.Path[1:])
	Stop()
}
