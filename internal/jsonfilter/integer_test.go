package jsonfilter

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestIntegerJSONPreservesExactValueAndBoundedExponentMemory(t *testing.T) {
	for _, check := range []struct{ input, expected string }{
		{`null`, `null`}, {`-0e-999999999999999999999`, `0`},
		{`9007199254740993`, `9007199254740993`}, {`-24.000`, `-24`}, {`2.4e1`, `24`},
		{`" +000024 "`, `24`}, {`"-09007199254740993"`, `-9007199254740993`},
		{`1e999999999999999999999999999999`, `1e999999999999999999999999999999`},
	} {
		t.Run(check.input, func(t *testing.T) {
			got, err := IntegerJSON(json.RawMessage(check.input))
			if err != nil {
				t.Fatal(err)
			}
			matched, err := EqualJSON(got, json.RawMessage(check.expected))
			if err != nil || !matched {
				t.Fatalf("got=%s expected=%s err=%v", got, check.expected, err)
			}
			if len(got) > len(check.input)+2 {
				t.Fatalf("normalization expanded input: %s", got)
			}
		})
	}
}

func TestIntegerJSONRejectsNonintegralAndOtherTypes(t *testing.T) {
	for _, input := range []string{`1.01`, `1e-999999999999999999999`, `true`, `false`, `[]`, `{}`, `""`, `"1.0"`, `"1e2"`, `"1_000"`, `"0x10"`, `"１２"`, `1 2`, `{]`} {
		t.Run(input, func(t *testing.T) {
			if got, err := IntegerJSON(json.RawMessage(input)); err == nil || got != nil {
				t.Fatalf("got=%s err=%v", got, err)
			}
		})
	}
	_, err := IntegerJSON(json.RawMessage(`{]`))
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatalf("lost JSON cause: %v", err)
	}
}
