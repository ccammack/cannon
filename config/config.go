package config

import (
	"errors"
	"fmt"
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
	// configLock.RLock()
	// defer configLock.RUnlock()
	key, err := key(s, ko)
	util.CheckPanic2(err, "error finding required key")
	return ko.Int(key)
}

func requiredStrings(s string, ko *koanf.Koanf) []string {
	// configLock.RLock()
	// defer configLock.RUnlock()
	key, err := key(s, ko)
	util.CheckPanic2(err, "error finding required key")
	return ko.Strings(key)
}

func optionalString(s string, ko *koanf.Koanf) string {
	// configLock.RLock()
	// defer configLock.RUnlock()
	key, err := key(s, ko)
	if err != nil {
		return ""
	}
	return ko.String(key)
}

func optionalStrings(s string, ko *koanf.Koanf) []string {
	// configLock.RLock()
	// defer configLock.RUnlock()
	key, err := key(s, ko)
	if err != nil {
		return nil
	}
	return ko.Strings(key)
}

// func optionalInterface(s string) interface{} {
// 	// configLock.RLock()
// 	// defer configLock.RUnlock()
// 	key, err := key(s)
// 	if err != nil {
// 		return nil
// 	}
// 	return config.Get(key)
// }

func Port() int         { return requiredInt("port", config) }
func Interval() int     { return requiredInt("interval", config) }
func Exit() int         { return requiredInt("exit", config) }
func Mime() []string    { return requiredStrings("mime", config) }
func Browser() []string { return requiredStrings("browser", config) }

type FileConversionRules []struct {
	Ext     []string
	Mime    []string
	Deps    []string
	Command []string
	Tag     string
}

func Rules() interface{} {
	// find the highest priority rule set prefix
	prefix, err := key("rules", config)
	if err != nil {
		return nil
	}

	configLock.RLock()
	defer configLock.RUnlock()

	// iface := config.Get(prefix)
	// entries := iface.([]interface{})
	// fmt.Printf("%s: type: %v, length: %d, value: %v\n", prefix, reflect.TypeOf(iface), len(entries), iface)

	slices := config.Slices(prefix)
	fmt.Printf("%v\n", slices)

	for _, v := range slices {
		ext := optionalStrings("ext", v)
		mim := optionalStrings("mim", v)
		dep := optionalStrings("dep", v)
		msg := optionalStrings("msg", v)
		cmd := optionalStrings("cmd", v)
		tag := optionalString("tag", v)
		fmt.Printf("%v %v %v %v %v %s\n", ext, mim, dep, msg, cmd, tag)
	}

	// tmp := koanf.New(".")
	// err := tmp.Load(iface, toml.Parser())

	// iterate the rule interface here
	// for i := 0; i != len(entries); i++ {
	// 	ith := strconv.Itoa(i)
	// 	ext := optionalStrings(prefix + ith + ".ext")
	// 	mime := optionalStrings(prefix + ith + ".mime")
	// 	deps := optionalStrings(prefix + ith + ".deps")
	// 	cmd := optionalStrings(prefix + ith + ".cmd")
	// 	tag := optionalString(prefix + ith + ".tag")
	// 	fmt.Printf("%v %v %v %v %s\n", ext, mime, deps, cmd, tag)
	// }

	return nil
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
	f := file.Provider(xdg.ConfigHome + "/cannon/cannon.toml")
	err := config.Load(f, toml.Parser())
	util.CheckPanic2(err, "error loading config")

	afterLoad()

	// log.Println("This is a log message")
	// log.Fatal("This is a fatal message")

	Rules()

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
