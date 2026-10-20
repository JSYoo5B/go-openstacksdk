package endpointgroups_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonEndpointGroupTransport func(*http.Request) (*http.Response, error)

func (transport pythonEndpointGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonEndpointGroupCall struct {
	method, path, query, body string
	header                    http.Header
}

func pythonEndpointGroupAPI(t *testing.T, calls *[]pythonEndpointGroupCall, reply func(*http.Request) (int, string)) *endpointgroups.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonEndpointGroupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonEndpointGroupCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Clone()})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return endpointgroups.New(client)
}

func pythonEndpointGroupField(t *testing.T, v *endpointgroups.Record, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		var got json.RawMessage
		if v != nil && v.Resource != nil {
			got = v.Resource.Body[key]
		}
		t.Fatalf("%s: got %s want %s", key, got, want)
	}
}

func pythonEndpointGroupOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "endpointgroups" {
		t.Fatal("operation context", err)
	}
}

func pythonEndpointGroupIDs(t *testing.T, seq func(func(*endpointgroups.Record, error) bool)) ([]string, error) {
	t.Helper()
	var ids []string
	for v, err := range seq {
		if err != nil {
			return ids, err
		}
		ids = append(ids, v.Text("id"))
	}
	return ids, nil
}

const pythonEndpointGroupBase = "/keystone/v3/OS-EP-FILTER/endpoint_groups"

func TestPythonEndpointGroupCreateSendsDeclaredAttrs(t *testing.T) {
	var calls []pythonEndpointGroupCall
	api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"endpoint_group":{"id":"eg1","name":"web","filters":{"interface":"public"},"links":{"self":"x"},"extra":1}}`
	})
	v, err := api.Create(context.Background(), endpointgroups.WithCreateField("name", "web"), endpointgroups.WithCreateField("filters", map[string]string{"interface": "public"}),
		endpointgroups.WithCreateField("description", nil), endpointgroups.WithCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"endpoint_group":{"description":null,"filters":{"interface":"public"},"name":"web"}}`
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonEndpointGroupBase || calls[0].query != "" || calls[0].body != want || calls[0].header.Get("Content-Type") != "application/json" {
		t.Fatal(calls)
	}
	pythonEndpointGroupField(t, v, "id", `"eg1"`)
	pythonEndpointGroupField(t, v, "name", `"web"`)
	pythonEndpointGroupField(t, v, "filters", `{"interface":"public"}`)
	pythonEndpointGroupField(t, v, "description", `null`)
	pythonEndpointGroupField(t, v, "links", `{"self":"x"}`)
	pythonEndpointGroupField(t, v, "location", `null`)
	if len(v.Resource.Body) != 6 || v.StatusCode != 201 || string(v.Wire.Body["extra"]) != `1` {
		t.Fatal(v.Resource.Body, v)
	}
	calls = nil
	if _, err := api.Create(context.Background()); err != nil || calls[0].body != `{"endpoint_group":{}}` {
		t.Fatal("Resource.create has no required-field precheck", err, calls)
	}
	calls = nil
	api = pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{"error":"conflict"}` })
	_, err = api.Create(context.Background(), endpointgroups.WithCreateField("name", "web"))
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 409 {
		t.Fatal(err)
	}
	pythonEndpointGroupOperation(t, err, "Create")
}

func TestPythonEndpointGroupGetStatusesAndEscapedID(t *testing.T) {
	for _, code := range []int{200, 203, 204, 301, 399, 400, 403, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []pythonEndpointGroupCall
			api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) {
				return code, `{"endpoint_group":{"name":"web","filters":{}}}`
			})
			v, err := api.Get(context.Background(), "eg1")
			if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonEndpointGroupBase+"/eg1" || calls[0].body != "" {
				t.Fatal(calls)
			}
			if code < 400 {
				if err != nil || v.StatusCode != code {
					t.Fatal(v, err)
				}
				pythonEndpointGroupField(t, v, "id", `"eg1"`)
				pythonEndpointGroupField(t, v, "name", `"web"`)
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.As(err, &native) || native.Actual != code {
				t.Fatal(v, err)
			}
			pythonEndpointGroupOperation(t, err, "Get")
		})
	}
	t.Run("non-object response keeps proof", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return 200, `[]` })
		_, err := api.Get(context.Background(), "eg1")
		var proof *resource.ResponseError
		if !errors.As(err, &proof) || proof.StatusCode != 200 {
			t.Fatal(err)
		}
	})
	t.Run("escaped", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
		if _, err := api.Get(context.Background(), "a/b c"); err != nil || calls[0].path != pythonEndpointGroupBase+"/a%2Fb%20c" {
			t.Fatal(err, calls)
		}
		if _, err := api.Get(context.Background(), " "); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
			t.Fatal("blank ID fails before HTTP", err, calls)
		}
	})
}

func TestPythonEndpointGroupUpdatePatchesDirtyAttrs(t *testing.T) {
	var calls []pythonEndpointGroupCall
	api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"endpoint_group":{"id":"eg1","description":"new"}}`
	})
	v, err := api.Update(context.Background(), "eg1", endpointgroups.WithUpdateField("description", "new"), endpointgroups.WithUpdateField("filters", map[string]any{}),
		endpointgroups.WithUpdateField("id", "other"), endpointgroups.WithUpdateField("unknown", 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPatch || calls[0].path != pythonEndpointGroupBase+"/eg1" || calls[0].body != `{"endpoint_group":{"description":"new","filters":{}}}` {
		t.Fatal(calls)
	}
	pythonEndpointGroupField(t, v, "id", `"eg1"`)
	pythonEndpointGroupField(t, v, "description", `"new"`)
	pythonEndpointGroupField(t, v, "filters", `{}`)
	pythonEndpointGroupField(t, v, "name", `null`)
	calls = nil
	v, err = api.Update(context.Background(), "eg1", endpointgroups.WithUpdateField("unknown", 1))
	if err != nil || len(calls) != 0 || v.StatusCode != 0 {
		t.Fatal("no dirty attribute sends nothing", v, err, calls)
	}
	pythonEndpointGroupField(t, v, "id", `"eg1"`)
	api = pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return 404, `missing` })
	_, err = api.Update(context.Background(), "eg1", endpointgroups.WithUpdateField("name", "x"))
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 404 {
		t.Fatal(err)
	}
	pythonEndpointGroupOperation(t, err, "Update")
}

