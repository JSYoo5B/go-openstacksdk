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
	_ []request.ReadCall     = []request.ReadCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Query("k", 1), request.BasePath("p"), request.IgnoreMissing(true)}
	_ []request.ListCall     = []request.ListCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.Query("k", 1), request.BasePath("p"), request.Limit(1), request.Marker("m"), request.MaxItems(1), request.Paginated(false)}
	_ []request.MutationCall = []request.MutationCall{request.Header("k", "v"), request.Headers(nil), request.Microversion("2.1"), request.IgnoreMissing(false), request.Field("k", 1)}
)

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
