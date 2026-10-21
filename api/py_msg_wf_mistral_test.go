package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/workflow/v2/crontriggers"
	"github.com/JSYoo5B/go-openstacksdk/workflow/v2/executions"
	"github.com/JSYoo5B/go-openstacksdk/workflow/v2/workflows"
)

const pyMsgWfDefinition = "---\nversion: '2.0'\nwf:\n  tasks:\n    t:\n      action: std.noop\n"

// Python Workflow.commit sends the definition as text/plain with ?scope=...;
// Go Update does the same through PUT /workflows and decodes the workflow list.
func TestPyMsgWfWorkflowUpdateSendsTextPlainPutAndRequires200(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opts    workflows.UpdateOpts
		options []workflows.UpdateOption
		status  int
		success bool
		query   url.Values
	}{
		{"scope", workflows.UpdateOpts{Scope: "private"}, nil, 200, true, url.Values{"scope": {"private"}}},
		{"scope namespace and extension", workflows.UpdateOpts{Scope: "public", Namespace: "ns"}, []workflows.UpdateOption{workflows.WithUpdateQuery("vendor", "x")}, 200, true, url.Values{"scope": {"public"}, "namespace": {"ns"}, "vendor": {"x"}}},
		{"omitted scope stays omitted", workflows.UpdateOpts{}, nil, 200, true, url.Values{}},
		{"created is not accepted", workflows.UpdateOpts{Scope: "private"}, nil, 201, false, url.Values{"scope": {"private"}}},
		{"no content is not accepted", workflows.UpdateOpts{Scope: "private"}, nil, 204, false, url.Values{"scope": {"private"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("PUT /mistral/v2/workflows", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				data, _ := io.ReadAll(r.Body)
				if string(data) != pyMsgWfDefinition || r.Header.Get("Content-Type") != "text/plain" {
					t.Errorf("body=%q content-type=%q", data, r.Header.Get("Content-Type"))
				}
				if got := r.URL.Query(); len(got) != len(tc.query) {
					t.Errorf("query=%v", got)
				} else {
					for key, want := range tc.query {
						if got.Get(key) != want[0] {
							t.Errorf("query=%v", got)
						}
					}
				}
				if accept := r.Header.Get("Accept"); strings.Contains(accept, "json") {
					t.Errorf("Accept=%q must not request JSON for a text body", accept)
				}
				if tc.status == 204 {
					w.WriteHeader(204)
					return
				}
				testcloud.JSON(w, tc.status, `{"workflows":[`+mistralWorkflowBody+`]}`)
			})
			opts := tc.opts
			opts.Definition = strings.NewReader(pyMsgWfDefinition)
			got, err := workflows.New(mistralClient(cloud)).Update(context.Background(), opts, tc.options...)
			if tc.success != (err == nil) || calls.Load() != 1 {
				t.Fatalf("error=%v calls=%d", err, calls.Load())
			}
			if tc.success && (len(got) != 1 || got[0].ID != "wf-1" || got[0].Name != "deploy") {
				t.Fatalf("workflows=%+v", got)
			}
			if !tc.success {
				mistralUnexpected(t, err, tc.status)
			}
		})
	}
}

func TestPyMsgWfWorkflowUpdateRejectsInvalidInputBeforeHTTPAndNeverReplays(t *testing.T) {
	cloud := testcloud.New(t)
	mistralNoRequests(t, cloud)
	api := workflows.New(mistralClient(cloud))
	ctx := context.Background()
	def := func() io.Reader { return strings.NewReader(pyMsgWfDefinition) }
	if _, err := api.Update(ctx, workflows.UpdateOpts{Scope: "private"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("missing definition: %v", err)
	}
	if _, err := api.Update(ctx, workflows.UpdateOpts{Definition: def()}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := api.Update(canceled, workflows.UpdateOpts{Definition: def()}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled update: %v", err)
	}

	failing := testcloud.New(t)
	var calls atomic.Int32
	failing.Mux.HandleFunc("PUT /mistral/v2/workflows", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 503, `{}`)
	})
	if _, err := workflows.New(mistralClient(failing)).Update(ctx, workflows.UpdateOpts{Definition: def()}); err == nil || calls.Load() != 1 {
		t.Fatalf("server error must surface once: %v calls=%d", err, calls.Load())
	}
}

func pyMsgWfListRows(t *testing.T, w http.ResponseWriter, key string, rows ...string) {
	t.Helper()
	testcloud.JSON(w, 200, `{"`+key+`":[`+strings.Join(rows, ",")+`]}`)
}

