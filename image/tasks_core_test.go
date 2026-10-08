package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type taskCoreTransport func(*http.Request) (*http.Response, error)

func (f taskCoreTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type taskCoreBody struct {
	reader   io.Reader
	closes   int
	closeErr error
}

func (b *taskCoreBody) Read(p []byte) (int, error) { return b.reader.Read(p) }
func (b *taskCoreBody) Close() error               { b.closes++; return b.closeErr }

type taskCoreReader struct {
	body   string
	err    error
	action func()
	read   bool
}

func (r *taskCoreReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, io.EOF
	}
	r.read = true
	if r.action != nil {
		r.action()
	}
	return copy(p, r.body), r.err
}
func taskCoreClient(f taskCoreTransport) *gophercloud.ServiceClient {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	provider.UseTokenLock()
	provider.SetToken("first-token")
	return &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: "https://glance.example/reverse/glance/v2/", Type: "image"}
}
func taskCoreHTTP(req *http.Request, code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Task-Proof": {"actual"}}, Body: body, Request: req}
}
func taskCoreJSON(req *http.Request, code int, body string) *http.Response {
	return taskCoreHTTP(req, code, io.NopCloser(strings.NewReader(body)))
}
func taskCorePayload(t *testing.T, req *http.Request) map[string]json.RawMessage {
	t.Helper()
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}
func taskCoreProof(t *testing.T, err error, code int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body || proof.Header.Get("X-Task-Proof") != "actual" {
		t.Fatalf("owned proof: %#v, %v", proof, err)
	}
	return proof
}
func taskCoreText(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestTaskCoreFixedRoutesAndObjectInput(t *testing.T) {
	calls := 0
	id := "literal %?#한"
	kind := "future\nkind"
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Auth-Token") != "first-token" {
			t.Fatal("live auth absent")
		}
		if calls == 1 {
			if req.Method != "POST" || req.URL.EscapedPath() != "/reverse/glance/v2/tasks" || req.URL.RawQuery != "" {
				t.Fatal(req.Method, req.URL)
			}
			fields := taskCorePayload(t, req)
			if len(fields) != 2 || taskCoreText(t, fields["type"]) != kind || string(fields["input"]) != "{}" {
				t.Fatal(fields)
			}
			reply := taskCoreJSON(req, 201, `{"id":"returned","type":"future","input":{"n":900719925474099312345},"result":null,"created_at":"literal-date","self":"https://foreign/tasks","unknown":1}`)
			reply.Header.Set("Location", "https://foreign/tasks/other")
			return reply, nil
		}
		if calls == 2 {
			if req.Method != "GET" || req.URL.EscapedPath() != "/reverse/glance/v2/tasks/"+url.PathEscape(id) || req.URL.RawQuery != "" {
				t.Fatal(req.Method, req.URL)
			}
			if req.Body != nil {
				t.Fatal("GET has a body")
			}
			return taskCoreJSON(req, 200, `{"id":"different","status":"future","updated_at":"not-time"}`), nil
		}
		t.Fatal("extra request")
		return nil, nil
	})
	service := New(client)
	created, err := service.CreateTask(context.Background(), kind)
	if err != nil || created == nil || created.StatusCode != 201 || created.ID == nil || *created.ID != "returned" || *created.CreatedAt != "literal-date" || string(created.Input["n"]) != "900719925474099312345" {
		t.Fatal(created, err)
	}
	got, err := service.GetTask(context.Background(), id)
	if err != nil || got == nil || *got.ID != "different" || *got.Status != "future" || *got.UpdatedAt != "not-time" || calls != 2 {
		t.Fatal(got, err, calls)
	}
}

