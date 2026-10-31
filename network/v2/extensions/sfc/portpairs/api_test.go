package portpairs_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type ppCall struct{ method, path, query, body, accept string }

const ppBase = "/neutron/v2.0/sfc/port_pairs"

func ppAPI(t *testing.T, calls *[]ppCall, reply func(*http.Request) (int, string)) *portpairs.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, ppCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return portpairs.New(client)
}

func ppOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "port_pairs" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func ppStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const ppRow = `{"id":"x1","name":"web","description":"d","project_id":"p1","tenant_id":"p1","ingress":"in-1","egress":"out-1","service_function_parameters":{"correlation":null,"weight":1},"extra":1}`

func ppCheck(t *testing.T, v *portpairs.PortPair) {
	t.Helper()
	if v == nil || v.ID != "x1" || v.Name != "web" || v.Description != "d" || v.ProjectID != "p1" || v.TenantID != "p1" {
		t.Fatalf("decoded PortPair: %#v", v)
	}
	if v.Ingress == nil || *v.Ingress != "in-1" || v.Egress == nil || *v.Egress != "out-1" || v.ServiceFunctionParameters["weight"] != float64(1) {
		t.Fatalf("decoded PortPair fields: %#v", v)
	}
	if len(v.Body) != 8 || v.Body["extra"] != nil || string(v.Wire["extra"]) != "1" {
		t.Fatalf("raw views: %#v", v.Body)
	}
}

func TestPortPairCreatePostsEnvelope(t *testing.T) {
	var calls []ppCall
	api := ppAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"port_pair":` + ppRow + `}` })
	v, err := api.Create(context.Background(),
		portpairs.WithCreateField("name", "web"),
		portpairs.WithCreateField("ingress", "in-1"),
		portpairs.WithCreateField("egress", "out-1"),
		portpairs.WithCreateField("service_function_parameters", map[string]any{"weight": 1}),
		portpairs.WithCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != ppBase || calls[0].query != "" || calls[0].body != `{"port_pair":{"egress":"out-1","ingress":"in-1","name":"web","service_function_parameters":{"weight":1}}}` {
		t.Fatal(calls)
	}
	ppCheck(t, v)
	if v.StatusCode != 201 {
		t.Fatal(v.StatusCode)
	}
	for _, code := range []int{200, 400, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []ppCall
			api := ppAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Create(context.Background(), portpairs.WithCreateField("tenant_id", "p1"))
			if calls[0].body != `{"port_pair":{"tenant_id":"p1"}}` {
				t.Fatal(calls)
			}
			if code < 400 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			ppOperation(t, err, "Create")
			ppStatus(t, err, code)
		})
	}
}

func TestPortPairGetDecodesDescriptorsAndRejects(t *testing.T) {
	var calls []ppCall
	api := ppAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"port_pair":` + ppRow + `}` })
	v, err := api.Get(context.Background(), "x1")
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != ppBase+"/x1" {
		t.Fatal(v, err, calls)
	}
	ppCheck(t, v)

	calls = nil
	api = ppAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"id":"x1","tenant_id":"t9","service_function_parameters":[1]}`
	})
	v, err = api.Get(context.Background(), "x/1")
	if err != nil || calls[0].path != ppBase+"/x%2F1" || v.ProjectID != "t9" || string(v.Body["service_function_parameters"]) != "{}" || len(v.ServiceFunctionParameters) != 0 {
		t.Fatal("Python descriptor conversion and the tenant_id alias", v, err, calls)
	}

	calls = nil
	api = ppAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{"NeutronError":{}}` })
	_, err = api.Get(context.Background(), "missing")
	ppOperation(t, err, "Get")
	ppStatus(t, err, 404)
	if _, err := api.Get(context.Background(), ""); err == nil || len(calls) != 1 {
		t.Fatal("blank ID must fail before HTTP", err, calls)
	}
}

