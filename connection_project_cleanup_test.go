package openstack_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

// setCleanupScope records a Keystone v3 token result with an optional
// project scope and a catalog of the given service types on the fixture.
func setCleanupScope(t *testing.T, cloud *testcloud.Cloud, project string, catalog ...string) {
	t.Helper()
	var entries []any
	for _, serviceType := range catalog {
		entries = append(entries, map[string]any{"type": serviceType, "name": serviceType, "endpoints": []any{
			map[string]any{"interface": "public", "region": "RegionOne", "url": cloud.Server.URL + "/" + serviceType},
		}})
	}
	token := map[string]any{"catalog": entries}
	if project != "" {
		token["project"] = map[string]any{"id": project, "name": project + "-name", "domain": map[string]any{"id": "default", "name": "Default"}}
	}
	auth := tokens.CreateResult{}
	auth.Body = map[string]any{"token": token}
	auth.Header = http.Header{"X-Subject-Token": []string{"test-token"}}
	if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
}

// cleanupCloud is a project-scoped fixture whose catalog lists
// orchestration and load-balancer. Its connection overrides compute,
// network, image, block storage, orchestration and load balancer; object
// storage and key manager are absent.
type cleanupCloud struct {
	*testcloud.Cloud
	conn      *sdk.Connection
	mu        sync.Mutex
	mutations []string
}

func newCleanupCloud(t *testing.T) *cleanupCloud {
	t.Helper()
	cloud := testcloud.New(t)
	setCleanupScope(t, cloud, "project", "orchestration", "load-balancer")
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"),
		sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"),
		sdk.WithEndpoint(sdk.Image, cloud.Server.URL+"/image/v2"),
		sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/volume/v3/project"),
		sdk.WithEndpoint(sdk.Orchestration, cloud.Server.URL+"/orchestration"),
		sdk.WithEndpoint(sdk.LoadBalancer, cloud.Server.URL+"/load-balancer"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return &cleanupCloud{Cloud: cloud, conn: conn}
}

func (c *cleanupCloud) record(event string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mutations = append(c.mutations, event)
}

func (c *cleanupCloud) events() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.mutations)
}

// serveCompute lists the given rows; any unexpected request fails the test.
func (c *cleanupCloud) serveCompute(t *testing.T, servers, groups string) {
	c.Mux.HandleFunc("GET /compute/v2.1/project/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("all_tenants") || r.URL.Query().Has("all_projects") {
			t.Error("all-projects listing", r.URL)
		}
		c.record("list servers")
		testcloud.JSON(w, 200, `{"servers":[`+servers+`]}`)
	})
	c.Mux.HandleFunc("GET /compute/v2.1/project/os-server-groups", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("all_projects") {
			t.Error("all-projects listing", r.URL)
		}
		c.record("list groups")
		testcloud.JSON(w, 200, `{"server_groups":[`+groups+`]}`)
	})
	c.Mux.HandleFunc("/", cleanupCatchAll(t))
}

// cleanupEmptyLists are the list paths of registered services other than
// compute. The catch-all answers them with an empty project-scoped list so
// that compute-focused tests stay independent of the other services; a test
// of a service registers its own, more specific handler.
var cleanupEmptyLists = map[string]string{
	"/image/v2/images": "images",
	// network: without a VPN provider and networks, nothing else is listed.
	"/network/v2.0/service-providers": "service_providers",
	"/network/v2.0/floatingips":       "floatingips",
	"/network/v2.0/security-groups":   "security_groups",
	"/network/v2.0/networks":          "networks",
	"/network/v2.0/routers":           "routers",
}

// cleanupCatchAll serves cleanupEmptyLists and fails on any other request.
func cleanupCatchAll(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if plural, ok := cleanupEmptyLists[r.URL.Path]; ok && r.Method == http.MethodGet {
			if r.URL.Query().Has("all_tenants") || r.URL.Query().Has("all_projects") {
				t.Error("all-projects listing", r.URL)
			}
			testcloud.JSON(w, 200, `{"`+plural+`":[]}`)
			return
		}
		t.Error("unexpected request", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	}
}

func refIDs(candidates []projectcleanup.Candidate) []string {
	var ids []string
	for _, c := range candidates {
		ids = append(ids, c.Kind+":"+c.ID)
	}
	return ids
}

func exclusionReasons(plan *projectcleanup.Plan) map[string]string {
	out := map[string]string{}
	for _, e := range plan.Excluded {
		out[e.Kind+":"+e.ID] = e.Reason
	}
	return out
}

