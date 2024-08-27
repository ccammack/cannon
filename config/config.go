package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"github.com/ccammack/cannon/util"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var (
	config     = koanf.New(".")
	configLock = new(sync.RWMutex)
	callbacks  []func(string)
)

func platform() string { return strings.ToLower(runtime.GOOS) }

func hostname() string {
	hostname, err := os.Hostname()
	util.CheckPanic(err, "error reading hostname")
	return strings.ToLower(hostname)
}

func key(key string, ko *koanf.Koanf) (string, error) {
	hostKey := "host." + hostname() + "." + key
	if ko.Exists(hostKey) {
		return hostKey, nil
	}
	osKey := "os." + platform() + "." + key
	if ko.Exists(osKey) {
		return osKey, nil
	}
	defaultKey := key
	if ko.Exists(defaultKey) {
		return defaultKey, nil
	}
	return "", errors.New(key)
}

func requiredInt(s string, ko *koanf.Koanf) int {
	key, err := key(s, ko)
	util.CheckPanic(err, "error finding required key")
	return ko.Int(key)
}

func requiredStrings(s string, ko *koanf.Koanf) []string {
	key, err := key(s, ko)
	util.CheckPanic(err, "error finding required key")
	return ko.Strings(key)
}

func optionalString(s string, ko *koanf.Koanf) string {
	key, err := key(s, ko)
	if err != nil {
		return ""
	}
	return ko.String(key)
}

func optionalStrings(s string, ko *koanf.Koanf) []string {
	key, err := key(s, ko)
	if err != nil {
		return nil
	}
	return ko.Strings(key)
}

func Port() int         { return requiredInt("port", config) }
func Interval() int     { return requiredInt("interval", config) }
func Exit() int         { return requiredInt("exit", config) }
func Mime() []string    { return requiredStrings("mime", config) }
func Browser() []string { return requiredStrings("browser", config) }

type FileConversionRule struct {
	Ext []string
	Mim []string
	Dep []string
	Msg string
	Cmd []string
	Tag string
}

func Rules() []FileConversionRule {
	// find the highest priority rule set prefix
	prefix, err := key("rules", config)
	if err != nil {
		return nil
	}

	configLock.RLock()
	defer configLock.RUnlock()

	rules := []FileConversionRule{}
	for _, v := range config.Slices(prefix) {
		ext := slices.Clone(optionalStrings("ext", v))
		mim := slices.Clone(optionalStrings("mim", v))
		dep := slices.Clone(optionalStrings("dep", v))
		msg := strings.Clone(optionalString("msg", v))
		cmd := slices.Clone(optionalStrings("cmd", v))
		tag := strings.Clone(optionalString("tag", v))
		rule := FileConversionRule{ext, mim, dep, msg, cmd, tag}
		rules = append(rules, rule)
	}

	return rules
}

func RegisterCallback(callback func(string)) {
	callbacks = append(callbacks, callback)
}

func afterLoad() {
	// redirect log output to logfile if defined
	fname := optionalString("logfile", config)
	if fname != "" {
		file, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Printf("error setting logfile: %v", err)
		} else {
			log.SetOutput(file)
		}
	}
}

func init() {
	// load config file
	f := file.Provider(xdg.ConfigHome + "/cannon/cannon.toml.yml")
	err := config.Load(f, yaml.Parser())
	util.CheckPanic(err, "error loading config")

	afterLoad()

	fmt.Printf("%v\n", Rules())

	// watch for config file changes and reload
	f.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Printf("watch error: %v", err)
			return
		}

		// reload config file
		tmp := koanf.New(".")
		if err := tmp.Load(f, yaml.Parser()); err != nil {
			log.Printf("error loading config: %v", err)
			return
		}

		// update config
		configLock.Lock()
		config = tmp
		configLock.Unlock()

		afterLoad()

		// notify subscribers
		for _, callback := range callbacks {
			callback("reload")
		}
	})
}
