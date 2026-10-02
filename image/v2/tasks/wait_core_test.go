package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

type taskWaitTestTransport func(*http.Request) (*http.Response, error)

func (f taskWaitTestTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func taskWaitTestAPI(transport taskWaitTestTransport) *API {
	provider := &gophercloud.ProviderClient{TokenID: "initial", HTTPClient: http.Client{Transport: transport}}
	provider.UseTokenLock()
	return New(&gophercloud.ServiceClient{ProviderClient: provider, Endpoint: "https://glance.example/v2/", ResourceBase: "https://glance.example/reverse/image/v2/", Type: "image", MoreHeaders: map[string]string{"X-Proof": "source"}})
}

func taskWaitTestHTTP(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Actual": {"evidence"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestTaskWaitCoreOneBudgetRawRecreationFixedRoutesAndLiveToken(t *testing.T) {
	var deadline time.Time
	count := 0
	var api *API
	api = taskWaitTestAPI(func(req *http.Request) (*http.Response, error) {
		count++
		selected, present := req.Context().Deadline()
		if !present || (count > 1 && !selected.Equal(deadline)) {
			t.Errorf("budget changed at request%d: %v -> %v", count, deadline, selected)
		}
		deadline = selected
		if req.Header.Get("X-Proof") != "caller" || req.Header.Get("X-Auth-Token") != map[bool]string{true: "initial", false: "rotated"}[count == 1] {
			t.Errorf("header/provider ownership lost: %#v", req.Header)
		}
		if req.Context().Value("task-proof") != "context" {
			t.Error("caller context value lost")
		}
		switch count {
		case 1:
			if req.Method != "GET" || req.URL.Path != "/reverse/image/v2/tasks/original" {
				t.Errorf("initial route changed: %s %s", req.Method, req.URL)
			}
			api.client.ProviderClient.SetToken("rotated")
			// A changed valid prefix cannot retarget a captured task workflow.
			api.client.ResourceBase = "https://glance.example/changed/"
			return taskWaitTestHTTP(200, `{"id":"decoy","status":"failure","message":"Image cannot be imported. Error code: '396'","type":"import","input":{"large":9007199254740993,"nested":[{"number":1e1000}]}}`), nil
		case 2:
			if req.Method != "POST" || req.URL.Path != "/reverse/image/v2/tasks" {
				t.Errorf("recreation route changed: %s %s", req.Method, req.URL)
			}
			body, _ := io.ReadAll(req.Body)
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(body, &fields); err != nil || len(fields) != 2 || string(fields["type"]) != `"import"` || !strings.Contains(string(fields["input"]), "9007199254740993") || !strings.Contains(string(fields["input"]), "1e1000") {
				t.Errorf("fetched type/input lost: %s %v", body, err)
			}
			return taskWaitTestHTTP(201, `{"id":"new-one","status":"success","input":{"keep":9007199254740993}}`), nil
		case 3:
			if req.Method != "GET" || req.URL.Path != "/reverse/image/v2/tasks/new-one" {
				t.Errorf("new GET was omitted or rerouted: %s %s", req.Method, req.URL)
			}
			return taskWaitTestHTTP(200, `{"id":"another-decoy","status":"failure","message":"Image cannot be imported. Error code: '396'","type":"second","input":null}`), nil
		case 4:
			body, _ := io.ReadAll(req.Body)
			if req.Method != "POST" || string(body) != `{"input":null,"type":"second"}` {
				t.Errorf("second fetched input/type not used: %s %s", req.Method, body)
			}
			return taskWaitTestHTTP(201, `{"id":"new-two","status":"pending","input":{"keep":9007199254740993}}`), nil
		case 5:
			if req.URL.Path != "/reverse/image/v2/tasks/new-two" {
				t.Errorf("latest ID not followed: %s", req.URL)
			}
			return taskWaitTestHTTP(200, `{"id":"response-decoy","status":"SUCCESS","result":{"large":9007199254740993}}`), nil
		default:
			t.Fatalf("unexpected request%d", count)
			return nil, nil
		}
	})
	parent, cancel := context.WithTimeout(context.WithValue(context.Background(), "task-proof", "context"), time.Second)
	defer cancel()
	parentDeadline, _ := parent.Deadline()
	result, err := api.WaitForTask(parent, resource.ID("original"), WithTaskWaitPollInterval(time.Nanosecond), WithTaskWaitHeader("x-proof", "caller"))
	if err != nil || count != 5 || result.OriginalID != "original" || result.CurrentID != "new-two" || result.Recreated != 2 || result.Task.ID != "response-decoy" || result.StatusCode != 200 || result.Created.Task.Status != "pending" || result.Created.StatusCode != 201 {
		t.Fatalf("workflow result=%#v count=%d err=%v", result, count, err)
	}
	if !deadline.Equal(parentDeadline) || result.Task.Result["large"] != json.Number("9007199254740993") || result.Created.Task.Input["keep"] != json.Number("9007199254740993") {
		t.Fatal("parent budget or response precision lost")
	}
	if api.client.MoreHeaders["X-Proof"] != "source" {
		t.Fatal("source header map mutated")
	}
}

func TestTaskWaitCoreFailedCreationRetainsLastTaskAndActualAcceptedProof(t *testing.T) {
	for _, tc := range []struct {
		name, post string
		status     int
		accepted   bool
	}{
		{"native403", `{"error":"forbidden"}`, 403, false},
		{"bad-json", `{]`, 201, true},
		{"missing-id", `{"status":"success"}`, 201, true},
		{"null-id", `{"id":null}`, 201, true},
		{"case-alias-id", `{"ID":"decoy"}`, 201, true},
		{"unsafe-id", `{"id":"unsafe/path"}`, 201, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			api := taskWaitTestAPI(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskWaitTestHTTP(200, `{"id":"original","status":"failure","type":"import","input":{},"message":"Image cannot be imported. Error code: '396'"}`), nil
				}
				if calls != 2 || req.Method != "POST" {
					t.Fatalf("failure replayed: %s request%d", req.Method, calls)
				}
				return taskWaitTestHTTP(tc.status, tc.post), nil
			})
			result, err := api.WaitForTask(context.Background(), resource.ID("original"), WithTaskWaitPollInterval(time.Nanosecond))
			if err == nil || result == nil || result.Task.ID != "original" || result.CurrentID != "original" || result.Recreated != 0 || result.StatusCode != 200 || calls != 2 {
				t.Fatalf("failure lost last actual task: %#v %v calls%d", result, err, calls)
			}
			var operation *resource.OperationError
			if !errors.As(err, &operation) || operation.Resource != "tasks" || operation.Operation != "WaitForTask" {
				t.Fatalf("operation context lost: %v", err)
			}
			var proof *resource.ResponseError
			if tc.accepted {
				if !errors.As(err, &proof) || proof.StatusCode != 201 || string(proof.Body) != tc.post || result.Created == nil || string(result.Created.Body) != tc.post || result.Created.Header.Get("X-Actual") != "evidence" {
					t.Fatalf("accepted POST proof lost: %#v %v", result.Created, err)
				}
			} else if errors.As(err, &proof) || result.Created != nil || !gophercloud.ResponseCodeIs(err, 403) {
				t.Fatalf("native error fabricated accepted proof: %#v %v", result.Created, err)
			}
		})
	}
}

