package identityproviders_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type idpCall struct{ method, path, query, body, accept string }

const idpBase = "/keystone/v3/OS-FEDERATION/identity_providers"

func idpAPI(t *testing.T, calls *[]idpCall, reply func(*http.Request) (int, string)) (*identityproviders.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, idpCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return identityproviders.New(client), cloud
}

func idpOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "identity_providers" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func idpStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const idpRow = `{"id":"idp1","domain_id":"d1","description":"corp","enabled":true,"remote_ids":["r1","r2"],"authorization_ttl":30,"links":{"self":"x"}}`

func idpCheck(t *testing.T, v *identityproviders.IdentityProvider) {
	t.Helper()
	if v == nil || v.ID != "idp1" || *v.DomainID != "d1" || *v.Description != "corp" || !*v.Enabled || !reflect.DeepEqual(v.RemoteIDs, []string{"r1", "r2"}) || *v.AuthorizationTTL != 30 {
		t.Fatalf("decoded identity provider: %#v", v)
	}
	if string(v.Body["enabled"]) != "true" || len(v.Body) != 6 || string(v.Wire["links"]) != `{"self":"x"}` {
		t.Fatalf("raw views: %#v", v)
	}
}

func TestIdentityProviderCreatePutsMemberWithoutID(t *testing.T) {
	var calls []idpCall
	api, _ := idpAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"identity_provider":` + idpRow + `}`
	})
	v, err := api.Create(context.Background(), "idp1",
		identityproviders.WithCreateField("is_enabled", true),
		identityproviders.WithCreateField("description", "corp"),
		identityproviders.WithCreateField("remote_ids", []string{"r1"}),
		identityproviders.WithCreateField("name", "other"),
		identityproviders.WithCreateField("id", "other"),
		identityproviders.WithCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"identity_provider":{"description":"corp","enabled":true,"remote_ids":["r1"]}}`
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != idpBase+"/idp1" || calls[0].query != "" || calls[0].body != want {
		t.Fatal(calls)
	}
	idpCheck(t, v)
	if v.StatusCode != 201 {
		t.Fatal(v.StatusCode)
	}

	calls = nil
	api, _ = idpAPI(t, &calls, func(*http.Request) (int, string) { return 204, "" })
	v, err = api.Create(context.Background(), "a/b", identityproviders.WithCreateField("domain_id", nil))
	if err != nil || calls[0].path != idpBase+"/a%2Fb" || calls[0].body != `{"identity_provider":{"domain_id":null}}` {
		t.Fatal(v, err, calls)
	}
	if v.ID != "a/b" || v.DomainID != nil || string(v.Body["domain_id"]) != "null" || v.Wire != nil {
		t.Fatalf("seeded create result: %#v", v)
	}

	for _, code := range []int{200, 202, 299, 399, 400, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []idpCall
			api, _ := idpAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Create(context.Background(), "idp1")
			if code < 400 {
				if err != nil || calls[0].body != `{"identity_provider":{}}` {
					t.Fatal(err, calls)
				}
				return
			}
			idpOperation(t, err, "Create")
			idpStatus(t, err, code)
		})
	}
}

