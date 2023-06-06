package packageutil

import "testing"

func frameType(isApplication bool) string {
	if isApplication {
		return "application"
	}
	return "system"
}

var appIdentifier = "io.sentry.samples.android"

func TestIsAndroidApplicationPackage(t *testing.T) {
	tests := []struct {
		name          string
		pkg           string
		identifier    string
		isApplication bool
	}{
		{
			name:          "android system package",
			pkg:           "android.app",
			identifier:    "",
			isApplication: false,
		},
		{
			name:          "androidx system package",
			pkg:           "androidx.lifecycle",
			identifier:    "",
			isApplication: false,
		},
		{
			name:          "io.sentry.samples.android application package",
			pkg:           appIdentifier + ".foo",
			identifier:    appIdentifier,
			isApplication: true,
		},
		{
			name:          "io.sentry.samples.android package must be prefix",
			pkg:           appIdentifier,
			identifier:    appIdentifier,
			isApplication: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isApplication := IsAndroidApplicationPackage(tt.pkg, tt.identifier); isApplication != tt.isApplication {
				t.Fatalf("Expected %s frame but got %s frame", frameType(tt.isApplication), frameType(isApplication))
			}
		})
	}
}
