package policies_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonPolicyRecordsBodyFiltersStayLocal(t *testing.T) {
	recorder := new(pythonPolicyRecorder)
	const reply = `{"policies":[{"id":"one","project_id":"p","user_id":"u","blob":{"role":"reader","extra":true},"type":"json","extension":9007199254740993},{"id":"two","project_id":"other","blob":{"role":"reader"}}]}`
	api, _ := pythonPolicyAPI(t, recorder, func(r *http.Request) *http.Response {
		if r.URL.RawQuery != "limit=0&marker=first" || r.Header.Get("X-Trace") != "owned" {
			t.Error(r.URL, r.Header)
		}
		return pythonPolicyWire(203, reply)
	})
	rows, err := api.AllRecords(context.Background(), policies.WithRecordListLimit(0), policies.WithRecordListMarker("first"), policies.WithRecordListHeader("X-Trace", "owned"), policies.WithRecordListFilter("project_id", "p"), policies.WithRecordListFilter("user_id", "u"), policies.WithRecordListFilter("blob", map[string]any{"role": "reader"}), policies.WithRecordListFilter("unknown", func() {}))
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	row := rows[0]
	if string(row.Resource.Body["id"]) != `"one"` || string(row.Resource.Body["name"]) != "null" || row.Resource.Body["extension"] != nil || string(row.Wire.Body["extension"]) != "9007199254740993" || string(row.Envelope) != reply || row.StatusCode != 203 {
		t.Fatal(row)
	}
	if len(recorder.snapshot()) != 1 {
		t.Fatal(recorder.snapshot())
	}
}
func TestPythonPolicyRecordsPaginationPartialAndRawCap(t *testing.T) {
	recorder := new(pythonPolicyRecorder)
	const broken = `{"policies":[{"id":"three","project_id":"p"},null]}`
	api, _ := pythonPolicyAPI(t, recorder, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "two" {
			return pythonPolicyWire(202, broken)
		}
		return pythonPolicyWire(200, `{"policies":[{"id":"one","project_id":"skip"},{"id":"two","project_id":"p"}],"links":[{"rel":"next","href":"/v3/policies?marker=two"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), policies.WithRecordListFilter("project_id", "p"))
	var proof *resource.ResponseError
	if len(rows) != 2 || !errors.As(err, &proof) || string(proof.Body) != broken || proof.StatusCode != 202 || len(recorder.snapshot()) != 2 {
		t.Fatal(rows, err, proof, recorder.snapshot())
	}
	rows, err = api.AllRecords(context.Background(), policies.WithRecordListMaxItems(2), policies.WithRecordListFilter("project_id", "p"))
	if err != nil || len(rows) != 1 || len(recorder.snapshot()) != 3 || !strings.Contains(recorder.snapshot()[2].query, "limit=2") {
		t.Fatal(rows, err, recorder.snapshot())
	}
}
func TestPythonPolicyRecordsOwnReusableOptionsAndRemainLazy(t *testing.T) {
	recorder := new(pythonPolicyRecorder)
	api, _ := pythonPolicyAPI(t, recorder, func(*http.Request) *http.Response {
		return pythonPolicyWire(200, `{"policies":[{"id":"one","project_id":"p"}]}`)
	})
	raw := []byte(`"p"`)
	limit := 0
	option := policies.WithRecordListOpts(policies.RecordListOpts{Limit: &limit, Filters: map[string]json.RawMessage{"project_id": raw}})
	raw[1] = 'x'
	limit = 5
	seq := api.ListRecords(context.Background(), option)
	if len(recorder.snapshot()) != 0 {
		t.Fatal("eager HTTP")
	}
	for i := 0; i < 2; i++ {
		count := 0
		for row, err := range seq {
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			count++
		}
		if count != 1 {
			t.Fatal(count)
		}
	}
	for _, call := range recorder.snapshot() {
		if call.query != "limit=0" {
			t.Fatal(call)
		}
	}
}
func TestPythonPolicyRecordsGuardAndCancellation(t *testing.T) {
	recorder := new(pythonPolicyRecorder)
	api, _ := pythonPolicyAPI(t, recorder, func(*http.Request) *http.Response { return pythonPolicyWire(200, `{"policies":[]}`) })
	bad := func(*policies.RecordListOpts) error {
		client := *api.RawClient()
		*api = *policies.New(&client)
		return nil
	}
	if _, err := api.AllRecords(context.Background(), bad); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.AllRecords(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.AllRecords(context.Background(), policies.WithRecordListLimit(-1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop")
	cancel(cause)
	if _, err := api.AllRecords(ctx); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if len(recorder.snapshot()) != 0 {
		t.Fatal(recorder.snapshot())
	}
}
func TestPythonPolicyRecordsFiltersAllDeclaredFieldsAndEarlyBreak(t *testing.T) {
	recorder := new(pythonPolicyRecorder)
	api, _ := pythonPolicyAPI(t, recorder, func(*http.Request) *http.Response {
		return pythonPolicyWire(200, `{"policies":[{"id":"one","name":false,"blob":null,"type":"json","project_id":"p","user_id":3,"links":{"self":"local","other":true}},null]}`)
	})
	count := 0
	for row, err := range api.ListRecords(context.Background(), policies.WithRecordListFilter("id", "one"), policies.WithRecordListFilter("name", false), policies.WithRecordListFilter("blob", nil), policies.WithRecordListFilter("type", "json"), policies.WithRecordListFilter("project_id", "p"), policies.WithRecordListFilter("user_id", 3), policies.WithRecordListFilter("links", map[string]any{"self": "local"})) {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
		count++
		break
	}
	if count != 1 || recorder.snapshot()[0].query != "" {
		t.Fatal(count, recorder.snapshot())
	}
}
