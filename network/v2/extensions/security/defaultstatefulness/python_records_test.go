package defaultstatefulness_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonStatefulnessTransport func(*http.Request) (*http.Response, error)

func (transport pythonStatefulnessTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonStatefulnessCall struct {
	method, path, query, body string
	header                    http.Header
}

func pythonStatefulnessAPI(t *testing.T, calls *[]pythonStatefulnessCall, reply func(*http.Request) (int, string)) *defaultstatefulness.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonStatefulnessTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonStatefulnessCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Clone()})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return defaultstatefulness.New(client)
}

func pythonStatefulnessField(t *testing.T, v *defaultstatefulness.Record, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		var got json.RawMessage
		if v != nil && v.Resource != nil {
			got = v.Resource.Body[key]
		}
		t.Fatalf("%s: got %s want %s", key, got, want)
	}
}

func pythonStatefulnessFailure(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "defaultstatefulness" {
		t.Fatal("operation context", err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatal("native status", code, err)
	}
}

const pythonStatefulnessBase = "/neutron/v2.0/security-groups-default-statefulness"

func TestPythonStatefulnessCreateAndGet(t *testing.T) {
	var calls []pythonStatefulnessCall
	api := pythonStatefulnessAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"security_group_default_statefulness":{"id":"s1","project_id":"p","stateful":0,"revision_number":"1","tenant_id":"p"}}`
	})
	v, err := api.Create(context.Background(), defaultstatefulness.WithCreateField("project_id", "p"), defaultstatefulness.WithCreateField("stateful", false))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonStatefulnessBase || calls[0].body != `{"security_group_default_statefulness":{"project_id":"p","stateful":false}}` || calls[0].header.Get("If-Match") != "" {
		t.Fatal(calls)
	}
	pythonStatefulnessField(t, v, "stateful", `false`)
	pythonStatefulnessField(t, v, "revision_number", `1`)
	pythonStatefulnessField(t, v, "tenant_id", `"p"`)
	pythonStatefulnessField(t, v, "if_match", `null`)
	api = pythonStatefulnessAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
	_, err = api.Create(context.Background(), defaultstatefulness.WithCreateField("stateful", true))
	pythonStatefulnessFailure(t, err, "Create", 409)

	calls = nil
	api = pythonStatefulnessAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"security_group_default_statefulness":{"stateful":true}}`
	})
	v, err = api.Get(context.Background(), "s1", defaultstatefulness.WithGetFields("stateful"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonStatefulnessBase+"/s1" || calls[0].query != "fields=stateful" {
		t.Fatal(err, calls)
	}
	pythonStatefulnessField(t, v, "id", `"s1"`)
	pythonStatefulnessField(t, v, "stateful", `true`)
	api = pythonStatefulnessAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{}` })
	_, err = api.Get(context.Background(), "s1")
	pythonStatefulnessFailure(t, err, "Get", 404)
}

func TestPythonStatefulnessUpdateRevisionAndNoop(t *testing.T) {
	var calls []pythonStatefulnessCall
	api := pythonStatefulnessAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"security_group_default_statefulness":{"id":"s1","stateful":false,"revision_number":4}}`
	})
	v, err := api.Update(context.Background(), "s1", defaultstatefulness.WithUpdateRevision(3), defaultstatefulness.WithUpdateIfMatch("ignored"), defaultstatefulness.WithUpdateField("stateful", false))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pythonStatefulnessBase+"/s1" || calls[0].body != `{"security_group_default_statefulness":{"stateful":false}}` || calls[0].header.Get("If-Match") != "revision_number=3" {
		t.Fatal(calls)
	}
	pythonStatefulnessField(t, v, "revision_number", `4`)
	pythonStatefulnessField(t, v, "if_match", `["revision_number=3"]`)

	// A dirty Header alone still commits an empty Body.
	calls = nil
	if _, err := api.Update(context.Background(), "s1", defaultstatefulness.WithUpdateIfMatch("revision_number=1")); err != nil || len(calls) != 1 ||
		calls[0].body != `{"security_group_default_statefulness":{}}` || calls[0].header.Get("If-Match") != "revision_number=1" {
		t.Fatal(err, calls)
	}
	calls = nil
	v, err = api.Update(context.Background(), "s1", defaultstatefulness.WithUpdateField("if_revision", nil))
	if err != nil || len(calls) != 0 || v.StatusCode != 0 {
		t.Fatal("null if_revision and no attribute send nothing", v, err, calls)
	}
	pythonStatefulnessField(t, v, "id", `"s1"`)
	for _, option := range []defaultstatefulness.UpdateOption{defaultstatefulness.WithUpdateField("id", "x"), defaultstatefulness.WithUpdateField("if_revision", 1.5)} {
		if _, err := api.Update(context.Background(), "s1", option); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	}
	api = pythonStatefulnessAPI(t, &calls, func(*http.Request) (int, string) { return 412, `{}` })
	_, err = api.Update(context.Background(), "s1", defaultstatefulness.WithUpdateRevision(1), defaultstatefulness.WithUpdateField("stateful", true))
	pythonStatefulnessFailure(t, err, "Update", 412)
}

func TestPythonStatefulnessDeleteListAndFind(t *testing.T) {
	var calls []pythonStatefulnessCall
	api := pythonStatefulnessAPI(t, &calls, func(*http.Request) (int, string) { return 404, `` })
	result, err := api.Delete(context.Background(), "s1")
	if err != nil || !result.IgnoredMissing || len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonStatefulnessBase+"/s1" {
		t.Fatal(result, err, calls)
	}
	_, err = api.Delete(context.Background(), "s1", defaultstatefulness.WithDeleteIgnoreMissing(false))
	pythonStatefulnessFailure(t, err, "Delete", 404)

	calls = nil
	api = pythonStatefulnessAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path != pythonStatefulnessBase {
			return 404, `{}`
		}
		if req.URL.Query().Get("marker") == "" {
			next := req.URL.Query()
			next.Set("marker", "a")
			return 200, `{"security_groups_default_statefulness":[{"id":"a","project_id":"p","stateful":true}],"security_groups_default_statefulness_links":[{"rel":"next","href":"http://` + req.Host + `/neutron/v2.0/security-groups-default-statefulness?` + next.Encode() + `"}]}`
		}
		return 200, `{"security_groups_default_statefulness":[{"id":"b","project_id":"p","stateful":false,"name":"named"}]}`
	})
	var ids []string
	for v, err := range api.List(context.Background(), defaultstatefulness.WithListQuery("project_id", "p"), defaultstatefulness.WithListQuery("revision_number", nil)) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, v.Text("id"))
	}
	if !reflect.DeepEqual(ids, []string{"a", "b"}) || len(calls) != 2 || calls[0].query != "project_id=p" || calls[1].query != "marker=a&project_id=p" || calls[0].header.Get("Accept") != "application/json" {
		t.Fatal(ids, calls)
	}
	calls = nil
	v, err := api.Find(context.Background(), "named")
	if err != nil || v.Text("id") != "b" || len(calls) != 3 || calls[0].path != pythonStatefulnessBase+"/named" || calls[1].query != "" {
		t.Fatal("no name query hint", v, err, calls)
	}
	if _, err := api.Find(context.Background(), "none", defaultstatefulness.WithFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	api = pythonStatefulnessAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.All(context.Background())
	pythonStatefulnessFailure(t, err, "List", 500)
}
