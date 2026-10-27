package openstack_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func init() {
	cleanupEmptyLists["/orchestration/stacks"] = "stacks"
}

// serveStacks lists the given rows with Python's default query: no
// all-projects, nested or deleted listing.
func (c *cleanupCloud) serveStacks(t *testing.T, rows string) {
	c.Mux.HandleFunc("GET /orchestration/stacks", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Error("stack list query", r.URL)
		}
		c.record("list stacks")
		testcloud.JSON(w, 200, `{"stacks":[`+rows+`]}`)
	})
}

// serveStackMutations answers DELETE and GET on stacks/{name}/{id}. The
// DELETE status and the sequence of GET states (404 when exhausted) are
// keyed by ID.
func (c *cleanupCloud) serveStackMutations(t *testing.T, deletes map[string]int, states map[string][]string) {
	var mu sync.Mutex
	sent := map[string]bool{}
	c.Mux.HandleFunc("DELETE /orchestration/stacks/{name}/{id}", func(w http.ResponseWriter, r *http.Request) {
		name, id := r.PathValue("name"), r.PathValue("id")
		c.record("DELETE " + name + "/" + id)
		mu.Lock()
		if sent[id] {
			t.Error("DELETE sent twice", id)
		}
		sent[id] = true
		mu.Unlock()
		switch code := deletes[id]; code {
		case 0, 204:
			w.WriteHeader(204)
		default:
			w.Header().Set("X-Openstack-Request-Id", "req-"+id)
			testcloud.JSON(w, code, `{"explanation":"stack `+id+`","code":`+strconv.Itoa(code)+`}`)
		}
	})
	c.Mux.HandleFunc("GET /orchestration/stacks/{name}/{id}", func(w http.ResponseWriter, r *http.Request) {
		name, id := r.PathValue("name"), r.PathValue("id")
		c.record("GET " + name + "/" + id)
		mu.Lock()
		queue := states[id]
		var state string
		if len(queue) > 0 {
			state, states[id] = queue[0], queue[1:]
			if len(queue) == 1 {
				// The last state repeats.
				states[id] = queue
			}
		}
		mu.Unlock()
		if state == "" {
			testcloud.JSON(w, 404, `{"explanation":"The Stack could not be found.","code":404}`)
			return
		}
		testcloud.JSON(w, 200, `{"stack":{"id":"`+id+`","stack_name":"`+name+`","stack_status":"`+state+`"}}`)
	})
}

const cleanupStackRows = `{"id":"st-a","stack_name":"web","project":"project","stack_status":"CREATE_COMPLETE","creation_time":"2025-01-01T00:00:00Z","updated_time":null},` +
	`{"id":"st-b","stack_name":"db","stack_status":"CREATE_FAILED","creation_time":"2025-02-01T00:00:00Z","updated_time":"2025-03-01T00:00:00Z"},` +
	`{"id":"st-foreign","stack_name":"x","project":"other"},` +
	`{"id":"st-noname"}`

func TestProjectCleanupOrchestrationPlansTopLevelStacksBeforeCompute(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveStacks(t, cleanupStackRows)
	cloud.serveCompute(t, `{"id":"s1"}`, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"stack:st-a", "stack:st-b", "server:s1"}) {
		t.Fatalf("candidates %v", got)
	}
	if got := stageSummary(plan); !slices.Equal(got, []string{"stack+wait", "server+wait"}) {
		t.Fatalf("stages %v", got)
	}
	web := candidateByID(plan, "stack", "st-a")
	if web.Service != "orchestration" || web.Name != "web" || string(web.CreatedAt) != `"2025-01-01T00:00:00Z"` || string(web.UpdatedAt) != "null" || len(web.DependsOn) != 0 {
		t.Fatalf("stack %+v", web)
	}
	if got := candidateByID(plan, "server", "s1").DependsOn; len(got) != 0 {
		t.Fatalf("server depends on %v", got)
	}
	reasons := exclusionReasons(plan)
	if !strings.Contains(reasons["stack:st-foreign"], "owned by project other (project)") || !strings.Contains(reasons["stack:st-noname"], "no usable stack_name and id") || len(reasons) != 2 {
		t.Fatalf("exclusions %v", reasons)
	}
	for _, u := range plan.Unavailable {
		if u.Service == "orchestration" {
			t.Fatalf("orchestration unavailable: %+v", plan.Unavailable)
		}
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list stacks", "list servers", "list groups"}) {
		t.Fatalf("requests %v", got)
	}
}

