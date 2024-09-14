package config

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"github.com/ccammack/cannon/gen"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var (
	configPath = xdg.ConfigHome + "/cannon/cannon.yml"
	configFp   = new(file.File)
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
		log.Panicf("Error reading hostname: %v", err)
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

func ReplacePlaceholder(s, placeholder, replacement string) string {
	return strings.ReplaceAll(s, placeholder, replacement)
}

func ReplaceEnvPlaceholders(s string) string {
	re := regexp.MustCompile(`\{env.(.+)\}`)
	for {
		matches := re.FindStringSubmatch(s)
		if matches == nil {
			break
		}

		env := matches[1]
		value := os.Getenv(env)
		if value == "" {
			log.Printf("error looking for env var: %s", env)
			break
		}

		value = strings.ReplaceAll(value, "\\", "/")
		s = strings.Replace(s, `{env.`+env+`}`, value, 1)
	}
	return s
}

func applyEnvPlaceholder(key string, required bool, ko *koanf.Koanf) gen.Pair {
	var output string
	pair := optionalString(key, ko)
	if required && pair.V == nil {
		log.Panicf("Error trying to find required key: %v", pair)
	}
	k, entry := pair.String()
	output = ReplaceEnvPlaceholders(entry)
	return gen.Pair{K: k, V: output}
}

func applyEnvPlaceholders(key string, required bool, ko *koanf.Koanf) gen.Pair {
	var output []string
	pair := optionalStrings(key, ko)
	if required && pair.V == nil {
		log.Panicf("Error trying to find required key: %v", pair)
	}
	k, entries := pair.Strings()
	if required && len(entries) == 0 {
		log.Panicf("Error trying to find required values: %v", pair)
	}
	for _, v := range entries {
		output = append(output, ReplaceEnvPlaceholders(v))
	}
	return gen.Pair{K: k, V: output}
}

func Port() gen.Pair    { return applyEnvPlaceholder("port", true, config) }
func Timeout() gen.Pair { return applyEnvPlaceholder("timeout", true, config) }
func Exit() gen.Pair    { return applyEnvPlaceholder("exit", true, config) }
func Logfile() gen.Pair { return applyEnvPlaceholder("logfile", false, config) }
func Mime() gen.Pair    { return applyEnvPlaceholders("mime", true, config) }
func Browser() gen.Pair { return applyEnvPlaceholders("browser", false, config) }
func Style() gen.Pair   { return applyEnvPlaceholder("style", false, config) }

type FileConversionDep struct {
	Apps gen.Pair
	Desc gen.Pair
}

func Deps() (string, []FileConversionDep) {
	// collect the list of required applications
	key, err := key("deps", config)
	if err != nil {
		return "", nil
	}

	configLock.RLock()
	defer configLock.RUnlock()

	// clone the the rules
	deps := []FileConversionDep{}
	for _, v := range config.Slices(key) {
		apps := applyEnvPlaceholders("apps", false, v)
		desc := applyEnvPlaceholder("desc", false, v)
		deps = append(deps, FileConversionDep{apps, desc})
	}

	// TODO: make Deps() return a gen.Pair
	// return gen.Pair{K: key, V: rules}
	return key, deps
}

type FileConversionRule struct {
	Ext  gen.Pair
	Mime gen.Pair
	Cmd  gen.Pair
	Src  gen.Pair
	Html gen.Pair
}

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
		mime := optionalStrings("mime", v)
		cmd := applyEnvPlaceholders("cmd", false, v)
		src := optionalString("src", v)
		html := optionalString("html", v)

		rules = append(rules, FileConversionRule{ext, mime, cmd, src, html})
	}

	// TODO: make Rules() return a gen.Pair
	// return gen.Pair{K: key, V: rules}
	return key, rules
}

func RegisterCallback(callback func(string)) {
	callbacks = append(callbacks, callback)
}

func requiredExe(path string) {
	_, err := exec.LookPath(path)
	if err != nil {
		log.Panicf("Error finding required executable: %v", err)
	}
}

func optionalExe(path string) error {
	_, err := exec.LookPath(path)
	return err
}

func postLoad() {
	// redirect log output to logfile if defined
	logk, logv := Logfile().String()
	if logv != "" {
		file, err := os.OpenFile(logv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Printf("Error setting %s: %v", logk, err)
		} else {
			log.SetOutput(file)
		}
	}

	// check required fields
	Port()
	Timeout()
	Style()
}

func Validate() {
	// make sure configured executables exist
	_, mime := Mime().Strings()
	if len(mime) != 0 {
		requiredExe(mime[0])
	}
	_, browser := Browser().Strings()
	if len(browser) != 0 {
		requiredExe(browser[0])
	}

	// validate the specified deps
	depsk, depsv := Deps()
	for idx, rule := range depsv {
		usage := false
		appsk, appsv := rule.Apps.Strings()
		for _, app := range appsv {
			err := optionalExe(app)
			if err != nil {
				log.Printf("Error finding %s[%d].%s[%s]: %v", depsk, idx, appsk, app, err)
				usage = true
			}
		}
		if usage {
			_, desc := rule.Desc.String()
			log.Printf("%s", desc)
		}
	}
}

func init() {
	// load the config file on every invocation
	configFp = file.Provider(configPath)
	err := config.Load(configFp, yaml.Parser())
	if err != nil {
		log.Panicf("Error loading config: %v", err)
	}

	// check required fields
	Port()
}

func Start() {
	// perform additional config checks for --start
	postLoad()

	// watch for config file changes and reload
	configFp.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Printf("Watch error: %v", err)
			return
		}

		// reload config file
		tmp := koanf.New(".")
		if err := tmp.Load(configFp, yaml.Parser()); err != nil {
			log.Printf("Error loading config: %v", err)
			return
		}

		// notify subscribers
		for _, callback := range callbacks {
			callback("reload")
		}

		// update loaded config
		configLock.Lock()
		config = tmp
		configLock.Unlock()

		postLoad()
	})
}
