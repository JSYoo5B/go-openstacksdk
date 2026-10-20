package jsonfilter

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestEqualJSONUsesWholeValuesAndExactNumbers(t *testing.T) {
	for _, test := range []struct {
		name, left, right string
		equal             bool
	}{
		{"object key order", `{"big":9007199254740993,"fraction":1.25}`, `{"fraction":125e-2,"big":9007199254740993}`, true},
		{"large adjacent integers", `9007199254740993`, `9007199254740992`, false},
		{"large exponent", `1e1000000`, `10e999999`, true},
		{"negative zero", `-0e1000000000`, `0.00`, true},
		{"boolean is not number", `true`, `1`, false},
		{"false is not zero", `false`, `0`, false},
		{"missing member is not null", `{"x":null}`, `{}`, false},
		{"object equality is not subset", `{"x":1,"y":2}`, `{"x":1}`, false},
		{"empty object equality", `{}`, `{}`, true},
		{"array equality", `[1,{"x":2}]`, `[1.0,{"x":2e0}]`, true},
		{"array order", `[1,2]`, `[2,1]`, false},
		{"null equality", `null`, `null`, true},
		{"null differs from array", `null`, `[]`, false},
		{"unicode escape", `"\u0061"`, `"a"`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := EqualJSON(json.RawMessage(test.left), json.RawMessage(test.right))
			if err != nil || got != test.equal {
				t.Fatalf("equal=%v error=%v, want %v", got, err, test.equal)
			}
		})
	}
}

func TestEqualJSONRejectsMalformedSidesAndRetainsParseCauses(t *testing.T) {
	for _, side := range []string{"left", "right"} {
		for _, raw := range []string{"", "{", "null null", "01", "1e", "NaN", `{"x":[1,]}`} {
			t.Run(side+"/"+raw, func(t *testing.T) {
				left, right := json.RawMessage(`null`), json.RawMessage(`null`)
				if side == "left" {
					left = json.RawMessage(raw)
				} else {
					right = json.RawMessage(raw)
				}
				equal, err := EqualJSON(left, right)
				if equal || err == nil || !strings.HasPrefix(err.Error(), side+" JSON: ") {
					t.Fatal(equal, err)
				}
				if raw == "" && !errors.Is(err, io.EOF) {
					t.Fatal("empty input lost EOF cause", err)
				}
				if raw == "{" {
					if !errors.Is(err, io.ErrUnexpectedEOF) {
						t.Fatal("incomplete input lost parse cause", err)
					}
				}
				if raw == `{"x":[1,]}` {
					var syntax *json.SyntaxError
					if !errors.As(err, &syntax) {
						t.Fatal("invalid nested input lost SyntaxError", err)
					}
				}
			})
		}
	}
}

func TestJSONComparisonCanReuseReadOnlyInputsConcurrently(t *testing.T) {
	body := map[string]json.RawMessage{
		"rules": json.RawMessage(`[{"value":9007199254740993,"enabled":false}]`),
		"attrs": json.RawMessage(`{"nested":{"count":1e1000000,"extra":true}}`),
	}
	filters := map[string]json.RawMessage{
		"rules": json.RawMessage(`[{"enabled":false,"value":90071992547409930e-1}]`),
		"attrs": json.RawMessage(`{"nested":{"count":10e999999}}`),
	}
	beforeBody, beforeFilters := string(body["rules"]), string(filters["rules"])
	var group sync.WaitGroup
	for range 12 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 20 {
				match, err := MatchFilters(body, filters)
				if err != nil || !match {
					t.Error(match, err)
					return
				}
				equal, err := EqualJSON(body["rules"], filters["rules"])
				if err != nil || !equal {
					t.Error(equal, err)
					return
				}
			}
		}()
	}
	group.Wait()
	if string(body["rules"]) != beforeBody || string(filters["rules"]) != beforeFilters {
		t.Fatal("comparison changed shared inputs")
	}
}