func TestProjectCleanupOrchestrationFiltersAndSkip(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveStacks(t, cleanupStackRows)
	cloud.serveCompute(t, ``, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithUpdatedBefore(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"stack:st-b"}) {
		t.Fatalf("updated filter candidates %v", got)
	}
	if reason := exclusionReasons(plan)["stack:st-a"]; !strings.Contains(reason, "updated_at") {
		t.Fatalf("null updated_time: %q", reason)
	}
	plan, err = cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithCreatedBefore(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"stack:st-a"}) {
		t.Fatalf("created filter candidates %v", got)
	}
	var seen []string
	plan, err = cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithSelector(func(c projectcleanup.Candidate) (bool, error) {
		seen = append(seen, c.Kind+":"+c.Name)
		return c.Name == "db", nil
	}))
	if err != nil || !slices.Equal(refIDs(plan.Candidates()), []string{"stack:st-b"}) || !slices.Equal(seen, []string{"stack:web", "stack:db"}) {
		t.Fatalf("selector %v %v %v", plan, seen, err)
	}
	before := len(cloud.events())
	plan, err = cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources(), projectcleanup.WithSkip("orchestration.stack"))
	if err != nil || len(plan.Candidates()) != 0 {
		t.Fatalf("skip %v %v", plan, err)
	}
	if got := cloud.events()[before:]; !slices.Equal(got, []string{"list servers", "list groups"}) {
		t.Fatalf("skipped stacks were listed: %v", got)
	}
}

func TestProjectCleanupOrchestrationExecutesStacksThenCompute(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveStacks(t, `{"id":"ok","stack_name":"n-ok"},{"id":"done","stack_name":"n-done"},{"id":"failing","stack_name":"n-failing"},`+
		`{"id":"gone","stack_name":"n-gone"},{"id":"locked","stack_name":"n-locked"}`)
	cloud.serveStackMutations(t, map[string]int{"gone": 404, "locked": 409},
		map[string][]string{"done": {"DELETE_COMPLETE"}, "failing": {"CREATE_FAILED", "DELETE_FAILED"}})
	cloud.serveCompute(t, `{"id":"s1"}`, ``)
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("delete " + r.PathValue("id"))
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("get " + r.PathValue("id"))
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
	})
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithDeletionWait(30*time.Second))
	if !errors.Is(err, projectcleanup.ErrIncomplete) || result == nil {
		t.Fatalf("result %v err %v", result, err)
	}
	outcomes := map[string]projectcleanup.Outcome{}
	for _, o := range result.Outcomes {
		outcomes[o.Candidate.ID] = o
	}
	want := map[string]projectcleanup.State{"ok": projectcleanup.Deleted, "done": projectcleanup.Deleted, "failing": projectcleanup.Accepted,
		"gone": projectcleanup.AlreadyMissing, "locked": projectcleanup.Failed, "s1": projectcleanup.Deleted}
	for id, state := range want {
		if outcomes[id].State != state {
			t.Errorf("%s: %s, want %s (%v / %v)", id, outcomes[id].State, state, outcomes[id].Err, outcomes[id].WaitErr)
		}
	}
	if failing := outcomes["failing"]; !errors.Is(failing.WaitErr, resource.ErrFailedState) || !strings.Contains(failing.WaitErr.Error(), "DELETE_FAILED") {
		t.Fatalf("failing wait %v", failing.WaitErr)
	}
	if locked := outcomes["locked"]; locked.StatusCode != 409 || locked.Header.Get("X-Openstack-Request-Id") != "req-locked" {
		t.Fatalf("locked evidence %+v", locked)
	}
	// Each stack gets one DELETE on its name/ID route; the waits follow the
	// whole stage, and compute starts after them even when a stack failed,
	// as Python's dependency walk continues after a failed service.
	wantEvents := []string{"list stacks", "list servers", "list groups",
		"DELETE n-ok/ok", "DELETE n-done/done", "DELETE n-failing/failing", "DELETE n-gone/gone", "DELETE n-locked/locked",
		"GET n-ok/ok", "GET n-done/done", "GET n-failing/failing", "GET n-failing/failing",
		"delete s1", "get s1"}
	if got := cloud.events(); !slices.Equal(got, wantEvents) {
		t.Fatalf("events\n got %v\nwant %v", got, wantEvents)
	}
}

func TestProjectCleanupOrchestrationWaitTimeoutKeepsAcceptedStack(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveStacks(t, `{"id":"slow","stack_name":"n-slow"}`)
	cloud.serveStackMutations(t, nil, map[string][]string{"slow": {"DELETE_IN_PROGRESS"}})
	cloud.serveCompute(t, ``, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithDeletionWait(100*time.Millisecond))
	if !errors.Is(err, projectcleanup.ErrIncomplete) || len(result.Outcomes) != 1 {
		t.Fatalf("result %v err %v", result, err)
	}
	slow := result.Outcomes[0]
	if slow.State != projectcleanup.Accepted || slow.WaitErr == nil || errors.Is(slow.WaitErr, resource.ErrFailedState) {
		t.Fatalf("slow %+v", slow)
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list stacks", "list servers", "list groups", "DELETE n-slow/slow", "GET n-slow/slow"}) {
		t.Fatalf("events %v", got)
	}
}
