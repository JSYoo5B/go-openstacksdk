package extensions_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeExtensionTransport func(*http.Request) (*http.Response, error)

func (transport nativeExtensionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeExtensionWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeExtensionOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "extensions" {
		t.Fatal("generated extensions context", err, wrapped)
	}
}

type nativeExtensionCall struct{ method, path, query string }

func nativeExtensionRecorder(cloud *testcloud.Cloud, calls *[]nativeExtensionCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativeExtensionTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, nativeExtensionCall{req.Method, req.URL.Path, req.URL.RawQuery})
		return reply(req), nil
	})
}

func nativeExtensionStatus(t *testing.T, err error, operation string, code int, expected []int) {
	t.Helper()
	nativeExtensionOperation(t, err, operation)
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, expected) {
		t.Fatal(err, native)
	}
}

func nativeExtensionPagerStatus(t *testing.T, errs []error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if len(errs) != 1 || !errors.As(errs[0], &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200, 204, 300}) {
		t.Fatal(errs)
	}
}

func nativeExtensionClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return client
}

func TestNativeComputeExtensionsRoutesDecodeAndActionURL(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeExtensionCall
	nativeExtensionRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path == "/nova/v2.1/extensions" {
			return nativeExtensionWire(200, `{"extensions":[{"alias":"os-a","name":"A","namespace":"ns","description":"d","updated":"2014-12-03T00:00:00Z","links":[]},{"alias":"os-b"}],"extensions_links":[{"rel":"next","href":"http://other/next"}]}`)
		}
		return nativeExtensionWire(200, `{"extension":{"alias":"os-a","name":"A","updated":"2014-12-03T00:00:00Z"}}`)
	})
	api := extensions.New(nativeExtensionClient(cloud))
	ctx := context.Background()
	var aliases []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		aliases = append(aliases, value.Alias)
	}
	if !reflect.DeepEqual(aliases, []string{"os-a", "os-b"}) {
		t.Fatal(aliases)
	}
	got, err := api.Get(ctx, "os-a")
	if err != nil || got.Name != "A" || got.Updated != "2014-12-03T00:00:00Z" {
		t.Fatal(got, err)
	}
	// ActionURL only formats the server action URL; it sends nothing.
	if url := api.ActionURL(ctx, "s1"); url != cloud.Server.URL+"/nova/v2.1/servers/s1/action" {
		t.Fatal(url)
	}
	want := []nativeExtensionCall{
		{http.MethodGet, "/nova/v2.1/extensions", ""},
		{http.MethodGet, "/nova/v2.1/extensions/os-a", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeComputeExtensionsStatuses(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{201, 204, 404} {
		cloud := testcloud.New(t)
		var calls []nativeExtensionCall
		nativeExtensionRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeExtensionWire(code, `{"extension":{}}`) })
		_, err := extensions.New(nativeExtensionClient(cloud)).Get(ctx, "os-a")
		nativeExtensionStatus(t, err, "Get", code, []int{200})
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"extensions":[]}`}, {204, ""}} {
		cloud := testcloud.New(t)
		var calls []nativeExtensionCall
		nativeExtensionRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeExtensionWire(tc.code, tc.body) })
		var errs []error
		for _, err := range extensions.New(nativeExtensionClient(cloud)).List(ctx) {
			errs = append(errs, err)
		}
		switch tc.code {
		case 404:
			nativeExtensionPagerStatus(t, errs, 404)
		case 200:
			if len(errs) != 0 {
				t.Fatal(errs)
			}
		case 204:
			if len(errs) != 1 || !errors.Is(errs[0], io.EOF) {
				t.Fatal(errs)
			}
		}
	}
}
