package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/templateloader"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// heatStep is one expected request (method and path?query under /heat) and
// its canned response. Body, when set, is the exact JSON the request must
// carry (compared after decoding).
type heatStep struct {
	method, target string
	body           string
	status         int
	response       string
}

type heatScript struct {
	t     *testing.T
	mu    sync.Mutex
	steps []heatStep
	seen  []string
}

func newHeatScript(t *testing.T, steps ...heatStep) (*heatScript, *gophercloud.ServiceClient) {
	t.Helper()
	cloud := testcloud.New(t)
	script := &heatScript{t: t, steps: steps}
	cloud.Mux.HandleFunc("/", script.serve)
	t.Cleanup(func() {
		script.mu.Lock()
		defer script.mu.Unlock()
		if len(script.steps) != 0 {
			t.Errorf("unused steps %v after %v", script.steps, script.seen)
		}
	})
	return script, cloud.Client("orchestration", "/heat")
}

func (s *heatScript) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := strings.TrimPrefix(r.URL.RequestURI(), "/heat")
	call := r.Method + " " + target
	s.seen = append(s.seen, call)
	if len(s.steps) == 0 {
		s.t.Errorf("unexpected request %s", call)
		http.Error(w, "unexpected", http.StatusTeapot)
		return
	}
	step := s.steps[0]
	s.steps = s.steps[1:]
	if step.method+" "+step.target != call {
		s.t.Errorf("request %s, want %s %s", call, step.method, step.target)
	}
	if step.body != "" {
		raw, _ := io.ReadAll(r.Body)
		var got, want any
		if err := json.Unmarshal(raw, &got); err != nil {
			s.t.Errorf("%s body %q: %v", call, raw, err)
		}
		if err := json.Unmarshal([]byte(step.body), &want); err != nil {
			s.t.Fatalf("bad fixture body: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			s.t.Errorf("%s body\n got %s\nwant %s", call, raw, step.body)
		}
	}
	if step.response == "" {
		w.WriteHeader(step.status)
		return
	}
	testcloud.JSON(w, step.status, step.response)
}

func (s *heatScript) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.seen...)
}

const heatStackID = "54f1bb5e-5c5a-4f4c-9a43-e1b5d5f8d0aa"

func heatStackBody(status string, extra string) string {
	return fmt.Sprintf(`{"stack":{"id":%q,"stack_name":"app","stack_status":%q,"stack_status_reason":"because","links":[]%s}}`, heatStackID, status, extra)
}

func heatEvent(id, resourceName, physical, status string) string {
	return fmt.Sprintf(`{"id":%q,"resource_name":%q,"physical_resource_id":%q,"resource_status":%q,"links":[{"rel":"self","href":"x"},{"rel":"stack","href":"https://heat/stacks/app/%s"}]}`, id, resourceName, physical, status, heatStackID)
}

func heatFast() stackcloud.Option { return stackcloud.WithPollInterval(0) }

