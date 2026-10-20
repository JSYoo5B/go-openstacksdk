package tasks_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/tasks"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTaskTransport func(*http.Request) (*http.Response, error)

func (transport nativeTaskTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeTaskWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeTaskCall struct{ method, path, query, body string }

func nativeTaskAPI(t *testing.T, calls *[]nativeTaskCall, reply func(*http.Request) *http.Response) *tasks.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeTaskTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeTaskCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("image", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/glance/v2/"
	return tasks.New(client)
}

func nativeTaskOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "tasks" {
		t.Fatal("generated tasks context", err, wrapped)
	}
}

const nativeTaskRow = `{"id":"t-1","type":"import","status":"pending","input":{"import_from":"http://x/img.qcow2","image_properties":{"name":"i"}},"result":null,"owner":"p-1","message":"",` +
	`"expires_at":null,"created_at":"2018-07-25T08:59:13Z","updated_at":"2018-07-25T08:59:14Z","self":"/v2/tasks/t-1","schema":"/v2/schemas/task"}`

func TestNativeTaskRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTaskCall
	api := nativeTaskAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodPost:
			return nativeTaskWire(201, nativeTaskRow)
		case req.URL.Path == "/glance/v2/tasks" && req.URL.RawQuery == "marker=t-1":
			return nativeTaskWire(200, `{"tasks":[{"id":"t-2","status":"success"}],"next":""}`)
		case req.URL.Path == "/glance/v2/tasks" && strings.Contains(req.URL.RawQuery, "limit=1"):
			// Only the path and query of next are kept; its host is replaced by the service base.
			return nativeTaskWire(200, `{"tasks":[`+nativeTaskRow+`],"next":"http://elsewhere.invalid/v2/tasks?marker=t-1","first":"/v2/tasks"}`)
		case req.URL.Path == "/glance/v2/tasks":
			return nativeTaskWire(200, `{"tasks":[`+nativeTaskRow+`]}`)
		}
		// Get answers with a task without an envelope.
		return nativeTaskWire(200, nativeTaskRow)
	})
	created, err := api.Create(ctx, tasks.CreateOpts{Type: "import", Input: map[string]any{"import_from": "http://x/img.qcow2"}}, tasks.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "t-1" && created.Status == string(tasks.TaskStatusPending) && created.Owner == "p-1" && created.Result == nil &&
		created.CreatedAt.Equal(time.Date(2018, 7, 25, 8, 59, 13, 0, time.UTC)) && created.ExpiresAt.IsZero() && created.Self == "/v2/tasks/t-1") {
		t.Fatalf("%+v %v", created, err)
	}
	if _, err := api.Create(ctx, tasks.CreateOpts{Type: "import"}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "t-1")
	if err != nil || !(got.ID == "t-1" && got.Input["import_from"] == "http://x/img.qcow2") {
		t.Fatal(got, err)
	}
	var ids []string
	opts := tasks.ListOpts{Limit: 1, SortKey: "created_at", SortDir: "asc", Status: tasks.TaskStatusPending, ID: "t-9", Type: "import"}
	for value, err := range api.List(ctx, tasks.WithListOptions(opts), tasks.WithListQuery("type", "import")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+"/"+value.Status)
	}
	for _, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(ids, []string{"t-1/pending", "t-2/success"}) {
		t.Fatal(ids)
	}
	want := []nativeTaskCall{
		// The body has no envelope and a nil input is sent as null.
		{http.MethodPost, "/glance/v2/tasks", "", `{"input":{"import_from":"http://x/img.qcow2"},"type":"import","x_extension":1}`},
		{http.MethodPost, "/glance/v2/tasks", "", `{"input":null,"type":"import"}`},
		{http.MethodGet, "/glance/v2/tasks/t-1", "", ""},
		// ListOpts.ID and ListOpts.Type carry json tags only, so the native query omits them.
		{http.MethodGet, "/glance/v2/tasks", "limit=1&sort_dir=asc&sort_key=created_at&status=pending&type=import", ""},
		{http.MethodGet, "/glance/v2/tasks", "marker=t-1", ""},
		{http.MethodGet, "/glance/v2/tasks", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeTaskStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*tasks.API) error
	}{
		// Create accepts only 201.
		{"Create", []int{201}, func(api *tasks.API) error {
			_, err := api.Create(ctx, tasks.CreateOpts{Type: "import"})
			return err
		}},
		{"Get", []int{200}, func(api *tasks.API) error { _, err := api.Get(ctx, "t-1"); return err }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeTaskCall
				api := nativeTaskAPI(t, &calls, func(*http.Request) *http.Response { return nativeTaskWire(code, `{}`) })
				err := call.call(api)
				nativeTaskOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("decode", func(t *testing.T) {
		body := `{}`
		var calls []nativeTaskCall
		api := nativeTaskAPI(t, &calls, func(*http.Request) *http.Response { return nativeTaskWire(200, body) })
		if got, err := api.Get(ctx, "t-1"); err != nil || got == nil || got.ID != "" {
			t.Fatal(got, err)
		}
		// A JSON null body leaves the task pointer nil without an error.
		body = `null`
		if got, err := api.Get(ctx, "t-1"); err != nil || got != nil {
			t.Fatal(got, err)
		}
		// Timestamps must be RFC 3339 with a zone.
		body = `{"id":"t-1","created_at":"2018-07-25T08:59:13.000000"}`
		_, err := api.Get(ctx, "t-1")
		nativeTaskOperation(t, err, "Get")
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"tasks":[]}`}, {204, ""}} {
			var calls []nativeTaskCall
			api := nativeTaskAPI(t, &calls, func(*http.Request) *http.Response { return nativeTaskWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeTaskCall
		api := nativeTaskAPI(t, &calls, func(*http.Request) *http.Response { return nativeTaskWire(201, `{}`) })
		for name, check := range map[string]struct {
			opts   tasks.CreateOpts
			option tasks.CreateOption
		}{
			"type":       {tasks.CreateOpts{}, tasks.WithCreateField("x", 1)},
			"core field": {tasks.CreateOpts{Type: "import"}, tasks.WithCreateField("input", map[string]any{})},
			"nil option": {tasks.CreateOpts{Type: "import"}, nil},
		} {
			_, err := api.Create(ctx, check.opts, check.option)
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeTaskOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativeTaskOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
