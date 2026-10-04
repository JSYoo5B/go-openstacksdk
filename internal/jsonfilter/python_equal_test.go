package jsonfilter_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"gophercloudsdk/internal/jsonfilter"
)

func TestEqualPythonJSONSupportsRecursiveBooleanNumberEqualityAndCompleteShape(t *testing.T) {
	for _, tc := range []struct {
		left, right string
		equal       bool
	}{{`true`, `1`, true}, {`1.00`, `true`, true}, {`false`, `-0.0e999999999999999999999`, true}, {`true`, `2`, false}, {`false`, `null`, false}, {`true`, `"1"`, false}, {`[true,{"nested":[false,1]}]`, `[1.0,{"nested":[-0,true]}]`, true}, {`{"a":true,"b":[false]}`, `{"b":[0.0],"a":1e0}`, true}, {`{"a":1,"b":2}`, `{"a":true}`, false}, {`[true,false]`, `[0,1]`, false}, {`{"Case":true}`, `{"case":1}`, false}, {`[]`, `{}`, false}, {`null`, `null`, true}} {
		t.Run(tc.left+"/"+tc.right, func(t *testing.T) {
			equal, err := jsonfilter.EqualPythonJSON(json.RawMessage(tc.left), json.RawMessage(tc.right))
			if err != nil || equal != tc.equal {
				t.Fatal(equal, err)
			}
		})
	}
}

func TestEqualPythonJSONPreservesExactDecimalPrecisionAndLargeExponents(t *testing.T) {
	for _, tc := range []struct {
		left, right string
		equal       bool
	}{{`9007199254740993`, `9007199254740992`, false}, {`900719925474099300`, `9007199254740993e2`, true}, {`1.2345678901234567890123456789`, `1.2345678901234567890123456788`, false}, {`1e999999999999999999999999999999`, `10e999999999999999999999999999998`, true}, {`1e-999999999999999999999999999999`, `0.1e-999999999999999999999999999998`, true}, {`1.0000000000000000000000000001`, `true`, false}, {`0e-999999999999999999999999999999`, `false`, true}} {
		equal, err := jsonfilter.EqualPythonJSON(json.RawMessage(tc.left), json.RawMessage(tc.right))
		if err != nil || equal != tc.equal {
			t.Fatalf("left=%s right=%s equal=%v error=%v", tc.left, tc.right, equal, err)
		}
	}
}

func TestEqualPythonJSONErrorsOwnInputsAndExistingFiltersKeepBooleanNumberDistinction(t *testing.T) {
	left, right := json.RawMessage(`{"a":[true,9007199254740993]}`), json.RawMessage(`{"a":[1,9007199254740993.0]}`)
	beforeLeft, beforeRight := bytes.Clone(left), bytes.Clone(right)
	equal, err := jsonfilter.EqualPythonJSON(left, right)
	if err != nil || !equal || !bytes.Equal(left, beforeLeft) || !bytes.Equal(right, beforeRight) {
		t.Fatal(equal, err)
	}
	for _, invalid := range []json.RawMessage{nil, {}, json.RawMessage(`true false`), json.RawMessage(`{"key":true,}`), json.RawMessage([]byte{'"', 0xff, '"'})} {
		for _, reverse := range []bool{false, true} {
			a, b := invalid, json.RawMessage(`false`)
			if reverse {
				a, b = b, a
			}
			equal, err := jsonfilter.EqualPythonJSON(a, b)
			if err == nil || equal {
				t.Fatal("invalid JSON equality silently returned comparison", equal, err)
			}
		}
	}
	_, err = jsonfilter.EqualPythonJSON(json.RawMessage(`{"key":true,}`), json.RawMessage(`false`))
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatal("JSON parsing cause lost", err)
	}
	equal, err = jsonfilter.EqualJSON(json.RawMessage(`true`), json.RawMessage(`1`))
	if err != nil || equal {
		t.Fatal("existing exact equality changed", equal, err)
	}
	equal, err = jsonfilter.MatchFilters(map[string]json.RawMessage{"id": json.RawMessage(`true`)}, map[string]json.RawMessage{"id": json.RawMessage(`1`)})
	if err != nil || equal {
		t.Fatal("existing Body filter equality changed", equal, err)
	}
}
