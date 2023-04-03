package packageutil

import (
	"fmt"
	"path/filepath"
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

func ParseNodeModuleFromPath(p string) PackageInfo {
	if strings.HasPrefix(p, "node:") {
		return PackageInfo{
			InApp:  &f,
			Module: p,
		}
	}

	splits := strings.Split(p, "node_modules")

	// if there's no node_modules, user package
	if len(splits) == 1 {
		return PackageInfo{
			InApp: &t,
		}
	}

	// Take the last part of the string after the last node_modules without the first path divider
	module := splits[len(splits)-1][1:]
	module = strings.TrimSuffix(module, filepath.Ext(module))

	return PackageInfo{
		InApp:  &f,
		Module: module,
	}
}

func ParseNodePackageFromPath(p string) PackageInfo {
	// if it's a official node package
	if strings.HasPrefix(p, "node:") {
		return PackageInfo{
			InApp:   &f,
			Package: strings.Split(p, "/")[0],
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
				InApp:   &f,
				Package: fmt.Sprintf("%s/%s", results[1], results[2]),
			}
		}
		return PackageInfo{
			InApp:   &f,
			Package: results[1],
		}
	}

	return PackageInfo{
		InApp: &t,
	}
}
