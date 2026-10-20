package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/workflow/v2/crontriggers"
	"github.com/JSYoo5B/go-openstacksdk/workflow/v2/executions"
	"github.com/JSYoo5B/go-openstacksdk/workflow/v2/workflows"
	"github.com/gophercloud/gophercloud/v2"
)

const mistralWorkflowBody = `{"id":"wf-1","name":"deploy","namespace":"ns","project_id":"p1","scope":"private","tags":["blue"],"definition":"---\nversion: '2.0'\n","input":"a, b=false","created_at":"2026-01-02 03:04:05","updated_at":null}`

func mistralClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	return cloud.Client("workflowv2", "/mistral/v2")
}

func mistralUnexpected(t *testing.T, err error, status int) {
	t.Helper()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &unexpected) || unexpected.Actual != status {
		t.Fatalf("want unexpected response code %d, got %v", status, err)
	}
}

func mistralNoRequests(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
}

func mistralSlow(cloud *testcloud.Cloud, calls *atomic.Int32) {
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
}

func TestMistralWorkflowCreateSendsTextDefinitionAndScopeQuery(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /mistral/v2/workflows", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if got := r.Header.Get("Content-Type"); got != "text/plain" {
				t.Errorf("content type=%q", got)
			}
			if got := r.Header.Get("Accept"); got != "" {
				t.Errorf("accept=%q", got)
			}
			data, _ := io.ReadAll(r.Body)
			if string(data) != "version: '2.0'\nflag: false\n" {
				t.Errorf("body=%q", data)
			}
			if got := r.URL.Query(); got.Get("scope") != "public" || got.Get("namespace") != "ns" || got.Get("vendor") != "x" {
				t.Errorf("query=%v", got)
			}
			testcloud.JSON(w, tc.status, `{"workflows":[`+mistralWorkflowBody+`]}`)
		})
		got, err := workflows.New(cloud.Client("workflowv2", "/mistral/v2")).Create(context.Background(),
			workflows.CreateOpts{Scope: "public", Namespace: "ns", Definition: strings.NewReader("version: '2.0'\nflag: false\n")},
			workflows.WithCreateQuery("vendor", "x"))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d error=%v calls=%d", tc.status, err, calls.Load())
		}
		if tc.success && (len(got) != 1 || got[0].ID != "wf-1" || got[0].UpdatedAt != nil || got[0].CreatedAt.IsZero() || !reflect.DeepEqual(got[0].Tags, []string{"blue"})) {
			t.Fatalf("workflows=%+v", got)
		}
		if !tc.success {
			mistralUnexpected(t, err, tc.status)
		}
	}
}

func TestMistralWorkflowRejectsNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	mistralNoRequests(t, cloud)
	api := workflows.New(mistralClient(cloud))
	ctx := context.Background()
	if _, err := api.Create(ctx, workflows.CreateOpts{Definition: strings.NewReader("x")}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
}

func TestMistralWorkflowGetDeleteStatusCodesAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /mistral/v2/workflows/wf-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, mistralWorkflowBody)
	})
	cloud.Mux.HandleFunc("GET /mistral/v2/workflows/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("DELETE /mistral/v2/workflows/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "async":
			w.WriteHeader(202)
		case "odd":
			w.WriteHeader(200)
		default:
			w.WriteHeader(204)
		}
	})
	api := workflows.New(mistralClient(cloud))
	ctx := context.Background()
	value, err := api.Get(ctx, "wf-1")
	if err != nil || value.Name != "deploy" || value.Namespace != "ns" || value.Input != "a, b=false" || value.UpdatedAt != nil || value.CreatedAt.Year() != 2026 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Get(ctx, "missing"); err == nil {
		t.Fatal("native get of a missing workflow must fail")
	} else {
		mistralUnexpected(t, err, 404)
	}
	for _, id := range []string{"wf-1", "async"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	mistralUnexpected(t, api.Delete(ctx, "missing"), 404)
	mistralUnexpected(t, api.Delete(ctx, "odd"), 200)
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict Remove must report ErrNotFound: %v", err)
	}
}

