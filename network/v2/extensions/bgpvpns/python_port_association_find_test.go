package bgpvpns_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const portAssociationCollection = "/neutron/v2.0/bgpvpn/bgpvpns/vpn/port_associations"

func TestPythonPortAssociationFindMemberProjectionAndOwnership(t *testing.T) {
	const raw = `{"port_association":{"id":"reply","name":"wire","bgpvpn_id":"cannot-reroute","project_id":null,"tenant_id":"tenant","routes":{"type":"prefix"},"advertise_fixed_ips":0,"x_extension":9007199254740993}}`
	var calls []nativeBGPVPNCall
	api, _ := nativeBGPVPNAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.EscapedPath() != "/neutron/v2.0/bgpvpn/bgpvpns/vpn%2Fparent/port_associations/lookup%2Fid" || r.URL.Query().Get("port_id") != "port" || r.URL.Query().Get("x_seed") != "kept" || r.URL.Query().Has("bgpvpn_id") {
			t.Error(r.URL)
		}
		return nativeBGPVPNWire(203, raw)
	})
	value, err := api.FindPortAssociationRecord(context.Background(), "vpn/parent", "lookup/id", sdk.WithPortAssociationFindQuery("port_id", "port"), sdk.WithPortAssociationFindQuery("x_seed", "kept"))
	if err != nil || value == nil || len(calls) != 1 || value.StatusCode != 203 || string(value.Envelope) != raw {
		t.Fatal(value, err, calls)
	}
	for key, want := range map[string]string{"id": `"reply"`, "project_id": `null`, "tenant_id": `"tenant"`, "routes": `[{"type":"prefix"}]`, "advertise_fixed_ips": `0`, "x_extension": `9007199254740993`, "x_seed": `"kept"`} {
		if string(value.Resource.Body[key]) != want {
			t.Fatal(key, value.Resource.Body)
		}
	}
	if value.Resource.Body["bgpvpn_id"] != nil || value.Resource.Body["tags"] != nil || value.Wire.Body["x_seed"] != nil || string(value.Wire.Body["bgpvpn_id"]) != `"cannot-reroute"` {
		t.Fatal(value)
	}
	own := value.Clone()
	own.Resource.Body["name"][1] = 'X'
	own.Wire.Body["name"][1] = 'Y'
	own.Envelope[0] = 'X'
	if string(value.Resource.Body["name"]) != `"wire"` || string(value.Wire.Body["name"]) != `"wire"` || string(value.Envelope) != raw {
		t.Fatal("alias")
	}
}
func TestPythonPortAssociationFindFallbackLocalFiltersAndLateDuplicates(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		t.Run(map[bool]string{false: "one", true: "duplicate"}[duplicate], func(t *testing.T) {
			var calls []nativeBGPVPNCall
			api, _ := nativeBGPVPNAPI(t, &calls, func(r *http.Request) *http.Response {
				if strings.HasSuffix(r.URL.Path, "/wanted") {
					return nativeBGPVPNWire(404, `{}`)
				}
				if r.URL.Path != portAssociationCollection || r.URL.Query().Has("name") || r.URL.Query().Has("port_id") || r.URL.Query().Has("x_seed") {
					t.Error(r.URL)
				}
				if r.URL.Query().Get("marker") == "tail" {
					if duplicate {
						return nativeBGPVPNWire(200, `{"port_associations":[{"id":"one","name":"wanted","port_id":"port"}]}`)
					}
					return nativeBGPVPNWire(200, `{"port_associations":[]}`)
				}
				return nativeBGPVPNWire(200, `{"port_associations":[{"id":"skip","name":"wanted","port_id":"other"},{"id":"one","name":"wanted","port_id":"port"}],"port_associations_links":[{"rel":"next","href":"?marker=tail"}]}`)
			})
			value, err := api.FindPortAssociationRecord(context.Background(), "vpn", "wanted", sdk.WithPortAssociationFindQuery("port_id", "port"), sdk.WithPortAssociationFindQuery("x_seed", "get-only"))
			if len(calls) != 3 {
				t.Fatal(value, err, calls)
			}
			if duplicate {
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(value, err)
				}
				return
			}
			if err != nil || value == nil || string(value.Resource.Body["id"]) != `"one"` || value.Resource.Body["x_seed"] != nil || string(value.Resource.Body["routes"]) != `null` || string(value.Resource.Body["advertise_fixed_ips"]) != `null` {
				t.Fatal(value, err)
			}
		})
	}
}
func TestPythonPortAssociationFindMissingAndStrictJSONIdentity(t *testing.T) {
	for _, strict := range []bool{false, true} {
		var calls []nativeBGPVPNCall
		api, _ := nativeBGPVPNAPI(t, &calls, func(r *http.Request) *http.Response {
			if strings.HasSuffix(r.URL.Path, "/7") {
				return nativeBGPVPNWire(403, `{}`)
			}
			return nativeBGPVPNWire(200, `{"port_associations":[{"id":7,"name":"other"}]}`)
		})
		value, err := api.FindPortAssociationRecord(context.Background(), "vpn", "7", sdk.WithPortAssociationFindIgnoreMissing(!strict))
		if value != nil || len(calls) != 2 || strict && !errors.Is(err, resource.ErrNotFound) || !strict && err != nil {
			t.Fatal(value, err, calls)
		}
	}
}
func TestPythonPortAssociationFindRawFetchControlsAndHeadersPhase(t *testing.T) {
	var calls []nativeBGPVPNCall
	api, _ := nativeBGPVPNAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.Path == portAssociationCollection+"/lookup" {
			if r.Header.Get("X-Raw") != "" || r.URL.Query().Get("headers") != "X-Raw" {
				t.Error(r.URL, r.Header)
			}
			return nativeBGPVPNWire(400, `{}`)
		}
		if r.Header.Get("X-Raw") != "fallback" || r.URL.Query().Has("headers") {
			t.Error(r.URL, r.Header)
		}
		return nativeBGPVPNWire(200, `{"port_associations":[{"id":"found","name":"lookup"}]}`)
	})
	value, err := api.FindPortAssociationRecord(context.Background(), "vpn", "lookup", sdk.WithPortAssociationFindQuery("headers", map[string]string{"X-Raw": "fallback"}))
	if err != nil || value == nil || len(calls) != 2 {
		t.Fatal(value, err, calls)
	}
	api, _ = nativeBGPVPNAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.Path != portAssociationCollection {
			t.Error(r.URL)
		}
		return nativeBGPVPNWire(200, `{"custom":{"id":"collection","routes":false}}`)
	})
	value, err = api.FindPortAssociationRecord(context.Background(), "vpn", "lookup", sdk.WithPortAssociationFindQuery("requires_id", false), sdk.WithPortAssociationFindQuery("resource_response_key", "custom"), sdk.WithPortAssociationFindQuery("skip_cache", true))
	if err != nil || value == nil || string(value.Resource.Body["id"]) != `"collection"` || string(value.Resource.Body["routes"]) != `[false]` {
		t.Fatal(value, err)
	}
}
func TestPythonPortAssociationFindOptionsGuardAndPreflight(t *testing.T) {
	var calls []nativeBGPVPNCall
	api, _ := nativeBGPVPNAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Owned") != "original" || r.URL.Query().Get("port_id") != "port" {
			t.Error(r.Header, r.URL)
		}
		return nativeBGPVPNWire(200, `{"port_association":{}}`)
	})
	headers := map[string]string{"X-Owned": "original"}
	query := map[string]json.RawMessage{"port_id": json.RawMessage(`"port"`)}
	own := sdk.WithPortAssociationFindOpts(sdk.PortAssociationFindOpts{Headers: headers, Query: query})
	headers["X-Owned"] = "changed"
	query["port_id"][1] = 'X'
	if _, err := api.FindPortAssociationRecord(context.Background(), "vpn", "lookup", own); err != nil || len(calls) != 1 {
		t.Fatal(err, calls)
	}
	for _, opts := range [][]sdk.PortAssociationFindOption{{nil}, {sdk.WithPortAssociationFindQuery("bgpvpn_id", "foreign")}, {sdk.WithPortAssociationFindOpts(sdk.PortAssociationFindOpts{Query: map[string]json.RawMessage{"id": json.RawMessage(`"bad"`)}})}, {sdk.WithPortAssociationFindHeaders(map[string]string{"X-Auth-Token": "foreign"})}, {func(*sdk.PortAssociationFindOpts) error { api.RawClient().Endpoint += "changed/"; return nil }}} {
		if _, err := api.FindPortAssociationRecord(context.Background(), "vpn", "lookup", opts...); err == nil || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	}
}

