package policies_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativePolicyTransport func(*http.Request) (*http.Response, error)

func (transport nativePolicyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativePolicyWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativePolicyCall struct{ method, path, query, body string }

func nativePolicyAPI(t *testing.T, calls *[]nativePolicyCall, reply func(*http.Request) *http.Response) (*policies.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativePolicyTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativePolicyCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return policies.New(client), cloud
}

func nativePolicyOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "policies" {
		t.Fatal("generated policies context", err, wrapped)
	}
}

const nativePolicyRow = `{"id":"qp-1","name":"gold","shared":true,"is_default":false,"revision_number":4,"rules":[{"type":"bandwidth_limit","max_kbps":1000}],"tags":["a"],"created_at":"2026-10-11T01:02:03Z"}`

func TestNativeQoSPolicyRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativePolicyCall
	var cloud *testcloud.Cloud
	api, cloud := nativePolicyAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativePolicyWire(204, "")
		case req.Method == http.MethodPost:
			return nativePolicyWire(201, `{"policy":`+nativePolicyRow+`}`)
		case req.URL.Path == "/neutron/v2.0/qos/policies":
			return nativePolicyWire(200, `{"policies":[`+nativePolicyRow+`],"policies_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/policies?marker=x"}]}`)
		case req.URL.Path == "/other/policies":
			return nativePolicyWire(200, `{"policies":[{"id":"qp-2","rules":null,"description":null}]}`)
		}
		return nativePolicyWire(200, `{"policy":`+nativePolicyRow+`}`)
	})
	created, err := api.Create(ctx, policies.CreateOpts{Name: "gold", Shared: true, Description: "d"}, policies.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "qp-1" && created.Shared && created.RevisionNumber == 4 && created.Rules[0]["max_kbps"] == float64(1000) && created.CreatedAt.Year() == 2026) {
		t.Fatal(created, err)
	}
	// Name has no omitempty; the bool flags cannot send an explicit false.
	if _, err := api.Create(ctx, policies.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "qp-1")
	if err != nil || got.Tags[0] != "a" {
		t.Fatal(got, err)
	}
	revision := 0
	var rows []*policies.Policy
	for value, err := range api.List(ctx, policies.WithListOptions(policies.ListOpts{Name: "gold", Shared: gophercloud.Disabled, RevisionNumber: &revision, Limit: 1}), policies.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "qp-1" && rows[1].ID == "qp-2" && rows[1].Rules == nil && rows[1].Description == "") {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "qp-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 6 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[3].query)
	want := []nativePolicyCall{
		{http.MethodPost, "/neutron/v2.0/qos/policies", "", `{"policy":{"description":"d","name":"gold","shared":true,"x_extension":1}}`},
		{http.MethodPost, "/neutron/v2.0/qos/policies", "", `{"policy":{"name":""}}`},
		{http.MethodGet, "/neutron/v2.0/qos/policies/qp-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/qos/policies", calls[3].query, ""},
		{http.MethodGet, "/other/policies", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/qos/policies/qp-1", "", ""},
	}
	// The pointer revision filter keeps an explicit zero.
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"gold"}, "shared": {"false"}, "revision_number": {"0"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeQoSPolicyStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*policies.API) error
	}{
		// Create narrows the default POST codes to 201.
		{"Create", []int{201}, func(api *policies.API) error { _, err := api.Create(ctx, policies.CreateOpts{}); return err }},
		{"Get", []int{200}, func(api *policies.API) error { _, err := api.Get(ctx, "qp-1"); return err }},
		{"Delete", []int{202, 204}, func(api *policies.API) error { return api.Delete(ctx, "qp-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativePolicyCall
				api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(code, `{"policy":{}}`) })
				err := call.call(api)
				nativePolicyOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("timestamps accept RFC3339 only", func(t *testing.T) {
		var calls []nativePolicyCall
		api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response {
			return nativePolicyWire(200, `{"policy":{"id":"qp-1","created_at":"2026-10-11T01:02:03"}}`)
		})
		_, err := api.Get(ctx, "qp-1")
		nativePolicyOperation(t, err, "Get")
	})
	t.Run("list pager status, empty page, non-array rows and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"policies":[]}`}, {200, `{"policies":{}}`}, {204, ""}} {
			var calls []nativePolicyCall
			api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.body == `{"policies":[]}` && len(errs) == 0:
			case tc.body == `{"policies":{}}` && len(errs) == 1:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, tc.body, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativePolicyCall
		api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(201, `{}`) })
		for name, check := range map[string]func() error{
			"name extension": func() error {
				_, err := api.Create(ctx, policies.CreateOpts{}, policies.WithCreateField("name", "x"))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, policies.CreateOpts{}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativePolicyOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativePolicyOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