func TestPythonEndpointGroupDeleteIgnoreMissing(t *testing.T) {
	for _, code := range []int{200, 204, 301, 400, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []pythonEndpointGroupCall
			api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return code, "opaque" })
			err := api.Delete(context.Background(), "eg1")
			if (err == nil) != (code < 400 || code == 404) || len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonEndpointGroupBase+"/eg1" || calls[0].query != "" {
				t.Fatal(err, calls)
			}
			if err != nil {
				pythonEndpointGroupOperation(t, err, "Delete")
			}
			if code == 404 {
				err = api.Delete(context.Background(), "eg1", endpointgroups.WithDeleteIgnoreMissing(false))
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPythonEndpointGroupListQueryFiltersAndPaging(t *testing.T) {
	var calls []pythonEndpointGroupCall
	api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"endpoint_groups":[{"id":"a","name":"web","description":"d","filters":{"interface":"public","region_id":"r1"}},{"id":"b","name":"web","filters":{"interface":"admin"}}],"links":{"next":null,"self":"x"}}`
	})
	ids, err := pythonEndpointGroupIDs(t, api.List(context.Background(), endpointgroups.WithListName("web"), endpointgroups.WithListLimit(5), endpointgroups.WithListMarker("m"),
		endpointgroups.WithListQuery("filters", map[string]string{"interface": "public"}), endpointgroups.WithListQuery("unknown", 1)))
	if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(calls) != 1 || calls[0].path != pythonEndpointGroupBase || calls[0].query != "name=web" || calls[0].header.Get("Accept") != "application/json" {
		t.Fatal("name is wire query, limit and marker are dropped, filters is local", ids, err, calls)
	}
	calls = nil
	ids, err = pythonEndpointGroupIDs(t, api.List(context.Background(), endpointgroups.WithListQuery("description", nil), endpointgroups.WithListQuery("id", "b")))
	if err != nil || !reflect.DeepEqual(ids, []string{"b"}) || calls[0].query != "" {
		t.Fatal(ids, err, calls)
	}
	t.Run("max items sends limit and follows the last ID", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Get("marker") == "a" {
				return 200, `{"endpoint_groups":[{"id":"b"},{"id":"c"}]}`
			}
			return 200, `{"endpoint_groups":[{"id":"a"}]}`
		})
		ids, err := pythonEndpointGroupIDs(t, api.List(context.Background(), endpointgroups.WithListMaxItems(2)))
		if err != nil || !reflect.DeepEqual(ids, []string{"a", "b"}) || len(calls) != 2 || calls[0].query != "limit=2" || calls[1].query != "limit=2&marker=a" {
			t.Fatal(ids, err, calls)
		}
	})
	t.Run("paginated false reads one page", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"endpoint_groups":[{"id":"a"}],"next":"?marker=a"}`
		})
		ids, err := pythonEndpointGroupIDs(t, api.List(context.Background(), endpointgroups.WithListPaginated(false)))
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(calls) != 1 {
			t.Fatal(ids, err, calls)
		}
	})
	t.Run("top-level next is followed", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Get("marker") == "a" {
				return 200, `{"endpoint_groups":[{"id":"b"}]}`
			}
			return 200, `{"endpoint_groups":[{"id":"a"}],"next":"?marker=a"}`
		})
		ids, err := pythonEndpointGroupIDs(t, api.List(context.Background()))
		if err != nil || !reflect.DeepEqual(ids, []string{"a", "b"}) || len(calls) != 2 {
			t.Fatal(ids, err, calls)
		}
	})
	t.Run("rejected page keeps context", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return 403, `denied` })
		_, err := pythonEndpointGroupIDs(t, api.List(context.Background()))
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 403 {
			t.Fatal(err)
		}
		pythonEndpointGroupOperation(t, err, "List")
	})
}