func TestTaskWaitCoreTargetFailureSelectionDeadlineAndPreflight(t *testing.T) {
	for _, tc := range []struct {
		name, target, state, message string
		options                      []TaskWaitOption
		failure                      bool
		calls                        int
	}{
		{"target-priority", "failure", "FAILURE", taskWaitImportError396, nil, false, 1},
		{"ordinary-failure", "success", "failure", "other", nil, true, 1},
		{"exact-message", "success", "failure", strings.ToLower(taskWaitImportError396), nil, true, 1},
		{"empty-failures", "success", "failure", taskWaitImportError396, []TaskWaitOption{WithTaskWaitFailureStates()}, false, 2},
		{"replaced-failures", "success", "failure", taskWaitImportError396, []TaskWaitOption{WithTaskWaitFailureStates("ERROR")}, false, 2},
		{"exact-not-prefix", "success", "failure-extra", "other", nil, false, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			started := time.Now()
			api := taskWaitTestAPI(func(req *http.Request) (*http.Response, error) {
				calls++
				deadline, ok := req.Context().Deadline()
				if !ok || deadline.Sub(started) < 119*time.Second || deadline.Sub(started) > 121*time.Second {
					t.Errorf("120-second service default missing: %v", deadline)
				}
				if req.Method != "GET" {
					t.Fatalf("unselected failure recreated: %s", req.Method)
				}
				if calls == 1 {
					encoded, _ := json.Marshal(map[string]any{"status": tc.state, "message": tc.message})
					return taskWaitTestHTTP(200, string(encoded)), nil
				}
				return taskWaitTestHTTP(200, `{"status":"success"}`), nil
			})
			options := append([]TaskWaitOption{WithTaskWaitPollInterval(time.Nanosecond)}, tc.options...)
			result, err := api.WaitForTaskState(context.Background(), resource.ID("original"), tc.target, options...)
			if result == nil || errors.Is(err, resource.ErrFailedState) != tc.failure || (err != nil && !tc.failure) || calls != tc.calls {
				t.Fatalf("selection result=%#v err=%v calls%d", result, err, calls)
			}
		})
	}
	api := taskWaitTestAPI(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid preflight reached HTTP")
		return nil, nil
	})
	for _, tc := range []struct {
		ctx     context.Context
		ref     resource.Ref
		target  string
		options []TaskWaitOption
	}{
		{nil, resource.ID("id"), "success", nil},
		{context.Background(), resource.Name("task"), "success", nil},
		{context.Background(), resource.ID("unsafe/path"), "success", nil},
		{context.Background(), resource.ID("id"), " ", nil},
		{context.Background(), resource.ID("id"), "success", []TaskWaitOption{nil}},
		{context.Background(), resource.ID("id"), "success", []TaskWaitOption{WithTaskWaitTimeout(0)}},
	} {
		if result, err := api.WaitForTaskState(tc.ctx, tc.ref, tc.target, tc.options...); result != nil || err == nil {
			t.Fatalf("preflight accepted: %#v %v", result, err)
		}
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if result, err := api.WaitForTask(canceled, resource.ID("id")); result != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled caller reached HTTP: %#v %v", result, err)
	}
}

