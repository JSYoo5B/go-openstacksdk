package extensions_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonParityCoreExtensionAPI(t *testing.T, calls *[]string, reply func(*http.Request) (int, string)) *extensions.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return extensions.New(client)
}

// Python extensions() sends GET extensions and yields alias, name,
// description, links and updated. find_extension has no Go counterpart: Get
// returns the 404 for a name without the Python list fallback.
func TestPythonParityCoreExtensionListAndGetWithoutFallback(t *testing.T) {
	ctx := context.Background()
	var calls []string
	api := pythonParityCoreExtensionAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == "/neutron/v2.0/extensions" {
			return 200, `{"extensions":[{"alias":"segment","name":"Segment","description":"Segments extension.","updated":"2016-02-24T17:00:00-00:00","links":[]}]}`
		}
		return 404, `{"NeutronError":{"type":"ExtensionNotFound"}}`
	})
	var rows []*extensions.Extension
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if len(rows) != 1 || rows[0].Alias != "segment" || rows[0].Name != "Segment" || rows[0].Description != "Segments extension." || rows[0].Updated != "2016-02-24T17:00:00-00:00" {
		t.Fatalf("%+v", rows)
	}
	if _, err := api.Get(ctx, "Segment"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []string{"GET /neutron/v2.0/extensions?", "GET /neutron/v2.0/extensions/Segment?"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%v", calls)
	}
}
