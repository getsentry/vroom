package testutil

import (
	"math"
	"math/big"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

var (
	// Copied from
	// https://github.com/googleapis/google-cloud-go/blob/b62783c3c30aecc880f2cd7bce146d9e4e9e59be/internal/testutil/cmp.go#L15-L61
	alwaysEqual       = cmp.Comparer(func(_, _ interface{}) bool { return true })
	defaultCmpOptions = []cmp.Option{
		// Use protocmp.Transform for protobufs
		protocmp.Transform(),
		// Use big.Rat.Cmp for big.Rats
		cmp.Comparer(func(x, y *big.Rat) bool {
			if x == nil || y == nil {
				return x == y
			}
			return x.Cmp(y) == 0
		}),
		// NaNs compare equal
		cmp.FilterValues(func(x, y float64) bool {
			return math.IsNaN(x) && math.IsNaN(y)
		}, alwaysEqual),
		cmp.FilterValues(func(x, y float32) bool {
			return math.IsNaN(float64(x)) && math.IsNaN(float64(y))
		}, alwaysEqual),
	}

	False = false
	True  = true
)

func Diff(a, b interface{}, opts ...cmp.Option) string {
	opts = append(opts, defaultCmpOptions...)
	return cmp.Diff(a, b, opts...)
}
