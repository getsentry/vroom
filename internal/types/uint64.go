package types

import (
	"encoding/json"
	"strconv"
)

type (
	Uint64 uint64
)

func (u Uint64) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatUint(uint64(u), 10))
}

func (u *Uint64) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	var s string
	if b[0] == '"' {
		err := json.Unmarshal(b, &s)
		if err != nil {
			return err
		}
	} else {
		s = string(b)
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*u = Uint64(v)
	return nil
}
