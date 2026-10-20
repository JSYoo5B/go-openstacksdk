package groups_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeFWTransport func(*http.Request) (*http.Response, error)

func (transport nativeFWTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeFWWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeFWCall struct{ method, path, query, body string }

func nativeFWAPI(t *testing.T, calls *[]nativeFWCall, reply func(*http.Request) *http.Response) (*groups.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeFWTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeFWCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return groups.New(client), cloud
}

func nativeFWOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "groups" {
		t.Fatal("generated groups context", err, wrapped)
	}
}

func nativeFWStatuses(t *testing.T, name string, accepted []int, envelope string, call func(*groups.API) error) {
	t.Helper()
	for _, code := range []int{200, 201, 202, 204, 404} {
		if slices.Contains(accepted, code) {
			continue
		}
		t.Run(fmt.Sprintf("%s/%d", name, code), func(t *testing.T) {
			var calls []nativeFWCall
			api, _ := nativeFWAPI(t, &calls, func(*http.Request) *http.Response { return nativeFWWire(code, `{"`+envelope+`":{}}`) })
			err := call(api)
			nativeFWOperation(t, err, name)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, accepted) || len(calls) != 1 {
				t.Fatal(err, native)
			}
		})
	}
}

func nativeFWListStatuses(t *testing.T, collection string, list func(*groups.API) []error) {
	t.Helper()
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"` + collection + `":[]}`}, {204, ""}} {
		var calls []nativeFWCall
		api, _ := nativeFWAPI(t, &calls, func(*http.Request) *http.Response { return nativeFWWire(tc.code, tc.body) })
		errs := list(api)
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

const nativeGroupRow = `{"id":"fg-1","name":"edge","ingress_firewall_policy_id":"in","egress_firewall_policy_id":null,"admin_state_up":true,"ports":["p1"],"status":"ACTIVE","shared":false}`

func TestNativeFirewallGroupRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeFWCall
	var cloud *testcloud.Cloud
	api, cloud := nativeFWAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeFWWire(204, "")
		case req.Method == http.MethodPost:
			return nativeFWWire(201, `{"firewall_group":`+nativeGroupRow+`}`)
		case req.URL.Path == "/neutron/v2.0/fwaas/firewall_groups":
			return nativeFWWire(200, `{"firewall_groups":[`+nativeGroupRow+`],"firewall_groups_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/groups?marker=x"}]}`)
		case req.URL.Path == "/other/groups":
			return nativeFWWire(200, `{"firewall_groups":[{"id":"fg-2","ports":null}]}`)
		}
		return nativeFWWire(200, `{"firewall_group":`+nativeGroupRow+`}`)
	})
	created, err := api.Create(ctx, groups.CreateOpts{Name: "edge", IngressFirewallPolicyID: "in", AdminStateUp: gophercloud.Disabled, Ports: []string{"p1"}, Shared: gophercloud.Disabled}, groups.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "fg-1" && created.EgressFirewallPolicyID == "" && created.Ports[0] == "p1" && created.Status == "ACTIVE") {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "fg-1")
	if err != nil || got.IngressFirewallPolicyID != "in" {
		t.Fatal(got, err)
	}
	empty, ports := "", []string{}
	if _, err := api.Update(ctx, "fg-1", groups.UpdateOpts{Description: &empty, Ports: &ports, AdminStateUp: gophercloud.Enabled}, groups.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := api.RemoveIngressPolicy(ctx, "fg-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := api.RemoveEgressPolicy(ctx, "fg-1"); err != nil {
		t.Fatal(err)
	}
	filter := []string{"p1", "p2"}
	var rows []*groups.Group
	for value, err := range api.List(ctx, groups.WithListOptions(groups.ListOpts{Name: "edge", Ports: &filter, AdminStateUp: gophercloud.Enabled, Limit: 1}), groups.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "fg-1" && rows[1].ID == "fg-2" && rows[1].Ports == nil) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "fg-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 8 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[5].query)
	base := "/neutron/v2.0/fwaas/firewall_groups"
	want := []nativeFWCall{
		{http.MethodPost, base, "", `{"firewall_group":{"admin_state_up":false,"ingress_firewall_policy_id":"in","name":"edge","ports":["p1"],"shared":false,"x_extension":1}}`},
		{http.MethodGet, base + "/fg-1", "", ""},
		// A pointer to an empty port list clears the ports.
		{http.MethodPut, base + "/fg-1", "", `{"firewall_group":{"admin_state_up":true,"description":"","ports":[],"x_extension":1}}`},
		// The policy removals send an explicit null on the plain group route.
		{http.MethodPut, base + "/fg-1", "", `{"firewall_group":{"ingress_firewall_policy_id":null}}`},
		{http.MethodPut, base + "/fg-1", "", `{"firewall_group":{"egress_firewall_policy_id":null}}`},
		{http.MethodGet, base, calls[5].query, ""},
		{http.MethodGet, "/other/groups", "marker=x", ""},
		{http.MethodDelete, base + "/fg-1", "", ""},
	}
	// The port filter repeats once per value.
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"edge"}, "ports": {"p1", "p2"}, "admin_state_up": {"true"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeFirewallGroupStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	nativeFWStatuses(t, "Create", []int{201, 202}, "firewall_group", func(api *groups.API) error {
		_, err := api.Create(ctx, groups.CreateOpts{})
		return err
	})
	nativeFWStatuses(t, "Get", []int{200}, "firewall_group", func(api *groups.API) error { _, err := api.Get(ctx, "fg-1"); return err })
	nativeFWStatuses(t, "Update", []int{200}, "firewall_group", func(api *groups.API) error {
		_, err := api.Update(ctx, "fg-1", groups.UpdateOpts{})
		return err
	})
	nativeFWStatuses(t, "RemoveIngressPolicy", []int{200}, "firewall_group", func(api *groups.API) error {
		_, err := api.RemoveIngressPolicy(ctx, "fg-1")
		return err
	})
	nativeFWStatuses(t, "RemoveEgressPolicy", []int{200}, "firewall_group", func(api *groups.API) error {
		_, err := api.RemoveEgressPolicy(ctx, "fg-1")
		return err
	})
	nativeFWStatuses(t, "Delete", []int{202, 204}, "firewall_group", func(api *groups.API) error { return api.Delete(ctx, "fg-1") })
	nativeFWListStatuses(t, "firewall_groups", func(api *groups.API) (errs []error) {
		for _, err := range api.List(ctx) {
			errs = append(errs, err)
		}
		return errs
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeFWCall
		api, _ := nativeFWAPI(t, &calls, func(*http.Request) *http.Response { return nativeFWWire(201, `{}`) })
		for operation, err := range map[string]error{
			"Create": func() error {
				_, err := api.Create(ctx, groups.CreateOpts{}, groups.WithCreateField("ports", nil))
				return err
			}(),
			"Update": func() error { _, err := api.Update(ctx, "fg-1", groups.UpdateOpts{}, nil); return err }(),
		} {
			if err == nil {
				t.Fatal(operation, "accepted")
			}
			nativeFWOperation(t, err, operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeFWOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
