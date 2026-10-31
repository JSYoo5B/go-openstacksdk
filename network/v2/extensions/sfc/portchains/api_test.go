package portchains_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pcCall struct{ method, path, query, body, accept string }

const pcBase = "/neutron/v2.0/sfc/port_chains"

func pcAPI(t *testing.T, calls *[]pcCall, reply func(*http.Request) (int, string)) *portchains.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pcCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return portchains.New(client)
}

func pcOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "port_chains" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func pcStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const pcRow = `{"id":"x1","name":"web","description":"d","project_id":"p1","tenant_id":"p1","port_pair_groups":["g1","g2"],"flow_classifiers":["fc1"],"chain_parameters":{"correlation":"mpls","symmetric":false},"extra":1}`

func pcCheck(t *testing.T, v *portchains.PortChain) {
	t.Helper()
	if v == nil || v.ID != "x1" || v.Name != "web" || v.Description != "d" || v.ProjectID != "p1" || v.TenantID != "p1" {
		t.Fatalf("decoded PortChain: %#v", v)
	}
	if len(v.PortPairGroups) != 2 || v.PortPairGroups[0] != "g1" || len(v.FlowClassifiers) != 1 || v.FlowClassifiers[0] != "fc1" || v.ChainParameters["correlation"] != "mpls" {
		t.Fatalf("decoded PortChain fields: %#v", v)
	}
	if len(v.Body) != 8 || v.Body["extra"] != nil || string(v.Wire["extra"]) != "1" {
		t.Fatalf("raw views: %#v", v.Body)
	}
}

func TestPortChainCreatePostsEnvelope(t *testing.T) {
	var calls []pcCall
	api := pcAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"port_chain":` + pcRow + `}` })
	v, err := api.Create(context.Background(),
		portchains.WithCreateField("name", "web"),
		portchains.WithCreateField("port_pair_groups", []string{"g1", "g2"}),
		portchains.WithCreateField("flow_classifiers", []string{"fc1"}),
		portchains.WithCreateField("chain_parameters", map[string]any{"symmetric": false}),
		portchains.WithCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pcBase || calls[0].query != "" || calls[0].body != `{"port_chain":{"chain_parameters":{"symmetric":false},"flow_classifiers":["fc1"],"name":"web","port_pair_groups":["g1","g2"]}}` {
		t.Fatal(calls)
	}
	pcCheck(t, v)
	if v.StatusCode != 201 {
		t.Fatal(v.StatusCode)
	}
	for _, code := range []int{200, 400, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []pcCall
			api := pcAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Create(context.Background(), portchains.WithCreateField("tenant_id", "p1"))
			if calls[0].body != `{"port_chain":{"tenant_id":"p1"}}` {
				t.Fatal(calls)
			}
			if code < 400 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			pcOperation(t, err, "Create")
			pcStatus(t, err, code)
		})
	}
}

func TestPortChainGetDecodesDescriptorsAndRejects(t *testing.T) {
	var calls []pcCall
	api := pcAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"port_chain":` + pcRow + `}` })
	v, err := api.Get(context.Background(), "x1")
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pcBase+"/x1" {
		t.Fatal(v, err, calls)
	}
	pcCheck(t, v)

	calls = nil
	api = pcAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"id":"x1","tenant_id":"t9","flow_classifiers":"fc1","chain_parameters":"x"}`
	})
	v, err = api.Get(context.Background(), "x/1")
	if err != nil || calls[0].path != pcBase+"/x%2F1" || v.ProjectID != "t9" || len(v.FlowClassifiers) != 1 || string(v.Body["chain_parameters"]) != "{}" {
		t.Fatal("Python descriptor conversion and the tenant_id alias", v, err, calls)
	}

	calls = nil
	api = pcAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{"NeutronError":{}}` })
	_, err = api.Get(context.Background(), "missing")
	pcOperation(t, err, "Get")
	pcStatus(t, err, 404)
	if _, err := api.Get(context.Background(), ""); err == nil || len(calls) != 1 {
		t.Fatal("blank ID must fail before HTTP", err, calls)
	}
}

