package traits_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/traits"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonTraitTransport func(*http.Request) (*http.Response, error)

func (transport pythonTraitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonTraitWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type pythonTraitCall struct{ method, path, query, body, version string }

// Trait declares _max_microversion 1.6, so the helper pins that version.
func pythonTraitAPI(t *testing.T, calls *[]pythonTraitCall, reply func(*http.Request) *http.Response) *traits.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonTraitTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonTraitCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = "1.6"
	return traits.New(client)
}

func pythonTraitBool(v bool) *bool { return &v }

func TestPythonTraitProxyCalls(t *testing.T) {
	ctx := context.Background()
	var calls []pythonTraitCall
	api := pythonTraitAPI(t, &calls, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodPut:
			return pythonTraitWire(201, "")
		case http.MethodDelete:
			return pythonTraitWire(204, "")
		}
		if req.URL.Path == "/placement/traits" {
			return pythonTraitWire(200, `{"traits":["CUSTOM_A","CUSTOM_B"]}`)
		}
		return pythonTraitWire(204, "")
	})
	// create_trait("CUSTOM_A"); Python sends an empty JSON object, Go sends no body.
	if err := api.Create(ctx, "CUSTOM_A"); err != nil {
		t.Fatal(err)
	}
	// get_trait("CUSTOM_A") only proves existence; the 204 has no body.
	if err := api.Get(ctx, "CUSTOM_A"); err != nil {
		t.Fatal(err)
	}
	// traits(name="startswith:CUSTOM", associated=True)
	var names []string
	for name, err := range api.List(ctx, traits.WithListOptions(traits.ListOpts{Name: "startswith:CUSTOM", Associated: pythonTraitBool(true)})) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, *name)
	}
	if !reflect.DeepEqual(names, []string{"CUSTOM_A", "CUSTOM_B"}) {
		t.Fatal(names)
	}
	// traits() sends no query.
	for _, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
	}
	// delete_trait("CUSTOM_A")
	if err := api.Delete(ctx, "CUSTOM_A"); err != nil {
		t.Fatal(err)
	}
	want := []pythonTraitCall{
		{http.MethodPut, "/placement/traits/CUSTOM_A", "", "", "placement 1.6"},
		{http.MethodGet, "/placement/traits/CUSTOM_A", "", "", "placement 1.6"},
		{http.MethodGet, "/placement/traits", "associated=true&name=startswith%3ACUSTOM", "", "placement 1.6"},
		{http.MethodGet, "/placement/traits", "", "", "placement 1.6"},
		{http.MethodDelete, "/placement/traits/CUSTOM_A", "", "", "placement 1.6"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

// Python get_trait raises on 404 and delete_trait swallows it by default; Go reports both.
func TestPythonTraitMissing(t *testing.T) {
	ctx := context.Background()
	var calls []pythonTraitCall
	api := pythonTraitAPI(t, &calls, func(*http.Request) *http.Response { return pythonTraitWire(404, `{}`) })
	for name, err := range map[string]error{"get": api.Get(ctx, "CUSTOM_A"), "delete": api.Delete(ctx, "CUSTOM_A")} {
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 404 {
			t.Fatal(name, err)
		}
	}
}
