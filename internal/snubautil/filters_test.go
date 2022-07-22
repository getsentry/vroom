package snubautil

import (
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/getsentry/vroom/internal/testutil"
	"github.com/google/go-cmp/cmp"
)

func assertFails(t *testing.T, err error, contains string) {
	if err == nil {
		t.Fatal("expected error to be non-nil")
	}
	if !strings.Contains(err.Error(), contains) {
		t.Fatalf("expected error message to contain %q but was %q", contains, err.Error())
	}
}

func assertNoErr(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("expected no error: %q", err.Error())
	}
}

func assertFilterEquals(t *testing.T, actual []string, expected []string) {
	// the order of the filters are unimportant since they will be ANDed together
	trans := cmp.Transformer("Sort", func(in []string) []string {
		out := append([]string(nil), in...)
		sort.Strings(out)
		return out
	})

	if diff := testutil.Diff(actual, expected, trans); diff != "" {
		t.Fatalf(`expected "%v" but was "%v"`, expected, actual)
	}
}

func TestMakeFilterFuncs(t *testing.T) {
	tests := []struct {
		name           string
		makeFilterFunc func(url.Values) ([]string, error)
		params         url.Values
		filter         []string
		err            string
	}{
		{
			name:           "no projects",
			makeFilterFunc: MakeProjectsFilter,
			params:         map[string][]string{},
			filter:         nil,
			err:            "no project id in the request",
		},
		{
			name:           "invalid project",
			makeFilterFunc: MakeProjectsFilter,
			params: map[string][]string{
				"project_id": []string{"a"},
			},
			filter: nil,
			err:    "invalid project ID: a",
		},
		{
			name:           "single project",
			makeFilterFunc: MakeProjectsFilter,
			params: map[string][]string{
				"project_id": []string{"1"},
			},
			filter: []string{"project_id IN tuple(1)"},
			err:    "",
		},
		{
			name:           "multiple projects",
			makeFilterFunc: MakeProjectsFilter,
			params: map[string][]string{
				"project_id": []string{"1", "2", "3"},
			},
			filter: []string{"project_id IN tuple(1, 2, 3)"},
			err:    "",
		},
		{
			name: "no timestamp start",
			makeFilterFunc: func(params url.Values) ([]string, error) {
				return MakeTimeRangeFilter("timestamp", params)
			},
			params: map[string][]string{},
			filter: nil,
			err:    "no range start in the request",
		},
		{
			name: "no timestamp end",
			makeFilterFunc: func(params url.Values) ([]string, error) {
				return MakeTimeRangeFilter("timestamp", params)
			},
			params: map[string][]string{
				"start": []string{"2022-01-01T00:00:00.000000+00:00"},
			},
			filter: nil,
			err:    "no range end in the request",
		},
		{
			name: "start and end timestamps",
			makeFilterFunc: func(params url.Values) ([]string, error) {
				return MakeTimeRangeFilter("timestamp", params)
			},
			params: map[string][]string{
				"start": []string{"2022-01-01T00:00:00.000000+00:00"},
				"end":   []string{"2023-01-01T00:00:00.000000+00:00"},
			},
			filter: []string{
				"timestamp >= toDateTime('2022-01-01T00:00:00.000000+00:00')",
				"timestamp < toDateTime('2023-01-01T00:00:00.000000+00:00')",
			},
			err: "",
		},
		{
			name:           "no android api level",
			makeFilterFunc: MakeAndroidApiLevelFilter,
			params:         map[string][]string{},
			filter:         []string{},
			err:            "",
		},
		{
			name:           "bad android api level",
			makeFilterFunc: MakeAndroidApiLevelFilter,
			params: map[string][]string{
				"android_api_level": []string{"a"},
			},
			filter: nil,
			err:    "can't parse android api level",
		},
		{
			name:           "single android api level",
			makeFilterFunc: MakeAndroidApiLevelFilter,
			params: map[string][]string{
				"android_api_level": []string{"1"},
			},
			filter: []string{"android_api_level IN tuple(1)"},
			err:    "",
		},
		{
			name:           "multiple android api level",
			makeFilterFunc: MakeAndroidApiLevelFilter,
			params: map[string][]string{
				"android_api_level": []string{"1", "2", "3"},
			},
			filter: []string{"android_api_level IN tuple(1, 2, 3)"},
			err:    "",
		},
		{
			name:           "no version name or code",
			makeFilterFunc: MakeVersionNameAndCodeFilter,
			params:         map[string][]string{},
			filter:         []string{},
			err:            "",
		},
		{
			name:           "invalid version name or code",
			makeFilterFunc: MakeVersionNameAndCodeFilter,
			params: map[string][]string{
				"version": []string{"foo"},
			},
			filter: nil,
			err:    "cannot parse application_version: foo",
		},
		{
			name:           "single version name or code",
			makeFilterFunc: MakeVersionNameAndCodeFilter,
			params: map[string][]string{
				"version": []string{"12 (34)"},
			},
			filter: []string{
				"((version_name = '12' AND version_code = '34'))",
			},
			err: "",
		},
		{
			name:           "multiple version name or code",
			makeFilterFunc: MakeVersionNameAndCodeFilter,
			params: map[string][]string{
				"version": []string{"12 (34)", "56 (78)"},
			},
			filter: []string{
				"((version_name = '12' AND version_code = '34') OR (version_name = '56' AND version_code = '78'))",
			},
			err: "",
		},
		{
			name: "no filters",
			makeFilterFunc: func(params url.Values) ([]string, error) {
				fields := map[string]string{
					"a": "a",
					"b": "b",
					"c": "c",
				}
				return MakeFieldsFilter(fields, params)
			},
			params: map[string][]string{},
			filter: []string{},
			err:    "",
		},
		{
			name: "skips empty profile filters",
			makeFilterFunc: func(params url.Values) ([]string, error) {
				fields := map[string]string{
					"a": "a",
					"b": "b",
					"c": "c",
				}
				return MakeFieldsFilter(fields, params)
			},
			params: map[string][]string{"platform": []string{""}},
			filter: []string{},
			err:    "",
		},
		{
			name: "single value profile filters",
			makeFilterFunc: func(params url.Values) ([]string, error) {
				fields := map[string]string{
					"a": "a",
					"b": "b",
					"c": "c",
				}
				return MakeFieldsFilter(fields, params)
			},
			params: map[string][]string{
				"a": []string{"a"},
				"b": []string{"b"},
				"c": []string{"c"},
			},
			filter: []string{
				"a IN tuple('a')",
				"b IN tuple('b')",
				"c IN tuple('c')",
			},
			err: "",
		},
		{
			name:           "skips empty is_application",
			makeFilterFunc: MakeApplicationFilter,
			params:         map[string][]string{},
			filter:         []string{},
			err:            "",
		},
		{
			name:           "is_application is true",
			makeFilterFunc: MakeApplicationFilter,
			params: map[string][]string{
				"is_application": []string{"1"},
			},
			filter: []string{"is_application = 1"},
			err:    "",
		},
		{
			name:           "is_application is false",
			makeFilterFunc: MakeApplicationFilter,
			params: map[string][]string{
				"is_application": []string{"0"},
			},
			filter: []string{"is_application = 0"},
			err:    "",
		},
		{
			name:           "invalid is_application",
			makeFilterFunc: MakeApplicationFilter,
			params: map[string][]string{
				"is_application": []string{"asdf"},
			},
			filter: []string{},
			err:    "cannot parse is_application: asdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := tt.makeFilterFunc(tt.params)
			if tt.err != "" {
				assertFails(t, err, tt.err)
			} else {
				assertFilterEquals(t, filter, tt.filter)
			}
		})
	}
}