func TestIdentityProviderGetDecodesAndRejects(t *testing.T) {
	for _, item := range []struct {
		name, body string
		valid      bool
	}{{"envelope", `{"identity_provider":` + idpRow + `}`, true}, {"root", idpRow, true}, {"array", `[]`, false}, {"null member", `{"identity_provider":null}`, false}} {
		t.Run(item.name, func(t *testing.T) {
			var calls []idpCall
			api, _ := idpAPI(t, &calls, func(*http.Request) (int, string) { return 200, item.body })
			v, err := api.Get(context.Background(), "idp1")
			if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != idpBase+"/idp1" {
				t.Fatal(calls)
			}
			if !item.valid {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 {
					t.Fatal(v, err)
				}
				idpOperation(t, err, "Get")
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			idpCheck(t, v)
		})
	}
	var calls []idpCall
	api, _ := idpAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{"error":{}}` })
	if _, err := api.Get(context.Background(), "missing"); err == nil {
		t.Fatal("404 must fail")
	} else {
		idpOperation(t, err, "Get")
		idpStatus(t, err, 404)
	}
	if _, err := api.Get(context.Background(), " "); err == nil || len(calls) != 1 {
		t.Fatal("blank ID must fail before HTTP", err, calls)
	}
}

func TestIdentityProviderUpdatePatchesDirtyAttributes(t *testing.T) {
	var calls []idpCall
	api, _ := idpAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"identity_provider":` + idpRow + `}`
	})
	v, err := api.Update(context.Background(), "idp1",
		identityproviders.WithUpdateField("enabled", false),
		identityproviders.WithUpdateField("authorization_ttl", nil),
		identityproviders.WithUpdateField("id", "ignored"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPatch || calls[0].path != idpBase+"/idp1" || calls[0].body != `{"identity_provider":{"authorization_ttl":null,"enabled":false}}` {
		t.Fatal(calls)
	}
	idpCheck(t, v)

	calls = nil
	v, err = api.Update(context.Background(), "idp1", identityproviders.WithUpdateField("name", "x"), identityproviders.WithUpdateField("unknown", 1))
	if err != nil || len(calls) != 0 || v.ID != "idp1" || v.StatusCode != 0 {
		t.Fatal("nothing dirty must not send PATCH", v, err, calls)
	}
	if _, err := api.Update(context.Background(), "idp1", identityproviders.WithUpdateField("is_enabled", true), identityproviders.WithUpdateField("enabled", false)); err == nil || len(calls) != 0 {
		t.Fatal("conflicting aliases must fail before HTTP", err)
	}
	api, _ = idpAPI(t, &calls, func(*http.Request) (int, string) { return 403, `{}` })
	_, err = api.Update(context.Background(), "idp1", identityproviders.WithUpdateField("description", "x"))
	idpOperation(t, err, "Update")
	idpStatus(t, err, 403)
}

func TestIdentityProviderDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, item := range []struct {
		code    int
		options []identityproviders.DeleteOption
		fails   bool
	}{
		{204, nil, false}, {202, nil, false}, {404, nil, false},
		{404, []identityproviders.DeleteOption{identityproviders.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		t.Run(fmt.Sprint(item.code, len(item.options)), func(t *testing.T) {
			var calls []idpCall
			api, _ := idpAPI(t, &calls, func(*http.Request) (int, string) { return item.code, "" })
			err := api.Delete(context.Background(), "idp1", item.options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != idpBase+"/idp1" {
				t.Fatal(calls)
			}
			if !item.fails {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			idpOperation(t, err, "Delete")
			idpStatus(t, err, item.code)
		})
	}
}

func TestIdentityProviderListQueryFiltersAndPaging(t *testing.T) {
	var calls []idpCall
	api, _ := idpAPI(t, &calls, func(req *http.Request) (int, string) {
		return 200, `{"identity_providers":[` + idpRow + `,{"id":"idp2","description":"other","enabled":true}],"links":{"self":"x","next":null,"previous":null}}`
	})
	values, err := api.All(context.Background(), identityproviders.WithListEnabled(true), identityproviders.WithListID("idp1"),
		identityproviders.WithListQuery("description", "corp"), identityproviders.WithListQuery("unknown", make(chan int)))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	idpCheck(t, values[0])
	if calls[0].path != idpBase || calls[0].query != "enabled=True&id=idp1" || calls[0].accept != "application/json" {
		t.Fatal(calls)
	}
	calls = nil
	values, err = api.All(context.Background(), identityproviders.WithListQuery("name", "idp2"), identityproviders.WithListQuery("enabled", false), identityproviders.WithListQuery("is_enabled", true))
	if err != nil || len(values) != 1 || values[0].ID != "idp2" || calls[0].query != "enabled=True" {
		t.Fatal(values, err, calls)
	}

	calls = nil
	api, _ = idpAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"identity_providers":[{"id":"a"}],"links":{"next":null}}`
		case "a":
			return 200, `{"identity_providers":[{"id":"b"}],"links":{"next":null}}`
		}
		return 200, `{"identity_providers":[],"links":{"next":null}}`
	})
	values, err = api.All(context.Background(), identityproviders.WithListLimit(1))
	if err != nil || len(values) != 2 || len(calls) != 3 || calls[1].query != "limit=1&marker=a" || calls[2].query != "limit=1&marker=b" {
		t.Fatal("limit continues with the last ID as marker until an empty page", values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), identityproviders.WithListLimit(1), identityproviders.WithListPaginated(false))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	calls = nil
	values, err = api.All(context.Background(), identityproviders.WithListMaxItems(1))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].query != "limit=1" {
		t.Fatal(values, err, calls)
	}

	calls = nil
	api, _ = idpAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.All(context.Background())
	idpOperation(t, err, "List")
	idpStatus(t, err, 500)
}

func TestIdentityProviderFindFallsBackToList(t *testing.T) {
	list := `{"identity_providers":[{"id":"idp1"},{"id":"idp2"}],"links":{"next":null}}`
	for _, code := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []idpCall
			api, _ := idpAPI(t, &calls, func(req *http.Request) (int, string) {
				if req.URL.Path == idpBase {
					return 200, list
				}
				return code, `{}`
			})
			v, err := api.Find(context.Background(), "idp2")
			if err != nil || v == nil || v.ID != "idp2" || len(calls) != 2 || calls[0].path != idpBase+"/idp2" || calls[1].path != idpBase || calls[1].query != "" {
				t.Fatal(v, err, calls)
			}
			calls = nil
			v, err = api.Find(context.Background(), "absent")
			if err != nil || v != nil || len(calls) != 2 {
				t.Fatal("ignore_missing default returns nil", v, err, calls)
			}
			_, err = api.Find(context.Background(), "absent", identityproviders.WithFindIgnoreMissing(false))
			var missing *resource.NotFoundError
			if !errors.As(err, &missing) {
				t.Fatal(err)
			}
			idpOperation(t, err, "Find")
		})
	}
	var calls []idpCall
	api, _ := idpAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"identity_provider":` + idpRow + `}` })
	v, err := api.Find(context.Background(), "idp1")
	if err != nil || len(calls) != 1 {
		t.Fatal(v, err, calls)
	}
	idpCheck(t, v)

	calls = nil
	api, _ = idpAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == idpBase {
			return 200, `{"identity_providers":[{"id":"dup"},{"id":"dup"}]}`
		}
		return 404, `{}`
	})
	if _, err := api.Find(context.Background(), "dup"); err == nil {
		t.Fatal("duplicate match must fail")
	}
	calls = nil
	api, _ = idpAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.Find(context.Background(), "idp1")
	idpOperation(t, err, "Find")
	idpStatus(t, err, 500)
	if len(calls) != 1 {
		t.Fatal("500 must not fall back", calls)
	}
}

