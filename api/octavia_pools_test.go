package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/pools"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const octaviaPoolBody = `{"pool":{"id":"pool-1","name":"web","lb_algorithm":"ROUND_ROBIN","protocol":"HTTP","admin_state_up":false,"provisioning_status":"PENDING_CREATE","operating_status":"OFFLINE","tags":["blue"],"vendor_extra":1}}`
const octaviaPoolMemberBody = `{"member":{"id":"m-1","name":"backend","address":"192.0.2.10","protocol_port":8080,"weight":0,"admin_state_up":false,"backup":false,"provisioning_status":"PENDING_CREATE","tags":["blue"]}}`

func newOctaviaPools(t *testing.T, cloud *testcloud.Cloud) *pools.API {
	t.Helper()
	return pools.New(cloud.Client("load-balancer", "/v2"))
}

func octaviaPoolBool(value bool) *bool { return &value }
func octaviaPoolInt(value int) *int    { return &value }

// decodeOctaviaPoolRequest returns the raw members of the named envelope key.
func decodeOctaviaPoolRequest(t *testing.T, r *http.Request, key string) map[string]string {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("request body: %v", err)
	}
	if len(body) != 1 {
		t.Errorf("envelope keys=%v", body)
	}
	var inner map[string]json.RawMessage
	if err := json.Unmarshal(body[key], &inner); err != nil {
		t.Errorf("envelope %q: %v", key, err)
	}
	got := map[string]string{}
	for k, v := range inner {
		got[k] = string(v)
	}
	return got
}

func octaviaPoolWantStatus(t *testing.T, status int, success bool, err error) {
	t.Helper()
	if success != (err == nil) {
		t.Fatalf("status %d error=%v", status, err)
	}
	if !success {
		var unexpected gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &unexpected) || unexpected.Actual != status {
			t.Fatalf("status %d error=%v", status, err)
		}
	}
}

func TestOctaviaPoolCreateKeepsExplicitFalseZeroNestedMembersAndExtensionFields(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/lbaas/pools", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			want := map[string]string{
				"lb_algorithm": `"ROUND_ROBIN"`, "protocol": `"HTTP"`, "loadbalancer_id": `"lb-1"`, "admin_state_up": `false`,
				"session_persistence": `{"type":"HTTP_COOKIE"}`, "tags": `["blue"]`, "vendor": `{"enabled":false}`,
				"members": `[{"address":"192.0.2.10","protocol_port":8080,"weight":0,"admin_state_up":false}]`,
			}
			got := decodeOctaviaPoolRequest(t, r, "pool")
			if !jsonEqualOctaviaPool([]byte(got["members"]), want["members"]) {
				t.Errorf("members=%s", got["members"])
			}
			delete(got, "members")
			delete(want, "members")
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaPoolBody)
		})
		value, err := newOctaviaPools(t, cloud).Create(context.Background(), pools.CreateOpts{
			LBMethod: pools.LBMethodRoundRobin, Protocol: pools.ProtocolHTTP, LoadbalancerID: "lb-1",
			AdminStateUp: octaviaPoolBool(false), Persistence: &pools.SessionPersistence{Type: "HTTP_COOKIE"},
			Members: []pools.CreateMemberOpts{{Address: "192.0.2.10", ProtocolPort: 8080, Weight: octaviaPoolInt(0), AdminStateUp: octaviaPoolBool(false)}},
			Tags:    []string{"blue"},
		}, pools.WithCreateField("vendor", map[string]bool{"enabled": false}))
		octaviaPoolWantStatus(t, tc.status, tc.success, err)
		if calls.Load() != 1 {
			t.Fatalf("calls=%d", calls.Load())
		}
		if tc.success && (value == nil || value.ID != "pool-1" || value.AdminStateUp || value.LBMethod != "ROUND_ROBIN" || value.ProvisioningStatus != "PENDING_CREATE") {
			t.Fatalf("value=%+v", value)
		}
	}
}

