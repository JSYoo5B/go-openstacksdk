package routers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCloudRouterCreateTranslatedBodySeedCanonicalAndSourceStatus(t *testing.T) {
	for _, status := range []int{199, 201, 204, 302, 399} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls []nativeRouterCall
			api := pythonCoreRouterAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.Method != http.MethodPost || r.URL.Path != "/neutron/v2.0/routers" || r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
					t.Error(r.Method, r.URL, r.Header)
				}
				w := nativeRouterWire(status, `{"router":{"id":null,"location":{"forged":true},"microversion":"foreign","tenant_id":"tenant","x_new":9007199254740993}}`)
				w.Header.Set("If-Match", "HTTP")
				return w
			})
			api.RawClient().Microversion = "2.1"
			attrs := map[string]json.RawMessage{"admin_state_up": json.RawMessage(`"False"`), "name": json.RawMessage(`"seed"`), "project_id": json.RawMessage(`""`)}
			result, e := api.CreateCloudRecord(context.Background(), attrs)
			if e != nil || result == nil || result.Resource == nil || result.Wire == nil || result.StatusCode != status || len(calls) != 1 {
				t.Fatal(result, e, calls)
			}
			for key, want := range map[string]string{"is_admin_state_up": "true", "name": `"seed"`, "project_id": `""`, "tenant_id": `"tenant"`, "id": "null", "tags": "[]", "if_match": `["HTTP"]`, "location": "null", "microversion": `"2.1"`, `x_new`: "9007199254740993"} {
				if string(result.Resource.Body[key]) != want {
					t.Fatal(key, string(result.Resource.Body[key]), want)
				}
			}
			if result.Resource.Body["admin_state_up"] != nil || result.Wire.Body["name"] != nil || string(result.Wire.Body["microversion"]) != `"foreign"` {
				t.Fatal(result)
			}
			var posted map[string]map[string]json.RawMessage
			if e := json.Unmarshal([]byte(calls[0].body), &posted); e != nil || string(posted["router"]["admin_state_up"]) != `"False"` || posted["router"]["is_admin_state_up"] != nil || len(posted["router"]) != 3 {
				t.Fatal(posted, e)
			}
		})
	}
}
func TestPythonCloudRouterCreatePreflightAndResponseShapeErrors(t *testing.T) {
	var calls []nativeRouterCall
	api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(201, `{"router":{}}`) })
	for _, attrs := range []map[string]json.RawMessage{nil, {"admin_state_up": json.RawMessage("true"), "is_admin_state_up": json.RawMessage("false")}, {"admin_state_up": json.RawMessage("true"), "name": json.RawMessage(`bad`)}, {"admin_state_up": json.RawMessage("true"), "tenant_id": nil}} {
		result, e := api.CreateCloudRecord(context.Background(), attrs)
		if result != nil || !errors.Is(e, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(result, e, calls)
		}
	}
	result, e := api.CreateCloudRecord(context.Background(), map[string]json.RawMessage{"admin_state_up": nil})
	if e != nil || result == nil || string(result.Resource.Body["is_admin_state_up"]) != "null" || len(calls) != 1 {
		t.Fatal(result, e, calls)
	}
	for _, body := range []string{"null", "[]", `{"router":null}`, `{"router":[]}`, `{"router":{"routes":null}}`} {
		calls = nil
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(202, body) })
		result, e := api.CreateCloudRecord(context.Background(), map[string]json.RawMessage{"admin_state_up": json.RawMessage("false")})
		success := body == `{"router":{"routes":null}}`
		if result == nil || result.StatusCode != 202 || string(result.Envelope) != body || len(calls) != 1 || success && (e != nil || result.Resource == nil) || !success && (e == nil || result.Resource != nil) {
			t.Fatal(body, result, e, calls)
		}
	}
}
func TestPythonCloudRouterCreateNativeRejectionsAndStickyRestoredSourceProof(t *testing.T) {
	for _, status := range []int{400, 403, 404, 409, 503} {
		var calls []nativeRouterCall
		body := `{"error":"rejected router"}`
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
			w := nativeRouterWire(status, body)
			w.Header.Set("X-Request-Id", "actual-router")
			return w
		})
		result, e := api.CreateCloudRecord(context.Background(), map[string]json.RawMessage{"admin_state_up": json.RawMessage("true")})
		var native gophercloud.ErrUnexpectedResponseCode
		if result == nil || result.Resource != nil || result.Wire != nil || result.StatusCode != status || string(result.Envelope) != body || result.Header.Get("X-Request-Id") != "actual-router" || !errors.As(e, &native) || native.Actual != status || native.Method != http.MethodPost || len(calls) != 1 {
			t.Fatal(result, e, native, calls)
		}
		nativeRouterOperation(t, e, "CreateCloudRecord")
	}
	var calls []nativeRouterCall
	body := &pythonRouterActionObservedBody{body: []byte(`{"router":{"id":"physical"}}`)}
	var api *routers.API
	api = pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { w := nativeRouterWire(203, ""); w.Body = body; return w })
	original := api.RawClient().ResourceBase
	body.onRead = func() { api.RawClient().ResourceBase = original + "changed/" }
	body.onClose = func() { api.RawClient().ResourceBase = original }
	result, e := api.CreateCloudRecord(context.Background(), map[string]json.RawMessage{"admin_state_up": json.RawMessage("true")})
	var proof *resource.ResponseError
	if result == nil || result.Resource != nil || result.StatusCode != 203 || !errors.Is(e, resource.ErrInvalidOption) || !errors.As(e, &proof) || string(proof.Body) != string(body.body) || body.closed != 1 || len(calls) != 1 || api.RawClient().ResourceBase != original {
		t.Fatal(result, e, proof, body.closed, calls)
	}
}
