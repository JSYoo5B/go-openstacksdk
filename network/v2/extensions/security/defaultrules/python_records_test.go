package defaultrules_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonDefaultRuleTransport func(*http.Request) (*http.Response, error)

func (transport pythonDefaultRuleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonDefaultRuleCall struct {
	method, path, query, body string
	header                    http.Header
}

func pythonDefaultRuleAPI(t *testing.T, calls *[]pythonDefaultRuleCall, reply func(*http.Request) (int, string)) *defaultrules.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonDefaultRuleTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonDefaultRuleCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Clone()})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return defaultrules.New(client)
}

func pythonDefaultRuleField(t *testing.T, v *defaultrules.Record, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		var got json.RawMessage
		if v != nil && v.Resource != nil {
			got = v.Resource.Body[key]
		}
		t.Fatalf("%s: got %s want %s", key, got, want)
	}
}

func pythonDefaultRuleFailure(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "defaultrules" {
		t.Fatal("operation context", err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatal("native status", code, err)
	}
}

const pythonDefaultRuleBase = "/neutron/v2.0/default-security-group-rules"

func TestPythonDefaultRuleCreateMapsEtherTypeAndIfMatch(t *testing.T) {
	var calls []pythonDefaultRuleCall
	api := pythonDefaultRuleAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"default_security_group_rule":{"id":"d1","direction":"ingress","ethertype":"IPv6","port_range_min":"22","port_range_max":22,"used_in_default_sg":1,"used_in_non_default_sg":false,"revision_number":3,"remote_ip_prefix":null}}`
	})
	v, err := api.Create(context.Background(), defaultrules.WithCreateField("direction", "ingress"), defaultrules.WithCreateField("ether_type", "IPv6"),
		defaultrules.WithCreateField("protocol", "tcp"), defaultrules.WithCreateField("port_range_min", 22), defaultrules.WithCreateField("used_in_default_sg", true),
		defaultrules.WithCreateIfMatch("revision_number=1", "revision_number=2"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonDefaultRuleBase || calls[0].query != "" ||
		calls[0].header.Get("If-Match") != "revision_number=1, revision_number=2" || calls[0].header.Get("Content-Type") != "application/json" {
		t.Fatal(calls)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil || !reflect.DeepEqual(body, map[string]any{"default_security_group_rule": map[string]any{
		"direction": "ingress", "ethertype": "IPv6", "protocol": "tcp", "port_range_min": float64(22), "used_in_default_sg": true,
	}}) {
		t.Fatal(calls[0].body, err)
	}
	pythonDefaultRuleField(t, v, "ether_type", `"IPv6"`)
	pythonDefaultRuleField(t, v, "port_range_min", `22`)
	pythonDefaultRuleField(t, v, "used_in_default_sg", `true`)
	pythonDefaultRuleField(t, v, "used_in_non_default_sg", `false`)
	pythonDefaultRuleField(t, v, "revision_number", `3`)
	pythonDefaultRuleField(t, v, "protocol", `"tcp"`)
	pythonDefaultRuleField(t, v, "if_match", `["revision_number=1","revision_number=2"]`)
	pythonDefaultRuleField(t, v, "location", `null`)
	if v.Resource.Body["ethertype"] != nil || string(v.Wire.Body["ethertype"]) != `"IPv6"` {
		t.Fatal("wire name stays in Wire", v.Resource.Body)
	}

	calls = nil
	if _, err := api.Create(context.Background(), defaultrules.WithCreateField("if_match", 1)); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal("non-string If-Match fails before HTTP", err, calls)
	}
	api = pythonDefaultRuleAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{"NeutronError":{}}` })
	_, err = api.Create(context.Background(), defaultrules.WithCreateField("direction", "egress"))
	if calls[0].header.Get("If-Match") != "" {
		t.Fatal("no If-Match unless supplied", calls)
	}
	pythonDefaultRuleFailure(t, err, "Create", 409)
}