func TestTaskWaitCoreInvalidUTF8NeverRecreatesOrAcceptsCreatedTask(t *testing.T) {
	valid := `{"id":"original","status":"failure","message":"Image cannot be imported. Error code: '396'","type":"import","input":{}}`
	for _, tc := range []struct {
		name, invalid string
		post          bool
	}{
		{"get-type", strings.Replace(valid, "import\"", "im\xffport\"", 1), false},
		{"get-input", strings.Replace(valid, `"input":{}`, "\"input\":{\"text\":\"\xff\"}", 1), false},
		{"post-id", "{\"id\":\"new\xff\",\"status\":\"success\"}", true},
		{"post-input", "{\"id\":\"new\",\"input\":{\"text\":\"\xff\"}}", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			api := taskWaitTestAPI(func(req *http.Request) (*http.Response, error) {
				calls++
				if tc.post && calls == 1 {
					return taskWaitTestHTTP(200, valid), nil
				}
				if (!tc.post && req.Method != "GET") || (tc.post && req.Method != "POST") {
					t.Fatalf("invalid bytes replayed: %s", req.Method)
				}
				status := 200
				if tc.post {
					status = 201
				}
				return taskWaitTestHTTP(status, tc.invalid), nil
			})
			result, err := api.WaitForTask(context.Background(), resource.ID("original"), WithTaskWaitPollInterval(time.Nanosecond))
			var evidence *resource.ResponseError
			if !errors.As(err, &evidence) || string(evidence.Body) != tc.invalid || evidence.Header.Get("X-Actual") != "evidence" {
				t.Fatalf("invalid UTF8 accepted or evidence repaired: %#v %v", result, err)
			}
			if tc.post {
				if calls != 2 || result == nil || result.Task.ID != "original" || result.CurrentID != "original" || result.Recreated != 0 || result.Created == nil || result.Created.Task != nil || string(result.Created.Body) != tc.invalid || result.Created.StatusCode != 201 {
					t.Fatalf("invalid created evidence lost: %#v calls%d", result, calls)
				}
			} else if result != nil || calls != 1 {
				t.Fatalf("invalid GET led to recreation: %#v calls%d", result, calls)
			}
		})
	}
}
