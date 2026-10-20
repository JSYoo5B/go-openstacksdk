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

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const ironicPortBody = `{"uuid":"port-1","address":"52:54:00:aa:bb:cc","node_uuid":"node-1","portgroup_uuid":null,"local_link_connection":{"switch_id":"aa"},"pxe_enabled":false,"physical_network":"","internal_info":{},"extra":{},"created_at":"2026-01-02T03:04:05+00:00","updated_at":null,"links":[],"is_smartnic":false,"vendor_extra":1}`

func newIronicPorts(t *testing.T, cloud *testcloud.Cloud, version string) *ports.API {
	t.Helper()
	return ports.New(ironicClient(cloud, "baremetal", version))
}

func ironicPortBool(value bool) *bool { return &value }

func TestIronicPortCreateKeepsExplicitFalseAndExtensionFields(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v1/ports", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if got := r.Header.Get(ironicAPIVersionHeader); got != "1.53" {
			t.Errorf("microversion header=%q", got)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body: %v", err)
		}
		got := map[string]string{}
		for key, value := range body {
			got[key] = string(value)
		}
		want := map[string]string{
			"node_uuid": `"node-1"`, "address": `"52:54:00:aa:bb:cc"`, "portgroup_uuid": `"pg-1"`,
			"local_link_connection": `{"switch_id":"aa"}`, "pxe_enabled": `false`, "physical_network": `"physnet"`,
			"extra": `{"k":false}`, "is_smartnic": `false`, "vendor": `{"enabled":false}`,
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 201, ironicPortBody)
	})
	value, err := newIronicPorts(t, cloud, "1.53").Create(context.Background(), ports.CreateOpts{
		NodeUUID: "node-1", Address: "52:54:00:aa:bb:cc", PortGroupUUID: "pg-1",
		LocalLinkConnection: map[string]any{"switch_id": "aa"}, PXEEnabled: ironicPortBool(false),
		PhysicalNetwork: "physnet", Extra: map[string]any{"k": false}, IsSmartNIC: ironicPortBool(false),
	}, ports.WithCreateField("vendor", map[string]bool{"enabled": false}))
	if err != nil || value == nil || value.UUID != "port-1" || calls.Load() != 1 {
		t.Fatalf("value=%+v error=%v calls=%d", value, err, calls.Load())
	}
	if value.PXEEnabled || value.IsSmartNIC || value.PortGroupUUID != "" || !value.UpdatedAt.IsZero() || value.CreatedAt.IsZero() {
		t.Fatalf("decoded port=%+v", value)
	}
}

func TestIronicPortCreateOmitsUnsetFieldsAndAcceptsOnly201Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v1/ports", func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get(ironicAPIVersionHeader); got != "" {
				t.Errorf("no microversion header expected, got %q", got)
			}
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 || string(body["address"]) != `"52:54:00:aa:bb:cc"` {
				t.Errorf("unset and empty values must be omitted: %v err=%v", body, err)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, ironicPortBody)
		})
		value, err := newIronicPorts(t, cloud, "").Create(context.Background(), ports.CreateOpts{Address: "52:54:00:aa:bb:cc", PhysicalNetwork: "", Extra: map[string]any{}})
		if tc.success != (err == nil) || (tc.success && value == nil) {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			ironicRequireStatus(t, err, tc.status)
		}
	}
}

func TestIronicPortRejectsInvalidInputBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicForbidHTTP(t, cloud)
	api := newIronicPorts(t, cloud, "")
	ctx := context.Background()
	if _, err := api.Create(ctx, ports.CreateOpts{}, ports.WithCreateField("address", "shadow")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("core field collision: %v", err)
	}
	if _, err := api.Create(ctx, ports.CreateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Update(ctx, "port-1", nil, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil update option: %v", err)
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
	for _, err := range api.ListDetail(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list-detail option: %v", err)
		}
	}
	for _, err := range api.ListDetail(ctx, ports.WithListDetailOptions(ports.ListOpts{Fields: []string{"uuid"}})) {
		if err == nil {
			t.Fatal("fields on the detail listing must fail before HTTP")
		}
	}
}

