package snubautil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gojek/heimdall/v7/httpclient"
)

type (
	Client struct {
		consistent bool
		debug      bool
		dryRun     bool
		http       *httpclient.Client
		hub        *sentry.Hub
		turbo      bool
		url        string
	}

	// query fields
	QueryBuilder struct {
		client *Client
		entity string
		ctx    context.Context

		GroupBy         string
		Limit           int
		Offset          uint64
		OrderBy         string
		SelectCols      []string
		WhereConditions []string
	}

	body struct {
		Query      string `json:"query"`
		Dataset    string `json:"dataset"`
		Turbo      bool   `json:"turbo"`
		Consistent bool   `json:"consistent"`
		Debug      bool   `json:"debug"`
		DryRun     bool   `json:"dry_run"`
		Legacy     bool   `json:"legacy"`
	}

	ErrorResponse struct {
		Error Error `json:"error"`
	}

	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
)

func NewClient(host, port, dataset string, hub *sentry.Hub) (Client, error) {
	if host == "" {
		return Client{}, errors.New("host must be set")
	}
	if dataset == "" {
		return Client{}, errors.New("dataset must be set")
	}
	var u strings.Builder
	u.WriteString(host)
	if port != "" {
		u.WriteString(":")
		u.WriteString(port)
	}
	u.WriteString("/")
	u.WriteString(dataset)
	u.WriteString("/snql")
	return Client{
		url:  u.String(),
		http: httpclient.NewClient(httpclient.WithHTTPTimeout(30 * time.Second)),
	}, nil
}

func (c *Client) NewQuery(ctx context.Context, entity string) (QueryBuilder, error) {
	if entity == "" {
		return QueryBuilder{}, errors.New("no entity selected")
	}
	return QueryBuilder{
		client: c,
		entity: entity,
		ctx:    ctx,
	}, nil
}

func (c Client) URL() string {
	return c.url
}

func (q *QueryBuilder) Query() (string, error) {
	if len(q.SelectCols) == 0 {
		return "", errors.New("no column selected")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("MATCH (%s) SELECT %s", q.entity, strings.Join(q.SelectCols, ", ")))

	if q.GroupBy != "" {
		sb.WriteString(" BY ")
		sb.WriteString(q.GroupBy)
	}

	if len(q.WhereConditions) > 0 {
		sb.WriteString(fmt.Sprintf(" WHERE %s", strings.Join(q.WhereConditions, " AND ")))
	}

	if q.OrderBy != "" {
		sb.WriteString(fmt.Sprintf(" ORDER BY %s", q.OrderBy))
	}

	if q.Limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", q.Limit))
	}

	if q.Offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", q.Offset))
	}

	return sb.String(), nil
}

func (q *QueryBuilder) Do(r *sentry.Span) (io.ReadCloser, error) {
	o := r.StartChild("query_builder")
	o.Description = "Query Snuba"
	defer o.Finish()

	s := o.StartChild("query_builder")
	s.Description = "Prepare Body"
	body, err := q.body(s)
	s.Finish()
	if err != nil {
		return nil, err
	}

	s = o.StartChild("http.client")
	defer s.Finish()

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Add("sentry-trace", s.ToSentryTrace())
	resp, err := q.client.http.Post(q.client.URL(), body, headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		var errResponse ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResponse)
		return nil, fmt.Errorf("error while trying to query snuba. http status: %d, type: %s, message: %s", resp.StatusCode, errResponse.Error.Type, errResponse.Error.Message)
	}

	return resp.Body, nil
}

func (q *QueryBuilder) body(s *sentry.Span) (io.Reader, error) {
	query, err := q.Query()
	if err != nil {
		return nil, err
	}
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}
	s.Data["query"] = query
	spb := body{
		Query:      query,
		Turbo:      q.client.turbo,
		Consistent: q.client.consistent,
		Debug:      q.client.debug,
		DryRun:     q.client.dryRun,
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
