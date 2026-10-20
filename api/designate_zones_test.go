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

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zones"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const designateZoneBody = `{"id":"zone-1","pool_id":"pool-1","project_id":"p1","name":"example.org.","email":"admin@example.org","ttl":3600,"serial":1700000000,"status":"PENDING","action":"CREATE","description":"","type":"PRIMARY","masters":[],"attributes":{},"version":1,"created_at":"2026-01-02T03:04:05.000000","links":{"self":"http://dns.example/v2/zones/zone-1"},"vendor_extra":1}`

func newDesignateZones(t *testing.T, cloud *testcloud.Cloud) *zones.API {
	t.Helper()
	return zones.New(cloud.Client("dns", "/v2"))
}

func designateFlatBody(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("request body: %v", err)
	}
	return body
}

func designateStrings(body map[string]json.RawMessage) map[string]string {
	got := map[string]string{}
	for key, value := range body {
		got[key] = string(value)
	}
	return got
}

func TestDesignateZoneCreateSendsFlatBodyAndPreservesExtensionValues(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := map[string]string{"name": `"example.org."`, "email": `"admin@example.org"`, "ttl": `3600`, "type": `"PRIMARY"`, "attributes": `{"k":"v"}`, "masters": `["192.0.2.1"]`, "flag": `false`, "nothing": `null`, "empty": `""`}
		if got := designateStrings(designateFlatBody(t, r)); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		w.Header().Set("X-Openstack-Request-Id", "req-zone-create")
		testcloud.JSON(w, 202, designateZoneBody)
	})
	value, err := newDesignateZones(t, cloud).Create(context.Background(), zones.CreateOpts{
		Name: "example.org.", Email: "admin@example.org", TTL: 3600, Type: "PRIMARY",
		Attributes: map[string]string{"k": "v"}, Masters: []string{"192.0.2.1"},
	}, zones.WithCreateField("flag", false), zones.WithCreateField("nothing", nil), zones.WithCreateField("empty", ""))
	if err != nil || value == nil || value.ID != "zone-1" || value.Status != "PENDING" || value.Action != "CREATE" || value.Serial != 1700000000 || value.CreatedAt.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestDesignateZoneCreateOmitsZeroValuedOptionalFields(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		if got := designateStrings(designateFlatBody(t, r)); !reflect.DeepEqual(got, map[string]string{"name": `"example.org."`}) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 201, designateZoneBody)
	})
	if _, err := newDesignateZones(t, cloud).Create(context.Background(), zones.CreateOpts{Name: "example.org."}); err != nil {
		t.Fatal(err)
	}
}

func TestDesignateZoneMutationsAcceptOnlyDocumentedStatusCodes(t *testing.T) {
	type call func(api *zones.API) error
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		method string
		path   string
		ok     []int
		call   call
	}{
		{"create", "POST", "/v2/zones", []int{201, 202}, func(a *zones.API) error {
			_, err := a.Create(ctx, zones.CreateOpts{Name: "example.org."})
			return err
		}},
		{"update", "PATCH", "/v2/zones/zone-1", []int{200, 202}, func(a *zones.API) error {
			_, err := a.Update(ctx, "zone-1", zones.UpdateOpts{Email: "a@example.org"})
			return err
		}},
		{"delete", "DELETE", "/v2/zones/zone-1", []int{202}, func(a *zones.API) error {
			_, err := a.Delete(ctx, "zone-1")
			return err
		}},
		{"unshare", "DELETE", "/v2/zones/zone-1/shares/share-1", []int{204}, func(a *zones.API) error {
			return a.Unshare(ctx, "zone-1", "share-1")
		}},
	} {
		for _, status := range []int{200, 201, 202, 204} {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc(tc.method+" "+tc.path, func(w http.ResponseWriter, r *http.Request) {
				if status == 204 {
					w.WriteHeader(204)
					return
				}
				testcloud.JSON(w, status, designateZoneBody)
			})
			accepted := false
			for _, ok := range tc.ok {
				accepted = accepted || ok == status
			}
			err := tc.call(newDesignateZones(t, cloud))
			if accepted != (err == nil) {
				t.Fatalf("%s status %d error=%v", tc.name, status, err)
			}
			if !accepted {
				var unexpected gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &unexpected) || unexpected.Actual != status {
					t.Fatalf("%s status %d error=%v", tc.name, status, err)
				}
			}
		}
	}
}

