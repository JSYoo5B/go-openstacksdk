package connectiontest_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
)

// serveImages lists Glance pages: the first page links to the second with
// Glance's top-level next path. Only the owner query of Python is accepted.
func (c *cleanupCloud) serveImages(t *testing.T, first, second string) {
	c.Mux.HandleFunc("GET /image/v2/images", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("owner") != "project" {
			t.Error("image listing without the project owner filter", r.URL)
		}
		for key := range query {
			if key != "owner" && key != "marker" {
				t.Error("unexpected image query", key, r.URL)
			}
		}
		if query.Get("marker") == "" {
			c.record("list images")
			testcloud.JSON(w, 200, `{"images":[`+first+`],"first":"/v2/images?owner=project","next":"/v2/images?marker=page1&owner=project","schema":"/v2/schemas/images"}`)
			return
		}
		if query.Get("marker") != "page1" {
			t.Error("unexpected marker", r.URL)
		}
		c.record("list images page 2")
		testcloud.JSON(w, 200, `{"images":[`+second+`],"first":"/v2/images?owner=project","schema":"/v2/schemas/images"}`)
	})
}

func TestProjectCleanupImageDiscoversOnlyOwnedImages(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveImages(t,
		`{"id":"i-private","name":"private","owner":"project","visibility":"private","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-02T00:00:00Z"},`+
			`{"id":"i-public","name":"own-public","owner":"project","visibility":"public","protected":true},`+
			`{"id":"i-foreign-public","name":"cirros","owner":"admin-project","visibility":"public"}`,
		`{"id":"i-shared","name":"shared-in","owner":"other","visibility":"shared"},`+
			`{"id":"i-community","name":"community","owner":"other","visibility":"community"},`+
			`{"id":"i-ownerless","name":"no-owner","visibility":"private"},`+
			`{"id":"i-null-owner","owner":null},`+
			`{"name":"no-id","owner":"project"}`)
	cloud.serveCompute(t, `{"id":"s1","tenant_id":"project"}`, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"server:s1", "image:i-private", "image:i-public"}) {
		t.Fatalf("candidates %v", got)
	}
	last := plan.Stages[len(plan.Stages)-1]
	if last.Service != "image" || last.Kind != "image" || last.WaitForAbsence || len(last.Candidates) != 2 {
		t.Fatalf("image stage %+v", last)
	}
	private := last.Candidates[0]
	if string(private.CreatedAt) != `"2025-01-01T00:00:00Z"` || string(private.UpdatedAt) != `"2025-01-02T00:00:00Z"` || len(private.DependsOn) != 0 ||
		!strings.Contains(string(private.Raw), `"visibility":"private"`) || private.Name != "private" || private.Reason != "all resources" {
		t.Fatalf("private image %+v", private)
	}
	reasons := exclusionReasons(plan)
	for id, want := range map[string]string{
		"image:i-foreign-public": "owned by project admin-project (owner)",
		"image:i-shared":         "owned by project other (owner)",
		"image:i-community":      "owned by project other (owner)",
		"image:i-ownerless":      "owner is not reported",
		"image:i-null-owner":     "owner is not reported",
		"image:":                 "row has no usable id",
	} {
		if reasons[id] != want {
			t.Errorf("%s: %q, want %q", id, reasons[id], want)
		}
	}
	if len(reasons) != 6 {
		t.Fatalf("exclusions %v", reasons)
	}
	for _, u := range plan.Unavailable {
		if u.Service == "image" {
			t.Fatalf("image reported unavailable: %+v", plan.Unavailable)
		}
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list servers", "list groups", "list images", "list images page 2"}) {
		t.Fatalf("requests %v", got)
	}
}

