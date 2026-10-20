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

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/recordsets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const designateRecordSetBody = `{"id":"rr-1","zone_id":"zone-1","zone_name":"example.org.","project_id":"p1","name":"www.example.org.","type":"A","records":["192.0.2.1"],"ttl":300,"status":"PENDING","action":"CREATE","description":"","version":1,"created_at":"2026-01-02T03:04:05.000000","links":{"self":"http://dns.example/v2/zones/zone-1/recordsets/rr-1"}}`

func newDesignateRecordSets(t *testing.T, cloud *testcloud.Cloud) *recordsets.API {
	t.Helper()
	return recordsets.New(cloud.Client("dns", "/v2"))
}

func TestDesignateRecordSetCreateSendsFlatBodyHeadersAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/zones/zone-1/recordsets", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := map[string]string{"name": `"www.example.org."`, "type": `"A"`, "records": `["192.0.2.1"]`, "ttl": `300`, "description": `"web"`, "flag": `false`, "nothing": `null`}
		if got := designateStrings(designateFlatBody(t, r)); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		if r.Header.Get("X-Auth-All-Projects") != "true" || r.Header.Get("X-Auth-Sudo-Tenant-ID") != "p9" || r.Header.Get("X-Vendor") != "h" {
			t.Errorf("headers=%v", r.Header)
		}
		testcloud.JSON(w, 202, designateRecordSetBody)
	})
	value, err := newDesignateRecordSets(t, cloud).Create(context.Background(), "zone-1", recordsets.CreateOpts{
		Name: "www.example.org.", Type: "A", Records: []string{"192.0.2.1"}, TTL: 300, Description: "web", AllProjects: true, SudoTenantID: "p9",
	}, recordsets.WithCreateField("flag", false), recordsets.WithCreateField("nothing", nil), recordsets.WithCreateHeader("X-Vendor", "h"))
	if err != nil || value == nil || value.ID != "rr-1" || value.ZoneName != "example.org." || value.Status != "PENDING" || value.Action != "CREATE" || value.CreatedAt.IsZero() || len(value.Links) != 1 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestDesignateRecordSetUpdatePreservesEmptyZeroAndNullValues(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]string
	var headers []http.Header
	cloud.Mux.HandleFunc("PUT /v2/zones/zone-1/recordsets/rr-1", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, designateStrings(designateFlatBody(t, r)))
		headers = append(headers, r.Header.Clone())
		testcloud.JSON(w, 200, designateRecordSetBody)
	})
	api := newDesignateRecordSets(t, cloud)
	empty, zero, ttl := "", 0, 60
	ctx := context.Background()
	if _, err := api.Update(ctx, "zone-1", "rr-1", recordsets.UpdateOpts{Description: &empty, TTL: &zero, Records: []string{"192.0.2.2"}, AllProjects: true},
		recordsets.WithUpdateField("flag", false), recordsets.WithUpdateHeader("X-Vendor", "h")); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "zone-1", "rr-1", recordsets.UpdateOpts{TTL: &ttl}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "zone-1", "rr-1", recordsets.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{
		{"description": `""`, "ttl": `null`, "records": `["192.0.2.2"]`, "flag": `false`},
		{"ttl": `60`},
		{},
	}
	if !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies=%v", bodies)
	}
	if headers[0].Get("X-Auth-All-Projects") != "true" || headers[0].Get("X-Vendor") != "h" {
		t.Fatalf("headers=%v", headers[0])
	}
	if _, ok := headers[1]["X-Auth-All-Projects"]; ok {
		t.Fatalf("false AllProjects must not send a header: %v", headers[1])
	}
}

