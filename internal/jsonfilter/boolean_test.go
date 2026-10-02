package jsonfilter

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestBooleanJSONResponseTruthinessAndExactNumericZero(t *testing.T) {
	for _, check := range []struct{ input, expected string }{
		{`null`, `null`}, {`true`, `true`}, {`false`, `false`},
		{`0`, `false`}, {`-0.000e+900000000000000000000`, `false`},
		{`0E-999999999999999999999`, `false`}, {`1e-999999999999999999999`, `true`},
		{`-0.001e999999999999999999999`, `true`}, {`9007199254740993`, `true`},
		{`""`, `false`}, {`"false"`, `true`}, {`"0"`, `true`}, {`"\u0000"`, `true`},
		{`[]`, `false`}, {`[null]`, `true`}, {`[false,0,""]`, `true`},
		{`{}`, `false`}, {`{"n":null}`, `true`}, {`{"n":0}`, `true`},
	} {
		t.Run(check.input, func(t *testing.T) {
			got, err := BooleanJSON(json.RawMessage(check.input))
			if err != nil || string(got) != check.expected {
				t.Fatalf("got=%s expected=%s err=%v", got, check.expected, err)
			}
		})
	}
	// Exponent length is input-sized; normalization never allocates its value
	// as a big integer or emits an exponent-sized decimal expansion.
	for _, coefficient := range []string{"0.000", "-0.001"} {
		input := json.RawMessage(coefficient + "e-" + strings.Repeat("9", 10000))
		got, err := BooleanJSON(input)
		want := "false"
		if coefficient == "-0.001" {
			want = "true"
		}
		if err != nil || string(got) != want {
			t.Fatal(string(got), err)
		}
	}
}

func TestBooleanJSONRejectsMalformedAndMultipleValuesWithCauses(t *testing.T) {
	for _, input := range []string{"", `tru`, `{]`, `[1,]`, `01`, `NaN`, `true false`, `null {]`, `[] trailing`} {
		t.Run(input, func(t *testing.T) {
			got, err := BooleanJSON(json.RawMessage(input))
			if err == nil || got != nil {
				t.Fatalf("got=%s err=%v", got, err)
			}
		})
	}
	_, err := BooleanJSON(json.RawMessage(`null {]`))
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatal("lost trailing JSON cause", err)
	}
	_, err = BooleanJSON(nil)
	if !errors.Is(err, io.EOF) {
		t.Fatal("lost empty JSON cause", err)
	}
}

func TestBooleanJSONOwnsBytesAndLeavesCallerEqualityUnchanged(t *testing.T) {
	input := json.RawMessage(`true`)
	got, err := BooleanJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	got[0] = 'x'
	if string(input) != "true" {
		t.Fatal("normalization aliases input")
	}
	again, err := BooleanJSON(input)
	if err != nil || string(again) != "true" {
		t.Fatal("results share backing bytes", string(again), err)
	}
	actual, err := BooleanJSON(json.RawMessage(`1`))
	if err != nil {
		t.Fatal(err)
	}
	for _, filter := range []string{`1`, `"true"`} {
		matched, err := MatchFilters(map[string]json.RawMessage{"b": actual}, map[string]json.RawMessage{"b": json.RawMessage(filter)})
		if err != nil || matched {
			t.Fatal("caller filter was coerced", filter, matched, err)
		}
	}
}
