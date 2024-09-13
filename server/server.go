package server

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
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

func Start() {
	// catch exit signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		shutdown()
	}()

	// watch for config changes
	config.Start()

	// log server address
	_, port := config.Port().Int()
	url := fmt.Sprintf("http://localhost:%d", port)
	log.Printf("Starting server: %s", url)

	// lock pid
	pid.Lock()

	// validate server config
	config.Validate()

	// listen and serve
	mux := http.NewServeMux()
	mux.HandleFunc("/", cache.HandleRoot)
	mux.HandleFunc("/file/", cache.HandleFile)
	mux.HandleFunc("/display", cache.HandleDisplay)
	mux.HandleFunc("/stop", handleStop)
	mux.HandleFunc("/close", cache.HandleClose)
	server = &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: mux,
	}
	log.Fatal(server.ListenAndServe())
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	body := map[string]interface{}{"status": template.HTML("success")}
	util.RespondJson(w, body)

	shutdown()
}
