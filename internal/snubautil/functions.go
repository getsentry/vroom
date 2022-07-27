package snubautil

import (
	"encoding/json"
	"fmt"

	"github.com/getsentry/sentry-go"
)

type (
	Function struct {
		Name        string   `json:"name"`
		Package     string   `json:"package"`
		Path        string   `json:"path"`
		Fingerprint uint64   `json:"fingerprint"`
		P75         float64  `json:"p75"`
		P95         float64  `json:"p95"`
		P99         float64  `json:"p99"`
		Count       uint64   `json:"count"`
		Worst       string   `json:"worst"`
		Examples    []string `json:"examples"`
	}

	RawFunction struct {
		Name        string        `json:"name"`
		Package     string        `json:"package"`
		Path        string        `json:"path"`
		Fingerprint uint64        `json:"fingerprint"`
		P75         float64       `json:"p75"`
		P95         float64       `json:"p95"`
		P99         float64       `json:"p99"`
		Count       uint64        `json:"count"`
		Worst       interface{}   `json:"worst"`
		Examples    []interface{} `json:"examples"`
	}

	SnubaFunctionsResponse struct {
		Functions []RawFunction `json:"data"`
	}

	Example struct {
		ProfileId string `json:"string"`
		ThreadId  uint64 `json:"thread_id"`
	}
)

func GetFunctions(sqb QueryBuilder) ([]Function, error) {
	t := sentry.TransactionFromContext(sqb.ctx)
	rs := t.StartChild("snuba")
	defer rs.Finish()

	sqb.SelectCols = []string{
		"name",
		"package",
		"path",
		"fingerprint",
		"arrayElement(quantilesMerge(0.75)(percentiles), 1) AS p75",
		"arrayElement(quantilesMerge(0.95)(percentiles), 1) AS p95",
		"arrayElement(quantilesMerge(0.99)(percentiles), 1) AS p99",
		"countMerge(count) AS count",
		"argMaxMerge(worst) AS worst",
		"groupUniqArrayMerge(5)(examples) AS examples",
	}
	sqb.GroupBy = "name, package, path, fingerprint"

	rb, err := sqb.Do(rs)
	if err != nil {
		return nil, err
	}
	defer rb.Close()

	var sr SnubaFunctionsResponse
	err = json.NewDecoder(rb).Decode(&sr)
	if err != nil {
		return nil, err
	}

	functions := make([]Function, len(sr.Functions), len(sr.Functions))

	for i, fn := range sr.Functions {
		worst, err := ParseExample(fn.Worst)
		if err != nil {
			return nil, err
		}

		examples := make([]Example, len(fn.Examples), len(fn.Examples))
		for j, example := range fn.Examples {
			examples[j], err = ParseExample(example)
			if err != nil {
				return nil, err
			}
		}

		exampleProfileIds := make([]string, len(fn.Examples), len(fn.Examples))
		for j, example := range examples {
			exampleProfileIds[j] = example.ProfileId
		}

		functions[i] = Function{
			Name:        fn.Name,
			Package:     fn.Package,
			Path:        fn.Path,
			Fingerprint: fn.Fingerprint,
			P75:         fn.P75,
			P95:         fn.P95,
			P99:         fn.P99,
			Count:       fn.Count,
			Worst:       worst.ProfileId,
			Examples:    exampleProfileIds,
		}
	}

	return functions, nil
}

func ParseExample(ex interface{}) (Example, error) {
	var example Example

	if arr, ok := ex.([]interface{}); ok {
		if len(arr) != 2 {
			return example, fmt.Errorf("expected worst to be an array of length, but has length %d", len(arr))
		}

		if str, ok := arr[0].(string); ok {
			example.ProfileId = str
		} else {
			return example, fmt.Errorf("worst is an array but the first element is not a string")
		}

		if num, ok := arr[1].(float64); ok {
			example.ThreadId = uint64(num)
		} else {
			return example, fmt.Errorf("worst is an array but the second element is not a number")
		}
	} else if str, ok := ex.(string); ok {
		example.ProfileId = str
	} else {
		return example, fmt.Errorf("worst is an unexpected type")
	}

	return example, nil
}
