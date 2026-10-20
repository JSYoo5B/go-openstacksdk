package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/amphorae"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/apiversions"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/providers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const octaviaAmphoraBody = `{"amphora":{"id":"amp-1","loadbalancer_id":"lb-1","lb_network_ip":"192.0.2.5","compute_id":"srv-1","ha_ip":"192.0.2.9","role":"MASTER","status":"ALLOCATED","cert_busy":false,"vrrp_id":1,"vrrp_priority":100,"cached_zone":"az1","image_id":"img-1","cert_expiration":"2026-03-04T05:06:07","created_at":"2026-01-02T03:04:05","updated_at":"2026-01-02T03:04:06","vendor_extra":1}}`

func newOctaviaAmphorae(t *testing.T, cloud *testcloud.Cloud) *amphorae.API {
	t.Helper()
	return amphorae.New(cloud.Client("load-balancer", "/v2"))
}

func newOctaviaProviders(t *testing.T, cloud *testcloud.Cloud) *providers.API {
	t.Helper()
	return providers.New(cloud.Client("load-balancer", "/v2"))
}

func octaviaAmphoraUnexpected(t *testing.T, err error, status int) {
	t.Helper()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &unexpected) || unexpected.Actual != status {
		t.Fatalf("want unexpected status %d, got %v", status, err)
	}
}

func TestOctaviaAmphoraGetDecodesTimestampsAndKeepsNative404(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/octavia/amphorae/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, octaviaAmphoraBody)
	})
	api := newOctaviaAmphorae(t, cloud)
	value, err := api.Get(context.Background(), "amp-1")
	if err != nil || value.LoadbalancerID != "lb-1" || value.Role != "MASTER" || value.Status != "ALLOCATED" || value.CertBusy || value.VRRPPriority != 100 ||
		value.CertExpiration.IsZero() || value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() || value.CachedZone != "az1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(context.Background(), "missing")
	octaviaAmphoraUnexpected(t, err, 404)
}

func TestOctaviaAmphoraFailoverSendsNoBodyAndRequires202(t *testing.T) {
	for _, status := range []int{202, 200, 204} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("PUT /v2/octavia/amphorae/amp-1/failover", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if data, _ := io.ReadAll(r.Body); len(data) != 0 {
				t.Errorf("failover body=%q", data)
			}
			w.WriteHeader(status)
		})
		err := newOctaviaAmphorae(t, cloud).Failover(context.Background(), "amp-1")
		if (status == 202) != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d calls=%d error=%v", status, calls.Load(), err)
		}
		if status != 202 {
			octaviaAmphoraUnexpected(t, err, status)
		}
	}
}