func TestPythonDefaultRuleGetAndDelete(t *testing.T) {
	var calls []pythonDefaultRuleCall
	api := pythonDefaultRuleAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"default_security_group_rule":{"ethertype":"IPv4"}}`
	})
	v, err := api.Get(context.Background(), "d1", defaultrules.WithGetFields("ether_type", "direction"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonDefaultRuleBase+"/d1" || calls[0].query != "fields=ethertype&fields=direction" {
		t.Fatal(err, calls)
	}
	pythonDefaultRuleField(t, v, "id", `"d1"`)
	pythonDefaultRuleField(t, v, "ether_type", `"IPv4"`)
	pythonDefaultRuleField(t, v, "if_match", `null`)

	api = pythonDefaultRuleAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{}` })
	_, err = api.Get(context.Background(), "d1")
	pythonDefaultRuleFailure(t, err, "Get", 404)
	calls = nil
	result, err := api.Delete(context.Background(), "d1")
	if err != nil || !result.IgnoredMissing || len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonDefaultRuleBase+"/d1" || calls[0].body != "" {
		t.Fatal(result, err, calls)
	}
	_, err = api.Delete(context.Background(), "d1", defaultrules.WithDeleteIgnoreMissing(false))
	pythonDefaultRuleFailure(t, err, "Delete", 404)
	api = pythonDefaultRuleAPI(t, &calls, func(*http.Request) (int, string) { return 204, `` })
	if result, err := api.Delete(context.Background(), "d1"); err != nil || result.StatusCode != 204 || result.IgnoredMissing {
		t.Fatal(result, err)
	}
}

func TestPythonDefaultRuleListMapsQueryAliases(t *testing.T) {
	var calls []pythonDefaultRuleCall
	api := pythonDefaultRuleAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"default_security_group_rules":[{"id":"a","ethertype":"IPv4","name":"x"},{"id":"b","ethertype":"IPv4"}],"default_security_group_rules_links":[{"rel":"next","href":"http://` + req.Host + `/neutron/v2.0/default-security-group-rules?ethertype=IPv4&used_in_default_sg=True&marker=b"}]}`
		}
		return 200, `{"default_security_group_rules":[{"id":"c","ethertype":"IPv4"}]}`
	})
	var ids []string
	for v, err := range api.List(context.Background(), defaultrules.WithListQuery("ether_type", "IPv4"), defaultrules.WithListQuery("used_in_default_sg", true),
		defaultrules.WithListQuery("name", nil), defaultrules.WithListQuery("unknown", 1)) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, v.Text("id"))
	}
	// name is a declared Body attribute without a query mapping: local filter.
	if !reflect.DeepEqual(ids, []string{"b", "c"}) || len(calls) != 2 {
		t.Fatal(ids, calls)
	}
	if calls[0].path != pythonDefaultRuleBase || calls[0].query != "ethertype=IPv4&used_in_default_sg=True" || calls[0].header.Get("Accept") != "application/json" || calls[1].query != "ethertype=IPv4&marker=b&used_in_default_sg=True" {
		t.Fatal(calls)
	}
	api = pythonDefaultRuleAPI(t, &calls, func(*http.Request) (int, string) { return 401, `{}` })
	_, err := api.All(context.Background(), defaultrules.WithListMaxItems(1))
	pythonDefaultRuleFailure(t, err, "List", 401)
}

func TestPythonDefaultRuleFindHasNoNameHint(t *testing.T) {
	var calls []pythonDefaultRuleCall
	api := pythonDefaultRuleAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path != pythonDefaultRuleBase {
			return 404, `{}`
		}
		return 200, `{"default_security_group_rules":[{"id":"a","name":"ssh"},{"id":"b","name":"web"}]}`
	})
	v, err := api.Find(context.Background(), "ssh", defaultrules.WithFindQuery("direction", "ingress"))
	if err != nil || v.Text("id") != "a" || len(calls) != 2 {
		t.Fatal(v, err, calls)
	}
	if calls[0].path != pythonDefaultRuleBase+"/ssh" || calls[0].query != "direction=ingress" || calls[1].path != pythonDefaultRuleBase || calls[1].query != "direction=ingress" {
		t.Fatal(calls)
	}
	if v, err := api.Find(context.Background(), "none"); v != nil || err != nil {
		t.Fatal(v, err)
	}
	if _, err := api.Find(context.Background(), "none", defaultrules.WithFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
}
