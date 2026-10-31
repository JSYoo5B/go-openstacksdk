package connectiontest_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
)

const (
	cleanupDNSZonesPath = "/dns/v2/zones"
	cleanupDNSPTRPath   = "/dns/v2/reverse/floatingips"
)

func init() {
	cleanupEmptyLists[cleanupDNSZonesPath] = "zones"
	cleanupEmptyLists[cleanupDNSPTRPath] = "floatingips"
}

// checkDesignateScope fails the test when a request widens or changes the
// Designate project scope.
func checkDesignateScope(t *testing.T, r *http.Request) {
	t.Helper()
	for _, header := range []string{"X-Auth-All-Projects", "X-Auth-Sudo-Project-ID"} {
		if r.Header.Get(header) != "" {
			t.Error("Designate scope header", header, r.Method, r.URL)
		}
	}
}

// serveDNSZones serves the zone list in pages: each page after the first is
// reached through links.next with marker=<page index>.
func (c *cleanupCloud) serveDNSZones(t *testing.T, pages ...string) {
	c.Mux.HandleFunc("GET "+cleanupDNSZonesPath, func(w http.ResponseWriter, r *http.Request) {
		checkDesignateScope(t, r)
		page := 0
		if marker := r.URL.Query().Get("marker"); marker != "" {
			page, _ = strconv.Atoi(marker)
		} else if r.URL.RawQuery != "" {
			t.Error("zone list query", r.URL)
		}
		c.record("list zones " + strconv.Itoa(page))
		rows := ""
		if page < len(pages) {
			rows = pages[page]
		}
		links := `{"self":"` + c.Server.URL + cleanupDNSZonesPath + `"}`
		if page+1 < len(pages) {
			links = `{"self":"` + c.Server.URL + cleanupDNSZonesPath + `","next":"` + c.Server.URL + cleanupDNSZonesPath + `?marker=` + strconv.Itoa(page+1) + `"}`
		}
		testcloud.JSON(w, 200, `{"zones":[`+rows+`],"links":`+links+`,"metadata":{"total_count":null}}`)
	})
}

// serveDNSPTRs serves the floating IP PTR list without a query.
func (c *cleanupCloud) serveDNSPTRs(t *testing.T, rows string) {
	c.Mux.HandleFunc("GET "+cleanupDNSPTRPath, func(w http.ResponseWriter, r *http.Request) {
		checkDesignateScope(t, r)
		if r.URL.RawQuery != "" {
			t.Error("PTR list query", r.URL)
		}
		c.record("list ptrs")
		testcloud.JSON(w, 200, `{"floatingips":[`+rows+`],"links":{"self":"`+c.Server.URL+cleanupDNSPTRPath+`"}}`)
	})
}

// serveDNSMutations answers DELETE zones/{id} and PATCH
// reverse/floatingips/{id}. codes maps "zone:<id>" or "ptr:<id>" to the
// response status (-1 drops the connection); the default is 202. A GET of
// either resource fails the test, because Python does not wait.
func (c *cleanupCloud) serveDNSMutations(t *testing.T, codes map[string]int) {
	var mu sync.Mutex
	sent := map[string]bool{}
	respond := func(w http.ResponseWriter, key, body string) {
		mu.Lock()
		if sent[key] {
			t.Error("mutation sent twice", key)
		}
		sent[key] = true
		mu.Unlock()
		switch code := codes[key]; code {
		case 0:
			testcloud.JSON(w, 202, body)
		case -1:
			conn, _, err := http.NewResponseController(w).Hijack()
			if err == nil {
				_ = conn.Close()
			}
		default:
			w.Header().Set("X-Openstack-Request-Id", "req-"+key)
			testcloud.JSON(w, code, `{"code":`+strconv.Itoa(code)+`,"type":"rejected","message":"`+key+`"}`)
		}
	}
	c.Mux.HandleFunc("DELETE "+cleanupDNSZonesPath+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		checkDesignateScope(t, r)
		if r.Header.Get("X-Designate-Delete-Shares") != "" {
			t.Error("delete shares header", r.URL)
		}
		id := r.PathValue("id")
		c.record("DELETE zone " + id)
		respond(w, "zone:"+id, `{"id":"`+id+`","status":"PENDING","action":"DELETE"}`)
	})
	c.Mux.HandleFunc("PATCH "+cleanupDNSPTRPath+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		checkDesignateScope(t, r)
		id := r.PathValue("id")
		body, _ := io.ReadAll(r.Body)
		c.record("PATCH ptr " + id + " " + string(body))
		respond(w, "ptr:"+id, `{"id":"`+id+`","ptrdname":null,"status":"PENDING","action":"UPDATE"}`)
	})
	for _, path := range []string{cleanupDNSZonesPath + "/{id}", cleanupDNSPTRPath + "/{id}"} {
		c.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
			t.Error("unexpected poll", r.URL)
			http.Error(w, "unexpected", 500)
		})
	}
}

