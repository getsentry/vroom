package aggregate

import (
	"fmt"

	"github.com/getsentry/vroom/internal/android"
	"github.com/getsentry/vroom/internal/calltree"
)

// The maximum number of top functions to return for BACKTRACE and ANDROID_TRACE
const defaultNTopFunctions = 100

func NewAggregatorFromPlatform(platform string) (AggregatorP, error) {
	switch platform {
	case "cocoa":
		return &BacktraceAggregatorP{
			n:                          defaultNTopFunctions,
			bta:                        calltree.NewBacktraceAggregatorP(),
			profileIDToTransactionName: make(map[string]string),
			symbolsByProfileID:         make(map[string]map[string]Symbol),
		}, nil
	case "android":
		return &AndroidTraceAggregatorP{
			methodKeyToMethod:      make(map[methodKey]android.AndroidMethod),
			methodKeyToProfileData: make(map[methodKey][]profileMethodData),
			methodKeyToProfileIDs:  make(map[methodKey][]string),
			numFunctions:           defaultNTopFunctions,
			profileIDToInteraction: make(map[string]string),
		}, nil
	default:
		return nil, fmt.Errorf("platform <%s> not supported", platform)
	}
}
