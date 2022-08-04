package snubautil

import (
	"encoding/json"

	"github.com/getsentry/sentry-go"
)

type (
	Function struct {
		Name        string   `json:"name"`
		Package     string   `json:"package"`
		Path        string   `json:"path"`
		Fingerprint uint64   `json:"fingerprint"`
		Min         float64  `json:"min"`
		P01         float64  `json:"p01"`
		P25         float64  `json:"p25"`
		P50         float64  `json:"p50"`
		P75         float64  `json:"p75"`
		P99         float64  `json:"p99"`
		Max         float64  `json:"max"`
		Count       uint64   `json:"count"`
		Worst       string   `json:"worst"`
		Examples    []string `json:"examples"`
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
		"minMerge(min) AS min",
		"arrayElement(quantilesMerge(0.01, 0.25, 0.50, 0.75, 0.99)(percentiles) AS quantiles, 1) AS p01",
		"arrayElement(quantiles, 2) AS p25",
		"arrayElement(quantiles, 3) AS p50",
		"arrayElement(quantiles, 4) AS p75",
		"arrayElement(quantiles, 5) AS p99",
		"maxMerge(max) AS max",
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

	return sr.Functions, nil
}