// find_workflow: GET with the literal identity first (Mistral accepts a name
// or ID there); only 400/403/404 fall back to a complete list that is matched
// locally because Workflow declares no name query.
func TestPyMsgWfWorkflowFindIdentityGetsFirstThenListsWithoutNameFilter(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists atomic.Int32
	var listQueries []url.Values
	cloud.Mux.HandleFunc("GET /mistral/v2/workflows/{id}", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		switch r.PathValue("id") {
		case "deploy", "wf-1":
			testcloud.JSON(w, 200, mistralWorkflowBody)
		case "denied":
			testcloud.JSON(w, 403, `{}`)
		case "broken":
			testcloud.JSON(w, 500, `{}`)
		default:
			testcloud.JSON(w, 404, `{}`)
		}
	})
	cloud.Mux.HandleFunc("GET /mistral/v2/workflows", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		listQueries = append(listQueries, r.URL.Query())
		pyMsgWfListRows(t, w, "workflows",
			`{"id":"wf-1","name":"deploy"}`, `{"id":"wf-2","name":"denied"}`, `{"id":"wf-3","name":"dup"}`, `{"id":"wf-4","name":"dup"}`, `{"id":"dup2","name":"other"}`)
	})
	api := workflows.New(mistralClient(cloud))
	ctx := context.Background()

	got, err := api.FindIdentity(ctx, "deploy")
	if err != nil || got == nil || got.ID != "wf-1" || gets.Load() != 1 || lists.Load() != 0 {
		t.Fatalf("direct: %+v err=%v gets=%d lists=%d", got, err, gets.Load(), lists.Load())
	}
	got, err = api.FindIdentity(ctx, "denied")
	if err != nil || got == nil || got.ID != "wf-2" || lists.Load() != 1 {
		t.Fatalf("403 falls back: %+v err=%v lists=%d", got, err, lists.Load())
	}
	if listQueries[0].Has("name") || len(listQueries[0]) != 0 {
		t.Fatalf("workflow fallback must list without a name filter: %v", listQueries[0])
	}
	got, err = api.FindIdentity(ctx, "gone")
	if err != nil || got != nil {
		t.Fatalf("missing is nil,nil by default: %+v err=%v", got, err)
	}
	if _, err = api.FindIdentity(ctx, "gone", resource.WithIdentityFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict missing: %v", err)
	}
	if _, err = api.FindIdentity(ctx, "dup"); !errors.As(err, new(*resource.AmbiguousError)) {
		t.Fatalf("two rows named dup must be ambiguous: %v", err)
	}
	listsBefore := lists.Load()
	if got, err = api.FindIdentity(ctx, "broken"); err == nil || got != nil || lists.Load() != listsBefore {
		t.Fatalf("a 500 must not fall back to list: %+v err=%v", got, err)
	}
	// An ID that only matches the row ID after a failed direct GET is found too.
	if got, err = api.FindIdentity(ctx, "dup2"); err != nil || got == nil || got.Name != "other" {
		t.Fatalf("ID match in list fallback: %+v err=%v", got, err)
	}
}

func TestPyMsgWfWorkflowFindIdentityListFailuresAndCancellationAreNotAbsence(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /mistral/v2/workflows/{id}", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /mistral/v2/workflows", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 500, `{}`) })
	api := workflows.New(mistralClient(cloud))
	if got, err := api.FindIdentity(context.Background(), "x"); err == nil || got != nil || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("list failure must stay an error: %+v err=%v", got, err)
	}
	quiet := testcloud.New(t)
	mistralNoRequests(t, quiet)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := workflows.New(mistralClient(quiet)).FindIdentity(canceled, "x"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	if _, err := workflows.New(mistralClient(quiet)).FindIdentity(context.Background(), " "); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("blank identity: %v", err)
	}
}

