package image

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestImageTaskOptionsImmediateHeaderSnapshots(t *testing.T) {
	headers := map[string]string{"X-Policy": "owned"}
	whole := WithListImageTasksOpts(ListImageTasksOpts{Headers: headers, MaxItems: 1})
	many := WithListImageTasksHeaders(headers)
	headers["X-Policy"] = "mutated"
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Policy") != "owned" {
			t.Fatal(req.Header)
		}
		return taskCoreJSON(req, 200, `{"tasks":[{},{}]}`), nil
	}))
	values, err := service.AllImageTasks(context.Background(), resource.ID("id"), whole)
	if err != nil || len(values) != 1 {
		t.Fatal(values, err)
	}
	values, err = service.AllImageTasks(context.Background(), resource.ID("id"), many)
	if err != nil || len(values) != 2 || calls != 2 {
		t.Fatal(values, err, calls)
	}
}

func TestImageTaskOptionsReplacementAndLastWins(t *testing.T) {
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Erased") != "" || req.Header.Get("X-Policy") != "last" || req.URL.RawQuery != "" {
			t.Fatal(req.Header, req.URL)
		}
		return taskCoreJSON(req, 200, `{"tasks":[{},{},{}]}`), nil
	}))
	values, err := service.AllImageTasks(context.Background(), resource.ID("id"),
		WithListImageTasksMaxItems(1), WithListImageTasksHeader("X-Erased", "old"),
		WithListImageTasksOpts(ListImageTasksOpts{}),
		WithListImageTasksHeaders(map[string]string{"x-policy": "first"}), WithListImageTasksHeader("X-Policy", "last"),
		WithListImageTasksMaxItems(2), WithListImageTasksMaxItems(0))
	if err != nil || len(values) != 3 || calls != 1 {
		t.Fatal(values, err, calls)
	}
}

func TestImageTaskOptionsCallbacksLazyAndOwned(t *testing.T) {
	calls, callbacks := 0, 0
	var retained *ListImageTasksOpts
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Policy") != "owned" || req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Later") != "" {
			t.Fatal(req.Header)
		}
		return taskCoreJSON(req, 200, `{"tasks":[{},{}]}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	callback := ListImageTasksOption(func(o *ListImageTasksOpts) error {
		callbacks++
		if o.Headers == nil {
			t.Fatal("nil callback headers")
		}
		o.Headers["X-Policy"] = "owned"
		o.MaxItems = 1
		retained = o
		client.MoreHeaders["X-Source"] = "changed"
		return nil
	})
	opts := []ListImageTasksOption{callback, func(o *ListImageTasksOpts) error {
		retained.Headers["X-Policy"] = "rogue"
		retained.MaxItems = 99
		return nil
	}}
	service := New(client)
	iterator := service.ImageTasks(context.Background(), resource.ID("id"), opts...)
	opts[0] = func(o *ListImageTasksOpts) error { t.Fatal("variadic slice not owned"); return nil }
	if calls != 0 || callbacks != 0 {
		t.Fatal(calls, callbacks)
	}
	count := 0
	for value, err := range iterator {
		if err != nil || value == nil {
			t.Fatal(value, err)
		}
		count++
		retained.Headers["X-Later"] = "retained"
	}
	if calls != 1 || callbacks != 1 || count != 1 {
		t.Fatal(calls, callbacks, count)
	}
	cause := errors.New("callback error")
	values, err := service.AllImageTasks(context.Background(), resource.ID("id"), func(o *ListImageTasksOpts) error { return cause })
	if values != nil || !errors.Is(err, cause) || calls != 1 {
		t.Fatal(values, err, calls)
	}
	for _, option := range []ListImageTasksOption{nil, WithListImageTasksMaxItems(-1), WithListImageTasksHeader("Accept", "owned"), WithListImageTasksHeader("X-Auth-Token", "owned")} {
		values, err := service.AllImageTasks(context.Background(), resource.Name("parent"), option)
		if values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
			t.Fatal(values, err, calls)
		}
	}
}

func TestImageTaskOptionsParallelReusablePolicies(t *testing.T) {
	var calls atomic.Int64
	headers := map[string]string{"X-Policy": "shared"}
	option := WithListImageTasksOpts(ListImageTasksOpts{Headers: headers, MaxItems: 1})
	headers["X-Policy"] = "changed"
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.Header.Get("X-Policy") != "shared" {
			t.Error(req.Header)
		}
		return taskCoreJSON(req, 200, `{"tasks":[{},{}]}`), nil
	}))
	var group sync.WaitGroup
	for range 24 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 2 {
				values, err := service.AllImageTasks(context.Background(), resource.ID("id"), option)
				if err != nil || len(values) != 1 {
					t.Error(values, err)
				}
			}
		}()
	}
	group.Wait()
	if calls.Load() != 48 {
		t.Fatal(calls.Load())
	}
}
