package openstack_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
)

// swiftListing is one fixture response of a Swift listing request, keyed
// by the escaped path and the marker query value.
type swiftListing struct {
	status int
	count  string
	body   string
}

// serveObjectStore serves the account at /object-store/ with the given
// listings and records every request with its escaped path. Mutations are
// answered by deletes, keyed by the escaped path; others fail the test.
func (c *cleanupCloud) serveObjectStore(t *testing.T, listings map[string]swiftListing, deletes map[string]int) {
	setCleanupScope(t, c.Cloud, "project", "orchestration", "object-store")
	// The fixture provider has no endpoint locator, so the catalog entry
	// makes the service present and an override supplies its client.
	conn, err := sdk.FromProvider(c.Provider,
		sdk.WithEndpoint(sdk.Compute, c.Server.URL+"/compute/v2.1/project"),
		sdk.WithEndpoint(sdk.ObjectStorage, c.Server.URL+"/object-store"),
	)
	if err != nil {
		t.Fatal(err)
	}
	c.conn = conn
	c.Mux.HandleFunc("/object-store/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.EscapedPath(), "/object-store/")
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("format") != "json" {
				t.Error("listing without format=json", r.URL)
			}
			key := path
			if marker := r.URL.Query().Get("marker"); marker != "" {
				key += "?marker=" + marker
			}
			c.record("list /" + key)
			listing, ok := listings[key]
			if !ok {
				listing = swiftListing{status: 200, body: `[]`}
			}
			if listing.count != "" {
				w.Header().Set(map[bool]string{true: "X-Account-Container-Count", false: "X-Container-Object-Count"}[path == ""], listing.count)
			}
			testcloud.JSON(w, listing.status, listing.body)
		case http.MethodDelete:
			c.record("delete /" + path)
			status, ok := deletes[path]
			switch {
			case !ok:
				t.Error("unexpected delete", r.URL)
				http.Error(w, "unexpected", 500)
			case status == 0:
				conn, _, err := http.NewResponseController(w).Hijack()
				if err == nil {
					_ = conn.Close()
				}
			case status == 404:
				http.Error(w, "<html><h1>Not Found</h1></html>", 404)
			case status == 409:
				w.Header().Set("X-Trans-Id", "tx-conflict")
				http.Error(w, "<html><h1>Conflict</h1><p>There was a conflict when trying to complete your request.</p></html>", 409)
			default:
				w.WriteHeader(status)
			}
		default:
			t.Error("unexpected request", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	})
	c.serveCompute(t, ``, ``)
}

