package tags_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/tags"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTagTransport func(*http.Request) (*http.Response, error)

func (transport nativeTagTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeTagWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeTagClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return client
}

func nativeTagOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "tags" {
		t.Fatal("generated tag context", err, wrapped)
	}
}

type nativeTagCall struct{ method, path, body string }

func TestNativeTagsRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeTagCall
	cloud.Provider.HTTPClient.Transport = nativeTagTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		calls = append(calls, nativeTagCall{req.Method, req.URL.Path, raw})
		switch {
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/tags"):
			return nativeTagWire(200, `{"tags":["a","b"]}`), nil
		case req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/tags"):
			return nativeTagWire(200, `{"tags":["server-order","c"]}`), nil
		case req.Method == http.MethodPut:
			return nativeTagWire(201, ""), nil
		}
		return nativeTagWire(204, ""), nil
	})
	api := tags.New(nativeTagClient(cloud))
	ctx := context.Background()
	if err := api.Add(ctx, "s1", "managed"); err != nil {
		t.Fatal(err)
	}
	if present, err := api.Check(ctx, "s1", "managed"); err != nil || !present {
		t.Fatal(present, err)
	}
	if values, err := api.List(ctx, "s1"); err != nil || !reflect.DeepEqual(values, []string{"a", "b"}) {
		t.Fatal(values, err)
	}
	// The response is returned as Nova ordered it, not as requested.
	if values, err := api.ReplaceAll(ctx, "s1", tags.ReplaceAllOpts{Tags: []string{"c", "server-order"}}, tags.WithReplaceAllField("x_extension", 1)); err != nil || !reflect.DeepEqual(values, []string{"server-order", "c"}) {
		t.Fatal(values, err)
	}
	if _, err := api.ReplaceAll(ctx, "s1", tags.ReplaceAllOpts{Tags: []string{}}); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "s1", "a/b"); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteAll(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	want := []nativeTagCall{
		{http.MethodPut, "/nova/v2.1/servers/s1/tags/managed", ""},
		{http.MethodGet, "/nova/v2.1/servers/s1/tags/managed", ""},
		{http.MethodGet, "/nova/v2.1/servers/s1/tags", ""},
		// The tags array is not an object envelope, so the extension sits beside it.
		{http.MethodPut, "/nova/v2.1/servers/s1/tags", `{"tags":["c","server-order"],"x_extension":1}`},
		{http.MethodPut, "/nova/v2.1/servers/s1/tags", `{"tags":[]}`},
		// The native API joins the tag without escaping.
		{http.MethodDelete, "/nova/v2.1/servers/s1/tags/a/b", ""},
		{http.MethodDelete, "/nova/v2.1/servers/s1/tags", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeTagsStrictStatusesCheckAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*tags.API) error
	}{
		{"Add", []int{201, 204}, func(api *tags.API) error { return api.Add(ctx, "s1", "t") }},
		{"Check", []int{204}, func(api *tags.API) error { _, err := api.Check(ctx, "s1", "t"); return err }},
		{"List", []int{200}, func(api *tags.API) error { _, err := api.List(ctx, "s1"); return err }},
		{"ReplaceAll", []int{200}, func(api *tags.API) error {
			_, err := api.ReplaceAll(ctx, "s1", tags.ReplaceAllOpts{Tags: []string{"t"}})
			return err
		}},
		{"Delete", []int{204}, func(api *tags.API) error { return api.Delete(ctx, "s1", "t") }},
		{"DeleteAll", []int{204}, func(api *tags.API) error { return api.DeleteAll(ctx, "s1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 403} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeTagTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeTagWire(code, `{"tags":[]}`), nil
				})
				err := call.call(tags.New(nativeTagClient(cloud)))
				nativeTagOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("Check reports 404 as absent and other errors as not present", func(t *testing.T) {
		for _, code := range []int{404, 500} {
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = nativeTagTransport(func(req *http.Request) (*http.Response, error) {
				return nativeTagWire(code, `{}`), nil
			})
			present, err := tags.New(nativeTagClient(cloud)).Check(ctx, "s1", "t")
			if present || (code == 404) != (err == nil) {
				t.Fatal(code, present, err)
			}
		}
	})
	t.Run("ReplaceAll nil tags, collisions and nil options", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeTagTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeTagWire(200, `{"tags":[]}`), nil
		})
		api := tags.New(nativeTagClient(cloud))
		for name, opts := range map[string][]tags.ReplaceAllOption{
			"nil tags":      nil,
			"tags override": {tags.WithReplaceAllField("tags", []string{"x"})},
			"nil option":    {nil},
		} {
			input := tags.ReplaceAllOpts{Tags: []string{"t"}}
			if name == "nil tags" {
				input.Tags = nil
			}
			_, err := api.ReplaceAll(ctx, "s1", input, opts...)
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeTagOperation(t, err, "ReplaceAll")
		}
		if requests.Load() != 0 {
			t.Fatal(requests.Load())
		}
	})
}

func contains(values []int, value int) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