func TestTaskCoreCanonicalPresenceAndAtomicModels(t *testing.T) {
	data := []byte(`{"id":"id","ID":"alias","type":"future","status":"unknown","owner":"","message":"","self":"foreign","schema":"foreign","image_id":"image","request_id":"request","user_id":"user","expires_at":null,"created_at":"not-a-time","updated_at":"","input":{"n":9007199254740993123,"nil":null},"result":{},"links":false,"extra":{"n":1.00000000000000000001}}`)
	var model TaskInfo
	if err := json.Unmarshal(data, &model); err != nil {
		t.Fatal(err)
	}
	if model.ID == nil || *model.ID != "id" || model.ExpiresAt != nil || model.Result == nil || len(model.Result) != 0 || model.Links != nil || *model.CreatedAt != "not-a-time" || string(model.Body["ID"]) != `"alias"` {
		t.Fatal(model)
	}
	model.Input["n"][0] = '8'
	if bytes.Contains(model.Body["input"], []byte("800719")) {
		t.Fatal("typed input aliases raw Body")
	}
	model.Body["result"][0] = '['
	if model.Result == nil || len(model.Result) != 0 {
		t.Fatal("typed result changed")
	}
	data[0] = '['
	if string(model.Body["extra"]) != `{"n":1.00000000000000000001}` {
		t.Fatal("caller input aliases model")
	}
	for _, body := range []string{`{}`, `{"input":null,"result":null,"created_at":null}`} {
		var empty TaskInfo
		if err := json.Unmarshal([]byte(body), &empty); err != nil || empty.Input != nil || empty.Result != nil || empty.CreatedAt != nil {
			t.Fatal(empty, err)
		}
	}
	for _, field := range []string{"id", "type", "status", "owner", "message", "self", "schema", "image_id", "request_id", "user_id", "expires_at", "created_at", "updated_at", "input", "result"} {
		t.Run(field, func(t *testing.T) {
			original := "original"
			value := TaskInfo{ID: &original}
			body := []byte(fmt.Sprintf("{%q:42}", field))
			if err := json.Unmarshal(body, &value); err == nil || value.ID == nil || *value.ID != "original" {
				t.Fatal(value, err)
			}
		})
	}
	for _, body := range []string{`[]`, `null`, `{"input":[]}`, `{"result":"string"}`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		var value TaskInfo
		if err := json.Unmarshal([]byte(body), &value); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestTaskCoreCompletePreflight(t *testing.T) {
	calls, callbacks := 0, 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 201, "{}"), nil })
	service := New(client)
	callback := func(opts *CreateTaskOpts) error { callbacks++; return nil }
	for _, kind := range []string{"", string([]byte{0xff})} {
		if value, err := service.CreateTask(context.Background(), kind, callback); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	for _, id := range []string{"", ".", "..", "a/b", "a\\b", "a\n", string([]byte{0xff})} {
		if value, err := service.GetTask(context.Background(), id); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	for _, input := range []map[string]json.RawMessage{{"x": nil}, {"x": json.RawMessage("NaN")}, {"x": json.RawMessage("{")}, {string([]byte{0xff}): json.RawMessage("1")}, {"x": json.RawMessage([]byte{'"', 0xff, '"'})}} {
		if value, err := service.CreateTask(context.Background(), "import", WithCreateTaskOpts(CreateTaskOpts{Input: input})); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	for _, opts := range []CreateTaskOption{nil, WithCreateTaskHeader("Authorization", "private"), WithCreateTaskHeader("Accept", "other")} {
		if value, err := service.CreateTask(context.Background(), "import", opts); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	for _, opts := range []ListTasksOption{nil, WithListTasksLimit(-1), WithListTasksMaxItems(-1), WithListTasksSortDir("ASC"), WithListTasksMarker("\n"), WithListTasksType(string([]byte{0xff}))} {
		if value, err := service.AllTasks(context.Background(), opts); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	if value, err := service.CreateTask(nil, "import", callback); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(value, err)
	}
	cause := errors.New("stop")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if value, err := service.CreateTask(ctx, "import", callback); value != nil || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal(value, err)
	}
	client.MoreHeaders = map[string]string{"X-Auth-Token": "owned"}
	if value, err := service.CreateTask(context.Background(), "import", callback); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(value, err)
	}
	if calls != 0 || callbacks != 0 {
		t.Fatal("preflight performed work", calls, callbacks)
	}
}

func TestTaskCoreAdvertisedPagesAndLocalConsumption(t *testing.T) {
	t.Run("filters headers auth and prefix", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.EscapedPath() != "/reverse%20proxy/glance/v2/tasks" {
				t.Fatal(req.URL)
			}
			if req.URL.Query().Get("type") != "future-type" || req.URL.Query().Get("status") != "future-status" || req.URL.Query().Get("sort_key") != "future-key" || req.URL.Query().Get("sort_dir") != "asc" || req.URL.Query().Get("limit") != "1" {
				t.Fatal(req.URL)
			}
			if req.Header.Get("X-Fixed") != "option" {
				t.Fatal(req.Header)
			}
			if calls == 1 {
				if req.URL.Query().Get("marker") != "start" || req.Header.Get("X-Source") != "first" {
					t.Fatal(req.URL, req.Header)
				}
				return taskCoreJSON(req, 200, `{"tasks":[{"id":"one"}],"next":"/v2/tasks?limit=1&marker=second&type=future-type&status=future-status&sort_key=future-key&sort_dir=asc"}`), nil
			}
			if calls == 2 {
				if req.URL.Query().Get("marker") != "second" || req.Header.Get("X-Source") != "second" || req.Header.Get("X-Auth-Token") != "second-token" {
					t.Fatal(req.URL, req.Header)
				}
				return taskCoreJSON(req, 200, `{"tasks":[{"id":"two"}],"first":"https://foreign","links":[{"rel":"next","href":"https://foreign"}]}`), nil
			}
			t.Fatal("extra HTTP")
			return nil, nil
		})
		client.Endpoint = "https://glance.example/reverse%20proxy/glance/v2/"
		client.MoreHeaders = map[string]string{"X-Source": "first", "X-Fixed": "source"}
		service := New(client)
		seq := service.Tasks(context.Background(), WithListTasksLimit(1), WithListTasksMarker("start"), WithListTasksType("future-type"), WithListTasksStatus("future-status"), WithListTasksSortKey("future-key"), WithListTasksSortDir("asc"), WithListTasksHeader("X-Fixed", "option"))
		if calls != 0 {
			t.Fatal("not lazy")
		}
		var ids []string
		for value, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, *value.ID)
			if len(ids) == 1 {
				client.MoreHeaders["X-Source"] = "second"
				client.ProviderClient.SetToken("second-token")
			}
		}
		if !reflect.DeepEqual(ids, []string{"one", "two"}) || calls != 2 {
			t.Fatal(ids, calls)
		}
	})
	for _, mode := range []string{"cap", "break", "single"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.RawQuery != "" {
					t.Fatal("local cap invented wire query", req.URL)
				}
				body := `{"tasks":[{"id":"one"},42],"next":42}`
				if mode == "single" {
					body = `{"tasks":[{"id":"one"}],"next":42}`
				}
				return taskCoreJSON(req, 200, body), nil
			})
			service := New(client)
			opts := []ListTasksOption{}
			if mode == "cap" {
				opts = append(opts, WithListTasksMaxItems(1))
			}
			if mode == "single" {
				opts = append(opts, WithListTasksSinglePage(true))
			}
			seen := 0
			for value, err := range service.Tasks(context.Background(), opts...) {
				if err != nil || value == nil {
					t.Fatal(value, err)
				}
				seen++
				if mode == "break" {
					break
				}
			}
			if calls != 1 || seen != 1 {
				t.Fatal(calls, seen)
			}
		})
	}
	t.Run("zero and empty", func(t *testing.T) {
		calls := 0
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.Query().Get("limit") != "0" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 200, `{"tasks":[],"next":false}`), nil
		}))
		values, err := service.AllTasks(context.Background(), WithListTasksLimit(0))
		if err != nil || values == nil || len(values) != 0 || calls != 1 {
			t.Fatal(values, err, calls)
		}
	})
	for _, link := range []string{"https://foreign/v2/tasks?marker=x", "/v2/images?marker=x", "/v2/tasks?marker=x&type=added", "/v2/tasks?marker=x&marker=y", "/v2/tasks?marker=", "/v2/./tasks?marker=x", "/v2/%74asks?marker=x", "//glance.example/v2/tasks?marker=x", "/v2/tasks?marker=x#fragment"} {
		t.Run(link, func(t *testing.T) {
			calls := 0
			body, _ := json.Marshal(map[string]any{"tasks": []any{map[string]string{"id": "one"}}, "next": link})
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return taskCoreJSON(req, 200, string(body)), nil
			}))
			values, err := service.AllTasks(context.Background())
			if values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(values, err, calls)
			}
			taskCoreProof(t, err, 200, string(body))
		})
	}
	t.Run("cycle", func(t *testing.T) {
		calls := 0
		body := `{"tasks":[{"id":"one"}],"next":"/v2/tasks?marker=again"}`
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, body), nil }))
		values, err := service.AllTasks(context.Background())
		var cycle *resource.PaginationCycleError
		if values != nil || !errors.As(err, &cycle) || calls != 2 {
			t.Fatal(values, err, calls)
		}
		taskCoreProof(t, err, 200, body)
	})
	t.Run("late row", func(t *testing.T) {
		body := `{"tasks":[{"id":"one"},{"id":42}]}`
		values, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 200, body), nil })).AllTasks(context.Background())
		if values != nil || err == nil {
			t.Fatal(values, err)
		}
		taskCoreProof(t, err, 200, body)
	})
}