func TestMistralWorkflowListFiltersFollowNextLinkAndPreserveFilters(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /mistral/v2/workflows", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "wf-1" {
			testcloud.JSON(w, 200, `{"workflows":[{"id":"wf-2","name":"two"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"workflows":[{"id":"wf-1","name":"one"}],"next":"`+cloud.Server.URL+`/mistral/v2/workflows?marker=wf-1&limit=1"}`)
	})
	api := workflows.New(mistralClient(cloud))
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	var ids []string
	for value, err := range api.List(context.Background(),
		workflows.WithListOptions(workflows.ListOpts{
			Scope: "private", Tags: []string{"a", "b"}, Limit: 1, AllProjects: 1, SortKeys: "name", SortDirs: "desc",
			Name:      &workflows.ListFilter{Filter: workflows.FilterNEQ, Value: "skip"},
			CreatedAt: &workflows.ListDateFilter{Filter: workflows.FilterGTE, Value: when},
			Namespace: &workflows.ListFilter{Value: ""},
		}),
		workflows.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"wf-1", "wf-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	q := requests[0]
	if q.Get("scope") != "private" || q.Get("tags") != "a,b" || q.Get("limit") != "1" || q.Get("all_projects") != "1" ||
		q.Get("sort_keys") != "name" || q.Get("sort_dirs") != "desc" || q.Get("name") != "neq:skip" ||
		q.Get("created_at") != "gte:2026-01-02 03:04:05" || !q.Has("namespace") || q.Get("namespace") != "" || q.Get("vendor") != "x" {
		t.Fatalf("first query=%v", q)
	}
	if requests[1].Get("marker") != "wf-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestMistralWorkflowCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	mistralSlow(cloud, &calls)
	api := workflows.New(mistralClient(cloud))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	def := func() workflows.CreateOpts { return workflows.CreateOpts{Definition: strings.NewReader("x")} }
	if _, err := api.Create(canceled, def()); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Get(canceled, "wf-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if err := api.Delete(canceled, "wf-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, def()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

const mistralExecutionBody = `{"id":"ex-1","workflow_id":"wf-1","workflow_name":"deploy","workflow_namespace":"ns","description":"","state":"RUNNING","state_info":null,"root_execution_id":null,"task_execution_id":null,"project_id":"p1","input":"{\"a\": 1, \"flag\": false}","output":"{}","params":"{\"env\": {}}","created_at":"2026-01-02 03:04:05","updated_at":"2026-01-02 03:04:06"}`

func TestMistralExecutionCreateSendsFlatBodyAndPreservesNestedValues(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /mistral/v2/executions", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("body: %v", err)
			}
			got := map[string]string{}
			for key, value := range body {
				got[key] = string(value)
			}
			want := map[string]string{
				"workflow_name": `"deploy"`, "workflow_namespace": `"ns"`, "input": `{"flag":false,"n":0}`, "params": `{"env":{}}`, "description": `"run"`, "vendor": `null`,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			testcloud.JSON(w, tc.status, mistralExecutionBody)
		})
		got, err := executions.New(mistralClient(cloud)).Create(context.Background(), executions.CreateOpts{
			WorkflowName: "deploy", WorkflowNamespace: "ns", Description: "run",
			Input: map[string]any{"flag": false, "n": 0}, Params: map[string]any{"env": map[string]any{}},
		}, executions.WithCreateField("vendor", nil))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d error=%v calls=%d", tc.status, err, calls.Load())
		}
		if tc.success && (got.ID != "ex-1" || got.State != "RUNNING" || got.StateInfo != nil || got.RootExecutionID != nil || got.Input["flag"] != false || got.Output == nil || len(got.Output) != 0 || got.UpdatedAt.IsZero()) {
			t.Fatalf("execution=%+v", got)
		}
		if !tc.success {
			mistralUnexpected(t, err, tc.status)
		}
	}
}

func TestMistralExecutionRejectsInvalidInputBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	mistralNoRequests(t, cloud)
	api := executions.New(mistralClient(cloud))
	ctx := context.Background()
	opts := executions.CreateOpts{WorkflowID: "wf-1"}
	if _, err := api.Create(ctx, opts, executions.WithCreateField("workflow_id", "other")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
	if _, err := api.Create(ctx, opts, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
	if _, err := api.Create(ctx, executions.CreateOpts{}); err == nil {
		t.Fatal("a workflow ID or name is required before HTTP")
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
}

func TestMistralExecutionGetDeleteStatusCodesAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /mistral/v2/executions/ex-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, mistralExecutionBody)
	})
	cloud.Mux.HandleFunc("GET /mistral/v2/executions/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("DELETE /mistral/v2/executions/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "async":
			w.WriteHeader(202)
		case "odd":
			w.WriteHeader(200)
		default:
			w.WriteHeader(204)
		}
	})
	api := executions.New(mistralClient(cloud))
	ctx := context.Background()
	value, err := api.Get(ctx, "ex-1")
	if err != nil || value.State != "RUNNING" || value.WorkflowName != "deploy" || value.Params == nil {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Get(ctx, "missing"); err == nil {
		t.Fatal("native get of a missing execution must fail")
	} else {
		mistralUnexpected(t, err, 404)
	}
	for _, id := range []string{"ex-1", "async"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	mistralUnexpected(t, api.Delete(ctx, "missing"), 404)
	mistralUnexpected(t, api.Delete(ctx, "odd"), 200)
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
}

func TestMistralExecutionListFiltersAndJSONQueriesFollowNextLink(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /mistral/v2/executions", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "ex-1" {
			testcloud.JSON(w, 200, `{"executions":[{"id":"ex-2","state":"SUCCESS"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"executions":[`+mistralExecutionBody+`],"next":"`+cloud.Server.URL+`/mistral/v2/executions?marker=ex-1&limit=1"}`)
	})
	api := executions.New(mistralClient(cloud))
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	var ids []string
	for value, err := range api.List(context.Background(),
		executions.WithListOptions(executions.ListOpts{
			WorkflowID: "wf-1", Limit: 1, AllProjects: 1, IncludeOutput: true,
			State: &executions.ListFilter{Filter: executions.FilterIN, Value: "RUNNING,ERROR"},
			Input: map[string]any{"flag": false}, Output: map[string]any{}, Params: map[string]any{"env": nil},
			UpdatedAt: &executions.ListDateFilter{Filter: executions.FilterLT, Value: when},
		}),
		executions.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"ex-1", "ex-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	q := requests[0]
	if q.Get("workflow_id") != "wf-1" || q.Get("limit") != "1" || q.Get("all_projects") != "1" || q.Get("include_output") != "1" ||
		q.Get("state") != "in:RUNNING,ERROR" || q.Get("input") != `{"flag":false}` || q.Get("output") != `{}` || q.Get("params") != `{"env":null}` ||
		q.Get("updated_at") != "lt:2026-01-02 03:04:05" || q.Get("vendor") != "x" {
		t.Fatalf("first query=%v", q)
	}
	// Pinned Gophercloud also emits a stray "-=true" parameter for IncludeOutput.
	if q.Get("-") != "true" {
		t.Fatalf("stray IncludeOutput parameter changed: %v", q)
	}
	if requests[1].Get("marker") != "ex-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestMistralExecutionCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	mistralSlow(cloud, &calls)
	api := executions.New(mistralClient(cloud))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	opts := executions.CreateOpts{WorkflowID: "wf-1"}
	if _, err := api.Create(canceled, opts); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Get(canceled, "ex-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if err := api.Delete(canceled, "ex-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, opts); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

const mistralCronBody = `{"id":"ct-1","name":"nightly","pattern":"0 0 * * *","remaining_executions":0,"workflow_id":"wf-1","workflow_name":"deploy","workflow_input":"{\"flag\": false}","workflow_params":"{}","project_id":"p1","scope":"private","created_at":"2026-01-02 03:04:05","first_execution_time":null,"next_execution_time":"2026-01-03 00:00:00"}`

func TestMistralCronTriggerCreateSendsFormattedFirstExecutionAndRequires201Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /mistral/v2/cron_triggers", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("body: %v", err)
			}
			got := map[string]string{}
			for key, value := range body {
				got[key] = string(value)
			}
			want := map[string]string{
				"name": `"nightly"`, "pattern": `"0 0 * * *"`, "workflow_name": `"deploy"`, "workflow_input": `{"flag":false}`,
				"workflow_params": `{"env":{}}`, "first_execution_time": `"2026-01-02 03:04"`, "vendor": `false`,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			testcloud.JSON(w, tc.status, mistralCronBody)
		})
		first := time.Date(2026, 1, 2, 3, 4, 59, 0, time.UTC)
		got, err := crontriggers.New(mistralClient(cloud)).Create(context.Background(), crontriggers.CreateOpts{
			Name: "nightly", Pattern: "0 0 * * *", WorkflowName: "deploy", FirstExecutionTime: &first,
			WorkflowInput: map[string]any{"flag": false}, WorkflowParams: map[string]any{"env": map[string]any{}},
		}, crontriggers.WithCreateField("vendor", false))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d error=%v calls=%d", tc.status, err, calls.Load())
		}
		if tc.success && (got.ID != "ct-1" || got.RemainingExecutions != 0 || got.FirstExecutionTime != nil || got.NextExecutionTime == nil || got.WorkflowInput["flag"] != false) {
			t.Fatalf("trigger=%+v", got)
		}
		if !tc.success {
			mistralUnexpected(t, err, tc.status)
		}
	}
}

