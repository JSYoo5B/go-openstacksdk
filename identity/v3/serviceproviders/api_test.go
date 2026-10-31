package serviceproviders_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type spCall struct{ method, path, query, body, accept string }

const spBase = "/keystone/v3/OS-FEDERATION/service_providers"

func spAPI(t *testing.T, calls *[]spCall, reply func(*http.Request) (int, string)) (*serviceproviders.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, spCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return serviceproviders.New(client), cloud
}

func spOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "service_providers" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func spStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const spRow = `{"id":"sp1","auth_url":"https://keystone/auth","description":"corp","enabled":true,"relay_state_prefix":"ss:mem:","sp_url":"https://sp","links":{"self":"x"}}`

func spCheck(t *testing.T, v *serviceproviders.ServiceProvider) {
	t.Helper()
	if v == nil || v.ID != "sp1" || *v.AuthURL != "https://keystone/auth" || *v.Description != "corp" || !*v.Enabled || *v.RelayStatePrefix != "ss:mem:" || *v.SPURL != "https://sp" {
		t.Fatalf("decoded service provider: %#v", v)
	}
	if string(v.Body["enabled"]) != "true" || len(v.Body) != 6 || string(v.Wire["links"]) != `{"self":"x"}` {
		t.Fatalf("raw views: %#v", v)
	}
}

func TestServiceProviderCreatePutsMemberWithoutID(t *testing.T) {
	var calls []spCall
	api, _ := spAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"service_provider":` + spRow + `}`
	})
	v, err := api.Create(context.Background(), "sp1",
		serviceproviders.WithCreateField("is_enabled", true),
		serviceproviders.WithCreateField("description", "corp"),
		serviceproviders.WithCreateField("sp_url", "https://sp"),
		serviceproviders.WithCreateField("name", "other"),
		serviceproviders.WithCreateField("id", "other"),
		serviceproviders.WithCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"service_provider":{"description":"corp","enabled":true,"sp_url":"https://sp"}}`
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != spBase+"/sp1" || calls[0].query != "" || calls[0].body != want {
		t.Fatal(calls)
	}
	spCheck(t, v)
	if v.StatusCode != 201 {
		t.Fatal(v.StatusCode)
	}

	calls = nil
	api, _ = spAPI(t, &calls, func(*http.Request) (int, string) { return 204, "" })
	v, err = api.Create(context.Background(), "a/b", serviceproviders.WithCreateField("relay_state_prefix", nil))
	if err != nil || calls[0].path != spBase+"/a%2Fb" || calls[0].body != `{"service_provider":{"relay_state_prefix":null}}` {
		t.Fatal(v, err, calls)
	}
	if v.ID != "a/b" || v.RelayStatePrefix != nil || string(v.Body["relay_state_prefix"]) != "null" || v.Wire != nil {
		t.Fatalf("seeded create result: %#v", v)
	}

	for _, code := range []int{200, 202, 299, 399, 400, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []spCall
			api, _ := spAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Create(context.Background(), "sp1")
			if code < 400 {
				if err != nil || calls[0].body != `{"service_provider":{}}` {
					t.Fatal(err, calls)
				}
				return
			}
			spOperation(t, err, "Create")
			spStatus(t, err, code)
		})
	}
}

