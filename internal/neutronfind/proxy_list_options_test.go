package neutronfind_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestNormalizeProxyListOptionsPreservesOriginalExpressionAndOrderedOnePassMerge(t *testing.T) {
	version, base := "2.0", "/old"
	input := neutronfind.Options{
		Query: map[string]json.RawMessage{
			"name": json.RawMessage(`"old"`), "status": json.RawMessage(`"ACTIVE"`), "__conflicting_attrs": json.RawMessage(`{"name":"new","id":["one"],"microversion":"2.1","headers":{"X-Trace":"new"},"jmespath_filters":"[?","__conflicting_attrs":{"status":"no recursion"}}`), "microversion": json.RawMessage(`"2.0"`), "headers": json.RawMessage(`{"X-Trace":"old"}`), "base_path": json.RawMessage(`"/custom-networks"`), "jmespath_filters": json.RawMessage(`"@"`),
		},
		Order: []string{"name", "status", "__conflicting_attrs", "microversion", "headers", "base_path", "jmespath_filters"}, Headers: map[string]string{"X-Old": "old"}, Microversion: &version, BasePath: &base,
	}
	owned, expression, err := neutronfind.NormalizeProxyListOptions(input)
	wantQuery := map[string]json.RawMessage{"name": json.RawMessage(`"new"`), "status": json.RawMessage(`"ACTIVE"`), "id": json.RawMessage(`["one"]`)}
	if err != nil || string(expression) != `"@"` || !reflect.DeepEqual(owned.Query, wantQuery) || !reflect.DeepEqual(owned.Order, []string{"name", "status", "id"}) || owned.Microversion == nil || *owned.Microversion != "2.1" || owned.BasePath == nil || *owned.BasePath != "/custom-networks" || !reflect.DeepEqual(owned.Headers, map[string]string{"X-Trace": "new"}) {
		t.Fatal(owned, string(expression), err)
	}
	expression[1] = 'X'
	owned.Query["status"][1] = 'X'
	owned.Headers["X-Trace"] = "changed"
	*owned.Microversion = "changed"
	*owned.BasePath = "changed"
	if string(input.Query["jmespath_filters"]) != `"@"` || string(input.Query["status"]) != `"ACTIVE"` || input.Headers["X-Old"] != "old" || version != "2.0" || base != "/old" {
		t.Fatal(input, version, base)
	}
}

func TestNormalizeProxyListOptionsFalseyConflictAndUnusedFormalIgnoreValues(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `1e-400`, `""`, `[]`, `{}`} {
		input := neutronfind.Options{Query: map[string]json.RawMessage{"name": json.RawMessage(`"n"`), "__conflicting_attrs": json.RawMessage(raw), "allow_unknown_params": json.RawMessage(`invalid`), "connection": json.RawMessage(`invalid`), "_synchronized": json.RawMessage(`invalid`), "location": json.RawMessage(`invalid`), "list_base_path": json.RawMessage(`invalid`), "requires_id": json.RawMessage(`invalid`), "error_message": json.RawMessage(`invalid`), "skip_cache": json.RawMessage(`invalid`), "resource_response_key": json.RawMessage(`invalid`), "unknown": json.RawMessage(`invalid`)}}
		value, expression, err := neutronfind.NormalizeProxyListOptions(input)
		if err != nil || expression != nil || len(value.Query) != 2 || string(value.Query["name"]) != `"n"` || string(value.Query["unknown"]) != `invalid` || !reflect.DeepEqual(value.Order, []string{"name", "unknown"}) {
			t.Fatal(raw, value, expression, err)
		}
		if input.Query["connection"] == nil || input.Query["__conflicting_attrs"] == nil {
			t.Fatal("mutated caller", input)
		}
	}
}