func TestOctaviaPoolRejectsProtectedOrUnsupportedExtensionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newOctaviaPools(t, cloud)
	ctx := context.Background()
	create := pools.CreateOpts{LBMethod: pools.LBMethodRoundRobin, Protocol: pools.ProtocolHTTP}
	member := pools.CreateMemberOpts{Address: "192.0.2.10", ProtocolPort: 8080}
	cases := map[string]error{}
	_, cases["protected create field"] = api.Create(ctx, create, pools.WithCreateField("protocol", "TCP"))
	_, cases["protected update field"] = api.Update(ctx, "pool-1", pools.UpdateOpts{}, pools.WithUpdateField("admin_state_up", true))
	_, cases["protected member create field"] = api.CreateMember(ctx, "pool-1", member, pools.WithCreateMemberField("address", "x"))
	_, cases["protected member update field"] = api.UpdateMember(ctx, "pool-1", "m-1", pools.UpdateMemberOpts{}, pools.WithUpdateMemberField("weight", 1))
	_, cases["nil create option"] = api.Create(ctx, create, nil)
	_, cases["nil update option"] = api.Update(ctx, "pool-1", pools.UpdateOpts{}, nil)
	_, cases["nil member create option"] = api.CreateMember(ctx, "pool-1", member, nil)
	_, cases["nil member update option"] = api.UpdateMember(ctx, "pool-1", "m-1", pools.UpdateMemberOpts{}, nil)
	for _, err := range api.List(ctx, nil) {
		cases["nil list option"] = err
	}
	for _, err := range api.ListMembers(ctx, "pool-1", nil) {
		cases["nil list members option"] = err
	}
	if len(cases) != 10 {
		t.Fatalf("cases=%v", cases)
	}
	for name, err := range cases {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestOctaviaPoolUpdateSendsEmptyValuesAndNullsAndAcceptsOnly200(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/pools/pool-1", func(w http.ResponseWriter, r *http.Request) {
			want := map[string]string{
				"name": `""`, "description": `""`, "admin_state_up": `false`, "tls_enabled": `false`, "tags": `[]`,
				"session_persistence": `null`, "alpn_protocols": `null`, "tls_versions": `null`, "vendor": `1`,
			}
			if got := decodeOctaviaPoolRequest(t, r, "pool"); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaPoolBody)
		})
		empty := ""
		tags := []string{}
		alpn := []string{}
		versions := []pools.TLSVersion{}
		tlsEnabled := false
		value, err := newOctaviaPools(t, cloud).Update(context.Background(), "pool-1", pools.UpdateOpts{
			Name: &empty, Description: &empty, AdminStateUp: octaviaPoolBool(false), TLSEnabled: &tlsEnabled,
			Persistence: &pools.SessionPersistence{}, Tags: &tags, ALPNProtocols: &alpn, TLSVersions: &versions,
		}, pools.WithUpdateField("vendor", 1))
		octaviaPoolWantStatus(t, tc.status, tc.success, err)
		if tc.success && (value == nil || value.ID != "pool-1") {
			t.Fatalf("value=%+v", value)
		}
	}
}

