package ndpproxies_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonNDPProxyCall struct {
	method, path, query, body string
	header                    http.Header
}

func pythonNDPProxyAPI(t *testing.T, calls *[]pythonNDPProxyCall, reply func(*http.Request) (int, string)) *ndpproxies.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonNDPProxyCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Clone()})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return ndpproxies.New(client)
}

func pythonNDPProxyField(t *testing.T, v *ndpproxies.Record, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		var got json.RawMessage
		if v != nil && v.Resource != nil {
			got = v.Resource.Body[key]
		}
		t.Fatalf("%s: got %s want %s", key, got, want)
	}
}

func pythonNDPProxyBody(t *testing.T, raw string, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil || !reflect.DeepEqual(got, map[string]any{"ndp_proxy": want}) {
		t.Fatal(raw, want, err)
	}
}

func pythonNDPProxyFailure(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "ndpproxies" {
		t.Fatal("operation context", err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatal("native status", code, err)
	}
}

const pythonNDPProxyBase = "/neutron/v2.0/ndp_proxies"

func TestPythonNDPProxyCreateSendsProvidedAttrs(t *testing.T) {
	var calls []pythonNDPProxyCall
	api := pythonNDPProxyAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"ndp_proxy":{"id":"n1","router_id":"r1","port_id":"p1","ip_address":"2001:db8::10","revision_number":"0","tenant_id":"t"}}`
	})
	v, err := api.Create(context.Background(), ndpproxies.WithCreateField("router_id", "r1"), ndpproxies.WithCreateField("port_id", "p1"),
		ndpproxies.WithCreateField("ip_address", "2001:db8::10"), ndpproxies.WithCreateField("description", nil), ndpproxies.WithCreateField("x_extension", 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonNDPProxyBase || calls[0].query != "" || calls[0].header.Get("Content-Type") != "application/json" {
		t.Fatal(calls)
	}
	pythonNDPProxyBody(t, calls[0].body, map[string]any{"router_id": "r1", "port_id": "p1", "ip_address": "2001:db8::10", "description": nil, "x_extension": float64(1)})
	pythonNDPProxyField(t, v, "id", `"n1"`)
	pythonNDPProxyField(t, v, "description", `null`)
	pythonNDPProxyField(t, v, "revision_number", `"0"`)
	pythonNDPProxyField(t, v, "project_id", `null`)
	pythonNDPProxyField(t, v, "tenant_id", `"t"`)
	pythonNDPProxyField(t, v, "x_extension", `1`)
	pythonNDPProxyField(t, v, "location", `null`)
	if v.StatusCode != 201 || v.Text("router_id") != "r1" {
		t.Fatal(v)
	}
	calls = nil
	if _, err := api.Create(context.Background(), ndpproxies.WithCreateField("connection", "x")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal("constructor control fails before HTTP", err, calls)
	}
	api = pythonNDPProxyAPI(t, &calls, func(*http.Request) (int, string) { return 400, `{"NeutronError":{}}` })
	_, err = api.Create(context.Background())
	if calls[0].body != `{"ndp_proxy":{}}` {
		t.Fatal(calls)
	}
	pythonNDPProxyFailure(t, err, "Create", 400)
}

func TestPythonNDPProxyGetUpdateAndDelete(t *testing.T) {
	for _, code := range []int{200, 404} {
		t.Run(fmt.Sprint("get ", code), func(t *testing.T) {
			var calls []pythonNDPProxyCall
			api := pythonNDPProxyAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"ndp_proxy":{"name":"proxy"}}` })
			v, err := api.Get(context.Background(), "n/1", ndpproxies.WithGetFields("name", "ip_address"))
			if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonNDPProxyBase+"/n%2F1" || calls[0].query != "fields=name&fields=ip_address" {
				t.Fatal(calls)
			}
			if code == 404 {
				pythonNDPProxyFailure(t, err, "Get", 404)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			pythonNDPProxyField(t, v, "id", `"n/1"`)
			pythonNDPProxyField(t, v, "name", `"proxy"`)
		})
	}
	t.Run("update", func(t *testing.T) {
		var calls []pythonNDPProxyCall
		api := pythonNDPProxyAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"ndp_proxy":{"id":"n1","description":"new"}}`
		})
		v, err := api.Update(context.Background(), "n1", ndpproxies.WithUpdateField("description", "new"))
		if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pythonNDPProxyBase+"/n1" {
			t.Fatal(err, calls)
		}
		pythonNDPProxyBody(t, calls[0].body, map[string]any{"description": "new"})
		pythonNDPProxyField(t, v, "description", `"new"`)
		calls = nil
		if v, err := api.Update(context.Background(), "n1"); err != nil || len(calls) != 0 || v.Text("id") != "n1" {
			t.Fatal("no changed attribute sends nothing", v, err, calls)
		}
		api = pythonNDPProxyAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
		_, err = api.Update(context.Background(), "n1", ndpproxies.WithUpdateField("name", "x"))
		pythonNDPProxyFailure(t, err, "Update", 409)
	})
	t.Run("delete", func(t *testing.T) {
		var calls []pythonNDPProxyCall
		api := pythonNDPProxyAPI(t, &calls, func(*http.Request) (int, string) { return 404, `` })
		result, err := api.Delete(context.Background(), "n1")
		if err != nil || !result.IgnoredMissing || len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonNDPProxyBase+"/n1" {
			t.Fatal(result, err, calls)
		}
		_, err = api.Delete(context.Background(), "n1", ndpproxies.WithDeleteIgnoreMissing(false))
		pythonNDPProxyFailure(t, err, "Delete", 404)
		api = pythonNDPProxyAPI(t, &calls, func(*http.Request) (int, string) { return 204, `` })
		if result, err := api.Delete(context.Background(), "n1"); err != nil || result.StatusCode != 204 || result.IgnoredMissing {
			t.Fatal(result, err)
		}
	})
}

func TestPythonNDPProxyListReadsOnlyTheFirstPage(t *testing.T) {
	var calls []pythonNDPProxyCall
	api := pythonNDPProxyAPI(t, &calls, func(req *http.Request) (int, string) {
		return 200, `{"ndp_proxies":[{"id":"a","router_id":"r1","ip_address":"x"},{"id":"b","router_id":"r1","ip_address":"y"}],"ndp_proxies_links":[{"rel":"next","href":"http://` + req.Host + `/neutron/v2.0/ndp_proxies?marker=b"}]}`
	})
	var ids []string
	for v, err := range api.List(context.Background(), ndpproxies.WithListQuery("router_id", "r1"), ndpproxies.WithListQuery("revision_number", nil), ndpproxies.WithListLimit(2)) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, v.Text("id"))
	}
	// revision_number is not a query parameter, so null filters rows locally.
	if len(ids) != 2 || len(calls) != 1 || calls[0].path != pythonNDPProxyBase || calls[0].query != "limit=2&router_id=r1" || calls[0].header.Get("Accept") != "application/json" {
		t.Fatal(ids, calls)
	}
	calls = nil
	values, err := api.All(context.Background(), ndpproxies.WithListQuery("ip_address", "y"), ndpproxies.WithListQuery("created_at", nil))
	if err != nil || len(values) != 2 || len(calls) != 1 || calls[0].query != "ip_address=y" {
		t.Fatal(values, err, calls)
	}
	calls = nil
	if _, err := api.All(context.Background(), ndpproxies.WithListQuery("paginated", true)); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal("paginated duplicates the fixed proxy argument", err, calls)
	}
	api = pythonNDPProxyAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.All(context.Background())
	pythonNDPProxyFailure(t, err, "List", 500)
}

func TestPythonNDPProxyFindFallsBackToCompleteNameList(t *testing.T) {
	var calls []pythonNDPProxyCall
	api := pythonNDPProxyAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path != pythonNDPProxyBase {
			return 404, `{}`
		}
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"ndp_proxies":[{"id":"a","name":"other"}],"ndp_proxies_links":[{"rel":"next","href":"http://` + req.Host + `/neutron/v2.0/ndp_proxies?name=proxy&marker=a"}]}`
		}
		return 200, `{"ndp_proxies":[{"id":"b","name":"proxy"}]}`
	})
	v, err := api.Find(context.Background(), "proxy")
	if err != nil || v.Text("id") != "b" || len(calls) != 3 {
		t.Fatal(v, err, calls)
	}
	if calls[0].path != pythonNDPProxyBase+"/proxy" || calls[1].query != "name=proxy" || calls[2].query != "marker=a&name=proxy" {
		t.Fatal(calls)
	}
	api = pythonNDPProxyAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path != pythonNDPProxyBase {
			return 403, `{}`
		}
		return 200, `{"ndp_proxies":[]}`
	})
	if v, err := api.Find(context.Background(), "none"); v != nil || err != nil {
		t.Fatal(v, err)
	}
	if _, err := api.Find(context.Background(), "none", ndpproxies.WithFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
}
