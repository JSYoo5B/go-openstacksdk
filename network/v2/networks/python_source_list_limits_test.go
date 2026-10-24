package networks_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// networkProxyRows collects ProxyListRecords (Proxy.networks) rows.
func networkProxyRows(t *testing.T, api *sdk.API, options ...sdk.RecordListOption) ([]*sdk.NetworkRecord, error) {
	t.Helper()
	proxy, err := api.ProxyListRecords(context.Background(), options...)
	if err != nil {
		return nil, err
	}
	var rows []*sdk.NetworkRecord
	for row, cause := range proxy.Value.(*sdk.RecordListGenerator).Records {
		if cause != nil {
			return rows, cause
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// TestPythonNetworkProxyRawLimitSpellingAndTruthiness covers raw limit values
// Resource.list sends through Requests: the spelling (False, -1, 1.5, x,
// repeated values) reaches the wire, and Python keeps the original limit for
// its loop, so only a truthy limit performs the marker GET after a nonempty
// page without a next link. Both Proxy lanes (Proxy.networks and the
// connected cloud lane) behave the same.
func TestPythonNetworkProxyRawLimitSpellingAndTruthiness(t *testing.T) {
	for _, tc := range []struct {
		raw, first string
		marker     bool
	}{
		{`-1`, "limit=-1", true},
		{`[1,2]`, "limit=1&limit=2", true},
		{`true`, "limit=True", true},
		{`1.5`, "limit=1.5", true},
		{`"x"`, "limit=x", true},
		{`2`, "limit=2", true},
		{`false`, "limit=False", false},
		{`0.0`, "limit=0.0", false},
		{`0`, "limit=0", false},
		{`""`, "limit=", false},
		{`null`, "", false},
		{`[]`, "", false},
	} {
		for _, lane := range []string{"proxy", "cloud"} {
			t.Run(lane+"/"+tc.raw, func(t *testing.T) {
				var calls []nativeNetworkCall
				api := pythonCoreNetworkAPI(t, &calls, func(r *http.Request) *http.Response {
					if r.URL.Query().Get("marker") != "" {
						return nativeNetworkWire(200, `{"networks":[]}`)
					}
					return nativeNetworkWire(200, `{"networks":[{"id":"one"},{"id":"two"}]}`)
				})
				var rows []*sdk.NetworkRecord
				var err error
				if lane == "proxy" {
					rows, err = networkProxyRows(t, api, sdk.WithRecordListQuery("limit", json.RawMessage(tc.raw)))
				} else {
					rows, err = networkRawCapRows(t, api, context.Background(), sdk.WithCloudRecordListQuery("limit", json.RawMessage(tc.raw)))
				}
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
}

// TestPythonNetworkProxyCapEqualsWholePage covers Proxy.networks with an
// integer max_items equal to one whole page: Python checks the cap before the
// next raw row, so it performs the marker GET and observes its failure.
func TestPythonNetworkProxyCapEqualsWholePage(t *testing.T) {
	for _, status := range []int{200, 503} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(r *http.Request) *http.Response {
			if r.URL.Query().Get("marker") == "two" {
				return nativeNetworkWire(status, `{"networks":[{"id":"three"}]}`)
			}
			return nativeNetworkWire(200, `{"networks":[{"id":"one"},{"id":"two"}]}`)
		})
		rows, err := networkProxyRows(t, api, sdk.WithRecordListQuery("max_items", 2))
		if len(rows) != 2 || len(calls) != 2 || calls[0].query != "limit=2" || calls[1].query != "limit=2&marker=two" || (err != nil) != (status != 200) {
			t.Fatal(status, rows, err, calls)
		}
	}
}

// TestPythonNetworkProxyEmptyBasePathSkipsVersionNormalization covers
// base_path="": Resource.list never enters "while uri", so neither HTTP nor
// KSA microversion normalization happens and an invalid version is
// unobserved. A nonempty route still reports the invalid version before HTTP.
func TestPythonNetworkProxyEmptyBasePathSkipsVersionNormalization(t *testing.T) {
	for _, version := range []string{`"broken"`, `["wrong"]`} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
			t.Fatal("empty route reached HTTP")
			return nil
		})
		rows, err := networkRawCapRows(t, api, context.Background(), sdk.WithCloudRecordListQuery("base_path", json.RawMessage(`""`)), sdk.WithCloudRecordListQuery("microversion", json.RawMessage(version)))
		if err != nil || len(rows) != 0 || len(calls) != 0 {
			t.Fatal(version, rows, err, calls)
		}
		// The Proxy.networks lane binds microversion as a JSON string or null.
		if version[0] == '"' {
			rows, err = networkProxyRows(t, api, sdk.WithRecordListQuery("base_path", ""), sdk.WithRecordListQuery("microversion", json.RawMessage(version)))
			if err != nil || len(rows) != 0 || len(calls) != 0 {
				t.Fatal(version, rows, err, calls)
			}
		}
		_, err = networkRawCapRows(t, api, context.Background(), sdk.WithCloudRecordListQuery("microversion", json.RawMessage(version)))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(version, err, calls)
		}
	}
}