type portAssociationFaultBody struct {
	io.Reader
	cause  error
	closed int
}

func (b *portAssociationFaultBody) Read(p []byte) (int, error) {
	n, _ := b.Reader.Read(p)
	return n, b.cause
}
func (b *portAssociationFaultBody) Close() error { b.closed++; return nil }
func TestPythonPortAssociationFindPhysicalFaultNeverFallback(t *testing.T) {
	for _, code := range []int{200, 404, 503} {
		var calls []nativeBGPVPNCall
		cause := errors.New("association read failed")
		body := &portAssociationFaultBody{Reader: strings.NewReader(`{"port_association":{}}`), cause: cause}
		api, _ := nativeBGPVPNAPI(t, &calls, func(*http.Request) *http.Response { r := nativeBGPVPNWire(code, ""); r.Body = body; return r })
		hooks := 0
		api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nil
		}
		_, err := api.FindPortAssociationRecord(context.Background(), "vpn", "lookup")
		if !errors.Is(err, cause) || len(calls) != 1 || hooks != 0 || body.closed != 1 {
			t.Fatal(err, calls, hooks, body)
		}
		if code < 400 {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != `{"port_association":{}}` {
				t.Fatal(err, proof)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{"port_association":{}}` {
				t.Fatal(err, native)
			}
		}
	}
}