func TestHeatCloudSearchStacksFiltersPythonViewByGlobDictAndJMESPath(t *testing.T) {
	t.Parallel()
	rows := `{"stacks":[` +
		`{"id":"1","stack_name":"app-web","stack_status":"CREATE_COMPLETE","tags":["a"],"unknown":1},` +
		`{"id":"2","stack_name":"app-db","stack_status":"CREATE_FAILED","parameters":{"k":{"n":"v"}}},` +
		`{"id":"3","stack_name":"other","stack_status":"CREATE_COMPLETE","disable_rollback":"yes"}]}`
	run := func(input stackcloud.SearchRequest) (*stackcloud.SearchResult, error) {
		_, client := newHeatScript(t, heatStep{"GET", "/stacks", "", 200, rows})
		return stackcloud.SearchStacks(context.Background(), client, input)
	}
	names := func(result *stackcloud.SearchResult) []string {
		var out []string
		for _, row := range result.Stacks {
			out = append(out, row.Name)
		}
		return out
	}
	filter := func(raw string) *json.RawMessage { value := json.RawMessage(raw); return &value }

	result, err := run(stackcloud.SearchRequest{})
	if err != nil || len(result.Stacks) != 3 || len(result.Inventory) != 3 {
		t.Fatalf("unfiltered: %+v %v", result, err)
	}
	var view []map[string]json.RawMessage
	if err := json.Unmarshal(result.Value, &view); err != nil {
		t.Fatal(err)
	}
	if string(view[1]["tags"]) != "[]" || string(view[0]["status"]) != `"CREATE_COMPLETE"` || string(view[2]["is_rollback_disabled"]) != "true" ||
		string(view[0]["location"]) != "null" || view[0]["unknown"] != nil || view[0]["stack_status"] != nil || string(view[0]["name"]) != `"app-web"` {
		t.Fatalf("Python to_dict view: %s", result.Value)
	}
	if result, err = run(stackcloud.SearchRequest{NameOrID: "app-*"}); err != nil || !reflect.DeepEqual(names(result), []string{"app-web", "app-db"}) {
		t.Fatalf("glob: %v %v", names(result), err)
	}
	if result, err = run(stackcloud.SearchRequest{NameOrID: "3"}); err != nil || !reflect.DeepEqual(names(result), []string{"other"}) {
		t.Fatalf("id: %v %v", names(result), err)
	}
	if result, err = run(stackcloud.SearchRequest{NameOrID: "app-*", Filters: filter(`{"status":"CREATE_COMPLETE"}`)}); err != nil || !reflect.DeepEqual(names(result), []string{"app-web"}) {
		t.Fatalf("dict: %v %v", names(result), err)
	}
	if result, err = run(stackcloud.SearchRequest{Filters: filter(`{"parameters":{"k":{"n":"v"}}}`)}); err != nil || !reflect.DeepEqual(names(result), []string{"app-db"}) {
		t.Fatalf("nested: %v %v", names(result), err)
	}
	if result, err = run(stackcloud.SearchRequest{Filters: filter(`{"tags":[]}`)}); err != nil || !reflect.DeepEqual(names(result), []string{"app-db", "other"}) {
		t.Fatalf("default tags: %v %v", names(result), err)
	}
	if result, err = run(stackcloud.SearchRequest{Filters: filter(`"[?status=='CREATE_FAILED'].name"`)}); err != nil || !result.Expression || string(result.Value) != `["app-db"]` || len(result.Stacks) != 0 {
		t.Fatalf("jmespath: %+v %v", result, err)
	}
	if _, err = run(stackcloud.SearchRequest{Filters: filter(`{"stack_status":"CREATE_COMPLETE"}`)}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("wire names are not attributes: %v", err)
	}
	_, client := newHeatScript(t, heatStep{"GET", "/stacks", "", 404, `{"error":{}}`})
	if _, err := stackcloud.SearchStacks(context.Background(), client, stackcloud.SearchRequest{}); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatalf("list error: %v", err)
	}
}

