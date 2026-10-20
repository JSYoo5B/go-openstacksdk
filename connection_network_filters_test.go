package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionNetworkFiltersSharedFacadesAndRawDescriptors(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "network" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Error("catalog selection", options)
		}
		return cloud.Server.URL + "/catalog/neutron/v2.0/", nil
	}
	const path = "/reverse/neutron/v2.0/networks"
	cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"limit": {"2"}, "marker": {"first", "second"}, "fields": {"id", "subnets"}, "id": {"wire-id"}, "name": {"server-name"}, "status": {"wire-status"}, "project_id": {"wire-owner"}, "shared": {"false"}, "admin_state_up": {"false"}, "tags-any": {"canonical-tag"}, "ipv4_address_scope": {"wire-scope"}, "router:external": {"true"}, "provider:segmentation_id": {"100", "101"}}
		if r.URL.Query().Has("page") {
			want["page"] = []string{"2"}
		}
		if !reflect.DeepEqual(r.URL.Query(), want) || r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("Accept") != "application/json" {
			t.Error("local fields leaked or query aliases changed", r.URL, r.Header)
		}
		if !r.URL.Query().Has("page") {
			if r.Header.Get("X-Auth-Token") != "first-token" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("second-token")
			next, _ := json.Marshal(cloud.Server.URL + path + "?" + want.Encode() + "&page=2")
			testcloud.JSON(w, 200, `{"networks":[{"id":"other","subnets":[]}],"networks_links":[{"rel":"next","href":`+string(next)+`}]}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "second-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"selected","name":"different-returned-name","status":"ACTIVE","project_id":"native-owner","tenant_id":"native-tenant","shared":true,"admin_state_up":true,"subnets":["subnet-a",null],"availability_zone_hints":["az-a",null],"availability_zones":["az-b",{"n":9007199254740993}],"segments":[{"n":9007199254740993,"nullable":null}],"mtu":" +009007199254740993 ","revision_number":9007199254740993,"qos_policy_id":{"n":9007199254740993,"extra":null},"dns_domain":"example.","is_default":"false","vlan_qinq":-0.0e999999999999999999999,"vlan_transparent":[],"pvlan":{"empty":null},"created_at":"2026-10-02T01:02:03","updated_at":"2026-10-02T02:03:04"}]}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("region"), sdk.WithInterface(gophercloud.AvailabilityInternal))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Network(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	again, err := conn.Network(context.Background())
	if err != nil || again != service {
		t.Fatal("network facade is not cached", again, err)
	}
	v2, err := conn.NetworkV2(context.Background())
	if err != nil || v2.RawClient() != service.RawClient() || v2.Networks.RawClient() != service.API.Networks.RawClient() || lookups.Load() != 1 {
		t.Fatal("both facades must share the cached client", err, lookups.Load())
	}
	client := service.RawClient()
	client.ResourceBase = cloud.Server.URL + "/reverse/neutron/v2.0/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	filters := resource.WithFilters(map[string]any{
		"limit": 2, "marker": []string{"first", "second"}, "fields": []string{"id", "subnets"}, "id": "wire-id", "name": "server-name", "status": "wire-status", "project_id": "wire-owner", "is_shared": false, "is_admin_state_up": false, "any_tags": "canonical-tag", "tags-any": "ignored", "ipv4_address_scope_id": "wire-scope", "is_router_external": true, "provider_segmentation_id": []int{100, 101},
		"availability_zone_hints": json.RawMessage(`["az-a",null]`), "availability_zones": json.RawMessage(`["az-b",{"n":9007199254740993}]`), "created_at": "2026-10-02T01:02:03", "updated_at": "2026-10-02T02:03:04", "dns_domain": "example.", "qos_policy_id": json.RawMessage(`{"n":9007199254740993}`), "segments": json.RawMessage(`[{"n":9007199254740993,"nullable":null}]`), "subnet_ids": json.RawMessage(`["subnet-a",null]`), "mtu": json.Number("9007199254740993"), "revision_number": json.Number("9007199254740993"), "is_default": true, "is_vlan_qinq": false, "is_vlan_transparent": false, "pvlan": true,
		"subnets": make(chan int), "vlan_qinq": make(chan int), "tenant_id": make(chan int),
	})
	for _, collection := range []*resource.Collection[network.Network]{service.Networks, v2.Networks.Resources} {
		cloud.Provider.SetToken("first-token")
		values, err := collection.All(context.Background(), filters)
		if err != nil || len(values) != 1 {
			t.Fatal("semantic query/body classification changed", values, err)
		}
		value := values[0]
		if value.ID != "selected" || value.Name != "different-returned-name" || value.Status != "ACTIVE" || value.ProjectID != "native-owner" || value.TenantID != "native-tenant" || !value.Shared || !value.AdminStateUp || value.RevisionNumber != 9007199254740993 || !reflect.DeepEqual(value.Subnets, []string{"subnet-a", ""}) || !reflect.DeepEqual(value.AvailabilityZoneHints, []string{"az-a", ""}) || !value.CreatedAt.Equal(time.Date(2026, 10, 2, 1, 2, 3, 0, time.UTC)) {
			t.Fatal("local raw matching changed native return decoding", value)
		}
	}
	if calls.Load() != 4 || lookups.Load() != 1 || client.ProviderClient != cloud.Provider || len(client.MoreHeaders) != 1 {
		t.Fatal("continuation changed the shared client", calls.Load(), lookups.Load(), client.MoreHeaders)
	}
}

func TestConnectionNetworkFiltersPreflightAndLocalStatus(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /neutron/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"name": {"selected-name"}, "status": {"ACTIVE"}, "tenant_id": {"wire-extension"}, "shared": {"false"}}
		if !reflect.DeepEqual(r.URL.Query(), want) {
			t.Error("raw status or native name hint changed", r.URL)
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"name-miss","name":"other","status":"active","subnets":["subnet-a"]},{"id":"status-miss","name":"selected-name","status":"DOWN","subnets":["subnet-a"]},{"id":"selected","name":"selected-name","status":"aCtIvE","shared":true,"subnets":["subnet-a"]}]}`)
	})
	cloud.Mux.HandleFunc("GET /neutron/v2.0/networks/unfiltered", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if len(r.URL.Query()) != 0 {
			t.Error("list options leaked into Get", r.URL)
		}
		testcloud.JSON(w, 200, `{"network":{"id":"unfiltered","status":"DOWN"}}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/neutron/v2.0/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Network(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, collection := range []*resource.Collection[network.Network]{service.Networks, service.API.Networks.Resources} {
		before := calls.Load()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if values, err := collection.All(ctx, resource.WithFilter("subnet_ids", []string{"subnet-a"})); len(values) != 0 || !errors.Is(err, context.Canceled) || calls.Load() != before {
			t.Fatal("canceled list started HTTP", values, err, calls.Load())
		}
		for _, options := range [][]resource.ListOption{
			{resource.WithFilter("status", "ACTIVE"), resource.WithStatus("ACTIVE")},
			{resource.WithFilter("status", nil), resource.WithQuery("status", "")},
			{resource.WithFilter("is_shared", nil), resource.WithQuery("shared", "false")},
			{resource.WithFilter("any_tags", []string{}), resource.WithQuery("tags-any", "tag")},
			{resource.WithFilter("name", "selected-name"), resource.WithName("selected-name")},
			{resource.WithFilter("subnet_ids", []string{"subnet-a"}), resource.WithBodyFilter("subnets", []string{"subnet-a"})},
			{resource.WithFilter("is_vlan_qinq", false), resource.WithBodyFilter("vlan_qinq", false)},
			{resource.WithFilter("limit", nil), resource.WithPageSize(2)},
			{resource.WithFilter("mtu", make(chan int))},
		} {
			if values, err := collection.All(context.Background(), options...); len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != before {
				t.Fatal("preflight missed a target collision", values, err, calls.Load())
			}
		}
		values, err := collection.All(context.Background(), resource.WithFilters(map[string]any{"is_shared": false, "shared": true, "tenant_id": make(chan int), "subnet_ids": []string{"subnet-a"}}), resource.WithName("selected-name"), resource.WithStatus("earlier-status"), resource.WithQuery("status", "ACTIVE"), resource.WithQuery("tenant_id", "wire-extension"))
		if err != nil || len(values) != 1 || values[0].ID != "selected" || !values[0].Shared || calls.Load() != before+1 {
			t.Fatal("name/status local AND or last raw status changed", values, err, calls.Load())
		}
		value, err := collection.Get(context.Background(), "unfiltered")
		if err != nil || value.ID != "unfiltered" {
			t.Fatal("list filters escaped into Get", value, err)
		}
	}
	if gets.Load() != 2 {
		t.Fatal(gets.Load())
	}
}

func TestConnectionNetworkFiltersAdapterOwnershipAndWaitPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, manualPolls, generatedPolls atomic.Int32
	cloud.Mux.HandleFunc("GET /neutron/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if !reflect.DeepEqual(r.URL.Query(), url.Values{"shared": {"false"}}) {
			t.Error("mutating a separate adapter changed a collection", r.URL)
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"selected","subnets":["subnet-a"],"vlan_qinq":false}]}`)
	})
	cloud.Mux.HandleFunc("GET /neutron/v2.0/networks/manual", func(w http.ResponseWriter, r *http.Request) {
		status := "ERROR_CUSTOM"
		if manualPolls.Add(1) > 1 {
			status = "ACTIVE"
		}
		testcloud.JSON(w, 200, `{"network":{"id":"manual","status":"`+status+`"}}`)
	})
	cloud.Mux.HandleFunc("GET /neutron/v2.0/networks/generated", func(w http.ResponseWriter, r *http.Request) {
		generatedPolls.Add(1)
		testcloud.JSON(w, 200, `{"network":{"id":"generated","status":"ERROR_CUSTOM"}}`)
	})
	cloud.Mux.HandleFunc("GET /neutron/v2.0/networks/failed", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"network":{"id":"failed","status":"eRrOr"}}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/neutron/v2.0/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Network(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	adapter := service.API.Networks.ResourceAdapter()
	owned := resource.NewCollection(adapter)
	delete(adapter.BodyFilterFields, "subnet_ids")
	adapter.BodyFilterFields["vlan_qinq"] = "subnets"
	adapter.FilterDescriptor.Query["is_shared"] = "corrupted"
	delete(adapter.FilterDescriptor.Body, "subnet_ids")
	adapter.FilterDescriptor.Reserved[0] = "corrupted"
	fresh := service.API.Networks.ResourceAdapter()
	for _, collection := range []*resource.Collection[network.Network]{service.Networks, service.API.Networks.Resources, owned, resource.NewCollection(fresh)} {
		values, err := collection.All(context.Background(), resource.WithFilter("is_shared", false), resource.WithFilter("subnet_ids", []string{"subnet-a"}), resource.WithBodyFilter("is_vlan_qinq", false))
		if err != nil || len(values) != 1 || values[0].ID != "selected" {
			t.Fatal("adapter metadata ownership escaped", values, err)
		}
	}
	if lists.Load() != 4 || fresh.FilterDescriptor.Reserved[0] == "corrupted" {
		t.Fatal("adapter defaults were shared", lists.Load(), fresh.FilterDescriptor)
	}
	value, err := service.Networks.Wait(context.Background(), resource.ID("manual"), "active", resource.WithTimeout(time.Second), resource.WithPollInterval(time.Millisecond))
	if err != nil || value.Status != "ACTIVE" || manualPolls.Load() != 2 {
		t.Fatal("manual facade lost its exact ERROR waiter", value, err, manualPolls.Load())
	}
	value, err = service.API.Networks.Resources.Wait(context.Background(), resource.ID("generated"), "ACTIVE", resource.WithPollInterval(time.Millisecond))
	var failed *resource.FailedStateError
	if value != nil || !errors.As(err, &failed) || failed.Resource != "networks" || failed.Status != "ERROR_CUSTOM" || generatedPolls.Load() != 1 {
		t.Fatal("generated waiter policy changed", value, err, generatedPolls.Load())
	}
	value, err = service.Networks.Wait(context.Background(), resource.ID("failed"), "ACTIVE", resource.WithPollInterval(time.Millisecond))
	if value != nil || !errors.As(err, &failed) || failed.Resource != "network" || failed.Status != "eRrOr" {
		t.Fatal("manual waiter kind or case folding changed", value, err)
	}
	for i, collection := range []*resource.Collection[network.Network]{service.Networks, service.API.Networks.Resources} {
		_, err := collection.Get(context.Background(), "missing")
		var missing *resource.NotFoundError
		want := []string{"network", "networks"}[i]
		if !errors.As(err, &missing) || missing.Resource != want {
			t.Fatal("facade error kind changed", want, err)
		}
	}
}
