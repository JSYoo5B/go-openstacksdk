package flowclassifiers_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/flowclassifiers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type fcCall struct{ method, path, query, body, accept, contentType, version string }

const fcBase = "/neutron/v2.0/sfc/flow_classifiers"

func fcAPI(t *testing.T, calls *[]fcCall, reply func(*http.Request) (int, string)) (*flowclassifiers.API, *gophercloud.ServiceClient) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, fcCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept"), req.Header.Get("Content-Type"), req.Header.Get("OpenStack-API-Version")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"req-1"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return flowclassifiers.New(client), client
}

func fcOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "flow_classifiers" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func fcStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const fcRow = `{"id":"fc1","name":"web","description":"d","project_id":"p1","tenant_id":"p1","ethertype":"IPv4","protocol":"tcp","source_port_range_min":22,"source_port_range_max":"80","destination_port_range_min":null,"destination_port_range_max":443.9,"source_ip_prefix":"10.0.0.0/24","destination_ip_prefix":null,"logical_source_port":"port-a","logical_destination_port":null,"l7_parameters":{"url":"x"},"summary":"ignored","extra":1}`

func fcCheck(t *testing.T, v *flowclassifiers.FlowClassifier) {
	t.Helper()
	if v == nil || v.ID != "fc1" || v.Name != "web" || v.Description != "d" || v.ProjectID != "p1" || v.TenantID != "p1" || *v.Ethertype != "IPv4" || *v.Protocol != "tcp" {
		t.Fatalf("decoded flow classifier: %#v", v)
	}
	if *v.SourcePortRangeMin != 22 || *v.SourcePortRangeMax != 80 || v.DestinationPortRangeMin != nil || *v.DestinationPortRangeMax != 443 {
		t.Fatalf("Python int descriptors: %#v", v)
	}
	if *v.SourceIPPrefix != "10.0.0.0/24" || v.DestinationIPPrefix != nil || *v.LogicalSourcePort != "port-a" || v.LogicalDestinationPort != nil || v.L7Parameters["url"] != "x" {
		t.Fatalf("decoded flow classifier: %#v", v)
	}
	if len(v.Body) != 16 || string(v.Body["source_port_range_max"]) != "80" || v.Body["summary"] != nil || v.Body["extra"] != nil || string(v.Wire["extra"]) != "1" {
		t.Fatalf("raw views: %#v", v.Body)
	}
}

func TestFlowClassifierCreatePostsEnvelope(t *testing.T) {
	var calls []fcCall
	api, _ := fcAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"flow_classifier":` + fcRow + `}`
	})
	v, err := api.Create(context.Background(),
		flowclassifiers.WithCreateField("name", "web"),
		flowclassifiers.WithCreateField("source_port_range_min", 22),
		flowclassifiers.WithCreateField("l7_parameters", map[string]string{"url": "x"}),
		flowclassifiers.WithCreateField("project_id", "p1"),
		flowclassifiers.WithCreateField("summary", "computed"),
		flowclassifiers.WithCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"flow_classifier":{"l7_parameters":{"url":"x"},"name":"web","project_id":"p1","source_port_range_min":22}}`
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != fcBase || calls[0].query != "" || calls[0].body != want || calls[0].contentType != "application/json" || calls[0].version != "" {
		t.Fatal(calls)
	}
	fcCheck(t, v)
	if v.StatusCode != 201 || v.Header.Get("X-Request-Id") != "req-1" {
		t.Fatal(v.StatusCode, v.Header)
	}

	calls = nil
	api, _ = fcAPI(t, &calls, func(*http.Request) (int, string) { return 201, `` })
	v, err = api.Create(context.Background(), flowclassifiers.WithCreateField("id", "chosen"), flowclassifiers.WithCreateField("tenant_id", "t1"), flowclassifiers.WithCreateField("logical_source_port", nil))
	if err != nil || calls[0].body != `{"flow_classifier":{"id":"chosen","logical_source_port":null,"tenant_id":"t1"}}` {
		t.Fatal(v, err, calls)
	}
	if v.ID != "chosen" || v.TenantID != "t1" || v.ProjectID != "t1" || v.LogicalSourcePort != nil || v.Wire != nil {
		t.Fatalf("seeded create result with the tenant_id alias: %#v", v)
	}

	for _, code := range []int{200, 202, 399, 400, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []fcCall
			api, _ := fcAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Create(context.Background())
			if code < 400 {
				if err != nil || calls[0].body != `{"flow_classifier":{}}` {
					t.Fatal(err, calls)
				}
				return
			}
			fcOperation(t, err, "Create")
			fcStatus(t, err, code)
		})
	}
}

