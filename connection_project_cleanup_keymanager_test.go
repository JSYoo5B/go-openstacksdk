package openstack_test

import (
	"context"
	"errors"
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
)

// newKeyManagerCleanupCloud adds a key-manager catalog entry at
// <server>/key-manager and routes the key manager client there; compute is
// served empty so that only Barbican resources are planned.
func newKeyManagerCleanupCloud(t *testing.T) *cleanupCloud {
	t.Helper()
	cloud := newCleanupCloud(t)
	setCleanupScope(t, cloud.Cloud, "project", "orchestration", "load-balancer", "key-manager")
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"),
		sdk.WithEndpoint(sdk.Orchestration, cloud.Server.URL+"/orchestration"),
		sdk.WithEndpoint(sdk.LoadBalancer, cloud.Server.URL+"/load-balancer"),
		sdk.WithEndpoint(sdk.KeyManager, cloud.Server.URL+"/key-manager"),
	)
	if err != nil {
		t.Fatal(err)
	}
	cloud.conn = conn
	return cloud
}

// serveBarbicanLists serves the container and secret lists. A list query
// that widens the scope fails the test.
func (c *cleanupCloud) serveBarbicanLists(t *testing.T, containers string, secretPages ...string) {
	c.Mux.HandleFunc("GET /key-manager/v1/containers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Error("container list query", r.URL)
		}
		c.record("list containers")
		testcloud.JSON(w, 200, `{"containers":[`+containers+`],"total":0}`)
	})
	c.Mux.HandleFunc("GET /key-manager/v1/secrets", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Has("acl_only") || query.Has("all_projects") || query.Has("all_tenants") {
			t.Error("widened secret list", r.URL)
		}
		page := 0
		if query.Get("offset") == "2" {
			page = 1
		}
		c.record("list secrets " + r.URL.RawQuery)
		body := `{"secrets":[` + secretPages[page] + `]`
		if page+1 < len(secretPages) {
			body += `,"next":"` + c.Server.URL + `/key-manager/v1/secrets?limit=2&offset=2"`
		}
		testcloud.JSON(w, 200, body+`}`)
	})
}

func barbicanRef(cloud *cleanupCloud, plural, id string) string {
	return cloud.Server.URL + "/key-manager/v1/" + plural + "/" + id
}

func TestProjectCleanupKeyManagerDiscoversContainersThenSecrets(t *testing.T) {
	cloud := newKeyManagerCleanupCloud(t)
	ref := func(plural, id string) string { return barbicanRef(cloud, plural, id) }
	cloud.serveBarbicanLists(t,
		`{"container_ref":"`+ref("containers", "c1")+`","name":"bundle","created":"2025-03-09T12:46:43","updated":"2025-03-09T12:46:43","secret_refs":[{"name":"k","secret_ref":"`+ref("secrets", "s1")+`"}]}`+
			`,{"container_ref":"not a url","name":"broken"}`+
			`,{"name":"no-ref"}`+
			`,{"container_ref":"`+ref("containers", "c2")+`","name":"foreign","project_id":"other"}`,
		`{"secret_ref":"`+ref("secrets", "s1")+`?x=1","name":"key","created":"2025-01-01T00:00:00.123456","updated":"2025-01-02T00:00:00"},{"secret_ref":"/v1/secrets/rel","name":"relative"}`,
		`{"secret_ref":"`+ref("secrets", "s2")+`","name":"later","created":null}`)
	cloud.serveCompute(t, ``, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"container:c1", "secret:s1", "secret:s2"}) {
		t.Fatalf("candidates %v", got)
	}
	var stages []string
	for _, stage := range plan.Stages {
		stages = append(stages, stage.Service+"."+stage.Kind)
		if stage.WaitForAbsence {
			t.Errorf("%s waits for absence", stage.Kind)
		}
	}
	if !slices.Equal(stages, []string{"key_manager.container", "key_manager.secret"}) {
		t.Fatalf("stages %v", stages)
	}
	container := plan.Stages[0].Candidates[0]
	if container.Name != "bundle" || string(container.CreatedAt) != `"2025-03-09T12:46:43"` || len(container.DependsOn) != 0 || !strings.Contains(string(container.Raw), "secret_refs") {
		t.Fatalf("container %+v", container)
	}
	reasons := map[string]string{}
	for _, e := range plan.Excluded {
		reasons[e.Kind+":"+e.Name] = e.Reason
	}
	for key, want := range map[string]string{
		"container:broken": "container_ref cannot be converted to an id", "container:no-ref": "container_ref is not available",
		"container:foreign": "owned by project other", "secret:relative": "secret_ref cannot be converted to an id",
	} {
		if !strings.Contains(reasons[key], want) {
			t.Errorf("%s: %q, want %q", key, reasons[key], want)
		}
	}
	if len(reasons) != 4 {
		t.Fatalf("exclusions %v", reasons)
	}
	for _, u := range plan.Unavailable {
		if u.Service == "key_manager" {
			t.Fatalf("key_manager unavailable: %+v", u)
		}
	}
	want := []string{"list servers", "list groups", "list containers", "list secrets ", "list secrets limit=2&offset=2"}
	if got := cloud.events(); !slices.Equal(got, want) {
		t.Fatalf("requests %v", got)
	}
}

