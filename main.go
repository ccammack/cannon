package main

import (
        "fmt"
        "log"
        "io/ioutil"
        "sync"
        "crypto/md5"
	"encoding/hex"
        "gopkg.in/yaml.v3"
	"github.com/fsnotify/fsnotify"
)

var (
        config_hash string
        mu          sync.Mutex
        config      T
)

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

func reload_config() {
    // Create new watcher.
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Fatal(err)
    }
    defer watcher.Close()

    // Start listening for events.
    go func() {
        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return
                }
                log.Println("event:", event)
                if event.Has(fsnotify.Write) {
                    log.Println("modified file:", event.Name)
                }
                load_config()
            case err, ok := <-watcher.Errors:
                if !ok {
                    return
                }
                log.Println("error:", err)
            }
        }
    }()

    // Add a path.
    err = watcher.Add(get_config_path())
    if err != nil {
        log.Fatal(err)
    }

    // Block main goroutine forever.
    <-make(chan struct{})
}

func load_config() {
        data, err := ioutil.ReadFile(get_config_path())
        if err != nil {
                log.Fatalf("data.Get err #%v", err)
        }

	md5HashInBytes := md5.Sum([]byte(data))
	md5HashInString := hex.EncodeToString(md5HashInBytes[:])
	if (config_hash != md5HashInString) {
                mu.Lock()
                defer mu.Unlock()
		config_hash = md5HashInString 
                err = yaml.Unmarshal([]byte(data), &config)
                if err != nil {
                        log.Fatalf("error: %v", err)
                }
                fmt.Printf("--- t:\n%v\n\n", config)
	}
}

func main() {
        load_config()
        reload_config()
}