func TestFlowClassifierGetDecodesAndRejects(t *testing.T) {
	for _, item := range []struct {
		name, body string
		valid      bool
	}{{"envelope", `{"flow_classifier":` + fcRow + `}`, true}, {"root", fcRow, true}, {"array", `[]`, false}, {"null member", `{"flow_classifier":null}`, false}} {
		t.Run(item.name, func(t *testing.T) {
			var calls []fcCall
			api, _ := fcAPI(t, &calls, func(*http.Request) (int, string) { return 200, item.body })
			v, err := api.Get(context.Background(), "fc1")
			if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != fcBase+"/fc1" || calls[0].query != "" {
				t.Fatal(calls)
			}
			if !item.valid {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 {
					t.Fatal(v, err)
				}
				fcOperation(t, err, "Get")
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			fcCheck(t, v)
		})
	}
	var calls []fcCall
	api, _ := fcAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"flow_classifier":{"id":"fc1","tenant_id":"t9","l7_parameters":"bad","source_port_range_min":"x"}}`
	})
	v, err := api.Get(context.Background(), "a/b")
	if err != nil || calls[0].path != fcBase+"/a%2Fb" || v.ProjectID != "t9" || string(v.Body["l7_parameters"]) != "{}" || *v.SourcePortRangeMin != 0 {
		t.Fatal("tenant_id alias and Python descriptor fallbacks", v, err, calls)
	}
	calls = nil
	api, _ = fcAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{"NeutronError":{}}` })
	if _, err := api.Get(context.Background(), "missing"); err == nil {
		t.Fatal("404 must fail")
	} else {
		fcOperation(t, err, "Get")
		fcStatus(t, err, 404)
	}
	if _, err := api.Get(context.Background(), " "); err == nil || len(calls) != 1 {
		t.Fatal("blank ID must fail before HTTP", err, calls)
	}
}

