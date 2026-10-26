package openstack_test

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

func init() {
	cleanupEmptyLists["/volume/v3/project/backups/detail"] = "backups"
	cleanupEmptyLists["/volume/v3/project/snapshots/detail"] = "snapshots"
	cleanupEmptyLists["/volume/v3/project/volumes/detail"] = "volumes"
}

// serveBlockStorage lists the given rows and checks that no list asks for
// other projects. Backups must be requested newest first, as Python does.
func (c *cleanupCloud) serveBlockStorage(t *testing.T, backups, snapshots, volumes string) {
	list := func(pattern, plural, rows string, check func(*http.Request)) {
		c.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Has("all_tenants") || r.URL.Query().Has("all_projects") || r.URL.Query().Has("project_id") {
				t.Error("cross-project listing", r.URL)
			}
			if check != nil {
				check(r)
			}
			c.record("list " + plural)
			testcloud.JSON(w, 200, `{"`+plural+`":[`+rows+`]}`)
		})
	}
	list("GET /volume/v3/project/backups/detail", "backups", backups, func(r *http.Request) {
		if q := r.URL.Query(); q.Get("sort_key") != "created_at" || q.Get("sort_dir") != "desc" {
			t.Error("backups not listed newest first", r.URL)
		}
	})
	list("GET /volume/v3/project/snapshots/detail", "snapshots", snapshots, nil)
	list("GET /volume/v3/project/volumes/detail", "volumes", volumes, nil)
}

// A chain on volume v1: full <- i1 <- i2, newest first as listed.
const cleanupBackupChain = `{"id":"b-i2","volume_id":"v1","is_incremental":true,"has_dependent_backups":false,"created_at":"2025-01-03T00:00:00.000000"},` +
	`{"id":"b-other","volume_id":"v2","is_incremental":false,"has_dependent_backups":false,"created_at":"2025-01-02T12:00:00.000000"},` +
	`{"id":"b-i1","volume_id":"v1","is_incremental":true,"has_dependent_backups":true,"created_at":"2025-01-02T00:00:00.000000"},` +
	`{"id":"b-full","volume_id":"v1","is_incremental":false,"has_dependent_backups":true,"created_at":"2025-01-01T00:00:00.000000"}`

func stageSummary(plan *projectcleanup.Plan) []string {
	var out []string
	for _, s := range plan.Stages {
		item := s.Kind
		if s.Level > 0 {
			item += "/" + string(rune('0'+s.Level))
		}
		if s.WaitForAbsence {
			item += "+wait"
		}
		out = append(out, item)
	}
	return out
}

func candidateByID(plan *projectcleanup.Plan, kind, id string) projectcleanup.Candidate {
	for _, c := range plan.Candidates() {
		if c.Kind == kind && c.ID == id {
			return c
		}
	}
	return projectcleanup.Candidate{}
}

