package addressscopes_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeScopeTransport func(*http.Request) (*http.Response, error)

func (transport nativeScopeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeScopeWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeScopeCall struct{ method, path, query, body string }

func nativeScopeAPI(t *testing.T, calls *[]nativeScopeCall, reply func(*http.Request) *http.Response) (*addressscopes.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeScopeTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeScopeCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return addressscopes.New(client), cloud
}

func nativeScopeOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "addressscopes" {
		t.Fatal("generated addressscopes context", err, wrapped)
	}
}

const nativeScopeRow = `{"id":"as-1","name":"scope","tenant_id":"t","project_id":"p","ip_version":6,"shared":true,"x_unknown":1}`

func TestNativeAddressScopeRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeScopeCall
	var cloud *testcloud.Cloud
	api, cloud := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeScopeWire(204, "")
		case req.Method == http.MethodPost:
			return nativeScopeWire(201, `{"address_scope":`+nativeScopeRow+`}`)
		case req.URL.Path == "/neutron/v2.0/address-scopes":
			return nativeScopeWire(200, `{"address_scopes":[`+nativeScopeRow+`],"address_scopes_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/address-scopes?marker=x"}]}`)
		case req.URL.Path == "/other/address-scopes":
			return nativeScopeWire(200, `{"address_scopes":[{"id":"as-2","name":null,"shared":null}]}`)
		}
		return nativeScopeWire(200, `{"address_scope":`+nativeScopeRow+`}`)
	})
	created, err := api.Create(ctx, addressscopes.CreateOpts{Name: "scope", IPVersion: 6, Shared: true, ProjectID: "p"}, addressscopes.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "as-1" && created.IPVersion == 6 && created.Shared && created.TenantID == "t") {
		t.Fatal(created, err)
	}
	// Name and ip_version have no omitempty; zero values are sent as given.
	if _, err := api.Create(ctx, addressscopes.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "as-1")
	if err != nil || got.Name != "scope" {
		t.Fatal(got, err)
	}
	name, shared := "renamed", false
	if _, err := api.Update(ctx, "as-1", addressscopes.UpdateOpts{Name: &name, Shared: &shared}, addressscopes.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "as-1", addressscopes.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	var rows []*addressscopes.AddressScope
	for value, err := range api.List(ctx, addressscopes.WithListOptions(addressscopes.ListOpts{Name: "scope", IPVersion: 6, Shared: gophercloud.Enabled, Limit: 1}), addressscopes.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "as-1" && rows[1].ID == "as-2" && rows[1].Name == "" && !rows[1].Shared) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "as-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 8 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[5].query)
	want := []nativeScopeCall{
		{http.MethodPost, "/neutron/v2.0/address-scopes", "", `{"address_scope":{"ip_version":6,"name":"scope","project_id":"p","shared":true,"x_extension":1}}`},
		{http.MethodPost, "/neutron/v2.0/address-scopes", "", `{"address_scope":{"ip_version":0,"name":""}}`},
		{http.MethodGet, "/neutron/v2.0/address-scopes/as-1", "", ""},
		// Pointer fields allow an explicit false.
		{http.MethodPut, "/neutron/v2.0/address-scopes/as-1", "", `{"address_scope":{"name":"renamed","shared":false,"x_extension":1}}`},
		{http.MethodPut, "/neutron/v2.0/address-scopes/as-1", "", `{"address_scope":{}}`},
		{http.MethodGet, "/neutron/v2.0/address-scopes", calls[5].query, ""},
		{http.MethodGet, "/other/address-scopes", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/address-scopes/as-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"scope"}, "ip_version": {"6"}, "shared": {"true"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeAddressScopeStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*addressscopes.API) error
	}{
		// Create narrows the default POST codes to 201.
		{"Create", []int{201}, func(api *addressscopes.API) error {
			_, err := api.Create(ctx, addressscopes.CreateOpts{})
			return err
		}},
		{"Get", []int{200}, func(api *addressscopes.API) error { _, err := api.Get(ctx, "as-1"); return err }},
		{"Update", []int{200}, func(api *addressscopes.API) error {
			_, err := api.Update(ctx, "as-1", addressscopes.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *addressscopes.API) error { return api.Delete(ctx, "as-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeScopeCall
				api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(code, `{"address_scope":{}}`) })
				err := call.call(api)
				nativeScopeOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("decode errors", func(t *testing.T) {
		for _, body := range []string{`{"address_scope":[]}`, `{`} {
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(200, body) })
			_, err := api.Get(ctx, "as-1")
			nativeScopeOperation(t, err, "Get")
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"address_scopes":[]}`}, {204, ""}} {
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
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
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(201, `{}`) })
		for operation, err := range map[string]error{
			"Create": func() error {
				_, err := api.Create(ctx, addressscopes.CreateOpts{}, addressscopes.WithCreateField("ip_version", 4))
				return err
			}(),
			"Update": func() error {
				_, err := api.Update(ctx, "as-1", addressscopes.UpdateOpts{}, nil)
				return err
			}(),
		} {
			if err == nil {
				t.Fatal(operation, "accepted")
			}
			nativeScopeOperation(t, err, operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeScopeOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
