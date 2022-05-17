package snubautil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type (
	SnubaQueryBuilder struct {
		Endpoint   string
		Port       int
		Dataset    string
		Turbo      bool
		Consistent bool
		Debug      bool
		DryRun     bool
		Legacy     bool

		// query fields
		Entity          string
		SelectCols      []string
		WhereConditions []string
		OrderBy         string
		Limit           int
		Offset          uint64
	}

	SnubaPostBody struct {
		Query      string `json:"query"`
		Dataset    string `json:"dataset"`
		Turbo      bool   `json:"turbo"`
		Consistent bool   `json:"consistent"`
		Debug      bool   `json:"debug"`
		DryRun     bool   `json:"dry_run"`
		Legacy     bool   `json:"legacy"`
	}
)

func (sqb *SnubaQueryBuilder) URL() (string, error) {
	if sqb.Endpoint == "" {
		return "", errors.New("endpoint must be set")
	}
	if sqb.Dataset == "" {
		return "", errors.New("dataset must be set")
	}
	var sb strings.Builder
	sb.WriteString(sqb.Endpoint)
	if sqb.Port != 0 {
		sb.WriteString(fmt.Sprintf(":%d", sqb.Port))
	}
	sb.WriteString(fmt.Sprintf("/%s/snql", sqb.Dataset))
	return sb.String(), nil
}

func (sqb *SnubaQueryBuilder) query() (string, error) {
	if len(sqb.SelectCols) == 0 {
		return "", errors.New("no column selected")
	}
	if sqb.Entity == "" {
		return "", errors.New("no entity selected")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("MATCH (%s) SELECT %s", sqb.Entity, strings.Join(sqb.SelectCols, ", ")))
	if len(sqb.WhereConditions) > 0 {
		sb.WriteString(fmt.Sprintf(" WHERE %s", strings.Join(sqb.WhereConditions, " AND ")))
	}

	if sqb.OrderBy != "" {
		sb.WriteString(fmt.Sprintf(" ORDER BY %s", sqb.OrderBy))
	}

	if sqb.Limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", sqb.Limit))
	}

	if sqb.Offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", sqb.Offset))
	}

	return sb.String(), nil
}

func (sqb *SnubaQueryBuilder) Do() (io.ReadCloser, error) {
	url, err := sqb.URL()
	if err != nil {
		return nil, err
	}

	body, err := sqb.body()
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error while trying to query snuba. Http status code: %d. Snuba error: %s", resp.StatusCode, string(b))
	}

	return resp.Body, nil
}

func (sqb *SnubaQueryBuilder) body() (io.Reader, error) {
	query, err := sqb.query()
	if err != nil {
		return nil, err
	}
	spb := SnubaPostBody{
		Query:      query,
		Dataset:    sqb.Dataset,
		Turbo:      sqb.Turbo,
		Consistent: sqb.Consistent,
		Debug:      sqb.Debug,
		DryRun:     sqb.DryRun,
		Legacy:     sqb.Legacy,
	}
	body, err := json.Marshal(spb)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(body), nil
}

func Escape(input string) string {
	input = strings.ReplaceAll(input, "\\", "\\\\")
	input = strings.ReplaceAll(input, "'", "\\'")
	return strings.ReplaceAll(input, "\n", "\\n")
}
