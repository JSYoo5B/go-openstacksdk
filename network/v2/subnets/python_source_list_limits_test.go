package subnets_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TestPythonSubnetProxyRawLimitTruthiness covers raw limit values: the
// Requests spelling reaches the wire and only a Python-truthy original limit
// performs the marker GET after a nonempty page without a next link.
func TestPythonSubnetProxyRawLimitTruthiness(t *testing.T) {
	for _, tc := range []struct {
		raw, first string
		marker     bool
	}{
		{`-1`, "limit=-1", true},
		{`[1,2]`, "limit=1&limit=2", true},
		{`"x"`, "limit=x", true},
		{`false`, "limit=False", false},
		{`0.0`, "limit=0.0", false},
		{`""`, "limit=", false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			var calls []nativeSubnetCall
			api := pythonCoreSubnetAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.URL.Query().Get("marker") != "" {
					return nativeSubnetWire(200, `{"subnets":[]}`)
				}
				return nativeSubnetWire(200, `{"subnets":[{"id":"one"},{"id":"two"}]}`)
			})
			rows, err := subnetRawCapRows(t, api, context.Background(), sdk.WithRecordListQuery("limit", json.RawMessage(tc.raw)))
			want := 1
			if tc.marker {
				want = 2
			}
			if err != nil || len(rows) != 2 || len(calls) != want || calls[0].query != tc.first {
				t.Fatal(rows, err, calls)
			}
			if tc.marker && calls[1].query != tc.first+"&marker=two" {
				t.Fatal(calls)
			}
		})
	}
}

// TestPythonSubnetProxyEmptyBasePathSkipsVersionNormalization covers
// base_path="": Resource.list performs no session call, so an invalid raw
// microversion is unobserved; a nonempty route still rejects it before HTTP.
func TestPythonSubnetProxyEmptyBasePathSkipsVersionNormalization(t *testing.T) {
	for _, version := range []string{`"broken"`, `["wrong"]`} {
		var calls []nativeSubnetCall
		api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response {
			t.Fatal("empty route reached HTTP")
			return nil
		})
		rows, err := subnetRawCapRows(t, api, context.Background(), sdk.WithRecordListQuery("base_path", ""), sdk.WithRecordListQuery("microversion", json.RawMessage(version)))
		if err != nil || len(rows) != 0 || len(calls) != 0 {
			t.Fatal(version, rows, err, calls)
		}
		_, err = subnetRawCapRows(t, api, context.Background(), sdk.WithRecordListQuery("microversion", json.RawMessage(version)))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(version, err, calls)
		}
	}
}
