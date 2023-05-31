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
		turbo      bool
		url        string
	}

	// QueryBuilder offers a way to build a query for Snuba.
	QueryBuilder struct {
		client *Client
		entity string
		ctx    context.Context

		Granularity     uint64
		GroupBy         string
		Limit           uint64
		Offset          uint64
		OrderBy         string
		OrganizationID  uint64
		SelectCols      []string
		WhereConditions []string
	}

	Tenant struct {
		OrganizationID uint64 `json:"organization_id"`
		Referrer       string `json:"referrer"`
	}

	body struct {
		Query      string `json:"query"`
		Dataset    string `json:"dataset"`
		Turbo      bool   `json:"turbo"`
		Consistent bool   `json:"consistent"`
		AppID      string `json:"app_id"`
		Tenant     Tenant `json:"tenant_ids"`
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

func NewClient(host, dataset string) (Client, error) {
	if host == "" {
		return Client{}, errors.New("host must be set")
	}
	if dataset == "" {
		return Client{}, errors.New("dataset must be set")
	}
	return Client{
		url:  fmt.Sprintf("%s/%s/snql", host, dataset),
		http: httpclient.NewClient(httpclient.WithHTTPTimeout(30 * time.Second)),
	}, nil
}

func (c *Client) NewQuery(ctx context.Context, entity string, orgID uint64) (QueryBuilder, error) {
	if entity == "" {
		return QueryBuilder{}, errors.New("no entity selected")
	}
	if orgID == 0 {
		return QueryBuilder{}, errors.New("no org ID")
	}
	return QueryBuilder{
		OrganizationID: orgID,
		client:         c,
		ctx:            ctx,
		entity:         entity,
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

	if q.Granularity > 0 {
		sb.WriteString(fmt.Sprintf(" GRANULARITY %d", q.Granularity))
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
	headers.Set("content-type", "application/json")
	headers.Set("sentry-trace", s.ToSentryTrace())
	headers.Set("referer", "api.vroom")
	resp, err := q.client.http.Post(q.client.URL(), body, headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		var errResponse ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResponse)
		return nil, fmt.Errorf(
			"error while trying to query snuba. http status: %d, type: %s, message: %s",
			resp.StatusCode,
			errResponse.Error.Type,
			errResponse.Error.Message,
		)
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
	sqb := body{
		Query:      query,
		Turbo:      q.client.turbo,
		Tenant:     Tenant{Referrer: "vroom", OrganizationID: q.OrganizationID},
		Consistent: q.client.consistent,
		Debug:      q.client.debug,
		DryRun:     q.client.dryRun,
	}
	body, err := json.Marshal(sqb)
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