func TestMistralCronTriggerRejectsInvalidInputBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	mistralNoRequests(t, cloud)
	api := crontriggers.New(mistralClient(cloud))
	ctx := context.Background()
	opts := crontriggers.CreateOpts{Name: "n", Pattern: "* * * * *", WorkflowID: "wf-1"}
	if _, err := api.Create(ctx, opts, crontriggers.WithCreateField("pattern", "x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
	if _, err := api.Create(ctx, opts, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
	if _, err := api.Create(ctx, crontriggers.CreateOpts{Name: "n", Pattern: "* * * * *"}); err == nil {
		t.Fatal("a workflow ID or name is required before HTTP")
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
}

func TestMistralCronTriggerGetDeleteStatusCodesAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /mistral/v2/cron_triggers/ct-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, mistralCronBody)
	})
	cloud.Mux.HandleFunc("GET /mistral/v2/cron_triggers/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("DELETE /mistral/v2/cron_triggers/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "async":
			w.WriteHeader(202)
		case "odd":
			w.WriteHeader(200)
		default:
			w.WriteHeader(204)
		}
	})
	api := crontriggers.New(mistralClient(cloud))
	ctx := context.Background()
	value, err := api.Get(ctx, "ct-1")
	if err != nil || value.Name != "nightly" || value.Pattern != "0 0 * * *" || value.RemainingExecutions != 0 || value.FirstExecutionTime != nil || value.NextExecutionTime == nil {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Get(ctx, "missing"); err == nil {
		t.Fatal("native get of a missing trigger must fail")
	} else {
		mistralUnexpected(t, err, 404)
	}
	for _, id := range []string{"ct-1", "async"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	mistralUnexpected(t, api.Delete(ctx, "missing"), 404)
	mistralUnexpected(t, api.Delete(ctx, "odd"), 200)
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
}

func TestMistralCronTriggerListFiltersAndTypedFiltersFollowNextLink(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /mistral/v2/cron_triggers", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "ct-1" {
			testcloud.JSON(w, 200, `{"cron_triggers":[{"id":"ct-2","name":"two"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"cron_triggers":[`+mistralCronBody+`],"next":"`+cloud.Server.URL+`/mistral/v2/cron_triggers?marker=ct-1&limit=1"}`)
	})
	api := crontriggers.New(mistralClient(cloud))
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	var ids []string
	for value, err := range api.List(context.Background(),
		crontriggers.WithListOptions(crontriggers.ListOpts{
			WorkflowID: "wf-1", Scope: "private", Limit: 1, AllProjects: 1,
			RemainingExecutions: &crontriggers.ListIntFilter{Filter: crontriggers.FilterEQ, Value: 0},
			Name:                &crontriggers.ListFilter{Filter: crontriggers.FilterHas, Value: "night"},
			NextExecutionTime:   &crontriggers.ListDateFilter{Filter: crontriggers.FilterGT, Value: when},
			WorkflowInput:       map[string]any{"flag": false},
		}),
		crontriggers.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"ct-1", "ct-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	q := requests[0]
	if q.Get("workflow_id") != "wf-1" || q.Get("scope") != "private" || q.Get("limit") != "1" || q.Get("all_projects") != "1" ||
		q.Get("remaining_executions") != "eq:0" || q.Get("name") != "has:night" || q.Get("next_execution_time") != "gt:2026-01-02 03:04:05" ||
		q.Get("workflow_input") != `{"flag":false}` || q.Get("vendor") != "x" {
		t.Fatalf("first query=%v", q)
	}
	if requests[1].Get("marker") != "ct-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestMistralCronTriggerCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	mistralSlow(cloud, &calls)
	api := crontriggers.New(mistralClient(cloud))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	opts := crontriggers.CreateOpts{Name: "n", Pattern: "* * * * *", WorkflowID: "wf-1"}
	if _, err := api.Create(canceled, opts); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Get(canceled, "ct-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if err := api.Delete(canceled, "ct-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, opts); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
