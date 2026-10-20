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
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/loadbalancers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const octaviaLoadBalancerBody = `{"loadbalancer":{"id":"lb-1","name":"web","admin_state_up":false,"provisioning_status":"PENDING_CREATE","operating_status":"OFFLINE","vip_address":"192.0.2.10","vip_subnet_id":"subnet-1","tags":["blue"],"created_at":"2026-01-02T03:04:05","vendor_extra":1}}`

func newOctaviaLoadBalancers(t *testing.T, cloud *testcloud.Cloud) *loadbalancers.API {
	t.Helper()
	return loadbalancers.New(cloud.Client("load-balancer", "/v2"))
}

func octaviaLoadBalancerFalse() *bool { value := false; return &value }

func decodeOctaviaLoadBalancerRequest(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	var body struct {
		LoadBalancer map[string]json.RawMessage `json:"loadbalancer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("request body: %v", err)
	}
	return body.LoadBalancer
}

func TestOctaviaLoadBalancerCreateKeepsExplicitFalseAndExtensionFields(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body := decodeOctaviaLoadBalancerRequest(t, r)
		want := map[string]string{"name": `"web"`, "admin_state_up": `false`, "vip_subnet_id": `"subnet-1"`, "tags": `["blue"]`, "vendor": `{"enabled":false}`}
		got := map[string]string{}
		for key, value := range body {
			got[key] = string(value)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		w.Header().Set("X-Openstack-Request-Id", "create-lb")
		testcloud.JSON(w, 201, octaviaLoadBalancerBody)
	})
	value, err := newOctaviaLoadBalancers(t, cloud).Create(context.Background(), loadbalancers.CreateOpts{
		Name: "web", AdminStateUp: octaviaLoadBalancerFalse(), VipSubnetID: "subnet-1", Tags: []string{"blue"},
	}, loadbalancers.WithCreateField("vendor", map[string]bool{"enabled": false}))
	if err != nil || value == nil || value.ID != "lb-1" || value.AdminStateUp || value.ProvisioningStatus != "PENDING_CREATE" || value.CreatedAt.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestOctaviaLoadBalancerRejectsProtectedOrUnsupportedExtensionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newOctaviaLoadBalancers(t, cloud)
	ctx := context.Background()
	if _, err := api.Create(ctx, loadbalancers.CreateOpts{VipSubnetID: "subnet-1"}, loadbalancers.WithCreateField("name", "shadow")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected create field: %v", err)
	}
	if _, err := api.Update(ctx, "lb-1", loadbalancers.UpdateOpts{}, loadbalancers.WithUpdateField("admin_state_up", true)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected update field: %v", err)
	}
	if _, err := api.Create(ctx, loadbalancers.CreateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if err := api.Delete(ctx, "lb-1", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil delete option: %v", err)
	}
	for range api.List(ctx, nil) {
		// The sequence yields one invalid-option error and no HTTP request.
	}
}

func TestOctaviaLoadBalancerUpdateSendsEmptyValuesAndAcceptsOnly200Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, true}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/loadbalancers/lb-1", func(w http.ResponseWriter, r *http.Request) {
			body := decodeOctaviaLoadBalancerRequest(t, r)
			if len(body) != 4 || string(body["name"]) != `""` || string(body["description"]) != `""` || string(body["admin_state_up"]) != `false` || string(body["tags"]) != `[]` {
				t.Errorf("body=%v", body)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaLoadBalancerBody)
		})
		empty := ""
		tags := []string{}
		value, err := newOctaviaLoadBalancers(t, cloud).Update(context.Background(), "lb-1", loadbalancers.UpdateOpts{
			Name: &empty, Description: &empty, AdminStateUp: octaviaLoadBalancerFalse(), Tags: &tags,
		})
		if tc.success != (err == nil) || (tc.success && (value == nil || value.ID != "lb-1")) {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			var unexpected gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &unexpected) || unexpected.Actual != tc.status {
				t.Fatalf("status %d error=%v", tc.status, err)
			}
		}
	}
}

func TestOctaviaLoadBalancerDeleteCascadeQueryAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/loadbalancers/{id}", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{"faultstring":"gone"}`)
			return
		}
		w.WriteHeader(204)
	})
	api := newOctaviaLoadBalancers(t, cloud)
	ctx := context.Background()
	if err := api.Delete(ctx, "lb-1", loadbalancers.WithDeleteOptions(loadbalancers.DeleteOpts{Cascade: true}), loadbalancers.WithDeleteQuery("vendor", "x")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "lb-2"); err != nil {
		t.Fatal(err)
	}
	if want := []url.Values{{"cascade": {"true"}, "vendor": {"x"}}, {}}; !reflect.DeepEqual(queries, want) {
		t.Fatalf("queries=%v", queries)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if err := api.Delete(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
}

func TestOctaviaLoadBalancerFailoverSendsNoBodyAndRequires202(t *testing.T) {
	for _, status := range []int{202, 200, 204} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/loadbalancers/lb-1/failover", func(w http.ResponseWriter, r *http.Request) {
			if data, _ := io.ReadAll(r.Body); len(data) != 0 {
				t.Errorf("failover body=%q", data)
			}
			w.WriteHeader(status)
		})
		err := newOctaviaLoadBalancers(t, cloud).Failover(context.Background(), "lb-1")
		if (status == 202) != (err == nil) {
			t.Fatalf("status %d error=%v", status, err)
		}
	}
}

