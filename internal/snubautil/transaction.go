package snubautil

import (
	"encoding/json"
	"time"
)

type (
	SnubaTransactionsResponse struct {
		Transactions []Transaction `json:"data"`
	}
	Transaction struct {
		DurationNS      []float64   `json:"duration_ns"`
		LastProfileAt   time.Time   `json:"last_profile_at"`
		ProfilesCount   int         `json:"profiles_count"`
		TransactionName string      `json:"transaction_name"`
		Versions        [][2]string `json:"versions"`
	}
)

func GetTransactions(sqb SnubaQueryBuilder) ([]Transaction, error) {
	sqb.SelectCols = []string{
		"transaction_name",
		"groupUniqArray(tuple(version_name, version_code)) AS versions",
		"quantiles(0.5, 0.75, 0.9, 0.95, 0.99)(duration_ns) AS duration_ns",
		"anyLast(received) AS last_profile_at",
		"count() AS profiles_count",
	}
	sqb.GroupBy = "transaction_name"
	sqb.OrderBy = "transaction_name ASC"

	r, err := sqb.Do()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var sr SnubaTransactionsResponse
	err = json.NewDecoder(r).Decode(&sr)
	if err != nil {
		return nil, err
	}

	if len(sr.Transactions) == 0 {
		return []Transaction{}, nil
	}

	return sr.Transactions, err
}