func TestFlowClassifierUpdatePutsDirtyAttributes(t *testing.T) {
	var calls []fcCall
	api, _ := fcAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"flow_classifier":` + fcRow + `}`
	})
	v, err := api.Update(context.Background(), "fc1",
		flowclassifiers.WithUpdateField("description", "d"),
		flowclassifiers.WithUpdateField("name", nil),
		flowclassifiers.WithUpdateField("id", "ignored"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != fcBase+"/fc1" || calls[0].body != `{"flow_classifier":{"description":"d","name":null}}` {
		t.Fatal(calls)
	}
	fcCheck(t, v)

	calls = nil
	v, err = api.Update(context.Background(), "fc1", flowclassifiers.WithUpdateField("summary", "x"), flowclassifiers.WithUpdateField("unknown", 1))
	if err != nil || len(calls) != 0 || v.ID != "fc1" || v.StatusCode != 0 {
		t.Fatal("nothing dirty must not send PUT", v, err, calls)
	}
	api, _ = fcAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
	_, err = api.Update(context.Background(), "fc1", flowclassifiers.WithUpdateField("description", "x"))
	fcOperation(t, err, "Update")
	fcStatus(t, err, 409)
}

func TestFlowClassifierDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, item := range []struct {
		code    int
		options []flowclassifiers.DeleteOption
		fails   bool
	}{
		{204, nil, false}, {202, nil, false}, {404, nil, false},
		{404, []flowclassifiers.DeleteOption{flowclassifiers.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		t.Run(fmt.Sprint(item.code, len(item.options)), func(t *testing.T) {
			var calls []fcCall
			api, _ := fcAPI(t, &calls, func(*http.Request) (int, string) { return item.code, "" })
			err := api.Delete(context.Background(), "fc1", item.options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != fcBase+"/fc1" || calls[0].body != "" {
				t.Fatal(calls)
			}
			if !item.fails {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			fcOperation(t, err, "Delete")
			fcStatus(t, err, item.code)
		})
	}
}

func TestFlowClassifierListQueryFiltersAndPaging(t *testing.T) {
	var calls []fcCall
	api, _ := fcAPI(t, &calls, func(req *http.Request) (int, string) {
		return 200, `{"flow_classifiers":[` + fcRow + `,{"id":"fc2","source_ip_prefix":"10.9.0.0/24","l7_parameters":{}}],"flow_classifiers_links":[]}`
	})
	values, err := api.All(context.Background(), flowclassifiers.WithListName("web"), flowclassifiers.WithListProjectID("p1"),
		flowclassifiers.WithListQuery("protocol", "tcp"), flowclassifiers.WithListQuery("source_port_range_min", 22),
		flowclassifiers.WithListQuery("tenant_id", "p1"), flowclassifiers.WithListQuery("source_ip_prefix", "10.0.0.0/24"),
		flowclassifiers.WithListQuery("summary", "x"), flowclassifiers.WithListQuery("unknown", make(chan int)))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	fcCheck(t, values[0])
	if calls[0].path != fcBase || calls[0].query != "name=web&project_id=p1&protocol=tcp&source_port_range_min=22&tenant_id=p1" || calls[0].accept != "application/json" {
		t.Fatal(calls)
	}
	calls = nil
	values, err = api.All(context.Background(), flowclassifiers.WithListQuery("l7_parameters", map[string]string{"url": "x"}), flowclassifiers.WithListQuery("id", "fc1"))
	if err != nil || len(values) != 1 || values[0].ID != "fc1" || calls[0].query != "" {
		t.Fatal("dict and id filters run locally", values, err, calls)
	}

	calls = nil
	api, _ = fcAPI(t, &calls, func(req *http.Request) (int, string) {
		origin := "http://" + req.Host
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"flow_classifiers":[{"id":"a"}],"flow_classifiers_links":[{"rel":"next","href":"` + origin + fcBase + `?limit=1&marker=a"}]}`
		case "a":
			return 200, `{"flow_classifiers":[{"id":"b"}],"flow_classifiers_links":[{"rel":"previous","href":"x"}]}`
		case "b":
			return 200, `{"flow_classifiers":[]}`
		}
		return 500, `{}`
	})
	values, err = api.All(context.Background(), flowclassifiers.WithListLimit(1))
	if err != nil || len(values) != 2 || len(calls) != 3 || calls[0].query != "limit=1" || calls[1].query != "limit=1&marker=a" || calls[2].query != "limit=1&marker=b" {
		t.Fatal("next link then limit-driven marker continuation until an empty page", values, err, calls)
	}
	calls = nil
	api, _ = fcAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"flow_classifiers":[{"id":"a"}],"flow_classifiers_links":[{"rel":"next","href":"/v2.0/sfc/flow_classifiers?marker=a"}]}`
		}
		return 200, `{"flow_classifiers":[{"id":"b"}]}`
	})
	values, err = api.All(context.Background())
	if err != nil || len(values) != 2 || len(calls) != 2 || calls[1].path != fcBase || calls[1].query != "marker=a" {
		t.Fatal("a version-prefixed relative next link resolves below the endpoint", values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), flowclassifiers.WithListLimit(1), flowclassifiers.WithListPaginated(false))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), flowclassifiers.WithListMaxItems(1))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].query != "limit=1" {
		t.Fatal(values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), flowclassifiers.WithListMarker("a"))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].query != "marker=a" {
		t.Fatal(values, err, calls)
	}

	calls = nil
	api, _ = fcAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.All(context.Background())
	fcOperation(t, err, "List")
	fcStatus(t, err, 500)
}

func TestFlowClassifierFindFallsBackToNameList(t *testing.T) {
	list := `{"flow_classifiers":[{"id":"fc1","name":"web"},{"id":"fc2","name":"db"}]}`
	for _, code := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []fcCall
			api, _ := fcAPI(t, &calls, func(req *http.Request) (int, string) {
				if req.URL.Path == fcBase {
					return 200, list
				}
				return code, `{}`
			})
			v, err := api.Find(context.Background(), "db")
			if err != nil || v == nil || v.ID != "fc2" || len(calls) != 2 || calls[0].path != fcBase+"/db" || calls[1].path != fcBase || calls[1].query != "name=db" || calls[1].accept != "application/json" {
				t.Fatal(v, err, calls)
			}
			calls = nil
			v, err = api.Find(context.Background(), "fc1")
			if err != nil || v == nil || v.Name != "web" || calls[1].query != "name=fc1" {
				t.Fatal("list rows also match by exact ID", v, err, calls)
			}
			calls = nil
			v, err = api.Find(context.Background(), "absent")
			if err != nil || v != nil || len(calls) != 2 {
				t.Fatal("ignore_missing default returns nil", v, err, calls)
			}
			_, err = api.Find(context.Background(), "absent", flowclassifiers.WithFindIgnoreMissing(false))
			var missing *resource.NotFoundError
			if !errors.As(err, &missing) {
				t.Fatal(err)
			}
			fcOperation(t, err, "Find")
		})
	}
	var calls []fcCall
	api, _ := fcAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"flow_classifier":` + fcRow + `}` })
	v, err := api.Find(context.Background(), "fc1")
	if err != nil || len(calls) != 1 || calls[0].query != "" {
		t.Fatal(v, err, calls)
	}
	fcCheck(t, v)

	calls = nil
	api, _ = fcAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == fcBase {
			return 200, `{"flow_classifiers":[{"id":"fc7","name":"web","project_id":"p1","source_ip_prefix":"10.0.0.0/24"},{"id":"fc8","name":"web","source_ip_prefix":"other"}]}`
		}
		return 404, `{}`
	})
	v, err = api.Find(context.Background(), "web", flowclassifiers.WithFindQuery("project_id", "p1"), flowclassifiers.WithFindQuery("source_ip_prefix", "10.0.0.0/24"), flowclassifiers.WithFindQuery("all_projects", true))
	if err != nil || v == nil || v.ID != "fc7" || len(calls) != 2 {
		t.Fatal(v, err, calls)
	}
	if calls[0].query != "project_id=p1&source_ip_prefix=10.0.0.0%2F24" || calls[1].query != "name=web&project_id=p1" {
		t.Fatal("find query goes to the member GET and the list fallback", calls)
	}

	calls = nil
	api, _ = fcAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == fcBase {
			return 200, `{"flow_classifiers":[{"id":"x1","name":"dup"},{"id":"x2","name":"dup"}]}`
		}
		return 404, `{}`
	})
	var ambiguous *resource.AmbiguousError
	if _, err := api.Find(context.Background(), "dup"); !errors.As(err, &ambiguous) {
		t.Fatal("duplicate match must fail", err)
	}
	calls = nil
	api, _ = fcAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.Find(context.Background(), "fc1")
	fcOperation(t, err, "Find")
	fcStatus(t, err, 500)
	if len(calls) != 1 {
		t.Fatal("500 must not fall back", calls)
	}
}