const cleanupDNSZoneRows = `{"id":"z-a","name":"a.example.","project_id":"project","status":"ACTIVE","created_at":"2025-01-01T00:00:00.000000","updated_at":null},` +
	`{"id":"z-b","name":"b.example.","project_id":"project","status":"ACTIVE","created_at":"2025-02-01T00:00:00.000000","updated_at":"2025-03-01T00:00:00.000000"}`

const cleanupDNSZoneOtherRows = `{"id":"z-shared","name":"shared.example.","project_id":"other","shared":true},` +
	`{"id":"z-unowned","name":"unowned.example."},` +
	`{"name":"no-id.example.","project_id":"project"}`

const cleanupDNSPTRRows = `{"id":"RegionOne:fip-a","address":"203.0.113.10","ptrdname":"vm.a.example.","status":"ACTIVE"},` +
	`{"id":"RegionOne:fip-b","address":"203.0.113.11","ptrdname":null,"status":null},` +
	`{"id":"RegionOne:fip-c","address":"203.0.113.12","project_id":"other"},` +
	`{"address":"203.0.113.13"}`

func TestProjectCleanupDNSPlansZonesAndPTRRecordsBeforeCompute(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveDNSZones(t, cleanupDNSZoneRows, cleanupDNSZoneOtherRows)
	cloud.serveDNSPTRs(t, cleanupDNSPTRRows)
	cloud.serveCompute(t, `{"id":"s1"}`, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"zone:z-a", "zone:z-b", "floating_ip:RegionOne:fip-a", "floating_ip:RegionOne:fip-b", "server:s1"}) {
		t.Fatalf("candidates %v", got)
	}
	if got := stageSummary(plan); !slices.Equal(got, []string{"zone", "floating_ip", "server+wait"}) {
		t.Fatalf("stages %v", got)
	}
	zone := candidateByID(plan, "zone", "z-a")
	if zone.Service != "dns" || zone.Name != "a.example." || string(zone.CreatedAt) != `"2025-01-01T00:00:00.000000"` || string(zone.UpdatedAt) != "null" ||
		len(zone.DependsOn) != 0 || zone.Reason != "all resources" || !strings.Contains(string(zone.Raw), `"status":"ACTIVE"`) {
		t.Fatalf("zone %+v", zone)
	}
	ptr := candidateByID(plan, "floating_ip", "RegionOne:fip-b")
	if ptr.Service != "dns" || ptr.Name != "203.0.113.11" || ptr.CreatedAt != nil || ptr.UpdatedAt != nil || !strings.Contains(string(ptr.Raw), `"ptrdname":null`) {
		t.Fatalf("PTR record %+v", ptr)
	}
	reasons := exclusionReasons(plan)
	if !strings.Contains(reasons["zone:z-shared"], "owned by project other (project_id)") ||
		!strings.Contains(reasons["zone:z-unowned"], "project_id is not reported") ||
		!strings.Contains(reasons["zone:"], "no usable id") ||
		!strings.Contains(reasons["floating_ip:RegionOne:fip-c"], "owned by project other (project_id)") ||
		!strings.Contains(reasons["floating_ip:"], "no usable id") || len(reasons) != 5 {
		t.Fatalf("exclusions %v", reasons)
	}
	for _, u := range plan.Unavailable {
		if u.Service == "dns" {
			t.Fatalf("dns unavailable: %+v", plan.Unavailable)
		}
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list zones 0", "list zones 1", "list ptrs", "list servers", "list groups"}) {
		t.Fatalf("requests %v", got)
	}
}

