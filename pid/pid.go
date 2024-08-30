package pid

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/adrg/xdg"
)

var pidPath = xdg.RuntimeDir + "/cannon.pid"

func pidfileContents() (int, error) {
	contents, err := ioutil.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			// file does not exist
			return 0, nil
		} else {
			// file exists but cannot be read; requires manual intervention
			log.Panicf("error reading PID file: %v", err)
			// return 0, err
		}
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(contents)))
	if err != nil {
		// file exists but cannot be parsed; requires manual intervention
		log.Panicf("error parsing PID file: %v", err)
		// return 0, err
	}

	// found a valid pid
	return pid, nil
}

func pidIsRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))

	if err != nil && err.Error() == "no such process" {
		return false
	}

	if err != nil && err.Error() == "os: process already finished" {
		return false
	}

	return true
}

func IsRunning() error {
	// read pidfile
	pid, _ := pidfileContents()
	if pid != 0 && pidIsRunning(pid) {
		// the process is still running
		return errors.New("process is already running")
	}

	// no such process or stale pid file
	return nil
}

func Lock() error {
	// read pidfile
	pid, _ := pidfileContents()
	if pid != 0 && pidIsRunning(pid) {
		return errors.New("process is already running")
	}

	// write pid to file
	return ioutil.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644)
}

func Unlock() error {
	// read pidfile
	pid, _ := pidfileContents()
	if pid == 0 || pidIsRunning(pid) {
		return os.RemoveAll(pidPath)
	}

	return nil
}
