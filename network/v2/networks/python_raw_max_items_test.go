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

func networkRawCapRows(t *testing.T, api *sdk.API, ctx context.Context, options ...sdk.RecordListOption) ([]*sdk.NetworkRecord, error) {
	t.Helper()
	proxy, err := api.ProxyCloudListRecords(ctx, nil, options...)
	if err != nil {
		return nil, err
	}
	var rows []*sdk.NetworkRecord
	generator, ok := proxy.Value.(*sdk.RecordListGenerator)
	if !ok {
		t.Fatalf("expected opaque generator: %#v", proxy)
	}
	for row, err := range generator.Records {
		if err != nil {
			return rows, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}
func TestPythonNetworkRawMaxItemsJSONDomainAndDeferredComparison(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		rows  int
		fails bool
	}{{"null", 3, false}, {"false", 3, false}, {"0", 3, false}, {`""`, 3, false}, {"[]", 3, false}, {"{}", 3, false}, {"true", 1, false}, {"-1", 0, false}, {"0.5", 1, false}, {"1.5", 2, false}, {`"2"`, 0, true}, {"[1]", 0, true}, {`{"x":1}`, 0, true}} {
		for _, empty := range []bool{false, true} {
			t.Run(tc.raw+map[bool]string{false: "/rows", true: "/empty"}[empty], func(t *testing.T) {
				var calls []nativeNetworkCall
				body := `{"networks":[{"id":"one"},{"id":"two"},{"id":"three"}]}`
				if empty {
					body = `{"networks":[]}`
				}
				api := pythonCoreNetworkAPI(t, &calls, func(r *http.Request) *http.Response {
					if r.URL.Query().Get("limit") != "10" || r.URL.Query().Has("max_items") {
						t.Fatal(r.URL)
					}
					if r.URL.Query().Get("marker") != "" {
						if r.URL.Query().Get("marker") != "three" {
							t.Fatal(r.URL)
						}
						return nativeNetworkWire(203, `{"networks":[]}`)
					}
					return nativeNetworkWire(203, body)
				})
				proxy, err := api.ProxyCloudListRecords(context.Background(), nil, sdk.WithCloudRecordListQuery("max_items", json.RawMessage(tc.raw)), sdk.WithRecordListLimit(10))
				if err != nil || len(calls) != 0 {
					t.Fatal(proxy, err, calls)
				}
				var rows []*sdk.NetworkRecord
				for row, cause := range proxy.Value.(*sdk.RecordListGenerator).Records {
					if cause != nil {
						err = cause
						break
					}
					rows = append(rows, row)
				}
				want := tc.rows
				if empty {
					want = 0
				}
				fail := tc.fails && !empty
				wantCalls := 1
				if !empty && tc.rows == 3 {
					wantCalls = 2
				}
				if len(calls) != wantCalls || len(rows) != want || (err != nil) != fail {
					t.Fatal(tc, empty, len(rows), err, calls)
				}
				if fail {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
func TestPythonNetworkRawMaxItemsExactPageBoundaryObservesNextResponse(t *testing.T) {
	for _, mode := range []string{"empty", "row", "http-error", "close-error", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			var calls []nativeNetworkCall
			cause := errors.New("cap boundary " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			api := pythonCoreNetworkAPI(t, &calls, func(r *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeNetworkWire(200, `{"networks":[{"id":"one"},{"id":"two"}],"networks_links":[{"rel":"next","href":"?marker=two"}]}`)
				}
				if len(calls) != 2 || r.URL.Query().Get("marker") != "two" {
					t.Fatal(r.URL, calls)
				}
				if mode == "http-error" {
					return nativeNetworkWire(500, `{"error":"next page"}`)
				}
				if mode == "empty" {
					return nativeNetworkWire(200, `{"networks":[]}`)
				}
				response := nativeNetworkWire(203, `{"networks":[{"id":"never-yielded"}]}`)
				if mode == "close-error" {
					response.Body = &pythonFindFaultBody{data: `{"networks":[{"id":"never-yielded"}]}`, close: cause}
				}
				if mode == "cancel" {
					response.Body = &pythonFindFaultBody{data: `{"networks":[{"id":"never-yielded"}]}`, onClose: func() { cancel(cause) }}
				}
				return response
			})
			rows, err := networkRawCapRows(t, api, ctx, sdk.WithCloudRecordListQuery("max_items", json.RawMessage("2")), sdk.WithRecordListLimit(10))
			if len(calls) != 2 || len(rows) != 2 {
				t.Fatal(rows, err, calls)
			}
			if mode == "empty" || mode == "row" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal(rows, calls)
			}
			if mode == "close-error" || mode == "cancel" {
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestPythonNetworkRawMaxItemsCountsFilteredRowsAndStopsBeforeConstructor(t *testing.T) {
	for _, mode := range []string{"local-filter", "malformed-row", "negative-first-row"} {
		t.Run(mode, func(t *testing.T) {
			var calls []nativeNetworkCall
			body := `{"networks":[{"id":"filtered","dns_domain":"drop"},{"id":"kept","dns_domain":"keep"},{"id":"beyond","dns_domain":"keep"}]}`
			options := []sdk.RecordListOption{sdk.WithCloudRecordListQuery("max_items", json.RawMessage("2")), sdk.WithRecordListLimit(10)}
			want := 1
			if mode == "local-filter" {
				options = append(options, sdk.WithRecordListQuery("dns_domain", "keep"))
			}
			if mode == "malformed-row" {
				body = `{"networks":[{"id":"kept"},false]}`
				options[0] = sdk.WithCloudRecordListQuery("max_items", json.RawMessage("1"))
			}
			if mode == "negative-first-row" {
				body = `{"networks":[false]}`
				options[0] = sdk.WithCloudRecordListQuery("max_items", json.RawMessage("-1"))
				want = 0
			}
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return nativeNetworkWire(200, body) })
			rows, err := networkRawCapRows(t, api, context.Background(), options...)
			if err != nil || len(rows) != want || len(calls) != 1 {
				t.Fatal(rows, err, calls)
			}
			if want == 1 && string(rows[0].Resource.Body["id"]) != `"kept"` {
				t.Fatal(rows[0])
			}
		})
	}
}
func TestPythonNetworkRawMaxItemsOpaqueJMESAndOwnedJSON(t *testing.T) {
	var calls []nativeNetworkCall
	api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeNetworkWire(200, `{"networks":[{"id":"one"},{"id":"two"}]}`)
	})
	options := []sdk.RecordListOption{sdk.WithCloudRecordListQuery("max_items", json.RawMessage(`"wrong"`)), sdk.WithRecordListJMESPath("`[]`")}
	proxy, err := api.ProxyCloudListRecords(context.Background(), nil, options...)
	if err != nil || string(proxy.JSON) != "[]" || len(calls) != 0 {
		t.Fatal(proxy, err, calls)
	}
	raw := json.RawMessage("true")
	owned := sdk.WithCloudRecordListQuery("max_items", raw)
	raw[0] = 'X'
	rows, err := networkRawCapRows(t, api, context.Background(), owned, sdk.WithRecordListLimit(10))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("cancel before cap comparison")
	cancel(cause)
	_, err = networkRawCapRows(t, api, ctx, sdk.WithCloudRecordListQuery("max_items", json.RawMessage(`"bad"`)))
	if !errors.Is(err, cause) || len(calls) != 1 {
		t.Fatal(err, calls)
	}
}

func TestPythonNetworkRawMaxItemsFalseyAndRawLimitHint(t *testing.T) {
	for _, tc := range []struct {
		raw    string
		limits []string
	}{
		{"null", nil}, {"false", nil}, {"0", nil}, {`""`, nil}, {"[]", nil}, {"{}", nil},
		{"true", []string{"True"}}, {"-1", []string{"-1"}}, {"0.5", []string{"0.5"}}, {"1.5", []string{"1.5"}},
		{`"2"`, []string{"2"}}, {"[1,2]", []string{"1", "2"}}, {`{"x":1}`, []string{"x"}},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(r *http.Request) *http.Response {
				got := r.URL.Query()["limit"]
				if len(got) != len(tc.limits) {
					t.Fatal(tc.raw, r.URL)
				}
				for i, value := range got {
					if value != tc.limits[i] {
						t.Fatal(tc.raw, r.URL)
					}
				}
				return nativeNetworkWire(203, `{"networks":[]}`)
			})
			rows, err := networkRawCapRows(t, api, context.Background(), sdk.WithCloudRecordListQuery("max_items", json.RawMessage(tc.raw)))
			if err != nil || len(rows) != 0 || len(calls) != 1 {
				t.Fatal(rows, err, calls)
			}
		})
	}
}
