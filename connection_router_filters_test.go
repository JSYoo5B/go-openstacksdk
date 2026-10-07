package gophercloudsdk_test

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

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionRouterFiltersSharedClientAndRawRevision(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "network" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Error("catalog selection", options)
		}
		return cloud.Server.URL + "/catalog/neutron/v2.0/", nil
	}
	const path = "/reverse/neutron/v2.0/routers"
	cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"limit": {"2"}, "marker": {"first", "second"}, "fields": {"id", "routes"}, "id": {"wire-id"}, "name": {"server-name"}, "status": {"wire-status"}, "project_id": {"wire-owner"}, "distributed": {"false"}, "admin_state_up": {"false"}, "ha": {"true"}, "flavor_id": {"wire-flavor"}, "tags-any": {"canonical-tag"}}
		if r.URL.Query().Has("page") {
			want["page"] = []string{"2"}
		}
		if !reflect.DeepEqual(r.URL.Query(), want) || r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("Accept") != "application/json" {
			t.Error("local conditions leaked into the query", r.URL, r.Header)
		}
		if !r.URL.Query().Has("page") {
			if r.Header.Get("X-Auth-Token") != "first-token" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("second-token")
			next, _ := json.Marshal(cloud.Server.URL + path + "?" + want.Encode() + "&page=2")
			testcloud.JSON(w, 200, `{"routers":[{"id":"other","revision":"9007199254740992"}],"routers_links":[{"rel":"next","href":`+string(next)+`}]}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "second-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"routers":[{"id":"selected","name":"different-returned-name","status":"ACTIVE","project_id":"native-owner","tenant_id":null,"distributed":true,"admin_state_up":true,"revision_number":7,"revision":" +009007199254740993 ","evpn_vni":"+9007199254740993","enable_ndp_proxy":"false","availability_zone_hints":["az-a",null],"availability_zones":[{"n":9007199254740993}],"external_gateway_info":{"network_id":"external","enable_snat":false,"quota":9007199254740993,"external_fixed_ips":[{"ip_address":"192.0.2.8","subnet_id":"outside","extra":{"n":9007199254740993}},null]},"routes":[{"destination":"198.51.100.0/24","nexthop":"192.0.2.1","weight":9007199254740993,"nullable":null},null],"created_at":"2026-10-02T01:02:03","updated_at":"2026-10-02T02:03:04"}]}`)
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
		t.Fatal("network facade was not cached", again, err)
	}
	v2, err := conn.NetworkV2(context.Background())
	if err != nil || v2.RawClient() != service.RawClient() || v2.Routers.RawClient() != service.API.Routers.RawClient() || lookups.Load() != 1 {
		t.Fatal("router collection must share the cached client", err, lookups.Load())
	}
	client := service.RawClient()
	client.ResourceBase = cloud.Server.URL + "/reverse/neutron/v2.0/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	option := resource.WithFilters(map[string]any{
		"limit": 2, "marker": []string{"first", "second"}, "fields": []string{"id", "routes"}, "id": "wire-id", "name": "server-name", "status": "wire-status", "project_id": "wire-owner", "is_distributed": false, "is_admin_state_up": false, "is_ha": true, "flavor_id": "wire-flavor", "any_tags": "canonical-tag", "tags-any": "ignored-alias",
		"revision_number": json.Number("9007199254740993"), "evpn_vni": json.Number("9007199254740993"), "enable_ndp_proxy": true, "tenant_id": nil, "availability_zone_hints": json.RawMessage(`["az-a",null]`), "availability_zones": json.RawMessage(`[{"n":9007199254740993}]`), "created_at": "2026-10-02T01:02:03", "updated_at": "2026-10-02T02:03:04",
		"external_gateway_info": json.RawMessage(`{"quota":9007199254740993,"external_fixed_ips":[{"ip_address":"192.0.2.8","subnet_id":"outside","extra":{"n":9007199254740993}},null]}`), "routes": json.RawMessage(`[{"destination":"198.51.100.0/24","nexthop":"192.0.2.1","weight":9007199254740993,"nullable":null},null]`), "revision": make(chan int),
	})
	for _, collection := range []*resource.Collection[routers.Router]{service.API.Routers.Resources, v2.Routers.Resources} {
		cloud.Provider.SetToken("first-token")
		values, err := collection.All(context.Background(), option)
		if err != nil || len(values) != 1 {
			t.Fatal("query classification or local raw descriptors changed", values, err)
		}
		value := values[0]
		if value.ID != "selected" || value.Name != "different-returned-name" || value.Status != "ACTIVE" || value.ProjectID != "native-owner" || value.TenantID != "" || !value.Distributed || !value.AdminStateUp || value.RevisionNumber != 7 || !reflect.DeepEqual(value.AvailabilityZoneHints, []string{"az-a", ""}) || len(value.Routes) != 2 || value.Routes[1] != (routers.Route{}) || value.GatewayInfo.NetworkID != "external" || value.GatewayInfo.EnableSNAT == nil || *value.GatewayInfo.EnableSNAT || len(value.GatewayInfo.ExternalFixedIPs) != 2 || value.GatewayInfo.ExternalFixedIPs[1] != (routers.ExternalFixedIP{}) || !value.CreatedAt.Equal(time.Date(2026, 10, 2, 1, 2, 3, 0, time.UTC)) {
			t.Fatal("local revision/raw nested matching changed the native model", value)
		}
	}
	if calls.Load() != 4 || lookups.Load() != 1 || client.ProviderClient != cloud.Provider || len(client.MoreHeaders) != 1 {
		t.Fatal("continuation changed shared configuration", calls.Load(), lookups.Load(), client.MoreHeaders)
	}
}