func TestIdentityProviderRejectsInvalidInputBeforeHTTP(t *testing.T) {
	var calls []idpCall
	api, _ := idpAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
	ctx := context.Background()
	checks := map[string]func() error{
		"nil get option":    func() error { _, err := api.Get(ctx, "a", nil); return err },
		"nil create option": func() error { _, err := api.Create(ctx, "a", nil); return err },
		"blank create ID":   func() error { _, err := api.Create(ctx, ""); return err },
		"nil delete option": func() error { return api.Delete(ctx, "a", nil) },
		"nil find option":   func() error { _, err := api.Find(ctx, "a", nil); return err },
		"nil list option":   func() error { _, err := api.All(ctx, nil); return err },
		"negative limit":    func() error { _, err := api.All(ctx, identityproviders.WithListLimit(-1)); return err },
		"owned header": func() error {
			_, err := api.Get(ctx, "a", identityproviders.WithGetHeaders(map[string]string{"X-Auth-Token": "x"}))
			return err
		},
		"nil API": func() error { _, err := (*identityproviders.API)(nil).Get(ctx, "a"); return err },
	}
	for name, check := range checks {
		if err := check(); err == nil {
			t.Fatal(name, "must fail")
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	if _, err := api.Get(ctx, "a", identityproviders.WithGetHeaders(map[string]string{"X-Trace": "1"})); err != nil || len(calls) != 1 {
		t.Fatal(err, calls)
	}
}
