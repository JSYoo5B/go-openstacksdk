package jsonfilter

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestMatchFiltersRecursiveSubsetAndNull(t *testing.T) {
	for _, tc := range []struct {
		name, actual, filter string
		missing              bool
		want                 bool
	}{
		{"recursive subset", `{"outer":{"value":1,"extra":true},"extra":false}`, `{"outer":{"value":1}}`, false, true},
		{"nested mismatch", `{"outer":{"value":2}}`, `{"outer":{"value":1}}`, false, false},
		{"missing scalar is null", "", `null`, true, true},
		{"explicit null", `null`, `null`, false, true},
		{"missing string", "", `""`, true, false},
		{"missing nested null", `{"present":1}`, `{"absent":null}`, false, true},
		{"empty actual object", `{}`, `{}`, false, false},
		{"empty actual object with null subset", `{}`, `{"absent":null}`, false, false},
		{"empty filter nonempty object", `{"present":null}`, `{}`, false, true},
		{"empty nested actual", `{"outer":{}}`, `{"outer":{}}`, false, false},
		{"empty nested filter", `{"outer":{"present":false}}`, `{"outer":{}}`, false, true},
		{"missing nested object", `{"present":1}`, `{"outer":{}}`, false, false},
		{"null actual object", `null`, `{}`, false, false},
		{"array actual object", `[1]`, `{}`, false, false},
		{"scalar actual object", `1`, `{}`, false, false},
		{"unicode string equality", `"\u0061"`, `"a"`, false, true},
		{"string mismatch", `"a"`, `"b"`, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]json.RawMessage{}
			if !tc.missing {
				body["field"] = json.RawMessage(tc.actual)
			}
			got, err := MatchFilters(body, map[string]json.RawMessage{"field": json.RawMessage(tc.filter)})
			if err != nil || got != tc.want {
				t.Fatalf("match=%v want=%v error=%v", got, tc.want, err)
			}
		})
	}
	if match, err := MatchFilters(nil, nil); err != nil || !match {
		t.Fatalf("empty filters: match=%v error=%v", match, err)
	}
}

