package main

import (
        "fmt"
        //"os"
        "log"
        //"path/filepath"
        "io/ioutil"

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

func get_config_path() string {
        // ex, err := os.Executable()
        // if err != nil {
        //         panic(err)
        // }
        // exPath := filepath.Dir(ex)
        // fmt.Println(exPath)
        // return exPath
        return "/home/ccammack/work/cannon/cannon.yml"
}

func load_config() {
        configPath := get_config_path()
        configData, err := ioutil.ReadFile(configPath)
        if err != nil {
                log.Printf("configData.Get err   #%v ", err)
        }

        t := T{}

        //err := yaml.Unmarshal([]byte(data), &t)
        err = yaml.Unmarshal([]byte(configData), &t)
        if err != nil {
                log.Fatalf("error: %v", err)
        }
        fmt.Printf("--- t:\n%v\n\n", t)
}

func main() {
        load_config()
}

