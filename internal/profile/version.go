package profile

import "fmt"

func FormatVersion(name, code interface{}) string {
	if c, ok := code.(string); !ok || c == "" {
		return fmt.Sprintf("%v", name)
	}
	return fmt.Sprintf("%v (%v)", name, code)
}