func TestMatchFiltersArraysCompareExactly(t *testing.T) {
	for _, tc := range []struct {
		name, actual, filter string
		want                 bool
	}{
		{"ordered values", `[1,"a",null,true]`, `[1.0,"a",null,true]`, true},
		{"different order", `[1,2]`, `[2,1]`, false},
		{"different length", `[1,2]`, `[1]`, false},
		{"empty arrays", `[]`, `[]`, true},
		{"null is not array", `null`, `[]`, false},
		{"array object key order", `[{"a":1,"b":2}]`, `[{"b":2.0,"a":1e0}]`, true},
		{"array objects are not subsets", `[{"a":1,"extra":2}]`, `[{"a":1}]`, false},
		{"array object missing is not null", `[{"b":null}]`, `[{"a":null}]`, false},
		{"empty array objects use exact equality", `[{}]`, `[{}]`, true},
		{"nested array mismatch", `[[1,2]]`, `[[1]]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MatchFilters(map[string]json.RawMessage{"field": json.RawMessage(tc.actual)}, map[string]json.RawMessage{"field": json.RawMessage(tc.filter)})
			if err != nil || got != tc.want {
				t.Fatalf("match=%v want=%v error=%v", got, tc.want, err)
			}
		})
	}
}

func TestMatchFiltersExactDecimalNumbersAndHugeExponents(t *testing.T) {
	huge := "1" + strings.Repeat("0", 256)
	previous := strings.Repeat("9", 256)
	next := "1" + strings.Repeat("0", 255) + "1"
	for _, tc := range []struct {
		name, actual, filter string
		want                 bool
	}{
		{"integer and fraction", `1`, `1.0`, true},
		{"integer and exponent", `1`, `1e0`, true},
		{"positive signed exponent", `1`, `1.00E+000`, true},
		{"significand trailing zeros", `12000`, `12e3`, true},
		{"fraction leading zeros", `0.00000120`, `12e-7`, true},
		{"negative equality", `-12.3000e2`, `-1230`, true},
		{"different signs", `-1`, `1`, false},
		{"signed zero", `-0.000e1000000000`, `0`, true},
		{"large neighboring integers", `9007199254740993`, `9007199254740992`, false},
		{"large integer decimal equality", `90071992547409930e-1`, `9007199254740993`, true},
		{"large fraction mismatch", `9007199254740993.1`, `9007199254740993.2`, false},
		{"billion positive exponent", `1e1000000000`, `10e999999999`, true},
		{"billion negative exponent", `1e-1000000000`, `10e-1000000001`, true},
		{"exponent beyond machine range", "1e" + huge, "10e" + previous, true},
		{"negative exponent beyond machine range", "1e-" + huge, "10e-" + next, true},
		{"huge exponent mismatch", "1e" + huge, "1e" + previous, false},
		{"huge exponent zero", "-0e" + huge, `0.0`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MatchFilters(map[string]json.RawMessage{"field": json.RawMessage(tc.actual)}, map[string]json.RawMessage{"field": json.RawMessage(tc.filter)})
			if err != nil || got != tc.want {
				t.Fatalf("match=%v want=%v error=%v", got, tc.want, err)
			}
		})
	}
}

func TestMatchFiltersDeliberatelyDistinguishesBooleansAndNumbers(t *testing.T) {
	for _, tc := range []struct{ actual, filter string }{
		{`true`, `1`}, {`1`, `true`}, {`false`, `0`}, {`0`, `false`},
		{`{"enabled":true}`, `{"enabled":1}`}, {`[true]`, `[1]`},
		{`[{"enabled":false}]`, `[{"enabled":0}]`}, {`"1"`, `1`},
	} {
		got, err := MatchFilters(map[string]json.RawMessage{"field": json.RawMessage(tc.actual)}, map[string]json.RawMessage{"field": json.RawMessage(tc.filter)})
		if err != nil || got {
			t.Fatalf("JSON types compared equally: actual=%s filter=%s error=%v", tc.actual, tc.filter, err)
		}
	}
}

func TestMatchFiltersRejectsInvalidJSONWithoutHidingItBehindMismatch(t *testing.T) {
	for _, raw := range []string{"", "{", "null false", "true trailing", "01", "1.", "1e", "NaN", "Infinity", `{"nested":[1,]}`} {
		for _, invalidBody := range []bool{false, true} {
			body := map[string]json.RawMessage{"a": json.RawMessage(`1`), "z": json.RawMessage(`null`)}
			filters := map[string]json.RawMessage{"a": json.RawMessage(`2`), "z": json.RawMessage(`null`)}
			if invalidBody {
				body["z"] = json.RawMessage(raw)
			} else {
				filters["z"] = json.RawMessage(raw)
			}
			for range 20 {
				if match, err := MatchFilters(body, filters); err == nil || match {
					t.Fatalf("invalid JSON hidden: body=%v raw=%q match=%v error=%v", invalidBody, raw, match, err)
				}
			}
		}
	}
	_, err := MatchFilters(nil, map[string]json.RawMessage{"field": json.RawMessage(`{"broken":]}`)})
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) || !strings.Contains(err.Error(), `filter "field"`) {
		t.Fatalf("JSON cause/key context lost: %v", err)
	}
	_, err = MatchFilters(map[string]json.RawMessage{"field": nil}, map[string]json.RawMessage{"field": json.RawMessage(`null`)})
	if !errors.Is(err, io.EOF) || !strings.Contains(err.Error(), `body field "field"`) {
		t.Fatalf("present empty field was treated as missing: %v", err)
	}
}

func TestMatchFiltersOnlyReadsReferencedFieldsAndNeverMutatesInputs(t *testing.T) {
	body := map[string]json.RawMessage{"field": json.RawMessage(`{"nested":{"value":1,"extra":true}}`), "unreferenced": json.RawMessage(`invalid`)}
	filters := map[string]json.RawMessage{"field": json.RawMessage(`{"nested":{"value":1.0}}`)}
	beforeBody, beforeFilter := string(body["field"]), string(filters["field"])
	for range 2 {
		match, err := MatchFilters(body, filters)
		if err != nil || !match || string(body["field"]) != beforeBody || string(filters["field"]) != beforeFilter || string(body["unreferenced"]) != "invalid" {
			t.Fatalf("pure comparison changed inputs: match=%v error=%v", match, err)
		}
	}
}