func TestProjectCleanupPlanDiscoversWithoutMutation(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveCompute(t,
		`{"id":"s1","name":"web","tenant_id":"project","created":"2025-01-01T00:00:00Z"},{"id":"s2","name":"foreign","tenant_id":"other"}`,
		`{"id":"g1","name":"with-candidate","members":["s1"]},{"id":"g2","name":"with-foreign","members":["s2"]},{"id":"g3","name":"empty","members":[]},{"id":"g4","name":"other","members":[],"project_id":"other"}`)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Project != (projectcleanup.Project{ID: "project", Name: "project-name", DomainID: "default", DomainName: "Default"}) {
		t.Fatalf("project %+v", plan.Project)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"server:s1", "server_group:g1", "server_group:g3"}) {
		t.Fatalf("candidates %v", got)
	}
	if len(plan.Stages) != 2 || !plan.Stages[0].WaitForAbsence || plan.Stages[1].WaitForAbsence || plan.Stages[0].Kind != "server" {
		t.Fatalf("stages %+v", plan.Stages)
	}
	group := plan.Stages[1].Candidates[0]
	if !slices.Equal(group.DependsOn, []projectcleanup.Ref{{Service: "compute", Kind: "server", ID: "s1"}}) || string(plan.Stages[0].Candidates[0].CreatedAt) != `"2025-01-01T00:00:00Z"` {
		t.Fatalf("group %+v", group)
	}
	if !strings.Contains(string(plan.Stages[0].Candidates[0].Raw), `"tenant_id":"project"`) {
		t.Fatalf("raw row %s", plan.Stages[0].Candidates[0].Raw)
	}
	reasons := exclusionReasons(plan)
	if !strings.Contains(reasons["server:s2"], "owned by project other") || !strings.Contains(reasons["server_group:g2"], "members that are not cleanup candidates: s2") ||
		!strings.Contains(reasons["server_group:g4"], "owned by project other") || len(reasons) != 3 {
		t.Fatalf("exclusions %v", reasons)
	}
	wantUnavailable := []projectcleanup.Unavailable{
		{Service: "object_store", Reason: projectcleanup.NotInCatalog},
		{Service: "key_manager", Reason: projectcleanup.NotInCatalog},
	}
	if !slices.Equal(plan.Unavailable, wantUnavailable) {
		t.Fatalf("unavailable %+v", plan.Unavailable)
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list servers", "list groups"}) {
		t.Fatalf("requests %v", got)
	}
}

func TestProjectCleanupRequiresSelectionScopeAndUnchangedPlan(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveCompute(t, `{"id":"s1","name":"web"}`, ``)
	ctx := context.Background()
	if _, err := cloud.conn.PlanProjectCleanup(ctx); !errors.Is(err, projectcleanup.ErrNoSelection) || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("missing selection: %v", err)
	}
	if _, err := cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(), projectcleanup.WithSkip("compute.servers")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("unknown skip name: %v", err)
	}
	if len(cloud.events()) != 0 {
		t.Fatalf("requests before validation: %v", cloud.events())
	}
	plan, err := cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	modified := *plan
	modified.Stages = slices.Clone(plan.Stages)
	modified.Stages[0].Candidates = append(slices.Clone(plan.Stages[0].Candidates), projectcleanup.Candidate{Ref: projectcleanup.Ref{Service: "compute", Kind: "server", ID: "s9"}})
	if _, err := cloud.conn.ExecuteProjectCleanup(ctx, &modified); !errors.Is(err, projectcleanup.ErrPlanModified) {
		t.Fatalf("modified plan: %v", err)
	}
	if _, err := cloud.conn.ExecuteProjectCleanup(ctx, &projectcleanup.Plan{}); !errors.Is(err, projectcleanup.ErrPlanModified) {
		t.Fatalf("hand-built plan: %v", err)
	}
	// The token now belongs to another project: the plan is refused.
	setCleanupScope(t, cloud.Cloud, "second", "orchestration", "load-balancer")
	if _, err := cloud.conn.ExecuteProjectCleanup(ctx, plan); !errors.Is(err, projectcleanup.ErrProjectMismatch) {
		t.Fatalf("project mismatch: %v", err)
	}
	// An unscoped token cannot plan.
	setCleanupScope(t, cloud.Cloud, "", "orchestration", "load-balancer")
	if _, err := cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources()); !errors.Is(err, projectcleanup.ErrUnscoped) {
		t.Fatalf("unscoped: %v", err)
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list servers", "list groups"}) {
		t.Fatalf("requests %v", got)
	}
}

