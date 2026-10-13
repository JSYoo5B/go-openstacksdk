package resourcefilters_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonCinderResourceFiltersProxyExpressionsObserveOpaqueGeneratorBeforeHTTP(t *testing.T) {
	for _, check := range []struct{ expression, want string }{{"[*].resource", "null"}, {"[0]", "null"}, {"type(@)", "null"}, {"`true`", "true"}, {"@ == @", "true"}, {"[ @ ] == [ @ ]", "true"}, {"{same: @ == @}", `{"same":true}`}} {
		api, calls := filterFixture(t, "", func(http.ResponseWriter, *http.Request) {
			t.Fatal("opaque expression performed discovery or list HTTP")
		})
		result, err := api.ProxyListRecords(context.Background(), resourcefilters.WithRecordListJMESPath(check.expression))
		if err != nil || result == nil || string(result.JSON) != check.want || len(*calls) != 0 {
			t.Fatal(check, result, err, *calls)
		}
	}
	for _, expression := range []string{"length(@)", "contains(@,'x')", "sort_by(@,&resource)", "to_string(@)", "["} {
		api, calls := filterFixture(t, "", func(http.ResponseWriter, *http.Request) { t.Fatal("bad expression consumed generator") })
		result, err := api.ProxyListRecords(context.Background(), resourcefilters.WithRecordListJMESPath(expression))
		var op *resource.OperationError
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &op) || op.Operation != "ProxyListRecords" || len(*calls) != 0 {
			t.Fatal(expression, result, err, op, *calls)
		}
	}
}
func TestPythonCinderResourceFiltersProxyPreservesFalseyNonStringAndNestedGeneratorIdentity(t *testing.T) {
	for _, filter := range []any{nil, false, 0, json.RawMessage(`1e-400`), []any{}, map[string]any{}, true, 1, []any{"x"}, map[string]any{"x": true}, "", "@", "not_null(@,`[]`)", "to_array(@)", "{lazy:@}"} {
		api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RawQuery != "resource=volume" {
				t.Fatal(r.URL)
			}
			filterJSON(w, 200, `{"resource_filters":[{"resource":"volume"}]}`)
		})
		result, err := api.ProxyListRecords(context.Background(), resourcefilters.WithRecordListResource("volume"), resourcefilters.WithRecordListProxyJMESValue(filter))
		if err != nil || result == nil || result.JSON != nil || len(*calls) != 0 {
			t.Fatal(filter, result, err, *calls)
		}
		value := result.Value
		if array, ok := value.([]any); ok {
			if len(array) != 1 {
				t.Fatal(array)
			}
			value = array[0]
		}
		if object, ok := value.(map[string]any); ok {
			value = object["lazy"]
		}
		generator, ok := value.(*resourcefilters.RecordListGenerator)
		if !ok || generator == nil {
			t.Fatal(filter, value)
		}
		if _, err := json.Marshal(result.Value); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("opaque generator fabricated JSON", err)
		}
		n := 0
		for row, err := range generator.Records {
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			n++
		}
		if n != 1 || len(*calls) != 1 {
			t.Fatal(filter, n, *calls)
		}
	}
}
func TestPythonCinderResourceFiltersProxyRawQueryJMESAndLazyDiscoveryKeepCapturedSource(t *testing.T) {
	api, calls := filterFixture(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cinder/v3/" {
			filterJSON(w, 200, `{"version":{"id":"v3.0","min_version":"3.0","version":"3.71"}}`)
		} else {
			filterJSON(w, 200, `{"resource_filters":[{"resource":"volume"}]}`)
		}
	})
	result, err := api.ProxyListRecords(context.Background(), resourcefilters.WithRecordListQuery(map[string]any{"jmespath_filters": "@", "resource": "volume"}))
	generator, ok := result.Value.(*resourcefilters.RecordListGenerator)
	if err != nil || !ok || len(*calls) != 0 {
		t.Fatal(result, err, *calls)
	}
	n := 0
	for row, err := range generator.Records {
		if err != nil || row.Microversion != "3.33" || len(row.Discovery) != 1 {
			t.Fatal(row, err)
		}
		n++
	}
	if n != 1 || len(*calls) != 2 {
		t.Fatal(n, *calls)
	}
	api, calls = filterFixture(t, "3.33", func(http.ResponseWriter, *http.Request) { t.Fatal("changed captured source made HTTP") })
	result, err = api.ProxyListRecords(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	generator = result.Value.(*resourcefilters.RecordListGenerator)
	api.RawClient().Microversion = "3.90"
	seen := false
	for _, err := range generator.Records {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		seen = true
	}
	if !seen || len(*calls) != 0 {
		t.Fatal(seen, *calls)
	}
}
func TestPythonCinderResourceFiltersLoadedLocalFloatsRetainIntegerPrecisionAndNestedSubsets(t *testing.T) {
	for _, check := range []struct {
		row, filter string
		count       int
	}{
		{`{"id":1e-400}`, `{"id":0}`, 1},
		{`{"id":1e-400}`, `{"id":{}}`, 0},
		{`{"id":9007199254740993.0}`, `{"id":9007199254740993}`, 0},
		{`{"id":9007199254740993.0}`, `{"id":9007199254740992}`, 1},
		{`{"id":9007199254740993}`, `{"id":9007199254740992}`, 0},
		{`{"id":1e400}`, `{"id":10e399}`, 1},
		{`{"filters":[1e-400,{"n":0.10000000000000001}]}`, `{"filters":[false,{"n":0.1}]}`, 1},
		{`{"id":{"nested":{"n":1e-400,"extra":true}}}`, `{"id":{"nested":{"n":0}}}`, 1},
	} {
		api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, _ *http.Request) {
			filterJSON(w, 200, `{"resource_filters":[`+check.row+`]}`)
		})
		var query map[string]any
		var raw map[string]json.RawMessage
		if json.Unmarshal([]byte(check.filter), &raw) != nil {
			t.Fatal(check)
		}
		query = make(map[string]any, len(raw))
		for k, v := range raw {
			query[k] = v
		}
		rows, err := api.AllRecords(context.Background(), resourcefilters.WithRecordListQuery(query))
		if err != nil || len(rows) != check.count || len(*calls) != 1 {
			t.Fatal(check, rows, err, *calls)
		}
	}
}
