package cloudfilter_test

import (
	"encoding/json"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

func TestPythonJSONFloatTruthinessPreservesDefaultExactPolicyAndIntegerPrecision(t *testing.T) {
	for _, tc := range []struct {
		raw         string
		json, exact bool
	}{
		{"1e-400", false, true}, {"-1e-400", false, true}, {"1e400", true, true}, {"-1e400", true, true},
		{"0.0", false, false}, {"-0e900", false, false}, {"9007199254740993", true, true}, {"0", false, false},
		{"false", false, false}, {"null", false, false}, {`""`, false, false}, {`"0"`, true, true}, {"[]", false, false}, {"{}", false, false}, {"[null]", true, true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := cloudfilter.PythonJSONTruthy(json.RawMessage(tc.raw))
			if err != nil || got != tc.json {
				t.Fatal(got, err)
			}
			exact, err := cloudfilter.PythonTruthy(json.RawMessage(tc.raw))
			if err != nil || exact != tc.exact {
				t.Fatal(exact, err)
			}
		})
	}
}
func TestPythonJSONTruthyRejectsIncompleteAndInvalidUTF8WithoutChangingMissingDefault(t *testing.T) {
	if got, err := cloudfilter.PythonJSONTruthy(nil); got || err != nil {
		t.Fatal(got, err)
	}
	for _, raw := range []json.RawMessage{{}, json.RawMessage("1e"), json.RawMessage("null false"), {34, 255, 34}} {
		if got, err := cloudfilter.PythonJSONTruthy(raw); got || err == nil {
			t.Fatal(raw, got, err)
		}
	}
}
