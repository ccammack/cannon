package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/pid"
	"github.com/ccammack/cannon/server"
	"github.com/ccammack/cannon/util"
)

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
	if err := pid.IsRunning(); err != nil {
		log.Printf("Error starting server: %v", err)
	} else {
		// port
		_, port := config.Port().Int()
		url := fmt.Sprintf("http://localhost:%d", port)

		// start preview browser
		go startBrowser(url)

		// start server
		server.Start()
	}
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
