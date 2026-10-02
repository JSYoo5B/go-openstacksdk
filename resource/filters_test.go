package resource

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"math"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func semanticAdapter(rows []bodyFilterEntry, calls *atomic.Int32) Adapter[bodyFilterEntry] {
	adapter := bodyAdapter(rows, calls)
	adapter.FilterDescriptor = &FilterDescriptor{
		Query:    map[string]string{"is_enabled": "enabled", "name": "name", "fields": "fields", "number": "number", "limit": "limit", "marker": "marker"},
		Body:     map[string]string{"local_values": "values", "flag": "flag"},
		Reserved: []string{"max_items", "paginated", "headers", "microversion", "session", "base_path", "resource_type", "allow_unknown_params", "jmespath_filters"},
	}
	return adapter
}

func TestSemanticFiltersCanonicalPrecedencePresenceAndExactQueryEncoding(t *testing.T) {
	var calls atomic.Int32
	descriptor := semanticAdapter(nil, &calls).FilterDescriptor
	for _, value := range []any{nil, false, 0, "", []string{}} {
		options := listOptions{}
		if err := WithFilters(map[string]any{"is_enabled": value, "enabled": math.NaN(), "unknown": func() {}})(&options); err != nil {
			t.Fatal(err)
		}
		query, body, err := prepareFilters(descriptor, options.filters, nil)
		// This descriptor owns Body metadata even if only a query is selected.
		if !errors.Is(err, ErrInvalidOption) {
			t.Fatal("missing canonical Body metadata accepted", query, body, err)
		}
		query, body, err = prepareFilters(descriptor, options.filters, map[string]string{"values": "values", "flag": "flag"})
		if err != nil || len(query) != 1 || len(body) != 0 {
			t.Fatal(query, body, err)
		}
		if _, exists := query["enabled"]; !exists {
			t.Fatal("canonical nil/empty presence was lost", query)
		}
		if value == nil && query.Encode() != "" {
			t.Fatal("null query value leaked into URL", query)
		}
	}
	options := listOptions{}
	values := []any{json.Number("9007199254740993"), false, nil, "", json.Number("1e+1000000000")}
	if err := WithFilters(map[string]any{"fields": values, "number": json.Number("9007199254740993"), "unknown": json.RawMessage(`{]`)})(&options); err != nil {
		t.Fatal(err)
	}
	query, _, err := prepareFilters(descriptor, options.filters, map[string]string{"values": "values", "flag": "flag"})
	if err != nil || !reflect.DeepEqual(query["fields"], []string{"9007199254740993", "false", "", "1e+1000000000"}) || query.Get("number") != "9007199254740993" {
		t.Fatal("scalar array was joined, coerced or rounded", query, err)
	}
}

func TestSemanticFiltersSnapshotsConcurrentReuseAndServerOnlyName(t *testing.T) {
	var calls atomic.Int32
	adapter := semanticAdapter([]bodyFilterEntry{bodyEntry("match", "different-local-name", "ACTIVE", `[{"n":2}]`)}, &calls)
	base := adapter.Iterate
	adapter.Iterate = func(ctx context.Context, query url.Values) iter.Seq2[*bodyFilterEntry, error] {
		if query.Get("enabled") != "false" || query.Get("name") != "server-name" || !reflect.DeepEqual(query["fields"], []string{"id", "name"}) || query.Get("vendor") != "raw" || len(query) != 4 {
			t.Error("snapshot/classification changed", query)
		}
		query["fields"][0] = "mutated-iterator-query"
		return base(ctx, query)
	}
	collection := NewCollection(adapter)
	adapter.FilterDescriptor.Query["is_enabled"] = "changed"
	adapter.FilterDescriptor.Body["local_values"] = "flag"
	adapter.FilterDescriptor.Reserved[0] = "name"
	fields := []string{"id", "name"}
	local := []map[string]any{{"n": 2}}
	flag := false
	values := map[string]any{"is_enabled": &flag, "enabled": true, "fields": fields, "name": "server-name", "local_values": local, "unknown": math.NaN()}
	options := []ListOption{WithFilters(values), WithQuery("vendor", "raw"), WithBodyFilter("flag", true)}
	iterator := collection.List(context.Background(), options...)
	fields[0], local[0]["n"], values["name"], flag, options[0] = "changed", 99, "changed", true, nil
	if calls.Load() != 0 {
		t.Fatal("iterator construction made a request")
	}
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			count := 0
			for row, err := range iterator {
				count++
				if err != nil || row == nil || row.ID != "match" {
					t.Error(row, err)
				}
			}
			if count != 1 {
				t.Error("semantic name became a local predicate", count)
			}
		}()
	}
	group.Wait()
	if calls.Load() != 8 {
		t.Fatal(calls.Load())
	}
}

