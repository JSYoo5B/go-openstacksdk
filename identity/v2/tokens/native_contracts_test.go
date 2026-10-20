package tokens_test

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
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity/v2/tokens"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeV2TokenTransport func(*http.Request) (*http.Response, error)

func (transport nativeV2TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

// authToken records the X-Auth-Token header that reaches the wire.
type nativeV2TokenCall struct{ method, path, query, body, authToken string }

func nativeV2TokenAPI(t *testing.T, calls *[]nativeV2TokenCall, reply func(*http.Request) (int, string)) (*tokens.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeV2TokenTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeV2TokenCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("X-Auth-Token")})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}, "X-Openstack-Request-Id": {"req-1"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v2.0/"
	return tokens.New(client), cloud
}

func nativeV2TokenOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "tokens" {
		t.Fatal("generated tokens context", err, wrapped)
	}
}

const nativeV2TokenAccess = `{"access":{
	"token":{"id":"tok-1","expires":"2026-10-11T01:02:03.123456Z","issued_at":"2026-10-11T00:02:03Z","tenant":{"id":"t-1","name":"demo","description":"d","enabled":true}},
	"serviceCatalog":[{"name":"nova","type":"compute","endpoints":[{"tenantId":"t-1","publicURL":"https://nova.invalid/v2.1","internalURL":"http://nova.internal","adminURL":"http://nova.admin","region":"RegionOne","versionId":"2.1","versionInfo":"vi","versionList":"vl"}],"endpoints_links":[]}],
	"user":{"id":"u-1","name":"alice","username":"alice","roles":[{"name":"admin"},{"name":"member"}],"roles_links":[]},
	"metadata":{"is_admin":0,"roles":["r1"]}}}`

