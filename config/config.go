package config

import (
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"github.com/ccammack/cannon/util"
	"github.com/knadh/koanf/parsers/toml"
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
	util.CheckPanic(err)
	return strings.ToLower(hostname)
}

func key(key string) (string, bool) {
	hostKey := "host." + hostname() + "." + key
	if config.Exists(hostKey) {
		return hostKey, true
	}
	osKey := "os." + platform() + "." + key
	if config.Exists(osKey) {
		return osKey, true
	}
	if config.Exists(key) {
		return key, true
	}
	return "", false
}

func requiredKey(s string) string {
	key, ok := key(s)
	if !ok {
		log.Printf("error finding required key: %v", key)
	}
	return key
}

func requiredInt(key string) int {
	configLock.RLock()
	defer configLock.RUnlock()
	return config.Int(requiredKey(key))
}

func requiredStrings(key string) []string {
	configLock.RLock()
	defer configLock.RUnlock()
	return config.Strings(requiredKey(key))
}

func Port() int         { return requiredInt("port") }
func Interval() int     { return requiredInt("interval") }
func Exit() int         { return requiredInt("exit") }
func Mime() []string    { return requiredStrings("mime") }
func Browser() []string { return requiredStrings("browser") }

func RegisterCallback(callback func(string)) {
	callbacks = append(callbacks, callback)
}

func init() {
	// load config file
	f := file.Provider(xdg.ConfigHome + "/cannon/cannon.toml")
	if err := config.Load(f, toml.Parser()); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	// watch for config file changes and reload
	f.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Printf("watch error: %v", err)
			return
		}

		// reload config file
		tmp := koanf.New(".")
		if err := tmp.Load(f, toml.Parser()); err != nil {
			log.Printf("error loading config: %v", err)
			return
		}

		// update config
		configLock.Lock()
		config = tmp
		configLock.Unlock()

		// notify subscribers
		for _, callback := range callbacks {
			callback("reload")
		}
	})
}
