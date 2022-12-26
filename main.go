package main

import (
  "log"
  "fmt"
  "sync"
  "github.com/spf13/viper"
  "github.com/fsnotify/fsnotify"
)

var (
  config *Config
  configLock = new(sync.RWMutex)
)

type Config struct {
  FileConversionRules []struct {
    Type    string   `mapstructure:"type"`
    Matches []string `mapstructure:"matches"`
    Command string   `mapstructure:"command,omitempty"`
    Tag     string   `mapstructure:"tag"`
  } `mapstructure:"file_conversion_rules"`
}

func GetConfig() *Config {
  configLock.RLock()
  defer configLock.RUnlock()
  return config
}

func loadConfig() {
  if err := viper.ReadInConfig(); err != nil {
    log.Println("open config: ", err)
    panic(err)
  }
  temp := new(Config)
  if err := viper.Unmarshal(&temp); err != nil {
      log.Println("parse config: ", err)
      panic(err)
  }
  configLock.Lock()
  config = temp
  configLock.Unlock()
  fmt.Println(config)
}

func init() {
  viper.SetConfigType("yaml")
  viper.SetConfigName("config")
  viper.AddConfigPath(".")
  viper.OnConfigChange(func(e fsnotify.Event) {
    fmt.Println("Config file changed:", e.Name)
    loadConfig()
  })
  viper.WatchConfig()
  loadConfig()
}

func main() {
  for {
  }
}

