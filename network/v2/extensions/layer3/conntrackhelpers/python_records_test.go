package conntrackhelpers_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonConntrackTransport func(*http.Request) (*http.Response, error)

func (transport pythonConntrackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonConntrackCall struct {
	method, path, query, body string
	header                    http.Header
}

func pythonConntrackAPI(t *testing.T, calls *[]pythonConntrackCall, reply func(*http.Request) (int, string)) *conntrackhelpers.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonConntrackTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonConntrackCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Clone()})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return conntrackhelpers.New(client)
}

func pythonConntrackField(t *testing.T, v *conntrackhelpers.Record, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		var got json.RawMessage
		if v != nil && v.Resource != nil {
			got = v.Resource.Body[key]
		}
		t.Fatalf("%s: got %s want %s", key, got, want)
	}
}

func pythonConntrackFailure(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "conntrackhelpers" {
		t.Fatal("operation context", err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatal("native status", code, err)
	}
}

const pythonConntrackBase = "/neutron/v2.0/routers/r%2F1/conntrack_helpers"

func TestPythonConntrackCreateDropsUnknownAttrs(t *testing.T) {
	var calls []pythonConntrackCall
	api := pythonConntrackAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"conntrack_helper":{"id":"h1","helper":"ftp","protocol":"tcp","port":21,"router_id":"r/1","project_id":"p"}}`
	})
	v, err := api.Create(context.Background(), "r/1", conntrackhelpers.WithCreateField("helper", "ftp"), conntrackhelpers.WithCreateField("protocol", "tcp"),
		conntrackhelpers.WithCreateField("port", 21), conntrackhelpers.WithCreateField("x_unknown", true))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonConntrackBase || calls[0].query != "" ||
		calls[0].body != `{"conntrack_helper":{"helper":"ftp","port":21,"protocol":"tcp"}}` || calls[0].header.Get("Content-Type") != "application/json" {
		t.Fatal(calls)
	}
	pythonConntrackField(t, v, "id", `"h1"`)
	pythonConntrackField(t, v, "port", `21`)
	pythonConntrackField(t, v, "name", `null`)
	pythonConntrackField(t, v, "location", `null`)
	if v.RouterID != "r/1" || len(v.Resource.Body) != 6 || string(v.Wire.Body["project_id"]) != `"p"` {
		t.Fatal("undeclared response members stay only in Wire", v.Resource.Body, v)
	}
	calls = nil
	if _, err := api.Create(context.Background(), "r/1", conntrackhelpers.WithCreateField("router_id", "other")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal("router_id duplicates the bound argument", err, calls)
	}
	if _, err := api.Create(context.Background(), " "); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal("blank router fails before HTTP", err, calls)
	}
	api = pythonConntrackAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
	_, err = api.Create(context.Background(), "r/1")
	if calls[0].body != `{"conntrack_helper":{}}` {
		t.Fatal(calls)
	}
	pythonConntrackFailure(t, err, "Create", 409)
}

func TestPythonConntrackGetUpdateDelete(t *testing.T) {
	var calls []pythonConntrackCall
	api := pythonConntrackAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"conntrack_helper":{"helper":"tftp","port":"69"}}`
	})
	v, err := api.Get(context.Background(), "r/1", "h1")
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonConntrackBase+"/h1" || calls[0].query != "" {
		t.Fatal(err, calls)
	}
	pythonConntrackField(t, v, "id", `"h1"`)
	pythonConntrackField(t, v, "port", `"69"`)

	calls = nil
	v, err = api.Update(context.Background(), "r/1", "h1", conntrackhelpers.WithUpdateField("port", 2121), conntrackhelpers.WithUpdateField("if_revision", 3))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pythonConntrackBase+"/h1" || calls[0].body != `{"conntrack_helper":{"port":2121}}` || calls[0].header.Get("If-Match") != "" {
		t.Fatal("plain Resource ignores if_revision", err, calls)
	}
	calls = nil
	if v, err = api.Update(context.Background(), "r/1", "h1", conntrackhelpers.WithUpdateField("unknown", 1)); err != nil || len(calls) != 0 || v.Text("id") != "h1" || v.RouterID != "r/1" {
		t.Fatal("no declared attribute sends nothing", v, err, calls)
	}
	api = pythonConntrackAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{}` })
	_, err = api.Get(context.Background(), "r/1", "h1")
	pythonConntrackFailure(t, err, "Get", 404)
	_, err = api.Update(context.Background(), "r/1", "h1", conntrackhelpers.WithUpdateField("helper", "sip"))
	pythonConntrackFailure(t, err, "Update", 404)

	calls = nil
	result, err := api.Delete(context.Background(), "r/1", "h1")
	if err != nil || !result.IgnoredMissing || len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonConntrackBase+"/h1" {
		t.Fatal(result, err, calls)
	}
	_, err = api.Delete(context.Background(), "r/1", "h1", conntrackhelpers.WithDeleteIgnoreMissing(false))
	pythonConntrackFailure(t, err, "Delete", 404)
	api = pythonConntrackAPI(t, &calls, func(*http.Request) (int, string) { return 204, `` })
	if result, err := api.Delete(context.Background(), "r/1", "h1"); err != nil || result.StatusCode != 204 {
		t.Fatal(result, err)
	}
}

func TestPythonConntrackListFiltersLocallyAndPages(t *testing.T) {
	var calls []pythonConntrackCall
	api := pythonConntrackAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"conntrack_helpers":[{"id":"a","helper":"ftp"},{"id":"b","helper":"tftp"}],"conntrack_helpers_links":[{"rel":"next","href":"http://` + req.Host + `/neutron/v2.0/routers/r%2F1/conntrack_helpers?limit=2&marker=b"}]}`
		case "b":
			return 200, `{"conntrack_helpers":[{"id":"c","helper":"ftp"}]}`
		}
		return 200, `{"conntrack_helpers":[]}`
	})
	var ids []string
	for v, err := range api.List(context.Background(), "r/1", conntrackhelpers.WithListQuery("helper", "ftp"), conntrackhelpers.WithListLimit(2), conntrackhelpers.WithListQuery("project_id", "dropped")) {
		if err != nil {
			t.Fatal(err)
		}
		if v.RouterID != "r/1" {
			t.Fatal(v)
		}
		ids = append(ids, v.Text("id"))
	}
	if !reflect.DeepEqual(ids, []string{"a", "c"}) || len(calls) != 3 || calls[0].path != pythonConntrackBase || calls[0].query != "limit=2" || calls[1].query != "limit=2&marker=b" || calls[0].header.Get("Accept") != "application/json" {
		t.Fatal(ids, calls)
	}
	calls = nil
	if _, err := api.All(context.Background(), "r/1", conntrackhelpers.WithListQuery("router_id", "x")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal("router_id duplicates the bound argument", err, calls)
	}
	api = pythonConntrackAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{}` })
	_, err := api.All(context.Background(), "r/1")
	pythonConntrackFailure(t, err, "List", 404)
}