func TestPortPairUpdatePutsDirtyAttributes(t *testing.T) {
	var calls []ppCall
	api := ppAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"port_pair":` + ppRow + `}` })
	v, err := api.Update(context.Background(), "x1", portpairs.WithUpdateField("description", "d"), portpairs.WithUpdateField("name", nil), portpairs.WithUpdateField("id", "ignored"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != ppBase+"/x1" || calls[0].body != `{"port_pair":{"description":"d","name":null}}` {
		t.Fatal(v, err, calls)
	}
	ppCheck(t, v)
	calls = nil
	v, err = api.Update(context.Background(), "x1", portpairs.WithUpdateField("unknown", 1))
	if err != nil || len(calls) != 0 || v.ID != "x1" || v.StatusCode != 0 {
		t.Fatal("nothing dirty must not send PUT", v, err, calls)
	}
	api = ppAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
	_, err = api.Update(context.Background(), "x1", portpairs.WithUpdateField("description", "x"))
	ppOperation(t, err, "Update")
	ppStatus(t, err, 409)
}

func TestPortPairDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, item := range []struct {
		code    int
		options []portpairs.DeleteOption
		fails   bool
	}{
		{204, nil, false}, {404, nil, false},
		{404, []portpairs.DeleteOption{portpairs.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		t.Run(fmt.Sprint(item.code, len(item.options)), func(t *testing.T) {
			var calls []ppCall
			api := ppAPI(t, &calls, func(*http.Request) (int, string) { return item.code, "" })
			err := api.Delete(context.Background(), "x1", item.options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != ppBase+"/x1" {
				t.Fatal(calls)
			}
			if !item.fails {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			ppOperation(t, err, "Delete")
			ppStatus(t, err, item.code)
		})
	}
}

func TestPortPairListQueryFiltersAndPaging(t *testing.T) {
	var calls []ppCall
	api := ppAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"port_pairs":[` + ppRow + `,{"id":"x2","name":"web","service_function_parameters":{"weight":2}}]}`
	})
	values, err := api.All(context.Background(), portpairs.WithListName("web"),
		portpairs.WithListQuery("ingress", "in-1"),
		portpairs.WithListQuery("egress", "out-1"),
		portpairs.WithListQuery("service_function_parameters", map[string]any{"weight": 1}), portpairs.WithListQuery("weight", "x"))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	ppCheck(t, values[0])
	if calls[0].path != ppBase || calls[0].query != "egress=out-1&ingress=in-1&name=web" || calls[0].accept != "application/json" {
		t.Fatal(calls)
	}

	calls = nil
	api = ppAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"port_pairs":[{"id":"a"}],"port_pairs_links":[{"rel":"next","href":"http://` + req.Host + ppBase + `?marker=a"}]}`
		case "a":
			return 200, `{"port_pairs":[{"id":"b"}],"port_pairs_links":[]}`
		}
		return 500, `{}`
	})
	values, err = api.All(context.Background())
	if err != nil || len(values) != 2 || len(calls) != 2 || calls[1].query != "marker=a" {
		t.Fatal("port_pairs_links next continuation", values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), portpairs.WithListMaxItems(1))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].query != "limit=1" {
		t.Fatal(values, err, calls)
	}
	calls = nil
	api = ppAPI(t, &calls, func(*http.Request) (int, string) { return 503, `{}` })
	_, err = api.All(context.Background())
	ppOperation(t, err, "List")
	ppStatus(t, err, 503)
}

func TestPortPairFindFallsBackToNameList(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []ppCall
			api := ppAPI(t, &calls, func(req *http.Request) (int, string) {
				if req.URL.Path == ppBase {
					return 200, `{"port_pairs":[{"id":"x1","name":"web"},{"id":"x2","name":"db"}]}`
				}
				return code, `{}`
			})
			v, err := api.Find(context.Background(), "db")
			if err != nil || v == nil || v.ID != "x2" || len(calls) != 2 || calls[0].path != ppBase+"/db" || calls[1].path != ppBase || calls[1].query != "name=db" {
				t.Fatal(v, err, calls)
			}
			calls = nil
			v, err = api.Find(context.Background(), "absent")
			if err != nil || v != nil || len(calls) != 2 {
				t.Fatal("ignore_missing default returns nil", v, err, calls)
			}
			_, err = api.Find(context.Background(), "absent", portpairs.WithFindIgnoreMissing(false))
			var missing *resource.NotFoundError
			if !errors.As(err, &missing) {
				t.Fatal(err)
			}
			ppOperation(t, err, "Find")
		})
	}
	var calls []ppCall
	api := ppAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"port_pair":` + ppRow + `}` })
	v, err := api.Find(context.Background(), "x1", portpairs.WithFindQuery("project_id", "p1"))
	if err != nil || len(calls) != 1 || calls[0].query != "project_id=p1" {
		t.Fatal(v, err, calls)
	}
	ppCheck(t, v)

	calls = nil
	api = ppAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == ppBase {
			return 200, `{"port_pairs":[{"id":"d1","name":"dup"},{"id":"d2","name":"dup"}]}`
		}
		return 404, `{}`
	})
	var ambiguous *resource.AmbiguousError
	if _, err := api.Find(context.Background(), "dup"); !errors.As(err, &ambiguous) {
		t.Fatal("duplicate match must fail", err)
	}
	calls = nil
	api = ppAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.Find(context.Background(), "x1")
	ppOperation(t, err, "Find")
	ppStatus(t, err, 500)
	if len(calls) != 1 {
		t.Fatal("500 must not fall back", calls)
	}
}

func TestPortPairRejectsInvalidInputBeforeHTTP(t *testing.T) {
	var calls []ppCall
	api := ppAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
	ctx := context.Background()
	checks := map[string]func() error{
		"nil create option": func() error { _, err := api.Create(ctx, nil); return err },
		"nil get option":    func() error { _, err := api.Get(ctx, "a", nil); return err },
		"nil update option": func() error { _, err := api.Update(ctx, "a", nil); return err },
		"nil delete option": func() error { return api.Delete(ctx, "a", nil) },
		"find id query":     func() error { _, err := api.Find(ctx, "a", portpairs.WithFindQuery("id", "x")); return err },
		"nil list option":   func() error { _, err := api.All(ctx, nil); return err },
		"negative limit":    func() error { _, err := api.All(ctx, portpairs.WithListLimit(-1)); return err },
		"nil API":           func() error { _, err := (*portpairs.API)(nil).Get(ctx, "a"); return err },
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
