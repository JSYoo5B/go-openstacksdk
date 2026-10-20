package extensions_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonExtensionCall records the wire request openstacksdk would also send.
type pythonExtensionCall struct{ method, path, query, version string }

type pythonExtensionTransport func(*http.Request) (*http.Response, error)

func (transport pythonExtensionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonExtensionWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

const pythonExtensionRow = `{"alias":"os-hypervisors","name":"Hypervisors","namespace":"http://docs.openstack.org/compute/ext/fake_xml","updated":"2014-12-03T00:00:00Z","description":"Admin-only hypervisor administration.","links":[]}`

func TestPythonExtensionListAndGetHaveNoFindFallback(t *testing.T) {
	ctx := context.Background()
	var calls []pythonExtensionCall
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonExtensionTransport(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, pythonExtensionCall{req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("X-OpenStack-Nova-API-Version")})
		switch req.URL.Path {
		case "/nova/v2.1/extensions":
			return pythonExtensionWire(200, `{"extensions":[`+pythonExtensionRow+`]}`), nil
		case "/nova/v2.1/extensions/os-hypervisors":
			return pythonExtensionWire(200, `{"extension":`+pythonExtensionRow+`}`), nil
		}
		return pythonExtensionWire(404, `{"itemNotFound":{"message":"missing"}}`), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.1"
	api := extensions.New(client)

	// extensions()
	var aliases []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		if value.Name != "Hypervisors" || value.Namespace == "" || value.Updated != "2014-12-03T00:00:00Z" || value.Description == "" {
			t.Fatal(value)
		}
		aliases = append(aliases, value.Alias)
	}
	if !reflect.DeepEqual(aliases, []string{"os-hypervisors"}) {
		t.Fatal(aliases)
	}
	// find_extension("os-hypervisors") succeeds on its first GET by alias.
	if got, err := api.Get(ctx, "os-hypervisors"); err != nil || got.Alias != "os-hypervisors" {
		t.Fatal(got, err)
	}
	// find_extension("Hypervisors") would list after this 404 and match the
	// name; Go Get returns the 404 and has no fallback.
	if _, err := api.Get(ctx, "Hypervisors"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonExtensionCall{
		{http.MethodGet, "/nova/v2.1/extensions", "", "2.1"},
		{http.MethodGet, "/nova/v2.1/extensions/os-hypervisors", "", "2.1"},
		{http.MethodGet, "/nova/v2.1/extensions/Hypervisors", "", "2.1"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
