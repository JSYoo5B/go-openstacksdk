package request_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func custom() request.AnyCall { return request.Custom(func(*struct{}) error { return nil }) }

// Category membership is the compile-time contract; these assignments fail
// to build when a constructor loses a category.
var (
	_ = []request.GetCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), custom()}
	_ = []request.GetQueryCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Query("k", 1), custom()}
	_ = []request.FindCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Query("k", 1), request.IgnoreMissing(true), custom()}
	_ = []request.DeleteCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.IgnoreMissing(false), custom()}
	_ = []request.ListCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Query("k", 1), request.Limit(1), request.Marker("m"), request.MaxItems(1), request.Paginated(false), custom()}
	_ = []request.ListBasePathCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Query("k", 1), request.BasePath("p"), request.Limit(1), request.Marker("m"), request.MaxItems(1), request.Paginated(false), custom()}
	_ = []request.WriteCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Field("k", 1), custom()}
	_ = []request.NativeCall{custom()}
	_ = []request.NativeHeaderCall{request.Header("k", "v"), custom()}
	_ = []request.NativeQueryCall{request.Query("k", "v"), custom()}
	_ = []request.NativeQueryHeaderCall{request.Query("k", "v"), request.Header("k", "v"), custom()}
	_ = []request.NativeFieldCall{request.Field("k", 1), custom()}
	_ = []request.NativeFieldHeaderCall{request.Field("k", 1), request.Header("k", "v"), custom()}
	_ = []request.NativeFieldQueryCall{request.Field("k", 1), request.Query("k", "v"), custom()}
)

// callCategories is the package documentation's table: for each category,
// the shared constructors it accepts besides Custom.
var callCategories = map[string][]string{
	"GetCall":               {"Header", "Headers", "Microversion"},
	"GetQueryCall":          {"Header", "Headers", "Microversion", "Query"},
	"FindCall":              {"Header", "Headers", "Microversion", "Query", "IgnoreMissing"},
	"DeleteCall":            {"Header", "Headers", "Microversion", "IgnoreMissing"},
	"ListCall":              {"Header", "Headers", "Microversion", "Query", "Limit", "Marker", "MaxItems", "Paginated"},
	"ListBasePathCall":      {"Header", "Headers", "Microversion", "Query", "BasePath", "Limit", "Marker", "MaxItems", "Paginated"},
	"WriteCall":             {"Header", "Headers", "Microversion", "Field"},
	"NativeCall":            {},
	"NativeHeaderCall":      {"Header"},
	"NativeQueryCall":       {"Query"},
	"NativeQueryHeaderCall": {"Query", "Header"},
	"NativeFieldCall":       {"Field"},
	"NativeFieldHeaderCall": {"Field", "Header"},
	"NativeFieldQueryCall":  {"Field", "Query"},
}

func categoriesOf(c request.Call) map[string]bool {
	_, get := c.(request.GetCall)
	_, getQuery := c.(request.GetQueryCall)
	_, find := c.(request.FindCall)
	_, del := c.(request.DeleteCall)
	_, list := c.(request.ListCall)
	_, listBase := c.(request.ListBasePathCall)
	_, write := c.(request.WriteCall)
	_, native := c.(request.NativeCall)
	_, nativeHeader := c.(request.NativeHeaderCall)
	_, nativeQuery := c.(request.NativeQueryCall)
	_, nativeQueryHeader := c.(request.NativeQueryHeaderCall)
	_, nativeField := c.(request.NativeFieldCall)
	_, nativeFieldHeader := c.(request.NativeFieldHeaderCall)
	_, nativeFieldQuery := c.(request.NativeFieldQueryCall)
	return map[string]bool{
		"GetCall": get, "GetQueryCall": getQuery, "FindCall": find, "DeleteCall": del,
		"ListCall": list, "ListBasePathCall": listBase, "WriteCall": write,
		"NativeCall": native, "NativeHeaderCall": nativeHeader, "NativeQueryCall": nativeQuery,
		"NativeQueryHeaderCall": nativeQueryHeader, "NativeFieldCall": nativeField,
		"NativeFieldHeaderCall": nativeFieldHeader, "NativeFieldQueryCall": nativeFieldQuery,
	}
}

// Each constructor implements exactly the categories whose combination
// allows its control, so every other combination does not compile. Custom
// implements every category.
func TestCallCategoriesAreExact(t *testing.T) {
	for name, call := range map[string]request.Call{
		"Header":        request.Header("k", "v"),
		"Headers":       request.Headers(nil),
		"Microversion":  request.Microversion("2.1"),
		"Query":         request.Query("k", 1),
		"IgnoreMissing": request.IgnoreMissing(true),
		"BasePath":      request.BasePath("p"),
		"Limit":         request.Limit(1),
		"Marker":        request.Marker("m"),
		"MaxItems":      request.MaxItems(1),
		"Paginated":     request.Paginated(true),
		"Field":         request.Field("k", 1),
	} {
		got := categoriesOf(call)
		for category, allowed := range callCategories {
			if want := slices.Contains(allowed, name); got[category] != want {
				t.Fatalf("%s in %s: %v, want %v", name, category, got[category], want)
			}
		}
	}
	for category, ok := range categoriesOf(custom()) {
		if !ok {
			t.Fatalf("Custom is not a %s", category)
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
	values := []string{"a", "b"}
	headerCall, queryCall, fieldCall, valuesCall := request.Headers(headers), request.Query("q", raw), request.Field("f", raw), request.Query("v", values)
	headers["X-Trace"] = "changed"
	values[0] = "changed"
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
		{request.Query("t", "text"), request.CallEntry{Kind: request.CallQuery, Key: "t", Form: request.QueryText, Text: "text", JSON: json.RawMessage(`"text"`)}},
		{valuesCall, request.CallEntry{Kind: request.CallQuery, Key: "v", Form: request.QueryList, Values: []string{"a", "b"}, JSON: json.RawMessage(`["a","b"]`)}},
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
		if got.Values != nil {
			got.Values[0] = "escaped"
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