func TestServiceProviderGetDecodesAndRejects(t *testing.T) {
	for _, item := range []struct {
		name, body string
		valid      bool
	}{{"envelope", `{"service_provider":` + spRow + `}`, true}, {"root", spRow, true}, {"array", `[]`, false}, {"null member", `{"service_provider":null}`, false}} {
		t.Run(item.name, func(t *testing.T) {
			var calls []spCall
			api, _ := spAPI(t, &calls, func(*http.Request) (int, string) { return 200, item.body })
			v, err := api.Get(context.Background(), "sp1")
			if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != spBase+"/sp1" {
				t.Fatal(calls)
			}
			if !item.valid {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 {
					t.Fatal(v, err)
				}
				spOperation(t, err, "Get")
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			spCheck(t, v)
		})
	}
	var calls []spCall
	api, _ := spAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{"error":{}}` })
	if _, err := api.Get(context.Background(), "missing"); err == nil {
		t.Fatal("404 must fail")
	} else {
		spOperation(t, err, "Get")
		spStatus(t, err, 404)
	}
	if _, err := api.Get(context.Background(), " "); err == nil || len(calls) != 1 {
		t.Fatal("blank ID must fail before HTTP", err, calls)
	}
}

func TestServiceProviderUpdatePatchesDirtyAttributes(t *testing.T) {
	var calls []spCall
	api, _ := spAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"service_provider":` + spRow + `}`
	})
	v, err := api.Update(context.Background(), "sp1",
		serviceproviders.WithUpdateField("enabled", false),
		serviceproviders.WithUpdateField("auth_url", nil),
		serviceproviders.WithUpdateField("id", "ignored"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPatch || calls[0].path != spBase+"/sp1" || calls[0].body != `{"service_provider":{"auth_url":null,"enabled":false}}` {
		t.Fatal(calls)
	}
	spCheck(t, v)

	calls = nil
	v, err = api.Update(context.Background(), "sp1", serviceproviders.WithUpdateField("name", "x"), serviceproviders.WithUpdateField("unknown", 1))
	if err != nil || len(calls) != 0 || v.ID != "sp1" || v.StatusCode != 0 {
		t.Fatal("nothing dirty must not send PATCH", v, err, calls)
	}
	if _, err := api.Update(context.Background(), "sp1", serviceproviders.WithUpdateField("is_enabled", true), serviceproviders.WithUpdateField("enabled", false)); err == nil || len(calls) != 0 {
		t.Fatal("conflicting aliases must fail before HTTP", err)
	}
	api, _ = spAPI(t, &calls, func(*http.Request) (int, string) { return 403, `{}` })
	_, err = api.Update(context.Background(), "sp1", serviceproviders.WithUpdateField("description", "x"))
	spOperation(t, err, "Update")
	spStatus(t, err, 403)
}

func TestServiceProviderDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, item := range []struct {
		code    int
		options []serviceproviders.DeleteOption
		fails   bool
	}{
		{204, nil, false}, {202, nil, false}, {404, nil, false},
		{404, []serviceproviders.DeleteOption{serviceproviders.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		t.Run(fmt.Sprint(item.code, len(item.options)), func(t *testing.T) {
			var calls []spCall
			api, _ := spAPI(t, &calls, func(*http.Request) (int, string) { return item.code, "" })
			err := api.Delete(context.Background(), "sp1", item.options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != spBase+"/sp1" {
				t.Fatal(calls)
			}
			if !item.fails {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			spOperation(t, err, "Delete")
			spStatus(t, err, item.code)
		})
	}
}

func TestServiceProviderListQueryFiltersAndPaging(t *testing.T) {
	var calls []spCall
	api, _ := spAPI(t, &calls, func(req *http.Request) (int, string) {
		return 200, `{"service_providers":[` + spRow + `,{"id":"sp2","description":"other","enabled":true}],"links":{"self":"x","next":null,"previous":null}}`
	})
	values, err := api.All(context.Background(), serviceproviders.WithListEnabled(true), serviceproviders.WithListID("sp1"),
		serviceproviders.WithListQuery("description", "corp"), serviceproviders.WithListQuery("unknown", make(chan int)))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	spCheck(t, values[0])
	if calls[0].path != spBase || calls[0].query != "enabled=True&id=sp1" || calls[0].accept != "application/json" {
		t.Fatal(calls)
	}
	calls = nil
	values, err = api.All(context.Background(), serviceproviders.WithListQuery("name", "sp2"), serviceproviders.WithListQuery("enabled", false), serviceproviders.WithListQuery("is_enabled", true))
	if err != nil || len(values) != 1 || values[0].ID != "sp2" || calls[0].query != "enabled=True" {
		t.Fatal(values, err, calls)
	}

	calls = nil
	api, _ = spAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"service_providers":[{"id":"a"}],"links":{"next":null}}`
		case "a":
			return 200, `{"service_providers":[{"id":"b"}],"links":{"next":null}}`
		}
		return 200, `{"service_providers":[],"links":{"next":null}}`
	})
	values, err = api.All(context.Background(), serviceproviders.WithListLimit(1))
	if err != nil || len(values) != 2 || len(calls) != 3 || calls[1].query != "limit=1&marker=a" || calls[2].query != "limit=1&marker=b" {
		t.Fatal("limit continues with the last ID as marker until an empty page", values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), serviceproviders.WithListLimit(1), serviceproviders.WithListPaginated(false))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), serviceproviders.WithListMaxItems(1))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].query != "limit=1" {
		t.Fatal(values, err, calls)
	}

	calls = nil
	api, _ = spAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.All(context.Background())
	spOperation(t, err, "List")
	spStatus(t, err, 500)
}

func TestServiceProviderFindFallsBackToList(t *testing.T) {
	list := `{"service_providers":[{"id":"sp1"},{"id":"sp2"}],"links":{"next":null}}`
	for _, code := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []spCall
			api, _ := spAPI(t, &calls, func(req *http.Request) (int, string) {
				if req.URL.Path == spBase {
					return 200, list
				}
				return code, `{}`
			})
			v, err := api.Find(context.Background(), "sp2")
			if err != nil || v == nil || v.ID != "sp2" || len(calls) != 2 || calls[0].path != spBase+"/sp2" || calls[1].path != spBase || calls[1].query != "" {
				t.Fatal(v, err, calls)
			}
			calls = nil
			v, err = api.Find(context.Background(), "absent")
			if err != nil || v != nil || len(calls) != 2 {
				t.Fatal("ignore_missing default returns nil", v, err, calls)
			}
			_, err = api.Find(context.Background(), "absent", serviceproviders.WithFindIgnoreMissing(false))
			var missing *resource.NotFoundError
			if !errors.As(err, &missing) {
				t.Fatal(err)
			}
			spOperation(t, err, "Find")
		})
	}
	var calls []spCall
	api, _ := spAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"service_provider":` + spRow + `}` })
	v, err := api.Find(context.Background(), "sp1")
	if err != nil || len(calls) != 1 {
		t.Fatal(v, err, calls)
	}
	spCheck(t, v)

	calls = nil
	api, _ = spAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == spBase {
			return 200, `{"service_providers":[{"id":"dup"},{"id":"dup"}]}`
		}
		return 404, `{}`
	})
	if _, err := api.Find(context.Background(), "dup"); err == nil {
		t.Fatal("duplicate match must fail")
	}
	calls = nil
	api, _ = spAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.Find(context.Background(), "sp1")
	spOperation(t, err, "Find")
	spStatus(t, err, 500)
	if len(calls) != 1 {
		t.Fatal("500 must not fall back", calls)
	}
}