func TestDesignateRecordSetMutationsAcceptOnlyDocumentedStatusCodes(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, method, path string
		ok                 []int
		call               func(a *recordsets.API) error
	}{
		{"create", "POST", "/v2/zones/zone-1/recordsets", []int{201, 202}, func(a *recordsets.API) error {
			_, err := a.Create(ctx, "zone-1", recordsets.CreateOpts{Name: "a."})
			return err
		}},
		{"update", "PUT", "/v2/zones/zone-1/recordsets/rr-1", []int{200, 202}, func(a *recordsets.API) error {
			_, err := a.Update(ctx, "zone-1", "rr-1", recordsets.UpdateOpts{})
			return err
		}},
		{"delete", "DELETE", "/v2/zones/zone-1/recordsets/rr-1", []int{202}, func(a *recordsets.API) error {
			return a.Delete(ctx, "zone-1", "rr-1")
		}},
		{"delete with options", "DELETE", "/v2/zones/zone-1/recordsets/rr-1", []int{202}, func(a *recordsets.API) error {
			return a.DeleteWithOpts(ctx, "zone-1", "rr-1")
		}},
	} {
		for _, status := range []int{200, 201, 202, 204} {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc(tc.method+" "+tc.path, func(w http.ResponseWriter, r *http.Request) {
				if status == 204 {
					w.WriteHeader(204)
					return
				}
				testcloud.JSON(w, status, designateRecordSetBody)
			})
			accepted := false
			for _, ok := range tc.ok {
				accepted = accepted || ok == status
			}
			err := tc.call(newDesignateRecordSets(t, cloud))
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

func TestDesignateRecordSetDeleteHeadersAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var headers []http.Header
	cloud.Mux.HandleFunc("DELETE /v2/zones/zone-1/recordsets/{id}", func(w http.ResponseWriter, r *http.Request) {
		headers = append(headers, r.Header.Clone())
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{"code":404}`)
			return
		}
		w.WriteHeader(202)
	})
	api := newDesignateRecordSets(t, cloud)
	ctx := context.Background()
	if err := api.DeleteWithOpts(ctx, "zone-1", "rr-1", recordsets.WithDeleteWithOptsOptions(recordsets.DeleteOpts{AllProjects: true, SudoTenantID: "p9"}), recordsets.WithDeleteWithOptsHeader("X-Vendor", "h")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "zone-1", "rr-2"); err != nil {
		t.Fatal(err)
	}
	if headers[0].Get("X-Auth-All-Projects") != "true" || headers[0].Get("X-Auth-Sudo-Tenant-ID") != "p9" || headers[0].Get("X-Vendor") != "h" {
		t.Fatalf("headers=%v", headers[0])
	}
	if _, ok := headers[1]["X-Auth-All-Projects"]; ok {
		t.Fatalf("plain delete sent All-Projects: %v", headers[1])
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if err := api.Delete(ctx, "zone-1", "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	scope, err := api.InZone(ctx, resource.ID("zone-1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("scope Remove ignores missing by default: %v", err)
	}
}

func TestDesignateRecordSetRejectsProtectedOrUnsupportedExtensionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newDesignateRecordSets(t, cloud)
	ctx := context.Background()
	invalid := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	_, err := api.Create(ctx, "zone-1", recordsets.CreateOpts{Name: "a."}, recordsets.WithCreateField("records", []string{"x"}))
	invalid("protected create field", err)
	_, err = api.Create(ctx, "zone-1", recordsets.CreateOpts{Name: "a."}, recordsets.WithCreateHeader("x-auth-all-projects", "true"))
	invalid("protected create header", err)
	_, err = api.Create(ctx, "zone-1", recordsets.CreateOpts{Name: "a."}, nil)
	invalid("nil create option", err)
	_, err = api.Update(ctx, "zone-1", "rr-1", recordsets.UpdateOpts{}, recordsets.WithUpdateField("ttl", 1))
	invalid("protected update field", err)
	_, err = api.Update(ctx, "zone-1", "rr-1", recordsets.UpdateOpts{}, recordsets.WithUpdateHeader("X-Auth-Sudo-Tenant-ID", "p"))
	invalid("protected update header", err)
	_, err = api.Update(ctx, "zone-1", "rr-1", recordsets.UpdateOpts{}, nil)
	invalid("nil update option", err)
	invalid("nil delete option", api.DeleteWithOpts(ctx, "zone-1", "rr-1", nil))
	invalid("protected delete header", api.DeleteWithOpts(ctx, "zone-1", "rr-1", recordsets.WithDeleteWithOptsHeader("X-Auth-All-Projects", "true")))
	for _, err := range api.ListAll(ctx, nil) {
		invalid("nil list-all option", err)
	}
	for _, err := range api.ListByZone(ctx, "zone-1", recordsets.WithListByZoneHeader("X-Auth-All-Projects", "true")) {
		invalid("list header collision", err)
	}
}

func TestDesignateRecordSetListsUseZoneAndGlobalRoutesFiltersAndLinksNext(t *testing.T) {
	cloud := testcloud.New(t)
	var zoneQueries, allQueries []url.Values
	var zoneHeaders []http.Header
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1/recordsets", func(w http.ResponseWriter, r *http.Request) {
		zoneQueries = append(zoneQueries, r.URL.Query())
		zoneHeaders = append(zoneHeaders, r.Header.Clone())
		if r.URL.Query().Get("marker") == "rr-1" {
			testcloud.JSON(w, 200, `{"recordsets":[{"id":"rr-2","name":"b."}],"links":{}}`)
			return
		}
		next := cloud.Server.URL + "/v2/zones/zone-1/recordsets?marker=rr-1&limit=1"
		testcloud.JSON(w, 200, `{"recordsets":[{"id":"rr-1","name":"a."}],"links":{"next":"`+next+`"}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/recordsets", func(w http.ResponseWriter, r *http.Request) {
		allQueries = append(allQueries, r.URL.Query())
		if r.Header.Get("X-Auth-All-Projects") != "true" {
			t.Errorf("headers=%v", r.Header)
		}
		testcloud.JSON(w, 200, `{"recordsets":[{"id":"rr-9","zone_id":"zone-9","name":"z."}],"metadata":{"total_count":1}}`)
	})
	api := newDesignateRecordSets(t, cloud)
	var ids []string
	for value, err := range api.ListByZone(context.Background(), "zone-1",
		recordsets.WithListByZoneOptions(recordsets.ListOpts{Name: "a.", Type: "A", Data: "192.0.2.1", Status: "ACTIVE", TTL: 60, Limit: 1, SudoTenantID: "p9"}),
		recordsets.WithListByZoneQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"rr-1", "rr-2"}) || len(zoneQueries) != 2 {
		t.Fatalf("ids=%v queries=%v", ids, zoneQueries)
	}
	want := url.Values{"name": {"a."}, "type": {"A"}, "data": {"192.0.2.1"}, "status": {"ACTIVE"}, "ttl": {"60"}, "limit": {"1"}, "vendor": {"x"}}
	if !reflect.DeepEqual(zoneQueries[0], want) || zoneQueries[1].Get("marker") != "rr-1" {
		t.Fatalf("queries=%v", zoneQueries)
	}
	if zoneHeaders[0].Get("X-Auth-Sudo-Tenant-ID") != "p9" {
		t.Fatalf("headers=%v", zoneHeaders[0])
	}
	ids = nil
	for value, err := range api.ListAll(context.Background(), recordsets.WithListAllOptions(recordsets.ListOpts{ZoneID: "zone-9", AllProjects: true}), recordsets.WithListAllQuery("vendor", "y")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"rr-9"}) || len(allQueries) != 1 || allQueries[0].Get("zone_id") != "zone-9" || allQueries[0].Get("vendor") != "y" {
		t.Fatalf("ids=%v queries=%v", ids, allQueries)
	}
}

