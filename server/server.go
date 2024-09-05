package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	cache.Exit()

	// unlock pid
	pid.Unlock()

	go func() {
		// shutdown
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("error stopping server: %v", err)
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
	log.Printf("starting server: %s", url)

	// lock pid
	pid.Lock()

	// validate server config
	config.Validate()

	// listen and serve
	mux := http.NewServeMux()
	mux.HandleFunc("/", pageHandler)
	mux.HandleFunc("/status", statusHandler)
	mux.HandleFunc("/file/", fileHandler)
	mux.HandleFunc("/update", updateHandler)
	mux.HandleFunc("/stop", stopHandler)
	mux.HandleFunc("/reset", resetHandler)
	server = &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: mux,
	}
	log.Fatal(server.ListenAndServe())
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /status
	data := cache.FormatCurrentResourceData()

	// generate complete html page from template
	t, err := template.New("page").Parse(cache.PageTemplate)
	util.CheckPanicOld(err)

	accept := r.Header.Get("Accept")
	if accept == "" || strings.Contains(accept, "text/html") {
		// html
		err = t.Execute(w, data)
		util.CheckPanicOld(err)
	} else {
		// json
		data["page"] = template.HTML(t.Tree.Root.String())
		respondJson(w, r, data)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /status
	data := map[string]template.HTML{
		"action": "status",
	}
	maps.Copy(data, cache.FormatCurrentResourceData())
	respondJson(w, r, data)
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /<hash>
	cache.File(&w, r)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /update
	cache.Update(&w, r)
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	respondJson(w, r, map[string]template.HTML{
		"action": "stop",
	})

	shutdown()
}

func respondJson(w http.ResponseWriter, r *http.Request, data map[string]template.HTML) {
	// json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	// handle route /reset
	cache.Reset()
	respondJson(w, r, map[string]template.HTML{
		"action": "reset",
	})
}

//

//

//

// func defaultHandler(w http.ResponseWriter, r *http.Request) {
// 	// serve 404
// 	http.Error(w, "404", http.StatusNotFound)
// }

// func respond(w http.ResponseWriter, r *http.Request, data map[string]template.HTML) {
// 	accept := r.Header.Get("Accept")
// 	if accept == "" || strings.Contains(accept, "text/html") {
// 		// html
// 		w.Header().Set("Content-Type", "text/html")
// 		w.WriteHeader(http.StatusOK)
// 		fmt.Fprint(w, "s", data)
// 	} else {
// 		// json
// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(http.StatusOK)
// 		json.NewEncoder(w).Encode(data)
// 	}
// }

// func respond(w http.ResponseWriter, r *http.Request, data any) {
// 	accept := r.Header.Get("Accept")
// 	if accept == "" || strings.Contains(accept, "text/html") {
// 		// html
// 		w.Header().Set("Content-Type", "text/html")
// 		w.WriteHeader(http.StatusOK)
// 		fmt.Fprint(w, "s", data)
// 	} else if strings.Contains(accept, "application/json") {
// 		// json
// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(http.StatusOK)
// 		json.NewEncoder(w).Encode(data)
// 	} else {
// 		// unsupported
// 		log.Printf("unsupported request headers %s in server.respondJson()", accept)
// 		http.Error(w, "Invalid request headers in server.respondJson()", http.StatusInternalServerError)
// 	}
// }