func TestProjectCleanupDNSFiltersSelectorAndSkip(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveDNSZones(t, cleanupDNSZoneRows)
	cloud.serveDNSPTRs(t, cleanupDNSPTRRows)
	cloud.serveCompute(t, ``, ``)
	ctx := context.Background()
	plan, err := cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithCreatedBefore(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"zone:z-a"}) {
		t.Fatalf("created filter candidates %v", got)
	}
	reasons := exclusionReasons(plan)
	if !strings.Contains(reasons["zone:z-b"], "created_at") || reasons["floating_ip:RegionOne:fip-a"] != "created_at is not available for dns.floating_ip" ||
		reasons["floating_ip:RegionOne:fip-b"] != "created_at is not available for dns.floating_ip" {
		t.Fatalf("created filter exclusions %v", reasons)
	}
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithUpdatedBefore(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"zone:z-b"}) {
		t.Fatalf("updated filter candidates %v", got)
	}
	if reason := exclusionReasons(plan)["zone:z-a"]; !strings.Contains(reason, "updated_at") {
		t.Fatalf("null updated_at: %q", reason)
	}
	var seen []string
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithSelector(func(c projectcleanup.Candidate) (bool, error) {
		seen = append(seen, c.Kind+":"+c.Name)
		return c.Name == "b.example." || c.Name == "203.0.113.10", nil
	}))
	if err != nil || !slices.Equal(refIDs(plan.Candidates()), []string{"zone:z-b", "floating_ip:RegionOne:fip-a"}) ||
		!slices.Equal(seen, []string{"zone:a.example.", "zone:b.example.", "floating_ip:203.0.113.10", "floating_ip:203.0.113.11"}) {
		t.Fatalf("selector %v %v %v", plan, seen, err)
	}
	before := len(cloud.events())
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(), projectcleanup.WithSkip("dns.zone"))
	if err != nil || !slices.Equal(refIDs(plan.Candidates()), []string{"floating_ip:RegionOne:fip-a", "floating_ip:RegionOne:fip-b"}) {
		t.Fatalf("skip zones %v %v", plan, err)
	}
	if got := cloud.events()[before:]; !slices.Equal(got, []string{"list ptrs", "list servers", "list groups"}) {
		t.Fatalf("skipped zones were listed: %v", got)
	}
	before = len(cloud.events())
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(), projectcleanup.WithSkip("dns.floating_ip"))
	if err != nil || !slices.Equal(refIDs(plan.Candidates()), []string{"zone:z-a", "zone:z-b"}) {
		t.Fatalf("skip PTR records %v %v", plan, err)
	}
	if got := cloud.events()[before:]; !slices.Equal(got, []string{"list zones 0", "list servers", "list groups"}) {
		t.Fatalf("skipped PTR records were listed: %v", got)
	}
}

func TestProjectCleanupDNSExecutesZoneDeletesThenPTRUnsets(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveDNSZones(t, `{"id":"ok","project_id":"project"},{"id":"gone","project_id":"project"},{"id":"locked","project_id":"project"},{"id":"lost","project_id":"project"}`)
	cloud.serveDNSPTRs(t, `{"id":"RegionOne:f-ok"},{"id":"RegionOne:f-plain"},{"id":"RegionOne:f-none"},{"id":"RegionOne:f-denied"},{"id":"RegionOne:f-lost"}`)
	cloud.serveDNSMutations(t, map[string]int{"zone:gone": 404, "zone:locked": 400, "zone:lost": -1,
		"ptr:RegionOne:f-plain": 200, "ptr:RegionOne:f-none": 404, "ptr:RegionOne:f-denied": 403, "ptr:RegionOne:f-lost": -1})
	cloud.serveCompute(t, ``, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	var progress []string
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithProgress(func(e projectcleanup.Event) error {
		progress = append(progress, e.Outcome.Candidate.ID+"="+string(e.Outcome.State))
		return nil
	}))
	if !errors.Is(err, projectcleanup.ErrIncomplete) || result == nil {
		t.Fatalf("result %v err %v", result, err)
	}
	want := []string{"ok=accepted", "gone=already_missing", "locked=failed", "lost=uncertain",
		"RegionOne:f-ok=accepted", "RegionOne:f-plain=accepted", "RegionOne:f-none=already_missing", "RegionOne:f-denied=failed", "RegionOne:f-lost=uncertain"}
	var got []string
	outcomes := map[string]projectcleanup.Outcome{}
	for _, o := range result.Outcomes {
		got = append(got, o.Candidate.ID+"="+string(o.State))
		outcomes[o.Candidate.ID] = o
		if o.WaitErr != nil {
			t.Errorf("%s waited: %v", o.Candidate.ID, o.WaitErr)
		}
	}
	if !slices.Equal(got, want) || !slices.Equal(progress, want) {
		t.Fatalf("outcomes\n got %v\nprogress %v\nwant %v", got, progress, want)
	}
	if ok := outcomes["ok"]; ok.StatusCode != 0 || ok.Err != nil {
		t.Fatalf("accepted zone %+v", ok)
	}
	if locked := outcomes["locked"]; locked.StatusCode != 400 || locked.Header.Get("X-Openstack-Request-Id") != "req-zone:locked" || !strings.Contains(string(locked.Body), "zone:locked") {
		t.Fatalf("locked evidence %+v", locked)
	}
	if denied := outcomes["RegionOne:f-denied"]; denied.StatusCode != 403 || !strings.Contains(string(denied.Body), "ptr:RegionOne:f-denied") {
		t.Fatalf("denied evidence %+v", denied)
	}
	if gone := outcomes["gone"]; gone.StatusCode != 404 {
		t.Fatalf("gone %+v", gone)
	}
	null := `{"ptrdname":null}`
	wantEvents := []string{"list zones 0", "list ptrs", "list servers", "list groups",
		"DELETE zone ok", "DELETE zone gone", "DELETE zone locked", "DELETE zone lost",
		"PATCH ptr RegionOne:f-ok " + null, "PATCH ptr RegionOne:f-plain " + null, "PATCH ptr RegionOne:f-none " + null,
		"PATCH ptr RegionOne:f-denied " + null, "PATCH ptr RegionOne:f-lost " + null}
	if got := cloud.events(); !slices.Equal(got, wantEvents) {
		t.Fatalf("events\n got %v\nwant %v", got, wantEvents)
	}
}

