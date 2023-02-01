package timeutil

import (
	"strconv"
	"time"
)

type Time time.Time

func (t *Time) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		return nil
	}
	if s[0] == '"' {
		tt, err := time.Parse(`"`+time.RFC3339+`"`, s)
		if err != nil {
			return err
		}
		*t = Time(tt)
	} else {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*t = Time(time.Unix(i, 0))
	}
	return nil
}

func (t Time) Time() time.Time {
	return time.Time(t)
}
