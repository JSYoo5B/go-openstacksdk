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
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/trunks"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionTrunkFiltersSharedClientAndDistinctQueryBodyNamespaces(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "network" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Error("catalog selection", options)
		}
		return cloud.Server.URL + "/catalog/neutron/v2.0/", nil
	}
	const path = "/reverse/neutron/v2.0/trunks"
	cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"limit": {"2"}, "marker": {"first", "second"}, "fields": {"id", "tenant_id", "sub_ports"}, "id": {"wire-id"}, "tenant_id": {"wire-tenant"}, "project_id": {"wire-project"}, "name": {"server-name"}, "status": {"server-status"}, "description": {"wire-description"}, "port_id": {"wire-port"}, "admin_state_up": {"false"}, "sub_ports": {"wire-subports"}, "tags-any": {"canonical-tag"}}
		if r.URL.Query().Has("page") {
			want["page"] = []string{"2"}
		}
		if !reflect.DeepEqual(r.URL.Query(), want) || r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("Accept") != "application/json" {
			t.Error("Body/query namespaces or aliases changed", r.URL, r.Header)
		}
		if !r.URL.Query().Has("page") {
			if r.Header.Get("X-Auth-Token") != "first-token" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("second-token")
			next, _ := json.Marshal(cloud.Server.URL + path + "?" + want.Encode() + "&page=2")
			testcloud.JSON(w, 200, `{"trunks":[{"id":"other","tenant_id":"local-tenant"}],"links":{"next":`+string(next)+`}}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "second-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"trunks":[{"id":"selected","tenant_id":"local-tenant","project_id":"returned-project","name":"returned-name","status":"DOWN","description":"returned-description","port_id":"returned-port","admin_state_up":true,"revision_number":7,"created_at":"2026-10-02T01:02:03Z","updated_at":null,"sub_ports":[null,{"port_id":"native-subport","segmentation_type":"vlan","segmentation_id":23,"extension":9007199254740993}]}]}`)
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
		t.Fatal("network facade was not cached", err)
	}
	v2, err := conn.NetworkV2(context.Background())
	if err != nil || v2.RawClient() != service.RawClient() || v2.Trunks.RawClient() != service.API.Trunks.RawClient() || lookups.Load() != 1 {
		t.Fatal("Trunk facades must share one cached client", err, lookups.Load())
	}
	client := service.RawClient()
	client.ResourceBase = cloud.Server.URL + "/reverse/neutron/v2.0/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	option := resource.WithFilters(map[string]any{
		"id": "selected", "tenant_id": "local-tenant", "project_id": "wire-project", "limit": 2, "marker": []string{"first", "second"}, "fields": []string{"id", "tenant_id", "sub_ports"}, "name": "server-name", "status": "server-status", "description": "wire-description", "port_id": "wire-port", "is_admin_state_up": false, "sub_ports": "wire-subports", "any_tags": "canonical-tag", "tags-any": "ignored-alias",
		"created_at": make(chan int), "updated_at": json.RawMessage(`{`), "revision_number": func() {},
	})
	for _, collection := range []*resource.Collection[trunks.Trunk]{service.API.Trunks.Resources, v2.Trunks.Resources} {
		cloud.Provider.SetToken("first-token")
		values, err := collection.All(context.Background(), option, resource.WithQuery("id", "wire-id"), resource.WithQuery("tenant_id", "wire-tenant"))
		if err != nil || len(values) != 1 {
			t.Fatal("local identity and server conditions changed", values, err)
		}
		value := values[0]
		if value.ID != "selected" || value.TenantID != "local-tenant" || value.ProjectID != "returned-project" || value.Name != "returned-name" || value.Status != "DOWN" || !value.AdminStateUp || value.RevisionNumber != 7 || value.PortID != "returned-port" || value.Description != "returned-description" || len(value.Subports) != 2 || value.Subports[0] != (trunks.Subport{}) || value.Subports[1].PortID != "native-subport" || value.Subports[1].SegmentationID != 23 || !value.CreatedAt.Equal(time.Date(2026, 10, 2, 1, 2, 3, 0, time.UTC)) || !value.UpdatedAt.IsZero() {
			t.Fatal("raw matching changed native response projection", value)
		}
	}
	if calls.Load() != 4 || lookups.Load() != 1 || client.ProviderClient != cloud.Provider || len(client.MoreHeaders) != 1 {
		t.Fatal("continuation changed shared configuration", calls.Load(), lookups.Load(), client.MoreHeaders)
	}
}

