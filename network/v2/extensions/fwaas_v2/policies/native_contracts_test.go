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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
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

func nativeFWAPI(t *testing.T, calls *[]nativeFWCall, reply func(*http.Request) *http.Response) (*policies.API, *testcloud.Cloud) {
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
	return policies.New(client), cloud
}

func nativeFWOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "policies" {
		t.Fatal("generated policies context", err, wrapped)
	}
}

func nativeFWStatuses(t *testing.T, name string, accepted []int, envelope string, call func(*policies.API) error) {
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

func nativeFWListStatuses(t *testing.T, collection string, list func(*policies.API) []error) {
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

const nativePolicyRow = `{"id":"fp-1","name":"web","audited":true,"shared":false,"firewall_rules":["r1","r2"]}`

func TestNativeFirewallPolicyRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeFWCall
	var cloud *testcloud.Cloud
	api, cloud := nativeFWAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeFWWire(204, "")
		case req.Method == http.MethodPost:
			return nativeFWWire(201, `{"firewall_policy":`+nativePolicyRow+`}`)
		case req.URL.Path == "/neutron/v2.0/fwaas/firewall_policies":
			return nativeFWWire(200, `{"firewall_policies":[`+nativePolicyRow+`],"firewall_policies_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/policies?marker=x"}]}`)
		case req.URL.Path == "/other/policies":
			return nativeFWWire(200, `{"firewall_policies":[{"id":"fp-2","firewall_rules":[]}]}`)
		}
		return nativeFWWire(200, `{"firewall_policy":`+nativePolicyRow+`}`)
	})
	created, err := api.Create(ctx, policies.CreateOpts{Name: "web", Audited: gophercloud.Enabled, Shared: gophercloud.Disabled, FirewallRules: []string{"r1", "r2"}}, policies.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "fp-1" && created.Audited && reflect.DeepEqual(created.Rules, []string{"r1", "r2"})) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "fp-1")
	if err != nil || got.Name != "web" {
		t.Fatal(got, err)
	}
	rules := []string{}
	if _, err := api.Update(ctx, "fp-1", policies.UpdateOpts{FirewallRules: &rules, Audited: gophercloud.Disabled}, policies.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var rows []*policies.Policy
	for value, err := range api.List(ctx, policies.WithListOptions(policies.ListOpts{Name: "web", Audited: gophercloud.Enabled, Limit: 1}), policies.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "fp-1" && rows[1].ID == "fp-2") {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "fp-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 6 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[3].query)
	base := "/neutron/v2.0/fwaas/firewall_policies"
	want := []nativeFWCall{
		{http.MethodPost, base, "", `{"firewall_policy":{"audited":true,"firewall_rules":["r1","r2"],"name":"web","shared":false,"x_extension":1}}`},
		{http.MethodGet, base + "/fp-1", "", ""},
		// A pointer to an empty rule list clears the rules.
		{http.MethodPut, base + "/fp-1", "", `{"firewall_policy":{"audited":false,"firewall_rules":[],"x_extension":1}}`},
		{http.MethodGet, base, calls[3].query, ""},
		{http.MethodGet, "/other/policies", "marker=x", ""},
		{http.MethodDelete, base + "/fp-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"web"}, "audited": {"true"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeFirewallPolicyRuleOperations(t *testing.T) {
	ctx := context.Background()
	var calls []nativeFWCall
	api, _ := nativeFWAPI(t, &calls, func(*http.Request) *http.Response {
		// Rule insertion and removal return the policy without an envelope.
		return nativeFWWire(200, nativePolicyRow)
	})
	inserted, err := api.InsertRule(ctx, "fp-1", policies.InsertRuleOpts{ID: "r3", InsertAfter: "r2"}, policies.WithInsertRuleField("x_extension", 1))
	if err != nil || inserted.ID != "fp-1" || len(inserted.Rules) != 2 {
		t.Fatal(inserted, err)
	}
	removed, err := api.RemoveRule(ctx, "fp-1", "r3")
	if err != nil || removed.ID != "fp-1" {
		t.Fatal(removed, err)
	}
	// RemoveRule sends the rule ID even when it is empty.
	if _, err := api.RemoveRule(ctx, "fp-1", ""); err != nil {
		t.Fatal(err)
	}
	base := "/neutron/v2.0/fwaas/firewall_policies/fp-1"
	want := []nativeFWCall{
		{http.MethodPut, base + "/insert_rule", "", `{"firewall_rule_id":"r3","insert_after":"r2","x_extension":1}`},
		{http.MethodPut, base + "/remove_rule", "", `{"firewall_rule_id":"r3"}`},
		{http.MethodPut, base + "/remove_rule", "", `{"firewall_rule_id":""}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	calls = nil
	for name, opts := range map[string]policies.InsertRuleOpts{
		"neither position": {ID: "r3"},
		"both positions":   {ID: "r3", InsertBefore: "r1", InsertAfter: "r2"},
		"missing rule":     {InsertAfter: "r2"},
	} {
		_, err := api.InsertRule(ctx, "fp-1", opts)
		if err == nil {
			t.Fatal(name, "accepted")
		}
		nativeFWOperation(t, err, "InsertRule")
	}
	_, err = api.InsertRule(ctx, "fp-1", policies.InsertRuleOpts{ID: "r3", InsertAfter: "r2"}, policies.WithInsertRuleField("firewall_rule_id", "x"))
	nativeFWOperation(t, err, "InsertRule")
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestNativeFirewallPolicyStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	insert := policies.InsertRuleOpts{ID: "r3", InsertBefore: "r1"}
	nativeFWStatuses(t, "Create", []int{201, 202}, "firewall_policy", func(api *policies.API) error {
		_, err := api.Create(ctx, policies.CreateOpts{})
		return err
	})
	nativeFWStatuses(t, "Get", []int{200}, "firewall_policy", func(api *policies.API) error { _, err := api.Get(ctx, "fp-1"); return err })
	nativeFWStatuses(t, "Update", []int{200}, "firewall_policy", func(api *policies.API) error {
		_, err := api.Update(ctx, "fp-1", policies.UpdateOpts{})
		return err
	})
	nativeFWStatuses(t, "Delete", []int{202, 204}, "firewall_policy", func(api *policies.API) error { return api.Delete(ctx, "fp-1") })
	nativeFWStatuses(t, "InsertRule", []int{200}, "firewall_policy", func(api *policies.API) error {
		_, err := api.InsertRule(ctx, "fp-1", insert)
		return err
	})
	nativeFWStatuses(t, "RemoveRule", []int{200}, "firewall_policy", func(api *policies.API) error {
		_, err := api.RemoveRule(ctx, "fp-1", "r3")
		return err
	})
	nativeFWListStatuses(t, "firewall_policies", func(api *policies.API) (errs []error) {
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
				_, err := api.Create(ctx, policies.CreateOpts{}, policies.WithCreateField("firewall_rules", nil))
				return err
			}(),
			"Update": func() error { _, err := api.Update(ctx, "fp-1", policies.UpdateOpts{}, nil); return err }(),
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
