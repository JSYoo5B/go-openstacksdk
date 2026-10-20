package troveroot

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

func TestExtractPreservesLiteralTrueMapSemantics(t *testing.T) {
	type namedBool bool
	for _, tc := range []struct {
		name string
		body map[string]any
		want bool
	}{
		{"true", map[string]any{"rootEnabled": true}, true},
		{"false", map[string]any{"rootEnabled": false}, false},
		{"typed nil map", nil, false},
		{"absent", map[string]any{}, false},
		{"wrong case", map[string]any{"RootEnabled": true}, false},
		{"null field", map[string]any{"rootEnabled": nil}, false},
		{"string field", map[string]any{"rootEnabled": "true"}, false},
		{"native number field", map[string]any{"rootEnabled": json.Number("1")}, false},
		{"array field", map[string]any{"rootEnabled": []any{true}}, false},
		{"object field", map[string]any{"rootEnabled": map[string]any{"enabled": true}}, false},
		{"named bool field", map[string]any{"rootEnabled": namedBool(true)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := make(map[string]any, len(tc.body))
			for key, value := range tc.body {
				before[key] = value
			}
			if tc.body == nil {
				before = nil
			}
			got, err := Extract(gophercloud.Result{Body: tc.body})
			if got != tc.want || err != nil || !reflect.DeepEqual(tc.body, before) {
				t.Fatalf("enabled=%v want=%v error=%v body=%#v", got, tc.want, err, tc.body)
			}
		})
	}
}

func TestExtractRejectsSuccessfulNonObjectEnvelopeWithoutPanic(t *testing.T) {
	type namedMap map[string]any
	for _, tc := range []struct {
		name string
		body any
	}{
		{"nil", nil},
		{"array", []any{}},
		{"typed nil array", []any(nil)},
		{"string", "root"},
		{"number", json.Number("1")},
		{"bool", true},
		{"other map", map[string]string{"rootEnabled": "true"}},
		{"named map", namedMap{"rootEnabled": true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Extract(gophercloud.Result{Body: tc.body})
			var envelope *EnvelopeError
			if got || !errors.As(err, &envelope) || envelope.Actual == "" || !strings.Contains(err.Error(), "JSON object") {
				t.Fatalf("enabled=%v error=%v", got, err)
			}
		})
	}
}

func TestExtractOriginalErrorWinsBeforeReadingBody(t *testing.T) {
	cause := &gophercloud.ErrUnexpectedResponseCode{Actual: 403, Body: []byte("native failure")}
	for _, body := range []any{nil, []any{}, "root", map[string]any(nil), map[string]any{"rootEnabled": true}} {
		result := gophercloud.Result{Body: body, Err: cause}
		got, err := Extract(result)
		if got || err != cause || !errors.Is(err, cause) {
			t.Fatalf("body=%#v enabled=%v error=%v want original cause %v", body, got, err, cause)
		}
		var envelope *EnvelopeError
		if errors.As(err, &envelope) {
			t.Fatalf("original error was replaced by an envelope error: %v", err)
		}
	}
}
