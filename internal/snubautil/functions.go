package snubautil

import (
	"encoding/json"

	"github.com/getsentry/sentry-go"
)

type (
	Function struct {
		Name        string  `json:"name"`
		Package     string  `json:"package"`
		Path        string  `json:"path"`
		Fingerprint uint64  `json:"fingerprint"`
		P75         float64 `json:"p75"`
		P95         float64 `json:"p95"`
		P99         float64 `json:"p99"`
		Count       uint64  `json:"count"`
	}

	SnubaFunctionsResponse struct {
		Functions []Function `json:"data"`
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
		"arrayElement(quantilesMerge(0.75)(percentiles), 1) AS p99",
		"countMerge(count) AS count",
		"argMaxMerge(worst) AS worst",
		"groupUniqArray(5)(examples) AS examples",
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

	return sr.Functions, nil
}
