package volumes_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonVolumeSummarySelectsDetailsAndPreservesQuery(t *testing.T) {
	for _, details := range []bool{false, true} {
		t.Run(fmt.Sprint(details), func(t *testing.T) {
			api, calls := pythonVolumeAPI(t, func(req *http.Request) *http.Response {
				if req.Header.Get("X-Auth-Token") != "test-token" {
					t.Error("missing auth", req.Header)
				}
				if req.URL.Query().Get("marker") != "" {
					return pythonVolumeWire(200, `{"volumes":[{"id":"vol-2","name":"second"}]}`)
				}
				return pythonVolumeWire(200, `{"volumes":[{"id":"vol-1","name":"first"}],"volumes_links":[{"rel":"next","href":"http://`+req.Host+req.URL.Path+`?marker=vol-1"}]}`)
			})
			var ids []string
			for value, err := range api.ListWithDetails(context.Background(), details,
				volumes.WithListOptions(volumes.ListOpts{Name: "data", Status: "available", Limit: 4}),
				volumes.WithListQuery("all_tenants", "True")) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, value.ID)
			}
			path := pythonVolumeBase
			if details {
				path += "/detail"
			}
			if !reflect.DeepEqual(ids, []string{"vol-1", "vol-2"}) || len(*calls) != 2 {
				t.Fatal(ids, *calls)
			}
			first, _ := url.ParseQuery((*calls)[0].query)
			expected := url.Values{"name": {"data"}, "status": {"available"}, "limit": {"4"}, "all_tenants": {"True"}}
			if (*calls)[0].path != path || (*calls)[1].path != path || (*calls)[0].method != "GET" || (*calls)[0].body != "" || !reflect.DeepEqual(first, expected) || (*calls)[1].query != "marker=vol-1" {
				t.Fatal(*calls)
			}
		})
	}
}

func TestPythonVolumeSummaryLocalControlsAndConsumerBreak(t *testing.T) {
	for _, check := range []struct {
		name    string
		control resource.ListControl
		stop    bool
		want    []string
	}{
		{"raw row cap", resource.ListControl{MaxItems: 1}, false, []string{"vol-1"}},
		{"first page", resource.ListControl{SinglePage: true}, false, []string{"vol-1", "vol-2"}},
		{"consumer break", resource.ListControl{}, true, []string{"vol-1"}},
	} {
		t.Run(check.name, func(t *testing.T) {
			api, calls := pythonVolumeAPI(t, func(req *http.Request) *http.Response {
				return pythonVolumeWire(200, `{"volumes":[{"id":"vol-1"},{"id":"vol-2"}],"volumes_links":[{"rel":"next","href":"http://`+req.Host+req.URL.Path+`?marker=vol-2"}]}`)
			})
			var ids []string
			for value, err := range api.ListSummaryWithControl(context.Background(), check.control, volumes.WithListOptions(volumes.ListOpts{Limit: 9})) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, value.ID)
				if check.stop {
					break
				}
			}
			if !reflect.DeepEqual(ids, check.want) || len(*calls) != 1 || (*calls)[0].path != pythonVolumeBase || (*calls)[0].query != "limit=9" {
				t.Fatal(ids, *calls)
			}
		})
	}
	t.Run("whole native page decode remains observable beyond cap", func(t *testing.T) {
		api, calls := pythonVolumeAPI(t, func(*http.Request) *http.Response {
			return pythonVolumeWire(200, `{"volumes":[{"id":"vol-1"},{"id":"vol-2","size":"broken"}]}`)
		})
		rows := 0
		var failure error
		for _, err := range api.ListSummaryWithControl(context.Background(), resource.ListControl{MaxItems: 1}) {
			if err != nil {
				failure = err
			} else {
				rows++
			}
		}
		if failure == nil || rows != 0 || len(*calls) != 1 {
			t.Fatal(rows, failure, *calls)
		}
	})
}

func TestPythonVolumeSummaryPreflightAndPartialFailures(t *testing.T) {
	t.Run("nil receiver and context", func(t *testing.T) {
		var api *volumes.API
		for _, ctx := range []context.Context{nil, context.Background()} {
			count := 0
			for value, err := range api.ListSummary(ctx) {
				count++
				if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(value, err)
				}
			}
			if count != 1 {
				t.Fatal(count)
			}
		}
	})
	t.Run("negative cap before HTTP", func(t *testing.T) {
		api, calls := pythonVolumeAPI(t, func(*http.Request) *http.Response {
			t.Error("unexpected HTTP")
			return pythonVolumeWire(200, `{"volumes":[]}`)
		})
		for _, err := range api.ListSummaryWithControl(context.Background(), resource.ListControl{MaxItems: -1}) {
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		}
		if len(*calls) != 0 {
			t.Fatal(*calls)
		}
	})
	t.Run("option failure retains callback and cancellation cause", func(t *testing.T) {
		api, calls := pythonVolumeAPI(t, func(*http.Request) *http.Response {
			t.Error("unexpected HTTP")
			return pythonVolumeWire(200, `{"volumes":[]}`)
		})
		callbackFailure := errors.New("callback failure")
		cause := errors.New("selected cancel cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		option := func(*request.Config[volumes.ListOpts]) error { cancel(cause); return callbackFailure }
		for _, err := range api.ListSummary(ctx, option) {
			if !errors.Is(err, callbackFailure) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		}
		if len(*calls) != 0 {
			t.Fatal(*calls)
		}
	})
	t.Run("later page HTTP error follows earlier rows", func(t *testing.T) {
		api, calls := pythonVolumeAPI(t, func(req *http.Request) *http.Response {
			if req.URL.Query().Get("marker") != "" {
				return pythonVolumeWire(403, `{"forbidden":{"message":"second page"}}`)
			}
			return pythonVolumeWire(200, `{"volumes":[{"id":"vol-1"}],"volumes_links":[{"rel":"next","href":"http://`+req.Host+req.URL.Path+`?marker=vol-1"}]}`)
		})
		var ids []string
		var failure error
		for value, err := range api.ListSummary(context.Background()) {
			if err != nil {
				failure = err
			} else {
				ids = append(ids, value.ID)
			}
		}
		var native gophercloud.ErrUnexpectedResponseCode
		var operation *resource.OperationError
		if !reflect.DeepEqual(ids, []string{"vol-1"}) || len(*calls) != 2 || !errors.As(failure, &native) || native.Actual != 403 || !errors.As(failure, &operation) || operation.Operation != "ListSummary" || operation.Resource != "volumes" {
			t.Fatal(ids, failure, native, operation, *calls)
		}
	})
	t.Run("cancel after response retains context cause", func(t *testing.T) {
		cause := errors.New("cancel during summary HTTP")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		api, calls := pythonVolumeAPI(t, func(*http.Request) *http.Response {
			cancel(cause)
			return pythonVolumeWire(200, `{"volumes":[{"id":"vol-1"}]}`)
		})
		rows := 0
		var failure error
		for _, err := range api.ListSummary(ctx) {
			if err != nil {
				failure = err
			} else {
				rows++
			}
		}
		if rows != 0 || len(*calls) != 1 || !errors.Is(failure, cause) || !errors.Is(failure, context.Canceled) {
			t.Fatal(rows, failure, *calls)
		}
	})
}
