package main

import (
  "log"
  "os"
  "fmt"
  "sync"
  "github.com/spf13/viper"
  "github.com/fsnotify/fsnotify"
//  "github.com/mitchellh/mapstructure"
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

func loadConfig() {
  if err := viper.ReadInConfig(); err != nil {
    log.Println("open config: ", err)
    os.Exit(1)
  }
  fmt.Println(viper.AllSettings())
  temp := new(Config)
  if err := viper.Unmarshal(&temp); err != nil {
  //if err := mapstructure.Decode(viper.AllSettings(), &temp); err != nil {
  //if err := mapstructure.Decode(viper.GetStringMapString("file_conversion_rules"), temp); err != nil {
  //if err := mapstructure.Decode(viper.GetStringMap("file_conversion_rules"), temp); err != nil {
  //if err := mapstructure.Decode(viper.Get("file_conversion_rules"), temp); err != nil {
  //if err := mapstructure.Decode(viper.GetStringSlice("file_conversion_rules"), temp); err != nil {
      fmt.Println(err)
      log.Println("parse config: ", err)
      os.Exit(1)
  }

  fmt.Println(temp)
  fmt.Printf("--- t:\n%v\n\n", temp)
  configLock.Lock()
  config = temp
  fmt.Println(config)
  fmt.Printf("--- t:\n%v\n\n", config)
  configLock.Unlock()
  fmt.Println(viper.AllSettings())
}

func init() {
  viper.SetConfigType("yaml")
  viper.SetConfigName("config")
  viper.AddConfigPath("/home/ccammack/work/cannon")
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

