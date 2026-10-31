package portpairgroups_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type ppgCall struct{ method, path, query, body, accept string }

const ppgBase = "/neutron/v2.0/sfc/port_pair_groups"

func ppgAPI(t *testing.T, calls *[]ppgCall, reply func(*http.Request) (int, string)) *portpairgroups.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, ppgCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return portpairgroups.New(client)
}

func ppgOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "port_pair_groups" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func ppgStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const ppgRow = `{"id":"x1","name":"web","description":"d","project_id":"p1","tenant_id":"p1","port_pairs":["pp1","pp2"],"port_pair_group_parameters":{"lb_fields":["ip_src"]},"tap_enabled":false,"extra":1}`

func ppgCheck(t *testing.T, v *portpairgroups.PortPairGroup) {
	t.Helper()
	if v == nil || v.ID != "x1" || v.Name != "web" || v.Description != "d" || v.ProjectID != "p1" || v.TenantID != "p1" {
		t.Fatalf("decoded PortPairGroup: %#v", v)
	}
	if len(v.PortPairs) != 2 || v.PortPairs[1] != "pp2" || v.PortPairGroupParameters["lb_fields"] == nil || v.TapEnabled == nil || *v.TapEnabled {
		t.Fatalf("decoded PortPairGroup fields: %#v", v)
	}
	if len(v.Body) != 8 || v.Body["extra"] != nil || string(v.Wire["extra"]) != "1" {
		t.Fatalf("raw views: %#v", v.Body)
	}
}

func TestPortPairGroupCreatePostsEnvelope(t *testing.T) {
	var calls []ppgCall
	api := ppgAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"port_pair_group":` + ppgRow + `}` })
	v, err := api.Create(context.Background(),
		portpairgroups.WithCreateField("name", "web"),
		portpairgroups.WithCreateField("port_pairs", []string{"pp1", "pp2"}),
		portpairgroups.WithCreateField("is_tap_enabled", false),
		portpairgroups.WithCreateField("port_pair_group_parameters", map[string]any{"lb_fields": []string{"ip_src"}}),
		portpairgroups.WithCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != ppgBase || calls[0].query != "" || calls[0].body != `{"port_pair_group":{"name":"web","port_pair_group_parameters":{"lb_fields":["ip_src"]},"port_pairs":["pp1","pp2"],"tap_enabled":false}}` {
		t.Fatal(calls)
	}
	ppgCheck(t, v)
	if v.StatusCode != 201 {
		t.Fatal(v.StatusCode)
	}
	for _, code := range []int{200, 400, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []ppgCall
			api := ppgAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Create(context.Background(), portpairgroups.WithCreateField("tenant_id", "p1"))
			if calls[0].body != `{"port_pair_group":{"tenant_id":"p1"}}` {
				t.Fatal(calls)
			}
			if code < 400 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			ppgOperation(t, err, "Create")
			ppgStatus(t, err, code)
		})
	}
}

func TestPortPairGroupGetDecodesDescriptorsAndRejects(t *testing.T) {
	var calls []ppgCall
	api := ppgAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"port_pair_group":` + ppgRow + `}` })
	v, err := api.Get(context.Background(), "x1")
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != ppgBase+"/x1" {
		t.Fatal(v, err, calls)
	}
	ppgCheck(t, v)

	calls = nil
	api = ppgAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"id":"x1","tenant_id":"t9","port_pairs":"pp1","tap_enabled":"no"}`
	})
	v, err = api.Get(context.Background(), "x/1")
	if err != nil || calls[0].path != ppgBase+"/x%2F1" || v.ProjectID != "t9" || len(v.PortPairs) != 1 || v.PortPairs[0] != "pp1" || v.TapEnabled == nil || !*v.TapEnabled {
		t.Fatal("Python descriptor conversion and the tenant_id alias", v, err, calls)
	}

	calls = nil
	api = ppgAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{"NeutronError":{}}` })
	_, err = api.Get(context.Background(), "missing")
	ppgOperation(t, err, "Get")
	ppgStatus(t, err, 404)
	if _, err := api.Get(context.Background(), ""); err == nil || len(calls) != 1 {
		t.Fatal("blank ID must fail before HTTP", err, calls)
	}
}