func TestProjectCleanupObjectStoreDiscovery(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveObjectStore(t, map[string]swiftListing{
		// The count header says a second page exists; it ends the listing.
		"":              {200, "4", `[{"name":"logs","count":2,"bytes":3},{"name":"mixed","count":2,"bytes":2}]`},
		"?marker=mixed": {200, "4", `[{"name":"empty","count":0,"bytes":0},{"name":"with space","count":1,"bytes":1}]`},
		// Without a count header the listing ends on an empty page.
		"logs":         {200, "", `[{"name":"a.log","last_modified":"2025-01-01T00:00:00.000000"},{"name":"dir/b c%.log","last_modified":"2025-01-02T00:00:00.000000"}]`},
		"mixed":        {200, "2", `[{"name":"drop","last_modified":"2025-01-01T00:00:00.000000"},{"name":"keep-me","last_modified":"2025-01-01T00:00:00.000000"}]`},
		"empty":        {204, "0", ``},
		"with%20space": {200, "1", `[{"name":"x","last_modified":"2025-01-01T00:00:00.000000"}]`},
	}, nil)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithSelector(func(c projectcleanup.Candidate) (bool, error) {
		return c.Name != "keep-me", nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range plan.Candidates() {
		got = append(got, c.Kind+":"+c.Parent+"/"+c.ID)
	}
	want := []string{"object:logs/a.log", "object:logs/dir/b c%.log", "object:mixed/drop", "object:with space/x",
		"container:/logs", "container:/empty", "container:/with space"}
	if !slices.Equal(got, want) {
		t.Fatalf("candidates %v", got)
	}
	if len(plan.Stages) != 2 || plan.Stages[0].Kind != "object" || plan.Stages[1].Kind != "container" || plan.Stages[0].WaitForAbsence || plan.Stages[1].WaitForAbsence {
		t.Fatalf("stages %+v", plan.Stages)
	}
	containers := plan.Stages[1].Candidates
	wantDeps := []projectcleanup.Ref{{Service: "object_store", Kind: "object", ID: "a.log", Parent: "logs"}, {Service: "object_store", Kind: "object", ID: "dir/b c%.log", Parent: "logs"}}
	if !slices.Equal(containers[0].DependsOn, wantDeps) || len(containers[1].DependsOn) != 0 || !strings.Contains(containers[0].Reason, "all objects are object candidates") {
		t.Fatalf("container dependencies %+v", containers)
	}
	object := plan.Stages[0].Candidates[0]
	if string(object.UpdatedAt) != `"2025-01-01T00:00:00.000000"` || object.CreatedAt != nil || !strings.Contains(string(object.Raw), `"name":"a.log"`) {
		t.Fatalf("object row %+v", object)
	}
	reasons := exclusionReasons(plan)
	if reasons["object:keep-me"] != "rejected by selector" || !strings.Contains(reasons["container:mixed"], "keeps 1 object(s) that are not cleanup candidates") || len(reasons) != 2 {
		t.Fatalf("exclusions %v", reasons)
	}
	wantRequests := []string{"list servers", "list groups", "list /", "list /logs", "list /logs?marker=dir/b c%.log", "list /mixed", "list /?marker=mixed", "list /empty", "list /with%20space"}
	if got := cloud.events(); !slices.Equal(got, wantRequests) {
		t.Fatalf("requests %v", got)
	}
}

func TestProjectCleanupObjectStoreTimeFilters(t *testing.T) {
	listings := map[string]swiftListing{
		"": {200, "1", `[{"name":"c1","count":5,"bytes":5,"last_modified":"2025-01-01T00:00:00.000000"}]`},
		"c1": {200, "5", `[{"name":"old","last_modified":"2025-06-01T00:00:00.000000"},{"name":"edge","last_modified":"2026-01-01T00:00:00"},` +
			`{"name":"new","last_modified":"2026-01-01T00:00:00.000001"},{"name":"missing"},{"name":"garbled","last_modified":"yesterday"}]`},
	}
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cloud := newCleanupCloud(t)
	cloud.serveObjectStore(t, listings, nil)
	// updated_at is the listing's last_modified, read as UTC without an offset.
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithUpdatedBefore(cutoff))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"object:old", "object:edge"}) {
		t.Fatalf("candidates %v", got)
	}
	reasons := exclusionReasons(plan)
	for id, want := range map[string]string{
		"object:new": "after the cutoff", "object:missing": "updated_at is not available", "object:garbled": `"yesterday" cannot be parsed`,
		"container:c1": "keeps 3 object(s)",
	} {
		if !strings.Contains(reasons[id], want) {
			t.Errorf("%s: %q, want %q", id, reasons[id], want)
		}
	}
	// Python's Object has no created_at; nothing is selected.
	plan, err = cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithCreatedBefore(cutoff))
	if err != nil || len(plan.Candidates()) != 0 {
		t.Fatalf("created filter %+v %v", plan, err)
	}
	if reasons := exclusionReasons(plan); !strings.Contains(reasons["object:old"], "created_at is not available for object_store.object") || !strings.Contains(reasons["container:c1"], "keeps 5 object(s)") {
		t.Fatalf("created filter exclusions %v", reasons)
	}
	// Python's Container has no updated_at: an empty container is kept too.
	cloud = newCleanupCloud(t)
	cloud.serveObjectStore(t, map[string]swiftListing{"": {200, "1", `[{"name":"c2","count":0,"last_modified":"2025-01-01T00:00:00.000000"}]`}}, nil)
	plan, err = cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithUpdatedBefore(cutoff))
	if err != nil || len(plan.Candidates()) != 0 || !strings.Contains(exclusionReasons(plan)["container:c2"], "updated_at is not available for object_store.container") {
		t.Fatalf("container filter %+v %v", plan, err)
	}
}

func TestProjectCleanupObjectStoreSkipNames(t *testing.T) {
	for _, skip := range []string{"object_store.object", "object_store.container"} {
		cloud := newCleanupCloud(t)
		cloud.serveObjectStore(t, map[string]swiftListing{"": {200, "1", `[{"name":"c1","count":0}]`}}, nil)
		plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources(), projectcleanup.WithSkip(skip))
		if err != nil || len(plan.Candidates()) != 0 || len(plan.Excluded) != 0 {
			t.Fatalf("%s: %+v %v", skip, plan, err)
		}
		if got := cloud.events(); !slices.Equal(got, []string{"list servers", "list groups"}) {
			t.Fatalf("%s: requests %v", skip, got)
		}
	}
}

