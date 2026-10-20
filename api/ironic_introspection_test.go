package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/introspection"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const ironicInspectorVersionHeader = "X-OpenStack-Ironic-Inspector-API-Version"

func newIronicIntrospection(t *testing.T, cloud *testcloud.Cloud, version string) *introspection.API {
	t.Helper()
	return introspection.New(ironicClient(cloud, "baremetal-introspection", version))
}

func TestIronicIntrospectionGetStatusParsesTimesAndKeepsFalseAndNull(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/introspection/{id}", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(ironicInspectorVersionHeader); got != "1.13" {
			t.Errorf("inspector microversion header=%q", got)
		}
		if got := r.Header.Get(ironicAPIVersionHeader); got != "" {
			t.Errorf("Ironic header must not be sent to the inspector: %q", got)
		}
		switch r.PathValue("id") {
		case "node-1":
			testcloud.JSON(w, 200, `{"uuid":"node-1","finished":false,"state":"waiting","error":null,"started_at":"2026-01-02T03:04:05","finished_at":null,"links":[]}`)
		case "accepted":
			testcloud.JSON(w, 202, `{}`)
		default:
			testcloud.JSON(w, 404, `{"error":{"message":"gone"}}`)
		}
	})
	api := newIronicIntrospection(t, cloud, "1.13")
	value, err := api.GetIntrospectionStatus(context.Background(), "node-1")
	if err != nil || value.UUID != "node-1" || value.Finished || value.State != "waiting" || value.Error != "" || value.StartedAt.IsZero() || !value.FinishedAt.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.GetIntrospectionStatus(context.Background(), "accepted")
	ironicRequireStatus(t, err, 202)
	_, err = api.GetIntrospectionStatus(context.Background(), "missing")
	ironicRequireStatus(t, err, 404)
	if _, err := api.Resources.Find(context.Background(), resource.ID("missing")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Collection lookup adds ErrNotFound: %v", err)
	}
}

func TestIronicIntrospectionGetDataDecodesDocumentAndRequires200(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/introspection/{id}/data", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "accepted" {
			testcloud.JSON(w, 202, `{}`)
			return
		}
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, `{"cpu_arch":"x86_64","cpus":8,"memory_mb":16384,"local_gb":100,"macs":["52:54:00:aa:bb:cc"],"boot_interface":"52:54:00:aa:bb:cc","error":null,"ipmi_address":"192.0.2.9","interfaces":{"eth0":{"mac":"52:54:00:aa:bb:cc","ip":"192.0.2.5","pxe":false}}}`)
	})
	api := newIronicIntrospection(t, cloud, "")
	data, err := api.GetIntrospectionData(context.Background(), "node-1")
	if err != nil || data.CPUArch != "x86_64" || data.CPUs != 8 || data.MemoryMB != 16384 || data.Error != "" || !reflect.DeepEqual(data.MACs, []string{"52:54:00:aa:bb:cc"}) || data.IPMIAddress != "192.0.2.9" {
		t.Fatalf("data=%+v error=%v", data, err)
	}
	if iface, ok := data.Interfaces["eth0"]; !ok || iface.MAC != "52:54:00:aa:bb:cc" || iface.PXE {
		t.Fatalf("interfaces=%+v", data.Interfaces)
	}
	_, err = api.GetIntrospectionData(context.Background(), "accepted")
	ironicRequireStatus(t, err, 202)
	_, err = api.GetIntrospectionData(context.Background(), "missing")
	ironicRequireStatus(t, err, 404)
}

func TestIronicIntrospectionAbortAndReApplyRequire202AndSendNoBody(t *testing.T) {
	type call func(*introspection.API) error
	for _, tc := range []struct {
		name string
		path string
		do   call
	}{
		{"abort", "/v1/introspection/node-1/abort", func(a *introspection.API) error { return a.AbortIntrospection(context.Background(), "node-1") }},
		{"reapply", "/v1/introspection/node-1/data/unprocessed", func(a *introspection.API) error {
			return a.ReApplyIntrospection(context.Background(), "node-1")
		}},
	} {
		for _, status := range []int{202, 200, 204, 409} {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc(tc.path, func(w http.ResponseWriter, r *http.Request) {
				data, _ := io.ReadAll(r.Body)
				if r.Method != http.MethodPost || len(data) != 0 || r.URL.RawQuery != "" {
					t.Errorf("%s: method=%s body=%q query=%q", tc.name, r.Method, data, r.URL.RawQuery)
				}
				if got := r.Header.Get(ironicInspectorVersionHeader); got != "1.13" {
					t.Errorf("inspector microversion header=%q", got)
				}
				if status == 204 || status == 202 {
					w.WriteHeader(status)
					return
				}
				testcloud.JSON(w, status, `{"error":{"message":"x"}}`)
			})
			err := tc.do(newIronicIntrospection(t, cloud, "1.13"))
			if (status == 202) != (err == nil) {
				t.Fatalf("%s status %d error=%v", tc.name, status, err)
			}
			if status != 202 {
				ironicRequireStatus(t, err, status)
			}
		}
	}
}

func TestIronicIntrospectionListFiltersFollowLinksAndExtensionQuery(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v1/introspection", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "node-1" {
			testcloud.JSON(w, 200, `{"introspection":[{"uuid":"node-2","finished":true,"state":"finished","error":null}]}`)
			return
		}
		next := cloud.Server.URL + "/v1/introspection?marker=node-1&limit=1"
		testcloud.JSON(w, 200, `{"introspection":[{"uuid":"node-1","finished":false,"state":"processing"}],"introspection_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newIronicIntrospection(t, cloud, "")
	var seen []string
	for value, err := range api.ListIntrospections(context.Background(),
		introspection.WithListIntrospectionsOptions(introspection.ListIntrospectionsOpts{Limit: 1}),
		introspection.WithListIntrospectionsQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		seen = append(seen, value.UUID+":"+value.State)
	}
	if !reflect.DeepEqual(seen, []string{"node-1:processing", "node-2:finished"}) || len(requests) != 2 {
		t.Fatalf("seen=%v requests=%v", seen, requests)
	}
	if want := (url.Values{"limit": {"1"}, "vendor": {"x"}}); !reflect.DeepEqual(requests[0], want) || requests[1].Get("marker") != "node-1" {
		t.Fatalf("queries=%v", requests)
	}
	all, err := api.All(context.Background(), resource.WithPaginated(false))
	if err != nil || len(all) != 1 || len(requests) != 3 {
		t.Fatalf("single-page All=%d error=%v requests=%d", len(all), err, len(requests))
	}
}

func TestIronicIntrospectionRejectsInvalidOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicForbidHTTP(t, cloud)
	api := newIronicIntrospection(t, cloud, "")
	for _, err := range api.ListIntrospections(context.Background(), nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil option: %v", err)
		}
	}
	for _, err := range api.ListIntrospections(context.Background(), introspection.WithListIntrospectionsQuery(" ", "x")) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("empty query key: %v", err)
		}
	}
}

func TestIronicIntrospectionCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud, calls := ironicBlockingCloud(t)
	api := newIronicIntrospection(t, cloud, "")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	ctx := canceled
	if err := api.AbortIntrospection(ctx, "node-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled abort: %v", err)
	}
	if err := api.ReApplyIntrospection(ctx, "node-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled reapply: %v", err)
	}
	if _, err := api.GetIntrospectionStatus(ctx, "node-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled status: %v", err)
	}
	if _, err := api.GetIntrospectionData(ctx, "node-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled data: %v", err)
	}
	for _, err := range api.ListIntrospections(ctx) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := api.AbortIntrospection(deadline, "node-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
