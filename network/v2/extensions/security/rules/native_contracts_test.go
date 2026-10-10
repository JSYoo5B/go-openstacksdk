package rules_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRuleTransport func(*http.Request) (*http.Response, error)

func (transport nativeRuleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeRuleWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeRuleClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client
}

func nativeRuleOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "rules" {
		t.Fatal("generated rules context", err, wrapped)
	}
}

type nativeRuleCall struct{ method, path, query, body string }

func nativeRuleRecorder(cloud *testcloud.Cloud, calls *[]nativeRuleCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativeRuleTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeRuleCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
}

const nativeRuleRow = `{"id":"id-1","direction":"ingress","ethertype":"IPv4","security_group_id":"sg","protocol":"tcp","port_range_min":22,"port_range_max":22,"remote_ip_prefix":"0.0.0.0/0","remote_group_id":null}`

func TestNativeRuleRoutesBodiesPagingAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeRuleCall
	nativeRuleRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeRuleWire(204, "")
		case req.Method == http.MethodPost:
			return nativeRuleWire(201, `{"security_group_rule":`+nativeRuleRow+`}`)
		case req.URL.Path == "/neutron/v2.0/security-group-rules" && req.URL.Query().Get("marker") == "":
			return nativeRuleWire(200, `{"security_group_rules":[`+nativeRuleRow+`],"security_group_rules_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/security-group-rules?marker=x"}]}`)
		case req.URL.Path == "/other/security-group-rules":
			return nativeRuleWire(200, `{"security_group_rules":[{"id":"id-2","protocol":null,"port_range_min":null}]}`)
		}
		return nativeRuleWire(200, `{"security_group_rule":`+nativeRuleRow+`}`)
	})
	api := rules.New(nativeRuleClient(cloud))
	ctx := context.Background()
	created, err := api.Create(ctx, rules.CreateOpts{Direction: rules.DirIngress, EtherType: rules.EtherType4, SecGroupID: "sg", Protocol: rules.ProtocolTCP, PortRangeMin: 22, PortRangeMax: 22, RemoteIPPrefix: "0.0.0.0/0"}, rules.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "id-1" && created.PortRangeMin == 22 && created.RemoteGroupID == "") {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "id-1")
	if err != nil || !(got.SecGroupID == "sg" && got.Protocol == "tcp") {
		t.Fatal(got, err)
	}
	var rows []*rules.SecGroupRule
	for value, err := range api.List(ctx, rules.WithListOptions(rules.ListOpts{SecGroupID: "sg", Direction: "ingress", PortRangeMin: 22, Limit: 1}), rules.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "id-1" && rows[1].ID == "id-2" && rows[1].Protocol == "" && rows[1].PortRangeMin == 0) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "id-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 5 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[2].query)
	want := []nativeRuleCall{
		{http.MethodPost, "/neutron/v2.0/security-group-rules", "", `{"security_group_rule":{"direction":"ingress","ethertype":"IPv4","port_range_max":22,"port_range_min":22,"protocol":"tcp","remote_ip_prefix":"0.0.0.0/0","security_group_id":"sg","x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/security-group-rules/id-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/security-group-rules", calls[2].query, ""},
		// The next href is followed as received.
		{http.MethodGet, "/other/security-group-rules", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/security-group-rules/id-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"security_group_id": {"sg"}, "direction": {"ingress"}, "port_range_min": {"22"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeRuleStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*rules.API) error
	}{
		{"Create", []int{201, 202}, func(api *rules.API) error {
			_, err := api.Create(ctx, rules.CreateOpts{Direction: rules.DirIngress, EtherType: rules.EtherType4, SecGroupID: "sg"})
			return err
		}},
		{"Get", []int{200}, func(api *rules.API) error { _, err := api.Get(ctx, "id-1"); return err }},
		{"Delete", []int{202, 204}, func(api *rules.API) error { return api.Delete(ctx, "id-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls []nativeRuleCall
				nativeRuleRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeRuleWire(code, `{"security_group_rule":{}}`) })
				err := call.call(rules.New(nativeRuleClient(cloud)))
				nativeRuleOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"security_group_rules":[]}`}, {204, ""}} {
			cloud := testcloud.New(t)
			var calls []nativeRuleCall
			nativeRuleRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeRuleWire(tc.code, tc.body) })
			var errs []error
			for _, err := range rules.New(nativeRuleClient(cloud)).List(ctx) {
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
		cloud := testcloud.New(t)
		var calls []nativeRuleCall
		nativeRuleRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeRuleWire(201, `{}`) })
		api := rules.New(nativeRuleClient(cloud))
		for name, check := range map[string]func() error{
			"missing direction": func() error {
				_, err := api.Create(ctx, rules.CreateOpts{EtherType: rules.EtherType4, SecGroupID: "sg"})
				return err
			},
			"missing ethertype": func() error {
				_, err := api.Create(ctx, rules.CreateOpts{Direction: rules.DirIngress, SecGroupID: "sg"})
				return err
			},
			"missing group": func() error {
				_, err := api.Create(ctx, rules.CreateOpts{Direction: rules.DirIngress, EtherType: rules.EtherType4})
				return err
			},
			"direction extension": func() error {
				_, err := api.Create(ctx, rules.CreateOpts{Direction: rules.DirIngress, EtherType: rules.EtherType4, SecGroupID: "sg"}, rules.WithCreateField("direction", "egress"))
				return err
			},
			"nil option": func() error {
				_, err := api.Create(ctx, rules.CreateOpts{Direction: rules.DirIngress, EtherType: rules.EtherType4, SecGroupID: "sg"}, nil)
				return err
			},
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeRuleOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativeRuleOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}

