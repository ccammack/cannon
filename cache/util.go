package cache

import (
	"crypto/md5"
	"encoding/hex"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"
	"golang.org/x/exp/maps"
)

// max display length for unknown file types
const byteLength = 4096

func GetMimeType(file string) string {
	_, command := config.Mime().Strings()
	if len(command) > 0 {
		cmd, args := util.FormatCommand(command, map[string]string{"{input}": file})
		out, _ := exec.Command(cmd, args...).CombinedOutput()
		return strings.TrimSuffix(string(out), "\n")
	}
	return ""
}

func makeHash(s string) string {
	hash := md5.New()
	hash.Write([]byte(s))
	hashstr := hex.EncodeToString(hash.Sum(nil))
	return hashstr
}

func isBinaryFile(file string) ([]byte, int, bool) {
	// treat the file as binary if it contains a NUL anywhere in the first byteLength bytes
	fp, err := os.Open(file)
	util.CheckPanicOld(err)
	fs, err := fp.Stat()
	util.CheckPanicOld(err)
	b := make([]byte, util.Min(byteLength, fs.Size()))
	n, err := fp.Read(b)
	util.CheckPanicOld(err)
	for i := 0; i < n; i++ {
		if b[i] == '\x00' {
			return b, int(fs.Size()), true
		}
	}
	return b, int(fs.Size()), false
}

func getFileWithExtension(file string) string {
	// find the file matching pattern with the longest name
	matches, err := filepath.Glob(file + "*")
	util.CheckPanicOld(err)
	for _, match := range matches {
		if len(match) > len(file) {
			file = match
		}
	}
	return file
}

func formatRawFileElement(file string) string {
	bytes, size, _ := isBinaryFile(file)
	s := string(bytes)
	html := "<xmp>" + s + "\n\n"
	if size > byteLength {
		html += "[...]"
	}
	html += "</xmp>"
	return html
}

func formatRawStringElement(raw string) string {
	size := len(raw)
	bytes := raw[0:util.Min(size, byteLength)]
	s := string(bytes)
	html := "<xmp>" + s + "\n\n"
	if size > byteLength {
		html += "[...]"
	}
	html += "</xmp>"
	return html
}

func formatCurrentResourceData() map[string]template.HTML {
	// return the current resource for display

	// set default values
	_, interval := config.Interval().String()

	data := map[string]template.HTML{
		"interval": template.HTML(interval),
	}

	// look up the current resource if it exists
	// resource, ok := getResource(getCurrentHash())
	resource, ok := Resource{}, false
	if !ok {
		// serve default values until the first resource is added
		html := "<p>Waiting for file...</p>"
		maps.Copy(data, map[string]template.HTML{
			"title":    "Cannon preview",
			"html":     template.HTML(html),
			"htmlhash": template.HTML(makeHash(html)),
		})
	} else {
		if !resource.ready {
			// serve a spinner until ready is true - https://codepen.io/nikhil8krishnan/pen/rVoXJa
			maps.Copy(data, map[string]template.HTML{
				"title":    template.HTML(filepath.Base(resource.inputName)),
				"html":     template.HTML(SpinnerTemplate),
				"htmlhash": template.HTML(makeHash(SpinnerTemplate)),
			})
		} else {
			// serve the converted output file (or error text on failure)
			maps.Copy(data, map[string]template.HTML{
				"title":    template.HTML(filepath.Base(resource.inputName)),
				"html":     template.HTML(resource.html),
				"htmlhash": template.HTML(resource.htmlHash),
			})
		}
	}

	return data
}
