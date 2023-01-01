/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"cannon/server"
	"cannon/util"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	start  *bool
	stop   *bool
	toggle *bool
)

var rootCmd = &cobra.Command{
	Use:   "cannon",
	Short: "Cannon is a brute-force previewer for terminal file managers",
	Long: `Cannon is a brute-force previewer for terminal file managers like these:

	https://github.com/dylanaraps/fff
	https://github.com/gokcehan/lf
	https://github.com/jarun/nnn
	https://github.com/ranger/ranger

It uses rules defined in the configuration file to convert each selected
file into its web-standard equivalent and then displays the converted file
in a web browser using a static http server.`,
	Run: func(cmd *cobra.Command, args []string) {
		if *start {
			server.Start()
		} else if *stop {
			server.Stop()
		} else if *toggle {
			server.Toggle()
		} else if len(args) > 0 {
			// fmt.Println(args[0])
			// log.Fatal(args[0])
			// util.Append(args[0])
			util.Append(fmt.Sprintf("%v", args))

			// TODO: figure out how to handle this for all console viewers
			os.Exit(255) // return non-zero exit code to disable preview cache
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	start = rootCmd.Flags().BoolP("start", "s", false, "Start the preview server")
	stop = rootCmd.Flags().BoolP("stop", "p", false, "Stop the preview server")
	toggle = rootCmd.Flags().BoolP("toggle", "t", false, "Toggle the preview server on and off")
}