func TestTaskCoreAcceptedEvidenceAndStrictStatus(t *testing.T) {
	for _, method := range []string{"create", "get"} {
		t.Run(method, func(t *testing.T) {
			readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			body := &taskCoreBody{reader: &taskCoreReader{body: "prefix", err: readErr, action: func() { cancel(cause) }}, closeErr: closeErr}
			calls := 0
			code := 200
			if method == "create" {
				code = 201
			}
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, code, body), nil }))
			var value *TaskInfo
			var err error
			if method == "create" {
				value, err = service.CreateTask(ctx, "import")
			} else {
				value, err = service.GetTask(ctx, "literal")
			}
			if value != nil || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != 1 || body.closes != 1 {
				t.Fatal(value, err, calls, body.closes)
			}
			taskCoreProof(t, err, code, "prefix")
		})
	}
	for _, body := range []string{"", `[]`, `{"id":42}`, `{"input":[]}`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		value, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 201, body), nil })).CreateTask(context.Background(), "import")
		if value != nil || err == nil {
			t.Fatal(value, err)
		}
		taskCoreProof(t, err, 201, body)
	}
	for _, code := range []int{202, 204, 400, 403, 404, 409, 429} {
		calls := 0
		value, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return taskCoreJSON(req, code, "native body"), nil
		})).CreateTask(context.Background(), "import")
		if value != nil || !gophercloud.ResponseCodeIs(err, code) || calls != 1 {
			t.Fatal(value, err, calls)
		}
	}
	t.Run("expanded codes retain native cause", func(t *testing.T) {
		calls := 0
		readErr, closeErr := errors.New("read"), errors.New("close")
		body := &taskCoreBody{reader: &taskCoreReader{body: "unexpected", err: readErr}, closeErr: closeErr}
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return taskCoreJSON(req, 503, "retry"), nil
			}
			return taskCoreHTTP(req, 202, body), nil
		})
		client.ProviderClient.RetryFunc = func(ctx context.Context, method, endpoint string, opts *gophercloud.RequestOpts, original error, count uint) error {
			opts.OkCodes = append(opts.OkCodes, 202)
			return nil
		}
		value, err := New(client).CreateTask(context.Background(), "import")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{201}) || string(native.Body) != "unexpected" || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || calls != 2 || body.closes != 1 {
			t.Fatal(value, err, native, calls, body.closes)
		}
	})
}