func TestProjectCleanupTimeFiltersExcludeMissingAndUnparsableTimes(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveCompute(t, strings.Join([]string{
		`{"id":"old","created":"2025-06-01T00:00:00Z","updated":"2025-06-02T00:00:00Z"}`,
		`{"id":"edge","created":"2026-01-01T00:00:00Z","updated":"2025-06-02T00:00:00Z"}`,
		`{"id":"new","created":"2026-03-01T00:00:00Z","updated":"2025-06-02T00:00:00Z"}`,
		`{"id":"missing","updated":"2025-06-02T00:00:00Z"}`,
		`{"id":"null","created":null,"updated":"2025-06-02T00:00:00Z"}`,
		`{"id":"garbled","created":"last tuesday","updated":"2025-06-02T00:00:00Z"}`,
		`{"id":"stale","created":"2025-06-01T00:00:00Z","updated":"2026-06-02T00:00:00Z"}`,
		`{"id":"noupdate","created":"2025-06-01T00:00:00Z"}`,
	}, ","), `{"id":"g1","members":[]}`)
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithCreatedBefore(cutoff), projectcleanup.WithUpdatedBefore(cutoff))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"server:old", "server:edge"}) {
		t.Fatalf("candidates %v", got)
	}
	reasons := exclusionReasons(plan)
	for id, want := range map[string]string{
		"server:new": "after the cutoff", "server:missing": "created_at is not available", "server:null": "created_at is not available",
		"server:garbled": `"last tuesday" cannot be parsed`, "server:stale": "updated_at", "server:noupdate": "updated_at is not available",
		"server_group:g1": "created_at is not available for compute.server_group",
	} {
		if !strings.Contains(reasons[id], want) {
			t.Errorf("%s: %q, want %q", id, reasons[id], want)
		}
	}
	if !plan.CreatedBefore.Equal(cutoff) || plan.AllResources || plan.SelectorActive {
		t.Fatalf("recorded filters %+v", plan)
	}
}

func TestProjectCleanupSelectorAndSkipNames(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveCompute(t, `{"id":"keep","name":"keep-me"},{"id":"drop","name":"drop-me"}`, `{"id":"g1","members":[]}`)
	ctx := context.Background()
	var seen []string
	plan, err := cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithSelector(func(c projectcleanup.Candidate) (bool, error) {
		seen = append(seen, c.Kind+":"+c.ID)
		c.Raw[0] = 'X' // the selector receives a copy
		return c.Name == "drop-me" || c.Kind == "server_group", nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"server:drop", "server_group:g1"}) || !plan.SelectorActive {
		t.Fatalf("candidates %v", got)
	}
	if !slices.Equal(seen, []string{"server:keep", "server:drop", "server_group:g1"}) || plan.Candidates()[0].Raw[0] != '{' {
		t.Fatalf("selector calls %v", seen)
	}
	if reasons := exclusionReasons(plan); reasons["server:keep"] != "rejected by selector" {
		t.Fatalf("exclusions %v", reasons)
	}
	boom := errors.New("boom")
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithSelector(func(projectcleanup.Candidate) (bool, error) { return false, boom }))
	if !errors.Is(err, boom) || plan != nil {
		t.Fatalf("selector error: %v %v", plan, err)
	}
	before := len(cloud.events())
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(), projectcleanup.WithSkip("compute.server", "network.port"))
	if err != nil || len(plan.Candidates()) != 0 || !slices.Equal(plan.Skip, []string{"compute.server", "network.port"}) {
		t.Fatalf("skip: %+v %v", plan, err)
	}
	if len(cloud.events()) != before {
		t.Fatalf("skipped compute was listed: %v", cloud.events()[before:])
	}
}

