package federationprotocols_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type protocolTransport func(*http.Request) (*http.Response, error)

func (transport protocolTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type protocolCall struct{ method, path, query, body, accept string }

const protocolBase = "/keystone/v3/OS-FEDERATION/identity_providers/idp1/protocols"

func protocolAPI(t *testing.T, calls *[]protocolCall, reply func(*http.Request) (int, string)) *federationprotocols.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = protocolTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, protocolCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return federationprotocols.New(client)
}

func protocolOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "federation_protocols" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func protocolStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const protocolRow = `{"id":"saml2","mapping_id":"m1","links":{"self":"x","identity_provider":"y"}}`

func protocolCheck(t *testing.T, v *federationprotocols.FederationProtocol) {
	t.Helper()
	if v == nil || v.ID != "saml2" || v.IdentityProviderID != "idp1" || v.MappingID == nil || *v.MappingID != "m1" {
		t.Fatalf("decoded protocol: %#v", v)
	}
	if string(v.Body["mapping_id"]) != `"m1"` || len(v.Body) != 2 || string(v.Wire["links"]) != `{"self":"x","identity_provider":"y"}` {
		t.Fatalf("raw views: %#v", v)
	}
}

func TestFederationProtocolCreatePutsMemberWithoutID(t *testing.T) {
	var calls []protocolCall
	api := protocolAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"protocol":` + protocolRow + `}` })
	v, err := api.Create(context.Background(), "idp1", "saml2",
		federationprotocols.WithCreateField("mapping_id", "m1"),
		federationprotocols.WithCreateField("name", "other"),
		federationprotocols.WithCreateField("idp_id", "other"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != protocolBase+"/saml2" || calls[0].query != "" || calls[0].body != `{"protocol":{"mapping_id":"m1"}}` {
		t.Fatal(calls)
	}
	protocolCheck(t, v)

	calls = nil
	api = protocolAPI(t, &calls, func(*http.Request) (int, string) { return 200, "" })
	v, err = api.Create(context.Background(), "a/b", "c d")
	if err != nil || calls[0].path != "/keystone/v3/OS-FEDERATION/identity_providers/a%2Fb/protocols/c%20d" || calls[0].body != `{"protocol":{}}` {
		t.Fatal(v, err, calls)
	}
	if v.ID != "c d" || v.IdentityProviderID != "a/b" || v.MappingID != nil || string(v.Body["mapping_id"]) != "null" {
		t.Fatalf("seeded create result: %#v", v)
	}
	api = protocolAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
	_, err = api.Create(context.Background(), "idp1", "saml2")
	protocolOperation(t, err, "Create")
	protocolStatus(t, err, 409)
}

func TestFederationProtocolGetUpdateDelete(t *testing.T) {
	var calls []protocolCall
	api := protocolAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Method == http.MethodDelete {
			return 204, ""
		}
		return 200, `{"protocol":` + protocolRow + `}`
	})
	v, err := api.Get(context.Background(), "idp1", "saml2")
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != protocolBase+"/saml2" || calls[0].query != "" {
		t.Fatal(v, err, calls)
	}
	protocolCheck(t, v)

	calls = nil
	v, err = api.Update(context.Background(), "idp1", "saml2", federationprotocols.WithUpdateField("mapping_id", "m2"), federationprotocols.WithUpdateField("id", "x"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPatch || calls[0].path != protocolBase+"/saml2" || calls[0].body != `{"protocol":{"mapping_id":"m2"}}` {
		t.Fatal(v, err, calls)
	}
	protocolCheck(t, v)
	calls = nil
	if v, err = api.Update(context.Background(), "idp1", "saml2"); err != nil || len(calls) != 0 || v.ID != "saml2" {
		t.Fatal("nothing dirty must not send PATCH", v, err, calls)
	}

	if err := api.Delete(context.Background(), "idp1", "saml2"); err != nil || len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != protocolBase+"/saml2" {
		t.Fatal(err, calls)
	}
	for _, item := range []struct {
		code   int
		strict bool
		fails  bool
	}{{404, false, false}, {404, true, true}, {500, false, true}} {
		t.Run(fmt.Sprint(item.code, item.strict), func(t *testing.T) {
			var calls []protocolCall
			api := protocolAPI(t, &calls, func(*http.Request) (int, string) { return item.code, `{}` })
			var options []federationprotocols.DeleteOption
			if item.strict {
				options = append(options, federationprotocols.WithDeleteIgnoreMissing(false))
			}
			err := api.Delete(context.Background(), "idp1", "saml2", options...)
			if !item.fails {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			protocolOperation(t, err, "Delete")
			protocolStatus(t, err, item.code)
			_, err = api.Get(context.Background(), "idp1", "saml2")
			protocolOperation(t, err, "Get")
			protocolStatus(t, err, item.code)
		})
	}
}

func TestFederationProtocolListQueryFiltersAndPaging(t *testing.T) {
	var calls []protocolCall
	api := protocolAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"protocols":[` + protocolRow + `,{"id":"oidc","mapping_id":"m2"}],"links":{"self":"x","next":null,"previous":null}}`
	})
	values, err := api.All(context.Background(), "idp1", federationprotocols.WithListID("saml2"), federationprotocols.WithListQuery("mapping_id", "m1"), federationprotocols.WithListQuery("enabled", true))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].path != protocolBase || calls[0].query != "id=saml2" || calls[0].accept != "application/json" {
		t.Fatal(values, err, calls)
	}
	protocolCheck(t, values[0])
	calls = nil
	values, err = api.All(context.Background(), "idp1", federationprotocols.WithListQuery("name", "oidc"))
	if err != nil || len(values) != 1 || values[0].ID != "oidc" || values[0].IdentityProviderID != "idp1" || calls[0].query != "" {
		t.Fatal(values, err, calls)
	}

	calls = nil
	api = protocolAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"protocols":[{"id":"a"}]}`
		}
		return 200, `{"protocols":[]}`
	})
	values, err = api.All(context.Background(), "a/b", federationprotocols.WithListLimit(1))
	if err != nil || len(values) != 1 || len(calls) != 2 || calls[1].path != "/keystone/v3/OS-FEDERATION/identity_providers/a%2Fb/protocols" || calls[1].query != "limit=1&marker=a" {
		t.Fatal(values, err, calls)
	}
	calls = nil
	if _, err := api.All(context.Background(), ""); err == nil || len(calls) != 0 {
		t.Fatal("blank identity provider must fail before HTTP", err, calls)
	} else {
		protocolOperation(t, err, "List")
	}
}

func TestFederationProtocolFindSendsIdentityProviderQuery(t *testing.T) {
	var calls []protocolCall
	api := protocolAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == protocolBase {
			return 200, `{"protocols":[` + protocolRow + `]}`
		}
		return 404, `{}`
	})
	v, err := api.Find(context.Background(), "idp1", "saml2")
	if err != nil || len(calls) != 2 || calls[0].path != protocolBase+"/saml2" || calls[0].query != "idp_id=idp1" || calls[1].path != protocolBase || calls[1].query != "" {
		t.Fatal(v, err, calls)
	}
	protocolCheck(t, v)
	calls = nil
	if v, err := api.Find(context.Background(), "idp1", "absent"); err != nil || v != nil || len(calls) != 2 {
		t.Fatal("ignore_missing default returns nil", v, err, calls)
	}
	_, err = api.Find(context.Background(), "idp1", "absent", federationprotocols.WithFindIgnoreMissing(false))
	var missing *resource.NotFoundError
	if !errors.As(err, &missing) {
		t.Fatal(err)
	}
	protocolOperation(t, err, "Find")

	calls = nil
	api = protocolAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"protocol":` + protocolRow + `}` })
	v, err = api.Find(context.Background(), "idp1", "saml2")
	if err != nil || len(calls) != 1 || calls[0].query != "idp_id=idp1" {
		t.Fatal(v, err, calls)
	}
	protocolCheck(t, v)
}

func TestFederationProtocolRejectsInvalidInputBeforeHTTP(t *testing.T) {
	var calls []protocolCall
	api := protocolAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
	ctx := context.Background()
	checks := map[string]func() error{
		"blank idp get":    func() error { _, err := api.Get(ctx, "", "a"); return err },
		"control idp":      func() error { _, err := api.Create(ctx, "a\nb", "a"); return err },
		"blank protocol":   func() error { _, err := api.Update(ctx, "idp1", " "); return err },
		"blank idp delete": func() error { return api.Delete(ctx, "..", "a") },
		"blank idp find":   func() error { _, err := api.Find(ctx, "", "a"); return err },
		"nil option":       func() error { _, err := api.Get(ctx, "idp1", "a", nil); return err },
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
