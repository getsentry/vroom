package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/getsentry/vroom/internal/snubautil"
)

var (
	appVersionRegex = regexp.MustCompile(`(.+)\s\(build\s(.+)\)`)

	snubaFilterFields = []string{
		"device_classification",
		"device_locale",
		"device_manufacturer",
		"device_model",
		"device_os_build_number",
		"device_os_name",
		"device_os_version",
		"environment",
		"platform",
		"transaction_name",
	}
)

type (
	VersionBuild struct {
		Name string
		Code string
	}
)

func (e *environment) snubaQueryBuilderFromRequest(ctx context.Context, p url.Values) (snubautil.QueryBuilder, error) {
	sqb, err := e.snuba.NewQuery(ctx, "profiles")
	if err != nil {
		return snubautil.QueryBuilder{}, err
	}
	sqb.WhereConditions = make([]string, 0, 5)

	if projects, exists := p["project_id"]; exists && len(projects) > 0 {
		for _, p := range projects {
			_, err := strconv.ParseUint(p, 10, 64)
			if err != nil {
				return snubautil.QueryBuilder{}, fmt.Errorf("invalid project ID: %s", p)
			}
		}
		sqb.WhereConditions = append(sqb.WhereConditions, fmt.Sprintf("project_id IN tuple(%s)", strings.Join(projects, ", ")))
	}

	if periodStart := p.Get("start"); periodStart != "" {
		sqb.WhereConditions = append(sqb.WhereConditions, fmt.Sprintf("received >= toDateTime('%s')", periodStart))
	} else {
		return snubautil.QueryBuilder{}, errors.New("no range start in the request")
	}

	if periodEnd := p.Get("end"); periodEnd != "" {
		sqb.WhereConditions = append(sqb.WhereConditions, fmt.Sprintf("received < toDateTime('%s')", periodEnd))
	} else {
		return snubautil.QueryBuilder{}, errors.New("no range end in the request")
	}

	if v := p.Get("limit"); v != "" {
		limit, err := strconv.Atoi(v)
		if err != nil {
			log.Err(err).Str("limit", v).Msg("can't parse limit value")
			return snubautil.QueryBuilder{}, err
		}
		sqb.Limit = limit
	}

	if v := p.Get("offset"); v != "" {
		offset, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			log.Err(err).Str("offset", v).Msg("can't parse offset value")
			return snubautil.QueryBuilder{}, err
		}
		sqb.Offset = offset
	}

	// fields
	for _, field := range snubaFilterFields {
		v, exists := p[field]
		if !exists {
			continue
		}
		escaped := make([]string, 0, len(v))
		for _, s := range v {
			if s == "" {
				continue
			}
			escaped = append(escaped, fmt.Sprintf("'%s'", snubautil.Escape(s)))
		}
		sqb.WhereConditions = append(sqb.WhereConditions, fmt.Sprintf("%s IN tuple(%s)", field, strings.Join(escaped, ", ")))
	}

	// android api level
	if levels, exists := p["android_api_level"]; exists {
		for _, l := range levels {
			_, err := strconv.ParseUint(l, 10, 64)
			if err != nil {
				return snubautil.QueryBuilder{}, errors.New("can't parse android api level")
			}
		}
		sqb.WhereConditions = append(sqb.WhereConditions, fmt.Sprintf("android_api_level IN tuple(%s)", strings.Join(levels, ", ")))
	}

	if versions, exists := p["version"]; exists {
		versionBuilds, err := GetVersionBuildFromAppVersions(versions)
		if err != nil {
			return snubautil.QueryBuilder{}, err
		}
		pairs := make([]string, 0, len(versionBuilds))
		for _, vb := range versionBuilds {
			pairs = append(pairs, fmt.Sprintf("(version_name = '%s' AND version_code = '%s')", snubautil.Escape(vb.Name), snubautil.Escape(vb.Code)))
		}
		sqb.WhereConditions = append(sqb.WhereConditions, fmt.Sprintf("(%s)", strings.Join(pairs, " OR ")))
	}

	return sqb, nil
}

func GetVersionBuildFromAppVersions(appVersions []string) ([]VersionBuild, error) {
	var versionBuilds []VersionBuild
	for _, version := range appVersions {
		versionBuild, err := GetVersionBuildFromAppVersion(version)
		if err != nil {
			return nil, err
		}
		versionBuilds = append(versionBuilds, versionBuild)
	}
	return versionBuilds, nil
}

func GetVersionBuildFromAppVersion(appVersion string) (VersionBuild, error) {
	if appVersionRegex.MatchString(appVersion) {
		s := appVersionRegex.FindStringSubmatch(appVersion)
		return VersionBuild{Name: s[1], Code: s[2]}, nil
	}
	return VersionBuild{}, fmt.Errorf("cannot parse application_versions: %v", appVersion)
}

func snubaProfileToProfileResult(profile snubautil.Profile) ProfileResult {
	return ProfileResult{
		AndroidAPILevel:      profile.AndroidAPILevel,
		DeviceClassification: profile.DeviceClassification,
		DeviceLocale:         profile.DeviceLocale,
		DeviceManufacturer:   profile.DeviceManufacturer,
		DeviceModel:          profile.DeviceModel,
		DeviceOsBuildNumber:  profile.DeviceOsBuildNumber,
		DeviceOsName:         profile.DeviceOsName,
		DeviceOsVersion:      profile.DeviceOsVersion,
		ID:                   profile.ProfileID,
		ProjectID:            strconv.FormatUint(profile.ProjectID, 10),
		Timestamp:            profile.Received.Unix(),
		TraceDurationMs:      float64(profile.DurationNs) / 1_000_000,
		TransactionID:        profile.TransactionID,
		TransactionName:      profile.TransactionName,
		VersionCode:          profile.VersionCode,
		VersionName:          profile.VersionName,
	}
}
