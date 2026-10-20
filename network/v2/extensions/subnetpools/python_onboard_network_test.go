package subnetpools_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonOnboardNetworkSubnetsUnwrappedBodyAndArbitraryJSON(t *testing.T) {
	for _, raw := range []string{`{"onboarded_subnets":["s1"]}`, `[null,false,9007199254740993]`, `null`, `false`, `0`, `"done"`} {
		t.Run(raw, func(t *testing.T) {
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.Method != "PUT" || r.URL.EscapedPath() != poolRecordsPath+"/pool%2Fid%3F%25/onboard_network_subnets" || r.URL.RawQuery != "" {
					t.Error(r.Method, r.URL)
				}
				reply := nativePoolWire(203, raw)
				reply.Header.Set("X-Request-Id", "actual")
				return reply
			})
			value, err := api.OnboardNetworkSubnetsRecord(context.Background(), "pool/id?%", "network/id")
			if err != nil || value == nil || len(calls) != 1 || calls[0].body != `{"network_id":"network/id"}` || string(value.Value) != raw || string(value.Envelope) != raw || value.StatusCode != 203 || value.Header.Get("X-Request-Id") != "actual" {
				t.Fatal(value, err, calls)
			}
			own := value.Clone()
			own.Value[0] = 'X'
			own.Envelope[0] = 'X'
			own.Header.Set("X-Request-Id", "changed")
			if string(value.Value) != raw || string(value.Envelope) != raw || value.Header.Get("X-Request-Id") != "actual" {
				t.Fatal("proof alias", value)
			}
		})
	}
}
func TestPythonOnboardNetworkSubnetsHTTPAndDecodeFailures(t *testing.T) {
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{"error":"missing"}`}, {503, "fault"}, {204, ""}, {200, "{bad"}} {
		t.Run(tc.body, func(t *testing.T) {
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(tc.code, tc.body) })
			value, err := api.OnboardNetworkSubnetsRecord(context.Background(), "pool", "")
			if err == nil || len(calls) != 1 || calls[0].body != `{"network_id":""}` {
				t.Fatal(value, err, calls)
			}
			if tc.code >= 400 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != tc.code || string(native.Body) != tc.body {
					t.Fatal(err)
				}
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != tc.code || string(proof.Body) != tc.body || value == nil {
					t.Fatal(value, err)
				}
			}
		})
	}
}
func TestPythonOnboardNetworkSubnetsOptionsOwnershipAndGuard(t *testing.T) {
	var calls []nativePoolCall
	api, _ := nativePoolAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Owned") != "original" || r.Header.Get("OpenStack-API-Version") != "network 2.1" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.Header)
		}
		return nativePoolWire(200, `{}`)
	})
	api.RawClient().Microversion = "2.0"
	api.RawClient().MoreHeaders = map[string]string{"OpenStack-API-Version": "network 2.0"}
	headers := map[string]string{"X-Owned": "original"}
	version := "2.1"
	option := subnetpools.WithOnboardNetworkOpts(subnetpools.OnboardNetworkOpts{Headers: headers, Microversion: &version})
	headers["X-Owned"] = "changed"
	version = "9.9"
	_, err := api.OnboardNetworkSubnetsRecord(context.Background(), "pool", "net", option)
	if err != nil || len(calls) != 1 || api.RawClient().Microversion != "2.0" || api.RawClient().MoreHeaders["OpenStack-API-Version"] != "network 2.0" {
		t.Fatal(err, calls)
	}
	for _, opts := range [][]subnetpools.OnboardNetworkOption{{nil}, {subnetpools.WithOnboardNetworkHeaders(map[string]string{"X-Auth-Token": "foreign"})}, {func(*subnetpools.OnboardNetworkOpts) error { api.RawClient().Endpoint += "changed/"; return nil }}} {
		_, err = api.OnboardNetworkSubnetsRecord(context.Background(), "pool", "net", opts...)
		if err == nil || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	}
}
func TestPythonOnboardNetworkSubnetsPhysicalFaultsStopCallbacks(t *testing.T) {
	for _, code := range []int{200, 404, 503} {
		for _, stage := range []string{"read", "close", "cancel"} {
			t.Run(stage+http.StatusText(code), func(t *testing.T) {
				var calls []nativePoolCall
				cause := errors.New("physical onboard cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				physical := &poolRecordPhysicalBody{body: []byte(`{"actual":true}`)}
				if stage == "read" {
					physical.readErr = cause
				}
				if stage == "close" {
					physical.closeErr = cause
				}
				if stage == "cancel" {
					physical.onRead = func() { cancel(cause) }
				}
				api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { r := nativePoolWire(code, ""); r.Body = physical; return r })
				callbacks := 0
				api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					callbacks++
					return nil
				}
				_, err := api.OnboardNetworkSubnetsRecord(ctx, "pool", "net")
				if !errors.Is(err, cause) || len(calls) != 1 || callbacks != 0 || physical.closed != 1 {
					t.Fatal(err, calls, callbacks, physical)
				}
				if code < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != `{"actual":true}` {
						t.Fatal(err, proof)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{"actual":true}` {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}
func TestPythonOnboardNetworkSubnetsPreflight(t *testing.T) {
	var calls []nativePoolCall
	api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { t.Error("unexpected HTTP"); return nativePoolWire(200, `{}`) })
	for _, id := range []string{"", "..", "bad\n", "\xff"} {
		if _, err := api.OnboardNetworkSubnetsRecord(context.Background(), id, "net"); err == nil {
			t.Fatal(id)
		}
	}
	if _, err := api.OnboardNetworkSubnetsRecord(context.Background(), "pool", string([]byte{255})); err == nil {
		t.Fatal("invalid network UTF-8")
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("cancel before onboard")
	cancel(cause)
	if _, err := api.OnboardNetworkSubnetsRecord(ctx, "pool", "net"); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatal(strings.TrimSpace(calls[0].body))
	}
}
