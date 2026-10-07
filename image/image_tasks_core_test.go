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

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestImageTaskCoreFixedRouteAndFullRecords(t *testing.T) {
	for _, id := range []string{"이미지% ?#:parent", strings.Repeat("한", 300)} {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != "GET" || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id)+"/tasks" || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Auth-Token") != "first-token" || req.Header.Get("OpenStack-API-Version") != "image 2.12" {
				t.Fatal(req.Method, req.URL, req.Header)
			}
			return taskCoreJSON(req, 200, `{"tasks":[{"id":"task","image_id":"different","request_id":"request","user_id":"user","type":"future","status":"future","owner":"","message":"","deleted":false,"deleted_at":null,"created_at":"literal date","updated_at":null,"expires_at":null,"input":{"n":900719925474099312345,"null":null},"result":{},"links":false,"request-id":"alias","user":"alias"},{"deleted":true,"deleted_at":""}],"next":42}`), nil
		})
		client.Microversion = "2.12"
		values, err := New(client).AllImageTasks(context.Background(), resource.ID(id))
		if err != nil || calls != 1 || len(values) != 2 {
			t.Fatal(values, err, calls)
		}
		first := values[0]
		if first.ID == nil || *first.ID != "task" || *first.ImageID != "different" || *first.RequestID != "request" || *first.UserID != "user" || first.Deleted == nil || *first.Deleted || first.DeletedAt != nil || first.UpdatedAt != nil || *first.CreatedAt != "literal date" || first.StatusCode != 200 || first.Header.Get("X-Task-Proof") != "actual" || first.Links != nil || first.Result == nil || *values[1].DeletedAt != "" || !*values[1].Deleted || values[1].ImageID != nil {
			t.Fatal(first, values[1])
		}
		if string(first.Input["n"]) != "900719925474099312345" || string(first.Body["request-id"]) != `"alias"` {
			t.Fatal(first)
		}
		first.Input["n"][0] = '8'
		if bytes.Contains(first.Body["input"], []byte("800719")) {
			t.Fatal("input aliases raw body")
		}
		first.Header.Set("X-Task-Proof", "changed")
		if values[1].Header.Get("X-Task-Proof") != "actual" {
			t.Fatal("row headers alias")
		}
	}
}

func TestImageTaskCoreDeletionPresenceAndAtomicDecoder(t *testing.T) {
	for _, tc := range []struct {
		body    string
		deleted *bool
		date    *string
	}{
		{`{}`, nil, nil}, {`{"deleted":null,"deleted_at":null}`, nil, nil},
		{`{"deleted":false,"deleted_at":""}`, taskOptionPointer(false), taskOptionPointer("")},
		{`{"deleted":true,"deleted_at":"not-time"}`, taskOptionPointer(true), taskOptionPointer("not-time")},
	} {
		value := ImageTaskInfo{TaskInfo: TaskInfo{}}
		if err := json.Unmarshal([]byte(tc.body), &value); err != nil || !reflect.DeepEqual(value.Deleted, tc.deleted) || !reflect.DeepEqual(value.DeletedAt, tc.date) {
			t.Fatal(value, err)
		}
	}
	value := ImageTaskInfo{}
	good := `{"id":"owned","deleted":false,"deleted_at":"literal","input":{"n":900719925474099312345},"links":"foreign"}`
	if err := json.Unmarshal([]byte(good), &value); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(value)
	for _, body := range []string{
		`{"deleted":"false"}`, `{"deleted":0}`, `{"deleted":{}}`, `{"deleted_at":false}`,
		`{"id":42,"deleted":true}`, `{"input":[]}`, `{"result":"scalar"}`, `[]`, `null`,
	} {
		if err := json.Unmarshal([]byte(body), &value); err == nil {
			t.Fatal("accepted bad model", body)
		}
		after, _ := json.Marshal(value)
		if !bytes.Equal(before, after) {
			t.Fatal("partial receiver mutation", body)
		}
	}
	if err := json.Unmarshal([]byte(`{"Deleted":true,"deleted-at":9,"request-id":"x","user":"y","input":null,"result":null}`), &value); err != nil || value.Deleted != nil || value.DeletedAt != nil || value.RequestID != nil || value.UserID != nil || value.Input != nil || value.Result != nil || value.ID != nil {
		t.Fatal(value, err)
	}
}

