package util

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/exp/constraints"
)

func RespondJson(w http.ResponseWriter, data map[string]interface{}) {
	// json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
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

func Filename() (string, error) {
	// Filename is the __filename equivalent
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", errors.New("unable to get the current filename")
	}
	return filename, nil
}

func Dirname() (string, error) {
	// Dirname is the __dirname equivalent
	filename, err := Filename()
	if err != nil {
		return "", err
	}
	return filepath.Dir(filename), nil
}

func GetFileLength(file string) (int64, error) {
	info, err := os.Stat(file)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func GetFileBytes(file string, length int64) ([]byte, int64, error) {
	buffer := make([]byte, length)
	fp, err := os.Open(file)
	if err != nil {
		return buffer, 0, err
	}
	defer fp.Close()
	count, err := fp.Read(buffer)
	if err != nil && err != io.EOF {
		return buffer, int64(count), err
	}
	return buffer, int64(count), nil
}

func MakeHash(s string) string {
	hash := md5.New()
	hash.Write([]byte(s))
	hashstr := hex.EncodeToString(hash.Sum(nil))
	return hashstr
}

func IsBinaryFile(file string) ([]byte, int, bool) {
	// treat the file as binary if it contains a NUL in the first 4096 bytes
	fp, err := os.Open(file)
	if err != nil {
		log.Printf("error opening file: %v", err)
	}

	fs, err := fp.Stat()
	if err != nil {
		log.Printf("error getting info: %v", err)
	}

	b := make([]byte, Min(4096, fs.Size()))
	n, err := fp.Read(b)
	if err != nil {
		log.Printf("error reading file: %v", err)
	}

	for i := 0; i < n; i++ {
		if b[i] == '\x00' {
			return b, int(fs.Size()), true
		}
	}
	return b, int(fs.Size()), false
}

func HashPath(file string) (string, string, error) {
	path, err := filepath.Abs(file)
	if err != nil {
		log.Printf("Error generating absolute path: %v", err)
		return "", "", err
	}
	fp, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return "", "", err
	}
	defer fp.Close()
	hash := MakeHash(path)
	return hash, path, nil
}

func CopyFileContents(src, dst string) (err error) {
	// https://stackoverflow.com/a/21067803
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func CopyFileContentsAsync(src, dst string) chan error {
	ch := make(chan error)
	go func() {
		err := CopyFileContents(src, dst)
		ch <- err
	}()
	return ch
}
