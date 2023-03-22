package testutil

import (
	"math"

	"github.com/getsentry/vroom/internal/timeutil"
	"github.com/google/go-cmp/cmp"
)

var (
	alwaysEqual       = cmp.Comparer(func(_, _ interface{}) bool { return true })
	defaultCmpOptions = []cmp.Option{
		// NaNs compare equal
		cmp.FilterValues(func(x, y float64) bool {
			return math.IsNaN(x) && math.IsNaN(y)
		}, alwaysEqual),
		cmp.FilterValues(func(x, y float32) bool {
			return math.IsNaN(float64(x)) && math.IsNaN(float64(y))
		}, alwaysEqual),
		cmp.AllowUnexported(timeutil.Time{}),
	}

	False = false
	True  = true
)

func Diff(a, b interface{}, opts ...cmp.Option) string {
	opts = append(opts, defaultCmpOptions...)
	return cmp.Diff(a, b, opts...)
}
