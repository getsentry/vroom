package transaction

import "time"

type (
	Metadata struct {
		AppIdentifier     string    `json:"app.identifier,omitempty"`
		Dist              string    `json:"dist,omitempty"`
		Environment       string    `json:"environment,omitempty"`
		HTTPMethod        string    `json:"http.method,omitempty"`
		Release           string    `json:"release,omitempty"`
		Transaction       string    `json:"transaction,omitempty"`
		TransactionEnd    time.Time `json:"transaction.end"`
		TransactionOp     string    `json:"transaction.op,omitempty"`
		TransactionStart  time.Time `json:"transaction.start"`
		TransactionStatus string    `json:"transaction.status,omitempty"`
	}
)
