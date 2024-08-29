package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/ccammack/cannon/cache"
	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"
)

var (
	server *http.Server
)

func ServerIsRunnning() (int, bool) {
	// read config
	_, port := config.Port().Int()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", port))

	// return true if the port is already in use (assume that's the server)
	if err != nil {
		return port, true
	}

	_ = ln.Close()
	return port, false
}

func startBrowser() {
	_, port := config.Port().Int()
	url := fmt.Sprintf("%s:%d", "https://localhost", port)
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

	// validate server config and watch for changes
	config.Start()

	// start the local preview browser
	go startBrowser()

	// start the server if the port is not in use
	if port, running := ServerIsRunnning(); !running {
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
	if port, running := ServerIsRunnning(); running {
		url := fmt.Sprintf("http://localhost:%v/%s", port, "stop")
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("error stopping server: %v", err)
		}
		defer resp.Body.Close()

		// normal cleanup
		cache.Exit()
	}
}

func Toggle() {
	// stop the server if the port is in use; start it otherwise
	if _, running := ServerIsRunnning(); running {
		Stop()
	} else {
		Start()
	}
}

func Page() {
	// display the current page HTML for testing
	if _, running := ServerIsRunnning(); running {
		cache.Page(nil)
	} else {
		fmt.Println("Cannon server is not running. Use --start or --toggle to start it.")
	}
}

func Reset() {
	// reset the connection from the server side to unlock the file so it can be moved/deleted
	log.Panicln("server.Reset() is not yet implemented")
}

func Status() {
	// display the server status for testing
	if _, running := ServerIsRunnning(); running {
		cache.Status(nil)
	} else {
		fmt.Println("Cannon server is not running. Use --start or --toggle to start it.")
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
		}
	}()
}