// find_execution: Execution has no name attribute, so only IDs can match.
func TestPyMsgWfExecutionFindIdentityMatchesOnlyIDsInListFallback(t *testing.T) {
	cloud := testcloud.New(t)
	var lists atomic.Int32
	cloud.Mux.HandleFunc("GET /mistral/v2/executions/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "ex-1" {
			testcloud.JSON(w, 200, mistralExecutionBody)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /mistral/v2/executions", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if len(r.URL.Query()) != 0 {
			t.Errorf("fallback list query=%v", r.URL.Query())
		}
		pyMsgWfListRows(t, w, "executions", `{"id":"ex-2","workflow_name":"deploy","state":"RUNNING"}`)
	})
	api := executions.New(mistralClient(cloud))
	ctx := context.Background()
	if got, err := api.FindIdentity(ctx, "ex-1"); err != nil || got == nil || got.State != "RUNNING" || lists.Load() != 0 {
		t.Fatalf("direct: %+v err=%v", got, err)
	}
	if got, err := api.FindIdentity(ctx, "ex-2"); err != nil || got == nil || got.ID != "ex-2" || lists.Load() != 1 {
		t.Fatalf("ID in list: %+v err=%v", got, err)
	}
	if got, err := api.FindIdentity(ctx, "deploy"); err != nil || got != nil {
		t.Fatalf("workflow_name is not an execution name: %+v err=%v", got, err)
	}
	if _, err := api.FindIdentity(ctx, "deploy", resource.WithIdentityFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict missing: %v", err)
	}
}

// find_cron_trigger: the list fallback carries name=<identity>, and the
// cross-project flag is only sent for list fallback.
func TestPyMsgWfCronTriggerFindIdentitySendsNameQueryAndAllProjects(t *testing.T) {
	cloud := testcloud.New(t)
	var listQueries []url.Values
	cloud.Mux.HandleFunc("GET /mistral/v2/cron_triggers/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("direct GET query=%q", r.URL.RawQuery)
		}
		if r.PathValue("id") == "ct-1" {
			testcloud.JSON(w, 200, mistralCronBody)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /mistral/v2/cron_triggers", func(w http.ResponseWriter, r *http.Request) {
		listQueries = append(listQueries, r.URL.Query())
		pyMsgWfListRows(t, w, "cron_triggers", `{"id":"ct-2","name":"nightly","pattern":"0 0 * * *"}`, `{"id":"ct-3","name":"other"}`)
	})
	api := crontriggers.New(mistralClient(cloud))
	ctx := context.Background()
	if got, err := api.FindIdentity(ctx, "ct-1"); err != nil || got == nil || got.Name != "nightly" || len(listQueries) != 0 {
		t.Fatalf("direct: %+v err=%v", got, err)
	}
	if _, err := api.FindIdentity(ctx, "nightly", resource.WithIdentityFindQuery("vendor", "x")); !errors.Is(err, resource.ErrUnsupported) || len(listQueries) != 0 {
		t.Fatalf("a query needs a direct-GET query hook, which Mistral GET does not have: %v", err)
	}
	got, err := api.FindIdentityAllProjects(ctx, "nightly")
	if err != nil || got == nil || got.ID != "ct-2" || len(listQueries) != 1 {
		t.Fatalf("list fallback: %+v err=%v", got, err)
	}
	if q := listQueries[0]; q.Get("name") != "nightly" || q.Get("all_projects") != "1" {
		t.Fatalf("list query=%v", q)
	}
	if _, err := api.FindIdentity(ctx, "nightly"); err != nil {
		t.Fatal(err)
	}
	if q := listQueries[1]; q.Get("name") != "nightly" || q.Has("all_projects") {
		t.Fatalf("default list query=%v", q)
	}
	if _, err := api.FindIdentity(ctx, "nightly", resource.WithIdentityFindAllProjects(true)); !errors.Is(err, resource.ErrUnsupported) || len(listQueries) != 2 {
		t.Fatalf("typed all-projects is not wired for this binding: %v", err)
	}
	if _, err := api.FindIdentity(ctx, "nightly", resource.WithIdentityFindQuery("all_projects", "1")); !errors.Is(err, resource.ErrInvalidOption) || len(listQueries) != 2 {
		t.Fatalf("raw all_projects is an SDK control: %v", err)
	}
	if got, err := api.FindIdentity(ctx, "nobody"); err != nil || got != nil {
		t.Fatalf("missing: %+v err=%v", got, err)
	}
}

func pyMsgWfExecutionState(state string) string {
	return strings.Replace(mistralExecutionBody, `"state":"RUNNING"`, `"state":"`+state+`"`, 1)
}

