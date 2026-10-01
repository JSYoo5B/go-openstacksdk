package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"math"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2/pagination"
	"gophercloudsdk/internal/testcloud"
)

type bodyFilterEntry struct {
	ID, Name, Status string
	Body             map[string]json.RawMessage
}

func bodyEntry(id, name, status, raw string) bodyFilterEntry {
	return bodyFilterEntry{id, name, status, map[string]json.RawMessage{"values": json.RawMessage(raw), "flag": json.RawMessage("true")}}
}

func bodyAdapter(rows []bodyFilterEntry, calls *atomic.Int32) Adapter[bodyFilterEntry] {
	return Adapter[bodyFilterEntry]{Kind: "entries", BodyFilterFields: map[string]string{"values": "values", "aliases": "values", "flag": "flag"},
		BodyFilterValue: func(v *bodyFilterEntry, key string) (json.RawMessage, error) {
			if raw, exists := v.Body[key]; exists {
				return raw, nil
			}
			return json.RawMessage("null"), nil
		},
		ID: func(v *bodyFilterEntry) string { return v.ID }, Name: func(v *bodyFilterEntry) string { return v.Name },
		Status: func(v *bodyFilterEntry) string { return v.Status }, NameQuery: func(name string) string { return name },
		Iterate: func(_ context.Context, q url.Values) iter.Seq2[*bodyFilterEntry, error] {
			return func(yield func(*bodyFilterEntry, error) bool) {
				calls.Add(1)
				for i := range rows {
					if !yield(&rows[i], nil) {
						return
					}
				}
			}
		}}
}

func TestCollectionBodyFiltersSnapshotsBulkAliasesAndConcurrentReuse(t *testing.T) {
	var calls atomic.Int32
	a := bodyAdapter([]bodyFilterEntry{bodyEntry("one", "", "", `[{"n":1}]`), bodyEntry("two", "", "", `[{"n":2}]`)}, &calls)
	fields := a.BodyFilterFields
	c := NewCollection(a)
	fields["aliases"] = "flag"
	value := []map[string]any{{"n": 2}}
	bulk := map[string]any{"values": []map[string]any{{"n": 1}}}
	opts := []ListOption{WithBodyFilter("flag", false), WithBodyFilters(bulk), WithBodyFilter("aliases", value)}
	seq := c.List(context.Background(), opts...)
	value[0]["n"], bulk["values"], opts[0] = 99, nil, nil
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			matched := 0
			for row, err := range seq {
				matched++
				if err != nil || row.ID != "two" {
					t.Error(row, err)
				}
			}
			if matched != 1 {
				t.Error("snapshot yielded", matched)
			}
		}()
	}
	group.Wait()
	if calls.Load() != 8 {
		t.Fatal(calls.Load())
	}
	for _, clear := range []ListOption{WithBodyFilters(nil), WithBodyFilters(map[string]any{})} {
		rows, err := c.All(context.Background(), WithBodyFilter("values", nil), clear)
		if err != nil || len(rows) != 2 {
			t.Fatal(rows, err)
		}
	}
}

func TestCollectionBodyFiltersOptionApplicationOwnsRawBytes(t *testing.T) {
	var calls, applied atomic.Int32
	raw := json.RawMessage(`[1]`)
	option := WithBodyFilter("values", raw)
	raw[1] = '9'
	mutate := func(o *listOptions) error {
		if applied.Add(1) == 1 {
			o.bodyFilters[0].values["values"][1] = '2'
		}
		return nil
	}
	c := NewCollection(bodyAdapter([]bodyFilterEntry{bodyEntry("one", "", "", `[1]`), bodyEntry("two", "", "", `[2]`)}, &calls))
	seq := c.List(context.Background(), option, mutate)
	for _, want := range []string{"two", "one"} {
		var ids []string
		for row, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, row.ID)
		}
		if len(ids) != 1 || ids[0] != want {
			t.Fatal(ids, want)
		}
	}
}

func TestCollectionBodyFiltersPreserveNullEmptyAndExactArrayValues(t *testing.T) {
	checks := []struct {
		body   string
		filter any
		match  bool
	}{
		{"null", nil, true}, {"[]", nil, false}, {"null", []string{}, false}, {"[]", []string{}, true},
		{`["a","b"]`, []string{"b", "a"}, false}, {`["a","b"]`, []string{"a"}, false},
		{`[{"a":1,"b":2}]`, json.RawMessage(`[{"a":1}]`), false},
		{`[{"a":1,"b":2}]`, json.RawMessage(`[{"b":2,"a":1.0}]`), true},
		{`[false]`, []int{0}, false}, {`[1]`, 1, false},
	}
	for i, check := range checks {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var calls atomic.Int32
			c := NewCollection(bodyAdapter([]bodyFilterEntry{bodyEntry("row", "", "", check.body)}, &calls))
			rows, err := c.All(context.Background(), WithBodyFilter("values", check.filter))
			if err != nil || (len(rows) == 1) != check.match || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
		})
	}
}

