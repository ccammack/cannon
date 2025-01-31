package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/resources"
	"github.com/ccammack/cannon/server"
	"github.com/ccammack/cannon/util"
	"github.com/urfave/cli/v2"
)

func main() {
	// process command line
	close := false

	app := &cli.App{
		Name:     "Cannon",
		Version:  "v0.0.1",
		Compiled: time.Now(),
		Authors: []*cli.Author{
			{
				Name:  "Chris Cammack",
				Email: "clc1024@hotmail.com",
			},
		},
		Copyright: "(c) 2022 Chris Cammack",
		HelpName:  "cannon",
		Usage:     "send a filename to the Cannon server for display in the web browser.",
		UsageText: "cannon [OPTION]... file",
		ArgsUsage: "[global options] file",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "suppress log output",
				Action: func(ctx *cli.Context, v bool) error {
					log.SetOutput(io.Discard)
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "close",
				Aliases: []string{"c"},
				Usage:   "close the specified file.",
				Action: func(ctx *cli.Context, v bool) error {
					close = v
					return nil
				},
			},
		},

		Action: func(cCtx *cli.Context) error {
			if cCtx.Args().Len() == 0 {
				cli.ShowAppHelpAndExit(cCtx, 1)
			}

			fname := cCtx.Args().Get(0)
			// width := cCtx.Args().Get(1)
			// height := cCtx.Args().Get(2)
			// hpos := cCtx.Args().Get(3)
			// vpos := cCtx.Args().Get(4)

			if close {
				// close the specified file
				var hash, file string
				var err error
				if hash, file, err = util.HashPath(fname); err != nil {
					log.Printf("Error generating file hash: %v", err)
				}
				params := map[string]string{
					"file": file,
					"hash": hash,
				}
				server.Request("POST", "close", params)
			} else {
				// display contents
				go displayContents(fname)

				// display metadata
				displayMetadata(fname)
			}

			// lf requires a non-zero return value to disable caching
			_, exit := config.Exit().Int()
			return cli.Exit("", exit)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func displayMetadata(v string) {
	var wg sync.WaitGroup
	wg.Add(2)
	var mime, meta string
	go func() {
		defer wg.Done()
		mime = resources.GetMimeType(v)
	}()
	go func() {
		defer wg.Done()
		meta, _ = util.GetMetadataDisplayString(v)
	}()
	wg.Wait()
	fmt.Println(mime)
	fmt.Println(meta)
}

func displayContents(v string) {
	// display the specified file
	var hash, file string
	var err error
	if hash, file, err = util.HashPath(v); err != nil {
		log.Printf("Error generating file hash: %v", err)
	}
	params := map[string]string{
		"file": file,
		"hash": hash,
	}
	server.Request("POST", "display", params)
}
