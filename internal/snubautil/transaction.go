package snubautil

import (
	"encoding/json"
	"time"

	"github.com/getsentry/sentry-go"
)

type (
	SnubaTransactionsResponse struct {
		Transactions []Transaction `json:"data"`
	}
	Transaction struct {
		DurationNS      []float64   `json:"duration_ns"`
		LastProfileAt   time.Time   `json:"last_profile_at"`
		ProfilesCount   int         `json:"profiles_count"`
		ProjectID       uint64      `json:"project_id"`
		TransactionName string      `json:"transaction_name"`
		Versions        [][2]string `json:"versions"`
	}
)

func GetTransactions(sqb QueryBuilder) ([]Transaction, error) {
	t := sentry.TransactionFromContext(sqb.ctx)
	rs := t.StartChild("snuba")
	defer rs.Finish()

	sqb.SelectCols = []string{
		"project_id",
		"transaction_name",
		"groupUniqArray(tuple(version_name, version_code)) AS versions",
		"quantiles(0.5, 0.75, 0.9, 0.95, 0.99)(duration_ns) AS duration_ns",
		"anyLast(received) AS last_profile_at",
		"count() AS profiles_count",
	}
	sqb.GroupBy = "project_id, transaction_name"
	sqb.OrderBy = "transaction_name ASC"

	rb, err := sqb.Do(rs)
	if err != nil {
		return nil, err
	}
	defer rb.Close()

	s := rs.StartChild("json.unmarshal")
	s.Description = "Decode response from Snuba"
	defer s.Finish()

	var sr SnubaTransactionsResponse
	err = json.NewDecoder(rb).Decode(&sr)
	if err != nil {
		return nil, err
	}

	if len(sr.Transactions) == 0 {
		return []Transaction{}, nil
	}

	return sr.Transactions, err
}
