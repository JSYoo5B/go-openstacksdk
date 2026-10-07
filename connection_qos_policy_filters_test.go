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

func TestConnectionQoSPolicyFiltersSharedFacadesAndExactRawRules(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "network" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Error("catalog selection", options)
		}
		return cloud.Server.URL + "/catalog/neutron/v2.0/", nil
	}
	const path = "/reverse/neutron/v2.0/qos/policies"
	cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"limit": {"2"}, "marker": {"first", "second"}, "fields": {"id", "rules"}, "id": {"wire-id"}, "name": {"server-name"}, "project_id": {"wire-owner"}, "shared": {"false"}, "is_default": {"false"}, "tags": {"red", "blue"}, "tags-any": {"canonical-tag"}, "status": {"vendor-status"}}
		if r.URL.Query().Has("page") {
			want["page"] = []string{"2"}
		}
		if !reflect.DeepEqual(r.URL.Query(), want) || r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("Accept") != "application/json" {
			t.Error("local fields leaked or aliases changed", r.URL, r.Header)
		}
		if !r.URL.Query().Has("page") {
			if r.Header.Get("X-Auth-Token") != "first-token" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("second-token")
			next := cloud.Server.URL + path + "?" + want.Encode() + "&page=2"
			encodedNext, _ := json.Marshal(next)
			testcloud.JSON(w, 200, `{"policies":[{"id":"rounded","tenant_id":"","rules":[{"quota":9007199254740992}]}],"policies_links":[{"rel":"next","href":`+string(encodedNext)+`}]}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "second-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"policies":[{"id":"selected","name":"different-returned-name","project_id":"native-owner","tenant_id":null,"shared":true,"is_default":true,"tags":["other"],"rules":[{"quota":9007199254740993}]}]}`)
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
	if err != nil || v2.RawClient() != service.RawClient() || lookups.Load() != 1 || v2.QoSPolicies.RawClient() != service.API.QoSPolicies.RawClient() {
		t.Fatal("versioned and convenience facades do not share the client", v2, err, lookups.Load())
	}
	client := service.RawClient()
	client.ResourceBase = cloud.Server.URL + "/reverse/neutron/v2.0/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	cloud.Provider.SetToken("first-token")
	values, err := service.API.QoSPolicies.Resources.All(context.Background(), resource.WithFilters(map[string]any{
		"limit": json.Number("2"), "marker": []string{"first", "second"}, "fields": []string{"id", "rules"}, "id": "wire-id", "name": "server-name", "project_id": "wire-owner", "is_shared": false, "is_default": false,
		"tags": []string{"red", "blue"}, "any_tags": "canonical-tag", "tags-any": "ignored-alias", "tenant_id": nil, "rules": json.RawMessage(`[{"quota":9007199254740993}]`), "created_at": make(chan int),
	}), resource.WithQuery("status", "vendor-status"))
	if err != nil || len(values) != 1 || values[0].ID != "selected" || values[0].Name != "different-returned-name" || values[0].ProjectID != "native-owner" || values[0].TenantID != "" || !values[0].Shared || !values[0].IsDefault || !reflect.DeepEqual(values[0].Tags, []string{"other"}) || calls.Load() != 2 || lookups.Load() != 1 {
		t.Fatal("query-only fields or raw rules/null tenant comparison changed", values, err, calls.Load(), lookups.Load())
	}
	quota, ok := values[0].Rules[0]["quota"].(float64)
	if !ok || quota != float64(9007199254740992) || len(client.MoreHeaders) != 1 || client.ProviderClient != cloud.Provider {
		t.Fatal("raw matching replaced native return decoding or shared configuration", values[0], client.MoreHeaders)
	}
}

func TestConnectionQoSPolicyFiltersPreflightAliasesAndNativeNameHint(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /neutron/v2.0/qos/policies", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !reflect.DeepEqual(r.URL.Query(), url.Values{"name": {"selected-name"}, "tenant_id": {"wire-extension"}, "status": {"vendor-status"}, "shared": {"false"}}) {
			t.Error("raw tenant/status, canonical shared or Name hint changed", r.URL)
		}
		testcloud.JSON(w, 200, `{"policies":[{"id":"other","name":"other-name","tenant_id":"local-tenant","rules":[]},{"id":"selected","name":"selected-name","tenant_id":"local-tenant","shared":true,"rules":[]}]}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/neutron/v2.0/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.NetworkV2(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	policies := service.QoSPolicies.Resources
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if values, err := policies.All(ctx, resource.WithFilter("tenant_id", "local-tenant")); len(values) != 0 || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal("canceled list started HTTP", values, err, calls.Load())
	}
	for _, options := range [][]resource.ListOption{
		{resource.WithFilter("is_shared", nil), resource.WithQuery("shared", "false")},
		{resource.WithFilter("any_tags", []string{}), resource.WithQuery("tags-any", "tag")},
		{resource.WithFilter("name", "selected-name"), resource.WithName("selected-name")},
		{resource.WithFilter("tenant_id", "local-tenant"), resource.WithBodyFilter("tenant_id", "local-tenant")},
	} {
		if values, err := policies.All(context.Background(), options...); len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("same-destination collision lost alias/presence ownership", values, err, calls.Load())
		}
	}
	if values, err := policies.All(context.Background(), resource.WithFilter("rules", []any{}), resource.WithStatus("ACTIVE")); len(values) != 0 || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
		t.Fatal("statusless model acquired status support", values, err, calls.Load())
	}
	values, err := policies.All(context.Background(), resource.WithFilters(map[string]any{"is_shared": false, "shared": true, "tenant_id": "local-tenant"}), resource.WithName("selected-name"), resource.WithQuery("tenant_id", "wire-extension"), resource.WithQuery("status", "vendor-status"))
	if err != nil || len(values) != 1 || values[0].ID != "selected" || !values[0].Shared || calls.Load() != 1 {
		t.Fatal("canonical alias precedence, local tenant and native Name hint changed", values, err, calls.Load())
	}
}
