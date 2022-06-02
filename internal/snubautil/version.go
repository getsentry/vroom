package snubautil

import "fmt"

func FormatVersion(name, code interface{}) string {
	return fmt.Sprintf("%v (%v)", name, code)
}
