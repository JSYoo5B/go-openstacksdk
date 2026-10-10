package extensions_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
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

func nativeExtensionAPI(t *testing.T, paths *[]string, code int, body func(*http.Request) string) *extensions.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeExtensionTransport(func(req *http.Request) (*http.Response, error) {
		*paths = append(*paths, req.Method+" "+req.URL.Path)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body(req))), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return extensions.New(client)
}

func TestNativeNetworkExtensionsRoutesAndDecode(t *testing.T) {
	ctx := context.Background()
	var paths []string
	api := nativeExtensionAPI(t, &paths, 200, func(req *http.Request) string {
		if req.URL.Path == "/neutron/v2.0/extensions" {
			// The listing is a single page; links are not followed.
			return `{"extensions":[{"alias":"router","name":"Neutron L3 Router","description":"d","updated":"2012-07-20T10:00:00-00:00","links":[]},{"alias":"qos"}],"extensions_links":[{"rel":"next","href":"/never"}]}`
		}
		return `{"extension":{"alias":"router","name":"Neutron L3 Router","updated":"2012-07-20T10:00:00-00:00"}}`
	})
	var aliases []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		aliases = append(aliases, value.Alias)
	}
	got, err := api.Get(ctx, "router")
	// Updated stays the raw string the server sent.
	if err != nil || got.Name != "Neutron L3 Router" || got.Updated != "2012-07-20T10:00:00-00:00" {
		t.Fatal(got, err)
	}
	want := []string{"GET /neutron/v2.0/extensions", "GET /neutron/v2.0/extensions/router"}
	if !reflect.DeepEqual(aliases, []string{"router", "qos"}) || !reflect.DeepEqual(paths, want) {
		t.Fatal(aliases, paths)
	}
}

func TestNativeNetworkExtensionsStatuses(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{201, 204, 404} {
		var paths []string
		_, err := nativeExtensionAPI(t, &paths, code, func(*http.Request) string { return `{"extension":{}}` }).Get(ctx, "router")
		var wrapped *resource.OperationError
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &wrapped) || wrapped.Operation != "Get" || wrapped.Resource != "extensions" || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(paths) != 1 {
			t.Fatal(code, err)
		}
	}
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"extensions":[]}`}, {204, ""}} {
		var paths []string
		var errs []error
		for _, err := range nativeExtensionAPI(t, &paths, tc.code, func(*http.Request) string { return tc.body }).List(ctx) {
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
	}
}
