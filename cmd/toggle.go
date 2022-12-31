/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"cannon/server"

	"github.com/spf13/cobra"
)

// toggleCmd represents the toggle command
var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle the preview server",
	Long: `cannon toggle

Toggle the preview server on and off.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("toggle called")
		server.Toggle()
	},
}

func init() {
	rootCmd.AddCommand(toggleCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// toggleCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// toggleCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