func TestDesignateRecordSetGetAndZoneScopeResolveWithoutExtraLookup(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1/recordsets/rr-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, designateRecordSetBody)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1/recordsets/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		testcloud.JSON(w, 200, `{"zones":[{"id":"zone-1","name":"example.org."}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1/recordsets", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"recordsets":[{"id":"rr-1","name":"www.example.org."},{"id":"rr-2","name":"www.example.org.x"}]}`)
	})
	api := newDesignateRecordSets(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "zone-1", "rr-1")
	if err != nil || value.Name != "www.example.org." || !reflect.DeepEqual(value.Records, []string{"192.0.2.1"}) || value.TTL != 300 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(ctx, "zone-1", "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	scope, err := api.InZone(ctx, resource.ID("zone-1"))
	if err != nil || lookups.Load() != 0 {
		t.Fatalf("explicit ID needs no lookup: error=%v lookups=%d", err, lookups.Load())
	}
	found, err := scope.Find(ctx, resource.Name("www.example.org."))
	if err != nil || found == nil || found.ID != "rr-1" {
		t.Fatalf("found=%+v error=%v", found, err)
	}
	if _, err := api.InZone(ctx, resource.Name("example.org.")); err != nil || lookups.Load() != 1 {
		t.Fatalf("named parent resolves once: error=%v lookups=%d", err, lookups.Load())
	}
}

func TestDesignateRecordSetCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := newDesignateRecordSets(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "zone-1", "rr-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if _, err := api.Create(canceled, "zone-1", recordsets.CreateOpts{Name: "a."}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if err := api.Delete(canceled, "zone-1", "rr-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	for _, err := range api.ListAll(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Update(deadline, "zone-1", "rr-1", recordsets.UpdateOpts{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