func TestProjectCleanupBlockStoragePlansBackupLevelsSnapshotsAndVolumes(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveBlockStorage(t,
		cleanupBackupChain+
			`,{"id":"b-foreign","volume_id":"v9","os-backup-project-attr:project_id":"other"}`+
			`,{"id":"b-orphan","volume_id":"v4","is_incremental":false,"has_dependent_backups":true,"created_at":"2024-12-01T00:00:00Z"}`+
			`,{"name":"no-id"}`,
		`{"id":"sn1","volume_id":"v1","os-extended-snapshot-attributes:project_id":"project"},{"id":"sn-foreign","volume_id":"v1","os-extended-snapshot-attributes:project_id":"other"}`,
		`{"id":"v1","os-vol-tenant-attr:tenant_id":"project","attachments":[{"server_id":"s1"},{"server_id":"s-unknown"}]},{"id":"v2","attachments":[]},{"id":"v-foreign","os-vol-tenant-attr:tenant_id":"other"}`)
	cloud.serveCompute(t, `{"id":"s1"}`, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"server:s1", "backup:b-i2", "backup:b-other", "backup:b-i1", "backup:b-full", "snapshot:sn1", "volume:v1", "volume:v2"}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, want) {
		t.Fatalf("candidates %v", got)
	}
	if got := stageSummary(plan); !slices.Equal(got, []string{"server+wait", "backup+wait", "backup/1+wait", "backup/2+wait", "snapshot+wait", "volume"}) {
		t.Fatalf("stages %v", got)
	}
	backup := func(id string) projectcleanup.Ref {
		return projectcleanup.Ref{Service: "block_storage", Kind: "backup", ID: id}
	}
	if got := candidateByID(plan, "backup", "b-i1").DependsOn; !slices.Equal(got, []projectcleanup.Ref{backup("b-i2")}) {
		t.Fatalf("b-i1 depends on %v", got)
	}
	full := candidateByID(plan, "backup", "b-full")
	if !slices.Equal(full.DependsOn, []projectcleanup.Ref{backup("b-i2"), backup("b-i1")}) || !strings.Contains(full.Reason, "possible dependent backups") {
		t.Fatalf("b-full %+v", full)
	}
	if got := candidateByID(plan, "backup", "b-other").DependsOn; len(got) != 0 {
		t.Fatalf("b-other depends on %v", got)
	}
	wantVolume := []projectcleanup.Ref{{Service: "block_storage", Kind: "snapshot", ID: "sn1"}, {Service: "compute", Kind: "server", ID: "s1"}}
	if got := candidateByID(plan, "volume", "v1").DependsOn; !slices.Equal(got, wantVolume) {
		t.Fatalf("v1 depends on %v", got)
	}
	if string(candidateByID(plan, "backup", "b-i2").CreatedAt) != `"2025-01-03T00:00:00.000000"` {
		t.Fatalf("created_at not kept")
	}
	reasons := exclusionReasons(plan)
	for id, reason := range map[string]string{
		"backup:b-foreign": "owned by project other", "backup:b-orphan": "dependent backups that are not listed",
		"backup:": "no usable id", "snapshot:sn-foreign": "owned by project other", "volume:v-foreign": "owned by project other",
	} {
		if !strings.Contains(reasons[id], reason) {
			t.Errorf("%s: %q, want %q", id, reasons[id], reason)
		}
	}
	if len(reasons) != 5 {
		t.Fatalf("exclusions %v", reasons)
	}
	for _, u := range plan.Unavailable {
		if u.Service == "block_storage" {
			t.Fatalf("block storage unavailable: %+v", plan.Unavailable)
		}
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list servers", "list groups", "list backups", "list snapshots", "list volumes"}) {
		t.Fatalf("requests %v", got)
	}
}

func TestProjectCleanupBlockStorageKeepsBackupsWithKeptDependents(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveBlockStorage(t, cleanupBackupChain+
		`,{"id":"b-garbled","volume_id":"v5","has_dependent_backups":true,"created_at":"soon"}`, ``, ``)
	cloud.serveCompute(t, ``, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithSelector(func(c projectcleanup.Candidate) (bool, error) {
		return c.ID != "b-i2", nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"backup:b-other"}) {
		t.Fatalf("candidates %v", got)
	}
	reasons := exclusionReasons(plan)
	for id, reason := range map[string]string{
		"backup:b-i2": "rejected by selector", "backup:b-i1": "not cleanup candidates: b-i2",
		"backup:b-full": "not cleanup candidates: b-i2, b-i1", "backup:b-garbled": "created_at cannot order them",
	} {
		if !strings.Contains(reasons[id], reason) {
			t.Errorf("%s: %q, want %q", id, reasons[id], reason)
		}
	}

	// A time filter that keeps only the older backups keeps their parents
	// too, because their newer dependents stay.
	cutoff := time.Date(2025, 1, 2, 6, 0, 0, 0, time.UTC)
	plan, err = cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithCreatedBefore(cutoff))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); len(got) != 0 {
		t.Fatalf("candidates %v", got)
	}
	reasons = exclusionReasons(plan)
	if !strings.Contains(reasons["backup:b-i2"], "after the cutoff") || !strings.Contains(reasons["backup:b-i1"], "b-i2") || !strings.Contains(reasons["backup:b-full"], "b-i2, b-i1") {
		t.Fatalf("exclusions %v", reasons)
	}
}

