package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ccammack/cannon/cache"
	"github.com/ccammack/cannon/cmd"
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
