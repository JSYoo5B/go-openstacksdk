package image_test

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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/image"
	nativeTasks "github.com/JSYoo5B/gophercloudsdk/image/v2/tasks"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const taskContractsPrefix = "/reverse/task/glance/v2/"
const taskContractsBase = "https://glance.invalid" + taskContractsPrefix
const taskContractsType = "future/type\n"
const taskContractsObject = `{"id":"response-id","type":"future/type","status":"future/status","input":{"number":9007199254740993,"nested":{"value":1e1000}},"result":{},"owner":"owner","message":"","self":"https://passive.invalid/task","schema":"https://passive.invalid/schema","image_id":"image","request_id":"request","user_id":"user","created_at":"literal-created","updated_at":"literal-updated","expires_at":"literal-expiry","links":17,"x-number":9007199254740995}`
const taskContractsList = `{"tasks":[` + taskContractsObject + `],"first":"https://passive.invalid/first","schema":42}`

type taskContractsTransport func(*http.Request) (*http.Response, error)

func (f taskContractsTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, err := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, err
}

type taskContractsBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *taskContractsBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type taskContractsReader func([]byte) (int, error)

func (f taskContractsReader) Read(p []byte) (int, error) { return f(p) }
func taskContractsWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual-task"}}, Body: body}
}
func taskContractsClient(f taskContractsTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "image", Endpoint: taskContractsBase}
}

type taskContractsOptions struct {
	create []image.CreateTaskOption
	get    []image.GetTaskOption
	list   []image.ListTasksOption
}

func taskContractsCall(s *image.Service, ctx context.Context, op string, opts taskContractsOptions) ([]*image.TaskInfo, error) {
	switch op {
	case "Create":
		v, err := s.CreateTask(ctx, taskContractsType, opts.create...)
		if v == nil {
			return nil, err
		}
		return []*image.TaskInfo{v}, err
	case "Get":
		v, err := s.GetTask(ctx, "fixed-id", opts.get...)
		if v == nil {
			return nil, err
		}
		return []*image.TaskInfo{v}, err
	case "All":
		return s.AllTasks(ctx, opts.list...)
	case "Tasks":
		rows := make([]*image.TaskInfo, 0)
		for v, err := range s.Tasks(ctx, opts.list...) {
			if err != nil {
				return nil, err
			}
			rows = append(rows, v)
		}
		return rows, nil
	default:
		panic("unknown task operation")
	}
}

var taskContractsOperations = []struct {
	name, method, suffix, raw string
	code                      int
}{
	{"Create", "POST", "", taskContractsObject, 201},
	{"Get", "GET", "/fixed-id", taskContractsObject, 200},
	{"Tasks", "GET", "", taskContractsList, 200},
	{"All", "GET", "", taskContractsList, 200},
}

func taskContractsHeaders() taskContractsOptions {
	return taskContractsOptions{
		create: []image.CreateTaskOption{image.WithCreateTaskHeaders(map[string]string{"X-Option": "owned"}), image.WithCreateTaskHeader("X-Final", "yes")},
		get:    []image.GetTaskOption{image.WithGetTaskHeaders(map[string]string{"X-Option": "owned"}), image.WithGetTaskHeader("X-Final", "yes")},
		list:   []image.ListTasksOption{image.WithListTasksHeaders(map[string]string{"X-Option": "owned"}), image.WithListTasksHeader("X-Final", "yes")},
	}
}
func taskContractsProof(t *testing.T, err error, code int, raw []byte) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Request-Id") != "actual-task" {
		t.Fatalf("response proof: %v %+v", err, proof)
	}
	return proof
}
func taskContractsJSON(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil || body == nil {
		t.Fatalf("JSON object %q: %v", raw, err)
	}
	return body
}
func taskContractsJSONEqual(t *testing.T, raw []byte, want any) {
	t.Helper()
	expected, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	got := json.NewDecoder(bytes.NewReader(raw))
	got.UseNumber()
	expectedDecoder := json.NewDecoder(bytes.NewReader(expected))
	expectedDecoder.UseNumber()
	if got.Decode(&gotValue) != nil || expectedDecoder.Decode(&wantValue) != nil || !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("body got %s want %s", raw, expected)
	}
}

type taskContractsMarshaler struct {
	calls *atomic.Int32
	raw   []byte
	cause error
}

func (m taskContractsMarshaler) MarshalJSON() ([]byte, error) {
	if m.calls != nil {
		m.calls.Add(1)
	}
	return append([]byte(nil), m.raw...), m.cause
}

