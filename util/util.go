/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

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

func RespondJson(w *http.ResponseWriter, jsonMap map[string]string) {
	// TODO: try using templates for this or find a one-liner
	body, _ := json.Marshal(jsonMap)
	if w != nil {
		(*w).Header().Set("Content-Type", "application/json")
		(*w).WriteHeader(http.StatusOK)
		(*w).Write(body)
	} else {
		fmt.Fprintln(os.Stdout, string(body))
	}
}

func Find(a []string, x string) int {
	// https://yourbasic.org/golang/find-search-contains-slice/
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}