func TestOctaviaAmphoraListFiltersFollowLinksAndExtensionQuery(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/octavia/amphorae", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "amp-1" {
			testcloud.JSON(w, 200, `{"amphorae":[{"id":"amp-2","status":"BOOTING"}],"amphorae_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/octavia/amphorae?marker=amp-1&limit=1"
		testcloud.JSON(w, 200, `{"amphorae":[{"id":"amp-1","status":"ALLOCATED"}],"amphorae_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newOctaviaAmphorae(t, cloud)
	var ids []string
	for value, err := range api.List(context.Background(),
		amphorae.WithListOptions(amphorae.ListOpts{LoadbalancerID: "lb-1", Role: "MASTER", Status: "ALLOCATED", Limit: 1}),
		amphorae.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"amp-1", "amp-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	if first := requests[0]; first.Get("loadbalancer_id") != "lb-1" || first.Get("role") != "MASTER" || first.Get("status") != "ALLOCATED" || first.Get("limit") != "1" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "amp-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestOctaviaAmphoraNilListOptionFailsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	seen := 0
	for _, err := range newOctaviaAmphorae(t, cloud).List(context.Background(), nil) {
		seen++
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
	if seen != 1 {
		t.Fatalf("seen=%d", seen)
	}
}

func TestOctaviaAmphoraCommonCollectionPassesStatusAndQueryThrough(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/octavia/amphorae", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"amphorae":[{"id":"amp-1","status":"ALLOCATED"},{"id":"amp-2","status":"BOOTING"}]}`)
	})
	rows, err := newOctaviaAmphorae(t, cloud).All(context.Background(), resource.WithQuery("vendor", "x"))
	if err != nil || len(rows) != 2 || len(queries) != 1 || queries[0].Get("vendor") != "x" {
		t.Fatalf("rows=%v error=%v queries=%v", rows, err, queries)
	}
}

func TestOctaviaProviderListFieldsFollowLinksAndExtensionQuery(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/providers", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "amphora" {
			testcloud.JSON(w, 200, `{"providers":[{"name":"ovn","description":"o"}],"providers_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/lbaas/providers?marker=amphora"
		testcloud.JSON(w, 200, `{"providers":[{"name":"amphora","description":"a"}],"providers_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	var names []string
	for value, err := range newOctaviaProviders(t, cloud).List(context.Background(),
		providers.WithListOptions(providers.ListOpts{Fields: []string{"name", "description"}}), providers.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name)
	}
	if !reflect.DeepEqual(names, []string{"amphora", "ovn"}) || len(requests) != 2 {
		t.Fatalf("names=%v requests=%v", names, requests)
	}
	if first := requests[0]; !reflect.DeepEqual(first["fields"], []string{"name", "description"}) || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
}

func TestOctaviaProviderCapabilitiesUseProviderPathFieldsAndExtensionQuery(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/providers/amphora/flavor_capabilities", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"flavor_capabilities":[{"name":"loadbalancer_topology","description":"t"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/providers/amphora/availability_zone_capabilities", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"availability_zone_capabilities":[{"name":"compute_zone","description":"z"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/providers/missing/flavor_capabilities", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/providers/missing/availability_zone_capabilities", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	api := newOctaviaProviders(t, cloud)
	ctx := context.Background()
	flavorCaps, err := api.ListFlavorCapabilities(ctx, "amphora",
		providers.WithListFlavorCapabilitiesOptions(providers.ListFlavorCapabilitiesOpts{Fields: []string{"name"}}), providers.WithListFlavorCapabilitiesQuery("vendor", "x"))
	if err != nil || len(flavorCaps) != 1 || flavorCaps[0].Name != "loadbalancer_topology" || flavorCaps[0].Description != "t" {
		t.Fatalf("caps=%+v error=%v", flavorCaps, err)
	}
	zoneCaps, err := api.ListAvailabilityZoneCapabilities(ctx, "amphora",
		providers.WithListAvailabilityZoneCapabilitiesOptions(providers.ListAvailabilityZoneCapabilitiesOpts{Fields: []string{"name"}}), providers.WithListAvailabilityZoneCapabilitiesQuery("vendor", "y"))
	if err != nil || len(zoneCaps) != 1 || zoneCaps[0].Name != "compute_zone" {
		t.Fatalf("caps=%+v error=%v", zoneCaps, err)
	}
	if len(queries) != 2 || !reflect.DeepEqual(queries[0], url.Values{"fields": {"name"}, "vendor": {"x"}}) || !reflect.DeepEqual(queries[1], url.Values{"fields": {"name"}, "vendor": {"y"}}) {
		t.Fatalf("queries=%v", queries)
	}
	_, err = api.ListFlavorCapabilities(ctx, "missing")
	octaviaAmphoraUnexpected(t, err, 404)
	_, err = api.ListAvailabilityZoneCapabilities(ctx, "missing")
	octaviaAmphoraUnexpected(t, err, 404)
}

func TestOctaviaProviderRejectsNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newOctaviaProviders(t, cloud)
	ctx := context.Background()
	if _, err := api.ListFlavorCapabilities(ctx, "amphora", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil flavor capabilities option: %v", err)
	}
	if _, err := api.ListAvailabilityZoneCapabilities(ctx, "amphora", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil availability zone capabilities option: %v", err)
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
}

func TestOctaviaAPIVersionListReadsDiscoveryDocumentFromServiceRoot(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	cloud.Mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		testcloud.JSON(w, 200, `{"versions":[{"id":"v2.0","status":"SUPPORTED"},{"id":"v2.1","status":"CURRENT"}]}`)
	})
	var got []apiversions.APIVersion
	for value, err := range apiversions.New(cloud.Client("load-balancer", "/v2")).List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, *value)
	}
	want := []apiversions.APIVersion{{ID: "v2.0", Status: "SUPPORTED"}, {ID: "v2.1", Status: "CURRENT"}}
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(paths, []string{"/"}) {
		t.Fatalf("versions=%v paths=%v", got, paths)
	}
}

func TestOctaviaAPIVersionListHandlesEmptyAndErrorResponses(t *testing.T) {
	cloud := testcloud.New(t)
	status := 200
	cloud.Mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if status == 204 {
			w.WriteHeader(204)
			return
		}
		testcloud.JSON(w, status, `{"versions":[]}`)
	})
	api := apiversions.New(cloud.Client("load-balancer", "/v2"))
	for _, code := range []int{200, 204} {
		status = code
		count := 0
		for _, err := range api.List(context.Background()) {
			count++
			if err != nil {
				t.Fatalf("status %d: %v", code, err)
			}
		}
		if count != 0 {
			t.Fatalf("status %d yielded %d rows", code, count)
		}
	}
	status = 500
	for _, err := range api.List(context.Background()) {
		octaviaAmphoraUnexpected(t, err, 500)
	}
}

func TestOctaviaAmphoraProviderAPIVersionCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	am, pr := newOctaviaAmphorae(t, cloud), newOctaviaProviders(t, cloud)
	av := apiversions.New(cloud.Client("load-balancer", "/v2"))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	errs := map[string]error{}
	_, errs["amphora get"] = am.Get(canceled, "amp-1")
	errs["amphora failover"] = am.Failover(canceled, "amp-1")
	_, errs["flavor capabilities"] = pr.ListFlavorCapabilities(canceled, "amphora")
	_, errs["zone capabilities"] = pr.ListAvailabilityZoneCapabilities(canceled, "amphora")
	for _, err := range am.List(canceled) {
		errs["amphora list"] = err
	}
	for _, err := range pr.List(canceled) {
		errs["provider list"] = err
	}
	for _, err := range av.List(canceled) {
		errs["apiversion list"] = err
	}
	for name, err := range errs {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled %s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := am.Failover(deadline, "amp-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