func TestNormalizeProxyListOptionsRejectsOnlyBoundAndRestoredDuplicateArguments(t *testing.T) {
	for _, key := range []string{"self", "cls", "session", "resource_type"} {
		value, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{key: json.RawMessage(`invalid`)}})
		if !errors.Is(err, resource.ErrInvalidOption) || value.Query[key] == nil {
			t.Fatal(key, value, err)
		}
	}
	for _, key := range []string{"self", "cls", "session", "resource_type", "paginated", "base_path"} {
		for _, raw := range []string{`null`, `false`, `0`, `""`} {
			conflict := json.RawMessage(`{"` + key + `":` + raw + `}`)
			_, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{"__conflicting_attrs": conflict}})
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(key, raw, err)
			}
		}
	}
	for _, raw := range []string{`true`, `1`, `"truthy"`, `[1]`, `invalid`, `{} false`} {
		_, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{"__conflicting_attrs": json.RawMessage(raw)}})
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(raw, err)
		}
	}
}

func TestNormalizeProxyListOptionsHeaderFalseyStringMapAndNullableVersionPath(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `1e-400`, `""`, `[]`, `{}`} {
		version, base := "2.4", "/old"
		value, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{"headers": json.RawMessage(raw), "microversion": json.RawMessage(`null`), "base_path": json.RawMessage(`null`)}, Headers: map[string]string{"X-Trace": "old"}, Microversion: &version, BasePath: &base})
		if err != nil || len(value.Query) != 0 || value.Microversion != nil || value.BasePath != nil || value.Headers == nil || len(value.Headers) != 0 || version != "2.4" || base != "/old" {
			t.Fatal(raw, value, err, version, base)
		}
	}
	value, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{"headers": json.RawMessage(`{"Accept":"custom","X-Trace":""}`), "microversion": json.RawMessage(`""`), "base_path": json.RawMessage(`""`)}})
	if err != nil || value.Microversion == nil || *value.Microversion != "" || value.BasePath == nil || *value.BasePath != "" || !reflect.DeepEqual(value.Headers, map[string]string{"Accept": "custom", "X-Trace": ""}) {
		t.Fatal(value, err)
	}
	for _, key := range []string{"headers", "microversion", "base_path"} {
		for _, raw := range []string{`true`, `1`, `[1]`, `{} false`, "\"\xff\""} {
			_, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{key: json.RawMessage(raw)}})
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(key, raw, err)
			}
		}
	}
	for _, raw := range []string{`{"X-Trace":null}`, `{"X-Trace":false}`, `{"X-Trace":1}`, `{"X-Trace":[]}`} {
		_, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{"headers": json.RawMessage(raw)}})
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(raw, err)
		}
	}
}

func TestNormalizeProxyListOptionsJMESNonstringsAreReturnedWithoutInventingQueries(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `1e-400`, `""`, `[]`, `{}`, `true`, `1`, `[1]`, `{"x":true}`, `"@"`} {
		input := neutronfind.Options{Query: map[string]json.RawMessage{"jmespath_filters": json.RawMessage(raw), "__conflicting_attrs": json.RawMessage(`{"jmespath_filters":"overridden","max_items":3,"paginated_body_extension":true}`)}}
		value, expression, err := neutronfind.NormalizeProxyListOptions(input)
		if err != nil || string(expression) != raw || value.Query["jmespath_filters"] != nil || string(value.Query["max_items"]) != "3" || string(value.Query["paginated_body_extension"]) != "true" {
			t.Fatal(raw, value, string(expression), err)
		}
	}
	// A nil optional expression represents Source None and leaves the lazy
	// generator unchanged; nonnil empty bytes are a supplied invalid document.
	value, expression, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{"jmespath_filters": nil}})
	if err != nil || expression != nil || len(value.Query) != 0 {
		t.Fatal(value, expression, err)
	}
	for _, raw := range []json.RawMessage{{}, json.RawMessage(`invalid`), json.RawMessage("\"\xff\"")} {
		_, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{"jmespath_filters": raw}})
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(raw, err)
		}
	}
}
