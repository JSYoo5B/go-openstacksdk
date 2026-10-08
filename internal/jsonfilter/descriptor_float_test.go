package jsonfilter

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestDescriptorFloatJSONSourceConversionsAndOwnership(t *testing.T) {
	for _, check := range []struct {
		name, input, expected string
		negativeZero          bool
	}{
		{"missing", "", "null", false},
		{"null", `null`, `null`, false},
		{"padded null", " \nnull\t", `null`, false},
		{"true", `true`, `1`, false},
		{"false", `false`, `0`, false},
		{"integer", `2`, `2`, false},
		{"negative integer", `-2`, `-2`, false},
		{"fraction", `1.25`, `1.25`, false},
		{"exponent", `1e2`, `100`, false},
		{"integer IEEE rounding", `9007199254740993`, `9007199254740992`, false},
		{"empty list", `[]`, `0`, false},
		{"nonempty list", `[1]`, `0`, false},
		{"empty object", `{}`, `0`, false},
		{"nonempty object", `{"x":1}`, `0`, false},
		{"padded signed string", `" +1.25 "`, `1.25`, false},
		{"leading dot", `".5"`, `0.5`, false},
		{"trailing dot", `"1."`, `1`, false},
		{"trailing dot exponent", `"1.e2"`, `100`, false},
		{"digit separators", `"1_2.3_4e+1"`, `123.4`, false},
		{"string leading zero", `"001.25"`, `1.25`, false},
		{"Unicode decimal and whitespace", `"\u00a0１２.٣\u2003"`, `12.3`, false},
		{"Unicode exponent and separators", `"١_٢e٣"`, `12000`, false},
		{"plain integer negative zero", `-0`, `0`, false},
		{"fraction negative zero", `-0.0`, `0`, true},
		{"exponent negative zero", `-0e1`, `0`, true},
		{"string negative zero", `"-0"`, `0`, true},
		{"positive underflow", `1e-400`, `0`, false},
		{"negative underflow", `-1e-400`, `0`, true},
		{"negative string underflow", `"-1e-400"`, `0`, true},
	} {
		t.Run(check.name, func(t *testing.T) {
			got, err := DescriptorFloatJSON(json.RawMessage(check.input))
			if err != nil {
				t.Fatal(err)
			}
			matched, err := EqualJSON(got, json.RawMessage(check.expected))
			if err != nil || !matched {
				t.Fatalf("got=%s expected=%s err=%v", got, check.expected, err)
			}
			if check.expected == "0" {
				floating, err := strconv.ParseFloat(string(got), 64)
				if err != nil || math.Signbit(floating) != check.negativeZero {
					t.Fatalf("got=%s negativeZero=%v err=%v", got, check.negativeZero, err)
				}
			}
		})
	}
	input := json.RawMessage(`1.25`)
	got, err := DescriptorFloatJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	got[0] = 'x'
	if string(input) != "1.25" {
		t.Fatal("normalization aliases input")
	}
	again, err := DescriptorFloatJSON(input)
	if err != nil || string(again) != "1.25" {
		t.Fatal("results share backing bytes", string(again), err)
	}
	matched, err := MatchFilters(map[string]json.RawMessage{"f": again}, map[string]json.RawMessage{"f": json.RawMessage(`"1.25"`)})
	if err != nil || matched {
		t.Fatal("caller string filter was coerced", matched, err)
	}
}

func TestDescriptorFloatJSONRejectsInvalidStringsAndNonfiniteWithCauses(t *testing.T) {
	for _, input := range []string{
		`""`, `" \t\n "`, `"abc"`, `"true"`, `"²"`, `"1 2"`, `"0x1p2"`,
		`"1__0"`, `"_1"`, `"1_"`, `"1._0"`, `"1e_2"`, `"−1"`, `"１．５"`,
		`"\u200b1"`, `"\u001c1"`, `"1\u0000"`, `"NaN"`, `"+nan"`, `"-nAn"`,
		`"inf"`, `"+Infinity"`, `"-INFINITY"`, `1e309`, `"1e309"`,
		`NaN`, `Infinity`, `{]`, `1 2`, `null {]`, " \t\n ",
	} {
		t.Run(input, func(t *testing.T) {
			got, err := DescriptorFloatJSON(json.RawMessage(input))
			if err == nil || got != nil {
				t.Fatalf("got=%s err=%v", got, err)
			}
		})
	}
	if got, err := DescriptorFloatJSON(json.RawMessage{'"', 0xff, '"'}); err == nil || got != nil {
		t.Fatalf("invalid UTF-8 got=%s err=%v", got, err)
	}
	_, err := DescriptorFloatJSON(json.RawMessage(`null {]`))
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatal("lost trailing JSON cause", err)
	}
	_, err = DescriptorFloatJSON(json.RawMessage(" \t\n "))
	if !errors.Is(err, io.EOF) {
		t.Fatal("lost whitespace-only JSON cause", err)
	}
	for _, input := range []string{`"1__0"`, `"1e"`, `1e309`} {
		_, err := DescriptorFloatJSON(json.RawMessage(input))
		var number *strconv.NumError
		if !errors.As(err, &number) {
			t.Fatal("lost numeric conversion cause", input, err)
		}
		want := strconv.ErrSyntax
		if input == `1e309` {
			want = strconv.ErrRange
		}
		if !errors.Is(err, want) {
			t.Fatal("wrong numeric conversion cause", input, err)
		}
	}
	// Parsing is bounded by input size, rather than expanding the exponent.
	for _, check := range []struct {
		prefix   string
		succeeds bool
	}{
		{"1e", false}, {"1e-", true}, {"-1e-", true},
	} {
		input := json.RawMessage(check.prefix + strings.Repeat("9", 10000))
		got, err := DescriptorFloatJSON(input)
		if check.succeeds {
			if err != nil || len(got) > 2 || !json.Valid(got) {
				t.Fatalf("underflow got=%s err=%v", got, err)
			}
			floating, err := strconv.ParseFloat(string(got), 64)
			if err != nil || floating != 0 || math.Signbit(floating) != strings.HasPrefix(check.prefix, "-") {
				t.Fatalf("underflow sign got=%s err=%v", got, err)
			}
		} else if err == nil || got != nil {
			t.Fatalf("overflow got=%s err=%v", got, err)
		}
	}
}