func TestHeatCloudDeleteStackLookupResultAndSingleDelete(t *testing.T) {
	t.Parallel()
	script, client := newHeatScript(t, heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 404, `{}`})
	result, err := stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "app"})
	if err != nil || result.Deleted || result.Deletion != nil {
		t.Fatalf("missing: %+v %v %v", result, err, script.calls())
	}

	_, client = newHeatScript(t,
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("CREATE_COMPLETE", "")},
		heatStep{"DELETE", "/stacks/" + heatStackID, "", 204, ""})
	result, err = stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "app"})
	if err != nil || !result.Deleted || result.Deletion.StatusCode != 204 || result.Stack.ID != heatStackID {
		t.Fatalf("delete: %+v %v", result, err)
	}

	_, client = newHeatScript(t,
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("CREATE_COMPLETE", "")},
		heatStep{"DELETE", "/stacks/" + heatStackID, "", 404, `{}`})
	if result, err = stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "app"}); err != nil || !result.Deleted || !result.NotFound {
		t.Fatalf("ignored 404: %+v %v", result, err)
	}

	_, client = newHeatScript(t,
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("CREATE_COMPLETE", "")},
		heatStep{"DELETE", "/stacks/" + heatStackID, "", 400, `{}`})
	if result, err = stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "app"}); !gophercloud.ResponseCodeIs(err, 400) || result.Deleted {
		t.Fatalf("delete error: %+v %v", result, err)
	}

	// get_stack's _filter_list rejects a result whose id and name differ
	// from the reference.
	_, client = newHeatScript(t, heatStep{"GET", "/stacks/alias?resolve_outputs=False", "", 200, heatStackBody("CREATE_COMPLETE", "")})
	if result, err = stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "alias"}); err != nil || result.Deleted {
		t.Fatalf("name mismatch is missing: %+v %v", result, err)
	}

	_, client = newHeatScript(t, heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("DELETE_COMPLETE", "")})
	if result, err = stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "app"}); err != nil || result.Deleted {
		t.Fatalf("deleted stack is missing: %+v %v", result, err)
	}
}

func TestHeatCloudDeleteStackWaitPollsEventsFromMarkerAndReportsDeleteFailed(t *testing.T) {
	t.Parallel()
	_, client := newHeatScript(t,
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("CREATE_COMPLETE", "")},
		heatStep{"GET", "/stacks/app/events?limit=1&sort_dir=desc", "", 200, `{"events":[` + heatEvent("m1", "app", heatStackID, "CREATE_COMPLETE") + `]}`},
		heatStep{"DELETE", "/stacks/" + heatStackID, "", 204, ""},
		heatStep{"GET", "/stacks/app/events?marker=m1&sort_dir=asc", "", 200, `{"events":[` +
			heatEvent("e1", "server", "srv", "DELETE_COMPLETE") + `,` + heatEvent("e2", "app", heatStackID, "DELETE_COMPLETE") + `]}`},
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 404, `{}`})
	result, err := stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "app", Wait: true}, heatFast())
	if err != nil || !result.Deleted || string(result.Marker) != `"m1"` || result.WaitStatus != "DELETE_COMPLETE" || result.Final != nil {
		t.Fatalf("wait: %+v %v", result, err)
	}

	// By ID, no marker event, an events 404 ends polling silently and the
	// final lookup reports DELETE_FAILED.
	_, client = newHeatScript(t,
		heatStep{"GET", "/stacks/" + heatStackID + "?resolve_outputs=False", "", 200, heatStackBody("CREATE_COMPLETE", "")},
		heatStep{"GET", "/stacks/" + heatStackID + "/events?limit=1&sort_dir=desc", "", 200, `{"events":[]}`},
		heatStep{"DELETE", "/stacks/" + heatStackID, "", 204, ""},
		heatStep{"GET", "/stacks/" + heatStackID + "/events?sort_dir=asc", "", 404, `{}`},
		heatStep{"GET", "/stacks/" + heatStackID + "?resolve_outputs=False", "", 200, heatStackBody("DELETE_FAILED", "")})
	result, err = stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: heatStackID, Wait: true}, heatFast())
	if !errors.Is(err, stackcloud.ErrDeleteFailed) || !strings.Contains(err.Error(), "Failed to delete stack "+heatStackID+": because") {
		t.Fatalf("delete failed: %v", err)
	}
	if result == nil || result.Deletion == nil || result.Marker != nil || !gophercloud.ResponseCodeIs(result.WaitHTTPError, 404) || result.Final.Status != "DELETE_FAILED" {
		t.Fatalf("evidence: %+v", result)
	}

	// A non-HTTP polling failure (malformed events) is not swallowed and keeps
	// the deletion acknowledgement.
	_, client = newHeatScript(t,
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("CREATE_COMPLETE", "")},
		heatStep{"GET", "/stacks/app/events?limit=1&sort_dir=desc", "", 200, `{}`},
		heatStep{"DELETE", "/stacks/" + heatStackID, "", 204, ""},
		heatStep{"GET", "/stacks/app/events?sort_dir=asc", "", 200, `{"events":null}`})
	result, err = stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "app", Wait: true}, heatFast())
	if err == nil || result.Deletion == nil || !result.Deleted {
		t.Fatalf("malformed events: %+v %v", result, err)
	}
}

