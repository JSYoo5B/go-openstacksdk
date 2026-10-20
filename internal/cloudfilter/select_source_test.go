package cloudfilter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSelectSourceRoundsFloatDataAndLiteralWithoutRoundingIntegers(t *testing.T) {
	rows := []json.RawMessage{json.RawMessage(`{"id":"float","rules":{"x":9007199254740992.0}}`), json.RawMessage(`{"id":"integer","rules":{"x":9007199254740993}}`)}
	query := json.RawMessage(`"[?rules.x == ` + "`9007199254740993.0`" + `].id"`)
	result, err := SelectSource(rows, "", &query, nil)
	if err != nil || string(result.Value) != `["float"]` {
		t.Fatal(result, err)
	}
	baseline, err := SelectLoaded(rows, "", &query, nil)
	if err != nil || string(baseline.Value) != `["integer"]` {
		t.Fatal(baseline, err)
	}
	bigRows := []json.RawMessage{json.RawMessage(`{"id":"float","x":1e23}`), json.RawMessage(`{"id":"integer","x":100000000000000000000000}`)}
	bigQuery := json.RawMessage(`"[?x == ` + "`100000000000000000000000`" + `].id"`)
	bigResult, err := SelectSource(bigRows, "", &bigQuery, nil)
	if err != nil || string(bigResult.Value) != `["integer"]` {
		t.Fatal(bigResult, err)
	}
	floatQuery := json.RawMessage(`"[?x == ` + "`1e23`" + `].id"`)
	floatResult, err := SelectSource(bigRows, "", &floatQuery, nil)
	if err != nil || string(floatResult.Value) != `["float"]` {
		t.Fatal(floatResult, err)
	}
	tiny := json.RawMessage(`"` + "`1e-400`" + `"`)
	r, err := SelectSource(nil, "", &tiny, nil)
	if err != nil || string(r.Value) != "0" {
		t.Fatal(r, err)
	}
	huge := json.RawMessage(`"` + "`1e400`" + `"`)
	if _, err := SelectSource(nil, "", &huge, nil); err == nil || !strings.Contains(err.Error(), "nonfinite") {
		t.Fatal(err)
	}
	if string(rows[0]) != `{"id":"float","rules":{"x":9007199254740992.0}}` {
		t.Fatal("input changed")
	}
}