func TestOctaviaLoadBalancerReadsStatsAndStatusTree(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers/lb-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, octaviaLoadBalancerBody)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers/lb-1/stats", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"stats":{"active_connections":1,"bytes_in":2,"bytes_out":3,"request_errors":4,"total_connections":5}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers/lb-1/status", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"statuses":{"loadbalancer":{"id":"lb-1","name":"web","provisioning_status":"ACTIVE","operating_status":"ONLINE"}}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	api := newOctaviaLoadBalancers(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "lb-1")
	if err != nil || value.Name != "web" || value.VipAddress != "192.0.2.10" || !reflect.DeepEqual(value.Tags, []string{"blue"}) {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	stats, err := api.GetStats(ctx, "lb-1")
	if err != nil || *stats != (loadbalancers.Stats{ActiveConnections: 1, BytesIn: 2, BytesOut: 3, RequestErrors: 4, TotalConnections: 5}) {
		t.Fatalf("stats=%+v error=%v", stats, err)
	}
	tree, err := api.GetStatuses(ctx, "lb-1")
	if err != nil || tree.Loadbalancer == nil || tree.Loadbalancer.ProvisioningStatus != "ACTIVE" || tree.Loadbalancer.OperatingStatus != "ONLINE" {
		t.Fatalf("tree=%+v error=%v", tree, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
}

func TestOctaviaLoadBalancerListFiltersFollowLinksAndPreserveFalse(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "lb-1" {
			testcloud.JSON(w, 200, `{"loadbalancers":[{"id":"lb-2","name":"two"}],"loadbalancers_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/lbaas/loadbalancers?marker=lb-1&limit=1"
		testcloud.JSON(w, 200, `{"loadbalancers":[{"id":"lb-1","name":"one"}],"loadbalancers_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newOctaviaLoadBalancers(t, cloud)
	var ids []string
	for value, err := range api.List(context.Background(),
		loadbalancers.WithListOptions(loadbalancers.ListOpts{AdminStateUp: octaviaLoadBalancerFalse(), Tags: []string{"a", "b"}, Limit: 1}),
		loadbalancers.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"lb-1", "lb-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	first := requests[0]
	if first.Get("admin_state_up") != "false" || !reflect.DeepEqual(first["tags"], []string{"a", "b"}) || first.Get("limit") != "1" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "lb-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestOctaviaLoadBalancerCommonLookupMapsStatusAndFindsExactName(t *testing.T) {
	cloud := testcloud.New(t)
	var statusQueries []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers/web", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		statusQueries = append(statusQueries, r.URL.Query())
		testcloud.JSON(w, 200, `{"loadbalancers":[{"id":"lb-1","name":"web"},{"id":"lb-2","name":"web-2"}]}`)
	})
	api := newOctaviaLoadBalancers(t, cloud)
	value, err := api.Find(context.Background(), resource.Name("web"))
	if err != nil || value == nil || value.ID != "lb-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if len(statusQueries) == 0 || statusQueries[0].Get("name") != "web" {
		t.Fatalf("queries=%v", statusQueries)
	}
	statusQueries = nil
	if _, err := api.All(context.Background(), resource.WithStatus("ACTIVE")); err != nil {
		t.Fatal(err)
	}
	if len(statusQueries) != 1 || statusQueries[0].Get("provisioning_status") != "ACTIVE" || statusQueries[0].Has("status") {
		t.Fatalf("status was not mapped to provisioning_status: %v", statusQueries)
	}
}

func TestOctaviaLoadBalancerCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := newOctaviaLoadBalancers(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "lb-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if err := api.Failover(canceled, "lb-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled failover: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, loadbalancers.CreateOpts{VipSubnetID: "subnet-1"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
