package config

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"github.com/ccammack/cannon/gen"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var (
	config     = koanf.New(".")
	configLock = new(sync.RWMutex)
	callbacks  []func(string)
)

func platform() string {
	return strings.ToLower(runtime.GOOS)
}

func hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Panicf("error reading hostname: %v", err)
	}
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
	return key, errors.New(key)
}

// func required(s string, ko *koanf.Koanf) Pair {
// 	key, err := key(s, ko)
// 	if err != nil {
// 		log.Panicf("error finding required key: %v", err)
// 	}
// 	return Pair{key, ko.Get(key)}
// }

// func optional(s string, ko *koanf.Koanf) Pair {
// 	key, err := key(s, ko)
// 	if err != nil {
// 		return Pair{key, nil}
// 	}
// 	return Pair{key, ko.Get(key)}
// }

func requiredInt(s string, ko *koanf.Koanf) gen.Pair {
	key, err := key(s, ko)
	if err != nil {
		log.Panicf("error finding required key: %v", err)
	}
	s = ko.String(key)
	_, err = strconv.Atoi(s)
	if err != nil {
		log.Panicf("error converting required value to integer: %v", err)
	}
	return gen.Pair{K: key, V: s}
}

func requiredStrings(s string, ko *koanf.Koanf) gen.Pair {
	key, err := key(s, ko)
	if err != nil {
		log.Panicf("error finding required key: %v", err)
	}
	return gen.Pair{K: key, V: ko.Strings(key)}
}

func optionalString(s string, ko *koanf.Koanf) gen.Pair {
	key, err := key(s, ko)
	if err != nil {
		return gen.Pair{K: key, V: nil}
	}
	return gen.Pair{K: key, V: ko.String(key)}
}

func optionalStrings(s string, ko *koanf.Koanf) gen.Pair {
	key, err := key(s, ko)
	if err != nil {
		return gen.Pair{K: key, V: nil}
	}
	return gen.Pair{K: key, V: ko.Strings(key)}
}

func Port() gen.Pair     { return requiredInt("port", config) }
func Interval() gen.Pair { return requiredInt("interval", config) }
func Exit() gen.Pair     { return requiredInt("exit", config) }
func Mime() gen.Pair     { return requiredStrings("mime", config) }
func Browser() gen.Pair  { return optionalStrings("browser", config) }

type FileConversionRule struct {
	Ext gen.Pair
	Mim gen.Pair
	Dep gen.Pair
	Msg gen.Pair
	Cmd gen.Pair
	Tag gen.Pair
}

// TODO: make Rules() return a gen.Pair
func Rules() (string, []FileConversionRule) {
	// find the highest priority rule set
	key, err := key("rules", config)
	if err != nil {
		return "", nil
	}

	configLock.RLock()
	defer configLock.RUnlock()

	// clone the the rules
	rules := []FileConversionRule{}
	for _, v := range config.Slices(key) {
		ext := optionalStrings("ext", v)
		mim := optionalStrings("mim", v)
		dep := optionalStrings("dep", v)
		cmd := optionalStrings("cmd", v)

		msg := optionalString("msg", v)
		tag := optionalString("tag", v)

		rules = append(rules, FileConversionRule{ext, mim, dep, msg, cmd, tag})
	}

	// return gen.Pair{K: key, V: rules}
	return key, rules
}

func RegisterCallback(callback func(string)) {
	callbacks = append(callbacks, callback)
}

func requiredExe(path string) {
	_, err := exec.LookPath(path)
	if err != nil {
		log.Panicf("error finding specified executable: %v", err)
	}
}

func optionalExe(path string) error {
	_, err := exec.LookPath(path)
	return err
}

func postLoad() {
	// redirect log output to logfile if defined
	logk, logv := optionalString("logfile", config).String()
	if logv != "" {
		file, err := os.OpenFile(logv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Printf("error setting %s: %v", logk, err)
		} else {
			log.SetOutput(file)
		}
	}

	// check required fields
	Port()
	Interval()

	// make sure configured executables exist
	_, mime := Mime().Strings()
	if len(mime) != 0 {
		requiredExe(mime[0])
	}
	_, browser := Browser().Strings()
	if len(browser) != 0 {
		requiredExe(browser[0])
	}

	// validate the specified conversion rules
	rulesk, rulesv := Rules()
	for idx, rule := range rulesv {
		usage := false
		depsk, depsv := rule.Dep.Strings()
		for _, dep := range depsv {
			err := optionalExe(dep)
			if err != nil {
				log.Printf("error finding %s[%d].%s[%s]: %v", rulesk, idx, depsk, dep, err)
				usage = true
			}
		}
		if usage {
			_, msg := rule.Msg.String()
			log.Printf("%s", msg)
		}
	}
}

func init() {
	// load config file
	file := file.Provider(xdg.ConfigHome + "/cannon/cannon.yml")
	err := config.Load(file, yaml.Parser())
	if err != nil {
		log.Panicf("error loading config: %v", err)
	}

	postLoad()

	// watch for config file changes and reload
	file.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Printf("watch error: %v", err)
			return
		}

		// reload config file
		tmp := koanf.New(".")
		if err := tmp.Load(file, yaml.Parser()); err != nil {
			log.Printf("error loading config: %v", err)
			return
		}

		// update loaded config
		configLock.Lock()
		defer configLock.Unlock()
		config = tmp

		postLoad()

		// notify subscribers
		for _, callback := range callbacks {
			callback("reload")
		}
	})
}
