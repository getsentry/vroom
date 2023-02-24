package packageutil

import "strings"

// isApplicationSymbol determines whether the image represents that of the application
// binary (or a binary embedded in the application binary) by checking its path.
func IsRustApplicationPackage(path string) bool {
	// `/library/std/src/` and `/usr/lib/system/` come from a real profile collected on macos.
	// In this case the function belongs to a shared library, not to the profiled application.
	return !strings.Contains(path, "/library/std/src/") &&
		!strings.HasPrefix(path, "/usr/lib/system/") &&
		// the following a prefixes of functions belonging to either core lib
		// or third party libs
		!strings.HasPrefix(path, "/rustc/") &&
		!strings.HasPrefix(path, "/usr/local/rustup/") &&
		!strings.HasPrefix(path, "/usr/local/cargo/") &&
		!(path == "")
}

// isApplicationSymbol determines whether the image represents that of the application
// binary (or a binary embedded in the application binary) by checking its path.
func IsCocoaApplicationPackage(path string) bool {
	// These are the path patterns that iOS uses for applications, system
	// libraries are stored elsewhere.
	return strings.HasPrefix(path, "/private/var/containers") ||
		strings.HasPrefix(path, "/var/containers") ||
		strings.Contains(path, "/Developer/Xcode/DerivedData") ||
		strings.Contains(path, "/data/Containers/Bundle/Application")
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

// Checking if synmbol belongs to an Android system package.
func IsAndroidApplicationPackage(packageName string) bool {
	for _, p := range androidPackagePrefixes {
		if strings.HasPrefix(packageName, p) {
			return false
		}
	}
	return true
}