func TestOctaviaPoolDeleteAcceptsOnly202Or204AndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/pools/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{"faultstring":"gone"}`)
		case "ok-200":
			testcloud.JSON(w, 200, `{}`)
		case "ok-202":
			w.WriteHeader(202)
		default:
			w.WriteHeader(204)
		}
	})
	api := newOctaviaPools(t, cloud)
	ctx := context.Background()
	for _, id := range []string{"pool-1", "ok-202"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if err := api.Delete(ctx, "ok-200"); !errors.As(err, &unexpected) || unexpected.Actual != 200 {
		t.Fatalf("200 is not a delete success: %v", err)
	}
	if err := api.Delete(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
}

func TestOctaviaPoolGetDecodesModelAndKeepsNative404(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/lbaas/pools/pool-1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, octaviaPoolBody) })
	cloud.Mux.HandleFunc("GET /v2/lbaas/pools/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := newOctaviaPools(t, cloud)
	value, err := api.Get(context.Background(), "pool-1")
	if err != nil || value.Name != "web" || value.Protocol != "HTTP" || value.AdminStateUp || !reflect.DeepEqual(value.Tags, []string{"blue"}) {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(context.Background(), "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
}

func TestOctaviaPoolListFiltersFollowLinksAndPreserveFalse(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/pools", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "pool-1" {
			testcloud.JSON(w, 200, `{"pools":[{"id":"pool-2","name":"two"}],"pools_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/lbaas/pools?marker=pool-1&limit=1"
		testcloud.JSON(w, 200, `{"pools":[{"id":"pool-1","name":"one"}],"pools_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	var ids []string
	for value, err := range newOctaviaPools(t, cloud).List(context.Background(),
		pools.WithListOptions(pools.ListOpts{AdminStateUp: octaviaPoolBool(false), LBMethod: "ROUND_ROBIN", LoadbalancerID: "lb-1", Tags: []string{"a", "b"}, Limit: 1}),
		pools.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"pool-1", "pool-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	first := requests[0]
	if first.Get("admin_state_up") != "false" || first.Get("lb_algorithm") != "ROUND_ROBIN" || first.Get("loadbalancer_id") != "lb-1" ||
		!reflect.DeepEqual(first["tags"], []string{"a", "b"}) || first.Get("limit") != "1" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "pool-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestOctaviaPoolCommonLookupFindsExactNameAndFiltersStatusLocally(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/pools/web", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/lbaas/pools", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"pools":[{"id":"pool-1","name":"web","provisioning_status":"ACTIVE"},{"id":"pool-2","name":"web-2","provisioning_status":"ERROR"}]}`)
	})
	api := newOctaviaPools(t, cloud)
	value, err := api.Find(context.Background(), resource.Name("web"))
	if err != nil || value == nil || value.ID != "pool-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if len(queries) == 0 || queries[0].Get("name") != "web" {
		t.Fatalf("queries=%v", queries)
	}
	queries = nil
	all, err := api.All(context.Background(), resource.WithStatus("ACTIVE"))
	if err != nil || len(all) != 1 || all[0].ID != "pool-1" {
		t.Fatalf("all=%+v error=%v", all, err)
	}
	if len(queries) != 1 || queries[0].Has("status") || queries[0].Has("provisioning_status") {
		t.Fatalf("status must not be sent on the wire: %v", queries)
	}
}

func TestOctaviaPoolCreateMemberKeepsExplicitFalseZeroAndAcceptsOnly201Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/lbaas/pools/pool-1/members", func(w http.ResponseWriter, r *http.Request) {
			want := map[string]string{
				"address": `"192.0.2.10"`, "protocol_port": `8080`, "weight": `0`, "admin_state_up": `false`, "backup": `false`,
				"monitor_port": `0`, "tags": `["blue"]`, "vendor": `{"enabled":false}`,
			}
			if got := decodeOctaviaPoolRequest(t, r, "member"); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaPoolMemberBody)
		})
		value, err := newOctaviaPools(t, cloud).CreateMember(context.Background(), "pool-1", pools.CreateMemberOpts{
			Address: "192.0.2.10", ProtocolPort: 8080, Weight: octaviaPoolInt(0), AdminStateUp: octaviaPoolBool(false),
			Backup: octaviaPoolBool(false), MonitorPort: octaviaPoolInt(0), Tags: []string{"blue"},
		}, pools.WithCreateMemberField("vendor", map[string]bool{"enabled": false}))
		octaviaPoolWantStatus(t, tc.status, tc.success, err)
		if tc.success && (value == nil || value.ID != "m-1" || value.Weight != 0 || value.AdminStateUp || value.Address != "192.0.2.10") {
			t.Fatalf("value=%+v", value)
		}
	}
}

func TestOctaviaPoolUpdateMemberSendsEmptyValuesAndAcceptsOnly200To202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {201, true}, {202, true}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/pools/pool-1/members/m-1", func(w http.ResponseWriter, r *http.Request) {
			// UpdateMemberOpts.Tags is a plain slice with omitempty, so an empty
			// slice is omitted rather than sent as [].
			want := map[string]string{"name": `""`, "weight": `0`, "admin_state_up": `false`, "backup": `false`, "monitor_address": `""`, "monitor_port": `0`, "vendor": `1`}
			if got := decodeOctaviaPoolRequest(t, r, "member"); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaPoolMemberBody)
		})
		empty := ""
		value, err := newOctaviaPools(t, cloud).UpdateMember(context.Background(), "pool-1", "m-1", pools.UpdateMemberOpts{
			Name: &empty, Weight: octaviaPoolInt(0), AdminStateUp: octaviaPoolBool(false), Backup: octaviaPoolBool(false),
			MonitorAddress: &empty, MonitorPort: octaviaPoolInt(0), Tags: []string{},
		}, pools.WithUpdateMemberField("vendor", 1))
		octaviaPoolWantStatus(t, tc.status, tc.success, err)
		if tc.success && (value == nil || value.ID != "m-1") {
			t.Fatalf("value=%+v", value)
		}
	}
}

