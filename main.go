package main

import (
        "fmt"
        "log"

        "gopkg.in/yaml.v3"
)


var data = `
file_conversion_rules:
  -
    type: extension
    matches: [mp3]
    tag: blabla-hostname
  -
    type: extension
    matches: [rule1]
    command: blablabla-host
    tag: blabla-hostname
  -
    type: extension
    matches: [mp4, jpg]
    command: ls -alg
    tag: video
  -
    type: mime
    matches: [application/text]
    command: more t.txt
    tag: head
`

type T struct {
	FileConversionRules []struct {
		Type    string   `yaml:"type"`
		Matches []string `yaml:"matches"`
		Command string   `yaml:"command,omitempty"`
		Tag     string   `yaml:"tag"`
	} `yaml:"file_conversion_rules"`
}

func main() {
        t := T{}

        err := yaml.Unmarshal([]byte(data), &t)
        if err != nil {
                log.Fatalf("error: %v", err)
        }
        fmt.Printf("--- t:\n%v\n\n", t)

}

