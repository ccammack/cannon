package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ccammack/cannon/cache"
	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/pid"
	"github.com/ccammack/cannon/server"
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
					server.Start()
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "stop",
				Aliases: []string{"p"},
				Usage:   "Stop the preview server",
				Action: func(ctx *cli.Context, v bool) error {
					server.Stop()
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "toggle",
				Aliases: []string{"t"},
				Usage:   "Toggle the preview server on and off",
				Action: func(ctx *cli.Context, v bool) error {
					server.Toggle()
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "reset",
				Aliases: []string{"r"},
				Usage:   "Reset the connection to close the current file.",
				Action: func(ctx *cli.Context, v bool) error {
					server.Reset()
					return nil
				},
			},
			&cli.BoolFlag{
				Name: "page",
				// Aliases: []string{"g"},
				Usage: "Display the current page HTML for testing",
				Action: func(ctx *cli.Context, v bool) error {
					server.Page()
					return nil
				},
			},
			&cli.BoolFlag{
				Name: "status",
				// Aliases: []string{"u"},
				Usage: "Display the server status for testing",
				Action: func(ctx *cli.Context, v bool) error {
					server.Status()
					return nil
				},
			},
		},

		Action: func(cCtx *cli.Context) error {
			if cCtx.Args().Len() == 0 {
				return nil
			}

			path, err := filepath.Abs(cCtx.Args().Get(0))
			if err != nil {
				return err
			}

			fp, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fp.Close()

			// write mime type to stdout for display in the right pane
			// fmt.Println(cache.GetMimeType(path))

			// send file path argument to /update endpoint
			if err := pid.IsRunning(); err != nil {
				_, port := config.Port().Int()
				url := fmt.Sprintf("http://localhost:%d/%s", port, "update")
				postBody, _ := json.Marshal(map[string]string{
					"file": path,
				})
				responseBody := bytes.NewBuffer(postBody)
				resp, err := http.Post(url, "application/json", responseBody)
				if err != nil {
					log.Printf("error during post request: %v", err)
				}
				defer resp.Body.Close()
			}

			// lf requires a non-zero return value to disable caching
			_, exit := config.Exit().Int()
			mimetype := cache.GetMimeType(path)
			return cli.Exit(mimetype, exit)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
