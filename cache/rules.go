package cache

import (
	"fmt"
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
	src       string
	html      string
}

func matchConversionRules(res *Resource) (string, []ConversionRule) {
	res.progress = append(res.progress, fmt.Sprintf("Select file: %s", res.file))

	extension := strings.ToLower(strings.TrimLeft(path.Ext(res.file), "."))
	mimetype := strings.ToLower(GetMimeType(res.file))

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
			_, src := rule.Src.String()
			_, html := rule.Html.String()

			match := ConversionRule{idx, matchExt, exts, matchMime, mimes, cmd, src, html}
			res.progress = append(res.progress, fmt.Sprintf("Match rule[%d]: %v", idx, match))
			matches = append(matches, match)
		}
	}

	return rulesk, matches
}