func TestProjectCleanupBlockStorageTimeFiltersAndSkipNames(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveBlockStorage(t,
		`{"id":"b-old","created_at":"2025-01-01T00:00:00","updated_at":"2025-01-01T00:00:00"},{"id":"b-new","created_at":"2026-02-01T00:00:00","updated_at":"2026-02-01T00:00:00"}`,
		`{"id":"sn-old","volume_id":"v-old","created_at":"2025-01-01T00:00:00","updated_at":null},{"id":"sn-missing","volume_id":"v-old"}`,
		`{"id":"v-old","created_at":"2025-01-01T00:00:00.000000","updated_at":"2025-01-01T00:00:00.000000"},{"id":"v-null","created_at":null}`)
	cloud.serveCompute(t, ``, ``)
	ctx := context.Background()
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	plan, err := cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithCreatedBefore(cutoff))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"backup:b-old", "snapshot:sn-old", "volume:v-old"}) {
		t.Fatalf("candidates %v", got)
	}
	reasons := exclusionReasons(plan)
	if !strings.Contains(reasons["backup:b-new"], "after the cutoff") || !strings.Contains(reasons["snapshot:sn-missing"], "created_at is not available") ||
		!strings.Contains(reasons["volume:v-null"], "created_at is not available") {
		t.Fatalf("exclusions %v", reasons)
	}
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithUpdatedBefore(cutoff))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"backup:b-old", "volume:v-old"}) {
		t.Fatalf("updated filter candidates %v", got)
	}

	// Each skip name skips only its own kind and its list.
	for _, tc := range []struct {
		skip  string
		lists []string
		want  []string
	}{
		{"block_storage.backup", []string{"list snapshots", "list volumes"}, []string{"snapshot:sn-old", "snapshot:sn-missing", "volume:v-old", "volume:v-null"}},
		{"block_storage.snapshot", []string{"list backups", "list volumes"}, []string{"backup:b-old", "backup:b-new", "volume:v-old", "volume:v-null"}},
		{"block_storage.volume", []string{"list backups", "list snapshots"}, []string{"backup:b-old", "backup:b-new", "snapshot:sn-old", "snapshot:sn-missing"}},
	} {
		before := len(cloud.events())
		plan, err := cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(), projectcleanup.WithSkip(tc.skip))
		if err != nil {
			t.Fatal(err)
		}
		if got := refIDs(plan.Candidates()); !slices.Equal(got, tc.want) {
			t.Errorf("%s: candidates %v", tc.skip, got)
		}
		if tc.skip == "block_storage.snapshot" {
			if deps := candidateByID(plan, "volume", "v-old").DependsOn; len(deps) != 0 {
				t.Errorf("skipped snapshots still ordered: %v", deps)
			}
		}
		var lists []string
		for _, e := range cloud.events()[before:] {
			if e != "list servers" && e != "list groups" {
				lists = append(lists, e)
			}
		}
		if !slices.Equal(lists, tc.lists) {
			t.Errorf("%s: lists %v", tc.skip, lists)
		}
	}
}

// blockStorageMutations serves block storage DELETE and GET requests. A
// DELETE is answered by deleteStatus (0 drops the connection); a GET
// answers 404 unless getStatus names another status for the ID.
func (c *cleanupCloud) blockStorageMutations(t *testing.T, deleteStatus, getStatus map[string]int) {
	var mu sync.Mutex
	sent := map[string]bool{}
	for _, collection := range []string{"backups", "snapshots", "volumes"} {
		c.Mux.HandleFunc("DELETE /volume/v3/project/"+collection+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			c.record("delete " + id)
			mu.Lock()
			if sent[id] {
				t.Error("DELETE sent twice", id)
			}
			sent[id] = true
			mu.Unlock()
			status, ok := deleteStatus[id]
			switch {
			case !ok:
				w.WriteHeader(202)
			case status == 0:
				conn, _, err := http.NewResponseController(w).Hijack()
				if err == nil {
					_ = conn.Close()
				}
			default:
				testcloud.JSON(w, status, `{"badRequest":{"message":"refused `+id+`"}}`)
			}
		})
		c.Mux.HandleFunc("GET /volume/v3/project/"+collection+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			c.record("get " + id)
			if status, ok := getStatus[id]; ok {
				testcloud.JSON(w, status, `{"message":"poll `+id+`"}`)
				return
			}
			testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
		})
	}
	c.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		c.record("delete " + id)
		if status, ok := deleteStatus[id]; ok {
			testcloud.JSON(w, status, `{"conflictingRequest":{"message":"locked"}}`)
			return
		}
		w.WriteHeader(204)
	})
	c.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		c.record("get " + r.PathValue("id"))
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
	})
}

