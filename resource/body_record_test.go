package resource

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/url"
	"testing"

	"gophercloudsdk/internal/jsonfilter"
)

func TestBodyRecordPairingPreservesOriginalRowsAndOwnsFields(t *testing.T) {
	// Identical IDs deliberately prove that pairing is by position, not lookup.
	values := []bodyFilterEntry{{ID: "same", Name: "first"}, {ID: "same", Name: "second"}, {}}
	body := map[string]any{"items": []any{
		map[string]any{"id": "same", "extra": map[string]any{"n": json.Number("9007199254740993"), "nullable": nil}},
		map[string]any{"id": "same", "extra": []any{nil, ""}}, nil,
	}}
	records, err := pairBodyRecords(values, body, "items")
	if err != nil || len(records) != 3 {
		t.Fatal(records, err)
	}
	body["items"].([]any)[0].(map[string]any)["extra"] = false
	values[0].Name = "changed"
	if records[0].Value.Name != "first" || records[1].Value.Name != "second" {
		t.Fatalf("pairing changed: %#v", records)
	}
	matched, err := jsonfilter.EqualJSON(records[0].Fields["extra"], json.RawMessage(`{"n":9007199254740993,"nullable":null}`))
	if err != nil || !matched {
		t.Fatal(string(records[0].Fields["extra"]), err)
	}
	if records[2].Fields != nil {
		t.Fatal("native-accepted null row must retain missing fields", records[2])
	}
	projected, err := BodyRecordField(records[0].Fields, "extra", BodyFieldJSON)
	if err != nil {
		t.Fatal(err)
	}
	projected[0] = '['
	if records[0].Fields["extra"][0] != '{' {
		t.Fatal("projection mutated record")
	}
}

func TestBodyRecordPairingRejectsUnprovenShapeAndCount(t *testing.T) {
	for _, check := range []struct {
		name, envelope string
		body           any
		count          int
	}{
		{"empty envelope", "", map[string]any{"items": []any{}}, 0},
		{"missing envelope", "items", map[string]any{"other": []any{}}, 0},
		{"page array", "items", []any{}, 0},
		{"wrong envelope type", "items", map[string]any{"items": false}, 0},
		{"count mismatch", "items", map[string]any{"items": []any{map[string]any{}}}, 2},
		{"nonobject row", "items", map[string]any{"items": []any{false}}, 1},
	} {
		t.Run(check.name, func(t *testing.T) {
			_, err := pairBodyRecords(make([]bodyFilterEntry, check.count), check.body, check.envelope)
			if !errors.Is(err, ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestBodyRecordFieldNormalizationPresenceAndCauses(t *testing.T) {
	fields := map[string]json.RawMessage{"null": json.RawMessage(`null`), "empty": json.RawMessage(`[]`), "number": json.RawMessage(`" +000024 "`), "broken": json.RawMessage(`{]`)}
	for _, key := range []string{"missing", "null"} {
		for _, kind := range []BodyFieldType{BodyFieldJSON, BodyFieldInteger} {
			got, err := BodyRecordField(fields, key, kind)
			if err != nil || string(got) != "null" {
				t.Fatal(key, got, err)
			}
		}
	}
	got, err := BodyRecordField(fields, "number", BodyFieldInteger)
	if err != nil || string(got) != "24" {
		t.Fatal(got, err)
	}
	for _, kind := range []BodyFieldType{BodyFieldJSON, BodyFieldInteger} {
		_, err := BodyRecordField(fields, "broken", kind)
		var syntax *json.SyntaxError
		if !errors.Is(err, ErrInvalidOption) || !errors.As(err, &syntax) {
			t.Fatal("lost decoding cause", err)
		}
	}
	if _, err := BodyRecordField(fields, "empty", BodyFieldInteger); !errors.Is(err, ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := BodyRecordField(fields, "null", BodyFieldType(255)); !errors.Is(err, ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestCollectionBodyRecordLaneIsSelectedOnlyForPreparedFilters(t *testing.T) {
	typedCalls, rawCalls := 0, 0
	a := Adapter[bodyFilterEntry]{Kind: "entries", BodyFilterFields: map[string]string{"n": "n"},
		Name: func(v *bodyFilterEntry) string { return v.Name },
		IterateControlled: func(context.Context, url.Values, ListControl) iter.Seq2[*bodyFilterEntry, error] {
			return func(yield func(*bodyFilterEntry, error) bool) {
				typedCalls++
				yield(&bodyFilterEntry{ID: "typed"}, nil)
			}
		},
		IterateBodyControlled: func(_ context.Context, q url.Values, control ListControl) iter.Seq2[*BodyRecord[bodyFilterEntry], error] {
			return func(yield func(*BodyRecord[bodyFilterEntry], error) bool) {
				rawCalls++
				if q.Get("extension") != "wire" || control.MaxItems != 1 {
					t.Error("controls or raw query lost", q, control)
				}
				yield(&BodyRecord[bodyFilterEntry]{Value: bodyFilterEntry{ID: "raw"}, Fields: map[string]json.RawMessage{"n": json.RawMessage(`9007199254740993`)}}, nil)
			}
		},
		BodyFilterRecordValue: func(record *BodyRecord[bodyFilterEntry], key string) (json.RawMessage, error) {
			return BodyRecordField(record.Fields, key, BodyFieldInteger)
		}}
	c := NewCollection(a)
	for _, opts := range [][]ListOption{nil, {WithBodyFilters(nil)}, {WithBodyFilter("n", 2), WithBodyFilters(map[string]any{})}} {
		rows, err := c.All(context.Background(), opts...)
		if err != nil || len(rows) != 1 || rows[0].ID != "typed" {
			t.Fatal(rows, err)
		}
	}
	rows, err := c.All(context.Background(), WithBodyFilter("n", json.Number("9007199254740993")), WithMaxItems(1), WithQuery("extension", "wire"))
	if err != nil || len(rows) != 1 || rows[0].ID != "raw" || typedCalls != 3 || rawCalls != 1 {
		t.Fatal(rows, err, typedCalls, rawCalls)
	}
	// Invalid selected raw values must not be hidden by an earlier name mismatch.
	a.BodyFilterRecordValue = func(*BodyRecord[bodyFilterEntry], string) (json.RawMessage, error) {
		return BodyRecordField(map[string]json.RawMessage{"n": json.RawMessage(`true`)}, "n", BodyFieldInteger)
	}
	_, err = NewCollection(a).All(context.Background(), WithName("other"), WithBodyFilter("n", 1), WithMaxItems(1), WithQuery("extension", "wire"))
	var operation *OperationError
	if !errors.Is(err, ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "list" {
		t.Fatal(err)
	}
	a.IterateBodyControlled = nil
	_, err = NewCollection(a).All(context.Background(), WithBodyFilters(nil))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatal("raw projection requires a raw iterator even for explicit clearing", err)
	}
}