func TestTaskCoreFixedSourceAndPrebodyPolicy(t *testing.T) {
	for _, field := range []string{"endpoint", "base", "type", "version", "provider", "client"} {
		t.Run(field, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 201, "{}"), nil })
			service := New(client)
			mutate := func() {
				switch field {
				case "endpoint":
					client.Endpoint = "https://glance.example/other/"
				case "base":
					client.ResourceBase = "https://glance.example/other/"
				case "type":
					client.Type = "other"
				case "version":
					client.Microversion = "changed"
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "client":
					service.client = taskCoreClient(nil)
				}
			}
			value, err := service.CreateTask(context.Background(), "import", func(opts *CreateTaskOpts) error { mutate(); return nil })
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(value, err, calls)
			}
		})
	}
	t.Run("accepted source mutation", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		body := &taskCoreBody{reader: &taskCoreReader{body: "{}", err: io.EOF, action: func() { client.Microversion = "changed" }}}
		client = taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreHTTP(req, 201, body), nil })
		value, err := New(client).CreateTask(context.Background(), "import")
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || body.closes != 1 {
			t.Fatal(value, err)
		}
		taskCoreProof(t, err, 201, "{}")
	})
	for _, mutate := range []bool{false, true} {
		t.Run(fmt.Sprint(mutate), func(t *testing.T) {
			calls := 0
			var first []byte
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				if calls == 1 {
					first = body
					return taskCoreJSON(req, 503, "retry"), nil
				}
				if !bytes.Equal(first, body) {
					t.Fatal(string(first), string(body))
				}
				return taskCoreJSON(req, 201, "{}"), nil
			})
			client.ProviderClient.RetryFunc = func(ctx context.Context, method, endpoint string, opts *gophercloud.RequestOpts, original error, count uint) error {
				raw := opts.JSONBody.(json.RawMessage)
				if mutate {
					opts.JSONBody = json.RawMessage(strings.Replace(string(raw), "owned", "rogue", 1))
				} else {
					opts.JSONBody = append(json.RawMessage(nil), raw...)
				}
				return nil
			}
			value, err := New(client).CreateTask(context.Background(), "import", WithCreateTaskInput(map[string]any{"name": "owned"}))
			if mutate {
				if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls != 1 {
					t.Fatal(value, err, calls)
				}
			} else if err != nil || value == nil || calls != 2 {
				t.Fatal(value, err, calls)
			}
		})
	}
}
