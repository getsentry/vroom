package aggregate

import (
	"github.com/getsentry/vroom/internal/snubautil"
)

// Aggregator is a type that performs aggregations of a metric. Rows of data
// are applied one at a time by calling Update(), and the final result is
// retrieved by calling Result()
type AggregatorP interface {
	// Update is called to apply a new row of data to the aggregation.
	UpdateFromProfile(profile snubautil.Profile) error

	// Result returns the final aggregated result as a BacktraceAggregate.
	Result() (Aggregate, error)

	SetTopNFunctions(n int)
}
