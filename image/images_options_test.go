package image

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"gophercloudsdk/resource"
)

func TestImageOptionsImmediateSnapshotsAndPresence(t *testing.T) {
	name := ""
	limit := 0
	size := int64(0)
	flag := false
	headers := map[string]string{"X-Policy": "owned"}
	tags := []string{"tag\n", "", "tag\n"}
	filters := url.Values{"extra": {"first", "last"}}
	full := WithListImagesOpts(ListImagesOpts{Headers: headers, Name: &name, Limit: &limit, SizeMin: &size, Protected: &flag, Hidden: &flag, Tags: tags, Filters: filters})
	many := WithListImagesTags(tags...)
	extension := WithListImagesFilters(filters)
	header := WithGetImageHeaders(headers)
	wholeGet := WithGetImageOpts(GetImageOpts{Headers: headers})
	name = "changed"
	limit = 99
	size = 99
	flag = true
	headers["X-Policy"] = "changed"
	tags[0] = "changed"
	filters["extra"][0] = "changed"
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Path == "/reverse/glance/v2/images/id" {
			if req.Header.Get("X-Policy") != "owned" {
				t.Fatal(req.Header)
			}
			return taskCoreJSON(req, 200, "{}"), nil
		}
		q := req.URL.Query()
		if calls == 1 {
			expected := url.Values{"name": {""}, "limit": {"0"}, "size_min": {"0"}, "protected": {"false"}, "os_hidden": {"false"}, "tag": {"tag\n", "", "tag\n"}, "extra": {"first", "last"}}
			if !reflect.DeepEqual(q, expected) || req.Header.Get("X-Policy") != "owned" {
				t.Fatal(q, req.Header)
			}
		} else if !reflect.DeepEqual(q, url.Values{"tag": {"tag\n", "", "tag\n"}, "extra": {"first", "last"}}) {
			t.Fatal(q)
		}
		return taskCoreJSON(req, 200, `{"images":[]}`), nil
	}))
	if v, e := service.AllImages(context.Background(), full); v == nil || e != nil {
		t.Fatal(v, e)
	}
	if v, e := service.AllImages(context.Background(), many, extension); v == nil || e != nil {
		t.Fatal(v, e)
	}
	for _, option := range []GetImageOption{header, wholeGet} {
		if v, e := service.GetImage(context.Background(), resource.ID("id"), option); v == nil || e != nil {
			t.Fatal(v, e)
		}
	}
	if calls != 4 {
		t.Fatal(calls)
	}
}

func TestImageOptionsReplacementExtensionsAndLastWins(t *testing.T) {
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Erased") != "" || req.Header.Get("X-Policy") != "last" {
			t.Fatal(req.Header)
		}
		if req.URL.Path == "/reverse/glance/v2/images/id" {
			return taskCoreJSON(req, 200, "{}"), nil
		}
		expected := url.Values{"id": {""}, "name": {""}, "visibility": {"future"}, "member_status": {"future"}, "owner": {""}, "status": {"future"}, "sort": {""}, "created_at": {"literal\n"}, "updated_at": {""}, "container_format": {"future"}, "disk_format": {"future"}, "size_min": {"9"}, "size_max": {"0"}, "protected": {"false"}, "os_hidden": {"false"}, "limit": {"0"}, "kept": {"last", "last"}}
		if !reflect.DeepEqual(req.URL.Query(), expected) {
			t.Fatal(req.URL.Query(), expected)
		}
		return taskCoreJSON(req, 200, `{"images":[{},{}]}`), nil
	}))
	values, err := service.AllImages(context.Background(),
		WithListImagesMaxItems(-1), WithListImagesLimit(-1), WithListImagesHeader("X-Erased", "old"), WithListImagesFilter("old", "x"),
		WithListImagesOpts(ListImagesOpts{}), WithListImagesHeaders(map[string]string{"x-policy": "first"}), WithListImagesHeader("X-Policy", "last"),
		WithListImagesID(""), WithListImagesName(""), WithListImagesVisibility("future"), WithListImagesMemberStatus("future"), WithListImagesOwner(""), WithListImagesStatus("future"),
		WithListImagesSort(""), WithListImagesCreatedAt("literal\n"), WithListImagesUpdatedAt(""), WithListImagesContainerFormat("future"), WithListImagesDiskFormat("future"),
		WithListImagesSizeMin(9), WithListImagesSizeMax(0), WithListImagesProtected(false), WithListImagesHidden(false), WithListImagesLimit(0), WithListImagesMarker(""),
		WithListImagesFilters(url.Values{"erased": {"x"}, "kept": {"first"}}), WithListImagesFilter("kept", "last", "last"), WithListImagesFilter("erased"), WithListImagesMaxItems(1), WithListImagesMaxItems(0), WithListImagesSinglePage(true))
	if err != nil || len(values) != 2 {
		t.Fatal(values, err)
	}
	if value, err := service.GetImage(context.Background(), resource.ID("id"), WithGetImageHeader("X-Erased", "old"), WithGetImageOpts(GetImageOpts{}), WithGetImageHeader("X-Policy", "first"), WithGetImageHeader("x-policy", "last")); value == nil || err != nil || calls != 2 {
		t.Fatal(value, err, calls)
	}
}

