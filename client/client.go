package client

import (
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
		util.CheckPanicOld(err)
	}
}

func Start() {
	if err := pid.IsRunning(); err != nil {
		log.Printf("error starting server: %v", err)
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
		log.Printf("error stopping server (already stopped?)")
	} else {
		Command("stop")
	}
}

func Toggle() {
	if err := pid.IsRunning(); err != nil {
		Stop()
	} else {
		Start()
	}
}

func Command(command string) {
	if err := pid.IsRunning(); err == nil {
		log.Printf("server is not running (use --start or --toggle to start)")
	}

	_, port := config.Port().Int()
	url := fmt.Sprintf("http://localhost:%d/%s", port, command)

	client := &http.Client{}

	// prepare request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("error creating request: %v", err)
		return
	}
	req.Header.Set("Accept", "application/json")

	// send it
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error making request: %v", err)
		return
	}
	defer resp.Body.Close()

	// check header
	// contentType := resp.Header.Get("Content-Type")
	// log.Printf("Content-Type: %s\n", contentType)

	// read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading response: %v", err)
		return
	}

	// log the result
	log.Printf("%v", string(body))
}
