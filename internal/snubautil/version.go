package snubautil

import "fmt"

func FormatVersion(name, code interface{}) string {
	if code == "" {
		return fmt.Sprintf("%v", name)
	}
	return fmt.Sprintf("%v (%v)", name, code)
}
