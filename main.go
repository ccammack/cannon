package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ccammack/cannon/cache"
	"github.com/ccammack/cannon/client"
	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"
	"github.com/urfave/cli/v2"
)

func main() {
	// process command line
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
		Usage:     "a browser-based file previewer for terminal file managers",
		UsageText: `Cannon is a browser-based file previewer for terminal file managers:

https://github.com/dylanaraps/fff
https://github.com/gokcehan/lf
https://github.com/jarun/nnn
https://github.com/ranger/ranger

Cannon uses rules defined in its configuration file to convert each selected
file into its web-standard equivalent and then displays the converted file
in a web browser using a static HTTP server.`,

		ArgsUsage: "[global options] file",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "Suppress server console output",
				Action: func(ctx *cli.Context, v bool) error {
					log.SetOutput(io.Discard)
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "start",
				Aliases: []string{"s"},
				Usage:   "Start the preview server",
				Action: func(ctx *cli.Context, v bool) error {
					client.Start()
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "stop",
				Aliases: []string{"p"},
				Usage:   "Stop the preview server",
				Action: func(ctx *cli.Context, v bool) error {
					client.Stop()
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "toggle",
				Aliases: []string{"t"},
				Usage:   "Toggle the preview server on and off",
				Action: func(ctx *cli.Context, v bool) error {
					client.Toggle()
					return nil
				},
			},
			&cli.StringFlag{
				Name:    "close",
				Aliases: []string{"c"},
				Usage:   "Close the specified file.",
				Action: func(ctx *cli.Context, v string) error {
					var hash, file string
					var err error
					if hash, file, err = util.HashPath(v); err != nil {
						log.Printf("Error generating file hash: %v", err)
					}
					params := map[string]string{
						"file": file,
						"hash": hash,
					}
					client.Request("POST", "close", params)
					return nil
				},
			},
		},

		Action: func(cCtx *cli.Context) error {
			if cCtx.Args().Len() == 0 {
				return nil
			}

			v := cCtx.Args().Get(0)

			// display contents
			go displayContents(v)

			// display metadata
			displayMetadata(v)

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
		mime = cache.GetMimeType(v)
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
	client.Request("POST", "display", params)

}