func TestConnectionRouterFiltersPreflightAndNativeSurfaces(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /neutron/v2.0/routers", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"name": {"selected-name"}, "status": {"ACTIVE"}, "project_id": {"wire-owner"}, "tenant_id": {"wire-tenant"}, "revision_number": {"wire-native"}}
		if !reflect.DeepEqual(r.URL.Query(), want) {
			t.Error("raw native revision or final status query changed", r.URL)
		}
		testcloud.JSON(w, 200, `{"routers":[{"id":"name-miss","name":"other","status":"ACTIVE","revision":7,"tenant_id":"local-tenant"},{"id":"status-miss","name":"selected-name","status":"DOWN","revision":7,"tenant_id":"local-tenant"},{"id":"revision-miss","name":"selected-name","status":"ACTIVE","revision":8,"tenant_id":"local-tenant"},{"id":"selected","name":"selected-name","status":"aCtIvE","revision":"7","revision_number":99,"project_id":"native-owner","tenant_id":"local-tenant"}]}`)
	})
	cloud.Mux.HandleFunc("GET /neutron/v2.0/routers/unfiltered", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if len(r.URL.Query()) != 0 {
			t.Error("list options leaked into Get", r.URL)
		}
		testcloud.JSON(w, 200, `{"router":{"id":"unfiltered","status":"DOWN","revision":9007199254740993,"revision_number":11}}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/neutron/v2.0/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Network(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	v2, err := conn.NetworkV2(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, collection := range []*resource.Collection[routers.Router]{service.API.Routers.Resources, v2.Routers.Resources} {
		before := calls.Load()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if values, err := collection.All(ctx, resource.WithFilter("revision_number", 7)); len(values) != 0 || !errors.Is(err, context.Canceled) || calls.Load() != before {
			t.Fatal("canceled list started HTTP", values, err, calls.Load())
		}
		for _, options := range [][]resource.ListOption{
			{resource.WithFilter("revision_number", 7), resource.WithBodyFilter("revision", 7)},
			{resource.WithFilter("tenant_id", nil), resource.WithBodyFilter("tenant_id", nil)},
			{resource.WithFilter("is_admin_state_up", nil), resource.WithQuery("admin_state_up", "false")},
			{resource.WithFilter("any_tags", []string{}), resource.WithQuery("tags-any", "tag")},
			{resource.WithFilter("name", "selected-name"), resource.WithName("selected-name")},
			{resource.WithFilter("status", "ACTIVE"), resource.WithStatus("ACTIVE")},
			{resource.WithFilter("limit", nil), resource.WithPageSize(2)},
			{resource.WithFilter("evpn_vni", make(chan int))},
		} {
			if values, err := collection.All(context.Background(), options...); len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != before {
				t.Fatal("preflight missed a target collision", values, err, calls.Load())
			}
		}
		values, err := collection.All(context.Background(), resource.WithFilters(map[string]any{"project_id": "wire-owner", "tenant_id": "local-tenant", "revision_number": 7, "revision": make(chan int)}), resource.WithName("selected-name"), resource.WithStatus("earlier-status"), resource.WithQuery("status", "ACTIVE"), resource.WithQuery("tenant_id", "wire-tenant"), resource.WithQuery("revision_number", "wire-native"))
		if err != nil || len(values) != 1 || values[0].ID != "selected" || values[0].RevisionNumber != 99 || calls.Load() != before+1 {
			t.Fatal("raw/native revision or local name/status/tenant composition changed", values, err, calls.Load())
		}
		value, err := collection.Get(context.Background(), "unfiltered")
		if err != nil || value.ID != "unfiltered" || value.RevisionNumber != 11 {
			t.Fatal("list filters changed the native member response", value, err)
		}
	}
	if gets.Load() != 2 {
		t.Fatal(gets.Load())
	}
}
