package floatingips_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonAvailableIPFalseyPortIDAndRawExtensionProjection(t *testing.T) {
	for _, raw := range []string{`null`, `""`, `false`, `0`, `0.0`, `1e-9999`, `[]`, `{}`, ``} {
		t.Run(raw, func(t *testing.T) {
			var calls []pythonParityCoreFloatingIPCall
			field := ``
			if raw != "" {
				field = `,"port_id":` + raw
			}
			api, _ := pythonParityCoreFloatingIPAPI(t, &calls, func(*http.Request) string {
				return `{"floatingips":[{"id":"busy","port_id":"port"},{"id":"free","floating_ip_address":"192.0.2.1","x_extension":{"n":9007199254740993}` + field + `},[]],"next":42}`
			})
			result, err := api.FindAvailableIPRecord(context.Background())
			if err != nil || result == nil || result.Resource == nil || string(result.Resource.Body["id"]) != `"free"` || len(calls) != 1 || calls[0].query != "" {
				t.Fatal(result, err, calls)
			}
			if string(result.Wire.Body["x_extension"]) != `{"n":9007199254740993}` || string(result.Resource.Body["name"]) != `"192.0.2.1"` {
				t.Fatal(result.Resource.Body, result.Wire.Body)
			}
			result.Resource.Body["x_extension"][0] = 'X'
			if !json.Valid(result.Wire.Body["x_extension"]) {
				t.Fatal("aliased Resource and Wire")
			}
		})
	}
}
func TestPythonAvailableIPEarlyStopPagingAndExhaustion(t *testing.T) {
	for _, free := range []bool{false, true} {
		t.Run(map[bool]string{false: "none", true: "second-page"}[free], func(t *testing.T) {
			var calls []pythonParityCoreFloatingIPCall
			var cloud *testcloud.Cloud
			api, created := pythonParityCoreFloatingIPAPI(t, &calls, func(r *http.Request) string {
				if r.URL.Query().Get("marker") == "tail" {
					if free {
						return `{"floatingips":[{"id":"free","port_id":null}]}`
					}
					return `{"floatingips":[]}`
				}
				return `{"floatingips":[{"id":"busy","port_id":[1]}],"floatingips_links":[{"rel":"next","href":"` + cloud.Server.URL + pythonParityCoreFloatingIPBase + `?marker=tail"}]}`
			})
			cloud = created
			result, err := api.FindAvailableIPRecord(context.Background())
			if err != nil || len(calls) != 2 || (result != nil) != free {
				t.Fatal(result, err, calls)
			}
		})
	}
}
func TestPythonAvailableIPAcceptedStatusAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    int
		body    string
		wantErr bool
	}{
		{"203", 203, `{"floatingips":[{"id":"free"}]}`, false},
		{"bad-page", 200, `{"floatingips":[`, true},
		{"bad-row", 200, `{"floatingips":[[]]}`, true},
		{"500", 500, `{"error":"failed"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			count := 0
			nativeFloatingIPRecorder(cloud, &[]nativeFloatingIPCall{}, func(*http.Request) *http.Response { count++; return nativeFloatingIPWire(tc.code, tc.body) })
			api := floatingips.New(nativeFloatingIPClient(cloud))
			result, err := api.FindAvailableIPRecord(context.Background())
			if (err != nil) != tc.wantErr || count != 1 {
				t.Fatal(result, err, count)
			}
			if !tc.wantErr && (result == nil || result.StatusCode != tc.code) {
				t.Fatal(result)
			}
			if tc.wantErr && result != nil {
				t.Fatal("fabricated available result", result)
			}
		})
	}
}
func TestPythonAvailableIPOwnedTransportOptionsAndCancellation(t *testing.T) {
	var calls []pythonParityCoreFloatingIPCall
	headers := map[string]string{"X-Trace": "owned"}
	opt := floatingips.WithAvailableIPHeaders(headers)
	headers["X-Trace"] = "changed"
	api, _ := pythonParityCoreFloatingIPAPI(t, &calls, func(r *http.Request) string {
		if r.Header.Get("X-Trace") != "owned" {
			t.Error(r.Header)
		}
		return `{"floatingips":[{"id":"free"}]}`
	})
	result, err := api.FindAvailableIPRecord(context.Background(), opt)
	if err != nil || result == nil || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
	for _, opts := range [][]floatingips.AvailableIPOption{{nil}, {floatingips.WithAvailableIPMicroversion("2.1\r\nbad")}} {
		result, err = api.FindAvailableIPRecord(context.Background(), opts...)
		if err == nil || result != nil || len(calls) != 1 {
			t.Fatal(result, err, calls)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	result, err = api.FindAvailableIPRecord(ctx, func(*floatingips.AvailableIPOpts) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) || result != nil || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
	result, err = api.FindAvailableIPRecord(context.Background(), func(*floatingips.AvailableIPOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || result != nil || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
}