func TestTaskContractsFixedRoutesAndInput(t *testing.T) {
	for _, op := range taskContractsOperations {
		t.Run(op.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/catalog/unused/")
			client.ResourceBase = cloud.Server.URL + taskContractsPrefix
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			client.Microversion = "2.18"
			cloud.Provider.SetToken("live")
			var calls atomic.Int32
			cloud.Mux.HandleFunc(taskContractsPrefix, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != op.method || r.URL.Path != taskContractsPrefix+"tasks"+op.suffix || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Final") != "yes" || r.Header.Get("OpenStack-API-Version") != "image 2.18" {
					t.Error(r.Method, r.URL, r.Header)
				}
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if op.name == "Create" {
					taskContractsJSONEqual(t, raw, map[string]any{"type": taskContractsType, "input": map[string]any{}})
				} else if len(raw) != 0 {
					t.Error("body on GET", string(raw))
				}
				w.Header().Set("Location", "https://passive.invalid/no-follow-up")
				w.Header().Set("X-Request-Id", "actual-task")
				testcloud.JSON(w, op.code, op.raw)
			})
			rows, err := taskContractsCall(image.New(client), context.Background(), op.name, taskContractsHeaders())
			if err != nil || len(rows) != 1 || calls.Load() != 1 || rows[0].StatusCode != op.code || *rows[0].ID != "response-id" || rows[0].Header.Get("Location") != "https://passive.invalid/no-follow-up" {
				t.Fatal(rows, err, calls.Load())
			}
		})
	}
	for _, id := range []string{"literal%2F ?#한글", strings.Repeat("界", 300)} {
		t.Run("literal task identity "+id, func(t *testing.T) {
			var calls atomic.Int32
			client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.EscapedPath() != taskContractsPrefix+"tasks/"+url.PathEscape(id) || r.URL.RawQuery != "" || r.Method != "GET" || r.Body != nil {
					t.Error(r.URL, r.URL.EscapedPath(), r.Method)
				}
				return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(taskContractsObject)}), nil
			})
			v, err := image.New(client).GetTask(context.Background(), id, image.WithGetTaskOpts(image.GetTaskOpts{}))
			if v == nil || err != nil || calls.Load() != 1 || *v.ID != "response-id" {
				t.Fatal(v, err, calls.Load())
			}
		})
	}
	t.Run("raw Input precision and nullable values", func(t *testing.T) {
		input := map[string]json.RawMessage{"n": json.RawMessage(`9007199254740993123456789`), "exp": json.RawMessage(`1e1000`), "null": json.RawMessage(`null`), "": json.RawMessage(`{"nested":[1,true,null]}`)}
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(r.Body)
			body := taskContractsJSON(t, raw)
			if len(body) != 2 || string(body["type"]) != `"not-an-enum\n"` {
				t.Error(string(raw))
			}
			got := taskContractsJSON(t, body["input"])
			for key, want := range input {
				if !bytes.Equal(got[key], want) {
					t.Errorf("%q precision got%s want%s", key, got[key], want)
				}
			}
			return taskContractsWire(201, &taskContractsBody{Reader: strings.NewReader(taskContractsObject)}), nil
		})
		if v, err := image.New(client).CreateTask(context.Background(), "not-an-enum\n", image.WithCreateTaskOpts(image.CreateTaskOpts{Input: input})); v == nil || err != nil {
			t.Fatal(v, err)
		}
	})
	t.Run("six literal query keys and zero limit", func(t *testing.T) {
		var calls atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			want := url.Values{"limit": {"0"}, "marker": {"not-uuid %#"}, "type": {"api_image_import"}, "status": {"future-status"}, "sort_key": {"future key"}, "sort_dir": {"asc"}}
			if !reflect.DeepEqual(r.URL.Query(), want) || r.Body != nil {
				t.Error(r.URL, r.Body)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[]}`)}), nil
		})
		rows, err := image.New(client).AllTasks(context.Background(), image.WithListTasksLimit(0), image.WithListTasksMarker("not-uuid %#"), image.WithListTasksType("api_image_import"), image.WithListTasksStatus("future-status"), image.WithListTasksSortKey("future key"), image.WithListTasksSortDir("asc"))
		if err != nil || rows == nil || len(rows) != 0 || calls.Load() != 1 {
			t.Fatal(rows, err, calls.Load())
		}
	})
	t.Run("empty string queries omit and local cap no hint", func(t *testing.T) {
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.RawQuery != "" {
				t.Error("invented query", r.URL)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(taskContractsList)}), nil
		})
		rows, err := image.New(client).AllTasks(context.Background(), image.WithListTasksOpts(image.ListTasksOpts{}), image.WithListTasksMarker(""), image.WithListTasksType(""), image.WithListTasksStatus(""), image.WithListTasksSortKey(""), image.WithListTasksSortDir(""), image.WithListTasksMaxItems(1))
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
	})
}

func TestTaskContractsCanonicalModelsAndRawOwnership(t *testing.T) {
	t.Run("all canonical fields literal and independently owned", func(t *testing.T) {
		raw := []byte(taskContractsObject)
		body := &taskContractsBody{Reader: bytes.NewReader(raw)}
		wire := taskContractsWire(200, body)
		client := taskContractsClient(func(*http.Request) (*http.Response, error) { return wire, nil })
		value, err := image.New(client).GetTask(context.Background(), "fixed-id")
		if err != nil || value == nil || value.Links != nil || value.StatusCode != 200 || *value.Type != "future/type" || *value.Status != "future/status" || *value.Owner != "owner" || *value.Message != "" || *value.ImageID != "image" || *value.RequestID != "request" || *value.UserID != "user" || *value.CreatedAt != "literal-created" || *value.UpdatedAt != "literal-updated" || *value.ExpiresAt != "literal-expiry" || string(value.Input["number"]) != "9007199254740993" || value.Result == nil || string(value.Body["x-number"]) != "9007199254740995" || body.closes.Load() != 1 {
			t.Fatal(value, err)
		}
		raw[0] = '!'
		wire.Header.Set("X-Request-Id", "late")
		value.Input["number"][0] = '1'
		*value.ID = "typed changed"
		if string(value.Body["input"]) != `{"number":9007199254740993,"nested":{"value":1e1000}}` || string(value.Body["id"]) != `"response-id"` || value.Header.Get("X-Request-Id") != "actual-task" {
			t.Fatal("typed/raw/borrowed alias", value)
		}
		value.Body["result"] = json.RawMessage(`{"changed":1}`)
		if len(value.Result) != 0 {
			t.Fatal("raw Result aliases typed")
		}
	})
	for _, raw := range []string{`{}`, `{"id":null,"input":null,"result":null,"created_at":null}`, `{"ID":42,"Input":17,"Status":false,"CreatedAt":[]}`, `{"id":"","type":"","status":"","input":{},"result":{}}`} {
		t.Run("optional shape "+raw, func(t *testing.T) {
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				return taskContractsWire(201, &taskContractsBody{Reader: strings.NewReader(raw)}), nil
			})
			value, err := image.New(client).CreateTask(context.Background(), "future")
			if err != nil || value == nil {
				t.Fatal(value, err)
			}
			if strings.Contains(raw, `"input":{}`) {
				if value.Input == nil || value.Result == nil || value.ID == nil || *value.ID != "" {
					t.Fatal(value)
				}
			} else if value.Input != nil || value.Result != nil || value.ID != nil || value.CreatedAt != nil {
				t.Fatal("null/missing/decoy coerced", value)
			}
		})
	}
	for _, field := range []string{"id", "type", "status", "owner", "message", "self", "schema", "image_id", "request_id", "user_id", "expires_at", "created_at", "updated_at", "input", "result"} {
		t.Run("nonnull wrong canonical "+field, func(t *testing.T) {
			raw := []byte(fmt.Sprintf(`{"%s":17}`, field))
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				return taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)}), nil
			})
			value, err := image.New(client).GetTask(context.Background(), "fixed-id")
			taskContractsProof(t, err, 200, raw)
			if value != nil {
				t.Fatal(value, err)
			}
		})
	}
	for _, raw := range [][]byte{[]byte(`null`), []byte(`[]`), []byte(`{"id":"ok"`), []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}} {
		t.Run(fmt.Sprintf("root%x", raw), func(t *testing.T) {
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				return taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)}), nil
			})
			value, err := image.New(client).GetTask(context.Background(), "fixed-id")
			taskContractsProof(t, err, 200, raw)
			if value != nil {
				t.Fatal(value, err)
			}
		})
	}
	t.Run("atomic public decoder and stub fields", func(t *testing.T) {
		value := image.TaskInfo{}
		if err := json.Unmarshal([]byte(taskContractsObject), &value); err != nil {
			t.Fatal(err)
		}
		before, _ := json.Marshal(value.Body)
		if err := json.Unmarshal([]byte(`{"id":"replaced","input":[]}`), &value); err == nil {
			t.Fatal("accepted bad map")
		}
		after, _ := json.Marshal(value.Body)
		if *value.ID != "response-id" || !bytes.Equal(before, after) {
			t.Fatal("partial decoder mutation", value)
		}
		client := taskContractsClient(func(*http.Request) (*http.Response, error) {
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[{"id":"stub","type":"import","status":"pending"}]}`)}), nil
		})
		rows, err := image.New(client).AllTasks(context.Background())
		if err != nil || len(rows) != 1 || rows[0].Input != nil || rows[0].Result != nil || rows[0].Message != nil {
			t.Fatal(rows, err)
		}
	})
}

