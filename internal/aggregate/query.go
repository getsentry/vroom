package aggregate

import (
	"github.com/getsentry/vroom/internal/snubautil"
)

func AggregateProfiles(profiles []snubautil.Profile, topNFunctions int) (AggregationResult, error) {
	if len(profiles) == 0 {
		return AggregationResult{}, nil
	}

	agg, err := NewAggregatorFromPlatform(profiles[0].Platform)
	if err != nil {
		return AggregationResult{}, err
	}

	if topNFunctions > 0 {
		agg.SetTopNFunctions(topNFunctions)
	}

	for _, profile := range profiles {
		err = agg.UpdateFromProfile(profile)
		if err != nil {
			return AggregationResult{}, err
		}
	}

	res, err := agg.Result()
	if err != nil {
		return AggregationResult{}, err
	}

	return AggregationResult{
		RowCount:    uint32(len(profiles)),
		Aggregation: res,
	}, nil

}