func TestImageTaskCoreCompletePreflight(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ref    resource.Ref
		alter  func(*gophercloud.ServiceClient)
		option ListImageTasksOption
	}{
		{"zero", resource.Ref{}, nil, nil}, {"dot", resource.ID("."), nil, nil}, {"slash", resource.ID("a/b"), nil, nil}, {"backslash", resource.ID("a\\b"), nil, nil}, {"control", resource.ID("a\nb"), nil, nil}, {"utf8", resource.ID(string([]byte{0xff})), nil, nil}, {"nameUTF8", resource.Name(string([]byte{0xff})), nil, nil},
		{"provider", resource.ID("id"), func(c *gophercloud.ServiceClient) { c.ProviderClient = nil }, nil}, {"type", resource.ID("id"), func(c *gophercloud.ServiceClient) { c.Type = "compute" }, nil}, {"sourceheader", resource.ID("id"), func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"X-Auth-Token": "bad"} }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, callbacks := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return taskCoreJSON(req, 200, `{"tasks":[]}`), nil
			})
			if tc.alter != nil {
				tc.alter(client)
			}
			values, err := New(client).AllImageTasks(context.Background(), tc.ref, func(o *ListImageTasksOpts) error { callbacks++; return nil })
			if values != nil || err == nil || calls != 0 || callbacks != 0 {
				t.Fatal(values, err, calls, callbacks)
			}
		})
	}
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { t.Fatal("preflight HTTP"); return nil, nil }))
	for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
		called := false
		if values, err := service.AllImageTasks(ctx, resource.ID("id"), func(o *ListImageTasksOpts) error { called = true; return nil }); values != nil || err == nil || called {
			t.Fatal(values, err, called)
		}
	}
	for _, options := range [][]ListImageTasksOption{{nil}, {WithListImageTasksMaxItems(-1)}, {WithListImageTasksHeader("Host", "foreign")}, {WithListImageTasksHeaders(map[string]string{"X": "\n"})}} {
		if values, err := service.AllImageTasks(context.Background(), resource.Name("parent"), options...); values != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(values, err)
		}
	}
	var nilService *Service
	if values, err := nilService.AllImageTasks(context.Background(), resource.ID("id")); values != nil || err == nil {
		t.Fatal(values, err)
	}
}

func TestImageTaskCoreFiniteConsumptionAndEnvelope(t *testing.T) {
	for _, body := range []string{`{"tasks":[]}`, `{"tasks":[{}],"next":{"foreign":true},"links":false,"first":42}`} {
		calls := 0
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			response := taskCoreJSON(req, 200, body)
			response.Header.Set("Link", `<https://foreign/next>; rel="next"`)
			return response, nil
		}))
		values, err := service.AllImageTasks(context.Background(), resource.ID("id"), WithListImageTasksMaxItems(20))
		if values == nil || err != nil || calls != 1 {
			t.Fatal(values, err, calls)
		}
	}
	body := `{"tasks":[{"id":"first","deleted":false},{"deleted":"invalid"},null],"next":"https://foreign/tasks"}`
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, body), nil }))
	values, err := service.AllImageTasks(context.Background(), resource.ID("id"), WithListImageTasksMaxItems(1))
	if err != nil || len(values) != 1 || calls != 1 {
		t.Fatal(values, err, calls)
	}
	for value, err := range service.ImageTasks(context.Background(), resource.ID("id")) {
		if err != nil || *value.ID != "first" {
			t.Fatal(value, err)
		}
		break
	}
	if calls != 2 {
		t.Fatal(calls)
	}
	values, err = service.AllImageTasks(context.Background(), resource.ID("id"))
	if values != nil || err == nil || calls != 3 {
		t.Fatal(values, err, calls)
	}
	taskCoreProof(t, err, 200, body)
	for _, body := range []string{`{}`, `{"tasks":null}`, `{"Tasks":[]}`, `{"tasks":{}}`, `[]`, `null`, `{"tasks":[null]}`, `{"tasks":[{}],`, string([]byte{'{', '"', 't', 'a', 's', 'k', 's', '"', ':', '[', ']', ',', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		values, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 200, body), nil })).AllImageTasks(context.Background(), resource.ID("id"))
		if values != nil || err == nil {
			t.Fatal(values, err, body)
		}
		taskCoreProof(t, err, 200, body)
	}
}

