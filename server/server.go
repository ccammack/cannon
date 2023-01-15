/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

package server

// https://gist.github.com/rgl/0351b6d9362abb32d6b55f86bd17ab65

import (
	"cannon/cache"
	"cannon/config"
	"cannon/util"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"strings"
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

func startBrowser() {
	address := config.GetConfig().Settings.Server
	port := config.GetConfig().Settings.Port
	url := fmt.Sprintf("%s:%d", address, port)
	browser := config.GetConfig().Settings.Browser
	command := browser[0]
	rest := browser[1:]
	args := []string{}
	for _, arg := range rest {
		arg := strings.Replace(arg, "{url}", url, 1)
		args = append(args, arg)
	}
	cmd := exec.Command(command, args...)
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func Start() {
	// start the local preview browser
	go startBrowser()

	// start the server if the port is not in use
	if port, running := serverIsRunnning(); !running {
		mux := http.NewServeMux()
		mux.HandleFunc("/", pageHandler)
		mux.HandleFunc("/status", statusHandler)
		mux.HandleFunc("/update", updateHandler)
		mux.HandleFunc("/file", fileHandler)
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

func Page() {
	// display the current page HTML for testing
	cache.Page(nil)
}

func Status() {
	// display the server status for testing
	cache.Status(nil)
}

func dumpRequest(r *http.Request) {
	// TODO: save this info in reference.org
	res, error := httputil.DumpRequest(r, true)
	if error != nil {
		log.Fatal(error)
	}
	fmt.Print(string(res))
	util.Append(string(res))
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /
	cache.Page(&w)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /status
	cache.Status(&w)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /update
	cache.Update(&w, r)
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /update
	cache.File(&w, r)
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /stop
	// body := map[string]string{
	// 	"state": "stopped",
	// }
	// util.RespondJson(&w, body)

	type StopMessage struct {
		State string `json:"state"`
	}

	body := StopMessage{
		State: "stopped",
	}
	if w != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(body)
	} else {
		json.NewEncoder(os.Stdout).Encode(body)
	}

	go func() {
		if err := server.Shutdown(context.Background()); err != nil {
		}
	}()
}
