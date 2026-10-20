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
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/monitors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const octaviaMonitorBody = `{"healthmonitor":{"id":"hm-1","name":"probe","type":"HTTP","delay":5,"timeout":3,"max_retries":2,"http_version":1.1,"admin_state_up":false,"provisioning_status":"PENDING_CREATE","operating_status":"OFFLINE","pools":[{"id":"pool-1"}],"tags":["blue"],"vendor_extra":1}}`

func newOctaviaMonitors(t *testing.T, cloud *testcloud.Cloud) *monitors.API {
	t.Helper()
	return monitors.New(cloud.Client("load-balancer", "/v2"))
}

func octaviaMonitorBool(value bool) *bool { return &value }

func octaviaMonitorRequest(t *testing.T, r *http.Request) map[string]string {
	t.Helper()
	var body struct {
		Monitor map[string]json.RawMessage `json:"healthmonitor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("request body: %v", err)
	}
	got := map[string]string{}
	for key, value := range body.Monitor {
		got[key] = string(value)
	}
	return got
}

func TestOctaviaMonitorCreateKeepsExplicitFalseHTTPVersionAndExtensionFields(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/lbaas/healthmonitors", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := map[string]string{"pool_id": `"pool-1"`, "type": `"HTTP"`, "delay": `5`, "timeout": `3`, "max_retries": `2`, "http_version": `1.1`, "admin_state_up": `false`, "tags": `["blue"]`, "vendor": `{"enabled":false}`}
		if got := octaviaMonitorRequest(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 201, octaviaMonitorBody)
	})
	value, err := newOctaviaMonitors(t, cloud).Create(context.Background(), monitors.CreateOpts{
		PoolID: "pool-1", Type: monitors.TypeHTTP, Delay: 5, Timeout: 3, MaxRetries: 2, HTTPVersion: "1.1",
		AdminStateUp: octaviaMonitorBool(false), Tags: []string{"blue"},
	}, monitors.WithCreateField("vendor", map[string]bool{"enabled": false}))
	if err != nil || value == nil || value.ID != "hm-1" || value.AdminStateUp || value.HTTPVersion != "1.1" || len(value.Pools) != 1 || value.Pools[0].ID != "pool-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestOctaviaMonitorCreateAcceptsOnly201Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/lbaas/healthmonitors", func(w http.ResponseWriter, r *http.Request) {
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaMonitorBody)
		})
		value, err := newOctaviaMonitors(t, cloud).Create(context.Background(), monitors.CreateOpts{Type: monitors.TypeTCP, Delay: 1, Timeout: 1, MaxRetries: 1})
		if tc.success != (err == nil) || (tc.success && (value == nil || value.ID != "hm-1")) {
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

func TestOctaviaMonitorRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newOctaviaMonitors(t, cloud)
	ctx := context.Background()
	valid := monitors.CreateOpts{Type: monitors.TypeTCP, Delay: 1, Timeout: 1, MaxRetries: 1}
	if _, err := api.Create(ctx, valid, monitors.WithCreateField("type", "PING")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected create field: %v", err)
	}
	if _, err := api.Update(ctx, "hm-1", monitors.UpdateOpts{}, monitors.WithUpdateField("admin_state_up", true)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected update field: %v", err)
	}
	if _, err := api.Create(ctx, valid, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Update(ctx, "hm-1", monitors.UpdateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil update option: %v", err)
	}
	if _, err := api.Create(ctx, monitors.CreateOpts{Delay: 1, Timeout: 1, MaxRetries: 1}); err == nil {
		t.Fatal("missing required type must fail locally")
	}
	if _, err := api.Update(ctx, "hm-1", monitors.UpdateOpts{HTTPVersion: octaviaMonitorString("not-a-number")}); err == nil {
		t.Fatal("non-numeric http_version must fail locally")
	}
	for range api.List(ctx, nil) {
		// The sequence yields one invalid-option error and no HTTP request.
	}
}

func octaviaMonitorString(value string) *string { return &value }

func TestOctaviaMonitorUpdateSendsEmptyValuesAndAcceptsOnly200Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, true}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/healthmonitors/hm-1", func(w http.ResponseWriter, r *http.Request) {
			want := map[string]string{"name": `""`, "domain_name": `""`, "admin_state_up": `false`, "http_version": `1`}
			if got := octaviaMonitorRequest(t, r); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaMonitorBody)
		})
		empty, version := "", "1.0"
		value, err := newOctaviaMonitors(t, cloud).Update(context.Background(), "hm-1", monitors.UpdateOpts{
			Name: &empty, DomainName: &empty, AdminStateUp: octaviaMonitorBool(false), HTTPVersion: &version,
		})
		if tc.success != (err == nil) || (tc.success && (value == nil || value.ID != "hm-1")) {
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

func TestOctaviaMonitorUpdateWithNoFieldsSendsEmptyEnvelopeAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]string
	cloud.Mux.HandleFunc("PUT /v2/lbaas/healthmonitors/hm-1", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, octaviaMonitorRequest(t, r))
		testcloud.JSON(w, 200, octaviaMonitorBody)
	})
	api := newOctaviaMonitors(t, cloud)
	if _, err := api.Update(context.Background(), "hm-1", monitors.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(context.Background(), "hm-1", monitors.UpdateOpts{Delay: 9}, monitors.WithUpdateField("vendor", nil)); err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{{}, {"delay": `9`, "vendor": `null`}}
	if !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies=%v", bodies)
	}
}

func TestOctaviaMonitorDeleteAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/healthmonitors/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{"faultstring":"gone"}`)
		case "ok200":
			testcloud.JSON(w, 200, `{}`)
		case "ok202":
			w.WriteHeader(202)
		default:
			w.WriteHeader(204)
		}
	})
	api := newOctaviaMonitors(t, cloud)
	ctx := context.Background()
	for _, id := range []string{"hm-1", "ok202"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if err := api.Delete(ctx, "ok200"); !errors.As(err, &unexpected) || unexpected.Actual != 200 {
		t.Fatalf("200 is not a delete success code: %v", err)
	}
	if err := api.Delete(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
}

func TestOctaviaMonitorGetDecodesModelAndKeepsNative404(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/lbaas/healthmonitors/hm-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, octaviaMonitorBody)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/healthmonitors/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	api := newOctaviaMonitors(t, cloud)
	value, err := api.Get(context.Background(), "hm-1")
	if err != nil || value.Name != "probe" || value.Type != "HTTP" || value.Delay != 5 || value.HTTPVersion != "1.1" || !reflect.DeepEqual(value.Tags, []string{"blue"}) || value.OperatingStatus != "OFFLINE" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(context.Background(), "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
}

func TestOctaviaMonitorListFiltersFollowLinksAndPreserveFalse(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/healthmonitors", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "hm-1" {
			testcloud.JSON(w, 200, `{"healthmonitors":[{"id":"hm-2","name":"two"}],"healthmonitors_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/lbaas/healthmonitors?marker=hm-1&limit=1"
		testcloud.JSON(w, 200, `{"healthmonitors":[{"id":"hm-1","name":"one"}],"healthmonitors_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newOctaviaMonitors(t, cloud)
	var ids []string
	for value, err := range api.List(context.Background(),
		monitors.WithListOptions(monitors.ListOpts{PoolID: "pool-1", AdminStateUp: octaviaMonitorBool(false), Tags: []string{"a", "b"}, Limit: 1}),
		monitors.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"hm-1", "hm-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	first := requests[0]
	if first.Get("admin_state_up") != "false" || first.Get("pool_id") != "pool-1" || !reflect.DeepEqual(first["tags"], []string{"a", "b"}) || first.Get("limit") != "1" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "hm-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestOctaviaMonitorCommonLookupFindsExactNameAndMapsNothingElse(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/healthmonitors/probe", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/healthmonitors", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"healthmonitors":[{"id":"hm-1","name":"probe"},{"id":"hm-2","name":"probe-2"}]}`)
	})
	api := newOctaviaMonitors(t, cloud)
	value, err := api.Find(context.Background(), resource.Name("probe"))
	if err != nil || value == nil || value.ID != "hm-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if len(queries) == 0 || queries[0].Get("name") != "probe" {
		t.Fatalf("queries=%v", queries)
	}
	queries = nil
	if _, err := api.All(context.Background(), resource.WithQuery("vendor", "x")); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 || queries[0].Get("vendor") != "x" {
		t.Fatalf("queries=%v", queries)
	}
}

func TestOctaviaMonitorCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := newOctaviaMonitors(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	valid := monitors.CreateOpts{Type: monitors.TypeTCP, Delay: 1, Timeout: 1, MaxRetries: 1}
	if _, err := api.Get(canceled, "hm-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if _, err := api.Create(canceled, valid); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Update(canceled, "hm-1", monitors.UpdateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled update: %v", err)
	}
	if err := api.Delete(canceled, "hm-1"); !errors.Is(err, context.Canceled) {
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
	if _, err := api.Create(deadline, valid); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
