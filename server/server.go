package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/ccammack/cannon/cache"
	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/pid"
	"github.com/ccammack/cannon/util"
)

var (
	server *http.Server
)

func shutdown() {
	// normal cleanup
	cache.Shutdown()

	// unlock pid
	pid.Unlock()

	go func() {
		// shutdown
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
	}()
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	body := map[string]interface{}{"status": template.HTML("success")}
	util.RespondJson(w, body)
	shutdown()
}

func startBrowser(url string) {
	_, command := config.Browser().Strings()
	if len(command) > 0 {
		cmd, args := util.FormatCommand(command, map[string]string{"{url}": url})
		proc := exec.Command(cmd, args...)
		err := proc.Start()
		if err != nil {
			log.Printf("error starting browser: %v", err)
		}
	}
}

func Start() {
	// catch exit signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		shutdown()
	}()

	// check for running server
	if err := pid.IsRunning(); err != nil {
		log.Printf("Error starting server: %v", err)
		return
	}

	// lock pid
	pid.Lock()

	// log server address
	_, port := config.Port().Int()
	url := fmt.Sprintf("http://localhost:%d", port)
	log.Printf("Starting server: %s", url)

	// start preview browser
	go startBrowser(url)

	// watch for config changes
	config.Watch()

	// validate server config
	config.Validate()

	// broadcast
	go func() {
		for {
			cache.BroadcastCurrent()
			time.Sleep(33 * time.Millisecond)
		}
	}()

	// listen and serve
	mux := http.NewServeMux()
	mux.HandleFunc("/", cache.HandleRoot)
	mux.HandleFunc("/src/", cache.HandleSrc)
	mux.HandleFunc("/display", cache.HandleDisplay)
	mux.HandleFunc("/stop", handleStop)
	mux.HandleFunc("/close", cache.HandleClose)
	server = &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: mux,
	}
	log.Fatal(server.ListenAndServe())
}

func Stop() {
	if err := pid.IsRunning(); err == nil {
		log.Printf("Error stopping server (already stopped?)")
	} else {
		Request("POST", "stop", nil)
	}
}

func Toggle() {
	if err := pid.IsRunning(); err != nil {
		Stop()
	} else {
		Start()
	}
}

func Request(method string, resource string, params map[string]string) {
	if err := pid.IsRunning(); err == nil {
		log.Printf("Server is not running (use --start or --toggle to start)")
	}

	_, port := config.Port().Int()
	url := fmt.Sprintf("http://localhost:%d/%s", port, resource)

	// prepare request
	json, err := json.Marshal(params)
	if err != nil {
		log.Printf("Error marshalling request params: %v", err)
		return
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(json))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}
	req.Header.Set("Accept", "application/json")

	// send it
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v", err)
		return
	}

	// check header
	// contentType := resp.Header.Get("Content-Type")
	// log.Printf("Content-Type: %s\n", contentType)

	// read response body
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		return
	}

	// log the result
	log.Printf("%v", string(body))
}
