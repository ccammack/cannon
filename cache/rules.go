package cache

import (
	"path"
	"strings"

	"github.com/ccammack/cannon/config"
	"github.com/ccammack/cannon/util"
)

type ConversionRule struct {
	idx       int
	matchExt  bool
	Ext       []string
	matchMime bool
	Mime      []string
	cmd       []string
	html      string
}

func matchConversionRules(file string) (string, []ConversionRule) {
	extension := strings.ToLower(strings.TrimLeft(path.Ext(file), "."))
	mimetype := strings.ToLower(GetMimeType(file))

	matches := []ConversionRule{}

	rulesk, rulesv := config.Rules()
	for idx, rule := range rulesv {
		// TODO: add support for glob patterns
		_, exts := rule.Ext.Strings()
		matchExt := len(extension) > 0 && len(exts) > 0 && util.Find(exts, extension) < len(exts)

		// TODO: add support for glob patterns
		_, mimes := rule.Mime.Strings()
		matchMime := len(mimetype) > 0 && len(mimes) > 0 && util.Find(mimes, mimetype) < len(mimes)

		if matchExt || matchMime {
			_, cmd := rule.Cmd.Strings()
			_, html := rule.Html.String()

			matches = append(matches, ConversionRule{idx, matchExt, exts, matchMime, mimes, cmd, html})
		}
	}

	return rulesk, matches
}
