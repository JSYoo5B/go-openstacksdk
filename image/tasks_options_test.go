package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type taskOptionMarshaler struct {
	calls *int
	cause error
}

func (value taskOptionMarshaler) MarshalJSON() ([]byte, error) {
	*value.calls++
	if value.cause != nil {
		return nil, value.cause
	}
	return []byte(`{"snap":"owned"}`), nil
}
func taskOptionPointer[T any](value T) *T { return &value }

func TestTaskOptionsImmediateFactoriesAndRawSnapshots(t *testing.T) {
	count := 0
	values := map[string]any{"custom": taskOptionMarshaler{calls: &count}, "precise": json.Number("900719925474099312345"), "nested": map[string]any{"name": "owned"}, "null": nil}
	helper := WithCreateTaskInput(values)
	if count != 1 {
		t.Fatal("factory not immediate", count)
	}
	values["nested"].(map[string]any)["name"] = "changed"
	values["precise"] = 0
	delete(values, "null")
	raw := json.RawMessage(`{"v":"owned"}`)
	headers := map[string]string{"X-Policy": "owned"}
	opts := WithCreateTaskOpts(CreateTaskOpts{Headers: headers, Input: map[string]json.RawMessage{"field": raw}})
	raw[6] = 'X'
	headers["X-Policy"] = "changed"
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		body := taskCorePayload(t, req)
		var input map[string]json.RawMessage
		if err := json.Unmarshal(body["input"], &input); err != nil {
			t.Fatal(err)
		}
		if calls == 1 {
			if string(input["precise"]) != "900719925474099312345" || string(input["nested"]) != `{"name":"owned"}` || string(input["null"]) != "null" || string(input["custom"]) != `{"snap":"owned"}` {
				t.Fatal(input)
			}
		} else {
			if req.Header.Get("X-Policy") != "owned" || string(input["field"]) != `{"v":"owned"}` {
				t.Fatal(input, req.Header)
			}
		}
		return taskCoreJSON(req, 201, "{}"), nil
	}))
	if _, err := service.CreateTask(context.Background(), "import", helper); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTask(context.Background(), "import", opts); err != nil {
		t.Fatal(err)
	}
	if count != 1 || calls != 2 {
		t.Fatal(count, calls)
	}
	badKeyCalls := 0
	badKey := WithCreateTaskInput(map[string]any{string([]byte{0xff}): taskOptionMarshaler{calls: &badKeyCalls}})
	value, err := service.CreateTask(context.Background(), "import", badKey)
	if value != nil || !errors.Is(err, resource.ErrInvalidOption) || badKeyCalls != 0 || calls != 2 {
		t.Fatal(value, err, badKeyCalls, calls)
	}
	cause := errors.New("custom marshal")
	failedCalls := 0
	failed := WithCreateTaskInput(map[string]any{"value": taskOptionMarshaler{calls: &failedCalls, cause: cause}})
	if failedCalls != 1 {
		t.Fatal(failedCalls)
	}
	for range 2 {
		if value, err := service.CreateTask(context.Background(), "import", failed); value != nil || !errors.Is(err, cause) || failedCalls != 1 || calls != 2 {
			t.Fatal(value, err, failedCalls, calls)
		}
	}
}

