/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

package main

import (
	"cannon/cache"
	"cannon/cmd"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// prepare exit strategy
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cache.Exit()
		os.Exit(0)
	}()

	// process command line
	cmd.Execute()

	// normal cleanup
	cache.Exit()
}
