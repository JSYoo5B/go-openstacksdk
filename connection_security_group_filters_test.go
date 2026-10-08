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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionSecurityGroupFiltersSharedClientAndOriginalRules(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "network" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Error("catalog selection", options)
		}
		return cloud.Server.URL + "/catalog/neutron/v2.0/", nil
	}
	const path = "/reverse/neutron/v2.0/security-groups"
	const originalRules = `[{"id":"rule-one","direction":"ingress","protocol":"tcp","port_range_min":80,"port_range_max":80,"security_group_id":"selected","weight":9007199254740993,"extension":{"nullable":null,"quota":9007199254740993},"created_at":"2026-10-02T01:02:03Z","updated_at":"2026-10-02T02:03:04Z"},null]`
	cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"limit": {"2"}, "marker": {"first", "second"}, "fields": {"id", "name", "security_group_rules", "created_at", "updated_at"}, "id": {"wire-id"}, "name": {"server-name"}, "description": {"wire-description"}, "project_id": {"wire-project"}, "tenant_id": {"wire-tenant"}, "revision_number": {"101"}, "stateful": {"false"}, "shared": {"false"}, "tags-any": {"canonical-tag"}, "status": {"wire-status"}}
		if r.URL.Query().Has("page") {
			want["page"] = []string{"2"}
		}
		if !reflect.DeepEqual(r.URL.Query(), want) || r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("Accept") != "application/json" {
			t.Error("local conditions leaked or query aliases changed", r.URL, r.Header)
		}
		if !r.URL.Query().Has("page") {
			if r.Header.Get("X-Auth-Token") != "first-token" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("second-token")
			next, _ := json.Marshal(cloud.Server.URL + path + "?" + want.Encode() + "&page=2")
			testcloud.JSON(w, 200, `{"security_groups":[{"id":"other","security_group_rules":[]}],"security_groups_links":[{"rel":"next","href":`+string(next)+`}]}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "second-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"security_groups":[{"id":"selected","name":"different-returned-name","description":"different-description","tenant_id":"native-tenant","project_id":"native-project","stateful":true,"revision_number":7,"shared":true,"security_group_rules":`+originalRules+`,"created_at":"2026-10-02T01:02:03","updated_at":"2026-10-02T02:03:04"}]}`)
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
	if err != nil || v2.RawClient() != service.RawClient() || v2.SecurityGroups.RawClient() != service.API.SecurityGroups.RawClient() || lookups.Load() != 1 {
		t.Fatal("security groups must share the cached client", err, lookups.Load())
	}
	client := service.RawClient()
	client.ResourceBase = cloud.Server.URL + "/reverse/neutron/v2.0/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	option := resource.WithFilters(map[string]any{
		"limit": 2, "marker": []string{"first", "second"}, "fields": []string{"id", "name", "security_group_rules", "created_at", "updated_at"}, "id": "wire-id", "name": "server-name", "description": "wire-description", "project_id": "wire-project", "tenant_id": "wire-tenant", "revision_number": 101, "stateful": false, "is_shared": false, "any_tags": "canonical-tag", "tags-any": "ignored-alias",
		"security_group_rules": json.RawMessage(originalRules), "created_at": "2026-10-02T01:02:03", "updated_at": "2026-10-02T02:03:04", "status": make(chan int),
	})
	for _, collection := range []*resource.Collection[groups.SecGroup]{service.API.SecurityGroups.Resources, v2.SecurityGroups.Resources} {
		cloud.Provider.SetToken("first-token")
		values, err := collection.All(context.Background(), option, resource.WithQuery("status", "wire-status"))
		if err != nil || len(values) != 1 {
			t.Fatal("server-only query or original rules changed", values, err)
		}
		value := values[0]
		if value.ID != "selected" || value.Name != "different-returned-name" || value.Description != "different-description" || value.TenantID != "native-tenant" || value.ProjectID != "native-project" || !value.Stateful || value.RevisionNumber != 7 || len(value.Rules) != 2 || value.Rules[0].ID != "rule-one" || value.Rules[0].PortRangeMin != 80 || value.Rules[1] != (rules.SecGroupRule{}) || !value.Rules[0].CreatedAt.Equal(time.Date(2026, 10, 2, 1, 2, 3, 0, time.UTC)) || !value.CreatedAt.Equal(value.Rules[0].CreatedAt) {
			t.Fatal("raw matching changed the native nested projection", value)
		}
	}
	if calls.Load() != 4 || lookups.Load() != 1 || client.ProviderClient != cloud.Provider || len(client.MoreHeaders) != 1 {
		t.Fatal("continuation changed shared configuration", calls.Load(), lookups.Load(), client.MoreHeaders)
	}
}