// wait_for_status on an Execution reads its status attribute (the body field
// "state"), compares case-insensitively and fails on the ERROR default. Go
// selects that field with WithStatusAttribute and the failure with
// WithFailureStates.
func TestPyMsgWfExecutionWaitForStatePollsAndFailsOnError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		states []string
		target string
		failed string
	}{
		{"reaches target case-insensitively", []string{"RUNNING", "PAUSED", "SUCCESS"}, "success", ""},
		{"failure state", []string{"RUNNING", "ERROR"}, "SUCCESS", "ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /mistral/v2/executions/ex-1", func(w http.ResponseWriter, r *http.Request) {
				n := int(calls.Add(1))
				if n > len(tc.states) {
					n = len(tc.states)
				}
				testcloud.JSON(w, 200, pyMsgWfExecutionState(tc.states[n-1]))
			})
			var progress []int
			got, err := executions.New(mistralClient(cloud)).WaitFor(context.Background(), resource.ID("ex-1"), tc.target,
				resource.WithStatusAttribute("state"), resource.WithFailureStates("ERROR"),
				resource.WithPollInterval(time.Millisecond), resource.WithUnlimitedWait(),
				resource.WithProgressCallback(func(p int) { progress = append(progress, p) }))
			if tc.failed == "" {
				if err != nil || got == nil || !strings.EqualFold(got.State, tc.target) || calls.Load() != 3 {
					t.Fatalf("got=%+v err=%v calls=%d", got, err, calls.Load())
				}
				// Python reports progress or 0 after every nonterminal fetch.
				if len(progress) != 2 || progress[0] != 0 || progress[1] != 0 {
					t.Fatalf("progress=%v", progress)
				}
				return
			}
			var failure *resource.FailedStateError
			if !errors.As(err, &failure) || failure.Status != tc.failed || got != nil {
				t.Fatalf("got=%+v err=%v", got, err)
			}
		})
	}
}

// Without a selected attribute the Go waiter has no status binding for these
// Mistral resources, so it refuses before HTTP rather than guessing a field.
func TestPyMsgWfWaitForRequiresExplicitStatusAttributeAndIsUnsupportedForWorkflowAndTrigger(t *testing.T) {
	cloud := testcloud.New(t)
	mistralNoRequests(t, cloud)
	ctx := context.Background()
	client := mistralClient(cloud)
	if _, err := executions.New(client).WaitFor(ctx, resource.ID("ex-1"), "SUCCESS"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("execution without attribute: %v", err)
	}
	if _, err := workflows.New(client).WaitFor(ctx, resource.ID("wf-1"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("workflow has no status: %v", err)
	}
	if _, err := crontriggers.New(client).WaitFor(ctx, resource.ID("ct-1"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("cron trigger has no status: %v", err)
	}
	if _, err := workflows.New(client).WaitFor(ctx, resource.ID("wf-1"), "ACTIVE", resource.WithStatusAttribute("does_not_exist")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("unknown attribute: %v", err)
	}
}

func TestPyMsgWfMistralWaitForDeletionPollsUntilNotFound(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		path string
		body string
		wait func(ctx context.Context, cloud *testcloud.Cloud, options ...resource.WaitOption) error
	}{
		{"workflow", "/mistral/v2/workflows/wf-1", mistralWorkflowBody, func(ctx context.Context, c *testcloud.Cloud, o ...resource.WaitOption) error {
			return workflows.New(mistralClient(c)).WaitForDeletion(ctx, resource.ID("wf-1"), o...)
		}},
		{"execution", "/mistral/v2/executions/ex-1", mistralExecutionBody, func(ctx context.Context, c *testcloud.Cloud, o ...resource.WaitOption) error {
			return executions.New(mistralClient(c)).WaitForDeletion(ctx, resource.ID("ex-1"), o...)
		}},
		{"cron trigger", "/mistral/v2/cron_triggers/ct-1", mistralCronBody, func(ctx context.Context, c *testcloud.Cloud, o ...resource.WaitOption) error {
			return crontriggers.New(mistralClient(c)).WaitForDeletion(ctx, resource.ID("ct-1"), o...)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cloud := testcloud.New(t)
			var calls atomic.Int32
			status := atomic.Int32{}
			status.Store(200)
			cloud.Mux.HandleFunc("GET "+tc.path, func(w http.ResponseWriter, r *http.Request) {
				if n := calls.Add(1); n >= 3 && status.Load() == 200 {
					testcloud.JSON(w, 404, `{}`)
					return
				}
				testcloud.JSON(w, int(status.Load()), tc.body)
			})
			ctx := context.Background()
			fast := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5 * time.Second)}
			if err := tc.wait(ctx, cloud, fast...); err != nil || calls.Load() != 3 {
				t.Fatalf("error=%v calls=%d", err, calls.Load())
			}
			status.Store(500)
			if err := tc.wait(ctx, cloud, fast...); err == nil || errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("a server error must fail the wait: %v", err)
			}
			status.Store(200)
			calls.Store(-1 << 20)
			if err := tc.wait(ctx, cloud, resource.WithPollInterval(time.Millisecond), resource.WithTimeout(50*time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("a resource that stays present must time out: %v", err)
			}
		})
	}
}
