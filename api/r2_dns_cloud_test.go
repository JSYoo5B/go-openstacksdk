package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	dnsv2 "github.com/JSYoo5B/go-openstacksdk/dns/v2"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/recordsets"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zones"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func r2dnsService(t *testing.T, cloud *testcloud.Cloud) *dnsv2.Service {
	t.Helper()
	return dnsv2.New(r2dnsClient(cloud))
}

func r2dnsStr(s string) *string { return &s }

func TestR2DNSZoneSharesTargetProjectFilterAndFindShare(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/zones/z1/shares", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("target_project_id") != "t&1" || r.Header.Get("X-Auth-All-Projects") != "true" {
			t.Errorf("query=%s headers=%v", r.URL.RawQuery, r.Header)
		}
		testcloud.JSON(w, 200, `{"shared_zones":[{"id":"s1","zone_id":"z1","target_project_id":"t&1"}]}`)
	})
	api := zones.New(r2dnsClient(cloud))
	yes := true
	n := 0
	for v, err := range api.ListSharesFiltered(context.Background(), "z1", zones.SharesQuery{TargetProjectID: r2dnsStr("t&1"), AllProjects: &yes}) {
		if err != nil || v.ID != "s1" {
			t.Fatal(v, err)
		}
		n++
	}
	if n != 1 {
		t.Fatal(n)
	}

	c2 := testcloud.New(t)
	c2.Mux.HandleFunc("GET /v2/zones/z1/shares/s1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"s1","zone_id":"z1"}`)
	})
	c2.Mux.HandleFunc("GET /v2/zones/z1/shares/s2", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	c2.Mux.HandleFunc("GET /v2/zones/z1/shares", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"shared_zones":[{"id":"s3","zone_id":"z1"}]}`)
	})
	a2 := zones.New(r2dnsClient(c2))
	ctx := context.Background()
	if v, err := a2.FindShare(ctx, "z1", "s1"); err != nil || v == nil || v.ID != "s1" {
		t.Fatalf("direct %+v %v", v, err)
	}
	if v, err := a2.FindShare(ctx, "z1", "s3"); err != nil || v == nil || v.ID != "s3" {
		t.Fatalf("GET 404 falls back to the share list: %+v %v", v, err)
	}
	if v, err := a2.FindShare(ctx, "z1", "s2"); err != nil || v != nil {
		t.Fatalf("missing default %+v %v", v, err)
	}
	if _, err := a2.FindShare(ctx, "z1", "s2", resource.WithIdentityFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict %v", err)
	}
}

func TestR2DNSRecordSetWaitsAndFindIdentityInZone(t *testing.T) {
	cloud := testcloud.New(t)
	polls := 0
	cloud.Mux.HandleFunc("GET /v2/zones/z1/recordsets/r1", func(w http.ResponseWriter, r *http.Request) {
		polls++
		if polls > 2 {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, `{"id":"r1","name":"www.example.org.","status":"PENDING_DELETE"}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/z1/recordsets/www.example.org.", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 400, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/zones/z1/recordsets", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") != "www.example.org." {
			t.Errorf("query=%s", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"recordsets":[{"id":"r1","name":"www.example.org."}]}`)
	})
	scope, err := recordsets.New(r2dnsClient(cloud)).InZone(context.Background(), resource.ID("z1"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if v, err := scope.FindIdentity(ctx, "www.example.org."); err != nil || v == nil || v.ID != "r1" {
		t.Fatalf("find %+v %v", v, err)
	}
	if err := scope.WaitDeleted(ctx, resource.ID("r1"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	c2 := testcloud.New(t)
	n := 0
	c2.Mux.HandleFunc("GET /v2/zones/z1/recordsets/r1", func(w http.ResponseWriter, r *http.Request) {
		n++
		status := "PENDING_CREATE"
		if n > 1 {
			status = "active"
		}
		testcloud.JSON(w, 200, `{"id":"r1","status":"`+status+`"}`)
	})
	s2, _ := recordsets.New(r2dnsClient(c2)).InZone(ctx, resource.ID("z1"))
	if v, err := s2.Wait(ctx, resource.ID("r1"), "ACTIVE", resource.WithPollInterval(time.Millisecond)); err != nil || v.Status != "active" {
		t.Fatalf("%+v %v", v, err)
	}
}

func TestR2DNSCloudCreateZoneBodyNullsSecondaryAndValidation(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]string
	cloud.Mux.HandleFunc("POST /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, r2dnsBody(t, r))
		testcloud.JSON(w, 202, `{"id":"z1","name":"example.org.","status":"PENDING","created_at":"2026-01-02T03:04:05.000000"}`)
	})
	s := r2dnsService(t, cloud)
	ctx := context.Background()
	if z, err := s.CreateDNSZone(ctx, "example.org."); err != nil || z.ID != "z1" {
		t.Fatalf("%+v %v", z, err)
	}
	ttl := 60
	if _, err := s.CreateDNSZone(ctx, "s.example.", dnsv2.WithDNSZoneCreateOptions(dnsv2.DNSZoneCreateOpts{ZoneType: r2dnsStr("secondary"), Email: r2dnsStr("a@b"), TTL: &ttl, Masters: []string{"192.0.2.1"}, Description: r2dnsStr("")})); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateDNSZone(ctx, "p.example.", dnsv2.WithDNSZoneCreateOptions(dnsv2.DNSZoneCreateOpts{ZoneType: r2dnsStr("primary"), Email: r2dnsStr("a@b"), TTL: &ttl})); err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{
		{"name": `"example.org."`, "email": `null`, "description": `null`},
		{"name": `"s.example."`, "description": `""`, "type": `"SECONDARY"`, "masters": `["192.0.2.1"]`},
		{"name": `"p.example."`, "email": `"a@b"`, "description": `null`, "ttl": `60`, "type": `"PRIMARY"`},
	}
	if !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies=%v", bodies)
	}
	before := len(bodies)
	if _, err := s.CreateDNSZone(ctx, "x.", dnsv2.WithDNSZoneCreateOptions(dnsv2.DNSZoneCreateOpts{ZoneType: r2dnsStr("alias")})); !errors.Is(err, resource.ErrInvalidOption) || len(bodies) != before {
		t.Fatalf("invalid type: %v", err)
	}
}

func TestR2DNSCloudZoneGetUpdateDeleteResolveNameOrID(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/zones/example.org.", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 400, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/zones/missing.", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 400, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") == "example.org." {
			testcloud.JSON(w, 200, `{"zones":[{"id":"z1","name":"example.org."}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"zones":[]}`)
	})
	cloud.Mux.HandleFunc("PATCH /v2/zones/z1", func(w http.ResponseWriter, r *http.Request) {
		if got := r2dnsBody(t, r); !reflect.DeepEqual(got, map[string]string{"description": `""`}) {
			t.Errorf("patch=%v", got)
		}
		testcloud.JSON(w, 202, `{"id":"z1","name":"example.org."}`)
	})
	deletes := r2dnsCounter(t, cloud, "DELETE /v2/zones/z1", 202, `{"id":"z1"}`)
	s := r2dnsService(t, cloud)
	ctx := context.Background()
	if z, err := s.GetDNSZone(ctx, "example.org."); err != nil || z == nil || z.ID != "z1" {
		t.Fatalf("%+v %v", z, err)
	}
	if z, err := s.GetDNSZone(ctx, "missing."); err != nil || z != nil {
		t.Fatalf("missing: %+v %v", z, err)
	}
	if _, err := s.UpdateDNSZone(ctx, "example.org.", zones.UpdateOpts{Description: r2dnsStr("")}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateDNSZone(ctx, "missing.", zones.UpdateOpts{}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("update missing: %v", err)
	}
	if ok, err := s.DeleteDNSZone(ctx, "missing."); err != nil || ok {
		t.Fatalf("delete missing: %v %v", ok, err)
	}
	if ok, err := s.DeleteDNSZone(ctx, "example.org."); err != nil || !ok || deletes.Load() != 1 {
		t.Fatalf("delete: %v %v %d", ok, err, deletes.Load())
	}
}

func TestR2DNSCloudRecordSetHelpers(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/zones/z1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"z1","name":"example.org."}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/gone", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/zones", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"zones":[]}`) })
	var created map[string]string
	cloud.Mux.HandleFunc("POST /v2/zones/z1/recordsets", func(w http.ResponseWriter, r *http.Request) {
		created = r2dnsBody(t, r)
		testcloud.JSON(w, 202, `{"id":"r1","name":"www.example.org.","type":"A","records":["192.0.2.1"]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/z1/recordsets/www.example.org.", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 400, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/zones/z1/recordsets/r1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"r1","name":"www.example.org.","type":"A"}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/z1/recordsets", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("name") == "www.example.org." || q.Get("name") == "" {
			testcloud.JSON(w, 200, `{"recordsets":[{"id":"r1","name":"www.example.org.","type":"A","ttl":300},{"id":"r2","name":"mail.example.org.","type":"MX","ttl":60}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"recordsets":[]}`)
	})
	var patch map[string]string
	cloud.Mux.HandleFunc("PUT /v2/zones/z1/recordsets/r1", func(w http.ResponseWriter, r *http.Request) {
		patch = r2dnsBody(t, r)
		testcloud.JSON(w, 202, `{"id":"r1"}`)
	})
	deletes := r2dnsCounter(t, cloud, "DELETE /v2/zones/z1/recordsets/r1", 202, `{"id":"r1"}`)
	s := r2dnsService(t, cloud)
	ctx := context.Background()
	if _, err := s.CreateDNSRecordSet(ctx, "z1", "www.example.org.", "a", []string{"192.0.2.1"}, "", 0); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(created, map[string]string{"name": `"www.example.org."`, "type": `"A"`, "records": `["192.0.2.1"]`}) {
		t.Fatalf("created=%v", created)
	}
	if v, err := s.GetDNSRecordSet(ctx, "z1", "r1"); err != nil || v == nil || v.ID != "r1" {
		t.Fatalf("get %+v %v", v, err)
	}
	if v, err := s.GetDNSRecordSet(ctx, "z1", "nothing."); err != nil || v != nil {
		t.Fatalf("get missing %+v %v", v, err)
	}
	if _, err := s.GetDNSRecordSet(ctx, "gone", "x"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing zone: %v", err)
	}
	ttl := 120
	if _, err := s.UpdateDNSRecordSet(ctx, "z1", "r1", recordsets.UpdateOpts{TTL: &ttl}); err != nil || !reflect.DeepEqual(patch, map[string]string{"ttl": `120`}) {
		t.Fatalf("update %v %v", patch, err)
	}
	if list, err := s.ListDNSRecordSets(ctx, "z1"); err != nil || len(list) != 2 {
		t.Fatalf("list %v %v", list, err)
	}
	res, err := s.SearchDNSRecordSets(ctx, "z1", "*.example.org.", dnsv2.WithDNSSearchFilters(map[string]any{"type": "MX", "ttl": 60}))
	if err != nil || len(res.Items) != 1 || res.Items[0].ID != "r2" {
		t.Fatalf("search %+v %v", res, err)
	}
	expr, err := s.SearchDNSRecordSets(ctx, "z1", "", dnsv2.WithDNSSearchFilters("[?type=='A'].id"))
	if err != nil || !expr.Expression || string(expr.Value) != `["r1"]` {
		t.Fatalf("expression %+v %v", expr, err)
	}
	if ok, err := s.DeleteDNSRecordSet(ctx, "z1", "r1"); err != nil || !ok || deletes.Load() != 1 {
		t.Fatalf("delete %v %v", ok, err)
	}
	if ok, err := s.DeleteDNSRecordSet(ctx, "z1", "nothing."); err != nil || ok {
		t.Fatalf("delete missing %v %v", ok, err)
	}
}

func TestR2DNSCloudSearchZonesSendsFiltersAsQueryAndFiltersLocally(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("status") != "ACTIVE" || q.Get("ttl") != "300" {
			t.Errorf("query=%s", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"zones":[{"id":"z1","name":"a.example.","status":"ACTIVE","ttl":300},{"id":"z2","name":"b.example.","status":"ACTIVE","ttl":300}]}`)
	})
	s := r2dnsService(t, cloud)
	res, err := s.SearchDNSZones(context.Background(), "a.*", dnsv2.WithDNSSearchFilters(map[string]any{"status": "ACTIVE", "ttl": 300}))
	if err != nil || len(res.Items) != 1 || res.Items[0].ID != "z1" {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := s.SearchDNSZones(context.Background(), "", dnsv2.WithDNSSearchFilters(map[string]any{"x": []string{"a"}})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("non-scalar: %v", err)
	}
	if _, err := s.SearchDNSZones(context.Background(), "", dnsv2.WithDNSSearchFilters("[?id]")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("string filter: %v", err)
	}
}

func TestR2DNSConnectionHelpersUseDNSServiceAndHonorCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /dns/v2/zones", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"zones":[{"id":"z1","name":"a."}]}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.DNS, cloud.Server.URL+"/dns/v2"))
	if err != nil {
		t.Fatal(err)
	}
	list, err := conn.ListDNSZones(context.Background(), zones.WithListQuery("name", url.QueryEscape("a.")))
	if err != nil || len(list) != 1 {
		t.Fatalf("%v %v", list, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := conn.DeleteDNSZone(ctx, "a."); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}
