package main

import (
	"io"
	"log"
	"os"
	"time"

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
		HelpName:  "cannond",
		Usage:     "receive a filename from the Cannon client and display it in the web browser.",
		UsageText: "cannond [OPTION]... command",
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
		},
		Commands: []*cli.Command{
			{
				Name:    "start",
				Aliases: []string{"s"},
				Usage:   "start the preview server",
				Action: func(cCtx *cli.Context) error {
					server.Start()
					return nil
				},
			},
			{
				Name:    "stop",
				Aliases: []string{"p"},
				Usage:   "stop the preview server",
				Action: func(cCtx *cli.Context) error {
					server.Stop()
					return nil
				},
			},
			{
				Name:    "toggle",
				Aliases: []string{"t"},
				Usage:   "toggle the preview server",
				Action: func(cCtx *cli.Context) error {
					server.Toggle()
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