func TestImageOptionsLazyCallbacksAndRetainedOwnership(t *testing.T) {
	calls, callbacks := 0, 0
	var retained *ListImagesOpts
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		q := req.URL.Query()
		if q.Get("name") != "owned" || q.Get("extra") != "owned" || q.Get("tag") != "owned" || q.Get("limit") != "1" || req.Header.Get("X-Policy") != "owned" || req.Header.Get("X-Later") != "" || req.Header.Get("X-Source") != map[bool]string{false: "captured", true: "changed"}[calls > 1] {
			t.Fatal(req.URL, req.Header)
		}
		if calls == 1 {
			return taskCoreJSON(req, 200, `{"images":[{"id":"first"}],"next":"?name=owned&extra=owned&tag=owned&limit=1&marker=next"}`), nil
		}
		return taskCoreJSON(req, 200, `{"images":[{"id":"last"}]}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	options := []ListImagesOption{func(o *ListImagesOpts) error {
		callbacks++
		if o.Headers == nil || o.Filters == nil {
			t.Fatal("callback maps not initialized")
		}
		o.Name = taskOptionPointer("owned")
		o.Limit = taskOptionPointer(1)
		o.Headers["X-Policy"] = "owned"
		o.Filters["extra"] = []string{"owned"}
		o.Tags = []string{"owned"}
		retained = o
		client.MoreHeaders["X-Source"] = "changed"
		return nil
	}, func(o *ListImagesOpts) error {
		*retained.Name = "rogue"
		*retained.Limit = 99
		retained.Headers["X-Policy"] = "rogue"
		retained.Filters["extra"][0] = "rogue"
		retained.Tags[0] = "rogue"
		return nil
	}}
	service := New(client)
	iterator := service.ListImages(context.Background(), options...)
	options[0] = func(o *ListImagesOpts) error { t.Fatal("option slice not snapshotted"); return nil }
	if calls != 0 || callbacks != 0 {
		t.Fatal(calls, callbacks)
	}
	count := 0
	for value, err := range iterator {
		if value == nil || err != nil {
			t.Fatal(value, err)
		}
		count++
		retained.Headers["X-Later"] = "rogue"
		retained.Filters["extra"][0] = "later"
	}
	if calls != 2 || callbacks != 1 || count != 2 {
		t.Fatal(calls, callbacks, count)
	}
	cause := errors.New("custom callback")
	values, err := service.AllImages(context.Background(), func(o *ListImagesOpts) error { return cause })
	if values != nil || !errors.Is(err, cause) || calls != 2 {
		t.Fatal(values, err, calls)
	}
}

func TestImageOptionsValidationAndParallelReusablePolicies(t *testing.T) {
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		t.Error("invalid option reached HTTP")
		return taskCoreJSON(req, 200, `{"images":[]}`), nil
	}))
	invalid := []ListImagesOption{nil, WithListImagesLimit(-1), WithListImagesMaxItems(-1), WithListImagesSizeMin(-1), WithListImagesSizeMax(-1),
		WithListImagesHeader("Accept", "owned"), WithListImagesHeader("Host", "foreign"), WithListImagesHeader("X-Auth-Token", "owned"),
		WithListImagesFilters(url.Values{"limit": {"1"}}), WithListImagesFilters(url.Values{"deleted": {"true"}}), WithListImagesFilters(url.Values{"x": nil}),
		WithListImagesFilter("", "value"), WithListImagesFilter("key\n", "value"), WithListImagesFilter(string([]byte{0xff}), "value"), WithListImagesFilter("key", string([]byte{0xff})),
		WithListImagesName(string([]byte{0xff})), WithListImagesMarker(string([]byte{0xff})), WithListImagesTags(string([]byte{0xff})),
		func(o *ListImagesOpts) error {
			o.Sort = taskOptionPointer("")
			o.SortKeys = []string{"name"}
			return nil
		},
		func(o *ListImagesOpts) error {
			o.SortDirs = []string{"asc", "desc"}
			o.SortKeys = []string{"name"}
			return nil
		},
	}
	for index, option := range invalid {
		values, err := service.AllImages(context.Background(), option)
		if values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(index, values, err, calls)
		}
	}
	for _, option := range []GetImageOption{nil, WithGetImageHeader("Accept", "owned")} {
		value, err := service.GetImage(context.Background(), resource.Name("Parent"), option)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(value, err, calls)
		}
	}
	var atomicCalls atomic.Int64
	headers := map[string]string{"X-Policy": "parallel"}
	tags := []string{"a", "a"}
	policy := WithListImagesOpts(ListImagesOpts{Headers: headers, Tags: tags, Filters: url.Values{"key": {"value"}}, MaxItems: 1})
	headers["X-Policy"] = "changed"
	tags[0] = "changed"
	parallel := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		atomicCalls.Add(1)
		if req.Header.Get("X-Policy") != "parallel" || !reflect.DeepEqual(req.URL.Query(), url.Values{"tag": {"a", "a"}, "key": {"value"}}) {
			t.Error(req.URL, req.Header)
		}
		return taskCoreJSON(req, 200, `{"images":[{},{"tags":[null]}],"next":false}`), nil
	}))
	var group sync.WaitGroup
	for range 20 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 2 {
				values, err := parallel.AllImages(context.Background(), policy)
				if err != nil || len(values) != 1 {
					t.Error(values, err)
				}
			}
		}()
	}
	group.Wait()
	if atomicCalls.Load() != 40 {
		t.Fatal(atomicCalls.Load())
	}
}
