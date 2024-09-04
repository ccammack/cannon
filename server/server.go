package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/ccammack/cannon/cache"
	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/pid"
	"github.com/ccammack/cannon/util"
)

var (
	server *http.Server
)

func startBrowser(url string) {
	_, command := config.Browser().Strings()
	if len(command) > 0 {
		cmd, args := util.FormatCommand(command, map[string]string{"{url}": url})
		proc := exec.Command(cmd, args...)
		err := proc.Start()
		util.CheckPanicOld(err)
	}
}

func Start() {
	// prepare exit strategy
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		Stop()
	}()

	// watch for config changes
	config.Start()

	// log server address
	_, port := config.Port().Int()
	url := fmt.Sprintf("http://localhost:%d", port)
	log.Printf("starting server: %s", url)

	// start the server
	if err := pid.IsRunning(); err != nil {
		log.Printf("cannot start server: %v", err)
		return
	}

	// lock pid
	pid.Lock()

	// start the local preview browser
	go startBrowser(url)

	// validate server config
	config.Validate()

	// serve files
	mux := http.NewServeMux()
	mux.HandleFunc("/", pageHandler)
	mux.HandleFunc("/status", statusHandler)
	mux.HandleFunc("/update", updateHandler)
	mux.HandleFunc("/stop", stopHandler)
	mux.HandleFunc("/reset", resetHandler)
	server = &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Printf("cannot start server: %v", err)
		return
	}
}

func Stop() {
	// stop the server if the port is already in use
	if err := pid.IsRunning(); err == nil {
		log.Printf("cannot stop server: %v", err)
		return
	}

	_, port := config.Port().Int()
	url := fmt.Sprintf("http://localhost:%d/%s", port, "stop")
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("error stopping server: %v", err)
	}
	defer resp.Body.Close()

	// unlock pid
	pid.Unlock()

	// normal cleanup
	cache.Exit()
}

func Toggle() {
	// stop the server if the port is in use; start it otherwise
	if err := pid.IsRunning(); err != nil {
		Stop()
	} else {
		Start()
	}
}

func Page() {
	// display the current page HTML for testing
	if err := pid.IsRunning(); err != nil {
		cache.Page(nil)
	} else {
		log.Printf("Cannon server is not running. Use --start or --toggle to start it.")
	}
}
func Reset() {
	// reset the current streaming connection
	if err := pid.IsRunning(); err == nil {
		log.Printf("cannot reset server: %v", err)
		return
	}

	_, port := config.Port().Int()
	url := fmt.Sprintf("http://localhost:%d/%s", port, "reset")
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("error resetting server: %v", err)
	}
	defer resp.Body.Close()
}

func Status() {
	// display the server status for testing
	if err := pid.IsRunning(); err != nil {
		cache.Status(nil)
	} else {
		log.Printf("Cannon server is not running. Use --start or --toggle to start it.")
	}
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		// handle route /
		cache.Page(&w)
	} else {
		// handle route /<hash>
		cache.File(&w, r)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /status
	cache.Status(&w)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /update
	cache.Update(&w, r)
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /stop
	body := map[string]string{
		"state": "stopped",
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
			log.Printf("error stopping server: %v", err)
		}
	}()
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("server.reset()")
	cache.Reset(&w, r)
}
