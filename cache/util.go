package cache

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"
)

func GetMimeType(file string) string {
	_, command := config.Mime().Strings()
	if len(command) > 0 {
		cmd, args := util.FormatCommand(command, map[string]string{"{input}": file})
		out, _ := exec.Command(cmd, args...).CombinedOutput()
		return strings.TrimSuffix(string(out), "\n")
	}
	return ""
}

func findMatchingOutputFile(output string) string {
	// find newly created files that match output*
	// TODO: add a vars block to the YAML and remove this function
	matches, err := filepath.Glob(output + "*")
	if err != nil {
		log.Printf("Error matching filename %s: %v", output, err)
	}
	if len(matches) > 2 {
		log.Printf("Error matched too many files for %s: %v", output, matches)
	}
	for _, match := range matches {
		if len(match) > len(output) {
			output = match
		}
	}
	return output
}

func runAndWait(resource *Resource, rule ConversionRule) int {
	cmd, args := util.FormatCommand(rule.cmd, map[string]string{
		"{input}":  resource.file,
		"{output}": resource.tmpOutputFile,
	})

	resource.progress = append(resource.progress, fmt.Sprintf("Run command: %v %v", cmd, args))

	// timeout
	_, timeout := config.Timeout().Int()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	// prepare command
	var outb, errb bytes.Buffer
	command := exec.CommandContext(ctx, cmd, args...)
	command.Stdout = &outb
	command.Stderr = &errb

	// run command
	err := command.Run()
	resource.stdout = outb.String()
	resource.stderr = errb.String()

	// fail if the command takes too long
	if ctx.Err() == context.DeadlineExceeded {
		resource.progress = append(resource.progress, "Command timed out!")
		return 255
	}

	// collect and return exit code
	exit := 0
	if err != nil {
		// there was an error
		exit = 255

		// extract the actual error
		if exitError, ok := err.(*exec.ExitError); ok {
			exit = exitError.ExitCode()
		}
	}

	return exit
}

func createPreviewFile(tempDir string) string {
	// create a temp file to hold the output preview file
	fp, err := os.CreateTemp(tempDir, "preview")
	if err != nil {
		log.Panicf("error creating temp file: %v", err)
	}
	defer fp.Close()
	return fp.Name()
}
