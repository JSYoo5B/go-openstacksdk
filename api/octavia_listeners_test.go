package api_test

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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/listeners"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const octaviaListenerBody = `{"listener":{"id":"ls-1","name":"front","protocol":"HTTP","protocol_port":80,"admin_state_up":false,"connection_limit":0,"provisioning_status":"PENDING_CREATE","operating_status":"OFFLINE","tags":["blue"],"vendor_extra":1}}`

func newOctaviaListeners(t *testing.T, cloud *testcloud.Cloud) *listeners.API {
	t.Helper()
	return listeners.New(cloud.Client("load-balancer", "/v2"))
}

func octaviaListenerBool(value bool) *bool { return &value }
func octaviaListenerInt(value int) *int    { return &value }

func decodeOctaviaListenerRequest(t *testing.T, r *http.Request) map[string]string {
	t.Helper()
	var body struct {
		Listener map[string]json.RawMessage `json:"listener"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("request body: %v", err)
	}
	got := map[string]string{}
	for key, value := range body.Listener {
		got[key] = string(value)
	}
	return got
}

func TestOctaviaListenerCreateKeepsExplicitFalseZeroAndExtensionFields(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/lbaas/listeners", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			want := map[string]string{
				"protocol": `"HTTP"`, "protocol_port": `80`, "loadbalancer_id": `"lb-1"`, "admin_state_up": `false`,
				"connection_limit": `0`, "timeout_client_data": `0`, "tags": `["blue"]`, "vendor": `{"enabled":false}`,
			}
			if got := decodeOctaviaListenerRequest(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaListenerBody)
		})
		value, err := newOctaviaListeners(t, cloud).Create(context.Background(), listeners.CreateOpts{
			LoadbalancerID: "lb-1", Protocol: listeners.ProtocolHTTP, ProtocolPort: 80,
			AdminStateUp: octaviaListenerBool(false), ConnLimit: octaviaListenerInt(0), TimeoutClientData: octaviaListenerInt(0),
			Tags: []string{"blue"},
		}, listeners.WithCreateField("vendor", map[string]bool{"enabled": false}))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d value=%+v error=%v calls=%d", tc.status, value, err, calls.Load())
		}
		if tc.success && (value == nil || value.ID != "ls-1" || value.AdminStateUp || value.ProvisioningStatus != "PENDING_CREATE") {
			t.Fatalf("value=%+v", value)
		}
		if !tc.success {
			var unexpected gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &unexpected) || unexpected.Actual != tc.status {
				t.Fatalf("status %d error=%v", tc.status, err)
			}
		}
	}
}

func TestOctaviaListenerRejectsProtectedOrUnsupportedExtensionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newOctaviaListeners(t, cloud)
	ctx := context.Background()
	create := listeners.CreateOpts{Protocol: listeners.ProtocolHTTP, ProtocolPort: 80}
	if _, err := api.Create(ctx, create, listeners.WithCreateField("protocol", "TCP")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected create field: %v", err)
	}
	if _, err := api.Update(ctx, "ls-1", listeners.UpdateOpts{}, listeners.WithUpdateField("admin_state_up", true)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected update field: %v", err)
	}
	if _, err := api.Create(ctx, create, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Update(ctx, "ls-1", listeners.UpdateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil update option: %v", err)
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
}

func TestOctaviaListenerUpdateSendsEmptyValuesAndNullsAndAcceptsOnly200Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, true}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/listeners/ls-1", func(w http.ResponseWriter, r *http.Request) {
			want := map[string]string{
				"name": `""`, "description": `""`, "admin_state_up": `false`, "connection_limit": `0`, "tags": `[]`,
				"default_pool_id": `null`, "alpn_protocols": `null`, "tls_versions": `null`, "vendor": `1`,
			}
			if got := decodeOctaviaListenerRequest(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaListenerBody)
		})
		empty := ""
		tags := []string{}
		alpn := []string{}
		tlsVersions := []listeners.TLSVersion{}
		value, err := newOctaviaListeners(t, cloud).Update(context.Background(), "ls-1", listeners.UpdateOpts{
			Name: &empty, Description: &empty, DefaultPoolID: &empty, AdminStateUp: octaviaListenerBool(false),
			ConnLimit: octaviaListenerInt(0), Tags: &tags, ALPNProtocols: &alpn, TLSVersions: &tlsVersions,
		}, listeners.WithUpdateField("vendor", 1))
		if tc.success != (err == nil) || (tc.success && (value == nil || value.ID != "ls-1")) {
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

func TestOctaviaListenerDeleteAcceptsOnly202Or204AndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/listeners/{id}", func(w http.ResponseWriter, r *http.Request) {
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
	api := newOctaviaListeners(t, cloud)
	ctx := context.Background()
	for _, id := range []string{"ls-1", "ok-202"} {
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

func TestOctaviaListenerGetAndStatsDecodeModelsAndKeepNative404(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/lbaas/listeners/ls-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, octaviaListenerBody)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/listeners/ls-1/stats", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"stats":{"active_connections":1,"bytes_in":2,"bytes_out":3,"request_errors":4,"total_connections":5}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/listeners/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/lbaas/listeners/missing/stats", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := newOctaviaListeners(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "ls-1")
	if err != nil || value.Name != "front" || value.ProtocolPort != 80 || value.AdminStateUp || !reflect.DeepEqual(value.Tags, []string{"blue"}) {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	stats, err := api.GetStats(ctx, "ls-1")
	if err != nil || *stats != (listeners.Stats{ActiveConnections: 1, BytesIn: 2, BytesOut: 3, RequestErrors: 4, TotalConnections: 5}) {
		t.Fatalf("stats=%+v error=%v", stats, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if _, err := api.GetStats(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native stats keeps the 404 response error: %v", err)
	}
}

func TestOctaviaListenerListFiltersFollowLinksAndPreserveFalseAndZero(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/listeners", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "ls-1" {
			testcloud.JSON(w, 200, `{"listeners":[{"id":"ls-2","name":"two"}],"listeners_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/lbaas/listeners?marker=ls-1&limit=1"
		testcloud.JSON(w, 200, `{"listeners":[{"id":"ls-1","name":"one"}],"listeners_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	var ids []string
	for value, err := range newOctaviaListeners(t, cloud).List(context.Background(),
		listeners.WithListOptions(listeners.ListOpts{
			AdminStateUp: octaviaListenerBool(false), TimeoutClientData: octaviaListenerInt(0),
			LoadbalancerID: "lb-1", Protocol: "HTTP", ProtocolPort: 80, Tags: []string{"a", "b"}, Limit: 1,
		}),
		listeners.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"ls-1", "ls-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	first := requests[0]
	if first.Get("admin_state_up") != "false" || first.Get("timeout_client_data") != "0" || first.Get("loadbalancer_id") != "lb-1" ||
		first.Get("protocol") != "HTTP" || first.Get("protocol_port") != "80" || !reflect.DeepEqual(first["tags"], []string{"a", "b"}) ||
		first.Get("limit") != "1" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "ls-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestOctaviaListenerCommonLookupFindsExactNameAndFiltersStatusLocally(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/listeners/front", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/lbaas/listeners", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"listeners":[{"id":"ls-1","name":"front","provisioning_status":"ACTIVE"},{"id":"ls-2","name":"front-2","provisioning_status":"ERROR"}]}`)
	})
	api := newOctaviaListeners(t, cloud)
	value, err := api.Find(context.Background(), resource.Name("front"))
	if err != nil || value == nil || value.ID != "ls-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if len(queries) == 0 || queries[0].Get("name") != "front" {
		t.Fatalf("queries=%v", queries)
	}
	queries = nil
	all, err := api.All(context.Background(), resource.WithStatus("ACTIVE"))
	if err != nil || len(all) != 1 || all[0].ID != "ls-1" {
		t.Fatalf("all=%+v error=%v", all, err)
	}
	if len(queries) != 1 || queries[0].Has("status") || queries[0].Has("provisioning_status") {
		t.Fatalf("status must not be sent on the wire: %v", queries)
	}
}

func TestOctaviaListenerCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := newOctaviaListeners(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	create := listeners.CreateOpts{Protocol: listeners.ProtocolHTTP, ProtocolPort: 80}
	if _, err := api.Get(canceled, "ls-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if _, err := api.GetStats(canceled, "ls-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled stats: %v", err)
	}
	if _, err := api.Create(canceled, create); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Update(canceled, "ls-1", listeners.UpdateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled update: %v", err)
	}
	if err := api.Delete(canceled, "ls-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
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
}