// A Neutron floating IP depends on the PTR record of the same floating IP:
// it is not deleted when the PTR record could not be unset, and its DELETE is
// sent after an accepted or missing one.
func TestProjectCleanupDNSPTRFailureKeepsNetworkFloatingIP(t *testing.T) {
	f := newNetworkFixture(t, ``, map[string]string{
		"service-providers?service_type=VPN": ``,
		"floatingips?project_id=project":     `{"id":"fip-a","project_id":"project"},{"id":"fip-b","project_id":"project"},{"id":"fip-c","project_id":"project"},{"id":"fip-d","project_id":"project"}`,
		"security-groups?project_id=project": ``,
		"networks?project_id=project":        ``,
		"routers?project_id=project":         ``,
	})
	f.serveDNSPTRs(t, `{"id":"RegionOne:fip-a"},{"id":"RegionOne:fip-b"},{"id":"RegionOne:fip-c"}`)
	f.serveDNSMutations(t, map[string]int{"ptr:RegionOne:fip-a": 500, "ptr:RegionOne:fip-c": 404})
	plan, err := f.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	if got := stageSummary(plan); !slices.Equal(got, []string{"floating_ip", "floating_ip"}) || plan.Stages[0].Service != "dns" || plan.Stages[1].Service != "network" {
		t.Fatalf("stages %v %+v", got, plan.Stages)
	}
	dnsRef := func(id string) projectcleanup.Ref {
		return projectcleanup.Ref{Service: "dns", Kind: "floating_ip", ID: "RegionOne:" + id}
	}
	for id, want := range map[string][]projectcleanup.Ref{"fip-a": {dnsRef("fip-a")}, "fip-b": {dnsRef("fip-b")}, "fip-c": {dnsRef("fip-c")}, "fip-d": nil} {
		c := candidateByID(plan, "floating_ip", id)
		if c.Service != "network" || !slices.Equal(c.DependsOn, want) {
			t.Fatalf("%s depends on %v", id, c.DependsOn)
		}
	}
	result, err := f.conn.ExecuteProjectCleanup(context.Background(), plan)
	if !errors.Is(err, projectcleanup.ErrIncomplete) {
		t.Fatalf("err %v", err)
	}
	var got []string
	for _, o := range result.Outcomes {
		got = append(got, o.Candidate.Service+"/"+o.Candidate.ID+"="+string(o.State))
		if o.Candidate.Ref == networkRef("floating_ip", "fip-a") && !strings.Contains(o.Reason, "dependency dns.floating_ip RegionOne:fip-a is failed") {
			t.Fatalf("blocked reason %q", o.Reason)
		}
	}
	want := []string{"dns/RegionOne:fip-a=failed", "dns/RegionOne:fip-b=accepted", "dns/RegionOne:fip-c=already_missing",
		"network/fip-a=not_attempted", "network/fip-b=accepted", "network/fip-c=accepted", "network/fip-d=accepted"}
	if !slices.Equal(got, want) {
		t.Fatalf("outcomes\n got %v\nwant %v", got, want)
	}
	var mutations []string
	for _, e := range f.events() {
		if strings.HasPrefix(e, "PATCH") || strings.HasPrefix(e, "DELETE") {
			mutations = append(mutations, e)
		}
	}
	wantMutations := []string{`PATCH ptr RegionOne:fip-a {"ptrdname":null}`, `PATCH ptr RegionOne:fip-b {"ptrdname":null}`, `PATCH ptr RegionOne:fip-c {"ptrdname":null}`,
		"DELETE floatingips/fip-b", "DELETE floatingips/fip-c", "DELETE floatingips/fip-d"}
	if !slices.Equal(mutations, wantMutations) {
		t.Fatalf("mutations\n got %v\nwant %v", mutations, wantMutations)
	}
}
