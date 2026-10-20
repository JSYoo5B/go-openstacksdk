package senlin

import (
	"encoding/json"
	"testing"
)

func TestEqualJSONPreservesCompleteObjectsAndExactNumbers(t *testing.T) {
	for _, test := range []struct {
		left, right string
		equal       bool
	}{
		{`{"big":9007199254740993,"fraction":1.25}`, `{"fraction":125e-2,"big":9007199254740993}`, true},
		{`9007199254740993`, `9007199254740992`, false},
		{`1e1000000`, `10e999999`, true},
		{`true`, `1`, false},
		{`false`, `0`, false},
		{`{"x":null}`, `{}`, false},
		{`{"x":1,"y":2}`, `{"x":1}`, false},
		{`[1,{"x":2}]`, `[1.0,{"x":2e0}]`, true},
		{`[1,2]`, `[2,1]`, false},
		{`null`, `null`, true},
		{``, `null`, false},
		{`null null`, `null`, false},
		{`{`, `{`, false},
	} {
		if got := EqualJSON(json.RawMessage(test.left), json.RawMessage(test.right)); got != test.equal {
			t.Errorf("%s vs %s: %v, want %v", test.left, test.right, got, test.equal)
		}
	}
}
