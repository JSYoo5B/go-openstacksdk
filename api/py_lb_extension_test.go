package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/l7policies"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/listeners"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/loadbalancers"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/pools"
)

// Python Resource attributes and query keys that the pinned Gophercloud typed
// options do not model are reachable through the library-owned extension
// options and reach the wire unchanged.
func TestPyLBPythonOnlyWireKeysTravelThroughExtensionOptions(t *testing.T) {
	cloud := testcloud.New(t)
	var createBody map[string]json.RawMessage
	queries := map[string]url.Values{}
	cloud.Mux.HandleFunc("POST /v2/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		createBody = decodeOctaviaLoadBalancerRequest(t, r)
		testcloud.JSON(w, 201, octaviaLoadBalancerBody)
	})
	for path, key := range map[string]string{
		"/v2/lbaas/loadbalancers":          "loadbalancers",
		"/v2/lbaas/listeners":              "listeners",
		"/v2/lbaas/pools/pool-1/members":   "members",
		"/v2/lbaas/l7policies/pol-1/rules": "rules",
	} {
		cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
			queries[r.URL.Path] = r.URL.Query()
			testcloud.JSON(w, 200, `{"`+key+`":[]}`)
		})
	}
	ctx := context.Background()
	client := cloud.Client("load-balancer", "/v2")
	lbs := loadbalancers.New(client)
	if _, err := lbs.Create(ctx, loadbalancers.CreateOpts{VipSubnetID: "subnet-1"}, loadbalancers.WithCreateField("vip_sg_ids", []string{"sg-1", "sg-2"})); err != nil {
		t.Fatal(err)
	}
	if string(createBody["vip_sg_ids"]) != `["sg-1","sg-2"]` || string(createBody["vip_subnet_id"]) != `"subnet-1"` {
		t.Fatalf("create body=%v", createBody)
	}
	for _, err := range lbs.List(ctx, loadbalancers.WithListQuery("vip_qos_policy_id", "qos-1")) {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, err := range listeners.New(client).List(ctx, listeners.WithListQuery("hsts_max_age", "10"), listeners.WithListQuery("alpn_protocols", "h2")) {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, err := range pools.New(client).ListMembers(ctx, "pool-1", pools.WithListMembersQuery("backup", "true")) {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, err := range l7policies.New(client).ListRules(ctx, "pol-1", l7policies.WithListRulesQuery("invert", "false")) {
		if err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]url.Values{
		"/v2/lbaas/loadbalancers":          {"vip_qos_policy_id": {"qos-1"}},
		"/v2/lbaas/listeners":              {"hsts_max_age": {"10"}, "alpn_protocols": {"h2"}},
		"/v2/lbaas/pools/pool-1/members":   {"backup": {"true"}},
		"/v2/lbaas/l7policies/pol-1/rules": {"invert": {"false"}},
	}
	if !reflect.DeepEqual(queries, want) {
		t.Fatalf("queries=%v", queries)
	}
}

// Python update_* only sends a request when an attribute is dirty (the base
// commit returns without HTTP otherwise). Go always sends the requested PUT,
// here an empty envelope, so a no-field update is observable and never silent.
func TestPyLBUpdateWithoutFieldsStillSendsEmptyEnvelope(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []string
	cloud.Mux.HandleFunc("PUT /v2/lbaas/loadbalancers/lb-1", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(data))
		testcloud.JSON(w, 200, octaviaLoadBalancerBody)
	})
	if _, err := newOctaviaLoadBalancers(t, cloud).Update(context.Background(), "lb-1", loadbalancers.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(bodies, []string{`{"loadbalancer":{}}`}) {
		t.Fatalf("bodies=%q", bodies)
	}
}