func TestIronicPortUpdateSendsJSONPatchWithExplicitFalseNullAndEmptyValues(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("PATCH /v1/ports/port-1", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if got := r.Header.Get(ironicAPIVersionHeader); got != "1.53" {
			t.Errorf("microversion header=%q", got)
		}
		var body []map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body: %v", err)
		}
		got := make([]map[string]string, len(body))
		for i, patch := range body {
			got[i] = map[string]string{}
			for key, value := range patch {
				got[i][key] = string(value)
			}
		}
		want := []map[string]string{
			{"op": `"replace"`, "path": `"/pxe_enabled"`, "value": `false`},
			{"op": `"replace"`, "path": `"/physical_network"`, "value": `""`},
			{"op": `"add"`, "path": `"/extra"`, "value": `{}`},
			{"op": `"remove"`, "path": `"/portgroup_uuid"`, "value": `null`},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("patch=%v", got)
		}
		testcloud.JSON(w, 200, ironicPortBody)
	})
	value, err := newIronicPorts(t, cloud, "1.53").Update(context.Background(), "port-1", ports.UpdateOpts{
		ports.UpdateOperation{Op: ports.ReplaceOp, Path: "/pxe_enabled", Value: false},
		ports.UpdateOperation{Op: ports.ReplaceOp, Path: "/physical_network", Value: ""},
		ports.UpdateOperation{Op: ports.AddOp, Path: "/extra", Value: map[string]any{}},
		ports.UpdateOperation{Op: ports.RemoveOp, Path: "/portgroup_uuid"},
	})
	if err != nil || value == nil || value.UUID != "port-1" || calls.Load() != 1 {
		t.Fatalf("value=%+v error=%v calls=%d", value, err, calls.Load())
	}
}

func TestIronicPortUpdateAcceptsOnly200AndOptionReplacesPatch(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PATCH /v1/ports/port-1", func(w http.ResponseWriter, r *http.Request) {
			var body []map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 || body[0]["path"] != "/address" {
				t.Errorf("WithUpdateOptions replaces the initial patch: %v err=%v", body, err)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, ironicPortBody)
		})
		value, err := newIronicPorts(t, cloud, "").Update(context.Background(), "port-1",
			ports.UpdateOpts{ports.UpdateOperation{Op: ports.ReplaceOp, Path: "/extra", Value: 1}},
			ports.WithUpdateOptions(ports.UpdateOpts{ports.UpdateOperation{Op: ports.ReplaceOp, Path: "/address", Value: "aa"}}))
		if tc.success != (err == nil) || (tc.success && value == nil) {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			ironicRequireStatus(t, err, tc.status)
		}
	}
}

func TestIronicPortGetAndDeleteStatusPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/ports/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "port-1":
			testcloud.JSON(w, 200, ironicPortBody)
		case "accepted":
			testcloud.JSON(w, 202, ironicPortBody)
		default:
			testcloud.JSON(w, 404, `{"error_message":"gone"}`)
		}
	})
	cloud.Mux.HandleFunc("DELETE /v1/ports/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "p202":
			w.WriteHeader(202)
		case "p204":
			w.WriteHeader(204)
		case "p200":
			w.WriteHeader(200)
		default:
			testcloud.JSON(w, 404, `{"error_message":"gone"}`)
		}
	})
	api := newIronicPorts(t, cloud, "")
	ctx := context.Background()
	value, err := api.Get(ctx, "port-1")
	if err != nil || value.Address != "52:54:00:aa:bb:cc" || value.NodeUUID != "node-1" || value.LocalLinkConnection["switch_id"] != "aa" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(ctx, "accepted")
	ironicRequireStatus(t, err, 202)
	_, err = api.Get(ctx, "missing")
	ironicRequireStatus(t, err, 404)
	for _, id := range []string{"p202", "p204"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	ironicRequireStatus(t, api.Delete(ctx, "p200"), 200)
	ironicRequireStatus(t, api.Delete(ctx, "missing"), 404)
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores a missing port by default: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Remove with WithMissingError: %v", err)
	}
}

