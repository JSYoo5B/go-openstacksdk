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

	"github.com/gophercloud/gophercloud/v2"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestConnectionSubnetPoolFiltersSharedFacadesAndRawDescriptors(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "network" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Error("catalog selection", options)
		}
		return cloud.Server.URL + "/catalog/neutron/v2.0/", nil
	}
	const path = "/reverse/neutron/v2.0/subnetpools"
	cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"limit": {"2"}, "marker": {"first", "second"}, "fields": {"id", "prefixes"}, "name": {"server-name"}, "project_id": {"wire-owner"}, "shared": {"false"}, "is_default": {"false"}, "tags-any": {"canonical-tag"}, "ip_version": {"6"}, "address_scope_id": {"wire-scope"}, "status": {"vendor-status"}}
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
			testcloud.JSON(w, 200, `{"subnetpools":[{"id":"other","default_prefixlen":24,"min_prefixlen":8,"max_prefixlen":32}],"subnetpools_links":[{"rel":"next","href":`+string(next)+`}]}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "second-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"subnetpools":[{"id":"selected","name":"different-returned-name","project_id":"native-owner","tenant_id":null,"shared":true,"is_default":true,"ip_version":4,"address_scope_id":"native-scope","default_prefixlen":"+24","min_prefixlen":"+8","max_prefixlen":32,"default_quota":9007199254740993,"revision_number":9007199254740993,"prefixes":["192.0.2.0/24",null],"created_at":"2026-10-02T01:02:03","updated_at":"2026-10-02T02:03:04"}]}`)
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
	if err != nil || v2.RawClient() != service.RawClient() || lookups.Load() != 1 || v2.SubnetPools.RawClient() != service.API.SubnetPools.RawClient() {
		t.Fatal("versioned and convenience facades do not share the client", v2, err, lookups.Load())
	}
	client := service.RawClient()
	client.ResourceBase = cloud.Server.URL + "/reverse/neutron/v2.0/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	cloud.Provider.SetToken("first-token")
	values, err := service.API.SubnetPools.Resources.All(context.Background(), resource.WithFilters(map[string]any{
		"limit": json.Number("2"), "marker": []string{"first", "second"}, "fields": []string{"id", "prefixes"}, "name": "server-name", "project_id": "wire-owner", "is_shared": false, "is_default": false,
		"ip_version": 6, "address_scope_id": "wire-scope", "any_tags": "canonical-tag", "tags-any": "ignored-alias",
		"id": "selected", "tenant_id": nil, "prefixes": json.RawMessage(`["192.0.2.0/24",null]`), "created_at": "2026-10-02T01:02:03", "updated_at": "2026-10-02T02:03:04",
		"default_prefix_length": 24, "minimum_prefix_length": 8, "maximum_prefix_length": 32, "default_quota": json.Number("9007199254740993"), "revision_number": json.Number("9007199254740993"), "default_prefixlen": make(chan int),
	}), resource.WithQuery("status", "vendor-status"))
	if err != nil || len(values) != 1 || calls.Load() != 2 || lookups.Load() != 1 {
		t.Fatal("query classification, raw descriptors or continuation changed", values, err, calls.Load(), lookups.Load())
	}
	value := values[0]
	if value.ID != "selected" || value.Name != "different-returned-name" || value.ProjectID != "native-owner" || value.TenantID != "" || !value.Shared || !value.IsDefault || value.IPversion != 4 || value.AddressScopeID != "native-scope" || value.DefaultPrefixLen != 24 || value.MinPrefixLen != 8 || value.MaxPrefixLen != 32 || value.DefaultQuota != 9007199254740993 || value.RevisionNumber != 9007199254740993 || !reflect.DeepEqual(value.Prefixes, []string{"192.0.2.0/24", ""}) || !value.CreatedAt.Equal(time.Date(2026, 10, 2, 1, 2, 3, 0, time.UTC)) || !value.UpdatedAt.Equal(time.Date(2026, 10, 2, 2, 3, 4, 0, time.UTC)) {
		t.Fatal("local matching changed the native returned model", value)
	}
	if len(client.MoreHeaders) != 1 || client.ProviderClient != cloud.Provider {
		t.Fatal("list changed shared client configuration", client.MoreHeaders)
	}
}

func TestConnectionSubnetPoolFiltersPreflightAndIndependentRawID(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /neutron/v2.0/subnetpools", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"name": {"selected-name"}, "id": {"wire-id"}, "tenant_id": {"wire-extension"}, "status": {"vendor-status"}, "shared": {"false"}}
		if !reflect.DeepEqual(r.URL.Query(), want) {
			t.Error("raw id/tenant/status, shared alias or Name hint changed", r.URL)
		}
		testcloud.JSON(w, 200, `{"subnetpools":[{"id":"other","name":"other-name","tenant_id":"local-tenant","default_prefixlen":24,"min_prefixlen":8,"max_prefixlen":32},{"id":"selected","name":"selected-name","tenant_id":"local-tenant","shared":true,"default_prefixlen":24,"min_prefixlen":8,"max_prefixlen":32}]}`)
	})
	cloud.Mux.HandleFunc("GET /neutron/v2.0/subnetpools/unfiltered", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if len(r.URL.Query()) != 0 {
			t.Error("list options leaked into Get", r.URL)
		}
		testcloud.JSON(w, 200, `{"subnetpool":{"id":"unfiltered","tenant_id":"another-tenant","default_prefixlen":24,"min_prefixlen":8,"max_prefixlen":32}}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/neutron/v2.0/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.NetworkV2(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	pools := service.SubnetPools.Resources
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if values, err := pools.All(ctx, resource.WithFilter("id", "selected")); len(values) != 0 || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal("canceled list started HTTP", values, err, calls.Load())
	}
	for _, options := range [][]resource.ListOption{
		{resource.WithFilter("is_shared", nil), resource.WithQuery("shared", "false")},
		{resource.WithFilter("any_tags", []string{}), resource.WithQuery("tags-any", "tag")},
		{resource.WithFilter("name", "selected-name"), resource.WithName("selected-name")},
		{resource.WithFilter("default_prefix_length", 24), resource.WithBodyFilter("default_prefixlen", 24)},
		{resource.WithFilter("limit", nil), resource.WithPageSize(2)},
		{resource.WithFilter("maximum_prefix_length", make(chan int))},
	} {
		if values, err := pools.All(context.Background(), options...); len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("preflight missed selected-value or target collision", values, err, calls.Load())
		}
	}
	if values, err := pools.All(context.Background(), resource.WithFilter("id", "selected"), resource.WithStatus("ACTIVE")); len(values) != 0 || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
		t.Fatal("statusless model acquired typed status support", values, err, calls.Load())
	}
	values, err := pools.All(context.Background(), resource.WithFilters(map[string]any{"is_shared": false, "shared": true, "tenant_id": "local-tenant", "id": "selected"}), resource.WithName("selected-name"), resource.WithQuery("id", "wire-id"), resource.WithQuery("tenant_id", "wire-extension"), resource.WithQuery("status", "vendor-status"))
	if err != nil || len(values) != 1 || values[0].ID != "selected" || !values[0].Shared || calls.Load() != 1 {
		t.Fatal("raw id became a semantic collision or query fields became local", values, err, calls.Load())
	}
	value, err := service.SubnetPools.Get(context.Background(), "unfiltered")
	if err != nil || value.ID != "unfiltered" || gets.Load() != 1 {
		t.Fatal("list filters escaped into the native Get", value, err, gets.Load())
	}
}