func TestServiceProviderRejectsInvalidInputBeforeHTTP(t *testing.T) {
	var calls []spCall
	api, _ := spAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
	ctx := context.Background()
	checks := map[string]func() error{
		"nil get option":    func() error { _, err := api.Get(ctx, "a", nil); return err },
		"nil create option": func() error { _, err := api.Create(ctx, "a", nil); return err },
		"blank create ID":   func() error { _, err := api.Create(ctx, ""); return err },
		"nil delete option": func() error { return api.Delete(ctx, "a", nil) },
		"nil find option":   func() error { _, err := api.Find(ctx, "a", nil); return err },
		"nil list option":   func() error { _, err := api.All(ctx, nil); return err },
		"negative limit":    func() error { _, err := api.All(ctx, serviceproviders.WithListLimit(-1)); return err },
		"owned header": func() error {
			_, err := api.Get(ctx, "a", serviceproviders.WithGetHeaders(map[string]string{"X-Auth-Token": "x"}))
			return err
		},
		"nil API": func() error { _, err := (*serviceproviders.API)(nil).Get(ctx, "a"); return err },
	}
	for name, check := range checks {
		if err := check(); err == nil {
			t.Fatal(name, "must fail")
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	if _, err := api.Get(ctx, "a", serviceproviders.WithGetHeaders(map[string]string{"X-Trace": "1"})); err != nil || len(calls) != 1 {
		t.Fatal(err, calls)
	}
}
