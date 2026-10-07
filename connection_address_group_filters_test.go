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

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionAddressGroupFiltersSharedFacadesAndRawTenant(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "network" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Error("catalog selection", options)
		}
		return cloud.Server.URL + "/catalog/neutron/v2.0/", nil
	}
	const path = "/reverse/neutron/v2.0/address-groups"
	cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"limit": {"2"}, "marker": {"first", "second"}, "fields": {"id", "addresses"}, "name": {"server-filter-name"}, "project_id": {"wire-owner"}, "status": {"vendor-status"}}
		if r.URL.Query().Has("page") {
			want["page"] = []string{"2"}
		}
		if !reflect.DeepEqual(r.URL.Query(), want) || r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("Accept") != "application/json" {
			t.Error("local attributes leaked or native queries changed", r.URL, r.Header)
		}
		if !r.URL.Query().Has("page") {
			if r.Header.Get("X-Auth-Token") != "first-token" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("second-token")
			next := cloud.Server.URL + path + "?" + want.Encode() + "&page=2"
			encodedNext, _ := json.Marshal(next)
			testcloud.JSON(w, 200, `{"address_groups":[{"id":"first","name":"other","tenant_id":{"owner":9007199254740992},"addresses":["192.0.2.1"]}],"address_groups_links":[{"rel":"next","href":`+string(encodedNext)+`}]}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "second-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"address_groups":[{"id":"selected","name":"different-returned-name","project_id":"native-owner","tenant_id":{"owner":9007199254740993,"unknown":true},"addresses":["192.0.2.1"]}]}`)
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
	if err != nil || v2.RawClient() != service.RawClient() || lookups.Load() != 1 || v2.SecurityAddressGroups.RawClient() != service.API.SecurityAddressGroups.RawClient() {
		t.Fatal("versioned and convenience facades do not share the client", v2, err, lookups.Load())
	}
	client := service.RawClient()
	client.ResourceBase = cloud.Server.URL + "/reverse/neutron/v2.0/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	cloud.Provider.SetToken("first-token")
	values, err := service.API.SecurityAddressGroups.Resources.All(context.Background(), resource.WithFilters(map[string]any{
		"limit": json.Number("2"), "marker": []string{"first", "second"}, "fields": []string{"id", "addresses"}, "name": "server-filter-name", "project_id": "wire-owner",
		"tenant_id": map[string]any{"owner": json.Number("9007199254740993")}, "addresses": []string{"192.0.2.1"}, "vendor": make(chan int),
	}), resource.WithQuery("status", "vendor-status"))
	if err != nil || len(values) != 1 || values[0].ID != "selected" || values[0].Name != "different-returned-name" || values[0].ProjectID != "native-owner" || calls.Load() != 2 || lookups.Load() != 1 {
		t.Fatal("query-only name/project or raw tenant comparison changed", values, err, calls.Load(), lookups.Load())
	}
	if len(client.MoreHeaders) != 1 || client.ProviderClient != cloud.Provider {
		t.Fatal("list filters replaced shared configuration")
	}
}

func TestConnectionAddressGroupFiltersPreflightAndNativeNameHint(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /neutron/v2.0/address-groups", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !reflect.DeepEqual(r.URL.Query(), url.Values{"name": {"selected-name"}, "tenant_id": {"wire-extension"}, "status": {"vendor-status"}}) {
			t.Error("raw tenant/status or Name hint changed", r.URL)
		}
		testcloud.JSON(w, 200, `{"address_groups":[{"id":"other","name":"other-name","tenant_id":"local-tenant","addresses":[]},{"id":"selected","name":"selected-name","tenant_id":"local-tenant","addresses":[]}]}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/neutron/v2.0/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.NetworkV2(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	groups := service.SecurityAddressGroups.Resources
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if values, err := groups.All(ctx, resource.WithFilter("tenant_id", "local-tenant")); len(values) != 0 || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal("canceled list started HTTP", values, err, calls.Load())
	}
	for _, options := range [][]resource.ListOption{
		{resource.WithFilter("marker", nil), resource.WithQuery("marker", "raw")},
		{resource.WithFilter("name", "selected-name"), resource.WithName("selected-name")},
		{resource.WithFilter("tenant_id", "local-tenant"), resource.WithBodyFilter("tenant_id", "local-tenant")},
	} {
		if values, err := groups.All(context.Background(), options...); len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("same-destination collision lost ownership", values, err, calls.Load())
		}
	}
	if values, err := groups.All(context.Background(), resource.WithFilter("tenant_id", "local-tenant"), resource.WithStatus("ACTIVE")); len(values) != 0 || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
		t.Fatal("statusless model acquired status support", values, err, calls.Load())
	}
	values, err := groups.All(context.Background(), resource.WithFilter("tenant_id", "local-tenant"), resource.WithName("selected-name"), resource.WithQuery("tenant_id", "wire-extension"), resource.WithQuery("status", "vendor-status"))
	if err != nil || len(values) != 1 || values[0].ID != "selected" || calls.Load() != 1 {
		t.Fatal("local tenant and native Name hint namespaces changed", values, err, calls.Load())
	}
}