func TestSemanticFiltersValidateOnlyFinalSelectedValues(t *testing.T) {
	var calls atomic.Int32
	adapter := semanticAdapter([]bodyFilterEntry{bodyEntry("match", "", "", `[]`)}, &calls)
	collection := NewCollection(adapter)
	for _, options := range [][]ListOption{
		{WithFilter("is_enabled", math.NaN()), WithFilter("enabled", false)},
		{WithFilter("enabled", [][]string{{"bad"}}), WithFilter("is_enabled", "valid")},
		{WithFilter("local_values", func() {}), WithFilter("local_values", []string{})},
		{WithFilter("fields", map[string]any{"bad": true}), WithFilters(map[string]any{"fields": []string{"id"}})},
		{WithFilter("fields", math.NaN()), WithFilter("local_values", func() {}), WithFilter("headers", true), WithFilters(nil)},
		{WithFilter("max_items", 2), WithFilters(map[string]any{})},
	} {
		rows, err := collection.All(context.Background(), options...)
		if err != nil || len(rows) != 1 || rows[0].ID != "match" {
			t.Fatal("superseded/cleared semantic error survived", rows, err)
		}
	}
	// Individual alias and canonical updates normalize to the same wire target.
	parsed := listOptions{}
	for _, option := range []ListOption{WithFilter("is_enabled", false), WithFilter("enabled", true), WithFilter("is_enabled", 0)} {
		if err := option(&parsed); err != nil {
			t.Fatal(err)
		}
	}
	query, _, err := prepareFilters(adapter.FilterDescriptor, parsed.filters, adapter.BodyFilterFields)
	if err != nil || query.Get("enabled") != "0" {
		t.Fatal(query, err)
	}
}

func TestSemanticFiltersRejectSelectedErrorsAndReservedControlsBeforeIteration(t *testing.T) {
	var calls atomic.Int32
	adapter := semanticAdapter(nil, &calls)
	collection := NewCollection(adapter)
	for _, check := range []struct {
		name   string
		option ListOption
		want   error
	}{
		{"query object", WithFilter("fields", map[string]any{"id": true}), ErrInvalidOption},
		{"nested array", WithFilter("fields", [][]string{{"id"}}), ErrInvalidOption},
		{"selected NaN", WithFilters(map[string]any{"is_enabled": math.NaN(), "enabled": true}), ErrInvalidOption},
		{"invalid local JSON", WithFilter("local_values", json.RawMessage(`{]`)), ErrInvalidOption},
		{"local control", WithFilter("max_items", 1), ErrInvalidOption},
		{"pagination control", WithFilter("paginated", false), ErrInvalidOption},
		{"unsupported control", WithFilter("headers", map[string]string{}), ErrUnsupported},
	} {
		t.Run(check.name, func(t *testing.T) {
			iterator := collection.List(context.Background(), check.option)
			if calls.Load() != 0 {
				t.Fatal("preflight was not lazy")
			}
			count := 0
			for value, err := range iterator {
				count++
				if value != nil || !errors.Is(err, check.want) {
					t.Fatal(value, err)
				}
			}
			if count != 1 || calls.Load() != 0 {
				t.Fatal(count, calls.Load())
			}
		})
	}
	_, err := collection.All(context.Background(), WithFilter("fields", math.NaN()))
	var cause *json.UnsupportedValueError
	if !errors.Is(err, ErrInvalidOption) || !errors.As(err, &cause) {
		t.Fatal("selected snapshot cause lost", err)
	}
	adapter.FilterDescriptor = nil
	for _, option := range []ListOption{WithFilter("unknown", nil), WithFilters(nil)} {
		_, err := NewCollection(adapter).All(context.Background(), option)
		var operation *OperationError
		if !errors.Is(err, ErrUnsupported) || !errors.As(err, &operation) || operation.Operation != "list" {
			t.Fatal(err)
		}
	}
}

