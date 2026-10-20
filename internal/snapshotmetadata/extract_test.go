package snapshotmetadata

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

func TestExtractReturnsCanonicalObjectAndExactNativeNumbers(t *testing.T) {
	var body any
	decoder := json.NewDecoder(strings.NewReader(`{"Metadata":{"decoy":true},"metadata":{"large":9007199254740993123456789,"exponent":1.234567890123456789e200,"nested":{"null":null,"array":[true,{"number":12345678901234567890}]}}}`))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		t.Fatal(err)
	}
	got, err := Extract(gophercloud.Result{Body: body})
	if err != nil || got["large"] != json.Number("9007199254740993123456789") || got["exponent"] != json.Number("1.234567890123456789e200") {
		t.Fatalf("metadata=%v err=%v", got, err)
	}
	nested := got["nested"].(map[string]any)
	if nested["null"] != nil || nested["array"].([]any)[1].(map[string]any)["number"] != json.Number("12345678901234567890") {
		t.Fatalf("nested=%v", nested)
	}
	// This is the decoded response object, not a re-marshaled copy or request seed.
	got["owned"] = "response"
	if body.(map[string]any)["metadata"].(map[string]any)["owned"] != "response" {
		t.Fatal("returned object was decoded again")
	}
	for _, empty := range []map[string]any{{}, {"null": nil}} {
		got, err := Extract(gophercloud.Result{Body: map[string]any{"metadata": empty}})
		if err != nil || got == nil || len(got) != len(empty) {
			t.Fatalf("empty/null-valued metadata=%v err=%v", got, err)
		}
	}
}

func TestExtractRejectsMalformedEnvelopeWithoutPanicking(t *testing.T) {
	for _, tc := range []struct {
		name  string
		body  any
		field string
	}{
		{"nil root", nil, "response"}, {"typed nil root", map[string]any(nil), "response"},
		{"array root", []any{}, "response"}, {"string root", "metadata", "response"},
		{"wrong map root", map[string]string{"metadata": "value"}, "response"},
		{"absent", map[string]any{}, "metadata"}, {"case alias", map[string]any{"Metadata": map[string]any{}}, "metadata"},
		{"null", map[string]any{"metadata": nil}, "metadata"},
		{"typed nil object", map[string]any{"metadata": map[string]any(nil)}, "metadata"},
		{"array", map[string]any{"metadata": []any{}}, "metadata"},
		{"string", map[string]any{"metadata": "value"}, "metadata"},
		{"number", map[string]any{"metadata": json.Number("1")}, "metadata"},
		{"wrong map", map[string]any{"metadata": map[string]string{}}, "metadata"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Extract(gophercloud.Result{Body: tc.body})
			var shape *EnvelopeError
			if got != nil || !errors.As(err, &shape) || shape.Field != tc.field || shape.Actual == "" || !strings.Contains(err.Error(), "non-null JSON object") {
				t.Fatalf("value=%v error=%v", got, err)
			}
		})
	}
}

func TestExtractPreservesOriginalErrorAndConcurrentResponseOwnership(t *testing.T) {
	cause := &gophercloud.ErrUnexpectedResponseCode{Actual: 403, Body: []byte("native error")}
	for _, body := range []any{nil, []any{}, map[string]any{"metadata": map[string]any{"ignored": true}}} {
		value, err := Extract(gophercloud.Result{Body: body, Err: cause})
		if value != nil || err != cause {
			t.Fatalf("value=%v error=%v (want original cause)", value, err)
		}
	}
	var wait sync.WaitGroup
	for i := 0; i < 16; i++ {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			own := map[string]any{"iteration": i}
			value, err := Extract(gophercloud.Result{Body: map[string]any{"metadata": own}})
			if err != nil || value["iteration"] != i {
				t.Errorf("value=%v err=%v", value, err)
			}
			value["changed"] = i
			if own["changed"] != i {
				t.Error("response map ownership changed")
			}
		}(i)
	}
	wait.Wait()
}