func TestIronicPortListAndListDetailFiltersFollowLinks(t *testing.T) {
	cloud := testcloud.New(t)
	var plain, detail []url.Values
	handler := func(record *[]url.Values, path string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			*record = append(*record, r.URL.Query())
			if r.URL.Query().Get("marker") == "port-1" {
				testcloud.JSON(w, 200, `{"ports":[{"uuid":"port-2"}]}`)
				return
			}
			next := cloud.Server.URL + path + "?marker=port-1&limit=1"
			testcloud.JSON(w, 200, `{"ports":[{"uuid":"port-1"}],"ports_links":[{"rel":"next","href":"`+next+`"}]}`)
		}
	}
	cloud.Mux.HandleFunc("GET /v1/ports", handler(&plain, "/v1/ports"))
	cloud.Mux.HandleFunc("GET /v1/ports/detail", handler(&detail, "/v1/ports/detail"))
	api := newIronicPorts(t, cloud, "")
	collect := func(seq func(func(*ports.Port, error) bool)) []string {
		var ids []string
		for value, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.UUID)
		}
		return ids
	}
	opts := ports.ListOpts{Node: "n1", NodeUUID: "u1", PortGroup: "pg", Address: "aa", Limit: 1, SortKey: "address", SortDir: "asc"}
	listOpts := opts
	listOpts.Fields = []string{"uuid", "address"}
	ids := collect(api.List(context.Background(), ports.WithListOptions(listOpts), ports.WithListQuery("vendor", "x")))
	if !reflect.DeepEqual(ids, []string{"port-1", "port-2"}) || len(plain) != 2 {
		t.Fatalf("list ids=%v requests=%v", ids, plain)
	}
	wantPlain := url.Values{"node": {"n1"}, "node_uuid": {"u1"}, "portgroup": {"pg"}, "address": {"aa"}, "fields": {"uuid,address"}, "limit": {"1"}, "sort_key": {"address"}, "sort_dir": {"asc"}, "vendor": {"x"}}
	if !reflect.DeepEqual(plain[0], wantPlain) || plain[1].Get("marker") != "port-1" {
		t.Fatalf("list queries=%v", plain)
	}
	ids = collect(api.ListDetail(context.Background(), ports.WithListDetailOptions(opts), ports.WithListDetailQuery("vendor", "y")))
	if !reflect.DeepEqual(ids, []string{"port-1", "port-2"}) || len(detail) != 2 {
		t.Fatalf("detail ids=%v requests=%v", ids, detail)
	}
	wantDetail := url.Values{"node": {"n1"}, "node_uuid": {"u1"}, "portgroup": {"pg"}, "address": {"aa"}, "limit": {"1"}, "sort_key": {"address"}, "sort_dir": {"asc"}, "vendor": {"y"}}
	if !reflect.DeepEqual(detail[0], wantDetail) {
		t.Fatalf("detail query=%v want %v", detail[0], wantDetail)
	}
}

func TestIronicPortCollectionAllUsesDetailListingAndRejectsStatus(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v1/ports/detail", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"ports":[{"uuid":"port-1"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v1/ports", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Collection lookups must use the detail route: %s", r.URL)
		testcloud.JSON(w, 200, `{"ports":[]}`)
	})
	values, err := newIronicPorts(t, cloud, "").All(context.Background(), resource.WithQuery("address", "aa"))
	if err != nil || len(values) != 1 || len(queries) != 1 {
		t.Fatalf("values=%d error=%v queries=%v", len(values), err, queries)
	}
	if queries[0].Get("address") != "aa" {
		t.Fatalf("query=%v", queries[0])
	}
	// Ports declare no status attribute, so a status filter is unsupported and sends nothing.
	if _, err := newIronicPorts(t, cloud, "").All(context.Background(), resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) || len(queries) != 1 {
		t.Fatalf("status filter error=%v queries=%v", err, queries)
	}
}

func TestIronicPortCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud, calls := ironicBlockingCloud(t)
	api := newIronicPorts(t, cloud, "")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "port-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if _, err := api.Create(canceled, ports.CreateOpts{Address: "aa"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Update(canceled, "port-1", ports.UpdateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled update: %v", err)
	}
	if err := api.Delete(canceled, "port-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	for _, err := range api.ListDetail(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list detail: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Update(deadline, "port-1", ports.UpdateOpts{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
