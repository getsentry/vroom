package packageutil

import "strings"

// isApplicationSymbol determines whether the image represents that of the application
// binary (or a binary embedded in the application binary) by checking its path.
func IsRustApplicationPackage(image string) bool {
	// `/library/std/src/` and `/usr/lib/system/` come from a real profile collected on macos.
	// In this case the function belongs to a shared library,
	// not to the profiled application
	return !strings.Contains(image, "/library/std/src/") &&
		!strings.HasPrefix(image, "/usr/lib/system/") &&
		!(image == "")
}

// isApplicationSymbol determines whether the image represents that of the application
// binary (or a binary embedded in the application binary) by checking its path.
func IsIOSApplicationPackage(image string) bool {
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

var (
	androidPackagePrefixes = []string{
		"android.",
		"androidx.",
		"com.android.",
		"com.google.android.",
		"com.motorola.",
		"java.",
		"javax.",
		"kotlin.",
		"kotlinx.",
		"retrofit2.",
		"sun.",
	}
)

// Checking if synmbol belongs to an Android system package
func IsAndroidApplicationPackage(packageName string) bool {
	for _, p := range androidPackagePrefixes {
		if strings.HasPrefix(packageName, p) {
			return false
		}
	}
	return true
}
