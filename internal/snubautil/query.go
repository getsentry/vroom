package snubautil

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrProfileNotFound = errors.New("profile not found")

const MaxRetentionInDays = 90

type SnubaProfilesResponse struct {
	Profiles []Profile `json:"data"`
}

func GetProfile(organizationID, projectID uint64, profileID string, sqb SnubaQueryBuilder) (Profile, error) {
	sqb.SelectCols = []string{
		"duration_ns",
		"platform",
		"profile",
		"profile_id",
		"project_id",
		"transaction_name",
		"version_code",
		"version_name",
	}
	now := time.Now().UTC()
	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("organization_id=%d", organizationID),
		fmt.Sprintf("project_id=%d", projectID),
		fmt.Sprintf("profile_id='%s'", Escape(profileID)),
		fmt.Sprintf("received >= toDateTime('%s')", now.AddDate(0, 0, -MaxRetentionInDays).Format(time.RFC3339)),
		fmt.Sprintf("received < toDateTime('%s')", now.Format(time.RFC3339)),
	)
	sqb.Limit = 1

	r, err := sqb.Do()
	if err != nil {
		return Profile{}, err
	}
	defer r.Close()
	var sr SnubaProfilesResponse
	err = json.NewDecoder(r).Decode(&sr)
	if err != nil {
		return Profile{}, err
	}

	if len(sr.Profiles) == 0 {
		return Profile{}, ErrProfileNotFound
	}

	return sr.Profiles[0], nil
}

func GetProfiles(sqb SnubaQueryBuilder, fetchPayload bool) ([]Profile, error) {
	sqb.SelectCols = []string{
		"organization_id", "project_id", "transaction_id", "profile_id", "received",
		"android_api_level", "device_classification", "device_locale", "device_manufacturer",
		"device_model", "device_os_build_number", "device_os_name", "device_os_version",
		"duration_ns", "environment", "platform", "trace_id", "transaction_name",
		"version_name", "version_code",
	}

	// Since the profile column containing the payload can be a few MB
	// this column is never retrieved unless necessary
	if fetchPayload {
		sqb.SelectCols = append(sqb.SelectCols, "profile")
	}

	sqb.OrderBy = "received DESC"

	r, err := sqb.Do()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var sr SnubaProfilesResponse
	err = json.NewDecoder(r).Decode(&sr)
	if err != nil {
		return nil, err
	}

	if len(sr.Profiles) == 0 {
		return []Profile{}, nil
	}

	return sr.Profiles, err
}

type SnubaFiltersResponse struct {
	Filters []map[string][]interface{} `json:"data"`
}

func GetFilters(sqb SnubaQueryBuilder) (map[string][]interface{}, error) {
	sqb.SelectCols = []string{
		"arraySort(groupUniqArray(android_api_level)) AS _android_api_level",
		"arraySort(groupUniqArray(device_model)) AS _device_model",
		"arraySort(groupUniqArray(device_classification)) AS _device_classification",
		"arraySort(groupUniqArray(device_locale)) AS _device_locale",
		"arraySort(groupUniqArray(device_manufacturer)) AS _device_manufacturer",
		"arraySort(groupUniqArray(device_os_build_number)) AS _device_os_build_number",
		"arraySort(groupUniqArray(device_os_name)) AS _device_os_name",
		"arraySort(groupUniqArray(device_os_version)) AS _device_os_version",
		"arraySort(groupUniqArray(platform)) AS _platform",
		"arraySort(groupUniqArray(transaction_name)) AS _transaction_name",
		"arraySort(groupUniqArray(tuple(version_name, version_code))) AS _version",
	}

	r, err := sqb.Do()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var sr SnubaFiltersResponse
	err = json.NewDecoder(r).Decode(&sr)
	if err != nil {
		return nil, err
	}
	filters := make(map[string][]interface{})
	for k, values := range sr.Filters[0] {
		k = strings.TrimPrefix(k, "_")
		switch k {
		case "version":
			filters[k] = make([]interface{}, 0, len(values))
			for _, v := range values {
				if versions, ok := v.([]interface{}); ok {
					filters[k] = append(filters[k], fmt.Sprintf("%s (build %s)", versions[0], versions[1]))
				}
			}
		default:
			filters[k] = values
		}
	}

	return filters, err
}
