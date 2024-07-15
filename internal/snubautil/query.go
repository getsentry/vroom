package snubautil

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/profile"
)

var ErrProfileNotFound = errors.New("profile not found")

const MaxRetentionInDays = 90

type SnubaProfilesResponse struct {
	Profiles []profile.Profile `json:"data"`
}

func GetProfiles(sqb QueryBuilder) ([]profile.Profile, error) {
	t := sentry.TransactionFromContext(sqb.ctx)
	rs := t.StartChild("snuba")
	rs.Description = "Get profiles"
	defer rs.Finish()

	sqb.SelectCols = []string{
		"android_api_level",
		"device_classification",
		"device_locale",
		"device_manufacturer",
		"device_model",
		"device_os_build_number",
		"device_os_name",
		"device_os_version",
		"duration_ns",
		"environment",
		"organization_id",
		"platform",
		"profile_id",
		"project_id",
		"received",
		"trace_id",
		"transaction_id",
		"transaction_name",
		"version_code",
		"version_name",
	}

	sqb.OrderBy = "received DESC"

	rb, err := sqb.Do(rs)
	if err != nil {
		return nil, err
	}
	defer rb.Close()

	s := rs.StartChild("json.unmarshal")
	s.Description = "Decode response from Snuba"
	defer s.Finish()

	var sr SnubaProfilesResponse
	err = json.NewDecoder(rb).Decode(&sr)
	if err != nil {
		return nil, err
	}

	if len(sr.Profiles) == 0 {
		return []profile.Profile{}, nil
	}

	return sr.Profiles, err
}

func GetProfileIDByTransactionID(organizationID, projectID uint64, transactionID string, sqb QueryBuilder) (string, error) {
	t := sentry.TransactionFromContext(sqb.ctx)
	rs := t.StartChild("snuba")
	rs.Description = "Get a profile ID from a transaction ID"
	defer rs.Finish()

	sqb.SelectCols = []string{"profile_id"}
	now := time.Now().UTC()
	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("organization_id=%d", organizationID),
		fmt.Sprintf("project_id=%d", projectID),
		fmt.Sprintf("transaction_id='%s'", Escape(transactionID)),
		fmt.Sprintf("received >= toDateTime('%s')", now.AddDate(0, 0, -MaxRetentionInDays).Format(time.RFC3339)),
		fmt.Sprintf("received < toDateTime('%s')", now.Format(time.RFC3339)),
	)
	sqb.Limit = 1

	rb, err := sqb.Do(rs)
	if err != nil {
		return "", err
	}
	defer rb.Close()

	s := rs.StartChild("json.unmarshal")
	s.Description = "Decode response from Snuba"
	defer s.Finish()

	var sr SnubaProfilesResponse
	err = json.NewDecoder(rb).Decode(&sr)
	if err != nil {
		return "", err
	}

	if len(sr.Profiles) == 0 {
		return "", ErrProfileNotFound
	}

	return sr.Profiles[0].ID(), nil
}

type GetProfileIDsResponse struct {
	IDs []ProfileID `json:"data"`
}
type ProfileID struct {
	ID string `json:"profile_id"`
}

func GetProfileIDs(organizationID, limit uint64, sqb QueryBuilder) ([]string, error) {
	t := sentry.TransactionFromContext(sqb.ctx)
	rs := t.StartChild("snuba")
	rs.Description = "Get a list of profile IDs from the params passed in the request"
	defer rs.Finish()

	sqb.SelectCols = []string{"profile_id"}
	now := time.Now().UTC()
	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("organization_id=%d", organizationID),
		fmt.Sprintf("received >= toDateTime('%s')", now.AddDate(0, 0, -MaxRetentionInDays).Format(time.RFC3339)),
		fmt.Sprintf("received < toDateTime('%s')", now.Format(time.RFC3339)),
	)
	sqb.Limit = limit
	sqb.OrderBy = "received DESC"

	rb, err := sqb.Do(rs)
	if err != nil {
		return nil, err
	}
	defer rb.Close()

	s := rs.StartChild("json.unmarshal")
	s.Description = "Decode response from Snuba"
	defer s.Finish()

	var resp GetProfileIDsResponse
	err = json.NewDecoder(rb).Decode(&resp)
	if err != nil {
		return nil, err
	}

	idS := make([]string, len(resp.IDs))
	for i, profID := range resp.IDs {
		idS[i] = profID.ID
	}

	return idS, nil
}
