package occurrence

import (
	"context"
	"errors"
	"sync"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/storageutil"
	"gocloud.dev/blob"
)

type RegressedFunction struct {
	OrganizationID           uint64  `json:"organization_id"`
	ProjectID                uint64  `json:"project_id"`
	ProfileID                string  `json:"profile_id"`
	Fingerprint              uint32  `json:"fingerprint"`
	AbsolutePercentageChange float64 `json:"absolute_percentage_change"`
	AggregateRange1          float64 `json:"aggregate_range_1"`
	AggregateRange2          float64 `json:"aggregate_range_2"`
	Breakpoint               uint64  `json:"breakpoint"`
	TrendDifference          float64 `json:"trend_difference"`
	TrendPercentage          float64 `json:"trend_percentage"`
	UnweightedPValue         float64 `json:"unweighted_p_value"`
	UnweightedTValue         float64 `json:"unweighted_t_value"`
}

func ProcessRegressedFunction(
	ctx context.Context,
	profilesBucket *blob.Bucket,
	regressedFunction RegressedFunction,
) (*Occurrence, error) {
	s := sentry.StartSpan(ctx, "profile.read")
	s.Description = "Read profile from GCS"
	var p profile.Profile
	objectName := profile.StoragePath(
		regressedFunction.OrganizationID,
		regressedFunction.ProjectID,
		regressedFunction.ProfileID,
	)
	err := storageutil.UnmarshalCompressed(ctx, profilesBucket, objectName, &p)
	s.Finish()
	if err != nil {
		return nil, err
	}

	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Generate call trees"
	calltreesByTID, err := p.CallTrees()
	s.Finish()

	if err != nil {
		return nil, err
	}

	calltrees, exists := calltreesByTID[p.Transaction().ActiveThreadID]
	if !exists {
		return nil, errors.New("calltree not found")
	}

	s = sentry.StartSpan(ctx, "processing")
	s.Description = "Searching for fingerprint"
	var node *nodetree.Node
	for _, calltree := range calltrees {
		node = calltree.FindNodeByFingerprint(regressedFunction.Fingerprint)
	}
	s.Finish()

	if node == nil {
		return nil, errors.New("fingerprint not found")
	}

	return FromRegressedFunction(p, regressedFunction, node.Frame), nil
}

func ProcessRegressedFunctions(
	ctx context.Context,
	hub *sentry.Hub,
	profilesBucket *blob.Bucket,
	regressedFunctions []RegressedFunction,
	numWorkers int,
) []*Occurrence {
	if len(regressedFunctions) < numWorkers {
		numWorkers = len(regressedFunctions)
	}

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	regressedChan := make(chan RegressedFunction, numWorkers)
	occurrenceChan := make(chan *Occurrence)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for regressedFunction := range regressedChan {
				occurrence, err := ProcessRegressedFunction(ctx, profilesBucket, regressedFunction)
				if err != nil {
					hub.CaptureException(err)
					continue
				} else if occurrence == nil {
					continue
				}

				occurrenceChan <- occurrence
			}
		}()
	}

	go func() {
		for _, regressedFunction := range regressedFunctions {
			regressedChan <- regressedFunction
		}
		close(regressedChan)

		// wait until all the profiles have been processed
		// then we can close the occurrence channel and collect
		// any occurrences that have been created
		wg.Wait()
		close(occurrenceChan)
	}()

	occurrences := []*Occurrence{}
	for occurrence := range occurrenceChan {
		occurrences = append(occurrences, occurrence)
	}

	return occurrences
}