func TestOctaviaPoolGetAndDeleteMemberStatusCodesAndNative404(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/lbaas/pools/pool-1/members/m-1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, octaviaPoolMemberBody) })
	cloud.Mux.HandleFunc("GET /v2/lbaas/pools/pool-1/members/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/pools/pool-1/members/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "ok-200":
			testcloud.JSON(w, 200, `{}`)
		case "ok-202":
			w.WriteHeader(202)
		default:
			w.WriteHeader(204)
		}
	})
	api := newOctaviaPools(t, cloud)
	ctx := context.Background()
	value, err := api.GetMember(ctx, "pool-1", "m-1")
	if err != nil || value.Address != "192.0.2.10" || value.ProtocolPort != 8080 || value.Backup {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.GetMember(ctx, "pool-1", "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get member keeps the 404 response error: %v", err)
	}
	for _, id := range []string{"m-1", "ok-202"} {
		if err := api.DeleteMember(ctx, "pool-1", id); err != nil {
			t.Fatalf("delete member %s: %v", id, err)
		}
	}
	if err := api.DeleteMember(ctx, "pool-1", "ok-200"); !errors.As(err, &unexpected) || unexpected.Actual != 200 {
		t.Fatalf("200 is not a member delete success: %v", err)
	}
	if err := api.DeleteMember(ctx, "pool-1", "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native member delete keeps the 404 response error: %v", err)
	}
}

func TestOctaviaPoolListMembersFiltersFollowLinksAndPreserveFalse(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/pools/pool-1/members", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "m-1" {
			testcloud.JSON(w, 200, `{"members":[{"id":"m-2"}],"members_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/lbaas/pools/pool-1/members?marker=m-1&limit=1"
		testcloud.JSON(w, 200, `{"members":[{"id":"m-1"}],"members_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	var ids []string
	for value, err := range newOctaviaPools(t, cloud).ListMembers(context.Background(), "pool-1",
		pools.WithListMembersOptions(pools.ListMembersOpts{AdminStateUp: octaviaPoolBool(false), Address: "192.0.2.10", ProtocolPort: 8080, Limit: 1}),
		pools.WithListMembersQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"m-1", "m-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	first := requests[0]
	if first.Get("admin_state_up") != "false" || first.Get("address") != "192.0.2.10" || first.Get("protocol_port") != "8080" || first.Get("limit") != "1" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "m-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestOctaviaPoolBatchUpdateMembersSendsReplacementListAndRequires202(t *testing.T) {
	for _, status := range []int{202, 200, 204} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/pools/pool-1/members", func(w http.ResponseWriter, r *http.Request) {
			data, _ := io.ReadAll(r.Body)
			want := `{"members":[{"address":"192.0.2.10","protocol_port":8080,"name":"","weight":0,"subnet_id":null,"admin_state_up":false,"backup":false,"monitor_address":"","monitor_port":0,"tags":["blue"]}]}`
			if !jsonEqualOctaviaPool(data, want) {
				t.Errorf("body=%s", data)
			}
			w.WriteHeader(status)
		})
		empty := ""
		err := newOctaviaPools(t, cloud).BatchUpdateMembers(context.Background(), "pool-1", []pools.BatchUpdateMemberOpts{{
			Address: "192.0.2.10", ProtocolPort: 8080, Name: &empty, Weight: octaviaPoolInt(0), SubnetID: &empty,
			AdminStateUp: octaviaPoolBool(false), Backup: octaviaPoolBool(false), MonitorAddress: &empty, MonitorPort: octaviaPoolInt(0), Tags: []string{"blue"},
		}})
		octaviaPoolWantStatus(t, status, status == 202, err)
	}
}

