package android

import (
	"strings"
)

func StripPackageNameFromFullMethodName(s, p string) string {
	return strings.TrimPrefix(s, p+".")
}
