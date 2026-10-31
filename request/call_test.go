package request_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Category membership is the compile-time contract; these assignments fail
// to build when a constructor gains or loses a category.
var (
	_ []request.GetCall    = []request.GetCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Custom(func(*struct{}) error { return nil })}
	_ []request.FindCall   = []request.FindCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Query("k", 1), request.IgnoreMissing(true), request.Custom(func(*struct{}) error { return nil })}
	_ []request.ListCall   = []request.ListCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Query("k", 1), request.BasePath("p"), request.Limit(1), request.Marker("m"), request.MaxItems(1), request.Paginated(false), request.Custom(func(*struct{}) error { return nil })}
	_ []request.DeleteCall = []request.DeleteCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.IgnoreMissing(false), request.Custom(func(*struct{}) error { return nil })}
	_ []request.WriteCall  = []request.WriteCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Field("k", 1), request.Custom(func(*struct{}) error { return nil })}
)

// Constructors return the narrowest union they belong to, so the reverse
// combinations (BasePath on Find/Get, Query on Get or Delete, IgnoreMissing on
// Get/List/Write, Field on Delete, paging outside List) do not compile. This
// test pins that each constructor implements exactly its documented
// categories.
func TestCallCategoriesAreExact(t *testing.T) {
	type cats struct{ get, find, list, del, write bool }
	of := func(c request.Call) cats {
		_, g := c.(request.GetCall)
		_, f := c.(request.FindCall)
		_, l := c.(request.ListCall)
		_, d := c.(request.DeleteCall)
		_, w := c.(request.WriteCall)
		return cats{g, f, l, d, w}
	}
	all := cats{true, true, true, true, true}
	for name, tc := range map[string]struct {
		call request.Call
		want cats
	}{
		"Header":        {request.Header("k", "v"), all},
		"Headers":       {request.Headers(nil), all},
		"Microversion":  {request.Microversion("2.1"), all},
		"Custom":        {request.Custom(func(*struct{}) error { return nil }), all},
		"Query":         {request.Query("k", 1), cats{find: true, list: true}},
		"IgnoreMissing": {request.IgnoreMissing(true), cats{find: true, del: true}},
		"BasePath":      {request.BasePath("p"), cats{list: true}},
		"Limit":         {request.Limit(1), cats{list: true}},
		"Marker":        {request.Marker("m"), cats{list: true}},
		"MaxItems":      {request.MaxItems(1), cats{list: true}},
		"Paginated":     {request.Paginated(true), cats{list: true}},
		"Field":         {request.Field("k", 1), cats{write: true}},
	} {
		if got := of(tc.call); got != tc.want {
			t.Fatalf("%s: categories %+v, want %+v", name, got, tc.want)
		}
	}
}

type customState struct{ N int }

func TestCustomCallCarriesTypedFunction(t *testing.T) {
	entry, err := request.Entry(request.Custom(func(s *customState) error { s.N++; return nil }))
	if err != nil || entry.Kind != request.CallCustom || entry.Kind.String() != "Custom" {
		t.Fatal(entry, err)
	}
	fn, ok := request.CustomFunc[customState](entry)
	if !ok || fn == nil {
		t.Fatal(fn != nil, ok)
	}
	var s customState
	if err := fn(&s); err != nil || s.N != 1 {
		t.Fatal(s, err)
	}
	if fn, ok := request.CustomFunc[struct{ N int }](entry); ok || fn != nil {
		t.Fatal("custom function matched a different state type")
	}
	header, _ := request.Entry(request.Header("k", "v"))
	if fn, ok := request.CustomFunc[customState](header); ok || fn != nil {
		t.Fatal("non-custom entry produced a function")
	}
	entry, err = request.Entry(request.Custom[customState](nil))
	if fn, ok := request.CustomFunc[customState](entry); err != nil || !ok || fn != nil {
		t.Fatal(fn != nil, ok, err)
	}
}

func TestCallEntriesSnapshotInputs(t *testing.T) {
	headers := map[string]string{"X-Trace": "owned"}
	raw := json.RawMessage(`"owned"`)
	headerCall, queryCall, fieldCall := request.Headers(headers), request.Query("q", raw), request.Field("f", raw)
	headers["X-Trace"] = "changed"
	raw[1] = 'X'
	for _, tc := range []struct {
		call request.Call
		want request.CallEntry
	}{
		{request.Header("X-One", "1"), request.CallEntry{Kind: request.CallHeader, Key: "X-One", Text: "1"}},
		{headerCall, request.CallEntry{Kind: request.CallHeaders, Headers: map[string]string{"X-Trace": "owned"}}},
		{request.Headers(nil), request.CallEntry{Kind: request.CallHeaders}},
		{request.Microversion("2.1"), request.CallEntry{Kind: request.CallMicroversion, Text: "2.1"}},
		{queryCall, request.CallEntry{Kind: request.CallQuery, Key: "q", JSON: json.RawMessage(`"owned"`)}},
		{request.BasePath("alt"), request.CallEntry{Kind: request.CallBasePath, Text: "alt"}},
		{request.IgnoreMissing(false), request.CallEntry{Kind: request.CallIgnoreMissing}},
		{request.Limit(3), request.CallEntry{Kind: request.CallLimit, Int: 3}},
		{request.Marker("m"), request.CallEntry{Kind: request.CallMarker, Text: "m"}},
		{request.MaxItems(0), request.CallEntry{Kind: request.CallMaxItems}},
		{request.Paginated(true), request.CallEntry{Kind: request.CallPaginated, Bool: true}},
		{fieldCall, request.CallEntry{Kind: request.CallField, Key: "f", JSON: json.RawMessage(`"owned"`)}},
	} {
		got, err := request.Entry(tc.call)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%v: got %#v err %v", tc.want.Kind, got, err)
		}
		if got.Headers != nil {
			got.Headers["X-Trace"] = "escaped"
		}
		if got.JSON != nil {
			got.JSON[1] = 'Y'
		}
		again, _ := request.Entry(tc.call)
		if !reflect.DeepEqual(again, tc.want) {
			t.Fatalf("%v: entry aliases call state: %#v", tc.want.Kind, again)
		}
	}
}

func TestCallEntryErrorsAreDeferredOrInvalid(t *testing.T) {
	entry, err := request.Entry(request.Query("bad", func() {}))
	var unsupported *json.UnsupportedTypeError
	if err != nil || entry.Kind != request.CallQuery || entry.JSON != nil || !errors.As(entry.JSONErr, &unsupported) {
		t.Fatal(entry, err)
	}
	if _, err := request.Entry(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := request.UnsupportedCall(request.CallEntry{Kind: request.CallBasePath}); !errors.Is(err, resource.ErrInvalidOption) || err.Error() != "invalid option: BasePath call option is unsupported by this operation" {
		t.Fatal(err)
	}
	if request.CallKind(0).String() != "CallKind(0)" {
		t.Fatal(request.CallKind(0))
	}
}
