/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

package config

import (
	"sync"

	"github.com/adrg/xdg"
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
		Server  string `mapstructure:"server"`
		Port    int    `mapstructure:"port"`
		Browser struct {
			Default []string `mapstructure:"default,omitempty"`
			// Aix       []string `mapstructure:"aix,omitempty"`
			// Android   []string `mapstructure:"android,omitempty"`
			// Darwin    []string `mapstructure:"darwin,omitempty"`
			// Dragonfly []string `mapstructure:"dragonfly,omitempty"`
			// Freebsd   []string `mapstructure:"freebsd,omitempty"`
			// Illumos   []string `mapstructure:"illumos,omitempty"`
			// Ios       []string `mapstructure:"ios,omitempty"`
			// Js        []string `mapstructure:"js,omitempty"`
			// Linux     []string `mapstructure:"linux,omitempty"`
			// Netbsd    []string `mapstructure:"netbsd,omitempty"`
			// Openbsd   []string `mapstructure:"openbsd,omitempty"`
			// Plan9     []string `mapstructure:"plan9,omitempty"`
			// Solaris   []string `mapstructure:"solaris,omitempty"`
			Windows []string `mapstructure:"windows,omitempty"`
		} `mapstructure:"browser,omitempty"`
		Interval int `mapstructure:"interval"`
		Precache int `mapstructure:"precache"`
		Exit     int `mapstructure:"exit"`
	} `mapstructure:"settings"`
	FileConversionRules []struct {
		Type    string   `mapstructure:"type"`
		Matches []string `mapstructure:"matches"`
		Tag     string   `mapstructure:"tag"`
		Command struct {
			Default []string `mapstructure:"default,omitempty"`
			// Aix       []string `mapstructure:"aix,omitempty"`
			// Android   []string `mapstructure:"android,omitempty"`
			// Darwin    []string `mapstructure:"darwin,omitempty"`
			// Dragonfly []string `mapstructure:"dragonfly,omitempty"`
			// Freebsd   []string `mapstructure:"freebsd,omitempty"`
			// Illumos   []string `mapstructure:"illumos,omitempty"`
			// Ios       []string `mapstructure:"ios,omitempty"`
			// Js        []string `mapstructure:"js,omitempty"`
			// Linux     []string `mapstructure:"linux,omitempty"`
			// Netbsd    []string `mapstructure:"netbsd,omitempty"`
			// Openbsd   []string `mapstructure:"openbsd,omitempty"`
			// Plan9     []string `mapstructure:"plan9,omitempty"`
			// Solaris   []string `mapstructure:"solaris,omitempty"`
			Windows []string `mapstructure:"windows,omitempty"`
		} `mapstructure:"command,omitempty"`
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
	// https: //github.com/gokcehan/lf
	// https: //pkg.go.dev/github.com/gokcehan/lf#hdr-Configuration
	// https: //github.com/gokcehan/lf/blob/master/etc/lfrc.example
	// https: //github.com/doronbehar/pistol

	// load config file
	viper.SetConfigType("yaml")
	viper.SetConfigName("cannon")
	viper.AddConfigPath(xdg.ConfigHome + "/cannon")
	viper.AddConfigPath("/home/ccammack/work/cannon")                // TODO: development only
	viper.AddConfigPath("C:/Users/clc/work/ccammack.private/cannon") // TODO: development only
	viper.OnConfigChange(func(e fsnotify.Event) {
		// reload and notify subscribers
		if err := loadConfig(); err != nil {
			panic(err)
		}
		for _, callback := range callbacks {
			callback("reload")
		}
	})
	viper.WatchConfig()
	if err := loadConfig(); err != nil {
		panic(err)
	}
}
