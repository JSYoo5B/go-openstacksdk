package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/blacklists"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/ptr/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/servicestatuses"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zoneexports"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zoneimports"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestR2DNSZoneExportCreateHasNoBodyAcceptsOnly202AndTextUsesAcceptHeader(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v2/zones/z1/tasks/export", func(w http.ResponseWriter, r *http.Request) {
		if b, _ := io.ReadAll(r.Body); len(b) != 0 {
			t.Errorf("body=%q", b)
		}
		testcloud.JSON(w, 202, `{"id":"e1","zone_id":"z1","status":"PENDING","version":1}`)
	})
	cloud.Mux.HandleFunc("POST /v2/zones/z200/tasks/export", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"id":"e1"}`) })
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/exports/e1/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/dns" {
			t.Errorf("accept=%s", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/dns")
		_, _ = w.Write([]byte("$ORIGIN example.org.\n"))
	})
	cloud.Mux.HandleFunc("DELETE /v2/zones/tasks/exports/e1", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	api := zoneexports.New(r2dnsClient(cloud))
	ctx := context.Background()
	v, err := api.Create(ctx, "z1")
	if err != nil || v.ID != "e1" || v.Status != "PENDING" || v.StatusCode != 202 {
		t.Fatalf("create %+v %v", v, err)
	}
	if _, err := api.Create(ctx, "z200"); err == nil {
		t.Fatal("200 must be rejected for create")
	}
	if text, err := api.GetText(ctx, "e1"); err != nil || text.Zone != "$ORIGIN example.org.\n" {
		t.Fatalf("text %+v %v", text, err)
	}
	if _, err := api.Delete(ctx, "e1"); err != nil {
		t.Fatal(err)
	}
}

func TestR2DNSZoneExportListFiltersAndScopeHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/exports", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("zone_id") != "z1" || q.Get("status") != "ACTIVE" || r.Header.Get("X-Auth-All-Projects") != "false" || r.Header.Get("X-Auth-Sudo-Project-ID") != "p1" {
			t.Errorf("query=%s headers=%v", r.URL.RawQuery, r.Header)
		}
		testcloud.JSON(w, 200, `{"exports":[{"id":"e1"}]}`)
	})
	no := false
	n := 0
	for v, err := range zoneexports.New(r2dnsClient(cloud)).List(context.Background(), zoneexports.WithListOptions(zoneexports.ListOpts{ZoneID: "z1", Status: "ACTIVE", AllProjects: &no, SudoProject: "p1"})) {
		if err != nil || v.ID != "e1" {
			t.Fatal(v, err)
		}
		n++
	}
	if n != 1 {
		t.Fatal(n)
	}
}

func TestR2DNSZoneImportSendsRawZoneFileAsTextDNS(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v2/zones/tasks/imports", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Header.Get("Content-Type") != "text/dns" || string(b) != "$ORIGIN a.\n" {
			t.Errorf("ct=%s body=%q", r.Header.Get("Content-Type"), b)
		}
		testcloud.JSON(w, 202, `{"id":"i1","status":"PENDING","message":null}`)
	})
	v, err := zoneimports.New(r2dnsClient(cloud)).Create(context.Background(), "$ORIGIN a.\n")
	if err != nil || v.ID != "i1" || string(v.Body["message"]) != "null" {
		t.Fatalf("%+v %v", v, err)
	}
	cloud2 := testcloud.New(t)
	cloud2.Mux.HandleFunc("POST /v2/zones/tasks/imports", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 201, `{"id":"i1"}`) })
	if _, err := zoneimports.New(r2dnsClient(cloud2)).Create(context.Background(), "x"); err == nil {
		t.Fatal("201 must be rejected")
	}
}

func TestR2DNSZoneExportAndImportWaitsAndDeletion(t *testing.T) {
	cloud := testcloud.New(t)
	polls := 0
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/exports/e1", func(w http.ResponseWriter, r *http.Request) {
		polls++
		status := "PENDING"
		if polls > 1 {
			status = "complete"
		}
		testcloud.JSON(w, 200, `{"id":"e1","status":"`+status+`"}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/imports/bad", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"id":"bad","status":"ERROR"}`) })
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/imports/gone", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/exports/gone", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	ctx := context.Background()
	exp := zoneexports.New(r2dnsClient(cloud))
	if v, err := exp.WaitFor(ctx, resource.ID("e1"), "COMPLETE", resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5*time.Second)); err != nil || v.Status != "complete" {
		t.Fatalf("%+v %v", v, err)
	}
	imp := zoneimports.New(r2dnsClient(cloud))
	if _, err := imp.WaitFor(ctx, resource.ID("bad"), "COMPLETE", resource.WithPollInterval(time.Millisecond)); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("failure state: %v", err)
	}
	if err := imp.WaitForDeletion(ctx, resource.ID("gone"), resource.WithPollInterval(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := exp.WaitForDeletion(ctx, resource.ID("gone"), resource.WithPollInterval(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
}

func TestR2DNSFloatingIPPTRSetUnsetBodiesCodesAndColonID(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]string
	cloud.Mux.HandleFunc("PATCH /v2/reverse/floatingips/RegionOne:f1", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, r2dnsBody(t, r))
		code := 202
		if len(bodies) == 2 {
			code = 200
		}
		testcloud.JSON(w, code, `{"id":"RegionOne:f1","address":"192.0.2.1","ptrdname":null,"ttl":0,"status":"PENDING","action":"UPDATE"}`)
	})
	cloud.Mux.HandleFunc("PATCH /v2/reverse/floatingips/RegionOne:bad", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 201, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/reverse/floatingips/RegionOne:f1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"RegionOne:f1","status":"ACTIVE"}`)
	})
	api := floatingips.New(r2dnsClient(cloud))
	ctx := context.Background()
	if _, err := api.Set(ctx, "RegionOne:f1", floatingips.SetOpts{PTRDName: request.Present("a.example."), TTL: request.Present(0), Description: request.Present("")}); err != nil {
		t.Fatal(err)
	}
	v, err := api.Unset(ctx, "RegionOne:f1")
	if err != nil || v.StatusCode != 200 {
		t.Fatalf("%+v %v", v, err)
	}
	want := []map[string]string{{"ptrdname": `"a.example."`, "ttl": `0`, "description": `""`}, {"ptrdname": `null`}}
	if !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies=%v", bodies)
	}
	if _, err := api.Set(ctx, "RegionOne:bad", floatingips.SetOpts{}); err == nil {
		t.Fatal("201 must be rejected")
	}
	if _, err := api.Set(ctx, "RegionOne:f1", floatingips.SetOpts{}, floatingips.WithSetField("ttl", 1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("collision: %v", err)
	}
	if v, err := api.WaitFor(ctx, resource.ID("RegionOne:f1"), "active", resource.WithPollInterval(time.Millisecond)); err != nil || v.Status != "ACTIVE" {
		t.Fatalf("wait %+v %v", v, err)
	}
}

func TestR2DNSFloatingIPListScopeHeadersAndGetMissing(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/reverse/floatingips", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-All-Projects") != "true" {
			t.Errorf("headers=%v", r.Header)
		}
		testcloud.JSON(w, 200, `{"floatingips":[{"id":"R:1","ttl":300}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/reverse/floatingips/R:2", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := floatingips.New(r2dnsClient(cloud))
	yes := true
	for v, err := range api.List(context.Background(), floatingips.WithListOptions(floatingips.ListOpts{AllProjects: &yes})) {
		if err != nil || v.TTL != 300 {
			t.Fatal(v, err)
		}
	}
	if _, err := api.Get(context.Background(), "R:2"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestR2DNSLimitsAndServiceStatuses(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/limits", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"max_page_limit":1000,"min_ttl":null,"max_zones":10}`)
	})
	cloud.Mux.HandleFunc("GET /v2/service_statuses", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"service_statuses":[{"id":"s1","service_name":"central","status":"UP","stats":{"x":1},"capabilities":{}}],"links":{}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/service_statuses/s1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"s1","hostname":"h"}`)
	})
	client := r2dnsClient(cloud)
	ctx := context.Background()
	l, err := limits.New(client).Get(ctx)
	if err != nil || l.MaxPageLimit != 1000 || l.MaxZones != 10 || string(l.Body["min_ttl"]) != "null" {
		t.Fatalf("%+v %v", l, err)
	}
	api := servicestatuses.New(client)
	n := 0
	for v, err := range api.List(ctx) {
		if err != nil || v.ServiceName != "central" || string(v.Stats) != `{"x":1}` {
			t.Fatal(v, err)
		}
		n++
	}
	if v, err := api.Get(ctx, "s1"); err != nil || v.Hostname != "h" || n != 1 {
		t.Fatalf("%+v %v", v, err)
	}
}

func TestR2DNSQuotaCurrentUsesProjectlessRouteAndHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/quotas", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Sudo-Project-ID") != "p1" || r.Header.Get("X-Auth-All-Projects") != "true" {
			t.Errorf("headers=%v", r.Header)
		}
		testcloud.JSON(w, 200, `{"zones":0,"zone_records":500}`)
	})
	v, err := quotas.New(r2dnsClient(cloud)).Current(context.Background(), quotas.WithCurrentSudoProject("p1"), quotas.WithCurrentAllProjects(true))
	if err != nil || v.ProjectID != "p1" || string(v.Body["zones"]) != "0" || v.ZoneRecords != 500 {
		t.Fatalf("%+v %v", v, err)
	}
	if _, err := quotas.New(r2dnsClient(cloud)).Current(context.Background(), quotas.WithCurrentSudoProject("a/b")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestR2DNSListPagingKillSwitchMarkerFallbackLoopAndForeignLink(t *testing.T) {
	cloud := testcloud.New(t)
	hits := 0
	cloud.Mux.HandleFunc("GET /v2/blacklists", func(w http.ResponseWriter, r *http.Request) {
		hits++
		q := r.URL.Query()
		switch {
		case q.Get("limit") == "1" && q.Get("marker") == "":
			testcloud.JSON(w, 200, `{"blacklists":[{"id":"a"}]}`)
		case q.Get("marker") == "a":
			testcloud.JSON(w, 200, `{"blacklists":[{"id":"b"}]}`)
		case q.Get("marker") == "b":
			testcloud.JSON(w, 200, `{"blacklists":[]}`)
		default:
			t.Errorf("unexpected %s", r.URL)
		}
	})
	api := blacklists.New(r2dnsClient(cloud))
	var ids []string
	for v, err := range api.List(context.Background(), blacklists.WithListOptions(blacklists.ListOpts{Limit: 1})) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, v.ID)
	}
	if !reflect.DeepEqual(ids, []string{"a", "b"}) || hits != 3 {
		t.Fatalf("ids=%v hits=%d", ids, hits)
	}

	c2 := testcloud.New(t)
	c2.Mux.HandleFunc("GET /v2/blacklists", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"blacklists":[{"id":"a"}],"metadata":{"total_count":1},"links":{"next":"`+c2.Server.URL+`/v2/blacklists?marker=a"}}`)
	})
	n := 0
	for _, err := range blacklists.New(r2dnsClient(c2)).List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 {
		t.Fatalf("total_count kill switch: %d", n)
	}

	c3 := testcloud.New(t)
	c3.Mux.HandleFunc("GET /v2/blacklists", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"blacklists":[{"id":"a"}],"links":{"next":"`+c3.Server.URL+`/v2/blacklists"}}`)
	})
	var loopErr error
	for _, err := range blacklists.New(r2dnsClient(c3)).List(context.Background()) {
		loopErr = err
	}
	if loopErr == nil {
		t.Fatal("repeated next link must fail")
	}

	c4 := testcloud.New(t)
	c4.Mux.HandleFunc("GET /v2/blacklists", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"blacklists":[{"id":"a"}],"links":{"next":"http://other.invalid/v2/blacklists?marker=a"}}`)
	})
	var foreign error
	for _, err := range blacklists.New(r2dnsClient(c4)).List(context.Background()) {
		foreign = err
	}
	if foreign == nil {
		t.Fatal("foreign next link must fail")
	}

	var capped int
	for _, err := range blacklists.New(r2dnsClient(c2)).Resources.List(context.Background(), resource.WithMaxItems(1)) {
		if err != nil {
			t.Fatal(err)
		}
		capped++
	}
	if capped != 1 {
		t.Fatalf("capped=%d", capped)
	}
}

func TestR2DNSTransferRequestWaitsForStatusAndDeletion(t *testing.T) {
	cloud := testcloud.New(t)
	polls := 0
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/transfer_requests/tr-1", func(w http.ResponseWriter, r *http.Request) {
		polls++
		if polls > 2 {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		status := "PENDING"
		if polls == 2 {
			status = "active"
		}
		testcloud.JSON(w, 200, `{"id":"tr-1","status":"`+status+`"}`)
	})
	api := newDesignateTransferRequests(t, cloud)
	ctx := context.Background()
	if v, err := api.WaitFor(ctx, resource.ID("tr-1"), "ACTIVE", resource.WithPollInterval(time.Millisecond)); err != nil || v.Status != "active" {
		t.Fatalf("%+v %v", v, err)
	}
	if err := api.WaitForDeletion(ctx, resource.ID("tr-1"), resource.WithPollInterval(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
}
