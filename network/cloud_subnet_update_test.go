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

func TestCloudSubnetUpdateEffectiveValuesAndDirtySelection(t *testing.T) {
	for _, tc := range []struct {
		name, body, attrs, want string
		noOp                    bool
	}{
		{"same", `{"id":"p","name":"old","enable_dhcp":true}`, `{"subnet_name":"old","enable_dhcp":1}`, ``, true},
		{"changed-only", `{"id":"p","name":"old","enable_dhcp":true}`, `{"subnet_name":"new","enable_dhcp":true}`, `{"subnet":{"name":"new"}}`, false},
		{"dhcp-raw-string", `{"id":"p","enable_dhcp":true}`, `{"enable_dhcp":"false"}`, `{"subnet":{"enable_dhcp":"false"}}`, false},
		{"dhcp-false", `{"id":"p","enable_dhcp":true}`, `{"enable_dhcp":false}`, `{"subnet":{"enable_dhcp":false}}`, false},
		{"disable-gateway", `{"id":"p","gateway_ip":"10.0.0.1"}`, `{"disable_gateway_ip":true}`, `{"subnet":{"gateway_ip":null}}`, false},
		{"truthy-lists", `{"id":"p"}`, `{"dns_nameservers":["1.1.1.1"],"allocation_pools":[{"start":"1","end":"2"}],"host_routes":[{"destination":"d","nexthop":"n"}]}`, `{"subnet":{"dns_nameservers":["1.1.1.1"],"allocation_pools":[{"start":"1","end":"2"}],"host_routes":[{"destination":"d","nexthop":"n"}]}}`, false},
		{"raw-string-bool", `{"id":"p","enable_dhcp":"false"}`, `{"enable_dhcp":true}`, `{"subnet":{"enable_dhcp":true}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/subnets/p": {{200, `{"subnet":` + tc.body + `}`}}, "PUT /v2.0/subnets/p": {{200, `{"subnet":{"id":"p","name":"response"}}`}}})
			result, err := network.New(client).UpdateSubnetRecord(context.Background(), "p", network.WithSubnetUpdateJSON(json.RawMessage(tc.attrs)))
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
				if result.Subnet != nil {
					t.Fatal("fabricated PUT response")
				}
				return
			}
			var actual, expected any
			json.Unmarshal([]byte(got[1].body), &actual)
			json.Unmarshal([]byte(tc.want), &expected)
			if !reflect.DeepEqual(actual, expected) || result.Subnet == nil || result.Subnet.Name != "response" {
				t.Fatal(got, result)
			}
		})
	}
}
func TestCloudSubnetUpdateIneffectiveAndConflictBeforeServiceLookup(t *testing.T) {
	var service *network.Service
	for _, raw := range []string{`{}`, `{"subnet_name":"","enable_dhcp":null,"gateway_ip":"","disable_gateway_ip":false,"allocation_pools":[],"dns_nameservers":[],"host_routes":[]}`, `{"subnet_name":false,"gateway_ip":0,"allocation_pools":{},"dns_nameservers":null,"host_routes":false}`} {
		result, err := service.UpdateSubnetRecord(context.Background(), "", network.WithSubnetUpdateJSON(json.RawMessage(raw)))
		if err != nil || result == nil || !result.NoOp || result.Lookup != nil || result.Subnet != nil {
			t.Fatal(raw, result, err)
		}
	}
	result, err := service.UpdateSubnetRecord(context.Background(), "", network.WithSubnetUpdateJSON(json.RawMessage(`{"disable_gateway_ip":true,"gateway_ip":"10.0.0.1"}`)))
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
}
func TestCloudSubnetUpdateOptionCancellationCannotBecomeNoOp(t *testing.T) {
	var service *network.Service
	ctx, cancel := context.WithCancel(context.Background())
	result, err := service.UpdateSubnetRecord(ctx, "", func(*network.SubnetUpdateOpts) error { cancel(); return nil })
	if result != nil || !errors.Is(err, context.Canceled) {
		t.Fatal(result, err)
	}
}

func TestCloudSubnetUpdateNameFallbackAndMissing(t *testing.T) {
	for _, found := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "name"}[found], func(t *testing.T) {
			rows := `[]`
			if found {
				rows = `[{"id":"actual","name":"target"}]`
			}
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/subnets/target": {{404, `{}`}}, "GET /v2.0/subnets": {{200, `{"subnets":` + rows + `}`}}, "PUT /v2.0/subnets/actual": {{200, `{"subnet":{"id":"actual"}}`}}})
			result, err := network.New(client).UpdateSubnetRecord(context.Background(), "target", network.WithSubnetUpdateName("new"))
			if found {
				if err != nil || result.Subnet == nil || len(calls()) != 3 {
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
func TestCloudSubnetUpdateOwnedOptionsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`{"subnet_name":"owned"}`)
	option := network.WithSubnetUpdateJSON(raw)
	raw[2] = 'X'
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/subnets/p": {{200, `{"subnet":{"id":"p","name":"owned"}}`}}})
	service := network.New(client)
	result, err := service.UpdateSubnetRecord(context.Background(), "p", option)
	if err != nil || !result.NoOp || len(calls()) != 1 {
		t.Fatal(result, err, calls())
	}
	for _, opts := range [][]network.SubnetUpdateOption{{nil}, {network.WithSubnetUpdateAttribute("is_dhcp_enabled", true)}, {network.WithSubnetUpdateAttribute("if_revision", 0)}, {network.WithSubnetUpdateAttribute("subnet_name", make(chan int))}, {network.WithSubnetUpdateJSON(json.RawMessage(`[]`))}, {network.WithSubnetUpdateJSON(json.RawMessage(`{"x_extension":1}`))}} {
		result, err = service.UpdateSubnetRecord(context.Background(), "p", opts...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
			t.Fatal(result, err, calls())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = service.UpdateSubnetRecord(ctx, "p")
	if result != nil || !errors.Is(err, context.Canceled) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
	result, err = service.UpdateSubnetRecord(context.Background(), "p", network.WithSubnetUpdateName("new"), func(*network.SubnetUpdateOpts) error { service.API = nil; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
}
func TestCloudSubnetUpdateConflictKeepsLookupAndSingleAttempt(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/subnets/p": {{200, `{"subnet":{"id":"p","name":"old"}}`}}, "PUT /v2.0/subnets/p": {{http.StatusConflict, `{"error":"conflict"}`}}})
	result, err := network.New(client).UpdateSubnetRecord(context.Background(), "p", network.WithSubnetUpdateName("new"))
	if err == nil || result == nil || result.Lookup == nil || result.Subnet != nil || result.NoOp || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}
