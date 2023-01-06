/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

package server

// https://gist.github.com/rgl/0351b6d9362abb32d6b55f86bd17ab65

import (
	"cannon/cache"
	"cannon/config"
	"context"
	"encoding/json"
	"fmt"
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
		mux := http.NewServeMux()
		mux.HandleFunc("/", pageHandler)
		mux.HandleFunc("/status", statusHandler)
		mux.HandleFunc("/update", updateHandler)
		mux.HandleFunc("/stop", stopHandler)

		server = &http.Server{
			Addr:    fmt.Sprintf(":%v", port),
			Handler: mux,
		}
		server.ListenAndServe()
	}
}

func Stop() {
	// stop the server if the port is already in use
	if port, running := serverIsRunnning(); running {
		url := fmt.Sprintf("http://localhost:%v/%s", port, "stop")
		resp, err := http.Get(url)
		if err != nil {
		}
		defer resp.Body.Close()
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

func respondJson(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func respondHtml(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	respondHtml(w, cache.Page(r))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	respondJson(w, cache.Status(r))
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	respondJson(w, cache.Update(r))
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := json.Marshal(map[string]string{
		"state": "stopped",
	})
	respondJson(w, body)

	go func() {
		if err := server.Shutdown(context.Background()); err != nil {
		}
	}()
}
