package servicegraphs_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/servicegraphs"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type sgCall struct{ method, path, query, body, accept string }

const sgBase = "/neutron/v2.0/sfc/service_graphs"

func sgAPI(t *testing.T, calls *[]sgCall, reply func(*http.Request) (int, string)) *servicegraphs.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, sgCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return servicegraphs.New(client)
}

func sgOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "service_graphs" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func sgStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const sgRow = `{"id":"x1","name":"web","description":"d","project_id":"p1","tenant_id":"p1","port_chains":{"c1":["c2","c3"],"c4":["c5"]},"extra":1}`

func sgCheck(t *testing.T, v *servicegraphs.ServiceGraph) {
	t.Helper()
	if v == nil || v.ID != "x1" || v.Name != "web" || v.Description != "d" || v.ProjectID != "p1" || v.TenantID != "p1" {
		t.Fatalf("decoded ServiceGraph: %#v", v)
	}
	if len(v.PortChains) != 2 || len(v.PortChains["c1"]) != 2 || v.PortChains["c4"][0] != "c5" {
		t.Fatalf("decoded ServiceGraph fields: %#v", v)
	}
	if len(v.Body) != 6 || v.Body["extra"] != nil || string(v.Wire["extra"]) != "1" {
		t.Fatalf("raw views: %#v", v.Body)
	}
}

func TestServiceGraphCreatePostsEnvelope(t *testing.T) {
	var calls []sgCall
	api := sgAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"service_graph":` + sgRow + `}` })
	v, err := api.Create(context.Background(),
		servicegraphs.WithCreateField("name", "web"),
		servicegraphs.WithCreateField("port_chains", map[string][]string{"c1": {"c2", "c3"}}),
		servicegraphs.WithCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != sgBase || calls[0].query != "" || calls[0].body != `{"service_graph":{"name":"web","port_chains":{"c1":["c2","c3"]}}}` {
		t.Fatal(calls)
	}
	sgCheck(t, v)
	if v.StatusCode != 201 {
		t.Fatal(v.StatusCode)
	}
	for _, code := range []int{200, 400, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []sgCall
			api := sgAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Create(context.Background(), servicegraphs.WithCreateField("tenant_id", "p1"))
			if calls[0].body != `{"service_graph":{"tenant_id":"p1"}}` {
				t.Fatal(calls)
			}
			if code < 400 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			sgOperation(t, err, "Create")
			sgStatus(t, err, code)
		})
	}
}

func TestServiceGraphGetDecodesDescriptorsAndRejects(t *testing.T) {
	var calls []sgCall
	api := sgAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"service_graph":` + sgRow + `}` })
	v, err := api.Get(context.Background(), "x1")
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != sgBase+"/x1" {
		t.Fatal(v, err, calls)
	}
	sgCheck(t, v)

	calls = nil
	api = sgAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"id":"x1","tenant_id":"t9","project_id":null,"port_chains":["c1"]}`
	})
	v, err = api.Get(context.Background(), "x/1")
	if err != nil || calls[0].path != sgBase+"/x%2F1" || v.ProjectID != "" || string(v.Body["project_id"]) != "null" || v.PortChains != nil || string(v.Body["port_chains"]) != `["c1"]` {
		t.Fatal("Python descriptor conversion and the tenant_id alias", v, err, calls)
	}

	calls = nil
	api = sgAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{"NeutronError":{}}` })
	_, err = api.Get(context.Background(), "missing")
	sgOperation(t, err, "Get")
	sgStatus(t, err, 404)
	if _, err := api.Get(context.Background(), ""); err == nil || len(calls) != 1 {
		t.Fatal("blank ID must fail before HTTP", err, calls)
	}
}