func TestProjectCleanupExecuteReportsEveryState(t *testing.T) {
	cloud := newCleanupCloud(t)
	var deletes sync.Map
	count := func(id string) int32 {
		v, _ := deletes.LoadOrStore(id, new(atomic.Int32))
		return v.(*atomic.Int32).Add(1)
	}
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		cloud.record("delete " + id)
		if count(id) > 1 {
			t.Error("DELETE sent twice", id)
		}
		switch id {
		case "gone":
			testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
		case "locked":
			w.Header().Set("X-Openstack-Request-Id", "req-locked")
			testcloud.JSON(w, 409, `{"conflictingRequest":{"message":"locked"}}`)
		case "lost":
			conn, _, err := http.NewResponseController(w).Hijack()
			if err == nil {
				_ = conn.Close()
			}
		default:
			w.WriteHeader(204)
		}
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("get " + r.PathValue("id"))
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
	})
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/os-server-groups/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("delete group " + r.PathValue("id"))
		w.WriteHeader(204)
	})
	cloud.serveCompute(t, `{"id":"ok"},{"id":"gone"},{"id":"locked"},{"id":"lost"}`,
		`{"id":"g-ok","members":["ok"]},{"id":"g-gone","members":["gone"]},{"id":"g-locked","members":["locked"]},{"id":"g-lost","members":["lost"]}`)
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
	states := map[string]projectcleanup.Outcome{}
	for _, o := range result.Outcomes {
		states[o.Candidate.ID] = o
	}
	want := map[string]projectcleanup.State{"ok": projectcleanup.Deleted, "gone": projectcleanup.AlreadyMissing, "locked": projectcleanup.Failed, "lost": projectcleanup.Uncertain,
		"g-ok": projectcleanup.Accepted, "g-gone": projectcleanup.Accepted, "g-locked": projectcleanup.NotAttempted, "g-lost": projectcleanup.NotAttempted}
	for id, state := range want {
		if states[id].State != state {
			t.Errorf("%s: %s, want %s (%v)", id, states[id].State, state, states[id].Err)
		}
	}
	locked := states["locked"]
	if locked.StatusCode != 409 || locked.Header.Get("X-Openstack-Request-Id") != "req-locked" || !strings.Contains(string(locked.Body), "locked") || locked.Err == nil {
		t.Fatalf("failed evidence %+v", locked)
	}
	if states["lost"].Err == nil || !strings.Contains(states["g-lost"].Reason, "uncertain") || !strings.Contains(states["g-locked"].Reason, "failed") {
		t.Fatalf("dependents %+v %+v", states["g-lost"], states["g-locked"])
	}
	if result.Count(projectcleanup.NotAttempted) != 2 || result.Project.ID != "project" {
		t.Fatalf("counts %+v", result)
	}
	// Servers are deleted, then the only accepted one is observed absent,
	// then server groups whose members succeeded are deleted.
	wantEvents := []string{"list servers", "list groups", "delete ok", "delete gone", "delete locked", "delete lost", "get ok", "delete group g-ok", "delete group g-gone"}
	if got := cloud.events(); !slices.Equal(got, wantEvents) {
		t.Fatalf("events %v", got)
	}
	wantProgress := []string{"ok=accepted", "gone=already_missing", "locked=failed", "lost=uncertain", "ok=deleted", "g-ok=accepted", "g-gone=accepted", "g-locked=not_attempted", "g-lost=not_attempted"}
	if !slices.Equal(progress, wantProgress) {
		t.Fatalf("progress %v", progress)
	}
}

func TestProjectCleanupWaitsForServerAbsenceBeforeDependents(t *testing.T) {
	cloud := newCleanupCloud(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("delete " + r.PathValue("id"))
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		cloud.record("get " + id)
		if id == "broken" {
			testcloud.JSON(w, 500, `{"computeFault":{"message":"down"}}`)
			return
		}
		if gets.Add(1) == 1 {
			testcloud.JSON(w, 200, `{"server":{"id":"slow","status":"DELETING"}}`)
			return
		}
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
	})
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/os-server-groups/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("delete group " + r.PathValue("id"))
		w.WriteHeader(204)
	})
	cloud.serveCompute(t, `{"id":"slow"},{"id":"broken"}`, `{"id":"g-slow","members":["slow"]},{"id":"g-broken","members":["broken"]}`)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithDeletionWait(30*time.Second))
	if !errors.Is(err, projectcleanup.ErrIncomplete) {
		t.Fatalf("wait failure not reported: %v", err)
	}
	wantEvents := []string{"list servers", "list groups", "delete slow", "delete broken", "get slow", "get slow", "get broken", "delete group g-slow"}
	if got := cloud.events(); !slices.Equal(got, wantEvents) {
		t.Fatalf("events %v", got)
	}
	states := map[string]projectcleanup.Outcome{}
	for _, o := range result.Outcomes {
		states[o.Candidate.ID] = o
	}
	if states["slow"].State != projectcleanup.Deleted || states["broken"].State != projectcleanup.Accepted || states["broken"].WaitErr == nil ||
		states["g-broken"].State != projectcleanup.NotAttempted || !strings.Contains(states["g-broken"].Reason, "absence of dependency") || states["g-slow"].State != projectcleanup.Accepted {
		t.Fatalf("outcomes %+v", states)
	}
}