func TestProjectCleanupBlockStorageExecutesLevelsAndWaits(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveBlockStorage(t, cleanupBackupChain, `{"id":"sn1","volume_id":"v1"}`, `{"id":"v1","attachments":[{"server_id":"s1"}]},{"id":"v2"}`)
	cloud.serveCompute(t, `{"id":"s1"}`, ``)
	cloud.blockStorageMutations(t, nil, nil)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithDeletionWait(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	wantEvents := []string{"list servers", "list groups", "list backups", "list snapshots", "list volumes",
		"delete s1", "get s1",
		"delete b-i2", "delete b-other", "get b-i2", "get b-other",
		"delete b-i1", "get b-i1",
		"delete b-full", "get b-full",
		"delete sn1", "get sn1",
		"delete v1", "delete v2"}
	if got := cloud.events(); !slices.Equal(got, wantEvents) {
		t.Fatalf("events %v", got)
	}
	states := map[string]projectcleanup.State{}
	for _, o := range result.Outcomes {
		states[o.Candidate.ID] = o.State
	}
	want := map[string]projectcleanup.State{"s1": projectcleanup.Deleted, "b-i2": projectcleanup.Deleted, "b-other": projectcleanup.Deleted, "b-i1": projectcleanup.Deleted,
		"b-full": projectcleanup.Deleted, "sn1": projectcleanup.Deleted, "v1": projectcleanup.Accepted, "v2": projectcleanup.Accepted}
	if len(states) != len(want) {
		t.Fatalf("outcomes %v", states)
	}
	for id, state := range want {
		if states[id] != state {
			t.Errorf("%s: %s, want %s", id, states[id], state)
		}
	}
}

func TestProjectCleanupBlockStorageReportsEveryOutcome(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveBlockStorage(t,
		cleanupBackupChain,
		`{"id":"sn-ok","volume_id":"v-ok"},{"id":"sn-poll","volume_id":"v-poll"},{"id":"sn-lost","volume_id":"v-lost"},{"id":"sn-gone","volume_id":"v-gone"}`,
		`{"id":"v-ok"},{"id":"v-poll"},{"id":"v-lost"},{"id":"v-gone"},{"id":"v-attached","attachments":[{"server_id":"s-locked"}]},{"id":"v-in-use"},{"id":"v-missing"}`)
	cloud.serveCompute(t, `{"id":"s-locked"}`, ``)
	cloud.blockStorageMutations(t,
		map[string]int{"b-i2": 400, "b-other": 404, "sn-lost": 0, "sn-gone": 404, "s-locked": 409, "v-in-use": 400, "v-missing": 404},
		map[string]int{"sn-poll": 500})
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithDeletionWait(30*time.Second))
	if !errors.Is(err, projectcleanup.ErrIncomplete) {
		t.Fatalf("result error %v", err)
	}
	outcomes := map[string]projectcleanup.Outcome{}
	for _, o := range result.Outcomes {
		outcomes[o.Candidate.ID] = o
	}
	want := map[string]projectcleanup.State{
		"s-locked": projectcleanup.Failed,
		"b-i2":     projectcleanup.Failed, "b-other": projectcleanup.AlreadyMissing, "b-i1": projectcleanup.NotAttempted, "b-full": projectcleanup.NotAttempted,
		"sn-ok": projectcleanup.Deleted, "sn-poll": projectcleanup.Accepted, "sn-lost": projectcleanup.Uncertain, "sn-gone": projectcleanup.AlreadyMissing,
		"v-ok": projectcleanup.Accepted, "v-poll": projectcleanup.NotAttempted, "v-lost": projectcleanup.NotAttempted, "v-gone": projectcleanup.Accepted,
		"v-attached": projectcleanup.NotAttempted, "v-in-use": projectcleanup.Failed, "v-missing": projectcleanup.AlreadyMissing,
	}
	for id, state := range want {
		if outcomes[id].State != state {
			t.Errorf("%s: %s, want %s (%s %v)", id, outcomes[id].State, state, outcomes[id].Reason, outcomes[id].Err)
		}
	}
	if b := outcomes["b-i2"]; b.StatusCode != 400 || !strings.Contains(string(b.Body), "refused b-i2") {
		t.Errorf("failed backup evidence %+v", b)
	}
	if outcomes["sn-poll"].WaitErr == nil || !strings.Contains(outcomes["v-poll"].Reason, "absence of dependency") ||
		!strings.Contains(outcomes["v-lost"].Reason, "uncertain") || !strings.Contains(outcomes["b-i1"].Reason, "failed") ||
		!strings.Contains(outcomes["v-attached"].Reason, "failed") {
		t.Errorf("dependent reasons %+v", outcomes)
	}
	var deletes []string
	for _, e := range cloud.events() {
		if strings.HasPrefix(e, "delete ") {
			deletes = append(deletes, strings.TrimPrefix(e, "delete "))
		}
	}
	wantDeletes := []string{"s-locked", "b-i2", "b-other", "sn-ok", "sn-poll", "sn-lost", "sn-gone", "v-ok", "v-gone", "v-in-use", "v-missing"}
	if !slices.Equal(deletes, wantDeletes) {
		t.Fatalf("deletes %v", deletes)
	}
}