func TestTaskContractsAdvertisedPagingAndLocalCaps(t *testing.T) {
	t.Run("canonical next refreshes headers and keeps captured target", func(t *testing.T) {
		var calls atomic.Int32
		var client *gophercloud.ServiceClient
		client = taskContractsClient(func(r *http.Request) (*http.Response, error) {
			page := calls.Add(1)
			if r.URL.EscapedPath() != taskContractsPrefix+"tasks" || r.Header.Get("X-Fixed") != "option" {
				t.Error(r.URL, r.Header)
			}
			if page == 1 {
				if r.Header.Get("X-Source") != "first" || r.Header.Get("X-Auth-Token") != "initial" || r.URL.Query().Get("marker") != "" {
					t.Error(r.URL, r.Header)
				}
				client.MoreHeaders = map[string]string{"X-Source": "second", "X-Fixed": "source"}
				client.ProviderClient.SetToken("fresh")
				return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[{"id":"first"}],"next":"/v2/tasks?type=import&limit=2&marker=advance"}`)}), nil
			}
			if page != 2 || r.Header.Get("X-Source") != "second" || r.Header.Get("X-Auth-Token") != "fresh" || !reflect.DeepEqual(r.URL.Query(), url.Values{"type": {"import"}, "limit": {"2"}, "marker": {"advance"}}) {
				t.Error(page, r.URL, r.Header)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[{"id":"second"}]}`)}), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "first"}
		rows, err := image.New(client).AllTasks(context.Background(), image.WithListTasksType("import"), image.WithListTasksLimit(2), image.WithListTasksHeader("X-Fixed", "option"))
		if err != nil || len(rows) != 2 || *rows[0].ID != "first" || *rows[1].ID != "second" || calls.Load() != 2 {
			t.Fatal(rows, err, calls.Load())
		}
		rows[0].Header.Set("X-Request-Id", "first mutated")
		if rows[1].Header.Get("X-Request-Id") != "actual-task" {
			t.Fatal("row headers alias")
		}
	})
	t.Run("exact encoded reverse prefix continues", func(t *testing.T) {
		var calls atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.EscapedPath() != "/reverse%20proxy/glance/v2/tasks" {
				t.Error(r.URL)
			}
			if calls.Add(1) == 1 {
				return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[{}],"next":"https://glance.invalid/reverse%20proxy/glance/v2/tasks?marker=a"}`)}), nil
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[]}`)}), nil
		})
		client.Endpoint = "https://glance.invalid/reverse%20proxy/glance/v2/"
		rows, err := image.New(client).AllTasks(context.Background())
		if err != nil || len(rows) != 1 || calls.Load() != 2 {
			t.Fatal(rows, err, calls.Load())
		}
	})
	for _, next := range []string{
		"https://foreign.invalid/v2/tasks?type=import&marker=a", "//glance.invalid/v2/tasks?type=import&marker=a",
		"https://user@glance.invalid" + taskContractsPrefix + "tasks?type=import&marker=a",
		"/v2/tasks?type=import&marker=a#fragment", "/v2/tasks?marker=a", "/v2/tasks?type=changed&marker=a",
		"/v2/tasks?type=import&type=import&marker=a", "/v2/tasks?type=import&marker=a&marker=b",
		"/v2/tasks?type=import&marker=", "/v2/tasks?type=import&marker=a&unknown=x",
		taskContractsPrefix + "tasks/other?type=import&marker=a", "/v2/%74asks?type=import&marker=a",
		"/v2/%2e%2e/tasks?type=import&marker=a", "/v2/tasks?type=import&marker=%zz",
	} {
		t.Run("unsafe continuation "+next, func(t *testing.T) {
			var calls atomic.Int32
			raw, _ := json.Marshal(map[string]any{"tasks": []any{map[string]any{"id": "good"}}, "next": next})
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)}), nil
			})
			rows, err := image.New(client).AllTasks(context.Background(), image.WithListTasksType("import"))
			taskContractsProof(t, err, 200, raw)
			if rows != nil || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
		})
	}
	t.Run("marker cycle retains last whole page", func(t *testing.T) {
		var calls atomic.Int32
		raw := []byte(`{"tasks":[{}],"next":"/v2/tasks?marker=a"}`)
		client := taskContractsClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)}), nil
		})
		rows, err := image.New(client).AllTasks(context.Background())
		var cycle *resource.PaginationCycleError
		if rows != nil || !errors.As(err, &cycle) || calls.Load() != 2 {
			t.Fatal(rows, err, calls.Load())
		}
		taskContractsProof(t, err, 200, raw)
	})
	for _, mode := range []string{"cap", "break", "single page"} {
		t.Run("unused rows and next "+mode, func(t *testing.T) {
			var calls atomic.Int32
			raw := `{"tasks":[{"id":"good"},null],"next":17}`
			if mode == "single page" {
				raw = `{"tasks":[{"id":"good"}],"next":17}`
			}
			client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("local control changed wire", r.URL)
				}
				return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(raw)}), nil
			})
			s := image.New(client)
			if mode == "break" {
				for row, err := range s.Tasks(context.Background()) {
					if err != nil || row == nil || *row.ID != "good" {
						t.Fatal(row, err)
					}
					break
				}
			} else {
				option := image.WithListTasksMaxItems(1)
				if mode == "single page" {
					option = image.WithListTasksSinglePage(true)
				}
				rows, err := s.AllTasks(context.Background(), option)
				if err != nil || len(rows) != 1 {
					t.Fatal(rows, err)
				}
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
	for _, raw := range []string{`{}`, `{"Tasks":[]}`, `{"tasks":null}`, `{"tasks":{}}`, `{"tasks":[{},null]}`, `{"tasks":[{},` + string([]byte{0xff}) + `]}`} {
		t.Run("strict list envelope "+raw, func(t *testing.T) {
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(raw)}), nil
			})
			rows, err := image.New(client).AllTasks(context.Background())
			taskContractsProof(t, err, 200, []byte(raw))
			if rows != nil {
				t.Fatal(rows, err)
			}
		})
	}
	t.Run("empty and unadvertised pages stop without fallback", func(t *testing.T) {
		for _, raw := range []string{`{"tasks":[],"next":17}`, `{"tasks":[{"id":"last-id"}],"Next":"/v2/tasks?marker=a","first":"/v2/tasks?marker=b","schema":"/v2/tasks?marker=c"}`} {
			var calls atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				wire := taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(raw)})
				wire.Header.Set("Link", `<https://foreign.invalid/tasks>; rel="next"`)
				return wire, nil
			})
			rows, err := image.New(client).AllTasks(context.Background(), image.WithListTasksLimit(1))
			if err != nil || rows == nil || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
		}
	})
	t.Run("later native failure discards earlier rows", func(t *testing.T) {
		var calls atomic.Int32
		client := taskContractsClient(func(*http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[{}],"next":"/v2/tasks?marker=a"}`)}), nil
			}
			return taskContractsWire(503, &taskContractsBody{Reader: strings.NewReader("late unavailable")}), nil
		})
		rows, err := image.New(client).AllTasks(context.Background())
		var code gophercloud.ErrUnexpectedResponseCode
		if rows != nil || !errors.As(err, &code) || code.Actual != 503 || string(code.Body) != "late unavailable" || calls.Load() != 2 {
			t.Fatal(rows, err, calls.Load())
		}
	})
	t.Run("lazy copied option slice reusable concurrently", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Choice") != "owned" {
				t.Error(r.Header)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[{}]}`)}), nil
		})
		options := []image.ListTasksOption{func(o *image.ListTasksOpts) error {
			callbacks.Add(1)
			o.Headers["X-Choice"] = "owned"
			return nil
		}}
		seq := image.New(client).Tasks(context.Background(), options...)
		options[0] = image.WithListTasksHeader("X-Choice", "mutated")
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("iterator eager")
		}
		var wg sync.WaitGroup
		for range 3 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				count := 0
				for row, err := range seq {
					if err != nil || row == nil {
						t.Error(row, err)
					}
					count++
				}
				if count != 1 {
					t.Error(count)
				}
			}()
		}
		wg.Wait()
		if calls.Load() != 3 || callbacks.Load() != 3 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
	})
}

