package main

import (
        "fmt"
        "log"
        "io/ioutil"

        "gopkg.in/yaml.v3"
)

var config = T{}

type T struct {
	FileConversionRules []struct {
		Type    string   `yaml:"type"`
		Matches []string `yaml:"matches"`
		Command string   `yaml:"command,omitempty"`
		Tag     string   `yaml:"tag"`
	} `yaml:"file_conversion_rules"`
}

func get_config_path() string {
        return "/home/ccammack/work/cannon/cannon.yml"
}

func load_config() {
        data, err := ioutil.ReadFile(get_config_path())
        if err != nil {
                log.Fatalf("data.Get err #%v", err)
        }

        err = yaml.Unmarshal([]byte(data), &config)
        if err != nil {
                log.Fatalf("error: %v", err)
        }
        fmt.Printf("--- t:\n%v\n\n", config)
}

func main() {
        load_config()
}

