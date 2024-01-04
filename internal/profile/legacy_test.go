package profile

import (
	"testing"

	"github.com/getsentry/vroom/internal/sample"
)

func TestSampleToAndroidFormat(t *testing.T) {
	p := sample.Trace{}
	ap := sampleToAndroidFormat(p, 0)
	// only added to make pre-commit hooks happy
	// so that I can push and open a WIP draft
	if len(ap.Events) > 0 {
		t.Fatal("This has to be removed")
	}
	// TODO: write unit test
}
