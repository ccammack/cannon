/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package util

import "os"

var (
	filename = "/home/ccammack/work/cannon/log.txt"
)

func Append(text string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(text + "\n"); err != nil {
		panic(err)
	}
}
