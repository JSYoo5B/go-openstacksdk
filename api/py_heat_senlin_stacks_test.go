package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func pyHeatSenlinAPI(cloud *testcloud.Cloud) *stacks.API {
	return stacks.New(cloud.Client("orchestration", "/heat"))
}

func pyHeatSenlinCanceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestPyHeatSenlinStackActionsSendFixedPathAndPythonBodies(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	api := pyHeatSenlinAPI(cloud)
	bodies := make(chan string, 8)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /heat/stacks/app/fixed/actions", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		raw, _ := io.ReadAll(r.Body)
		bodies <- strings.TrimSpace(string(raw))
		w.WriteHeader(http.StatusOK)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("actions must not look the stack up: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	scope := heatScope(t, api, "app", "fixed")
	for _, tc := range []struct {
		name string
		call func(context.Context) (*stacks.ActionAcknowledgement, error)
		want string
	}{
		{"check", scope.Check, `{"check":""}`},
		{"suspend", scope.Suspend, `{"suspend":null}`},
		{"resume", scope.Resume, `{"resume":null}`},
	} {
		ack, err := tc.call(context.Background())
		if err != nil || ack == nil || ack.Action != tc.name || ack.StatusCode != 200 {
			t.Fatalf("%s ack=%+v err=%v", tc.name, ack, err)
		}
		if got := <-bodies; got != tc.want {
			t.Fatalf("%s body=%s want %s", tc.name, got, tc.want)
		}
	}
	if calls.Load() != 3 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestPyHeatSenlinStackActionsStatusCodesNeverReplayAndCancelBeforeHTTP(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status int
		ok     bool
	}{{200, true}, {201, true}, {202, true}, {204, true}, {400, false}, {404, false}, {409, false}, {500, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /heat/stacks/app/fixed/actions", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(tc.status)
		})
		scope := heatScope(t, pyHeatSenlinAPI(cloud), "app", "fixed")
		ack, err := scope.Check(context.Background())
		if tc.ok != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d ack=%v err=%v calls=%d", tc.status, ack, err, calls.Load())
		}
		if !tc.ok && !gophercloud.ResponseCodeIs(err, tc.status) {
			t.Fatalf("native HTTP cause lost for %d: %v", tc.status, err)
		}
		if _, err := scope.Suspend(pyHeatSenlinCanceled()); !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled: %v", err)
		}
		if calls.Load() != 1 {
			t.Fatalf("canceled call reached HTTP")
		}
	}
}

