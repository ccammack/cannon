package main

import (
  "log"
  "fmt"
   "os"
  "os/signal"
  "syscall"
  "io/ioutil"
  "sync"
  "gopkg.in/yaml.v3"
)

var (
  config *Config
  configLock = new(sync.RWMutex)
)

type Config struct {
  FileConversionRules []struct {
    Type    string   `yaml:"type"`
    Matches []string `yaml:"matches"`
    Command string   `yaml:"command,omitempty"`
    Tag     string   `yaml:"tag"`
  } `yaml:"file_conversion_rules"`
}

func GetConfig() *Config {
  configLock.RLock()
  defer configLock.RUnlock()
  return config
}

func getConfigPath() string {
  return "/home/ccammack/work/cannon/cannon.yml"
}

func loadConfig(fail bool){
  file, err := ioutil.ReadFile(getConfigPath())
  if err != nil {
    log.Println("open config: ", err)
    if fail { os.Exit(1) }
  }

  temp := new(Config)
  if err = yaml.Unmarshal(file, temp); err != nil {
    log.Println("parse config: ", err)
    if fail { os.Exit(1) }
  }
  configLock.Lock()
  config = temp
  fmt.Printf("--- t:\n%v\n\n", config)
  configLock.Unlock()
}

func init() {
  loadConfig(true)
  s := make(chan os.Signal, 1)
  signal.Notify(s, syscall.SIGUSR2)
  go func() {
    for {
      <-s
      loadConfig(false)
      log.Println("Reloaded")
    }
  }()
}

func main() {
  for {
  }
}