func TestProjectCleanupKeyManagerFiltersSelectorAndSkipNames(t *testing.T) {
	cloud := newKeyManagerCleanupCloud(t)
	ref := func(plural, id string) string { return barbicanRef(cloud, plural, id) }
	cloud.serveBarbicanLists(t,
		`{"container_ref":"`+ref("containers", "old")+`","name":"old","created":"2025-06-01T00:00:00","updated":"2025-06-01T00:00:00"}`+
			`,{"container_ref":"`+ref("containers", "new")+`","name":"new","created":"2026-02-01T00:00:00","updated":"2026-02-01T00:00:00"}`,
		`{"secret_ref":"`+ref("secrets", "edge")+`","name":"edge","created":"2026-01-01T00:00:00","updated":"2025-06-01T00:00:00Z"}`+
			`,{"secret_ref":"`+ref("secrets", "missing")+`","name":"missing","updated":"2025-06-01T00:00:00"}`+
			`,{"secret_ref":"`+ref("secrets", "garbled")+`","name":"garbled","created":"soon","updated":"2025-06-01T00:00:00"}`+
			`,{"secret_ref":"`+ref("secrets", "stale")+`","name":"stale","created":"2025-06-01T00:00:00","updated":"2026-06-01T00:00:00"}`)
	cloud.serveCompute(t, ``, ``)
	ctx := context.Background()
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	plan, err := cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithCreatedBefore(cutoff), projectcleanup.WithUpdatedBefore(cutoff))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"container:old", "secret:edge"}) {
		t.Fatalf("candidates %v", got)
	}
	reasons := exclusionReasons(plan)
	for id, want := range map[string]string{
		"container:new": "after the cutoff", "secret:missing": "created_at is not available",
		"secret:garbled": `"soon" cannot be parsed`, "secret:stale": "updated_at",
	} {
		if !strings.Contains(reasons[id], want) {
			t.Errorf("%s: %q, want %q", id, reasons[id], want)
		}
	}

	var seen []string
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithSelector(func(c projectcleanup.Candidate) (bool, error) {
		if c.Service == "key_manager" {
			seen = append(seen, c.Kind+":"+c.ID)
		}
		return c.Kind == "secret" && c.ID == "stale", nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"secret:stale"}) || len(seen) != 6 {
		t.Fatalf("selector candidates %v seen %v", got, seen)
	}

	before := len(cloud.events())
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(), projectcleanup.WithSkip("key_manager.container"))
	if err != nil || !slices.Equal(refIDs(plan.Candidates()), []string{"secret:edge", "secret:missing", "secret:garbled", "secret:stale"}) {
		t.Fatalf("skip containers: %v %v", refIDs(plan.Candidates()), err)
	}
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(), projectcleanup.WithSkip("key_manager.secret"))
	if err != nil || !slices.Equal(refIDs(plan.Candidates()), []string{"container:old", "container:new"}) {
		t.Fatalf("skip secrets: %v %v", refIDs(plan.Candidates()), err)
	}
	want := []string{"list servers", "list groups", "list secrets ", "list servers", "list groups", "list containers"}
	if got := cloud.events()[before:]; !slices.Equal(got, want) {
		t.Fatalf("requests %v", got)
	}
}

