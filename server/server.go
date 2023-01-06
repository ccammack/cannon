/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

package server

// https://gist.github.com/rgl/0351b6d9362abb32d6b55f86bd17ab65

import (
	"cannon/cache"
	"cannon/config"
	"context"
	"fmt"
	"html"
	"net"
	"net/http"
)

var (
	server *http.Server
)

func serverIsRunnning() (int, bool) {
	// read config
	port := config.GetConfig().Settings.Port
	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", port))

	// return true if the port is already in use (assume that's the server)
	if err != nil {
		return port, true
	}

	_ = ln.Close()
	return port, false
}

func Start() {
	// start the server if the port is not in use
	if port, running := serverIsRunnning(); !running {
		// add routes
		mux := http.NewServeMux()
		mux.HandleFunc("/status", statusHandler)
		mux.HandleFunc("/update", updateHandler)
		mux.HandleFunc("/stop", stopHandler)

		// start server
		server = &http.Server{
			Addr:    fmt.Sprintf(":%v", port),
			Handler: mux,
		}
		// log.Fatal(server.ListenAndServe(), nil)
		server.ListenAndServe()
	}
}

func Stop() {
	// stop the server if the port is in use
	if port, running := serverIsRunnning(); running {
		// call /stop endpoint
		resp, err := http.Get(fmt.Sprintf("http://localhost:%v/stop", port))
		if err != nil {
			// handle error
		}
		defer resp.Body.Close()
		//body, err := io.ReadAll(resp.Body)
		//fmt.Println(body)
	}
}

func Toggle() {
	// stop the server if the port is in use; start it otherwise
	if _, running := serverIsRunnning(); running {
		Stop()
	} else {
		Start()
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "status, %q", html.EscapeString(r.URL.Path))
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	cache.Update()
	fmt.Fprintf(w, "update, %q", html.EscapeString(r.URL.RawQuery))
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	// fmt.Fprintf(w, "Stopped, %s!", r.URL.Path[1:])
	go func() {
		if err := server.Shutdown(context.Background()); err != nil {
			// log.Fatal(err)
		}
	}()
}