func TestConnectionTrunkFiltersPreflightFinalStatusAndNativeSurfaces(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /neutron/v2.0/trunks", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.Query().Get("case") == "native-error-after-cap" {
			testcloud.JSON(w, 200, `{"trunks":[{"id":"first"},{"id":"invalid","sub_ports":[{"segmentation_id":"bad"}]}]}`)
			return
		}
		want := url.Values{"name": {"selected-name"}, "status": {"DOWN"}, "project_id": {"wire-project"}, "revision_number": {"101"}, "tenant_id": {"wire-tenant"}}
		if !reflect.DeepEqual(r.URL.Query(), want) {
			t.Error("final status or server-only query changed", r.URL)
		}
		testcloud.JSON(w, 200, `{"trunks":[{"id":"wrong-name","name":"Selected-name","status":"down","tenant_id":null},{"id":"wrong-status","name":"selected-name","status":"ACTIVE","tenant_id":null},{"id":"empty-tenant","name":"selected-name","status":"down","tenant_id":""},{"id":"selected","name":"selected-name","status":"down","tenant_id":null,"project_id":"native-project","revision_number":7,"sub_ports":null}]}`)
	})
	cloud.Mux.HandleFunc("GET /neutron/v2.0/trunks/unfiltered", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if len(r.URL.Query()) != 0 {
			t.Error("list controls leaked into native Get", r.URL)
		}
		testcloud.JSON(w, 200, `{"trunk":{"id":"unfiltered","tenant_id":"native-tenant","sub_ports":[{"port_id":"native-port","segmentation_type":"vlan","segmentation_id":42}]}}`)
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
	for _, collection := range []*resource.Collection[trunks.Trunk]{service.API.Trunks.Resources, v2.Trunks.Resources} {
		before := lists.Load()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if values, err := collection.All(ctx, resource.WithFilter("tenant_id", nil)); len(values) != 0 || !errors.Is(err, context.Canceled) || lists.Load() != before {
			t.Fatal("canceled list started HTTP", values, err, lists.Load())
		}
		for _, options := range [][]resource.ListOption{
			{resource.WithFilter("id", nil), resource.WithBodyFilter("id", nil)},
			{resource.WithFilter("tenant_id", nil), resource.WithBodyFilter("tenant_id", nil)},
			{resource.WithFilter("project_id", "project"), resource.WithQuery("project_id", "other")},
			{resource.WithFilter("is_admin_state_up", false), resource.WithQuery("admin_state_up", "true")},
			{resource.WithFilter("any_tags", []string{}), resource.WithQuery("tags-any", "tag")},
			{resource.WithFilter("status", "ACTIVE"), resource.WithStatus("ACTIVE")},
			{resource.WithFilter("name", "selected-name"), resource.WithName("selected-name")},
			{resource.WithFilter("limit", nil), resource.WithPageSize(2)},
			{resource.WithFilter("tenant_id", make(chan int))},
		} {
			if values, err := collection.All(context.Background(), options...); len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || lists.Load() != before {
				t.Fatal("preflight missed a filter collision", values, err, lists.Load())
			}
		}
		values, err := collection.All(context.Background(), resource.WithFilters(map[string]any{"project_id": "wire-project", "tenant_id": nil, "revision_number": make(chan int)}), resource.WithName("selected-name"), resource.WithStatus("ACTIVE"), resource.WithQuery("status", "DOWN"), resource.WithQuery("revision_number", "101"), resource.WithQuery("tenant_id", "wire-tenant"))
		if err != nil || len(values) != 1 || values[0].ID != "selected" || values[0].Status != "down" || values[0].TenantID != "" || values[0].ProjectID != "native-project" || values[0].RevisionNumber != 7 || values[0].Subports != nil || lists.Load() != before+1 {
			t.Fatal("final local status or null tenant semantics changed", values, err, lists.Load())
		}
		value, err := collection.Get(context.Background(), "unfiltered")
		if err != nil || value.ID != "unfiltered" || value.TenantID != "native-tenant" || len(value.Subports) != 1 || value.Subports[0].SegmentationID != 42 {
			t.Fatal("list filters changed native member response", value, err)
		}
		values, err = collection.All(context.Background(), resource.WithBodyFilter("id", "first"), resource.WithQuery("case", "native-error-after-cap"), resource.WithMaxItems(1))
		var typed *json.UnmarshalTypeError
		if values != nil || !errors.As(err, &typed) || lists.Load() != before+2 {
			t.Fatal("raw cap hid a whole-page native nested error", values, err, lists.Load())
		}
	}
	if gets.Load() != 2 {
		t.Fatal(gets.Load())
	}
}
