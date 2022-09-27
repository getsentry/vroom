package snubautil

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	appVersionRegex = regexp.MustCompile(`(.+)\s\((.+)\)`)
)

type (
	VersionBuild struct {
		Name string
		Code string
	}
)

func MakeProjectsFilter(params url.Values) ([]string, error) {
	if projects, exists := params["project_id"]; exists && len(projects) > 0 {
		for _, project := range projects {
			_, err := strconv.ParseUint(project, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid project ID: %s", project)
			}
		}

		filter := fmt.Sprintf("project_id IN tuple(%s)", strings.Join(projects, ", "))
		return []string{filter}, nil
	}

	return []string{}, errors.New("no project id in the request")
}

func MakeTimeRangeFilter(column string, params url.Values) ([]string, error) {
	filters := make([]string, 0, 2)

	if periodStart := params.Get("start"); periodStart != "" {
		filters = append(filters, fmt.Sprintf("%s >= toDateTime('%s')", column, periodStart))
	} else {
		return filters, errors.New("no range start in the request")
	}

	if periodEnd := params.Get("end"); periodEnd != "" {
		filters = append(filters, fmt.Sprintf("%s < toDateTime('%s')", column, periodEnd))
	} else {
		return filters, errors.New("no range end in the request")
	}

	return filters, nil
}

func MakeAndroidApiLevelFilter(params url.Values) ([]string, error) {
	if levels, exists := params["android_api_level"]; exists && len(levels) > 0 {
		for _, l := range levels {
			_, err := strconv.ParseUint(l, 10, 64)
			if err != nil {
				return nil, errors.New("can't parse android api level")
			}
		}
		filter := fmt.Sprintf("android_api_level IN tuple(%s)", strings.Join(levels, ", "))
		return []string{filter}, nil
	}

	return []string{}, nil
}

func MakeVersionNameAndCodeFilter(params url.Values) ([]string, error) {
	if versions, exists := params["version"]; exists {
		versionBuilds, err := GetVersionBuildFromAppVersions(versions)
		if err != nil {
			return nil, err
		}

		pairs := make([]string, 0, len(versionBuilds))
		for _, vb := range versionBuilds {
			pairs = append(pairs, fmt.Sprintf("(version_name = '%s' AND version_code = '%s')", Escape(vb.Name), Escape(vb.Code)))
		}

		filter := fmt.Sprintf("(%s)", strings.Join(pairs, " OR "))
		return []string{filter}, nil
	}

	return []string{}, nil
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
	return VersionBuild{}, fmt.Errorf("cannot parse application_version: %v", appVersion)
}

func MakeFieldsFilter(fields map[string]string, params url.Values) ([]string, error) {
	filter := make([]string, 0)
	for field, column := range fields {
		values, exists := params[field]
		if !exists {
			continue
		}
		escaped := make([]string, 0, len(values))
		for _, value := range values {
			if value == "" {
				continue
			}
			escaped = append(escaped, fmt.Sprintf("'%s'", Escape(value)))
		}
		if len(escaped) > 0 {
			filter = append(filter, fmt.Sprintf("%s IN tuple(%s)", column, strings.Join(escaped, ", ")))
		}
	}
	return filter, nil
}

func MakeApplicationFilter(params url.Values) ([]string, error) {
	if is_application := params.Get("is_application"); is_application != "" {
		if is_application == "1" {
			return []string{"is_application = 1"}, nil
		} else if is_application == "0" {
			return []string{"is_application = 0"}, nil
		} else {
			return []string{}, fmt.Errorf("cannot parse is_application: %v", is_application)
		}
	}
	return []string{}, nil
}