func TestOctaviaPoolBatchUpdateMembersEmptyListClearsMembers(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("PUT /v2/lbaas/pools/pool-1/members", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if string(data) != `{"members":[]}` {
			t.Errorf("body=%s", data)
		}
		w.WriteHeader(202)
	})
	api := newOctaviaPools(t, cloud)
	if err := api.BatchUpdateMembers(context.Background(), "pool-1", nil); err != nil {
		t.Fatal(err)
	}
	if err := api.BatchUpdateMembers(context.Background(), "pool-1", []pools.BatchUpdateMemberOpts{}); err != nil {
		t.Fatal(err)
	}
}

func jsonEqualOctaviaPool(got []byte, want string) bool {
	var a, b any
	return json.Unmarshal(got, &a) == nil && json.Unmarshal([]byte(want), &b) == nil && reflect.DeepEqual(a, b)
}

func TestOctaviaPoolMemberScopeUsesExplicitParentWithoutLookup(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.Method + " " + r.URL.Path {
		case "POST /v2/lbaas/pools/pool-1/members":
			testcloud.JSON(w, 201, octaviaPoolMemberBody)
		case "PUT /v2/lbaas/pools/pool-1/members/m-1":
			testcloud.JSON(w, 200, octaviaPoolMemberBody)
		case "GET /v2/lbaas/pools/pool-1/members/m-1":
			testcloud.JSON(w, 200, octaviaPoolMemberBody)
		case "DELETE /v2/lbaas/pools/pool-1/members/missing":
			testcloud.JSON(w, 404, `{}`)
		default:
			t.Errorf("unexpected request: %s", r.Method+" "+r.URL.Path)
			http.Error(w, "unexpected", 500)
		}
	})
	ctx := context.Background()
	scope, err := newOctaviaPools(t, cloud).Members(ctx, resource.ID("pool-1"))
	if err != nil || len(paths) != 0 {
		t.Fatalf("scope error=%v paths=%v", err, paths)
	}
	if value, err := scope.Create(ctx, pools.CreateMemberOpts{Address: "192.0.2.10", ProtocolPort: 8080}); err != nil || value.ID != "m-1" {
		t.Fatalf("create value=%+v error=%v", value, err)
	}
	if value, err := scope.Update(ctx, resource.ID("m-1"), pools.UpdateMemberOpts{Weight: octaviaPoolInt(0)}); err != nil || value.ID != "m-1" {
		t.Fatalf("update value=%+v error=%v", value, err)
	}
	if value, err := scope.Find(ctx, resource.ID("m-1")); err != nil || value.ID != "m-1" {
		t.Fatalf("find value=%+v error=%v", value, err)
	}
	if err := scope.Delete(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("scope delete ignores missing by default: %v", err)
	}
}

func TestOctaviaPoolCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newOctaviaPools(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	create := pools.CreateOpts{LBMethod: pools.LBMethodRoundRobin, Protocol: pools.ProtocolHTTP}
	member := pools.CreateMemberOpts{Address: "192.0.2.10", ProtocolPort: 8080}
	checks := map[string]error{}
	_, checks["get"] = api.Get(canceled, "pool-1")
	_, checks["create"] = api.Create(canceled, create)
	_, checks["update"] = api.Update(canceled, "pool-1", pools.UpdateOpts{})
	checks["delete"] = api.Delete(canceled, "pool-1")
	_, checks["get member"] = api.GetMember(canceled, "pool-1", "m-1")
	_, checks["create member"] = api.CreateMember(canceled, "pool-1", member)
	_, checks["update member"] = api.UpdateMember(canceled, "pool-1", "m-1", pools.UpdateMemberOpts{})
	checks["delete member"] = api.DeleteMember(canceled, "pool-1", "m-1")
	checks["batch update members"] = api.BatchUpdateMembers(canceled, "pool-1", nil)
	for _, err := range api.List(canceled) {
		checks["list"] = err
	}
	for _, err := range api.ListMembers(canceled, "pool-1") {
		checks["list members"] = err
	}
	if len(checks) != 11 {
		t.Fatalf("checks=%v", checks)
	}
	for name, err := range checks {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled %s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, create); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
	deadline, stop2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop2()
	if err := api.BatchUpdateMembers(deadline, "pool-1", nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight batch deadline: %v", err)
	}
}
