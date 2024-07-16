package config

import (
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/ccammack/cannon/util"

	"github.com/adrg/xdg"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	config     *Config
	configLock = new(sync.RWMutex)
	callbacks  []func(string)
)

type PlatformCommand struct {
	// platform names must match the $GOOS list in https://go.dev/doc/install/source#environment
	Default   []string `mapstructure:"default"`
	Aix       []string `mapstructure:"aix,omitempty"`
	Android   []string `mapstructure:"android,omitempty"`
	Darwin    []string `mapstructure:"darwin,omitempty"`
	Dragonfly []string `mapstructure:"dragonfly,omitempty"`
	Freebsd   []string `mapstructure:"freebsd,omitempty"`
	Illumos   []string `mapstructure:"illumos,omitempty"`
	Ios       []string `mapstructure:"ios,omitempty"`
	Js        []string `mapstructure:"js,omitempty"`
	Linux     []string `mapstructure:"linux,omitempty"`
	Netbsd    []string `mapstructure:"netbsd,omitempty"`
	Openbsd   []string `mapstructure:"openbsd,omitempty"`
	Plan9     []string `mapstructure:"plan9,omitempty"`
	Solaris   []string `mapstructure:"solaris,omitempty"`
	Windows   []string `mapstructure:"windows,omitempty"`
}

func GetPlatformCommand(platformCommand PlatformCommand) (string, []string) {
	platform := strings.Title(runtime.GOOS)
	value := reflect.Indirect(reflect.ValueOf(platformCommand)).FieldByName(platform)
	if value.IsValid() && !value.IsZero() && !value.IsNil() {
		// return the platform-specific value that matches runtime.GOOS
		slice, ok := value.Interface().([]string)
		if !ok {
			panic("value not a []string")
		}
		if len(slice) > 0 {
			return runtime.GOOS + ":", slice
		}
	}
	return "default:", platformCommand.Default
}

type Config struct {
	Settings struct {
		Server   string          `mapstructure:"server"`
		Port     int             `mapstructure:"port"`
		Browser  PlatformCommand `mapstructure:"browser"`
		Interval int             `mapstructure:"interval"`
		Precache int             `mapstructure:"precache"`
		Mime     PlatformCommand `mapstructure:"mime,omitempty"`
		Exit     int             `mapstructure:"exit"`
	} `mapstructure:"settings"`
	FileConversionRules []struct {
		Ext     []string        `mapstructure:"ext,omitempty"`
		Mime    []string        `mapstructure:"mime,omitempty"`
		Tag     string          `mapstructure:"tag"`
		Command PlatformCommand `mapstructure:"command,omitempty"`
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
	viper.AddConfigPath(xdg.Home + "/.config/cannon") // allow ~/.config on windows
	viper.AddConfigPath(xdg.ConfigHome + "/cannon")   // default xdg locations on all platforms
	dir, err := util.Dirname()
	if err == nil {
		viper.AddConfigPath(dir + "/..") // search development config location last
	}
	viper.OnConfigChange(func(e fsnotify.Event) {
		// reload and notify subscribers
		err := loadConfig()
		util.CheckPanic(err)
		for _, callback := range callbacks {
			callback("reload")
		}
	})
	viper.WatchConfig()
	err = loadConfig()
	util.CheckPanic(err)
}
