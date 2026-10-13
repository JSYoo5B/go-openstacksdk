package jsonfilter_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

func TestEqualPythonLoadedJSONKeepsCPythonFloatAndIntegerSemanticsSeparateFromExactDefault(t *testing.T) {
	for _, tc := range []struct {
		left, right string
		equal       bool
	}{
		{"1e-400", "0", true}, {"-1e-400", "false", true},
		{"1e400", "10e399", true}, {"-1e400", "1e400", false},
		{"1e400", "" + strings.Repeat("9", 401), false},
		{"9007199254740993.0", "9007199254740993", false},
		{"9007199254740993.0", "9007199254740992", true},
		{"9007199254740993", "9007199254740992", false},
		{"0.1", "0.10000000000000001", true},
		{"true", "1.0", true}, {"false", "-0.0", true},
		{"null", "false", false}, {`"1"`, "1", false},
		{`[true,{"n":1e-400,"large":9007199254740993}]`, `[1,{"large":9007199254740993,"n":0}]`, true},
		{`{"x":1,"x":false}`, `{"x":0}`, true},
		{`[1,2]`, `[2,1]`, false}, {`{}`, `[]`, false},
		{`{"missing":null}`, `{}`, false},
	} {
		for _, pair := range [][2]string{{tc.left, tc.right}, {tc.right, tc.left}} {
			equal, err := jsonfilter.EqualPythonLoadedJSON(json.RawMessage(pair[0]), json.RawMessage(pair[1]))
			if err != nil || equal != tc.equal {
				t.Fatalf("%s == %s: %v %v", pair[0], pair[1], equal, err)
			}
		}
	}
	exact, err := jsonfilter.EqualPythonJSON(json.RawMessage(`1e-400`), json.RawMessage(`0`))
	if err != nil || exact {
		t.Fatal("existing exact decimal policy changed", exact, err)
	}
}

func TestEqualPythonLoadedJSONRejectsMalformedAndInvalidUTF8ForEitherSide(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage(``), json.RawMessage(`0 1`), json.RawMessage(`{"x":}`), json.RawMessage(`NaN`), json.RawMessage(`"` + string([]byte{0xff}) + `"`)} {
		for _, pair := range [][2]json.RawMessage{{raw, json.RawMessage(`null`)}, {json.RawMessage(`null`), raw}} {
			if equal, err := jsonfilter.EqualPythonLoadedJSON(pair[0], pair[1]); err == nil || equal {
				t.Fatal(string(raw), equal, err)
			}
		}
	}
}
