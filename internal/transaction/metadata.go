package transaction

import "time"

type (
	Metadata struct {
		Dist              string    `json:"dist"`
		Environment       string    `json:"environment"`
		HTTPMethod        string    `json:"http.method"`
		Release           string    `json:"release"`
		Transaction       string    `json:"transaction"`
		TransactionEnd    time.Time `json:"transaction.end"`
		TransactionOp     string    `json:"transaction.op"`
		TransactionStart  time.Time `json:"transaction.start"`
		TransactionStatus string    `json:"transaction.status"`
	}
)