func TestPortPairGroupUpdatePutsDirtyAttributes(t *testing.T) {
	var calls []ppgCall
	api := ppgAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"port_pair_group":` + ppgRow + `}` })
	v, err := api.Update(context.Background(), "x1", portpairgroups.WithUpdateField("description", "d"), portpairgroups.WithUpdateField("name", nil), portpairgroups.WithUpdateField("id", "ignored"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != ppgBase+"/x1" || calls[0].body != `{"port_pair_group":{"description":"d","name":null}}` {
		t.Fatal(v, err, calls)
	}
	ppgCheck(t, v)
	calls = nil
	v, err = api.Update(context.Background(), "x1", portpairgroups.WithUpdateField("unknown", 1))
	if err != nil || len(calls) != 0 || v.ID != "x1" || v.StatusCode != 0 {
		t.Fatal("nothing dirty must not send PUT", v, err, calls)
	}
	api = ppgAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
	_, err = api.Update(context.Background(), "x1", portpairgroups.WithUpdateField("description", "x"))
	ppgOperation(t, err, "Update")
	ppgStatus(t, err, 409)
}

func TestPortPairGroupDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, item := range []struct {
		code    int
		options []portpairgroups.DeleteOption
		fails   bool
	}{
		{204, nil, false}, {404, nil, false},
		{404, []portpairgroups.DeleteOption{portpairgroups.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		t.Run(fmt.Sprint(item.code, len(item.options)), func(t *testing.T) {
			var calls []ppgCall
			api := ppgAPI(t, &calls, func(*http.Request) (int, string) { return item.code, "" })
			err := api.Delete(context.Background(), "x1", item.options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != ppgBase+"/x1" {
				t.Fatal(calls)
			}
			if !item.fails {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			ppgOperation(t, err, "Delete")
			ppgStatus(t, err, item.code)
		})
	}
}

func TestPortPairGroupListQueryFiltersAndPaging(t *testing.T) {
	var calls []ppgCall
	api := ppgAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"port_pair_groups":[` + ppgRow + `,{"id":"x2","name":"web","tap_enabled":true,"port_pairs":["pp1","pp2"]}]}`
	})
	values, err := api.All(context.Background(), portpairgroups.WithListName("web"),
		portpairgroups.WithListQuery("description", "d"),
		portpairgroups.WithListQuery("is_tap_enabled", false), portpairgroups.WithListQuery("tap_enabled", "x"))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	ppgCheck(t, values[0])
	if calls[0].path != ppgBase || calls[0].query != "description=d&name=web" || calls[0].accept != "application/json" {
		t.Fatal(calls)
	}

	calls = nil
	api = ppgAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"port_pair_groups":[{"id":"a"}],"port_pair_groups_links":[{"rel":"next","href":"http://` + req.Host + ppgBase + `?marker=a"}]}`
		case "a":
			return 200, `{"port_pair_groups":[{"id":"b"}],"port_pair_groups_links":[]}`
		}
		return 500, `{}`
	})
	values, err = api.All(context.Background())
	if err != nil || len(values) != 2 || len(calls) != 2 || calls[1].query != "marker=a" {
		t.Fatal("port_pair_groups_links next continuation", values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), portpairgroups.WithListMaxItems(1))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].query != "limit=1" {
		t.Fatal(values, err, calls)
	}
	calls = nil
	api = ppgAPI(t, &calls, func(*http.Request) (int, string) { return 503, `{}` })
	_, err = api.All(context.Background())
	ppgOperation(t, err, "List")
	ppgStatus(t, err, 503)
}

func TestPortPairGroupFindFallsBackToNameList(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []ppgCall
			api := ppgAPI(t, &calls, func(req *http.Request) (int, string) {
				if req.URL.Path == ppgBase {
					return 200, `{"port_pair_groups":[{"id":"x1","name":"web"},{"id":"x2","name":"db"}]}`
				}
				return code, `{}`
			})
			v, err := api.Find(context.Background(), "db")
			if err != nil || v == nil || v.ID != "x2" || len(calls) != 2 || calls[0].path != ppgBase+"/db" || calls[1].path != ppgBase || calls[1].query != "name=db" {
				t.Fatal(v, err, calls)
			}
			calls = nil
			v, err = api.Find(context.Background(), "absent")
			if err != nil || v != nil || len(calls) != 2 {
				t.Fatal("ignore_missing default returns nil", v, err, calls)
			}
			_, err = api.Find(context.Background(), "absent", portpairgroups.WithFindIgnoreMissing(false))
			var missing *resource.NotFoundError
			if !errors.As(err, &missing) {
				t.Fatal(err)
			}
			ppgOperation(t, err, "Find")
		})
	}
	var calls []ppgCall
	api := ppgAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"port_pair_group":` + ppgRow + `}` })
	v, err := api.Find(context.Background(), "x1", portpairgroups.WithFindQuery("project_id", "p1"))
	if err != nil || len(calls) != 1 || calls[0].query != "project_id=p1" {
		t.Fatal(v, err, calls)
	}
	ppgCheck(t, v)

	calls = nil
	api = ppgAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == ppgBase {
			return 200, `{"port_pair_groups":[{"id":"d1","name":"dup"},{"id":"d2","name":"dup"}]}`
		}
		return 404, `{}`
	})
	var ambiguous *resource.AmbiguousError
	if _, err := api.Find(context.Background(), "dup"); !errors.As(err, &ambiguous) {
		t.Fatal("duplicate match must fail", err)
	}
	calls = nil
	api = ppgAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.Find(context.Background(), "x1")
	ppgOperation(t, err, "Find")
	ppgStatus(t, err, 500)
	if len(calls) != 1 {
		t.Fatal("500 must not fall back", calls)
	}
}

func TestPortPairGroupRejectsInvalidInputBeforeHTTP(t *testing.T) {
	var calls []ppgCall
	api := ppgAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
	ctx := context.Background()
	checks := map[string]func() error{
		"nil create option": func() error { _, err := api.Create(ctx, nil); return err },
		"nil get option":    func() error { _, err := api.Get(ctx, "a", nil); return err },
		"nil update option": func() error { _, err := api.Update(ctx, "a", nil); return err },
		"nil delete option": func() error { return api.Delete(ctx, "a", nil) },
		"find id query":     func() error { _, err := api.Find(ctx, "a", portpairgroups.WithFindQuery("id", "x")); return err },
		"nil list option":   func() error { _, err := api.All(ctx, nil); return err },
		"negative limit":    func() error { _, err := api.All(ctx, portpairgroups.WithListLimit(-1)); return err },
		"nil API":           func() error { _, err := (*portpairgroups.API)(nil).Get(ctx, "a"); return err },
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