func TestFlowClassifierRejectsInvalidInputBeforeHTTP(t *testing.T) {
	var calls []fcCall
	api, client := fcAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
	ctx := context.Background()
	checks := map[string]func() error{
		"nil get option":     func() error { _, err := api.Get(ctx, "a", nil); return err },
		"nil create option":  func() error { _, err := api.Create(ctx, nil); return err },
		"nil update option":  func() error { _, err := api.Update(ctx, "a", nil); return err },
		"blank update ID":    func() error { _, err := api.Update(ctx, ""); return err },
		"nil delete option":  func() error { return api.Delete(ctx, "a", nil) },
		"dot delete ID":      func() error { return api.Delete(ctx, "..") },
		"nil find option":    func() error { _, err := api.Find(ctx, "a", nil); return err },
		"find id query":      func() error { _, err := api.Find(ctx, "a", flowclassifiers.WithFindQuery("id", "x")); return err },
		"find control query": func() error { _, err := api.Find(ctx, "a", flowclassifiers.WithFindQuery("headers", "x")); return err },
		"nil list option":    func() error { _, err := api.All(ctx, nil); return err },
		"negative limit":     func() error { _, err := api.All(ctx, flowclassifiers.WithListLimit(-1)); return err },
		"owned header": func() error {
			_, err := api.Get(ctx, "a", flowclassifiers.WithGetHeaders(map[string]string{"X-Auth-Token": "x"}))
			return err
		},
		"nil API": func() error { _, err := (*flowclassifiers.API)(nil).Get(ctx, "a"); return err },
	}
	for name, check := range checks {
		if err := check(); err == nil {
			t.Fatal(name, "must fail")
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	if _, err := api.Get(ctx, "a", flowclassifiers.WithGetHeaders(map[string]string{"X-Trace": "1"})); err != nil || len(calls) != 1 {
		t.Fatal(err, calls)
	}
	if api.RawClient() != client || (*flowclassifiers.API)(nil).RawClient() != nil {
		t.Fatal("RawClient must return the shared client")
	}
}