func TestPyHeatSenlinStackDocumentsKeepRawBodyAndRejectBadShapes(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	api := pyHeatSenlinAPI(cloud)
	for _, route := range []string{"export", "environment", "files", "template"} {
		route := route
		cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/"+route, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Route", route)
			testcloud.JSON(w, 200, `{"big":9007199254740993,"nothing":null,"flag":false,"route":"`+route+`"}`)
		})
	}
	cloud.Mux.HandleFunc("GET /heat/stacks/app/array/files", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `["x"]`) })
	cloud.Mux.HandleFunc("GET /heat/stacks/app/null/files", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `null`) })
	cloud.Mux.HandleFunc("GET /heat/stacks/app/accepted/files", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 202, `{}`) })
	cloud.Mux.HandleFunc("GET /heat/stacks/app/gone/files", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{"error":{}}`) })
	scope := heatScope(t, api, "app", "fixed")
	for route, call := range map[string]func(context.Context) (*resource.RawResource, error){
		"export": scope.Export, "environment": scope.Environment, "files": scope.Files, "template": scope.Template,
	} {
		doc, err := call(context.Background())
		if err != nil || doc == nil || doc.StatusCode != 200 || doc.Header.Get("X-Route") != route {
			t.Fatalf("%s doc=%+v err=%v", route, doc, err)
		}
		if string(doc.Body["big"]) != "9007199254740993" || string(doc.Body["nothing"]) != "null" || string(doc.Body["flag"]) != "false" {
			t.Fatalf("%s raw body altered: %v", route, doc.Body)
		}
		if _, err := call(pyHeatSenlinCanceled()); !errors.Is(err, context.Canceled) {
			t.Fatalf("%s pre-canceled: %v", route, err)
		}
	}
	for _, name := range []string{"array", "null"} {
		if doc, err := heatScope(t, api, "app", name).Files(context.Background()); err == nil || doc != nil {
			t.Fatalf("%s body must be rejected: %v %v", name, doc, err)
		}
	}
	if _, err := heatScope(t, api, "app", "accepted").Files(context.Background()); !gophercloud.ResponseCodeIs(err, 202) {
		t.Fatalf("only 200 is a document: %v", err)
	}
	if _, err := heatScope(t, api, "app", "gone").Files(context.Background()); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatalf("404 cause lost: %v", err)
	}
}

func TestPyHeatSenlinFindNameOrIDUsesOneGETKeepsRawFieldsAndHidesDeleted(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	api := pyHeatSenlinAPI(cloud)
	var lists, gets, redirects atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/stacks", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		testcloud.JSON(w, 200, `{"stacks":[]}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app", func(w http.ResponseWriter, r *http.Request) {
		redirects.Add(1)
		if r.URL.RawQuery != "resolve_outputs=False" && r.URL.RawQuery != "" {
			t.Errorf("query=%q", r.URL.RawQuery)
		}
		target := "/heat/stacks/app/fixed"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusFound)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"app","id":"fixed","stack_status":"CREATE_COMPLETE","stack_owner":"root-owner","parent":"parent-id","deletion_time":null,"stack_user_project_id":"p1","files_container":"c1","environment_files":["e.yaml"],"files":{"a.yaml":"x"},"template":{"heat_template_version":"2018-08-31"},"environment":{"parameters":{}},"future":9007199254740993,"creation_time":"2026-10-01T01:02:03Z"}}`)
	})
	for _, tc := range []struct{ name, status string }{{"gone", "DELETE_COMPLETE"}, {"adopted", "ADOPT_COMPLETE"}, {"failed", "DELETE_FAILED"}} {
		tc := tc
		cloud.Mux.HandleFunc("GET /heat/stacks/"+tc.name, func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, 200, `{"stack":{"stack_name":"`+tc.name+`","id":"id-`+tc.name+`","stack_status":"`+tc.status+`"}}`)
		})
	}
	cloud.Mux.HandleFunc("GET /heat/stacks/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /heat/stacks/forbidden", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 403, `{}`) })
	cloud.Mux.HandleFunc("GET /heat/stacks/noenvelope", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"stack_name":"noenvelope","id":"x"}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/noidentity", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"noidentity"}}`)
	})
	ctx := context.Background()
	detail, err := api.FindNameOrID(ctx, "app", stacks.WithFindResolveOutputs(false))
	if err != nil || detail == nil || !detail.Detailed || detail.ID != "fixed" || detail.Name != "app" || detail.CreationTime.IsZero() {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
	for key, want := range map[string]string{"stack_owner": `"root-owner"`, "parent": `"parent-id"`, "deletion_time": "null", "stack_user_project_id": `"p1"`,
		"files_container": `"c1"`, "environment_files": `["e.yaml"]`, "files": `{"a.yaml":"x"}`, "future": "9007199254740993"} {
		if got := string(detail.Raw.Body[key]); got != want {
			t.Fatalf("raw %s=%s want %s", key, got, want)
		}
	}
	if _, ok := detail.Raw.Body["template"]; !ok {
		t.Fatal("raw template field lost")
	}
	if lists.Load() != 0 || redirects.Load() != 1 || gets.Load() != 1 {
		t.Fatalf("lists=%d redirects=%d gets=%d", lists.Load(), redirects.Load(), gets.Load())
	}
	if value, err := api.FindNameOrID(ctx, "app"); err != nil || value == nil {
		t.Fatalf("default outputs: %v %v", value, err)
	}
	for _, name := range []string{"gone", "adopted", "missing"} {
		if value, err := api.FindNameOrID(ctx, name); value != nil || !errors.Is(err, resource.ErrNotFound) {
			t.Fatalf("%s strict: %v %v", name, value, err)
		}
		if value, err := api.FindNameOrID(ctx, name, stacks.WithFindIgnoreMissing()); value != nil || err != nil {
			t.Fatalf("%s ignored: %v %v", name, value, err)
		}
	}
	if value, err := api.FindNameOrID(ctx, "failed"); err != nil || value == nil || value.Status != "DELETE_FAILED" {
		t.Fatalf("DELETE_FAILED stays visible: %v %v", value, err)
	}
	if _, err := api.FindNameOrID(ctx, "forbidden", stacks.WithFindIgnoreMissing()); !gophercloud.ResponseCodeIs(err, 403) {
		t.Fatalf("403 must not be ignored: %v", err)
	}
	for _, name := range []string{"noenvelope", "noidentity"} {
		if value, err := api.FindNameOrID(ctx, name); err == nil || value != nil {
			t.Fatalf("%s must be rejected: %v %v", name, value, err)
		}
	}
	before := lists.Load() + gets.Load() + redirects.Load()
	if _, err := api.FindNameOrID(ctx, ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty reference: %v", err)
	}
	if _, err := api.FindNameOrID(ctx, "app", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
	if _, err := api.FindNameOrID(pyHeatSenlinCanceled(), "app"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	if lists.Load()+gets.Load()+redirects.Load() != before {
		t.Fatal("invalid calls reached HTTP")
	}
}

func TestPyHeatSenlinScopedAbandonAndDeleteUseFixedIdentity(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	api := pyHeatSenlinAPI(cloud)
	var abandons, deletes atomic.Int32
	var abandonStatus, deleteStatus atomic.Int32
	abandonStatus.Store(200)
	deleteStatus.Store(204)
	cloud.Mux.HandleFunc("DELETE /heat/stacks/app/fixed/abandon", func(w http.ResponseWriter, r *http.Request) {
		abandons.Add(1)
		testcloud.JSON(w, int(abandonStatus.Load()), `{"name":"app","id":"fixed","status":"COMPLETE","resources":{"r":{"type":"OS::Heat::None"}},"unknown":9007199254740993}`)
	})
	cloud.Mux.HandleFunc("DELETE /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.WriteHeader(int(deleteStatus.Load()))
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("fixed identity must not trigger lookups: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	scope := heatScope(t, api, "app", "fixed")
	ctx := context.Background()
	value, err := scope.Abandon(ctx)
	if err != nil || value == nil || value.Name != "app" || value.ID != "fixed" || value.Status != "COMPLETE" || value.Resources["r"] == nil {
		t.Fatalf("abandon=%+v err=%v", value, err)
	}
	abandonStatus.Store(202)
	if value, err := scope.Abandon(ctx); err == nil || value != nil || !gophercloud.ResponseCodeIs(err, 202) {
		t.Fatalf("only HTTP 200 abandons: %v %v", value, err)
	}
	if _, err := scope.Abandon(pyHeatSenlinCanceled()); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled abandon: %v", err)
	}
	if abandons.Load() != 2 {
		t.Fatalf("abandon requests=%d", abandons.Load())
	}
	for _, code := range []int{202, 204} {
		deleteStatus.Store(int32(code))
		if err := scope.Delete(ctx); err != nil {
			t.Fatalf("delete %d: %v", code, err)
		}
	}
	deleteStatus.Store(403)
	if err := scope.Delete(ctx); !gophercloud.ResponseCodeIs(err, 403) {
		t.Fatalf("403 must not be ignored: %v", err)
	}
	deleteStatus.Store(500)
	if err := scope.Delete(ctx, resource.WithMissingError()); !gophercloud.ResponseCodeIs(err, 500) {
		t.Fatalf("server error must stay native: %v", err)
	}
	if deletes.Load() != 4 {
		t.Fatalf("delete requests=%d (no replay, no lookup)", deletes.Load())
	}
}

func TestPyHeatSenlinStackWaitTransientTimeoutCustomPolicyAndCancellation(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	api := pyHeatSenlinAPI(cloud)
	var phase atomic.Int32
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		n := gets.Add(1)
		status, reason := "CREATE_IN_PROGRESS", "working"
		switch phase.Load() {
		case 1:
			if n > 3 {
				status, reason = "CREATE_COMPLETE", "done"
			}
		case 2:
			status, reason = "CREATE_FAILED", "boom"
		case 3:
			status, reason = "UPDATE_IN_PROGRESS", "ready"
		}
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"app","id":"fixed","stack_status":"`+status+`","stack_status_reason":"`+reason+`"}}`)
	})
	scope := heatScope(t, api, "app", "fixed")
	ctx := context.Background()
	fast := resource.WithPollInterval(time.Millisecond)

	// Transient polling: nonterminal responses are polled until the target.
	phase.Store(1)
	var progress []int
	value, err := scope.Wait(ctx, "create_complete", fast, resource.WithTimeout(5*time.Second), resource.WithProgressCallback(func(p int) { progress = append(progress, p) }))
	if err != nil || value == nil || value.Status != "CREATE_COMPLETE" || gets.Load() != 4 {
		t.Fatalf("transient wait=%v err=%v gets=%d", value, err, gets.Load())
	}
	if len(progress) != 3 || progress[0] != 0 {
		t.Fatalf("Stack has no progress field so callbacks report zero for each nonterminal response: %v", progress)
	}

	// Timeout: a bounded wait on a never-complete stack ends with a deadline cause.
	phase.Store(0)
	if _, err := scope.Wait(ctx, "CREATE_COMPLETE", fast, resource.WithTimeout(30*time.Millisecond)); err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout cause: %v", err)
	}

	// Python's default failure list is only ERROR: custom failure states replace Go's *_FAILED default.
	phase.Store(2)
	before := gets.Load()
	value, err = scope.Wait(ctx, "CREATE_COMPLETE", fast, resource.WithTimeout(40*time.Millisecond), resource.WithFailureStates("ERROR"))
	if err == nil || !errors.Is(err, context.DeadlineExceeded) || value != nil || gets.Load()-before < 2 {
		t.Fatalf("custom ERROR-only policy must keep polling a *_FAILED stack: %v %v gets=%d", value, err, gets.Load()-before)
	}
	if _, err := scope.Wait(ctx, "CREATE_COMPLETE", fast, resource.WithTimeout(time.Second)); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("default failure state: %v", err)
	}

	// Attribute selection: waiting on stack_status_reason reaches a reason, not a status.
	phase.Store(3)
	value, err = scope.Wait(ctx, "ready", fast, resource.WithTimeout(time.Second), resource.WithStatusAttribute("stack_status_reason"))
	if err != nil || value == nil || value.StatusReason != "ready" {
		t.Fatalf("attribute wait=%v err=%v", value, err)
	}

	// In-flight cancellation by the parent context while a response is pending.
	phase.Store(0)
	cancelCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		_, err := scope.Wait(cancelCtx, "CREATE_COMPLETE", resource.WithPollInterval(time.Hour), resource.WithUnlimitedWait())
		done <- err
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel cause: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("wait ignored parent cancellation")
	}
}

func TestPyHeatSenlinStackWaitDeletedTransientAdoptAndCallback(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	api := pyHeatSenlinAPI(cloud)
	var gets atomic.Int32
	var final atomic.Value
	final.Store("DELETE_COMPLETE")
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		status := "DELETE_IN_PROGRESS"
		if gets.Add(1) >= 3 {
			status = final.Load().(string)
		}
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"app","id":"fixed","stack_status":"`+status+`"}}`)
	})
	scope := heatScope(t, api, "app", "fixed")
	var progress []int
	if err := scope.WaitDeleted(context.Background(), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5*time.Second), resource.WithProgressCallback(func(p int) { progress = append(progress, p) })); err != nil {
		t.Fatal(err)
	}
	if gets.Load() != 3 || len(progress) != 2 {
		t.Fatalf("gets=%d progress=%v: nonterminal responses report zero progress and terminal ones do not", gets.Load(), progress)
	}
	// Python's Stack.fetch hides ADOPT_COMPLETE as NotFound so its wait_for_delete returns;
	// the Go waiter only treats DELETE_COMPLETE or HTTP 404 as deletion.
	final.Store("ADOPT_COMPLETE")
	gets.Store(0)
	err := scope.WaitDeleted(context.Background(), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(40*time.Millisecond))
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ADOPT_COMPLETE is not inferred as deletion: %v", err)
	}
}