func TestProjectCleanupObjectStoreListingErrors(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveObjectStore(t, map[string]swiftListing{
		"":   {200, "2", `[{"name":"c1"},{"name":"c2"}]`},
		"c1": {200, "1", `[{"name":"a"}]`},
		"c2": {503, "", `{"error":"down"}`},
	}, nil)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	var serviceErr *projectcleanup.ServiceError
	if plan == nil || !errors.As(err, &serviceErr) || serviceErr.Service != "object_store" || !strings.Contains(err.Error(), "list objects of container c2") {
		t.Fatalf("plan %+v err %v", plan, err)
	}
	// A failed service keeps neither candidates nor exclusions.
	if len(plan.Candidates()) != 0 || len(plan.Excluded) != 0 || len(plan.Errors) != 1 {
		t.Fatalf("partial plan %+v", plan)
	}
	// A listing whose marker does not advance is an error, not a loop.
	cloud = newCleanupCloud(t)
	cloud.serveObjectStore(t, map[string]swiftListing{
		"":          {200, "", `[{"name":"b"}]`},
		"?marker=b": {200, "", `[{"name":"a"}]`},
	}, nil)
	if _, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources()); err == nil || !strings.Contains(err.Error(), "does not advance") {
		t.Fatalf("cycle: %v", err)
	}
}

func TestProjectCleanupObjectStoreExecuteReportsEveryState(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveObjectStore(t, map[string]swiftListing{
		"":       {200, "4", `[{"name":"ok"},{"name":"locked"},{"name":"lost"},{"name":"busy"}]`},
		"ok":     {200, "2", `[{"name":"a/b c"},{"name":"gone"}]`},
		"locked": {200, "2", `[{"name":"fine"},{"name":"held"}]`},
		"lost":   {200, "1", `[{"name":"x"}]`},
		"busy":   {200, "0", `[]`},
	}, map[string]int{
		"ok/a%2Fb%20c": 204, "ok/gone": 404, "locked/fine": 204, "locked/held": 409, "lost/x": 0,
		"ok": 204, "busy": 409,
	})
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan)
	if !errors.Is(err, projectcleanup.ErrIncomplete) || result == nil {
		t.Fatalf("result %v err %v", result, err)
	}
	states := map[string]projectcleanup.Outcome{}
	for _, o := range result.Outcomes {
		states[o.Candidate.Kind+":"+o.Candidate.Parent+"/"+o.Candidate.ID] = o
	}
	want := map[string]projectcleanup.State{
		"object:ok/a/b c": projectcleanup.Accepted, "object:ok/gone": projectcleanup.AlreadyMissing,
		"object:locked/fine": projectcleanup.Accepted, "object:locked/held": projectcleanup.Failed, "object:lost/x": projectcleanup.Uncertain,
		"container:/ok": projectcleanup.Accepted, "container:/locked": projectcleanup.NotAttempted, "container:/lost": projectcleanup.NotAttempted,
		"container:/busy": projectcleanup.Failed,
	}
	for key, state := range want {
		if states[key].State != state {
			t.Errorf("%s: %s, want %s (%v)", key, states[key].State, state, states[key].Err)
		}
	}
	if len(states) != len(want) {
		t.Fatalf("outcomes %v", states)
	}
	held := states["object:locked/held"]
	if held.StatusCode != 409 || held.Header.Get("X-Trans-Id") != "tx-conflict" || !strings.Contains(string(held.Body), "Conflict") {
		t.Fatalf("failed evidence %+v", held)
	}
	if !strings.Contains(states["container:/locked"].Reason, "failed") || !strings.Contains(states["container:/lost"].Reason, "uncertain") {
		t.Fatalf("dependents %+v %+v", states["container:/locked"], states["container:/lost"])
	}
	// Every object DELETE precedes every container DELETE; each is sent once.
	wantEvents := []string{"list servers", "list groups", "list /", "list /ok", "list /locked", "list /lost", "list /busy",
		"delete /ok/a%2Fb%20c", "delete /ok/gone", "delete /locked/fine", "delete /locked/held", "delete /lost/x",
		"delete /ok", "delete /busy"}
	if got := cloud.events(); !slices.Equal(got, wantEvents) {
		t.Fatalf("events %v", got)
	}
}