func TestHeatCloudPollFallsBackToStackGetAfterTwoEmptyPolls(t *testing.T) {
	t.Parallel()
	_, client := newHeatScript(t,
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("CREATE_COMPLETE", "")},
		heatStep{"GET", "/stacks/app/events?limit=1&sort_dir=desc", "", 200, `{"events":[` + heatEvent("m1", "app", heatStackID, "CREATE_COMPLETE") + `]}`},
		heatStep{"DELETE", "/stacks/" + heatStackID, "", 204, ""},
		heatStep{"GET", "/stacks/app/events?marker=m1&sort_dir=asc", "", 200, `{"events":[]}`},
		heatStep{"GET", "/stacks/app/events?marker=m1&sort_dir=asc", "", 200, `{"events":[]}`},
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("DELETE_IN_PROGRESS", "")},
		heatStep{"GET", "/stacks/app/events?marker=m1&sort_dir=asc", "", 200, `{"events":[` + heatEvent("e9", "server", "srv", "DELETE_COMPLETE") + `]}`},
		heatStep{"GET", "/stacks/app/events?marker=e9&sort_dir=asc", "", 200, `{"events":[]}`},
		heatStep{"GET", "/stacks/app/events?marker=e9&sort_dir=asc", "", 200, `{"events":[]}`},
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("DELETE_FAILED", "")},
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("DELETE_FAILED", "")})
	result, err := stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "app", Wait: true}, heatFast())
	if !errors.Is(err, stackcloud.ErrDeleteFailed) || result.WaitStatus != "DELETE_FAILED" {
		t.Fatalf("fallback: %+v %v", result, err)
	}
}

func heatTemplateFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stack.yaml")
	if err := os.WriteFile(path, []byte("heat_template_version: 2015-04-30\ndescription: test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const heatTemplateJSON = `{"heat_template_version":"2015-04-30","description":"test"}`

func TestHeatCloudCreateStackPostsPythonAttributesWaitsAndReturnsDetail(t *testing.T) {
	t.Parallel()
	path := heatTemplateFile(t)
	_, client := newHeatScript(t,
		heatStep{"POST", "/stacks", `{"stack_name":"app","tags":["t1"],"disable_rollback":false,"timeout_mins":60,"parameters":{},"template":` + heatTemplateJSON + `}`, 201, `{"stack":{"id":"` + heatStackID + `"}}`},
		heatStep{"GET", "/stacks/app", "", 200, heatStackBody("CREATE_COMPLETE", "")})
	result, err := stackcloud.CreateStack(context.Background(), client, stackcloud.CreateRequest{Name: "app", StackRequest: stackcloud.StackRequest{
		Tags: []string{"t1"}, Template: templateloader.StackInput{Template: templateloader.Source{File: path}},
	}})
	if err != nil || result.Accepted.StatusCode != 201 || result.Stack == nil || result.Stack.ID != heatStackID || result.WaitStatus != "" {
		t.Fatalf("create: %+v %v", result, err)
	}

	rollback, timeout := false, int64(-1)
	_, client = newHeatScript(t,
		heatStep{"POST", "/stacks", `{"stack_name":"app","tags":null,"disable_rollback":true,"timeout_mins":-1,"parameters":{"size":3,"none":null},"template":` + heatTemplateJSON + `}`, 202, `not json`},
		heatStep{"GET", "/stacks/app/events?sort_dir=asc", "", 200, `{"events":[` + heatEvent("e1", "app", heatStackID, "CREATE_IN_PROGRESS") + `,` + heatEvent("e2", "app", heatStackID, "CREATE_FAILED") + `]}`},
		heatStep{"GET", "/stacks/app", "", 200, heatStackBody("CREATE_FAILED", "")})
	result, err = stackcloud.CreateStack(context.Background(), client, stackcloud.CreateRequest{Name: "app", StackRequest: stackcloud.StackRequest{
		Template: templateloader.StackInput{Template: templateloader.Source{File: path}}, Rollback: &rollback, TimeoutSeconds: &timeout,
		Parameters: map[string]any{"size": 3, "none": nil}, Wait: true,
	}}, heatFast())
	if err != nil || result.WaitStatus != "CREATE_FAILED" || result.Stack.Status != "CREATE_FAILED" {
		t.Fatalf("failed status is returned, not raised: %+v %v", result, err)
	}
}

func TestHeatCloudCreateStackKeepsAcceptedEvidenceAndNeverReplays(t *testing.T) {
	t.Parallel()
	path := heatTemplateFile(t)
	script, client := newHeatScript(t,
		heatStep{"POST", "/stacks", "", 201, `{"stack":{"id":"x"}}`},
		heatStep{"GET", "/stacks/app/events?sort_dir=asc", "", 500, `{}`})
	result, err := stackcloud.CreateStack(context.Background(), client, stackcloud.CreateRequest{Name: "app", StackRequest: stackcloud.StackRequest{
		Template: templateloader.StackInput{Template: templateloader.Source{File: path}}, Wait: true,
	}}, heatFast())
	if !gophercloud.ResponseCodeIs(err, 500) || result == nil || result.Accepted == nil || result.Accepted.StatusCode != 201 || len(script.calls()) != 2 {
		t.Fatalf("wait failure: %+v %v %v", result, err, script.calls())
	}

	script, client = newHeatScript(t, heatStep{"POST", "/stacks", "", 409, `{}`})
	result, err = stackcloud.CreateStack(context.Background(), client, stackcloud.CreateRequest{Name: "app", StackRequest: stackcloud.StackRequest{
		Template: templateloader.StackInput{Template: templateloader.Source{File: path}},
	}})
	if !gophercloud.ResponseCodeIs(err, 409) || result.Accepted != nil || len(script.calls()) != 1 {
		t.Fatalf("rejected create: %+v %v", result, err)
	}

	script, client = newHeatScript(t, heatStep{"POST", "/stacks", "", 201, `[1]`})
	if result, err = stackcloud.CreateStack(context.Background(), client, stackcloud.CreateRequest{Name: "app", StackRequest: stackcloud.StackRequest{
		Template: templateloader.StackInput{Template: templateloader.Source{File: path}},
	}}); err == nil || result.Accepted == nil {
		t.Fatalf("non-object response: %+v %v", result, err)
	}

	_, client = newHeatScript(t)
	if _, err := stackcloud.CreateStack(context.Background(), client, stackcloud.CreateRequest{Name: "app", StackRequest: stackcloud.StackRequest{
		Template: templateloader.StackInput{Template: templateloader.Source{File: filepath.Join(t.TempDir(), "missing.yaml")}},
	}}); !errors.Is(err, templateloader.ErrTemplate) {
		t.Fatalf("loader error before HTTP: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := stackcloud.CreateStack(ctx, client, stackcloud.CreateRequest{Name: "app"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, client = newHeatScript(t,
		heatStep{"POST", "/stacks", "", 201, `{}`},
		heatStep{"GET", "/stacks/app/events?sort_dir=asc", "", 200, `{"events":[]}`})
	result, err = stackcloud.CreateStack(ctx, client, stackcloud.CreateRequest{Name: "app", StackRequest: stackcloud.StackRequest{
		Template: templateloader.StackInput{Template: templateloader.Source{File: path}}, Wait: true,
	}}, stackcloud.WithPollInterval(time.Hour))
	if !errors.Is(err, context.DeadlineExceeded) || result.Accepted == nil {
		t.Fatalf("deadline during wait: %+v %v", result, err)
	}
}

func TestHeatCloudUpdateStackCapturesMarkerPutsAndPollsUpdateEvents(t *testing.T) {
	t.Parallel()
	path := heatTemplateFile(t)
	_, client := newHeatScript(t,
		heatStep{"PUT", "/stacks/app", `{"tags":["t1"],"disable_rollback":false,"timeout_mins":1,"parameters":{},"template":` + heatTemplateJSON + `}`, 202, `{}`},
		heatStep{"GET", "/stacks/app", "", 200, heatStackBody("UPDATE_IN_PROGRESS", "")})
	timeout := int64(119)
	result, err := stackcloud.UpdateStack(context.Background(), client, stackcloud.UpdateRequest{NameOrID: "app", StackRequest: stackcloud.StackRequest{
		Tags: []string{"t1"}, Template: templateloader.StackInput{Template: templateloader.Source{File: path}}, TimeoutSeconds: &timeout,
	}})
	if err != nil || result.Accepted.StatusCode != 202 || result.Stack.Status != "UPDATE_IN_PROGRESS" || result.Marker != nil {
		t.Fatalf("update: %+v %v", result, err)
	}

	_, client = newHeatScript(t,
		heatStep{"GET", "/stacks/app/events?limit=1&sort_dir=desc", "", 200, `{"events":[` + heatEvent("m1", "app", heatStackID, "CREATE_COMPLETE") + `]}`},
		heatStep{"PUT", "/stacks/app", "", 202, `{}`},
		heatStep{"GET", "/stacks/app/events?marker=m1&sort_dir=asc", "", 200, `{"events":[` + heatEvent("e1", "app", heatStackID, "UPDATE_COMPLETE") + `]}`},
		heatStep{"GET", "/stacks/app", "", 200, heatStackBody("UPDATE_COMPLETE", "")})
	result, err = stackcloud.UpdateStack(context.Background(), client, stackcloud.UpdateRequest{NameOrID: "app", StackRequest: stackcloud.StackRequest{
		Template: templateloader.StackInput{Template: templateloader.Source{File: path}}, Wait: true,
	}}, heatFast())
	if err != nil || string(result.Marker) != `"m1"` || result.WaitStatus != "UPDATE_COMPLETE" || result.Stack.Status != "UPDATE_COMPLETE" {
		t.Fatalf("update wait: %+v %v", result, err)
	}

	// A marker capture failure stops before the PUT.
	script, client := newHeatScript(t, heatStep{"GET", "/stacks/app/events?limit=1&sort_dir=desc", "", 200, `{"events":[{"resource_name":"app"}]}`})
	result, err = stackcloud.UpdateStack(context.Background(), client, stackcloud.UpdateRequest{NameOrID: "app", StackRequest: stackcloud.StackRequest{
		Template: templateloader.StackInput{Template: templateloader.Source{File: path}}, Wait: true,
	}}, heatFast())
	if err == nil || result.Accepted != nil || len(script.calls()) != 1 {
		t.Fatalf("marker failure: %+v %v", result, err)
	}

	// A missing stack after the update returns a nil Stack, like Python None.
	_, client = newHeatScript(t,
		heatStep{"PUT", "/stacks/app", "", 202, ``},
		heatStep{"GET", "/stacks/app", "", 404, `{}`})
	result, err = stackcloud.UpdateStack(context.Background(), client, stackcloud.UpdateRequest{NameOrID: "app", StackRequest: stackcloud.StackRequest{
		Template: templateloader.StackInput{Template: templateloader.Source{File: path}},
	}})
	if err != nil || result.Accepted == nil || result.Stack != nil {
		t.Fatalf("missing after update: %+v %v", result, err)
	}
}