func TestPythonEndpointGroupFindByIDThenName(t *testing.T) {
	t.Run("direct ID", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"endpoint_group":{"id":"eg1","name":"web"}}` })
		v, err := api.Find(context.Background(), "eg1")
		if err != nil || len(calls) != 1 || calls[0].path != pythonEndpointGroupBase+"/eg1" {
			t.Fatal(v, err, calls)
		}
		pythonEndpointGroupField(t, v, "name", `"web"`)
	})
	for _, status := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls []pythonEndpointGroupCall
			api := pythonEndpointGroupAPI(t, &calls, func(req *http.Request) (int, string) {
				if strings.HasSuffix(req.URL.Path, "/web") {
					return status, "missing"
				}
				return 200, `{"endpoint_groups":[{"id":"eg1","name":"web"}],"links":{"next":null}}`
			})
			v, err := api.Find(context.Background(), "web")
			if err != nil || len(calls) != 2 || calls[1].path != pythonEndpointGroupBase || calls[1].query != "name=web" {
				t.Fatal(v, err, calls)
			}
			pythonEndpointGroupField(t, v, "id", `"eg1"`)
		})
	}
	for _, item := range []struct {
		name, list string
		strict     bool
		want       string
	}{
		{"missing", `[]`, false, ""}, {"strict missing", `[]`, true, "missing"},
		{"duplicate", `[{"id":"a","name":"web"},{"id":"b","name":"web"}]`, false, "ambiguous"},
		{"ID match in list", `[{"id":"web","name":"other"}]`, false, "web"},
	} {
		t.Run(item.name, func(t *testing.T) {
			var calls []pythonEndpointGroupCall
			api := pythonEndpointGroupAPI(t, &calls, func(req *http.Request) (int, string) {
				if strings.HasSuffix(req.URL.Path, "/web") {
					return 404, "missing"
				}
				return 200, `{"endpoint_groups":` + item.list + `}`
			})
			v, err := api.Find(context.Background(), "web", endpointgroups.WithFindIgnoreMissing(!item.strict))
			switch item.want {
			case "":
				if err != nil || v != nil {
					t.Fatal(v, err)
				}
			case "missing":
				var target *resource.NotFoundError
				if !errors.As(err, &target) {
					t.Fatal(err)
				}
				pythonEndpointGroupOperation(t, err, "Find")
			case "ambiguous":
				var target *resource.AmbiguousError
				if !errors.As(err, &target) {
					t.Fatal(err)
				}
			default:
				if err != nil || v.Text("id") != item.want {
					t.Fatal(v, err)
				}
			}
		})
	}
	t.Run("server error does not fall back", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return 500, "boom" })
		_, err := api.Find(context.Background(), "web")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 500 || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonEndpointGroupScopedLists(t *testing.T) {
	t.Run("endpoint group endpoints", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Has("marker") {
				return 200, `{"endpoints":[]}`
			}
			return 200, `{"endpoints":[{"id":"e1","interface":"public","enabled":true,"url":"http://a","region_id":"r1","service_id":"s1"},{"id":"e2","enabled":false}],"links":{"next":null}}`
		})
		ids, err := pythonEndpointGroupIDs(t, api.ListEndpoints(context.Background(), "eg1", endpointgroups.WithListQuery("interface", "public"), endpointgroups.WithListQuery("region_id", "r1"),
			endpointgroups.WithListQuery("service_id", "s1"), endpointgroups.WithListLimit(10), endpointgroups.WithListQuery("is_enabled", true)))
		if err != nil || !reflect.DeepEqual(ids, []string{"e1"}) || len(calls) != 2 || calls[0].path != pythonEndpointGroupBase+"/eg1/endpoints" || calls[0].query != "interface=public&limit=10&region_id=r1&service_id=s1" || calls[1].query != "interface=public&limit=10&marker=e2&region_id=r1&service_id=s1" {
			t.Fatal(ids, err, calls)
		}
		var first *endpointgroups.Record
		for v, err := range api.ListEndpoints(context.Background(), "eg1") {
			if err != nil {
				t.Fatal(err)
			}
			first = v
			break
		}
		pythonEndpointGroupField(t, first, "is_enabled", `true`)
		pythonEndpointGroupField(t, first, "endpoint_group_id", `"eg1"`)
		pythonEndpointGroupField(t, first, "url", `"http://a"`)
		pythonEndpointGroupField(t, first, "name", `null`)
		if _, ok := first.Resource.Body["enabled"]; ok || len(first.Resource.Body) != 10 {
			t.Fatal(first.Resource.Body)
		}
	})
	t.Run("endpoint group projects", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"projects":[{"id":"p1","name":"demo","domain_id":"d","enabled":true,"description":"x","vendor":"v"},{"id":"p2","description":"y","tags":"t"}]}`
		})
		ids, err := pythonEndpointGroupIDs(t, api.ListProjects(context.Background(), "eg1", endpointgroups.WithListQuery("is_enabled", true), endpointgroups.WithListQuery("any_tags", []string{"a", "b"}),
			endpointgroups.WithListQuery("domain_id", "d"), endpointgroups.WithListQuery("description", "x"), endpointgroups.WithListQuery("vendor", "v")))
		if err != nil || !reflect.DeepEqual(ids, []string{"p1"}) || calls[0].path != pythonEndpointGroupBase+"/eg1/projects" || calls[0].query != "domain_id=d&enabled=True&tags-any=a&tags-any=b" {
			t.Fatal(ids, err, calls)
		}
		var rows []*endpointgroups.Record
		for v, err := range api.ListProjects(context.Background(), "eg1") {
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, v)
		}
		pythonEndpointGroupField(t, rows[0], "is_enabled", `true`)
		pythonEndpointGroupField(t, rows[0], "tags", `[]`)
		pythonEndpointGroupField(t, rows[0], "vendor", `"v"`)
		pythonEndpointGroupField(t, rows[0], "endpoint_group_id", `"eg1"`)
		pythonEndpointGroupField(t, rows[1], "tags", `["t"]`)
		pythonEndpointGroupField(t, rows[1], "is_domain", `null`)
	})
	t.Run("project endpoint groups", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"endpoint_groups":[{"id":"eg1","name":"web"}]}`
		})
		var got *endpointgroups.Record
		for v, err := range api.ListProjectEndpointGroups(context.Background(), "p1", endpointgroups.WithListName("web"), endpointgroups.WithListLimit(3)) {
			if err != nil {
				t.Fatal(err)
			}
			got = v
		}
		if len(calls) != 1 || calls[0].path != "/keystone/v3/OS-EP-FILTER/projects/p1/endpoint_groups" || calls[0].query != "name=web" {
			t.Fatal(calls)
		}
		pythonEndpointGroupField(t, got, "project_id", `"p1"`)
		pythonEndpointGroupField(t, got, "name", `"web"`)
	})
	t.Run("endpoint projects", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"projects":[{"id":"p1","is_domain":false}]}`
		})
		var got *endpointgroups.Record
		for v, err := range api.ListEndpointProjects(context.Background(), "e1", endpointgroups.WithListQuery("name", "demo"), endpointgroups.WithListMarker("m")) {
			if err != nil {
				t.Fatal(err)
			}
			got = v
		}
		if len(calls) != 1 || calls[0].path != "/keystone/v3/OS-EP-FILTER/endpoints/e1/projects" || calls[0].query != "marker=m&name=demo" {
			t.Fatal(calls)
		}
		pythonEndpointGroupField(t, got, "endpoint_id", `"e1"`)
		pythonEndpointGroupField(t, got, "is_domain", `false`)
	})
	t.Run("rejected scope", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return 404, `missing` })
		_, err := pythonEndpointGroupIDs(t, api.ListProjects(context.Background(), "eg1"))
		pythonEndpointGroupOperation(t, err, "ListProjects")
		_, err = pythonEndpointGroupIDs(t, api.ListEndpoints(context.Background(), ""))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonEndpointGroupProjectAssociation(t *testing.T) {
	for _, item := range []struct {
		name, method string
		call         func(*endpointgroups.API) error
	}{
		{"AssociateProject", http.MethodPut, func(api *endpointgroups.API) error {
			return api.AssociateProject(context.Background(), "eg1", "p1")
		}},
		{"DisassociateProject", http.MethodDelete, func(api *endpointgroups.API) error {
			return api.DisassociateProject(context.Background(), "eg1", "p1")
		}},
	} {
		for _, code := range []int{200, 204, 404, 409} {
			t.Run(fmt.Sprint(item.name, code), func(t *testing.T) {
				var calls []pythonEndpointGroupCall
				api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return code, "" })
				err := item.call(api)
				if len(calls) != 1 || calls[0].method != item.method || calls[0].path != pythonEndpointGroupBase+"/eg1/projects/p1" || calls[0].body != "" || calls[0].query != "" {
					t.Fatal(calls)
				}
				if code < 400 {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code {
					t.Fatal(err)
				}
				pythonEndpointGroupOperation(t, err, item.name)
			})
		}
	}
}