func TestConnectionSecurityGroupFiltersPreflightAndNativeSurfaces(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /neutron/v2.0/security-groups", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.Query().Get("case") == "native-error-after-cap" {
			testcloud.JSON(w, 200, `{"security_groups":[{"id":"first"},{"id":"invalid","security_group_rules":[{"port_range_min":"bad"}]}]}`)
			return
		}
		want := url.Values{"name": {"selected-name"}, "status": {"wire-status"}, "tenant_id": {"wire-tenant"}, "project_id": {"wire-project"}, "revision_number": {"101"}}
		if !reflect.DeepEqual(r.URL.Query(), want) {
			t.Error("declared query or raw status changed", r.URL)
		}
		testcloud.JSON(w, 200, `{"security_groups":[{"id":"name-miss","name":"other","security_group_rules":null},{"id":"selected","name":"selected-name","tenant_id":"native-tenant","project_id":"native-project","revision_number":7,"security_group_rules":null}]}`)
	})
	cloud.Mux.HandleFunc("GET /neutron/v2.0/security-groups/unfiltered", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if len(r.URL.Query()) != 0 {
			t.Error("list options leaked into Get", r.URL)
		}
		testcloud.JSON(w, 200, `{"security_group":{"id":"unfiltered","security_group_rules":[{"id":"native-rule","extension":9007199254740993}]}}`)
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
	for _, collection := range []*resource.Collection[groups.SecGroup]{service.API.SecurityGroups.Resources, v2.SecurityGroups.Resources} {
		before := lists.Load()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if values, err := collection.All(ctx, resource.WithFilter("security_group_rules", nil)); len(values) != 0 || !errors.Is(err, context.Canceled) || lists.Load() != before {
			t.Fatal("canceled list started HTTP", values, err, lists.Load())
		}
		for _, options := range [][]resource.ListOption{
			{resource.WithFilter("security_group_rules", nil), resource.WithBodyFilter("security_group_rules", nil)},
			{resource.WithFilter("created_at", nil), resource.WithBodyFilter("created_at", nil)},
			{resource.WithFilter("tenant_id", nil), resource.WithQuery("tenant_id", "tenant")},
			{resource.WithFilter("revision_number", nil), resource.WithQuery("revision_number", "101")},
			{resource.WithFilter("is_shared", nil), resource.WithQuery("shared", "false")},
			{resource.WithFilter("any_tags", []string{}), resource.WithQuery("tags-any", "tag")},
			{resource.WithFilter("name", "selected-name"), resource.WithName("selected-name")},
			{resource.WithFilter("limit", nil), resource.WithPageSize(2)},
			{resource.WithFilter("updated_at", make(chan int))},
		} {
			if values, err := collection.All(context.Background(), options...); len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || lists.Load() != before {
				t.Fatal("preflight missed a target collision", values, err, lists.Load())
			}
		}
		if values, err := collection.All(context.Background(), resource.WithStatus("ACTIVE")); len(values) != 0 || !errors.Is(err, resource.ErrUnsupported) || lists.Load() != before {
			t.Fatal("native model without Status acquired a local predicate", values, err, lists.Load())
		}
		values, err := collection.All(context.Background(), resource.WithFilters(map[string]any{"project_id": "wire-project", "tenant_id": "wire-tenant", "revision_number": 101, "security_group_rules": nil, "status": make(chan int)}), resource.WithName("selected-name"), resource.WithQuery("status", "wire-status"))
		if err != nil || len(values) != 1 || values[0].ID != "selected" || values[0].RevisionNumber != 7 || values[0].Rules != nil || lists.Load() != before+1 {
			t.Fatal("server conditions became local or null rules changed", values, err, lists.Load())
		}
		value, err := collection.Get(context.Background(), "unfiltered")
		if err != nil || value.ID != "unfiltered" || len(value.Rules) != 1 || value.Rules[0].ID != "native-rule" {
			t.Fatal("list filters changed the native member response", value, err)
		}
		values, err = collection.All(context.Background(), resource.WithBodyFilter("security_group_rules", nil), resource.WithQuery("case", "native-error-after-cap"), resource.WithMaxItems(1))
		var typed *json.UnmarshalTypeError
		if values != nil || !errors.As(err, &typed) || lists.Load() != before+2 {
			t.Fatal("raw cap hid a native nested decoder failure", values, err, lists.Load())
		}
	}
	if gets.Load() != 2 {
		t.Fatal(gets.Load())
	}
}
