package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ccammack/cannon/cache"
	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/server"
	"github.com/urfave/cli/v2"
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
	// cmd.Execute()

	// process command line
	app := &cli.App{
		Name:     "Cannon",
		Version:  "v0.0.1",
		Compiled: time.Now(),
		// Authors: []*cli.Author{
		// 	&cli.Author{
		// 		Name:  "Chris Cammack",
		// 		Email: "clc1024@hotmail.com",
		// 	},
		// },
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
				Name: "start",
				// Aliases: []string{"s"},
				Usage: "Start the preview server",
				Action: func(ctx *cli.Context, v bool) error {
					server.Start()
					return nil
				},
			},
			&cli.BoolFlag{
				Name: "stop",
				// Aliases: []string{"p"},
				Usage: "Stop the preview server",
				Action: func(ctx *cli.Context, v bool) error {
					server.Stop()
					return nil
				},
			},
			&cli.BoolFlag{
				Name: "toggle",
				// Aliases: []string{"t"},
				Usage: "Toggle the preview server on and off",
				Action: func(ctx *cli.Context, v bool) error {
					server.Toggle()
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
			// fmt.Printf("%v", cCtx)
			// fmt.Printf("%v\n", strings.Join(cCtx.StringSlice("start"), `, `))
			// fmt.Printf("%v", cCtx.App)
			// fmt.Printf("%v", cCtx.App.Flags)
			// fmt.Printf("%v", cCtx.App.Commands)
			// fmt.Printf("%q", cCtx.NumFlags())
			// fmt.Printf("%q", cCtx.App.Flags[0])
			// fmt.Printf("%q", cCtx.Args().Get(0))

			fmt.Printf("%d", cCtx.Args().Len())

			if cCtx.Args().Len() > 0 {
				path, err := filepath.Abs(cCtx.Args().Get(0))
				if err != nil {
					fmt.Println(err)
				} else {
					fp, err := os.Open(path)
					if err != nil {
						fmt.Println(err)
					} else {
						defer fp.Close()

						// write mime type to stdout for display in the right pane
						fmt.Println(cache.GetMimeType(path))

						// send file path argument to /update endpoint
						if _, running := server.ServerIsRunnning(); running {
							port := config.Port()
							url := fmt.Sprintf("http://localhost:%v/%s", port, "update")
							postBody, _ := json.Marshal(map[string]string{
								"file": path,
							})
							responseBody := bytes.NewBuffer(postBody)
							resp, err := http.Post(url, "application/json", responseBody)
							if err != nil {
								fmt.Println(err)
							}
							defer resp.Body.Close()
						} else {
							// fmt.Println("Cannon server is not running. Use --start or --toggle to start it.")
						}
					}
				}

				// lf requires a non-zero return value to disable caching
				_, exit := config.Exit().Int()
				os.Exit(exit)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	// normal cleanup
	cache.Exit()
}
