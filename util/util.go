package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/exp/constraints"
)

var (
	filename = "/home/ccammack/work/cannon/log.txt"
)

func Append(text string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	CheckPanic(err)
	defer f.Close()
	_, err = f.WriteString(text + "\n")
	CheckPanic(err)
}

func RespondJson(w *http.ResponseWriter, jsonMap map[string]template.HTML) {
	if w != nil {
		(*w).Header().Set("Content-Type", "application/json")
		(*w).WriteHeader(http.StatusOK)
		json.NewEncoder(*w).Encode(jsonMap)
	} else {
		json.NewEncoder(os.Stdout).Encode(jsonMap)
	}
}

func Find(a []string, x string) int {
	if len(x) > 0 {
		for i, n := range a {
			if len(n) > 0 && (strings.Contains(n, x) || strings.Contains(x, n)) {
				return i
			}
		}
	}
	return len(a)
}

func DumpRequest(r *http.Request) {
	// TODO: save this info in reference.org
	res, error := httputil.DumpRequest(r, true)
	if error != nil {
		log.Fatal(error)
	}
	fmt.Print(string(res))
	// Append(string(res))
}

func FormatCommand(commandArr []string, subs map[string]string) (string, []string) {
	command := commandArr[0]
	rest := commandArr[1:]
	args := []string{}
	for _, arg := range rest {
		for k, v := range subs {
			arg = strings.ReplaceAll(arg, k, v)
		}
		args = append(args, arg)
	}
	return command, args
}

func Min[T constraints.Ordered](args ...T) T {
	min := args[0]
	for _, x := range args {
		if x < min {
			min = x
		}
	}
	return min
}

func Max[T constraints.Ordered](args ...T) T {
	max := args[0]
	for _, x := range args {
		if x > max {
			max = x
		}
	}
	return max
}

// Filename is the __filename equivalent
func Filename() (string, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", errors.New("unable to get the current filename")
	}
	return filename, nil
}

// Dirname is the __dirname equivalent
func Dirname() (string, error) {
	filename, err := Filename()
	if err != nil {
		return "", err
	}
	return filepath.Dir(filename), nil
}

func CopyFile(input string, output string) {
	// copy input file contents to output file
	data, err := ioutil.ReadFile(input)
	CheckPanic(err)
	err = ioutil.WriteFile(output, data, 0644)
	CheckPanic(err)
}

func CheckPanic(err error) {
	if err != nil {
		panic(err)
	}
}