func TestProjectCleanupStopsOnProgressErrorAndCancellation(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("delete " + r.PathValue("id"))
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("get " + r.PathValue("id"))
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.serveCompute(t, `{"id":"a"},{"id":"b"},{"id":"c"}`, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	stop := errors.New("stop")
	var calls int
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithProgress(func(projectcleanup.Event) error {
		calls++
		return stop
	}))
	if !errors.Is(err, stop) || calls != 1 {
		t.Fatalf("progress stop: %v calls=%d", err, calls)
	}
	if result.Outcomes[0].State != projectcleanup.Accepted || result.Outcomes[1].State != projectcleanup.NotAttempted || result.Outcomes[2].State != projectcleanup.NotAttempted {
		t.Fatalf("outcomes %+v", result.Outcomes)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result, err = cloud.conn.ExecuteProjectCleanup(ctx, plan, projectcleanup.WithProgress(func(e projectcleanup.Event) error {
		cancel()
		return nil
	}))
	if !errors.Is(err, context.Canceled) || result.Outcomes[0].State != projectcleanup.Accepted || result.Outcomes[1].State != projectcleanup.NotAttempted {
		t.Fatalf("cancellation: %v %+v", err, result)
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list servers", "list groups", "delete a", "delete a"}) {
		t.Fatalf("events %v", got)
	}
}

func TestProjectCleanupParallelismWithinStage(t *testing.T) {
	cloud := newCleanupCloud(t)
	var inFlight, peak atomic.Int32
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		inFlight.Add(-1)
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.serveCompute(t, `{"id":"a"},{"id":"b"},{"id":"c"},{"id":"d"}`, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithParallelism(0)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("invalid parallelism: %v", err)
	}
	var callbackActive atomic.Int32
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithParallelism(2), projectcleanup.WithProgress(func(projectcleanup.Event) error {
		if callbackActive.Add(1) != 1 {
			t.Error("concurrent progress callbacks")
		}
		callbackActive.Add(-1)
		return nil
	}))
	if err != nil || result.Count(projectcleanup.Deleted) != 4 {
		t.Fatalf("result %+v %v", result, err)
	}
	if peak.Load() != 2 {
		t.Fatalf("peak concurrency %d", peak.Load())
	}
	peak.Store(0)
	if _, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan); err != nil || peak.Load() != 1 {
		t.Fatalf("sequential default: peak %d %v", peak.Load(), err)
	}
}

func TestProjectCleanupDiscoveryErrorNeedsExplicitPartialExecution(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 503, `{"computeFault":{"message":"maintenance"}}`)
	})
	cloud.Mux.HandleFunc("/", cleanupCatchAll(t))
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	var serviceErr *projectcleanup.ServiceError
	if plan == nil || !errors.As(err, &serviceErr) || serviceErr.Service != "compute" || !strings.Contains(err.Error(), "list servers") {
		t.Fatalf("plan %+v err %v", plan, err)
	}
	if len(plan.Errors) != 1 || len(plan.Candidates()) != 0 || len(plan.Unavailable) != 2 {
		t.Fatalf("partial plan %+v", plan)
	}
	if _, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan); !errors.Is(err, projectcleanup.ErrIncompletePlan) {
		t.Fatalf("incomplete plan executed: %v", err)
	}
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithAllowIncompletePlan())
	if err != nil || len(result.Outcomes) != 0 {
		t.Fatalf("explicit partial execution: %+v %v", result, err)
	}
}

func TestProjectCleanupReportsDisabledCoreServices(t *testing.T) {
	cloud := testcloud.New(t)
	setCleanupScope(t, cloud, "project", "dns")
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithServiceEnabled("compute", false), sdk.WithServiceEnabled("dns", true))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected request", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	plan, err := conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprint(plan.Unavailable)
	want := "[{compute disabled} {block_storage not in catalog} {network not in catalog} {image not in catalog} {object_store not in catalog} {key_manager not in catalog} {dns not yet supported}]"
	if got != want {
		t.Fatalf("unavailable %s", got)
	}
}