func TestTaskContractsOwnedOptionsAndSource(t *testing.T) {
	t.Run("factory snapshots raw input headers and full replacement", func(t *testing.T) {
		raw := json.RawMessage(`{"n":9007199254740993}`)
		input := map[string]json.RawMessage{"snapshot": raw}
		headers := map[string]string{"X-Choice": "factory"}
		option := image.WithCreateTaskOpts(image.CreateTaskOpts{Input: input, Headers: headers})
		raw[1] = 'x'
		input["later"] = json.RawMessage(`true`)
		headers["X-Choice"] = "late"
		var retained *image.CreateTaskOpts
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(r.Body)
			taskContractsJSONEqual(t, body, map[string]any{"type": "future", "input": map[string]any{"snapshot": json.RawMessage(`{"n":9007199254740993}`)}})
			if r.Header.Get("X-Choice") != "factory" || r.Header.Get("X-Later") != "" {
				t.Error(r.Header)
			}
			return taskContractsWire(201, &taskContractsBody{Reader: strings.NewReader(`{}`)}), nil
		})
		value, err := image.New(client).CreateTask(context.Background(), "future", image.WithCreateTaskHeader("X-Later", "discarded"), option,
			func(o *image.CreateTaskOpts) error { retained = o; return nil },
			func(*image.CreateTaskOpts) error {
				retained.Headers["X-Choice"] = "mutated retained"
				retained.Input["snapshot"][0] = '!'
				return nil
			})
		if value == nil || err != nil {
			t.Fatal(value, err)
		}
	})
	t.Run("input helper immediate serialization and nil reset", func(t *testing.T) {
		var marshals atomic.Int32
		anyInput := map[string]any{"typed": taskContractsMarshaler{calls: &marshals, raw: []byte(`{"n":1e1000}`)}, "number": json.Number("9007199254740993")}
		option := image.WithCreateTaskInput(anyInput)
		anyInput["number"] = 7
		if marshals.Load() != 1 {
			t.Fatal("input factory did not snapshot", marshals.Load())
		}
		var calls atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(r.Body)
			want := map[string]any{"type": "future", "input": map[string]any{"typed": json.RawMessage(`{"n":1e1000}`), "number": json.Number("9007199254740993")}}
			if calls.Add(1) == 3 {
				want["input"] = map[string]any{}
			}
			taskContractsJSONEqual(t, body, want)
			return taskContractsWire(201, &taskContractsBody{Reader: strings.NewReader(`{}`)}), nil
		})
		for range 2 {
			if v, err := image.New(client).CreateTask(context.Background(), "future", option); v == nil || err != nil {
				t.Fatal(v, err)
			}
		}
		if v, err := image.New(client).CreateTask(context.Background(), "future", option, image.WithCreateTaskInput(nil)); v == nil || err != nil || marshals.Load() != 1 {
			t.Fatal(v, err, marshals.Load())
		}
	})
	t.Run("get and list option ownership", func(t *testing.T) {
		var calls atomic.Int32
		headers := map[string]string{"X-Choice": "snapshot"}
		limit := 0
		get := image.WithGetTaskOpts(image.GetTaskOpts{Headers: headers})
		list := image.WithListTasksOpts(image.ListTasksOpts{Headers: headers, Limit: &limit})
		headers["X-Choice"], limit = "late", 42
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Choice") != "snapshot" || r.Header.Get("X-Final") != "last" {
				t.Error(r.Header)
			}
			if strings.HasSuffix(r.URL.Path, "/fixed-id") {
				return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{}`)}), nil
			}
			if r.URL.Query().Get("limit") != "0" {
				t.Error(r.URL)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[]}`)}), nil
		})
		if v, err := image.New(client).GetTask(context.Background(), "fixed-id", get, image.WithGetTaskHeader("X-Final", "first"), image.WithGetTaskHeader("X-Final", "last")); v == nil || err != nil {
			t.Fatal(v, err)
		}
		var retained *image.ListTasksOpts
		rows, err := image.New(client).AllTasks(context.Background(), list, image.WithListTasksHeader("X-Final", "last"),
			func(o *image.ListTasksOpts) error { retained = o; return nil },
			func(*image.ListTasksOpts) error {
				*retained.Limit = 99
				retained.Headers["X-Choice"] = "late"
				return nil
			})
		if rows == nil || err != nil || calls.Load() != 2 {
			t.Fatal(rows, err, calls.Load())
		}
	})
	t.Run("input factory causes and original invalid keys", func(t *testing.T) {
		cause := errors.New("custom marshal cause")
		for _, mode := range []string{"custom cause", "unsupported type", "invalid original key"} {
			t.Run(mode, func(t *testing.T) {
				var calls, marshals atomic.Int32
				input := map[string]any{"x": taskContractsMarshaler{calls: &marshals, raw: []byte(`null`), cause: cause}}
				if mode == "unsupported type" {
					input["x"] = func() {}
				}
				if mode == "invalid original key" {
					input = map[string]any{string([]byte{0xff}): taskContractsMarshaler{calls: &marshals, raw: []byte(`null`)}}
				}
				option := image.WithCreateTaskInput(input)
				client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected wire") })
				value, err := image.New(client).CreateTask(context.Background(), "future", option)
				if value != nil || err == nil || calls.Load() != 0 {
					t.Fatal(value, err, calls.Load())
				}
				switch mode {
				case "custom cause":
					if !errors.Is(err, cause) || marshals.Load() != 1 {
						t.Fatal(err, marshals.Load())
					}
				case "unsupported type":
					var typed *json.UnsupportedTypeError
					if !errors.As(err, &typed) {
						t.Fatal(err)
					}
				case "invalid original key":
					if !errors.Is(err, resource.ErrInvalidOption) || marshals.Load() != 0 {
						t.Fatal(err, marshals.Load())
					}
				}
			})
		}
	})
	for _, mode := range []string{"nil context", "canceled context", "nil service", "nil provider", "wrong service", "bad endpoint", "source protected header", "source version"} {
		t.Run("source preflight "+mode, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected wire") })
			s := image.New(client)
			ctx := context.Background()
			cause := errors.New("custom cancellation")
			want := error(resource.ErrInvalidOption)
			switch mode {
			case "nil context":
				ctx = nil
			case "canceled context":
				c, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx, want = c, cause
			case "nil service":
				s = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong service":
				client.Type = "compute"
				want = resource.ErrUnsupported
			case "bad endpoint":
				client.Endpoint = "https://glance.invalid/v2/?query=x"
			case "source protected header":
				client.MoreHeaders = map[string]string{"X-Auth-Token": "forged"}
			case "source version":
				client.Microversion = "bad\nversion"
			}
			v, err := s.CreateTask(ctx, "future", func(*image.CreateTaskOpts) error { callbacks.Add(1); return nil })
			if v != nil || !errors.Is(err, want) || calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal(v, err, calls.Load(), callbacks.Load())
			}
		})
	}
	for _, id := range []string{"", ".", "..", "with/slash", "with\\slash", "line\n", string([]byte{0xff})} {
		t.Run("unsafe ID "+id, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected wire") })
			v, err := image.New(client).GetTask(context.Background(), id, func(*image.GetTaskOpts) error { callbacks.Add(1); return nil })
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal(v, err, calls.Load(), callbacks.Load())
			}
		})
	}
	for _, option := range []image.ListTasksOption{nil, image.WithListTasksLimit(-1), image.WithListTasksMaxItems(-1), image.WithListTasksSortDir("ASC"), image.WithListTasksMarker("line\n"), image.WithListTasksType(string([]byte{0xff})), image.WithListTasksHeader("Authorization", "forged")} {
		t.Run("invalid list option", func(t *testing.T) {
			var calls atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected wire") })
			v, err := image.New(client).AllTasks(context.Background(), option)
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(v, err, calls.Load())
			}
		})
	}
	for _, option := range []image.CreateTaskOption{nil, image.WithCreateTaskInput(map[string]any{"x": make(chan int)}), image.WithCreateTaskOpts(image.CreateTaskOpts{Input: map[string]json.RawMessage{"x": json.RawMessage(`not JSON`)}}), image.WithCreateTaskHeader("Content-Length", "9")} {
		t.Run("invalid create option", func(t *testing.T) {
			var calls atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected wire") })
			v, err := image.New(client).CreateTask(context.Background(), "future", option)
			if v != nil || err == nil || calls.Load() != 0 {
				t.Fatal(v, err, calls.Load())
			}
		})
	}
	t.Run("callback error and cancellation retain custom causes", func(t *testing.T) {
		for _, cancelNow := range []bool{false, true} {
			var calls, callbacks atomic.Int32
			cause := errors.New("callback cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected wire") })
			v, err := image.New(client).CreateTask(ctx, "future", func(*image.CreateTaskOpts) error {
				callbacks.Add(1)
				if cancelNow {
					cancel(cause)
					return nil
				}
				return cause
			}, func(*image.CreateTaskOpts) error { callbacks.Add(100); return nil })
			wantCallbacks := int32(1)
			if cancelNow {
				wantCallbacks = 101
			}
			if v != nil || !errors.Is(err, cause) || calls.Load() != 0 || callbacks.Load() != wantCallbacks {
				t.Fatal(v, err, calls.Load(), callbacks.Load())
			}
		}
	})
	for _, mode := range []string{"provider", "endpoint", "base", "type", "microversion"} {
		t.Run("callback target drift "+mode, func(t *testing.T) {
			var calls atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected wire") })
			v, err := image.New(client).GetTask(context.Background(), "fixed-id", func(*image.GetTaskOpts) error {
				switch mode {
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "endpoint":
					client.Endpoint = "https://other.invalid/v2/"
				case "base":
					client.ResourceBase = "https://other.invalid/v2/"
				case "type":
					client.Type = "compute"
				case "microversion":
					client.Microversion = "2.19"
				}
				return nil
			})
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(v, err, calls.Load())
			}
		})
	}
	t.Run("source headers captured before callback and live original token", func(t *testing.T) {
		var calls atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Source") != "before" || r.Header.Get("X-Auth-Token") != "live" {
				t.Error(r.Header)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{}`)}), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "before"}
		v, err := image.New(client).GetTask(context.Background(), "fixed-id", func(*image.GetTaskOpts) error {
			client.MoreHeaders["X-Source"] = "after"
			client.SetToken("live")
			return nil
		})
		if v == nil || err != nil || calls.Load() != 1 {
			t.Fatal(v, err, calls.Load())
		}
	})
	t.Run("source drift after yielded row keeps accepted page proof", func(t *testing.T) {
		raw := []byte(`{"tasks":[{"id":"first"},{"id":"second"}]}`)
		var calls atomic.Int32
		client := taskContractsClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)}), nil
		})
		count := 0
		var terminal error
		for row, err := range image.New(client).Tasks(context.Background()) {
			if err != nil {
				terminal = err
				continue
			}
			count++
			if row == nil || *row.ID != "first" {
				t.Error(row)
			}
			client.ResourceBase = "https://other.invalid/v2/"
		}
		taskContractsProof(t, terminal, 200, raw)
		if count != 1 || !errors.Is(terminal, resource.ErrInvalidOption) || calls.Load() != 1 {
			t.Fatal(count, terminal, calls.Load())
		}
	})
	t.Run("native Task models and ignored typed type query remain compatible", func(t *testing.T) {
		var calls atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if strings.HasSuffix(r.URL.Path, "/fixed-id") {
				return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"id":"native","created_at":"2026-10-03T01:02:03Z","input":{"n":9007199254740993}}`)}), nil
			}
			if r.URL.Query().Get("type") != "" || r.URL.Query().Get("status") != "pending" {
				t.Error(r.URL)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[{"id":"native","status":"pending"}]}`)}), nil
		})
		api := nativeTasks.New(client)
		v, err := api.Get(context.Background(), "fixed-id")
		if err != nil || v == nil || v.ID != "native" || v.CreatedAt.IsZero() {
			t.Fatal(v, err)
		}
		rows := 0
		for v, err := range api.List(context.Background(), nativeTasks.WithListOptions(nativeTasks.ListOpts{Type: "import", Status: "pending"})) {
			if err != nil || v == nil {
				t.Fatal(v, err)
			}
			rows++
		}
		if rows != 1 || calls.Load() != 2 {
			t.Fatal(rows, calls.Load())
		}
	})
}

func TestTaskContractsResponseOwnershipAndNativeHooks(t *testing.T) {
	for _, op := range taskContractsOperations {
		for _, failure := range []string{"read", "close", "cancel", "combined", "source drift"} {
			t.Run(op.name+" accepted "+failure, func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("body read cause"), errors.New("body close cause"), errors.New("custom context cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var calls, retries atomic.Int32
				body := &taskContractsBody{Reader: strings.NewReader(op.raw)}
				if failure == "read" || failure == "combined" {
					body.Reader = taskContractsReader(func(p []byte) (int, error) { return copy(p, op.raw), readCause })
				}
				if failure == "close" || failure == "combined" {
					body.closeErr = closeCause
				}
				var client *gophercloud.ServiceClient
				body.onClose = func() {
					if failure == "cancel" || failure == "combined" {
						cancel(cancelCause)
					}
					if failure == "source drift" {
						client.ProviderClient = &gophercloud.ProviderClient{}
					}
				}
				client = taskContractsClient(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return taskContractsWire(op.code, body), nil
				})
				originalProvider := client.ProviderClient
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return nil
				}
				rows, err := taskContractsCall(image.New(client), ctx, op.name, taskContractsOptions{})
				taskContractsProof(t, err, op.code, []byte(op.raw))
				if rows != nil || calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
					t.Fatal(rows, err, calls.Load(), retries.Load(), body.closes.Load())
				}
				if (failure == "read" || failure == "combined") && !errors.Is(err, readCause) {
					t.Fatal(err)
				}
				if (failure == "close" || failure == "combined") && !errors.Is(err, closeCause) {
					t.Fatal(err)
				}
				if (failure == "cancel" || failure == "combined") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal(err)
				}
				if failure == "source drift" && (!errors.Is(err, resource.ErrInvalidOption) || originalProvider == client.ProviderClient) {
					t.Fatal(err)
				}
			})
		}
	}
	for _, op := range taskContractsOperations {
		for _, code := range []int{202, 204, 404, 403} {
			t.Run(fmt.Sprintf("%s unexpected %d", op.name, code), func(t *testing.T) {
				var calls atomic.Int32
				body := &taskContractsBody{Reader: strings.NewReader("server failure")}
				client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return taskContractsWire(code, body), nil })
				rows, err := taskContractsCall(image.New(client), context.Background(), op.name, taskContractsOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if rows != nil || !errors.As(err, &native) || native.Actual != code || native.Method != op.method || native.URL != taskContractsBase+"tasks"+op.suffix || !reflect.DeepEqual(native.Expected, []int{op.code}) || string(native.Body) != "server failure" || native.ResponseHeader.Get("X-Request-Id") != "actual-task" || errors.As(err, &accepted) || calls.Load() != 1 || body.closes.Load() != 1 {
					t.Fatal(rows, err, native, calls.Load(), body.closes.Load())
				}
			})
		}
	}
	t.Run("prebody retries preserve serialized input and provider hooks", func(t *testing.T) {
		for _, initial := range []string{"503", "transport"} {
			t.Run(initial, func(t *testing.T) {
				var calls, hooks atomic.Int32
				transportCause := errors.New("initial transport")
				var bodies []*taskContractsBody
				var wires [][]byte
				var client *gophercloud.ServiceClient
				client = taskContractsClient(func(r *http.Request) (*http.Response, error) {
					body, _ := io.ReadAll(r.Body)
					wires = append(wires, body)
					if r.Method != "POST" || r.URL.String() != taskContractsBase+"tasks" {
						t.Error(r.Method, r.URL)
					}
					if calls.Add(1) == 1 {
						if initial == "transport" {
							return nil, transportCause
						}
						b := &taskContractsBody{Reader: strings.NewReader("retry503")}
						bodies = append(bodies, b)
						return taskContractsWire(503, b), nil
					}
					if r.Header.Get("X-Auth-Token") != "retry token" || r.Header.Get("X-Hook") != "advanced policy" {
						t.Error(r.Header)
					}
					b := &taskContractsBody{Reader: strings.NewReader(`{}`)}
					bodies = append(bodies, b)
					return taskContractsWire(201, b), nil
				})
				client.RetryFunc = func(_ context.Context, method, target string, o *gophercloud.RequestOpts, original error, _ uint) error {
					hooks.Add(1)
					if method != "POST" || target != taskContractsBase+"tasks" || initial == "503" && !gophercloud.ResponseCodeIs(original, 503) || initial == "transport" && !errors.Is(original, transportCause) {
						t.Error(method, target, original)
					}
					o.JSONBody = map[string]any{"type": "future", "input": map[string]any{"n": json.Number("9007199254740993")}}
					o.MoreHeaders = map[string]string{"X-Hook": "advanced policy"}
					client.SetToken("retry token")
					return nil
				}
				provider := client.ProviderClient
				originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
				v, err := image.New(client).CreateTask(context.Background(), "future", image.WithCreateTaskInput(map[string]any{"n": json.Number("9007199254740993")}))
				if v == nil || err != nil || calls.Load() != 2 || hooks.Load() != 1 || !bytes.Equal(wires[0], wires[1]) || client.ProviderClient != provider || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
					t.Fatal(v, err, calls.Load(), hooks.Load(), wires)
				}
				for _, body := range bodies {
					if body.closes.Load() != 1 {
						t.Fatal(body.closes.Load())
					}
				}
			})
		}
	})
	for _, change := range []string{"changed JSON", "in-place JSON", "JSON null", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSON", "expanded OkCodes"} {
		t.Run("private retry ownership "+change, func(t *testing.T) {
			var calls, hooks, borrowedReads atomic.Int32
			callbackCause, closeCause := errors.New("retry callback cause"), errors.New("expanded Close cause")
			borrowed := &taskContractsBody{Reader: taskContractsReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			var bodies []*taskContractsBody
			client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				_, _ = io.ReadAll(r.Body)
				n := calls.Add(1)
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = 200, "expanded200"
				}
				b := &taskContractsBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					b.closeErr = closeCause
				}
				bodies = append(bodies, b)
				return taskContractsWire(code, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
				hooks.Add(1)
				if !gophercloud.ResponseCodeIs(original, 503) || !o.KeepResponseBody || o.JSONResponse != nil || o.RawBody != nil {
					t.Error(original, o)
				}
				switch change {
				case "changed JSON":
					o.JSONBody = map[string]any{"type": "future", "input": map[string]any{"n": 2}}
				case "in-place JSON":
					raw, ok := o.JSONBody.(json.RawMessage)
					if !ok {
						t.Error("not private raw snapshot", o.JSONBody)
						break
					}
					at := bytes.Index(raw, []byte(`"n":1`))
					if at < 0 {
						t.Error(string(raw))
					} else {
						raw[at+4] = '2'
					}
				case "JSON null":
					o.JSONBody = json.RawMessage(`null`)
				case "KeepResponseBody":
					o.KeepResponseBody = false
				case "JSONResponse":
					o.JSONResponse = new(any)
				case "RawBody":
					o.RawBody = borrowed
				case "unsupported JSON":
					o.JSONBody = make(chan int)
				case "expanded OkCodes":
					o.OkCodes = []int{201, 200}
					return nil
				}
				return callbackCause
			}
			originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
			v, err := image.New(client).CreateTask(context.Background(), "future", image.WithCreateTaskInput(map[string]any{"n": 1}))
			if change == "expanded OkCodes" {
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if v != nil || !errors.As(err, &native) || native.Actual != 200 || !reflect.DeepEqual(native.Expected, []int{201}) || native.Method != "POST" || native.URL != taskContractsBase+"tasks" || string(native.Body) != "expanded200" || native.ResponseHeader.Get("X-Request-Id") != "actual-task" || !errors.Is(err, closeCause) || errors.As(err, &accepted) || calls.Load() != 2 {
					t.Fatal(v, err, native, calls.Load())
				}
			} else if v != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 1 {
				t.Fatal(v, err, calls.Load())
			}
			if change == "unsupported JSON" {
				var typed *json.UnsupportedTypeError
				if !errors.As(err, &typed) {
					t.Fatal(err)
				}
			}
			if hooks.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
				t.Fatal(hooks.Load(), borrowedReads.Load(), borrowed.closes.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	t.Run("bodyless GET null replacement cannot retry", func(t *testing.T) {
		var calls atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Body != nil {
				t.Error(r.Body)
			}
			return taskContractsWire(503, &taskContractsBody{Reader: strings.NewReader("original503")}), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.JSONBody = json.RawMessage(`null`)
			return nil
		}
		v, err := image.New(client).GetTask(context.Background(), "fixed-id")
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 1 {
			t.Fatal(v, err, calls.Load())
		}
	})
	for _, mode := range []string{"transport", "callback", "reauth"} {
		t.Run("nested 404 preserved "+mode, func(t *testing.T) {
			cause := errors.New("nested cause")
			nested := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}, Method: "GET", URL: taskContractsBase + "tasks/fixed-id"}
			var calls atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport" {
					return nil, errors.Join(cause, nested)
				}
				code := 503
				if mode == "reauth" {
					code = 401
				}
				return taskContractsWire(code, &taskContractsBody{Reader: strings.NewReader("original native failure")}), nil
			})
			if mode == "callback" {
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					return errors.Join(cause, nested)
				}
			}
			if mode == "reauth" {
				client.ReauthFunc = func(context.Context) error { return errors.Join(cause, nested) }
			}
			v, err := image.New(client).GetTask(context.Background(), "fixed-id")
			if v != nil || err == nil || calls.Load() != 1 {
				t.Fatal(v, err, calls.Load())
			}
			if mode == "reauth" {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &native) || !errors.Is(native.ErrReauth, cause) || !errors.Is(native.ErrReauth, nested) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) {
					t.Fatal("native reauth fields lost", err)
				}
			} else if !errors.Is(err, cause) || !errors.Is(err, nested) || mode == "callback" && !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal(err)
			}
		})
	}
	for _, mode := range []string{"same target", "other target", "other query", "changed method"} {
		t.Run("native redirect policy "+mode, func(t *testing.T) {
			var calls, redirects atomic.Int32
			var bodies []*taskContractsBody
			client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.Method != "GET" || r.URL.String() != taskContractsBase+"tasks/fixed-id" || r.Body != nil {
					t.Error(r.Method, r.URL, r.Body)
				}
				b := &taskContractsBody{Reader: strings.NewReader(`{}`)}
				bodies = append(bodies, b)
				if n == 2 {
					return taskContractsWire(200, b), nil
				}
				wire := taskContractsWire(307, b)
				target := taskContractsBase + "tasks/fixed-id"
				if mode == "other target" {
					target = "https://foreign.invalid/tasks/fixed-id"
				}
				if mode == "other query" {
					target += "?extra=1"
				}
				wire.Header.Set("Location", target)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(r *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				if mode == "changed method" {
					r.Method = "POST"
				}
				return nil
			}
			v, err := image.New(client).GetTask(context.Background(), "fixed-id")
			if mode == "same target" {
				if v == nil || err != nil || calls.Load() != 2 {
					t.Fatal(v, err, calls.Load())
				}
			} else if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(v, err, calls.Load())
			}
			if redirects.Load() != 1 {
				t.Fatal(redirects.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
}
