package request_test

import (
	"encoding/json"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/request"
)

func TestOptionalWirePresenceAndTypedRoundTrip(t *testing.T) {
	type update struct {
		Name  request.Optional[string] `json:"name,omitzero"`
		Count request.Optional[int64]  `json:"count,omitzero"`
		Force request.Optional[bool]   `json:"force,omitzero"`
	}
	for _, test := range []struct {
		name  string
		input update
		wire  string
	}{
		{"omitted", update{}, `{}`},
		{"null", update{Name: request.Null[string](), Count: request.Null[int64](), Force: request.Null[bool]()}, `{"name":null,"count":null,"force":null}`},
		{"zero", update{Name: request.Present(""), Count: request.Present[int64](0), Force: request.Present(false)}, `{"name":"","count":0,"force":false}`},
		{"exact_integer", update{Count: request.Present[int64](9007199254740993)}, `{"count":9007199254740993}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			wire, err := json.Marshal(test.input)
			if err != nil || string(wire) != test.wire {
				t.Fatalf("presence lost: wire=%s err=%v", wire, err)
			}
			var decoded update
			if err := json.Unmarshal(wire, &decoded); err != nil || decoded != test.input {
				t.Fatalf("typed round trip changed: value=%+v want=%+v err=%v", decoded, test.input, err)
			}
			if decoded.Name.IsSet() != test.input.Name.IsSet() || decoded.Name.IsNull() != test.input.Name.IsNull() {
				t.Fatal("presence flags changed")
			}
			value, present := decoded.Count.Get()
			wanted, wantedPresent := test.input.Count.Get()
			if value != wanted || present != wantedPresent {
				t.Fatal("exact count changed", value, present)
			}
		})
	}
}

func TestOptionalDecodeFailureRetainsPreviousValue(t *testing.T) {
	value := request.Present[int64](9007199254740993)
	for _, wire := range []string{`"0"`, `1.5`, `9223372036854775808`, `{}`, `[]`, `true`} {
		if err := json.Unmarshal([]byte(wire), &value); err == nil {
			t.Fatalf("wrong type accepted: %s", wire)
		}
		if number, present := value.Get(); !present || number != 9007199254740993 || value.IsNull() {
			t.Fatalf("failed decode changed value: %s %+v", wire, value)
		}
	}
	if err := json.Unmarshal([]byte(" \n null \t"), &value); err != nil || !value.IsSet() || !value.IsNull() {
		t.Fatal(value, err)
	}
	if _, present := value.Get(); present {
		t.Fatal("null was exposed as a supplied zero")
	}
}
