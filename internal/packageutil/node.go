package packageutil

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	packageRegex = regexp.MustCompile(
		`[/\\]([^/\\].+?)[/\\]([^/\\].+?)[/\\].*`,
	)
	t = true
	f = false
)

func ParseNodePackageFromPath(p string) PackageInfo {
	// if it's a official node package
	if strings.HasPrefix(p, "node:") {
		return PackageInfo{
			Package: strings.Split(p, "/")[0],
			InApp:   &f,
		}
	}

	splits := strings.Split(p, "node_modules")

	// if there's no node_modules, user package
	if len(splits) == 1 {
		return PackageInfo{
			InApp: &t,
		}
	}

	results := packageRegex.FindStringSubmatch(splits[len(splits)-1])

	// if it's a third party package
	if len(results) > 2 {
		if results[1][0] == '@' {
			return PackageInfo{
				Package: fmt.Sprintf("%s/%s", results[1], results[2]),
				InApp:   &f,
			}
		}
		return PackageInfo{
			Package: results[1],
			InApp:   &f,
		}
	}

	return PackageInfo{
		InApp: &t,
	}
}
