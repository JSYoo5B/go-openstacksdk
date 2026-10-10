package extensions_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonExtensionTransport func(*http.Request) (*http.Response, error)

func (transport pythonExtensionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonExtensionCall struct{ method, path, query string }

func pythonExtensionAPI(t *testing.T, calls *[]pythonExtensionCall, reply func(*http.Request) (int, string)) *extensions.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonExtensionTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pythonExtensionCall{req.Method, req.URL.Path, req.URL.RawQuery})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v2.0/"
	return extensions.New(client)
}

const pythonExtensionRow = `{"alias":"OS-KSADM","name":"OpenStack Keystone Admin","namespace":"http://docs.openstack.org/identity/api/ext/OS-KSADM/v1.0","updated":"2013-07-11T17:14:00-00:00","description":"admin","links":[{"rel":"describedby","href":"https://example.invalid"}]}`

// Python extensions() reads extensions.values from one GET /extensions and
// get_extension() fetches /extensions/{alias} unwrapping "extension".
func TestPythonExtensionListAndGet(t *testing.T) {
	ctx := context.Background()
	var calls []pythonExtensionCall
	api := pythonExtensionAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Path {
		case "/keystone/v2.0/extensions":
			return 200, `{"extensions":{"values":[` + pythonExtensionRow + `,{"alias":"OS-EC2","name":"ec2"}]}}`
		case "/keystone/v2.0/extensions/OS-KSADM":
			return 200, `{"extension":` + pythonExtensionRow + `}`
		}
		return 404, `{"error":{"code":404}}`
	})
	var aliases []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		aliases = append(aliases, value.Alias)
	}
	got, err := api.Get(ctx, "OS-KSADM")
	if err != nil || got.Alias != "OS-KSADM" || got.Name != "OpenStack Keystone Admin" || got.Namespace != "http://docs.openstack.org/identity/api/ext/OS-KSADM/v1.0" || got.Updated != "2013-07-11T17:14:00-00:00" || got.Description != "admin" || len(got.Links) != 1 {
		t.Fatal(got, err)
	}
	// get_extension raises NotFoundException on 404.
	if _, err := api.Get(ctx, "missing"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonExtensionCall{
		{http.MethodGet, "/keystone/v2.0/extensions", ""},
		{http.MethodGet, "/keystone/v2.0/extensions/OS-KSADM", ""},
		{http.MethodGet, "/keystone/v2.0/extensions/missing", ""},
	}
	if !reflect.DeepEqual(aliases, []string{"OS-KSADM", "OS-EC2"}) || !reflect.DeepEqual(calls, want) {
		t.Fatalf("%v %+v", aliases, calls)
	}
}