func TestCollectionBodyFiltersValidateBeforeHTTP(t *testing.T) {
	var calls atomic.Int32
	base := bodyAdapter(nil, &calls)
	checks := []struct {
		name    string
		adapter Adapter[bodyFilterEntry]
		option  ListOption
		want    error
	}{
		{"unknown", base, WithBodyFilter("unknown", nil), ErrInvalidOption},
		{"empty field", base, WithBodyFilter(" ", nil), ErrInvalidOption},
		{"bulk alias collision", base, WithBodyFilters(map[string]any{"values": nil, "aliases": nil}), ErrInvalidOption},
		{"malformed JSON", base, WithBodyFilter("values", json.RawMessage(`[] []`)), ErrInvalidOption},
		{"NaN", base, WithBodyFilter("values", math.NaN()), ErrInvalidOption},
		{"function", base, WithBodyFilter("values", func() {}), ErrInvalidOption},
	}
	unsupported := base
	unsupported.BodyFilterFields = nil
	checks = append(checks, struct {
		name    string
		adapter Adapter[bodyFilterEntry]
		option  ListOption
		want    error
	}{"unsupported clear", unsupported, WithBodyFilters(nil), ErrUnsupported})
	missingSelector := base
	missingSelector.BodyFilterValue = nil
	checks = append(checks, struct {
		name    string
		adapter Adapter[bodyFilterEntry]
		option  ListOption
		want    error
	}{"missing selector", missingSelector, WithBodyFilter("values", nil), ErrUnsupported})
	badDescriptor := base
	badDescriptor.BodyFilterFields = map[string]string{"aliases": "values"}
	checks = append(checks, struct {
		name    string
		adapter Adapter[bodyFilterEntry]
		option  ListOption
		want    error
	}{"bad descriptor", badDescriptor, WithBodyFilters(nil), ErrInvalidOption})
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			seq := NewCollection(check.adapter).List(context.Background(), check.option)
			if calls.Load() != 0 {
				t.Fatal("option was eager")
			}
			yields := 0
			for row, err := range seq {
				yields++
				if row != nil || !errors.Is(err, check.want) {
					t.Fatal(row, err)
				}
			}
			if yields != 1 || calls.Load() != 0 {
				t.Fatal(yields, calls.Load())
			}
		})
	}
}

type bodyFilterPage struct{ pagination.LinkedPageBase }

func (p bodyFilterPage) IsEmpty() (bool, error) {
	rows, err := extractBodyEntries(p)
	return len(rows) == 0, err
}
func extractBodyEntries(p pagination.Page) ([]bodyFilterEntry, error) {
	var body struct {
		Rows []bodyFilterEntry `json:"rows"`
	}
	err := p.(bodyFilterPage).ExtractInto(&body)
	return body.Rows, err
}

func TestCollectionBodyFiltersRunAfterRawCapsOnBothIteratorPaths(t *testing.T) {
	rows := []bodyFilterEntry{bodyEntry("body-miss", "wanted", "ACTIVE", `[0]`), bodyEntry("name-miss", "other", "ACTIVE", `[1]`), bodyEntry("status-miss", "wanted", "ERROR", `[1]`), bodyEntry("match", "wanted", "ACTIVE", `[1]`)}
	for _, pager := range []bool{false, true} {
		t.Run(fmt.Sprint("pager=", pager), func(t *testing.T) {
			var calls atomic.Int32
			a := bodyAdapter(rows, &calls)
			if pager {
				cloud := testcloud.New(t)
				client := cloud.Client("network", "/sdk")
				cloud.Mux.HandleFunc("GET /sdk/rows", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Has("values") || r.URL.Query().Get("vendor") != "wire" || r.URL.Query().Get("name") != "wanted" || r.URL.Query().Get("status") != "ACTIVE" {
						t.Error(r.URL)
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{"rows": rows})
				})
				a.Iterate = nil
				a.List = func(q url.Values) pagination.Pager {
					return pagination.NewPager(client, client.ServiceURL("rows")+"?"+q.Encode(), func(p pagination.PageResult) pagination.Page {
						return bodyFilterPage{pagination.LinkedPageBase{PageResult: p}}
					})
				}
				a.Extract = extractBodyEntries
			}
			c := NewCollection(a)
			for _, maximum := range []int{3, 4} {
				got, err := c.All(context.Background(), WithBodyFilter("values", []int{1}), WithName("wanted"), WithStatus("ACTIVE"), WithMaxItems(maximum), WithQuery("vendor", "wire"))
				want := 0
				if maximum == 4 {
					want = 1
				}
				if err != nil || len(got) != want || want == 1 && got[0].ID != "match" {
					t.Fatal(got, err)
				}
			}
			if calls.Load() != 2 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestCollectionBodyFiltersProjectionFailuresAreTerminalAndKeepCause(t *testing.T) {
	var calls atomic.Int32
	a := bodyAdapter([]bodyFilterEntry{bodyEntry("one", "other", "", `[]`), bodyEntry("two", "wanted", "", `[]`)}, &calls)
	cause := errors.New("projection failed")
	a.BodyFilterValue = func(*bodyFilterEntry, string) (json.RawMessage, error) { return nil, cause }
	rows, err := NewCollection(a).All(context.Background(), WithName("wanted"), WithBodyFilter("values", []string{}))
	if rows != nil || !errors.Is(err, cause) || calls.Load() != 1 {
		t.Fatal(rows, err, calls.Load())
	}
	a.BodyFilterValue = func(*bodyFilterEntry, string) (json.RawMessage, error) { return json.RawMessage(`{]`), nil }
	_, err = NewCollection(a).All(context.Background(), WithBodyFilter("values", nil))
	var syntax *json.SyntaxError
	if err == nil || !errors.As(err, &syntax) {
		t.Fatal(err)
	}
	_, err = NewCollection(a).All(context.Background(), WithBodyFilter("values", math.NaN()))
	var marshal *json.UnsupportedValueError
	if !errors.Is(err, ErrInvalidOption) || !errors.As(err, &marshal) {
		t.Fatal(err)
	}
}