func TestDesignateZoneUpdateKeepsEmptyDescriptionAndOmitsUnsetFields(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("PATCH /v2/zones/zone-1", func(w http.ResponseWriter, r *http.Request) {
		want := map[string]string{"description": `""`, "ttl": `300`, "masters": `["192.0.2.2"]`, "flag": `false`}
		if got := designateStrings(designateFlatBody(t, r)); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 202, designateZoneBody)
	})
	empty := ""
	value, err := newDesignateZones(t, cloud).Update(context.Background(), "zone-1",
		zones.UpdateOpts{Description: &empty, TTL: 300, Masters: []string{"192.0.2.2"}}, zones.WithUpdateField("flag", false))
	if err != nil || value == nil || value.ID != "zone-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	cloud = testcloud.New(t)
	cloud.Mux.HandleFunc("PATCH /v2/zones/zone-1", func(w http.ResponseWriter, r *http.Request) {
		if got := designateStrings(designateFlatBody(t, r)); len(got) != 0 {
			t.Errorf("zero options must send an empty object: %v", got)
		}
		testcloud.JSON(w, 200, designateZoneBody)
	})
	if _, err := newDesignateZones(t, cloud).Update(context.Background(), "zone-1", zones.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
}

func TestDesignateZoneDeleteReturnsAcknowledgedZoneNotCompletion(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v2/zones/zone-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 202, `{"id":"zone-1","name":"example.org.","status":"PENDING","action":"DELETE"}`)
	})
	cloud.Mux.HandleFunc("DELETE /v2/zones/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{"code":404,"type":"zone_not_found"}`)
	})
	api := newDesignateZones(t, cloud)
	value, err := api.Delete(context.Background(), "zone-1")
	if err != nil || value == nil || value.Status != "PENDING" || value.Action != "DELETE" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Delete(context.Background(), "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Remove(context.Background(), resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
}

func TestDesignateZoneRejectsProtectedOrUnsupportedExtensionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newDesignateZones(t, cloud)
	ctx := context.Background()
	invalid := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	_, err := api.Create(ctx, zones.CreateOpts{Name: "a."}, zones.WithCreateField("name", "shadow"))
	invalid("protected create field", err)
	_, err = api.Update(ctx, "zone-1", zones.UpdateOpts{}, zones.WithUpdateField("email", "x"))
	invalid("protected update field", err)
	_, err = api.Create(ctx, zones.CreateOpts{Name: "a."}, nil)
	invalid("nil create option", err)
	_, err = api.Update(ctx, "zone-1", zones.UpdateOpts{}, nil)
	invalid("nil update option", err)
	for _, seq := range []func(func(*zones.Zone, error) bool){
		api.List(ctx, nil),
		api.List(ctx, zones.WithListHeader("X-Auth-All-Projects", "true"), zones.WithListOptions(zones.ListOpts{AllProjects: true})),
		api.List(ctx, zones.WithListHeader("x-auth-sudo-tenant-id", "p"), zones.WithListOptions(zones.ListOpts{SudoTenantID: "q"})),
	} {
		seen := 0
		for _, err := range seq {
			seen++
			invalid("list option", err)
		}
		if seen != 1 {
			t.Fatalf("invalid list yielded %d items", seen)
		}
	}
	for _, err := range api.ListShares(ctx, "zone-1", nil) {
		invalid("nil list shares option", err)
	}
	for _, err := range api.ListShares(ctx, "zone-1", zones.WithListSharesHeader("X-Auth-All-Projects", "true"), zones.WithListSharesOptions(zones.ListSharesOpts{AllProjects: true})) {
		invalid("list shares header collision", err)
	}
}