func TestPyHeatSenlinStackListForwardsPythonFilterNamesThroughExplicitQueries(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	api := pyHeatSenlinAPI(cloud)
	queries := make(chan map[string][]string, 4)
	cloud.Mux.HandleFunc("GET /heat/stacks", func(w http.ResponseWriter, r *http.Request) {
		queries <- r.URL.Query()
		testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"app","id":"fixed","stack_status":"CREATE_COMPLETE"}]}`)
	})
	values, err := api.Resources().All(context.Background(),
		resource.WithQuery("action", "CREATE"), resource.WithQuery("owner_id", "o1"), resource.WithQuery("username", "u1"),
		resource.WithQuery("tenant_id", "p1"), resource.WithQuery("tags_any", "a,b"), resource.WithQuery("not_tags_any", "c"))
	if err != nil || len(values) != 1 || values[0].Detailed {
		t.Fatalf("values=%v err=%v", values, err)
	}
	got := <-queries
	for key, want := range map[string]string{"action": "CREATE", "owner_id": "o1", "username": "u1", "tenant_id": "p1", "tags_any": "a,b", "not_tags_any": "c"} {
		if len(got[key]) != 1 || got[key][0] != want {
			t.Fatalf("query %s=%v want %s (all %v)", key, got[key], want, got)
		}
	}
	// Python spells these project_id/any_tags/not_any_tags; Go sends exactly the wire names it is given.
	if _, err := api.Resources().All(context.Background(), resource.WithQuery("project_id", "p1")); err != nil {
		t.Fatal(err)
	}
	if got := <-queries; len(got["project_id"]) != 1 || len(got["tenant_id"]) != 0 {
		t.Fatalf("Python aliases must not be translated automatically: %v", got)
	}
}
