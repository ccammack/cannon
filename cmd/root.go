/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bytes"
	"cannon/config"
	"cannon/server"
	"cannon/util"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

			// read config and call /update endpoint
			// TODO: wrap this in a utility function
			port := config.GetConfig().Settings.Port
			url := fmt.Sprintf("http://localhost:%v/%s", port, "update")
			postBody, _ := json.Marshal(map[string]string{
				"file": args[0],
			})
			responseBody := bytes.NewBuffer(postBody)
			resp, err := http.Post(url, "application/json", responseBody)
			if err != nil {
				log.Fatalf("An Error Occured %v", err)
			}
			defer resp.Body.Close()

			// Read the response body
			//  body, err := ioutil.ReadAll(resp.Body)
			//  if err != nil {
			// 	log.Fatalln(err)
			//  }
			//  sb := string(body)
			//  log.Printf(sb)

			// TODO: define the proper return value for all viewers and add it to config
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