func TestDesignateZoneListFiltersHeadersAndFollowsLinksNext(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	var headers []http.Header
	cloud.Mux.HandleFunc("GET /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		headers = append(headers, r.Header.Clone())
		if r.URL.Query().Get("marker") == "zone-1" {
			testcloud.JSON(w, 200, `{"zones":[{"id":"zone-2","name":"two."}],"links":{"self":"x"}}`)
			return
		}
		next := cloud.Server.URL + "/v2/zones?marker=zone-1&limit=1"
		testcloud.JSON(w, 200, `{"zones":[{"id":"zone-1","name":"one."}],"links":{"next":"`+next+`"}}`)
	})
	var ids []string
	for value, err := range newDesignateZones(t, cloud).List(context.Background(),
		zones.WithListOptions(zones.ListOpts{Name: "one.", Status: "ACTIVE", Type: "PRIMARY", TTL: 60, SortKey: "name", SortDir: "asc", Limit: 1, AllProjects: true, SudoTenantID: "p9"}),
		zones.WithListQuery("vendor", "x"), zones.WithListHeader("X-Vendor", "h")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"zone-1", "zone-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	first := requests[0]
	want := url.Values{"name": {"one."}, "status": {"ACTIVE"}, "type": {"PRIMARY"}, "ttl": {"60"}, "sort_key": {"name"}, "sort_dir": {"asc"}, "limit": {"1"}, "vendor": {"x"}}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "zone-1" {
		t.Fatalf("second query=%v", requests[1])
	}
	if headers[0].Get("X-Auth-All-Projects") != "true" || headers[0].Get("X-Auth-Sudo-Tenant-ID") != "p9" || headers[0].Get("X-Vendor") != "h" {
		t.Fatalf("headers=%v", headers[0])
	}
	if headers[1].Get("X-Auth-All-Projects") != "true" {
		t.Fatalf("headers must be repeated on later pages: %v", headers[1])
	}
}

func TestDesignateZoneListOmitsFalseAllProjectsHeader(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["X-Auth-All-Projects"]; ok || r.URL.RawQuery != "" {
			t.Errorf("headers=%v query=%q", r.Header, r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"zones":[]}`)
	})
	for _, err := range newDesignateZones(t, cloud).List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestDesignateZoneReadsAndShareReadsUseDocumentedRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	var shareListHeader string
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, designateZoneBody) })
	cloud.Mux.HandleFunc("GET /v2/zones/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1/shares/share-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"share-1","zone_id":"zone-1","target_project_id":"p2"}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1/shares", func(w http.ResponseWriter, r *http.Request) {
		shareListHeader = r.Header.Get("X-Auth-All-Projects")
		testcloud.JSON(w, 200, `{"shared_zones":[{"id":"share-1","zone_id":"zone-1","target_project_id":"p2"}],"links":{}}`)
	})
	api := newDesignateZones(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "zone-1")
	if err != nil || value.Name != "example.org." || value.Type != "PRIMARY" || value.TTL != 3600 || value.Masters == nil || len(value.Masters) != 0 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	got, err := api.GetShare(ctx, "zone-1", "share-1")
	if err != nil || got.ZoneID != "zone-1" {
		t.Fatalf("share=%+v error=%v", got, err)
	}
	count := 0
	for item, err := range api.ListShares(ctx, "zone-1", zones.WithListSharesOptions(zones.ListSharesOpts{AllProjects: true})) {
		if err != nil || item.ID != "share-1" {
			t.Fatalf("item=%+v error=%v", item, err)
		}
		count++
	}
	if count != 1 || shareListHeader != "true" {
		t.Fatalf("count=%d header=%q", count, shareListHeader)
	}
}

func TestDesignateZoneCommonLookupFindsExactNameAndMapsStatus(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"zones":[{"id":"zone-1","name":"example.org."},{"id":"zone-2","name":"example.org.x"}]}`)
	})
	api := newDesignateZones(t, cloud)
	value, err := api.Find(context.Background(), resource.Name("example.org."))
	if err != nil || value == nil || value.ID != "zone-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if len(queries) == 0 || queries[0].Get("name") != "example.org." {
		t.Fatalf("queries=%v", queries)
	}
	queries = nil
	if _, err := api.All(context.Background(), resource.WithStatus("ACTIVE")); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 || queries[0].Get("status") != "ACTIVE" {
		t.Fatalf("queries=%v", queries)
	}
}

func TestDesignateZoneCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newDesignateZones(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "zone-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if _, err := api.Create(canceled, zones.CreateOpts{Name: "a."}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Delete(canceled, "zone-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	if err := api.Unshare(canceled, "zone-1", "share-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled unshare: %v", err)
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
	if _, err := api.Create(deadline, zones.CreateOpts{Name: "a."}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
