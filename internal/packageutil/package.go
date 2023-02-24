package packageutil

import "strings"

// IsRustApplicationPackage determines whether the image represents that of the application
// binary (or a binary embedded in the application binary) by checking its package path.
func IsRustApplicationPackage(p string) bool {
	return p != "" &&
		// `/library/std/src/` and `/usr/lib/system/` come from a real profile collected on macos.
		// In this case the function belongs to a shared library, not to the profiled application.
		!strings.Contains(p, "/library/std/src/") &&
		!strings.HasPrefix(p, "/usr/lib/system/") &&
		// the following a prefixes of functions belonging to either core lib
		// or third party libs
		!strings.HasPrefix(p, "/rustc/") &&
		!strings.HasPrefix(p, "/usr/local/rustup/") &&
		!strings.HasPrefix(p, "/usr/local/cargo/")
}

// IsCocoaApplicationPackage determines whether the image represents that of the application
// binary (or a binary embedded in the application binary) by checking its package path.
func IsCocoaApplicationPackage(p string) bool {
	// These are the path patterns that iOS uses for applications, system
	// libraries are stored elsewhere.
	return strings.HasPrefix(p, "/private/var/containers") ||
		strings.HasPrefix(p, "/var/containers") ||
		strings.Contains(p, "/Developer/Xcode/DerivedData") ||
		strings.Contains(p, "/data/Containers/Bundle/Application")
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

// IsAndroidApplicationPackage checks if a symbol belongs to an Android system package.
func IsAndroidApplicationPackage(packageName string) bool {
	for _, p := range androidPackagePrefixes {
		if strings.HasPrefix(packageName, p) {
			return false
		}
	}
	return true
}
