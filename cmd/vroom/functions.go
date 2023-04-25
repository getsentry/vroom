package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/httputil"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/julienschmidt/httprouter"
)

var legalOrderBys = map[string]struct{}{"p75": {}, "p95": {}, "p99": {}, "count": {}, "sum": {}}

type (
	GetFunctionsResponse struct {
		Functions []snubautil.Function `json:"functions"`
	}
)

func (env *environment) getFunctions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hub := sentry.GetHubFromContext(ctx)
	p, ok := httputil.GetRequiredQueryParameters(w, r, "project_id", "start", "end", "transaction_name")
	if !ok {
		return
	}

	hub.Scope().SetTags(p)

	ps := httprouter.ParamsFromContext(ctx)
	rawOrganizationID := ps.ByName("organization_id")
	_, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	rawProjectID := ps.ByName("project_id")
	projectID, err := strconv.ParseUint(rawProjectID, 10, 64)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("project_id", rawProjectID)

	queryParams := r.URL.Query()
	sqb, err := env.functionsQueryBuilderFromRequest(ctx, queryParams)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("project_id=%d", projectID),
	)

	rawOrderBy := queryParams.Get("sort")
	if rawOrderBy == "" {
		hub.CaptureException(errors.New("no sort in the request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	direction := "ASC"
	if strings.HasPrefix(rawOrderBy, "-") {
		direction = "DESC"
		rawOrderBy = strings.TrimPrefix(rawOrderBy, "-")
	}
	if _, exists := legalOrderBys[rawOrderBy]; !exists {
		hub.CaptureException(fmt.Errorf("unknown sort: %s", rawOrderBy))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sqb.OrderBy = strings.Join([]string{rawOrderBy, direction}, " ")

	functions, err := snubautil.GetFunctions(sqb)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(GetFunctionsResponse{
		Functions: functions,
	})
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
