package main

import (
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

func loadConfig() error {
  if err := viper.ReadInConfig(); err != nil {
    return err
  }
  temp := new(Config)
  if err := viper.Unmarshal(&temp); err != nil {
    return err
  }
  configLock.Lock()
  config = temp
  configLock.Unlock()
  return nil
}

func main() {
  // load config file
  viper.SetConfigType("yaml")
  viper.SetConfigName("config")
  viper.AddConfigPath(".")
  viper.OnConfigChange(func(e fsnotify.Event) {
    loadConfig()
  })
  viper.WatchConfig()
  if err := loadConfig(); err != nil {
    panic(err)
  }

  // start web server
  for {
  }
}