func TestImageTaskCoreExactNameAndSourceGuards(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			if req.URL.Path != "/reverse/glance/v2/images" || req.URL.Query().Get("name") != "Parent" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 200, `{"images":[{"id":"other","name":"parent"}],"next":"/v2/images?marker=second"}`), nil
		case 2:
			return taskCoreJSON(req, 200, `{"images":[{"id":"selected","name":"Parent"}]}`), nil
		case 3:
			if req.URL.Path != "/reverse/glance/v2/images/selected/tasks" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 200, `{"tasks":[{"image_id":"passive"}]}`), nil
		default:
			t.Fatal("extra request")
			return nil, nil
		}
	})
	if values, err := New(client).AllImageTasks(context.Background(), resource.Name("Parent")); err != nil || len(values) != 1 || calls != 3 {
		t.Fatal(values, err, calls)
	}
	for _, tc := range []struct {
		body     string
		sentinel error
	}{
		{`{"images":[]}`, resource.ErrNotFound}, {`{"images":[{"id":"one","name":"Parent"},{"id":"two","name":"Parent"}]}`, resource.ErrAmbiguous},
	} {
		calls := 0
		values, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if strings.HasSuffix(req.URL.Path, "/tasks") {
				t.Fatal("lookup failure reached tasks")
			}
			return taskCoreJSON(req, 200, tc.body), nil
		})).AllImageTasks(context.Background(), resource.Name("Parent"))
		if values != nil || !errors.Is(err, tc.sentinel) || calls != 1 {
			t.Fatal(values, err, calls)
		}
	}
	for _, field := range []string{"endpoint", "version", "provider"} {
		t.Run(field, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return taskCoreJSON(req, 200, `{"tasks":[]}`), nil
			})
			client.ResourceBase = client.Endpoint
			values, err := New(client).AllImageTasks(context.Background(), resource.ID("id"), func(o *ListImageTasksOpts) error {
				switch field {
				case "endpoint":
					client.Endpoint = "https://glance.example/changed/"
				case "version":
					client.Microversion = "2.17"
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				}
				return nil
			})
			if values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(values, err, calls)
			}
		})
	}
	t.Run("after consumed row", func(t *testing.T) {
		body := `{"tasks":[{},{}]}`
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 200, body), nil })
		count := 0
		for value, err := range New(client).ImageTasks(context.Background(), resource.ID("id")) {
			count++
			if count == 1 {
				if value == nil || err != nil {
					t.Fatal(value, err)
				}
				client.Microversion = "changed"
				continue
			}
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(value, err)
			}
			taskCoreProof(t, err, 200, body)
		}
		if count != 2 {
			t.Fatal(count)
		}
	})
}

func TestImageTaskCoreOwnedErrorsAndNativePolicy(t *testing.T) {
	readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("custom cancel")
	ctx, cancel := context.WithCancelCause(context.Background())
	body := &taskCoreBody{reader: &taskCoreReader{body: "prefix", err: readErr, action: func() { cancel(cause) }}, closeErr: closeErr}
	calls := 0
	values, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 200, body), nil })).AllImageTasks(ctx, resource.ID("id"))
	if values != nil || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != 1 || body.closes != 1 {
		t.Fatal(values, err, calls, body.closes)
	}
	taskCoreProof(t, err, 200, "prefix")
	for _, code := range []int{201, 202, 204, 206, 403, 404, 429, 500} {
		calls := 0
		values, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return taskCoreJSON(req, code, "native"), nil
		})).AllImageTasks(context.Background(), resource.ID("id"))
		if values != nil || !gophercloud.ResponseCodeIs(err, code) || calls != 1 {
			t.Fatal(values, err, calls)
		}
		var proof *resource.ResponseError
		if errors.As(err, &proof) {
			t.Fatal("native rejection claimed accepted response")
		}
	}
	for _, expanded := range []bool{false, true} {
		t.Run(fmt.Sprint(expanded), func(t *testing.T) {
			calls := 0
			var client *gophercloud.ServiceClient
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("X-Auth-Token") != map[bool]string{true: "second-token", false: "first-token"}[calls > 1] {
					t.Fatal(req.Header)
				}
				if calls == 1 {
					return taskCoreJSON(req, 503, "retry"), nil
				}
				if expanded {
					return taskCoreJSON(req, 202, "expanded"), nil
				}
				return taskCoreJSON(req, 200, `{"tasks":[]}`), nil
			})
			client.ProviderClient.RetryFunc = func(ctx context.Context, method, endpoint string, opts *gophercloud.RequestOpts, original error, count uint) error {
				client.ProviderClient.SetToken("second-token")
				if expanded {
					opts.OkCodes = append(opts.OkCodes, 202)
				}
				return nil
			}
			values, err := New(client).AllImageTasks(context.Background(), resource.ID("id"))
			if expanded {
				var native gophercloud.ErrUnexpectedResponseCode
				if values != nil || !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{200}) || string(native.Body) != "expanded" {
					t.Fatal(values, err, native)
				}
			} else if values == nil || err != nil {
				t.Fatal(values, err)
			}
			if calls != 2 {
				t.Fatal(calls)
			}
		})
	}
	t.Run("accepted source change", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		bodyText := `{"tasks":[]}`
		body := &taskCoreBody{reader: &taskCoreReader{body: bodyText, err: io.EOF, action: func() { client.Microversion = "changed" }}}
		client = taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreHTTP(req, 200, body), nil })
		values, err := New(client).AllImageTasks(context.Background(), resource.ID("id"))
		if values != nil || !errors.Is(err, resource.ErrInvalidOption) || body.closes != 1 {
			t.Fatal(values, err, body.closes)
		}
		taskCoreProof(t, err, 200, bodyText)
	})
}
