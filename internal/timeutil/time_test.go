package timeutil

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestParseInt64Timeutil(t *testing.T) {
	var tt Time
	b := []byte(`1675277158`)
	err := json.Unmarshal(b, &tt)
	if err != nil {
		t.Fatalf("error while parsing: %+v\n", err)
	}
	if string(b) != strconv.FormatInt(tt.Time().Unix(), 10) {
		t.Fatalf("wanted: %+v, got: %+v\n", string(b), tt.Time().Unix())
	}
}
func TestParseStringTimeutil(t *testing.T) {
	var tt Time
	b := []byte(`"2023-01-01T12:00:00+00:00"`)
	err := json.Unmarshal(b, &tt)
	if err != nil {
		t.Fatalf("error while parsing: %+v\n", err)
	}
	ttf := tt.Time().Format(`"2006-01-02T15:04:05-07:00"`)
	if string(b) != ttf {
		t.Fatalf("wanted: %+v, got: %+v\n", string(b), ttf)
	}
}
