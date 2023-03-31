package packageutil

import (
	"fmt"
	"regexp"
	"strings"
)

var packageRegex = regexp.MustCompile(
	`[/\\](.*?)[/\\](.*?)[/\\].*`,
)

func ParseNodePackageFromPath(p string) (string, bool) {
	if strings.HasPrefix(p, "node:") {
		return strings.Split(p, "/")[0], false
	}
	splits := strings.Split(p, "node_modules")
	p = splits[len(splits)-1]
	results := packageRegex.FindStringSubmatch(p)
	if len(results) > 2 {
		if results[1][0] == '@' {
			return fmt.Sprintf("%s/%s", results[1], results[2]), false
		}
		return results[1], false
	}
	return "", true
}