func TestProjectCleanupImageFiltersAndSkipName(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveImages(t, strings.Join([]string{
		`{"id":"old","owner":"project","created_at":"2025-06-01T00:00:00Z","updated_at":"2025-06-02T00:00:00Z"}`,
		`{"id":"edge","owner":"project","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`,
		`{"id":"new","owner":"project","created_at":"2026-03-01T00:00:00Z","updated_at":"2026-03-01T00:00:00Z"}`,
		`{"id":"missing","owner":"project","updated_at":"2025-06-02T00:00:00Z"}`,
		`{"id":"garbled","owner":"project","created_at":"yesterday","updated_at":"2025-06-02T00:00:00Z"}`,
		`{"id":"stale","owner":"project","created_at":"2025-06-01T00:00:00Z","updated_at":"2026-06-02T00:00:00Z"}`,
		`{"id":"noupdate","owner":"project","created_at":"2025-06-01T00:00:00Z","updated_at":null}`,
	}, ","), ``)
	cloud.serveCompute(t, ``, ``)
	ctx := context.Background()
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	plan, err := cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithCreatedBefore(cutoff), projectcleanup.WithUpdatedBefore(cutoff))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"image:old", "image:edge"}) {
		t.Fatalf("candidates %v", got)
	}
	reasons := exclusionReasons(plan)
	for id, want := range map[string]string{
		"image:new": "after the cutoff", "image:missing": "created_at is not available", "image:garbled": `"yesterday" cannot be parsed`,
		"image:stale": "updated_at", "image:noupdate": "updated_at is not available",
	} {
		if !strings.Contains(reasons[id], want) {
			t.Errorf("%s: %q, want %q", id, reasons[id], want)
		}
	}
	var seen []string
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithSelector(func(c projectcleanup.Candidate) (bool, error) {
		seen = append(seen, c.Service+"."+c.Kind+":"+c.ID)
		return c.ID == "new", nil
	}))
	if err != nil || !slices.Equal(refIDs(plan.Candidates()), []string{"image:new"}) || len(seen) != 7 || seen[0] != "image.image:old" {
		t.Fatalf("selector: %v %v %v", plan, seen, err)
	}
	before := len(cloud.events())
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(), projectcleanup.WithSkip("image.image"))
	if err != nil || len(plan.Candidates()) != 0 {
		t.Fatalf("skip: %+v %v", plan, err)
	}
	if got := cloud.events()[before:]; !slices.Equal(got, []string{"list servers", "list groups"}) {
		t.Fatalf("skipped image was listed: %v", got)
	}
}

func TestProjectCleanupImageExecuteReportsEveryState(t *testing.T) {
	cloud := newCleanupCloud(t)
	var mu sync.Mutex
	deletes := map[string]int{}
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("delete server " + r.PathValue("id"))
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("get server " + r.PathValue("id"))
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
	})
	cloud.Mux.HandleFunc("DELETE /image/v2/images/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		cloud.record("delete image " + id)
		mu.Lock()
		deletes[id]++
		again := deletes[id] > 1
		mu.Unlock()
		if again {
			t.Error("DELETE sent twice", id)
		}
		switch id {
		case "gone":
			testcloud.JSON(w, 404, `{"message":"No image found"}`)
		case "protected":
			w.Header().Set("X-Openstack-Request-Id", "req-protected")
			testcloud.JSON(w, 403, `{"message":"Image is protected"}`)
		case "lost":
			conn, _, err := http.NewResponseController(w).Hijack()
			if err == nil {
				_ = conn.Close()
			}
		default:
			w.WriteHeader(204)
		}
	})
	cloud.serveImages(t, `{"id":"ok","owner":"project"},{"id":"gone","owner":"project"},{"id":"protected","owner":"project","protected":true},{"id":"lost","owner":"project"}`, ``)
	cloud.serveCompute(t, `{"id":"s1"}`, ``)
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
	want := map[string]projectcleanup.State{"s1": projectcleanup.Deleted, "ok": projectcleanup.Accepted, "gone": projectcleanup.AlreadyMissing,
		"protected": projectcleanup.Failed, "lost": projectcleanup.Uncertain}
	for id, state := range want {
		if states[id].State != state {
			t.Errorf("%s: %s, want %s (%v)", id, states[id].State, state, states[id].Err)
		}
	}
	protected := states["protected"]
	if protected.StatusCode != 403 || protected.Header.Get("X-Openstack-Request-Id") != "req-protected" || !strings.Contains(string(protected.Body), "protected") {
		t.Fatalf("failed evidence %+v", protected)
	}
	if states["lost"].Err == nil || states["ok"].WaitErr != nil {
		t.Fatalf("outcomes %+v", states)
	}
	// Servers are deleted and observed absent first; images are not waited for.
	wantEvents := []string{"list servers", "list groups", "list images", "list images page 2", "delete server s1", "get server s1",
		"delete image ok", "delete image gone", "delete image protected", "delete image lost"}
	if got := cloud.events(); !slices.Equal(got, wantEvents) {
		t.Fatalf("events %v", got)
	}
	wantProgress := []string{"s1=accepted", "s1=deleted", "ok=accepted", "gone=already_missing", "protected=failed", "lost=uncertain"}
	if !slices.Equal(progress, wantProgress) {
		t.Fatalf("progress %v", progress)
	}
}

func TestProjectCleanupImageListErrorIsServiceError(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.Mux.HandleFunc("GET /image/v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 503, `{"message":"unavailable"}`)
	})
	cloud.serveCompute(t, `{"id":"s1"}`, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	var serviceErr *projectcleanup.ServiceError
	if plan == nil || !errors.As(err, &serviceErr) || serviceErr.Service != "image" || !strings.Contains(err.Error(), "list images") {
		t.Fatalf("plan %+v err %v", plan, err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"server:s1"}) {
		t.Fatalf("candidates %v", got)
	}
	if _, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan); !errors.Is(err, projectcleanup.ErrIncompletePlan) {
		t.Fatalf("incomplete plan executed: %v", err)
	}
}