func contains(values []int, value int) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func TestNativeSecGroupRuleCreateBulk(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls []nativeRuleCall
	nativeRuleRecorder(cloud, &calls, func(*http.Request) *http.Response {
		return nativeRuleWire(201, `{"security_group_rules":[{"id":"r1","direction":"ingress"},{"id":"r2","protocol":"icmp"}]}`)
	})
	api := rules.New(nativeRuleClient(cloud))
	got, err := api.CreateBulk(ctx, []rules.CreateOpts{
		{Direction: rules.DirIngress, EtherType: rules.EtherType4, SecGroupID: "sg", Protocol: rules.ProtocolTCP, PortRangeMin: 22, PortRangeMax: 22},
		{Direction: rules.DirEgress, EtherType: rules.EtherType6, SecGroupID: "sg", Protocol: rules.ProtocolICMP},
	})
	if err != nil || len(got) != 2 || got[0].Direction != "ingress" || got[1].Protocol != "icmp" {
		t.Fatal(got, err)
	}
	want := []nativeRuleCall{{http.MethodPost, "/neutron/v2.0/security-group-rules", "", `{"security_group_rules":[{"direction":"ingress","ethertype":"IPv4","port_range_max":22,"port_range_min":22,"protocol":"tcp","security_group_id":"sg"},{"direction":"egress","ethertype":"IPv6","protocol":"icmp","security_group_id":"sg"}]}`}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	// Any element missing a required field fails before HTTP.
	if _, err := api.CreateBulk(ctx, []rules.CreateOpts{{Direction: rules.DirIngress, EtherType: rules.EtherType4, SecGroupID: "sg"}, {Direction: rules.DirIngress}}); err == nil || len(calls) != 1 {
		t.Fatal(err, calls)
	}
	nativeRuleOperation(t, func() error { _, err := api.CreateBulk(ctx, []rules.CreateOpts{{}}); return err }(), "CreateBulk")
	for _, code := range []int{200, 204, 409} {
		cloud := testcloud.New(t)
		var calls []nativeRuleCall
		nativeRuleRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeRuleWire(code, `{"security_group_rules":[]}`) })
		_, err := rules.New(nativeRuleClient(cloud)).CreateBulk(ctx, []rules.CreateOpts{{Direction: rules.DirIngress, EtherType: rules.EtherType4, SecGroupID: "sg"}})
		nativeRuleOperation(t, err, "CreateBulk")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{201, 202}) || len(calls) != 1 {
			t.Fatal(err)
		}
	}
}
