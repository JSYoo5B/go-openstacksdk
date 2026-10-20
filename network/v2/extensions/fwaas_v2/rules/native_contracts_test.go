package rules_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules"
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

func nativeFWAPI(t *testing.T, calls *[]nativeFWCall, reply func(*http.Request) *http.Response) (*rules.API, *testcloud.Cloud) {
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
	return rules.New(client), cloud
}

func nativeFWOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "rules" {
		t.Fatal("generated rules context", err, wrapped)
	}
}

func nativeFWStatuses(t *testing.T, name string, accepted []int, envelope string, call func(*rules.API) error) {
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

func nativeFWListStatuses(t *testing.T, collection string, list func(*rules.API) []error) {
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

const nativeRuleRow = `{"id":"fr-1","name":"ssh","protocol":"tcp","action":"allow","ip_version":4,"destination_port":"22","enabled":true,"firewall_policy_id":["fp-1"]}`

func TestNativeFirewallRuleRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeFWCall
	var cloud *testcloud.Cloud
	api, cloud := nativeFWAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeFWWire(204, "")
		case req.Method == http.MethodPost:
			return nativeFWWire(201, `{"firewall_rule":`+nativeRuleRow+`}`)
		case req.URL.Path == "/neutron/v2.0/fwaas/firewall_rules":
			return nativeFWWire(200, `{"firewall_rules":[`+nativeRuleRow+`],"firewall_rules_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/rules?marker=x"}]}`)
		case req.URL.Path == "/other/rules":
			return nativeFWWire(200, `{"firewall_rules":[{"id":"fr-2","protocol":null,"firewall_policy_id":null}]}`)
		}
		return nativeFWWire(200, `{"firewall_rule":`+nativeRuleRow+`}`)
	})
	created, err := api.Create(ctx, rules.CreateOpts{Protocol: rules.ProtocolTCP, Action: rules.ActionAllow, IPVersion: gophercloud.IPv4, DestinationPort: "22", Enabled: gophercloud.Disabled}, rules.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "fr-1" && created.Protocol == "tcp" && created.IPVersion == 4 && reflect.DeepEqual(created.FirewallPolicyID, []string{"fp-1"})) {
		t.Fatal(created, err)
	}
	// The "any" protocol is sent as null.
	if _, err := api.Create(ctx, rules.CreateOpts{Protocol: rules.ProtocolAny, Action: rules.ActionDeny}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "fr-1")
	if err != nil || got.DestinationPort != "22" || !got.Enabled {
		t.Fatal(got, err)
	}
	protocol, empty := rules.ProtocolUDP, ""
	if _, err := api.Update(ctx, "fr-1", rules.UpdateOpts{Protocol: &protocol, SourcePort: &empty, Enabled: gophercloud.Enabled}, rules.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var rows []*rules.Rule
	for value, err := range api.List(ctx, rules.WithListOptions(rules.ListOpts{Protocol: rules.ProtocolTCP, Action: rules.ActionAllow, IPVersion: 4, Enabled: gophercloud.Enabled, Limit: 1}), rules.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "fr-1" && rows[1].ID == "fr-2" && rows[1].Protocol == "" && rows[1].FirewallPolicyID == nil) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "fr-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 7 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[4].query)
	base := "/neutron/v2.0/fwaas/firewall_rules"
	want := []nativeFWCall{
		{http.MethodPost, base, "", `{"firewall_rule":{"action":"allow","destination_port":"22","enabled":false,"ip_version":4,"protocol":"tcp","x_extension":1}}`},
		{http.MethodPost, base, "", `{"firewall_rule":{"action":"deny","protocol":null}}`},
		{http.MethodGet, base + "/fr-1", "", ""},
		{http.MethodPut, base + "/fr-1", "", `{"firewall_rule":{"enabled":true,"protocol":"udp","source_port":"","x_extension":1}}`},
		{http.MethodGet, base, calls[4].query, ""},
		{http.MethodGet, "/other/rules", "marker=x", ""},
		{http.MethodDelete, base + "/fr-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"protocol": {"tcp"}, "action": {"allow"}, "ip_version": {"4"}, "enabled": {"true"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeFirewallRuleStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	valid := rules.CreateOpts{Protocol: rules.ProtocolTCP, Action: rules.ActionAllow}
	nativeFWStatuses(t, "Create", []int{201, 202}, "firewall_rule", func(api *rules.API) error {
		_, err := api.Create(ctx, valid)
		return err
	})
	nativeFWStatuses(t, "Get", []int{200}, "firewall_rule", func(api *rules.API) error { _, err := api.Get(ctx, "fr-1"); return err })
	nativeFWStatuses(t, "Update", []int{200}, "firewall_rule", func(api *rules.API) error {
		_, err := api.Update(ctx, "fr-1", rules.UpdateOpts{})
		return err
	})
	nativeFWStatuses(t, "Delete", []int{202, 204}, "firewall_rule", func(api *rules.API) error { return api.Delete(ctx, "fr-1") })
	nativeFWListStatuses(t, "firewall_rules", func(api *rules.API) (errs []error) {
		for _, err := range api.List(ctx) {
			errs = append(errs, err)
		}
		return errs
	})
	t.Run("policy ID decodes as a list only", func(t *testing.T) {
		var calls []nativeFWCall
		api, _ := nativeFWAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeFWWire(200, `{"firewall_rule":{"id":"fr-1","firewall_policy_id":"fp-1"}}`)
		})
		_, err := api.Get(ctx, "fr-1")
		nativeFWOperation(t, err, "Get")
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeFWCall
		api, _ := nativeFWAPI(t, &calls, func(*http.Request) *http.Response { return nativeFWWire(201, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"missing protocol": {"Create", func() error { _, err := api.Create(ctx, rules.CreateOpts{Action: rules.ActionAllow}); return err }()},
			"missing action":   {"Create", func() error { _, err := api.Create(ctx, rules.CreateOpts{Protocol: rules.ProtocolTCP}); return err }()},
			"protocol extension": {"Create", func() error {
				_, err := api.Create(ctx, valid, rules.WithCreateField("protocol", "icmp"))
				return err
			}()},
			"update nil option": {"Update", func() error { _, err := api.Update(ctx, "fr-1", rules.UpdateOpts{}, nil); return err }()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeFWOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeFWOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
