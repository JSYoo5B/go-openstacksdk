package apiversions_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/apiversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeVersionTransport func(*http.Request) (*http.Response, error)

func (transport nativeVersionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeVersionWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeVersionOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "apiversions" {
		t.Fatal("generated apiversions context", err, wrapped)
	}
}

type nativeVersionCall struct{ method, path, query string }

func nativeVersionRecorder(cloud *testcloud.Cloud, calls *[]nativeVersionCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativeVersionTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, nativeVersionCall{req.Method, req.URL.Path, req.URL.RawQuery})
		return reply(req), nil
	})
}

func nativeVersionStatus(t *testing.T, err error, operation string, code int, expected []int) {
	t.Helper()
	nativeVersionOperation(t, err, operation)
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, expected) {
		t.Fatal(err, native)
	}
}

func nativeVersionPagerStatus(t *testing.T, errs []error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if len(errs) != 1 || !errors.As(errs[0], &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200, 204, 300}) {
		t.Fatal(errs)
	}
}

func TestNativeComputeAPIVersionsUseTheUnversionedRoot(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeVersionCall
	nativeVersionRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path == "/nova/" {
			return nativeVersionWire(200, `{"versions":[{"id":"v2.0","status":"SUPPORTED","version":"","min_version":"","updated":"2011-01-21T11:33:21Z"},{"id":"v2.1","status":"CURRENT","version":"2.100","min_version":"2.1","updated":"2013-07-23T11:33:21Z"}]}`)
		}
		return nativeVersionWire(200, `{"version":{"id":"v2.1","status":"CURRENT","version":"2.100","min_version":"2.1","updated":"2013-07-23T11:33:21Z"}}`)
	})
	// The project-scoped versioned endpoint is cut at its version segment.
	api := apiversions.New(cloud.Client("compute", "/nova/v2.1/project?x=1"))
	ctx := context.Background()
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+"="+value.Status)
	}
	if !reflect.DeepEqual(ids, []string{"v2.0=SUPPORTED", "v2.1=CURRENT"}) {
		t.Fatal(ids)
	}
	got, err := api.Get(ctx, "v2.1/")
	if err != nil || got.Version != "2.100" || got.MinVersion != "2.1" || !got.Updated.Equal(time.Date(2013, 7, 23, 11, 33, 21, 0, time.UTC)) {
		t.Fatal(got, err)
	}
	want := []nativeVersionCall{
		{http.MethodGet, "/nova/", ""},
		// A trailing slash in the version argument is trimmed before one is added back.
		{http.MethodGet, "/nova/v2.1/", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeComputeAPIVersionsStatusesAndMissingVersion(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{201, 204, 300, 404} {
		cloud := testcloud.New(t)
		var calls []nativeVersionCall
		nativeVersionRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeVersionWire(code, `{"version":{}}`) })
		_, err := apiversions.New(cloud.Client("compute", "/nova/v2.1/")).Get(ctx, "v2.1")
		nativeVersionStatus(t, err, "Get", code, []int{200})
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
	t.Run("missing version object", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls []nativeVersionCall
		nativeVersionRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeVersionWire(200, `{"versions":[]}`) })
		_, err := apiversions.New(cloud.Client("compute", "/nova/v2.1/")).Get(ctx, "v9")
		nativeVersionOperation(t, err, "Get")
		var missing apiversions.ErrVersionNotFound
		if !errors.As(err, &missing) {
			t.Fatal(err)
		}
	})
	t.Run("list pager status and empty page", func(t *testing.T) {
		for _, code := range []int{404, 200} {
			cloud := testcloud.New(t)
			var calls []nativeVersionCall
			nativeVersionRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeVersionWire(code, `{"versions":[]}`) })
			var errs []error
			for _, err := range apiversions.New(cloud.Client("compute", "/nova/v2.1/")).List(ctx) {
				errs = append(errs, err)
			}
			if code == 404 {
				nativeVersionPagerStatus(t, errs, 404)
			} else if len(errs) != 0 {
				t.Fatal(errs)
			}
		}
	})
}