func TestPortChainUpdatePutsDirtyAttributes(t *testing.T) {
	var calls []pcCall
	api := pcAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"port_chain":` + pcRow + `}` })
	v, err := api.Update(context.Background(), "x1", portchains.WithUpdateField("description", "d"), portchains.WithUpdateField("name", nil), portchains.WithUpdateField("id", "ignored"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pcBase+"/x1" || calls[0].body != `{"port_chain":{"description":"d","name":null}}` {
		t.Fatal(v, err, calls)
	}
	pcCheck(t, v)
	calls = nil
	v, err = api.Update(context.Background(), "x1", portchains.WithUpdateField("unknown", 1))
	if err != nil || len(calls) != 0 || v.ID != "x1" || v.StatusCode != 0 {
		t.Fatal("nothing dirty must not send PUT", v, err, calls)
	}
	api = pcAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
	_, err = api.Update(context.Background(), "x1", portchains.WithUpdateField("description", "x"))
	pcOperation(t, err, "Update")
	pcStatus(t, err, 409)
}

func TestPortChainDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, item := range []struct {
		code    int
		options []portchains.DeleteOption
		fails   bool
	}{
		{204, nil, false}, {404, nil, false},
		{404, []portchains.DeleteOption{portchains.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		t.Run(fmt.Sprint(item.code, len(item.options)), func(t *testing.T) {
			var calls []pcCall
			api := pcAPI(t, &calls, func(*http.Request) (int, string) { return item.code, "" })
			err := api.Delete(context.Background(), "x1", item.options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pcBase+"/x1" {
				t.Fatal(calls)
			}
			if !item.fails {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			pcOperation(t, err, "Delete")
			pcStatus(t, err, item.code)
		})
	}
}

func TestPortChainListQueryFiltersAndPaging(t *testing.T) {
	var calls []pcCall
	api := pcAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"port_chains":[` + pcRow + `,{"id":"x2","name":"web","flow_classifiers":["fc2"],"chain_parameters":{"symmetric":false}}]}`
	})
	values, err := api.All(context.Background(), portchains.WithListName("web"),
		portchains.WithListQuery("tenant_id", "p1"),
		portchains.WithListQuery("flow_classifiers", []string{"fc1"}), portchains.WithListQuery("symmetric", "x"))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	pcCheck(t, values[0])
	if calls[0].path != pcBase || calls[0].query != "name=web&tenant_id=p1" || calls[0].accept != "application/json" {
		t.Fatal(calls)
	}

	calls = nil
	api = pcAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"port_chains":[{"id":"a"}],"port_chains_links":[{"rel":"next","href":"http://` + req.Host + pcBase + `?marker=a"}]}`
		case "a":
			return 200, `{"port_chains":[{"id":"b"}],"port_chains_links":[]}`
		}
		return 500, `{}`
	})
	values, err = api.All(context.Background())
	if err != nil || len(values) != 2 || len(calls) != 2 || calls[1].query != "marker=a" {
		t.Fatal("port_chains_links next continuation", values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), portchains.WithListMaxItems(1))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].query != "limit=1" {
		t.Fatal(values, err, calls)
	}
	calls = nil
	api = pcAPI(t, &calls, func(*http.Request) (int, string) { return 503, `{}` })
	_, err = api.All(context.Background())
	pcOperation(t, err, "List")
	pcStatus(t, err, 503)
}

func TestPortChainFindFallsBackToNameList(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []pcCall
			api := pcAPI(t, &calls, func(req *http.Request) (int, string) {
				if req.URL.Path == pcBase {
					return 200, `{"port_chains":[{"id":"x1","name":"web"},{"id":"x2","name":"db"}]}`
				}
				return code, `{}`
			})
			v, err := api.Find(context.Background(), "db")
			if err != nil || v == nil || v.ID != "x2" || len(calls) != 2 || calls[0].path != pcBase+"/db" || calls[1].path != pcBase || calls[1].query != "name=db" {
				t.Fatal(v, err, calls)
			}
			calls = nil
			v, err = api.Find(context.Background(), "absent")
			if err != nil || v != nil || len(calls) != 2 {
				t.Fatal("ignore_missing default returns nil", v, err, calls)
			}
			_, err = api.Find(context.Background(), "absent", portchains.WithFindIgnoreMissing(false))
			var missing *resource.NotFoundError
			if !errors.As(err, &missing) {
				t.Fatal(err)
			}
			pcOperation(t, err, "Find")
		})
	}
	var calls []pcCall
	api := pcAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"port_chain":` + pcRow + `}` })
	v, err := api.Find(context.Background(), "x1", portchains.WithFindQuery("project_id", "p1"))
	if err != nil || len(calls) != 1 || calls[0].query != "project_id=p1" {
		t.Fatal(v, err, calls)
	}
	pcCheck(t, v)

	calls = nil
	api = pcAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == pcBase {
			return 200, `{"port_chains":[{"id":"d1","name":"dup"},{"id":"d2","name":"dup"}]}`
		}
		return 404, `{}`
	})
	var ambiguous *resource.AmbiguousError
	if _, err := api.Find(context.Background(), "dup"); !errors.As(err, &ambiguous) {
		t.Fatal("duplicate match must fail", err)
	}
	calls = nil
	api = pcAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.Find(context.Background(), "x1")
	pcOperation(t, err, "Find")
	pcStatus(t, err, 500)
	if len(calls) != 1 {
		t.Fatal("500 must not fall back", calls)
	}
}

func TestPortChainRejectsInvalidInputBeforeHTTP(t *testing.T) {
	var calls []pcCall
	api := pcAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
	ctx := context.Background()
	checks := map[string]func() error{
		"nil create option": func() error { _, err := api.Create(ctx, nil); return err },
		"nil get option":    func() error { _, err := api.Get(ctx, "a", nil); return err },
		"nil update option": func() error { _, err := api.Update(ctx, "a", nil); return err },
		"nil delete option": func() error { return api.Delete(ctx, "a", nil) },
		"find id query":     func() error { _, err := api.Find(ctx, "a", portchains.WithFindQuery("id", "x")); return err },
		"nil list option":   func() error { _, err := api.All(ctx, nil); return err },
		"negative limit":    func() error { _, err := api.All(ctx, portchains.WithListLimit(-1)); return err },
		"nil API":           func() error { _, err := (*portchains.API)(nil).Get(ctx, "a"); return err },
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
