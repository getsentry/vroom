package occurrence

import (
	"context"
	"errors"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/chunk"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/getsentry/vroom/internal/storageutil"
	"github.com/getsentry/vroom/internal/utils"
	"gocloud.dev/blob"
)

type RegressedFunction struct {
	OrganizationID           uint64                `json:"organization_id"`
	ProjectID                uint64                `json:"project_id"`
	ProfileID                string                `json:"profile_id"`
	Example                  utils.ExampleMetadata `json:"example"`
	Fingerprint              uint32                `json:"fingerprint"`
	AbsolutePercentageChange float64               `json:"absolute_percentage_change"`
	AggregateRange1          float64               `json:"aggregate_range_1"`
	AggregateRange2          float64               `json:"aggregate_range_2"`
	Breakpoint               uint64                `json:"breakpoint"`
	TrendDifference          float64               `json:"trend_difference"`
	TrendPercentage          float64               `json:"trend_percentage"`
	UnweightedPValue         float64               `json:"unweighted_p_value"`
	UnweightedTValue         float64               `json:"unweighted_t_value"`
}

func ProcessRegressedFunction(
	ctx context.Context,
	profilesBucket *blob.Bucket,
	regressedFunction RegressedFunction,
	jobs chan storageutil.ReadJob,
) (*Occurrence, error) {
	results := make(chan storageutil.ReadJobResult, 1)
	defer close(results)

	if regressedFunction.ProfileID != "" {
		// For back compat, we should be use the example moving forwards
		jobs <- profile.ReadJob{
			Ctx:            ctx,
			OrganizationID: regressedFunction.OrganizationID,
			ProjectID:      regressedFunction.ProjectID,
			ProfileID:      regressedFunction.ProfileID,
			Storage:        profilesBucket,
			Result:         results,
		}
	} else if regressedFunction.Example.ProfileID != "" {
		jobs <- profile.ReadJob{
			Ctx:            ctx,
			OrganizationID: regressedFunction.OrganizationID,
			ProjectID:      regressedFunction.ProjectID,
			ProfileID:      regressedFunction.Example.ProfileID,
			Storage:        profilesBucket,
			Result:         results,
		}
	} else {
		jobs <- chunk.ReadJob{
			Ctx:            ctx,
			OrganizationID: regressedFunction.OrganizationID,
			ProjectID:      regressedFunction.ProjectID,
			ProfilerID:     regressedFunction.Example.ProfilerID,
			ChunkID:        regressedFunction.Example.ChunkID,
			Storage:        profilesBucket,
			Result:         results,
		}
	}

	res := <-results
	platform, frame, err := getPlatformAndFrame(ctx, res, regressedFunction.Fingerprint)
	if err != nil {
		return nil, err
	}
	return FromRegressedFunction(platform, regressedFunction, frame), nil
}

func getPlatformAndFrame(
	ctx context.Context,
	res storageutil.ReadJobResult,
	target uint32,
) (platform.Platform, frame.Frame, error) {
	var platform platform.Platform
	var frame frame.Frame

	err := res.Error()
	if err != nil {
		return platform, frame, err
	}

	s := sentry.StartSpan(ctx, "processing")
	s.Description = "Searching for fingerprint"
	defer s.Finish()

	if result, ok := res.(profile.ReadJobResult); ok {
		platform = result.Profile.Platform()
		frame, err = result.Profile.GetFrameWithFingerprint(target)
		if err != nil {
			return platform, frame, err
		}
	} else if result, ok := res.(chunk.ReadJobResult); ok {
		platform = result.Chunk.GetPlatform()
		frame, err = result.Chunk.GetFrameWithFingerprint(target)
		if err != nil {
			return platform, frame, err
		}
	} else {
		// This should never happen
		return platform, frame, errors.New("unexpected result from storage")
	}

	return platform, frame, nil
}
