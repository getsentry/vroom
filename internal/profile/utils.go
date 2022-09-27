package profile

import (
	"strings"
)

// isApplicationSymbol determines whether the image represents that of the application
// binary (or a binary embedded in the application binary) by checking its path.
func IsRustApplicationImage(image string) bool {
	// `/library/std/src/` and `/usr/lib/system/` come from a real profile collected on macos.
	// In this case the function belongs to a shared library,
	// not to the profiled application
	return !strings.Contains(image, "/library/std/src/") &&
		!strings.HasPrefix(image, "/usr/lib/system/") &&
		!(image == "")
}

// Checking if synmbol belongs to an Android system package
func IsAndroidSystemPackage(packageName string) bool {
	return IsSystemPackage(packageName)
}

// isApplicationSymbol determines whether the image represents that of the application
// binary (or a binary embedded in the application binary) by checking its path.
func IsIOSApplicationImage(image string) bool {
	// These are the path patterns that iOS uses for applications, system
	// libraries are stored elsewhere.
	//
	// Must be kept in sync with the corresponding Python implementation of
	// this function in python/symbolicate/__init__.py
	return strings.HasPrefix(image, "/private/var/containers") ||
		strings.HasPrefix(image, "/var/containers") ||
		strings.Contains(image, "/Developer/Xcode/DerivedData") ||
		strings.Contains(image, "/data/Containers/Bundle/Application")
}
