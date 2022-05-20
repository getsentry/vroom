package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/getsentry/vroom/internal/aggregate"
	"github.com/getsentry/vroom/internal/snubautil"
	"github.com/julienschmidt/httprouter"
	"github.com/maruel/natural"
	"github.com/rs/zerolog/log"
)

type (
	Transaction struct {
		DurationMS    aggregate.Quantiles `json:"duration_ms"`
		LastProfileAt time.Time           `json:"last_profile_at"`
		Name          string              `json:"name"`
		ProfilesCount int                 `json:"profiles_count"`
		Versions      []string            `json:"versions"`
	}

	GetTransactionsResponse struct {
		Transactions []Transaction `json:"transactions"`
	}
)

func (env *environment) getTransactions(w http.ResponseWriter, r *http.Request) {
	ps := httprouter.ParamsFromContext(r.Context())
	rawOrganizationID := ps.ByName("organization_id")
	organizationID, err := strconv.ParseUint(rawOrganizationID, 10, 64)
	if err != nil {
		log.Err(err).Str("raw_organization_id", rawOrganizationID).Msg("aggregate: invalid organization_id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	logger := log.With().Uint64("organization_id", organizationID).Logger()
	sqb, err := snubaQueryBuilderFromRequest(r.URL.Query())
	if err != nil {
		logger.Err(err).Msg("aggregate: cannot build snuba query from request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sqb.Endpoint = env.SnubaHost
	sqb.Port = env.SnubaPort
	sqb.Dataset = "profiles"
	sqb.Entity = "profiles"
	sqb.WhereConditions = append(sqb.WhereConditions,
		fmt.Sprintf("organization_id=%d", organizationID),
	)
	transactions, err := snubautil.GetTransactions(sqb)
	if err != nil {
		logger.Err(err).Msg("aggregate: cannot fetch profile data from snuba")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tr := GetTransactionsResponse{
		Transactions: make([]Transaction, 0, len(transactions)),
	}
	for _, t := range transactions {
		tr.Transactions = append(tr.Transactions, snubaTransactionToTransaction(t))
	}
	b, err := json.Marshal(tr)
	if err != nil {
		logger.Err(err).Msg("aggregate: error creating chrome trace data")
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
		Versions:      versions,
	}
}
