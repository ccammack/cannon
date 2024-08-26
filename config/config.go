package config

import (
	"errors"
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
	util.CheckPanic2(err, "error reading hostname")
	return strings.ToLower(hostname)
}

func key(key string) (string, error) {
	hostKey := "host." + hostname() + "." + key
	if config.Exists(hostKey) {
		return hostKey, nil
	}
	osKey := "os." + platform() + "." + key
	if config.Exists(osKey) {
		return osKey, nil
	}
	if config.Exists(key) {
		return key, nil
	}
	return "", errors.New(key)
}

func requiredInt(s string) int {
	configLock.RLock()
	defer configLock.RUnlock()
	key, err := key(s)
	util.CheckPanic2(err, "error finding required key")
	return config.Int(key)
}

func requiredStrings(s string) []string {
	configLock.RLock()
	defer configLock.RUnlock()
	key, err := key(s)
	util.CheckPanic2(err, "error finding required key")
	return config.Strings(key)
}

func optionalString(s string) string {
	configLock.RLock()
	defer configLock.RUnlock()
	key, err := key(s)
	if err != nil {
		return ""
	}
	return config.String(key)
}

func Port() int         { return requiredInt("port") }
func Interval() int     { return requiredInt("interval") }
func Exit() int         { return requiredInt("exit") }
func Mime() []string    { return requiredStrings("mime") }
func Browser() []string { return requiredStrings("browser") }

func RegisterCallback(callback func(string)) {
	callbacks = append(callbacks, callback)
}

func afterLoad() {
	// redirect log output to logfile if defined
	fname := optionalString("logfile")
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
	f := file.Provider(xdg.ConfigHome + "/cannon/cannon.toml")
	err := config.Load(f, toml.Parser())
	util.CheckPanic2(err, "error loading config")

	afterLoad()

	log.Println("This is a log message")
	log.Fatal("This is a fatal message")

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

		afterLoad()

		// notify subscribers
		for _, callback := range callbacks {
			callback("reload")
		}
	})
}
