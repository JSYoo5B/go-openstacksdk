package cloudfilter

import (
	"encoding/json"
	"testing"
)

func TestFirstLoadedMatchesPythonFloatUnderflowBeforeLength(t *testing.T) {
	for _, raw := range []string{"1e-400", "-1e-400", "0.0", "-0.0", "false", "null", "[]", "{}", `""`} {
		v, err := FirstLoaded(json.RawMessage(raw))
		if err != nil || v != nil {
			t.Fatal(raw, v, err)
		}
	}
	for _, raw := range []string{"1e-300", "true", "1", "1e400"} {
		if _, err := FirstLoaded(json.RawMessage(raw)); err == nil {
			t.Fatal(raw, "nonzero scalar has no length")
		}
	}
	for _, row := range []struct{ input, want string }{{"[false]", "false"}, {"[0]", "0"}, {`[""]`, `""`}, {`[{"id":"owned"}]`, `{"id":"owned"}`}, {`"한"`, `"한"`}} {
		value, err := FirstLoaded(json.RawMessage(row.input))
		if err != nil || string(value) != row.want {
			t.Fatal(row, value, err)
		}
	}
	if v, err := FirstLoaded(nil); v != nil || err != nil {
		t.Fatal(v, err)
	}
	if _, err := First(json.RawMessage("1e-400")); err == nil {
		t.Fatal("legacy exact-decimal First changed")
	}
	if _, err := FirstLoaded(json.RawMessage("[1,2]")); err == nil {
		t.Fatal("ambiguity disappeared")
	}
	if _, err := FirstLoaded(json.RawMessage("[")); err == nil {
		t.Fatal("invalid JSON accepted")
	}
}
