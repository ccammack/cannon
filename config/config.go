/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

package config

import (
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	config     *Config
	configLock = new(sync.RWMutex)
	callbacks  []func(string)
)

type Config struct {
	Settings struct {
		Server   string   `mapstructure:"server"`
		Port     int      `mapstructure:"port"`
		Browser  []string `mapstructure:"browser"`
		Interval int      `mapstructure:"interval"`
		Precache int      `mapstructure:"precache"`
		Exit     int      `mapstructure:"exit"`
	} `mapstructure:"settings"`
	FileConversionRules []struct {
		Type    string   `mapstructure:"type"`
		Matches []string `mapstructure:"matches"`
		Tag     string   `mapstructure:"tag"`
		Command string   `mapstructure:"command,omitempty"`
	} `mapstructure:"file_conversion_rules"`
}

func GetConfig() *Config {
	configLock.RLock()
	defer configLock.RUnlock()
	return config
}

func RegisterCallback(callback func(string)) {
	callbacks = append(callbacks, callback)
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

func init() {
	// load config file
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/home/ccammack/work/cannon")
	viper.OnConfigChange(func(e fsnotify.Event) {
		// reload and notify subscribers
		loadConfig()
		for _, callback := range callbacks {
			callback("reload")
		}
	})
	viper.WatchConfig()
	if err := loadConfig(); err != nil {
		panic(err)
	}
}
