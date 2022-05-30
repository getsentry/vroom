package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/vroom/internal/aggregate"
	"github.com/getsentry/vroom/internal/httputil"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/julienschmidt/httprouter"
	"github.com/maruel/natural"
)

type (
	Transaction struct {
		DurationMS    aggregate.Quantiles `json:"duration_ms"`
		LastProfileAt time.Time           `json:"last_profile_at"`
		Name          string              `json:"name"`
		ProfilesCount int                 `json:"profiles_count"`
		ProjectID     string              `json:"project_id"`
		Versions      []string            `json:"versions"`
	}

	GetTransactionsResponse struct {
		Transactions []Transaction `json:"transactions"`
	}
)

func (env *environment) getTransactions(w http.ResponseWriter, r *http.Request) {
	hub := sentry.GetHubFromContext(r.Context())
	p, ok := httputil.GetRequiredQueryParameters(w, r, hub, "project_id", "start", "end")
	if !ok {
		return
	}

	hub.Scope().SetTag("project_id", p["project_id"])

	ctx := r.Context()
	ps := httprouter.ParamsFromContext(r.Context())
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		hub.Scope().SetContext("raw_organization_id", rawOrganizationID)
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hub.Scope().SetTag("organization_id", rawOrganizationID)

	sqb, err := env.snubaQueryBuilderFromRequest(ctx, r.URL.Query())
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sqb.OrderBy = "transaction_name ASC"
	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("organization_id=%d", organizationID),
	)

	transactions, err := snubautil.GetTransactions(sqb)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s := sentry.StartSpan(ctx, "json.marshal")
	defer s.Finish()

	tr := GetTransactionsResponse{
		Transactions: make([]Transaction, 0, len(transactions)),
	}
	for _, t := range transactions {
		tr.Transactions = append(tr.Transactions, snubaTransactionToTransaction(t))
	}

	b, err := json.Marshal(tr)
	if err != nil {
		hub.CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)

}

func snubaTransactionToTransaction(t snubautil.Transaction) Transaction {
	versions := make([]string, 0, len(t.Versions))
	for _, v := range t.Versions {
		versions = append(versions, fmt.Sprintf("%s (build %s)", v[0], v[1]))
	}
	sort.Sort(natural.StringSlice(versions))
	return Transaction{
		DurationMS: aggregate.Quantiles{
			P50: t.DurationNS[0] / 1_000_000,
			P75: t.DurationNS[1] / 1_000_000,
			P90: t.DurationNS[2] / 1_000_000,
			P95: t.DurationNS[3] / 1_000_000,
			P99: t.DurationNS[4] / 1_000_000,
		},
		LastProfileAt: t.LastProfileAt,
		Name:          t.TransactionName,
		ProfilesCount: t.ProfilesCount,
		ProjectID:     strconv.FormatUint(t.ProjectID, 10),
		Versions:      versions,
	}
}
