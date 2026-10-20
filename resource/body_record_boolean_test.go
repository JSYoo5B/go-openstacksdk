package resource

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestBodyRecordBooleanPresenceOwnershipAndErrorCauses(t *testing.T) {
	fields := map[string]json.RawMessage{"nil": json.RawMessage(`null`), "boolean": json.RawMessage(`false`), "coerced": json.RawMessage(`"false"`), "malformed": json.RawMessage(`[] {]`)}
	for _, key := range []string{"missing", "nil"} {
		value, err := BodyRecordField(fields, key, BodyFieldBoolean)
		if err != nil || string(value) != "null" {
			t.Fatal("missing/null acquired false default", key, string(value), err)
		}
	}
	value, err := BodyRecordField(fields, "coerced", BodyFieldBoolean)
	if err != nil || string(value) != "true" {
		t.Fatal(string(value), err)
	}
	value[0] = 'x'
	if string(fields["coerced"]) != `"false"` {
		t.Fatal("response field bytes changed")
	}
	again, err := BodyRecordField(fields, "coerced", BodyFieldBoolean)
	if err != nil || string(again) != "true" {
		t.Fatal(string(again), err)
	}
	_, err = BodyRecordField(fields, "malformed", BodyFieldBoolean)
	var syntax *json.SyntaxError
	if !errors.Is(err, ErrInvalidOption) || !errors.As(err, &syntax) {
		t.Fatal("lost invalid option or underlying JSON cause", err)
	}
	if _, err = BodyRecordField(fields, "nil", BodyFieldType(255)); !errors.Is(err, ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestBodyRecordBooleanKeepsJSONAndIntegerDescriptorContracts(t *testing.T) {
	if BodyFieldJSON != 0 || BodyFieldInteger != 1 || BodyFieldBoolean != 2 {
		t.Fatal("existing field enum changed")
	}
	fields := map[string]json.RawMessage{"number": json.RawMessage(`" +000024 "`), "bool": json.RawMessage(`true`), "decimal": json.RawMessage(`0.000`)}
	for _, check := range []struct {
		key  string
		kind BodyFieldType
		want string
	}{
		{"number", BodyFieldJSON, `" +000024 "`}, {"number", BodyFieldInteger, `24`},
		{"number", BodyFieldBoolean, `true`}, {"decimal", BodyFieldJSON, `0.000`},
		{"decimal", BodyFieldInteger, `0`}, {"decimal", BodyFieldBoolean, `false`},
	} {
		got, err := BodyRecordField(fields, check.key, check.kind)
		if err != nil || string(got) != check.want {
			t.Fatal(check, string(got), err)
		}
	}
	if _, err := BodyRecordField(fields, "bool", BodyFieldInteger); !errors.Is(err, ErrInvalidOption) {
		t.Fatal("boolean was accepted by integer descriptor", err)
	}
}