func TestSemanticFiltersNamespaceCollisionsAndIndependentClear(t *testing.T) {
	var calls atomic.Int32
	collection := NewCollection(semanticAdapter([]bodyFilterEntry{bodyEntry("match", "wanted", "", `[]`)}, &calls))
	for _, options := range [][]ListOption{
		{WithQuery("enabled", "same"), WithFilter("is_enabled", "same")},
		{WithFilter("enabled", nil), WithQuery("enabled", "")},
		{WithPageSize(3), WithFilter("limit", 3)},
		{WithName("wanted"), WithFilter("name", "wanted")},
		{WithFilter("local_values", []string{}), WithBodyFilter("values", []string{})},
		{WithBodyFilters(map[string]any{"aliases": nil}), WithFilter("local_values", nil)},
	} {
		rows, err := collection.All(context.Background(), options...)
		if rows != nil || !errors.Is(err, ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("ambiguous option namespaces reached iterator", rows, err, calls.Load())
		}
	}
	// Body and raw wire data with the same spelling are independent, while a
	// semantic clear removes neither explicit Body filters nor wire extensions.
	for _, options := range [][]ListOption{
		{WithQuery("values", "raw"), WithFilter("local_values", []string{})},
		{WithFilter("is_enabled", true), WithFilter("local_values", []int{9}), WithFilters(nil), WithQuery("enabled", "raw"), WithBodyFilter("values", []string{})},
		{WithFilter("local_values", []int{9}), WithBodyFilter("values", []string{}), WithFilters(map[string]any{})},
		{WithBodyFilter("values", []int{9}), WithBodyFilters(nil), WithFilter("local_values", []string{})},
	} {
		rows, err := collection.All(context.Background(), options...)
		if err != nil || len(rows) != 1 || rows[0].ID != "match" {
			t.Fatal(rows, err)
		}
	}
}

func TestSemanticFiltersDescriptorGateAndWireAliasBodyOverlap(t *testing.T) {
	var calls atomic.Int32
	base := semanticAdapter([]bodyFilterEntry{bodyEntry("match", "", "", `[]`)}, &calls)
	for _, descriptor := range []*FilterDescriptor{
		{Query: map[string]string{"bad ": "wire"}},
		{Query: map[string]string{"one": "same", "two": "same"}},
		{Query: map[string]string{"one": "two", "two": "other"}},
		{Body: map[string]string{"flag": "unknown"}},
		{Body: map[string]string{"one": "flag", "two": "flag"}},
		{Query: map[string]string{"flag": "flag"}, Body: map[string]string{"flag": "flag"}},
		{Query: map[string]string{"name": "name"}, Reserved: []string{"name"}},
		{Reserved: []string{"headers", "headers"}},
	} {
		adapter := base
		adapter.FilterDescriptor = descriptor
		_, err := NewCollection(adapter).All(context.Background(), WithFilters(nil))
		if !errors.Is(err, ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("invalid descriptor accepted", descriptor, err)
		}
	}
	// Python excludes canonical query names, not their wire aliases, from local
	// Body classification. This generic case deliberately reaches both lanes.
	base.FilterDescriptor = &FilterDescriptor{Query: map[string]string{"is_flag": "flag"}, Body: map[string]string{"flag": "flag"}}
	options := listOptions{}
	if err := WithFilters(map[string]any{"is_flag": "server", "flag": true})(&options); err != nil {
		t.Fatal(err)
	}
	query, body, err := prepareFilters(base.FilterDescriptor, options.filters, base.BodyFilterFields)
	if err != nil || query.Get("flag") != "server" || string(body["flag"]) != "true" {
		t.Fatal("wire alias Body attribute classification changed", query, body, err)
	}
	rows, err := NewCollection(base).All(context.Background(), WithFilter("flag", true))
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
}
