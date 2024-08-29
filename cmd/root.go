package cmd

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"os"
// 	"path/filepath"

// 	"github.com/ccammack/cannon/cache"
// 	"github.com/ccammack/cannon/config"
// 	"github.com/ccammack/cannon/server"

// 	"github.com/spf13/cobra"
// )

// var (
// 	start  *bool
// 	stop   *bool
// 	toggle *bool
// 	page   *bool
// 	status *bool
// )

// var rootCmd = &cobra.Command{
// 	Use:   "cannon",
// 	Short: "Cannon is a brute-force previewer for terminal file managers",
// 	Long: `Cannon is a brute-force previewer for terminal file managers like these:

// 	https://github.com/dylanaraps/fff
// 	https://github.com/gokcehan/lf
// 	https://github.com/jarun/nnn
// 	https://github.com/ranger/ranger

// It uses rules defined in the configuration file to convert each selected
// file into its web-standard equivalent and then displays the converted file
// in a web browser using a static http server.`,
// 	Run: func(cmd *cobra.Command, args []string) {
// 		if *start {
// 			server.Start()
// 		} else if *stop {
// 			server.Stop()
// 		} else if *toggle {
// 			server.Toggle()
// 		} else if *page {
// 			server.Page()
// 		} else if *status {
// 			server.Status()
// 		} else if len(args) > 0 {
// 			path, err := filepath.Abs(args[0])
// 			if err != nil {
// 				fmt.Println(err)
// 			} else {
// 				fp, err := os.Open(path)
// 				defer fp.Close()
// 				if err != nil {
// 					fmt.Println(err)
// 				} else {
// 					// write mime type to stdout for display in the right pane
// 					fmt.Println(cache.GetMimeType(path))

// 					// send file path argument to /update endpoint
// 					if _, running := server.ServerIsRunnning(); running {
// 						port := config.Port()
// 						url := fmt.Sprintf("http://localhost:%v/%s", port, "update")
// 						postBody, _ := json.Marshal(map[string]string{
// 							"file": path,
// 						})
// 						responseBody := bytes.NewBuffer(postBody)
// 						resp, err := http.Post(url, "application/json", responseBody)
// 						if err != nil {
// 							fmt.Println(err)
// 						}
// 						defer resp.Body.Close()
// 					} else {
// 						// fmt.Println("Cannon server is not running. Use --start or --toggle to start it.")
// 					}
// 				}
// 			}

// 			// lf requires a non-zero return value to disable caching
// 			_, exit := config.Exit().Int()
// 			os.Exit(exit)
// 		}
// 	},
// }

// func Execute() {
// 	err := rootCmd.Execute()
// 	if err != nil {
// 		os.Exit(1)
// 	}
// }

// func init() {
// 	start = rootCmd.Flags().BoolP("start", "s", false, "Start the preview server")
// 	stop = rootCmd.Flags().BoolP("stop", "p", false, "Stop the preview server")
// 	toggle = rootCmd.Flags().BoolP("toggle", "t", false, "Toggle the preview server on and off")
// 	page = rootCmd.Flags().BoolP("page", "g", false, "Display the current page HTML for testing")
// 	status = rootCmd.Flags().BoolP("status", "u", false, "Display the server status for testing")
// }