func TestProjectCleanupKeyManagerExecuteReportsEveryState(t *testing.T) {
	cloud := newKeyManagerCleanupCloud(t)
	ref := func(plural, id string) string { return barbicanRef(cloud, plural, id) }
	var deletes sync.Map
	handler := func(kind string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			cloud.record("delete " + kind + " " + id)
			v, _ := deletes.LoadOrStore(kind+id, new(atomic.Int32))
			if v.(*atomic.Int32).Add(1) > 1 {
				t.Error("DELETE sent twice", kind, id)
			}
			switch id {
			case "gone":
				testcloud.JSON(w, 404, `{"code":404,"title":"Not Found"}`)
			case "denied":
				w.Header().Set("X-Openstack-Request-Id", "req-denied")
				testcloud.JSON(w, 403, `{"code":403,"title":"Forbidden","description":"denied"}`)
			case "lost":
				conn, _, err := http.NewResponseController(w).Hijack()
				if err == nil {
					_ = conn.Close()
				}
			default:
				w.WriteHeader(204)
			}
		}
	}
	cloud.Mux.HandleFunc("DELETE /key-manager/v1/containers/{id}", handler("container"))
	cloud.Mux.HandleFunc("DELETE /key-manager/v1/secrets/{id}", handler("secret"))
	cloud.serveBarbicanLists(t,
		`{"container_ref":"`+ref("containers", "ok")+`"},{"container_ref":"`+ref("containers", "gone")+`"},{"container_ref":"`+ref("containers", "denied")+`"},{"container_ref":"`+ref("containers", "lost")+`"}`,
		`{"secret_ref":"`+ref("secrets", "ok")+`"},{"secret_ref":"`+ref("secrets", "gone")+`"}`)
	cloud.serveCompute(t, ``, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	var progress []string
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithProgress(func(e projectcleanup.Event) error {
		progress = append(progress, e.Outcome.Candidate.Kind+":"+e.Outcome.Candidate.ID+"="+string(e.Outcome.State))
		return nil
	}))
	if !errors.Is(err, projectcleanup.ErrIncomplete) || result == nil {
		t.Fatalf("result %v err %v", result, err)
	}
	// Python orders containers before secrets but declares no dependency
	// between them, so a failed or uncertain container does not hold back
	// any secret.
	wantProgress := []string{"container:ok=accepted", "container:gone=already_missing", "container:denied=failed", "container:lost=uncertain",
		"secret:ok=accepted", "secret:gone=already_missing"}
	if !slices.Equal(progress, wantProgress) {
		t.Fatalf("progress %v", progress)
	}
	var denied projectcleanup.Outcome
	for _, o := range result.Outcomes {
		if o.Candidate.Kind == "container" && o.Candidate.ID == "denied" {
			denied = o
		}
		if o.WaitErr != nil {
			t.Errorf("%s waited: %v", o.Candidate.Ref, o.WaitErr)
		}
	}
	if denied.StatusCode != 403 || denied.Header.Get("X-Openstack-Request-Id") != "req-denied" || !strings.Contains(string(denied.Body), "denied") || denied.Err == nil {
		t.Fatalf("failed evidence %+v", denied)
	}
	wantEvents := []string{"list servers", "list groups", "list containers", "list secrets ",
		"delete container ok", "delete container gone", "delete container denied", "delete container lost", "delete secret ok", "delete secret gone"}
	if got := cloud.events(); !slices.Equal(got, wantEvents) {
		t.Fatalf("events %v", got)
	}

	// A progress error stops new mutations: the remaining candidates are
	// not attempted and nothing is replayed.
	deletes.Clear()
	stop := errors.New("stop")
	result, err = cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithProgress(func(projectcleanup.Event) error { return stop }))
	if !errors.Is(err, stop) || result.Count(projectcleanup.NotAttempted) != 5 || result.Outcomes[0].State != projectcleanup.Accepted {
		t.Fatalf("stop: %v %+v", err, result)
	}
}
