package calltree

import (
	"path"
)

// ImageBaseName returns the basename of the image, if image is a path string.
func ImageBaseName(image string) string {
	if image == "" {
		return ""
	}
	return path.Base(image)
}

// IsImageEqual performs comparison of two images by normalizing them
// to a basename representation, if they are a path. e.g.
// /private/var/containers/<UUID>/App becomes just App, since the path components
// can contain random strings that are unique to specific devices/installations.
func IsImageEqual(image1, image2 string) bool {
	return ImageBaseName(image1) == ImageBaseName(image2)
}
