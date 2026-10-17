package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCloudRouterUpdateEffectiveValuesAndDirtySelection(t *testing.T) {
	for _, tc := range []struct {
		name, body, attrs, want string
		noOp                    bool
	}{
		{"same", `{"id":"p","name":"old","admin_state_up":true}`, `{"name":"old","admin_state_up":1}`, ``, true},
		{"changed-only", `{"id":"p","name":"old","admin_state_up":true}`, `{"name":"new","admin_state_up":true}`, `{"router":{"name":"new"}}`, false},
		{"admin-false", `{"id":"p","admin_state_up":true}`, `{"admin_state_up":false}`, `{"router":{"admin_state_up":false}}`, false},
		{"snat-false", `{"id":"p","external_gateway_info":{"network_id":"old","enable_snat":true}}`, `{"enable_snat":false}`, `{"router":{"external_gateway_info":{"enable_snat":false}}}`, false},
		{"gateway", `{"id":"p"}`, `{"ext_gateway_net_id":"external","ext_fixed_ips":[{"subnet_id":"s"}]}`, `{"router":{"external_gateway_info":{"network_id":"external","external_fixed_ips":[{"subnet_id":"s"}]}}}`, false},
		{"raw-string-bool", `{"id":"p","admin_state_up":"false"}`, `{"admin_state_up":true}`, `{"router":{"admin_state_up":true}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/routers/p": {{200, `{"router":` + tc.body + `}`}}, "PUT /v2.0/routers/p": {{200, `{"router":{"id":"p","name":"response"}}`}}})
			result, err := network.New(client).UpdateRouterRecord(context.Background(), "p", network.WithRouterUpdateJSON(json.RawMessage(tc.attrs)))
			if err != nil || result == nil || result.Lookup == nil || result.NoOp != tc.noOp {
				t.Fatal(result, err)
			}
			got := calls()
			wantCount := 2
			if tc.noOp {
				wantCount = 1
			}
			if len(got) != wantCount {
				t.Fatal(got)
			}
			if tc.noOp {
				if result.Router != nil {
					t.Fatal("fabricated PUT response")
				}
				return
			}
			var actual, expected any
			json.Unmarshal([]byte(got[1].body), &actual)
			json.Unmarshal([]byte(tc.want), &expected)
			if !reflect.DeepEqual(actual, expected) || result.Router == nil || result.Router.Name != "response" {
				t.Fatal(got, result)
			}
		})
	}
}
func TestCloudRouterUpdateIneffectiveBeforeServiceLookup(t *testing.T) {
	var service *network.Service
	for _, raw := range []string{`{}`, `{"name":"","admin_state_up":null,"ext_gateway_net_id":"","enable_snat":null,"ext_fixed_ips":[],"routes":null}`, `{"name":false,"ext_gateway_net_id":0,"ext_fixed_ips":{}}`} {
		result, err := service.UpdateRouterRecord(context.Background(), "", network.WithRouterUpdateJSON(json.RawMessage(raw)))
		if err != nil || result == nil || !result.NoOp || result.Lookup != nil || result.Router != nil {
			t.Fatal(raw, result, err)
		}
	}
}
func TestCloudRouterUpdateRoutesGateAndDrop(t *testing.T) {
	for _, tc := range []struct {
		name, attrs       string
		available, lookup bool
	}{
		{"clear", `{"routes":[]}`, true, true},
		{"false", `{"routes":false}`, true, true},
		{"drop-only", `{"routes":[]}`, false, false},
		{"drop-and-name", `{"routes":[],"name":"new"}`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extensions := `[]`
			if tc.available {
				extensions = `[{"alias":"extraroute"}]`
			}
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/extensions": {{200, `{"extensions":` + extensions + `}`}}, "GET /v2.0/routers/p": {{200, `{"router":{"id":"p","name":"old","routes":[{"destination":"d","nexthop":"n"}]}}`}}, "PUT /v2.0/routers/p": {{200, `{"router":{"id":"p"}}`}}})
			result, err := network.New(client).UpdateRouterRecord(context.Background(), "p", network.WithRouterUpdateJSON(json.RawMessage(tc.attrs)))
			if err != nil || result == nil || result.RoutesDropped == tc.available {
				t.Fatal(result, err)
			}
			want := 1
			if tc.lookup {
				want = 3
			}
			if len(calls()) != want {
				t.Fatal(calls())
			}
			if !tc.lookup {
				if !result.NoOp || result.Lookup != nil || result.Router != nil {
					t.Fatal(result)
				}
				return
			}
			var body struct{ Router map[string]json.RawMessage }
			json.Unmarshal([]byte(calls()[2].body), &body)
			routes, present := body.Router["routes"]
			if present != tc.available {
				t.Fatal(string(calls()[2].body))
			}
			if tc.available {
				var attrs map[string]json.RawMessage
				json.Unmarshal([]byte(tc.attrs), &attrs)
				if string(routes) != string(attrs["routes"]) {
					t.Fatal(string(routes))
				}
			}
		})
	}
}
func TestCloudRouterUpdateOptionCancellationCannotBecomeNoOp(t *testing.T) {
	var service *network.Service
	ctx, cancel := context.WithCancel(context.Background())
	result, err := service.UpdateRouterRecord(ctx, "", func(*network.RouterUpdateOpts) error { cancel(); return nil })
	if result != nil || !errors.Is(err, context.Canceled) {
		t.Fatal(result, err)
	}
}
func TestCloudRouterUpdateNameFallbackAndMissing(t *testing.T) {
	for _, found := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "name"}[found], func(t *testing.T) {
			rows := `[]`
			if found {
				rows = `[{"id":"actual","name":"target"}]`
			}
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/routers/target": {{404, `{}`}}, "GET /v2.0/routers": {{200, `{"routers":` + rows + `}`}}, "PUT /v2.0/routers/actual": {{200, `{"router":{"id":"actual"}}`}}})
			result, err := network.New(client).UpdateRouterRecord(context.Background(), "target", network.WithRouterUpdateName("new"))
			if found {
				if err != nil || result.Router == nil || len(calls()) != 3 {
					t.Fatal(result, err, calls())
				}
			} else {
				if !errors.Is(err, resource.ErrNotFound) || result.Lookup != nil || len(calls()) != 2 {
					t.Fatal(result, err, calls())
				}
			}
		})
	}
}
func TestCloudRouterUpdateOwnedOptionsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`{"name":"owned"}`)
	option := network.WithRouterUpdateJSON(raw)
	raw[2] = 'X'
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/routers/p": {{200, `{"router":{"id":"p","name":"owned"}}`}}})
	service := network.New(client)
	result, err := service.UpdateRouterRecord(context.Background(), "p", option)
	if err != nil || !result.NoOp || len(calls()) != 1 {
		t.Fatal(result, err, calls())
	}
	for _, opts := range [][]network.RouterUpdateOption{{nil}, {network.WithRouterUpdateAttribute("is_admin_state_up", true)}, {network.WithRouterUpdateAttribute("if_revision", 0)}, {network.WithRouterUpdateAttribute("name", make(chan int))}, {network.WithRouterUpdateJSON(json.RawMessage(`[]`))}, {network.WithRouterUpdateJSON(json.RawMessage(`{"x_extension":1}`))}} {
		result, err = service.UpdateRouterRecord(context.Background(), "p", opts...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
			t.Fatal(result, err, calls())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = service.UpdateRouterRecord(ctx, "p")
	if result != nil || !errors.Is(err, context.Canceled) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
	result, err = service.UpdateRouterRecord(context.Background(), "p", network.WithRouterUpdateName("new"), func(*network.RouterUpdateOpts) error { service.API = nil; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
}
func TestCloudRouterUpdateConflictKeepsLookupAndSingleAttempt(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/routers/p": {{200, `{"router":{"id":"p","name":"old"}}`}}, "PUT /v2.0/routers/p": {{http.StatusConflict, `{"error":"conflict"}`}}})
	result, err := network.New(client).UpdateRouterRecord(context.Background(), "p", network.WithRouterUpdateName("new"))
	if err == nil || result == nil || result.Lookup == nil || result.Router != nil || result.NoOp || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}

func TestCloudRouterUpdateExtensionFailureCannotBecomeDroppedRoutes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		replies []pythonCloudBReply
		count   int
	}{
		{"http", []pythonCloudBReply{{500, `{"error":"extension failure"}`}}, 1},
		{"late-invalid-alias", []pythonCloudBReply{{200, `{"extensions":[{"alias":"extraroute"},{"alias":[]}]}`}}, 1},
		{"later-page", []pythonCloudBReply{{200, `{"extensions":[{"alias":"extraroute"}],"next":"{{base}}/v2.0/extensions?marker=tail"}`}, {500, `{"error":"tail failure"}`}}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/extensions": tc.replies})
			result, err := network.New(client).UpdateRouterRecord(context.Background(), "p", network.WithRouterUpdateJSON(json.RawMessage(`{"routes":[],"name":"new"}`)))
			if err == nil || result == nil || result.RoutesDropped || result.NoOp || result.Lookup != nil || result.Router != nil || len(calls()) != tc.count {
				t.Fatal(result, err, calls())
			}
		})
	}
}