func TestNativeV2TokenRoutesBodiesAndAccessDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeV2TokenCall
	api, cloud := nativeV2TokenAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Method == http.MethodGet {
			// Get also accepts the non-authoritative 203.
			return 203, nativeV2TokenAccess
		}
		return 200, nativeV2TokenAccess
	})
	password, err := api.Create(ctx, tokens.AuthOptions{Username: "alice", Password: "secret", TenantID: "t-1", TokenID: "dropped"}, tokens.WithCreateField("x_extension", 1))
	if err != nil {
		t.Fatal(err)
	}
	wantToken := tokens.Token{ID: "tok-1", ExpiresAt: time.Date(2026, 10, 11, 1, 2, 3, 123456000, time.UTC)}
	wantToken.Tenant.ID, wantToken.Tenant.Name, wantToken.Tenant.Description, wantToken.Tenant.Enabled = "t-1", "demo", "d", true
	wantUser := tokens.User{ID: "u-1", Name: "alice", UserName: "alice", Roles: []tokens.Role{{Name: "admin"}, {Name: "member"}}}
	wantCatalog := tokens.ServiceCatalog{Entries: []tokens.CatalogEntry{{Name: "nova", Type: "compute", Endpoints: []tokens.Endpoint{{
		TenantID: "t-1", PublicURL: "https://nova.invalid/v2.1", InternalURL: "http://nova.internal", AdminURL: "http://nova.admin",
		Region: "RegionOne", VersionID: "2.1", VersionInfo: "vi", VersionList: "vl",
	}}}}}
	if !(reflect.DeepEqual(password.Token, wantToken) && reflect.DeepEqual(password.User, wantUser) && reflect.DeepEqual(password.Catalog, wantCatalog)) {
		t.Fatalf("%+v", password)
	}
	var body map[string]any
	if err := json.Unmarshal(password.Body, &body); err != nil || body["access"].(map[string]any)["metadata"] == nil || password.Header.Get("X-Openstack-Request-Id") != "req-1" {
		t.Fatal(string(password.Body), password.Header, err)
	}
	if _, err := api.Create(ctx, tokens.AuthOptions{TokenID: "tok-0", TenantName: "demo", Username: "ignored"}); err != nil {
		t.Fatal(err)
	}
	validated, err := api.Get(ctx, "tok-1")
	if err != nil || !reflect.DeepEqual(validated.Token, wantToken) || validated.User.ID != "u-1" {
		t.Fatal(validated, err)
	}
	want := []nativeV2TokenCall{
		// Password wins over TokenID. Native OmitHeaders runs before the provider token is
		// added, so an authenticated client still sends its X-Auth-Token on Create.
		{http.MethodPost, "/keystone/v2.0/tokens", "", `{"auth":{"passwordCredentials":{"password":"secret","username":"alice"},"tenantId":"t-1","x_extension":1}}`, "test-token"},
		{http.MethodPost, "/keystone/v2.0/tokens", "", `{"auth":{"tenantName":"demo","token":{"id":"tok-0"}}}`, "test-token"},
		{http.MethodGet, "/keystone/v2.0/tokens/tok-1", "", "", "test-token"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	// The URL helpers send nothing and do not escape the token path segment.
	if got := api.CreateURL(ctx); got != cloud.Server.URL+"/keystone/v2.0/tokens" {
		t.Fatal(got)
	}
	if got := api.GetURL(ctx, "a/b?c"); got != cloud.Server.URL+"/keystone/v2.0/tokens/a/b?c" {
		t.Fatal(got)
	}
	if len(calls) != 3 {
		t.Fatal(calls)
	}
}

func TestNativeV2TokenStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	valid := tokens.AuthOptions{Username: "alice", Password: "secret"}
	for _, call := range []struct {
		name string
		call func(*tokens.API) error
	}{
		{"Create", func(api *tokens.API) error { _, err := api.Create(ctx, valid); return err }},
		{"Get", func(api *tokens.API) error { _, err := api.Get(ctx, "tok-1"); return err }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeV2TokenCall
				api, _ := nativeV2TokenAPI(t, &calls, func(*http.Request) (int, string) { return code, nativeV2TokenAccess })
				err := call.call(api)
				if len(calls) != 1 {
					t.Fatal(calls)
				}
				if code == 200 {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				nativeV2TokenOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200, 203}) {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("access decode", func(t *testing.T) {
		for _, tc := range []struct {
			name, body string
			ok         bool
		}{
			// An unscoped token has no tenant, catalog or user, and still decodes.
			{"unscoped", `{"access":{"token":{"id":"tok-1","expires":"2026-10-11T01:02:03Z"}}}`, true},
			// expires must use RFC3339Milli with a literal Z.
			{"missing expires", `{"access":{"token":{"id":"tok-1"}}}`, false},
			{"offset expires", `{"access":{"token":{"id":"tok-1","expires":"2026-10-11T10:02:03+09:00"}}}`, false},
			{"bad catalog", `{"access":{"token":{"id":"tok-1","expires":"2026-10-11T01:02:03Z"},"serviceCatalog":{}}}`, false},
			{"bad user", `{"access":{"token":{"id":"tok-1","expires":"2026-10-11T01:02:03Z"},"user":[]}}`, false},
		} {
			for _, operation := range []string{"Create", "Get"} {
				var calls []nativeV2TokenCall
				api, _ := nativeV2TokenAPI(t, &calls, func(*http.Request) (int, string) { return 200, tc.body })
				var got *tokens.Authentication
				var err error
				if operation == "Create" {
					got, err = api.Create(ctx, valid)
				} else {
					got, err = api.Get(ctx, "tok-1")
				}
				if !tc.ok {
					if got != nil {
						t.Fatal(tc.name, got)
					}
					nativeV2TokenOperation(t, err, operation)
					continue
				}
				if err != nil || got.Token.ID != "tok-1" || got.Token.Tenant.ID != "" || got.Catalog.Entries != nil || got.User.ID != "" {
					t.Fatal(tc.name, got, err)
				}
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeV2TokenCall
		api, _ := nativeV2TokenAPI(t, &calls, func(*http.Request) (int, string) { return 200, nativeV2TokenAccess })
		for name, err := range map[string]error{
			"password without username": func() error { _, err := api.Create(ctx, tokens.AuthOptions{Password: "secret"}); return err }(),
			"no password or token":      func() error { _, err := api.Create(ctx, tokens.AuthOptions{Username: "alice"}); return err }(),
			"core extension":            func() error { _, err := api.Create(ctx, valid, tokens.WithCreateField("tenantName", "x")); return err }(),
			"envelope collision": func() error {
				_, err := api.Create(ctx, valid, tokens.WithCreateField("passwordCredentials", map[string]string{}))
				return err
			}(),
			"nil option": func() error { _, err := api.Create(ctx, valid, nil); return err }(),
		} {
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeV2TokenOperation(t, err, "Create")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
		// token is not a declared AuthOptions field, so it may be merged beside passwordCredentials.
		if _, err := api.Create(ctx, valid, tokens.WithCreateField("token", map[string]string{"id": "x"})); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 1 || calls[0].body != `{"auth":{"passwordCredentials":{"password":"secret","username":"alice"},"token":{"id":"x"}}}` {
			t.Fatal(calls)
		}
	})
}
