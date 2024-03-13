package utils

import "encoding/json"

type Options struct {
	ProjectDSN string `json:"dsn"`
}

func (o Options) MarshalJSON() ([]byte, error) {
	return json.Marshal(nil)
}