func TestTaskOptionsReplacementPresenceAndLastWins(t *testing.T) {
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method == "POST" {
			body := taskCorePayload(t, req)
			if string(body["input"]) != "{}" || req.Header.Get("X-Erased") != "" || req.Header.Get("X-Policy") != "last" {
				t.Fatal(body, req.Header)
			}
			return taskCoreJSON(req, 201, "{}"), nil
		}
		if req.Header.Get("X-Policy") != "last" || req.Header.Get("X-Erased") != "" {
			t.Fatal(req.Header)
		}
		if req.URL.Path == "/reverse/glance/v2/tasks/literal" {
			return taskCoreJSON(req, 200, "{}"), nil
		}
		if len(req.URL.Query()) != 1 || req.URL.Query().Get("limit") != "0" {
			t.Fatal(req.URL)
		}
		return taskCoreJSON(req, 200, `{"tasks":[]}`), nil
	}))
	_, err := service.CreateTask(context.Background(), "import", WithCreateTaskInput(map[string]any{"old": 1}), WithCreateTaskHeader("X-Erased", "old"), WithCreateTaskOpts(CreateTaskOpts{Input: map[string]json.RawMessage{}}), WithCreateTaskHeaders(map[string]string{"x-policy": "first"}), WithCreateTaskHeader("X-Policy", "last"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.GetTask(context.Background(), "literal", WithGetTaskHeader("X-Erased", "old"), WithGetTaskOpts(GetTaskOpts{}), WithGetTaskHeaders(map[string]string{"x-policy": "first"}), WithGetTaskHeader("X-Policy", "last"))
	if err != nil {
		t.Fatal(err)
	}
	limit := 2
	base := WithListTasksOpts(ListTasksOpts{Limit: &limit, Marker: "old", Type: "old", Status: "old", SortKey: "old", SortDir: "desc", MaxItems: 4, SinglePage: true})
	limit = -1
	_, err = service.AllTasks(context.Background(), base, WithListTasksHeader("X-Erased", "old"), WithListTasksOpts(ListTasksOpts{}), WithListTasksLimit(0), WithListTasksMarker(""), WithListTasksType(""), WithListTasksStatus(""), WithListTasksSortKey(""), WithListTasksSortDir(""), WithListTasksMaxItems(0), WithListTasksSinglePage(false), WithListTasksHeaders(map[string]string{"x-policy": "first"}), WithListTasksHeader("X-Policy", "last"))
	if err != nil || calls != 3 {
		t.Fatal(err, calls)
	}
	value, err := parseCreateTaskOptions([]CreateTaskOption{WithCreateTaskInput(map[string]any{"value": 1}), WithCreateTaskInput(nil)})
	if err != nil || value.Input == nil || len(value.Input) != 0 {
		t.Fatal(value, err)
	}
	queryValue, query, err := parseListTasksOptions([]ListTasksOption{base})
	if err != nil || *queryValue.Limit != 2 || query.Get("limit") != "2" {
		t.Fatal(queryValue, query, err)
	}
	queryValue.Limit = taskOptionPointer(-1)
	again, _, err := parseListTasksOptions([]ListTasksOption{base})
	if err != nil || *again.Limit != 2 {
		t.Fatal(again, err)
	}
}

func TestTaskOptionsCallbacksAndCompleteValidation(t *testing.T) {
	var retained *CreateTaskOpts
	calls, callbacks := 0, 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		retained.Input["name"][1] = 'X'
		retained.Headers["X-Policy"] = "changed"
		body := taskCorePayload(t, req)
		if string(body["input"]) != `{"name":"owned"}` || req.Header.Get("X-Policy") != "owned" || req.Header.Get("X-Source") != "first" {
			t.Fatal(body, req.Header)
		}
		return taskCoreJSON(req, 201, "{}"), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "first"}
	first := func(config *CreateTaskOpts) error {
		callbacks++
		if config.Headers == nil {
			t.Fatal("header map not initialized")
		}
		config.Headers["X-Policy"] = "owned"
		config.Input = map[string]json.RawMessage{"name": json.RawMessage(`"owned"`)}
		retained = config
		client.MoreHeaders["X-Source"] = "changed"
		return nil
	}
	second := func(config *CreateTaskOpts) error {
		callbacks++
		retained.Input["name"][1] = 'Y'
		retained.Headers["X-Policy"] = "retained"
		if string(config.Input["name"]) != `"owned"` || config.Headers["X-Policy"] != "owned" {
			t.Fatal("callback aliases previous config")
		}
		return nil
	}
	if value, err := New(client).CreateTask(context.Background(), "import", first, second); err != nil || value == nil || calls != 1 || callbacks != 2 {
		t.Fatal(value, err, calls, callbacks)
	}
	cause := errors.New("callback")
	if _, err := New(client).CreateTask(context.Background(), "import", func(config *CreateTaskOpts) error { return cause }); !errors.Is(err, cause) || calls != 1 {
		t.Fatal(err, calls)
	}
	for _, input := range []map[string]json.RawMessage{{"nil": nil}, {"bad": json.RawMessage("NaN")}, {"bad": json.RawMessage([]byte{'"', 0xff, '"'})}, {string([]byte{0xff}): json.RawMessage("1")}} {
		value, err := New(client).CreateTask(context.Background(), "import", func(config *CreateTaskOpts) error { config.Input = input; return nil })
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
			t.Fatal(value, err, calls)
		}
	}
	for _, option := range []CreateTaskOption{WithCreateTaskHeaders(map[string]string{"x-p": "a", "X-P": "b"}), WithCreateTaskInput(map[string]any{"unsupported": func() {}}), WithCreateTaskInput(map[string]any{"bad": json.Number("NaN")})} {
		value, err := New(client).CreateTask(context.Background(), "import", option)
		if value != nil || err == nil || calls != 1 {
			t.Fatal(value, err, calls)
		}
	}
}

func TestTaskOptionsParallelReusableHelpers(t *testing.T) {
	var calls atomic.Int64
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.Header.Get("X-Policy") != "owned" {
			return nil, fmt.Errorf("lost fixed header")
		}
		return taskCoreJSON(req, func() int {
			if req.Method == "POST" {
				return 201
			}
			return 200
		}(), `{"id":"wire","input":{"n":9007199254740993123}}`), nil
	})
	service := New(client)
	create := WithCreateTaskOpts(CreateTaskOpts{Headers: map[string]string{"X-Policy": "owned"}, Input: map[string]json.RawMessage{"n": json.RawMessage("9007199254740993123")}})
	get := WithGetTaskHeaders(map[string]string{"X-Policy": "owned"})
	const workers = 24
	errorsOut := make(chan error, workers*2)
	var workersDone sync.WaitGroup
	for range workers {
		workersDone.Add(1)
		go func() {
			defer workersDone.Done()
			value, err := service.CreateTask(context.Background(), "future", create)
			if err != nil || value == nil {
				errorsOut <- fmt.Errorf("create: %v", err)
				return
			}
			value.Input["n"][0] = '8'
			if string(value.Body["input"]) != `{"n":9007199254740993123}` {
				errorsOut <- fmt.Errorf("typed response aliases raw root")
			}
			value, err = service.GetTask(context.Background(), "literal", get)
			if err != nil || value == nil {
				errorsOut <- fmt.Errorf("get: %v", err)
			}
		}()
	}
	workersDone.Wait()
	close(errorsOut)
	for err := range errorsOut {
		t.Error(err)
	}
	if calls.Load() != workers*2 {
		t.Fatal(calls.Load())
	}
}
