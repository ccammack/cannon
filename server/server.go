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
	"io"
	"log"
	"net"
	"net/http"
)

var (
	server *http.Server
)

func portInUse(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", port))

	if err != nil {
		return true
	}

	_ = ln.Close()
	return false
}

func Start() {
	// read config
	port := config.GetConfig().Settings.Port

	// start the server if the port is not in use
	if !portInUse(port) {
		// add routes
		mux := http.NewServeMux()
		mux.HandleFunc("/status", statusHandler)
		mux.HandleFunc("/stop", stopHandler)

		// start server
		server = &http.Server{
			Addr:    fmt.Sprintf(":%v", port),
			Handler: mux,
		}
		log.Fatal(server.ListenAndServe(), nil)
	}
}

func Stop() {
	// read config
	port := config.GetConfig().Settings.Port

	// stop the server if the port is in use
	if portInUse(port) {
		// call rest /stop
		resp, err := http.Get(fmt.Sprintf("http://localhost:%v/stop", port))
		if err != nil {
			// handle error
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		fmt.Println(body)
	}
}

func Toggle() {
	// read config
	port := config.GetConfig().Settings.Port

	// stop the server if the port is in use; start it otherwise
	if portInUse(port) {
		Stop()
	} else {
		Start()
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "status, %q", html.EscapeString(r.URL.Path))
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Stopped, %s!", r.URL.Path[1:])
	go func() {
		if err := server.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()
}

// updateHandler