func TestServiceGraphUpdatePutsDirtyAttributes(t *testing.T) {
	var calls []sgCall
	api := sgAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"service_graph":` + sgRow + `}` })
	v, err := api.Update(context.Background(), "x1", servicegraphs.WithUpdateField("description", "d"), servicegraphs.WithUpdateField("name", nil), servicegraphs.WithUpdateField("id", "ignored"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != sgBase+"/x1" || calls[0].body != `{"service_graph":{"description":"d","name":null}}` {
		t.Fatal(v, err, calls)
	}
	sgCheck(t, v)
	calls = nil
	v, err = api.Update(context.Background(), "x1", servicegraphs.WithUpdateField("unknown", 1))
	if err != nil || len(calls) != 0 || v.ID != "x1" || v.StatusCode != 0 {
		t.Fatal("nothing dirty must not send PUT", v, err, calls)
	}
	api = sgAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
	_, err = api.Update(context.Background(), "x1", servicegraphs.WithUpdateField("description", "x"))
	sgOperation(t, err, "Update")
	sgStatus(t, err, 409)
}

func TestServiceGraphDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, item := range []struct {
		code    int
		options []servicegraphs.DeleteOption
		fails   bool
	}{
		{204, nil, false}, {404, nil, false},
		{404, []servicegraphs.DeleteOption{servicegraphs.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		t.Run(fmt.Sprint(item.code, len(item.options)), func(t *testing.T) {
			var calls []sgCall
			api := sgAPI(t, &calls, func(*http.Request) (int, string) { return item.code, "" })
			err := api.Delete(context.Background(), "x1", item.options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != sgBase+"/x1" {
				t.Fatal(calls)
			}
			if !item.fails {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			sgOperation(t, err, "Delete")
			sgStatus(t, err, item.code)
		})
	}
}

func TestServiceGraphListQueryFiltersAndPaging(t *testing.T) {
	var calls []sgCall
	api := sgAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"service_graphs":[` + sgRow + `,{"id":"x2","name":"web","port_chains":{"c1":["c9"]}}]}`
	})
	values, err := api.All(context.Background(), servicegraphs.WithListName("web"),
		servicegraphs.WithListQuery("project_id", "p1"),
		servicegraphs.WithListQuery("port_chains", map[string]any{"c1": []string{"c2", "c3"}}), servicegraphs.WithListQuery("c1", "x"))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	sgCheck(t, values[0])
	if calls[0].path != sgBase || calls[0].query != "name=web&project_id=p1" || calls[0].accept != "application/json" {
		t.Fatal(calls)
	}

	calls = nil
	api = sgAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"service_graphs":[{"id":"a"}],"service_graphs_links":[{"rel":"next","href":"http://` + req.Host + sgBase + `?marker=a"}]}`
		case "a":
			return 200, `{"service_graphs":[{"id":"b"}],"service_graphs_links":[]}`
		}
		return 500, `{}`
	})
	values, err = api.All(context.Background())
	if err != nil || len(values) != 2 || len(calls) != 2 || calls[1].query != "marker=a" {
		t.Fatal("service_graphs_links next continuation", values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), servicegraphs.WithListMaxItems(1))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].query != "limit=1" {
		t.Fatal(values, err, calls)
	}
	calls = nil
	api = sgAPI(t, &calls, func(*http.Request) (int, string) { return 503, `{}` })
	_, err = api.All(context.Background())
	sgOperation(t, err, "List")
	sgStatus(t, err, 503)
}

func TestServiceGraphFindFallsBackToNameList(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []sgCall
			api := sgAPI(t, &calls, func(req *http.Request) (int, string) {
				if req.URL.Path == sgBase {
					return 200, `{"service_graphs":[{"id":"x1","name":"web"},{"id":"x2","name":"db"}]}`
				}
				return code, `{}`
			})
			v, err := api.Find(context.Background(), "db")
			if err != nil || v == nil || v.ID != "x2" || len(calls) != 2 || calls[0].path != sgBase+"/db" || calls[1].path != sgBase || calls[1].query != "name=db" {
				t.Fatal(v, err, calls)
			}
			calls = nil
			v, err = api.Find(context.Background(), "absent")
			if err != nil || v != nil || len(calls) != 2 {
				t.Fatal("ignore_missing default returns nil", v, err, calls)
			}
			_, err = api.Find(context.Background(), "absent", servicegraphs.WithFindIgnoreMissing(false))
			var missing *resource.NotFoundError
			if !errors.As(err, &missing) {
				t.Fatal(err)
			}
			sgOperation(t, err, "Find")
		})
	}
	var calls []sgCall
	api := sgAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"service_graph":` + sgRow + `}` })
	v, err := api.Find(context.Background(), "x1", servicegraphs.WithFindQuery("project_id", "p1"))
	if err != nil || len(calls) != 1 || calls[0].query != "project_id=p1" {
		t.Fatal(v, err, calls)
	}
	sgCheck(t, v)

	calls = nil
	api = sgAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == sgBase {
			return 200, `{"service_graphs":[{"id":"d1","name":"dup"},{"id":"d2","name":"dup"}]}`
		}
		return 404, `{}`
	})
	var ambiguous *resource.AmbiguousError
	if _, err := api.Find(context.Background(), "dup"); !errors.As(err, &ambiguous) {
		t.Fatal("duplicate match must fail", err)
	}
	calls = nil
	api = sgAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.Find(context.Background(), "x1")
	sgOperation(t, err, "Find")
	sgStatus(t, err, 500)
	if len(calls) != 1 {
		t.Fatal("500 must not fall back", calls)
	}
}

func TestServiceGraphRejectsInvalidInputBeforeHTTP(t *testing.T) {
	var calls []sgCall
	api := sgAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
	ctx := context.Background()
	checks := map[string]func() error{
		"nil create option": func() error { _, err := api.Create(ctx, nil); return err },
		"nil get option":    func() error { _, err := api.Get(ctx, "a", nil); return err },
		"nil update option": func() error { _, err := api.Update(ctx, "a", nil); return err },
		"nil delete option": func() error { return api.Delete(ctx, "a", nil) },
		"find id query":     func() error { _, err := api.Find(ctx, "a", servicegraphs.WithFindQuery("id", "x")); return err },
		"nil list option":   func() error { _, err := api.All(ctx, nil); return err },
		"negative limit":    func() error { _, err := api.All(ctx, servicegraphs.WithListLimit(-1)); return err },
		"nil API":           func() error { _, err := (*servicegraphs.API)(nil).Get(ctx, "a"); return err },
	}
	for name, check := range checks {
		if err := check(); err == nil {
			t.Fatal(name, "must fail")
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}
