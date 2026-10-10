package tokens_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/catalog"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTokenTransport func(*http.Request) (*http.Response, error)

func (transport nativeTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeTokenCall struct{ method, path, body, authToken, subjectToken string }

type nativeTokenReply struct {
	code   int
	header http.Header
	body   string
}

func nativeTokenClient(t *testing.T, calls *[]nativeTokenCall, reply func(*http.Request) nativeTokenReply) *gophercloud.ServiceClient {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeTokenTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeTokenCall{req.Method, req.URL.Path, raw, req.Header.Get("X-Auth-Token"), req.Header.Get("X-Subject-Token")})
		r := reply(req)
		if r.header == nil {
			r.header = http.Header{}
		}
		r.header.Set("Content-Type", "application/json")
		return &http.Response{StatusCode: r.code, Body: io.NopCloser(strings.NewReader(r.body)), Header: r.header}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return client
}

func nativeTokenOperation(t *testing.T, err error, operation, resourceName string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != resourceName {
		t.Fatal("generated context", err, wrapped)
	}
}

const nativeTokenBody = `{"token":{"expires_at":"2026-10-11T01:02:03.000000Z","methods":["password"],"user":{"id":"u1"}}}`

func TestNativeTokenCreateBodiesScopesAndSubjectHeader(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTokenCall
	api := tokens.New(nativeTokenClient(t, &calls, func(*http.Request) nativeTokenReply {
		return nativeTokenReply{201, http.Header{"X-Subject-Token": {"issued"}}, nativeTokenBody}
	}))
	for _, opts := range []*tokens.AuthOptions{
		{Username: "u", Password: "p", DomainName: "d", Scope: tokens.Scope{ProjectName: "pr", DomainName: "d"}},
		{UserID: "u1", Password: "p", Scope: tokens.Scope{DomainID: "dom"}},
		{TokenID: "old", Scope: tokens.Scope{System: true}},
		{ApplicationCredentialID: "ac", ApplicationCredentialSecret: "s"},
	} {
		got, err := api.Create(ctx, opts)
		if err != nil || got.ID != "issued" || !got.ExpiresAt.Equal(time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)) {
			t.Fatal(got, err)
		}
	}
	// An extension field goes into the auth object only, never into the scope.
	if _, err := api.Create(ctx, &tokens.AuthOptions{Username: "u", Password: "p", DomainID: "d"}, tokens.WithCreateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	want := []string{
		`{"auth":{"identity":{"methods":["password"],"password":{"user":{"domain":{"name":"d"},"name":"u","password":"p"}}},"scope":{"project":{"domain":{"name":"d"},"name":"pr"}}}}`,
		`{"auth":{"identity":{"methods":["password"],"password":{"user":{"id":"u1","password":"p"}}},"scope":{"domain":{"id":"dom"}}}}`,
		`{"auth":{"identity":{"methods":["token"],"token":{"id":"old"}},"scope":{"system":{"all":true}}}}`,
		`{"auth":{"identity":{"application_credential":{"id":"ac","secret":"s"},"methods":["application_credential"]}}}`,
		`{"auth":{"identity":{"methods":["password"],"password":{"user":{"domain":{"id":"d"},"name":"u","password":"p"}}},"x_extension":1}}`,
	}
	if len(calls) != len(want) {
		t.Fatal(calls)
	}
	for i, call := range calls {
		// The native OmitHeaders runs before the provider token is applied, so an authenticated client still sends it.
		if call.method != http.MethodPost || call.path != "/keystone/v3/auth/tokens" || call.body != want[i] || call.authToken != "test-token" {
			t.Fatalf("%d %+v", i, call)
		}
	}
}

func TestNativeTokenGetValidateRevoke(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTokenCall
	api := tokens.New(nativeTokenClient(t, &calls, func(req *http.Request) nativeTokenReply {
		switch req.Method {
		case http.MethodHead:
			if req.Header.Get("X-Subject-Token") == "gone" {
				return nativeTokenReply{404, nil, ""}
			}
			return nativeTokenReply{200, nil, ""}
		case http.MethodDelete:
			return nativeTokenReply{204, nil, ""}
		}
		return nativeTokenReply{203, http.Header{"X-Subject-Token": {req.Header.Get("X-Subject-Token")}}, nativeTokenBody}
	}))
	// Get accepts 203 and reads the ID from the response subject header.
	got, err := api.Get(ctx, "subject")
	if err != nil || got.ID != "subject" || got.ExpiresAt.Year() != 2026 {
		t.Fatal(got, err)
	}
	valid, err := api.Validate(ctx, "subject")
	if err != nil || !valid {
		t.Fatal(valid, err)
	}
	// Validate folds 404 into false without an error.
	valid, err = api.Validate(ctx, "gone")
	if err != nil || valid {
		t.Fatal(valid, err)
	}
	revoked, err := api.Revoke(ctx, "subject")
	if err != nil || revoked == nil || revoked.ID != "" {
		t.Fatal(revoked, err)
	}
	want := []nativeTokenCall{
		{http.MethodGet, "/keystone/v3/auth/tokens", "", "test-token", "subject"},
		{http.MethodHead, "/keystone/v3/auth/tokens", "", "test-token", "subject"},
		{http.MethodHead, "/keystone/v3/auth/tokens", "", "test-token", "gone"},
		{http.MethodDelete, "/keystone/v3/auth/tokens", "", "test-token", "subject"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeTokenStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	status := func(code int) func(*http.Request) nativeTokenReply {
		return func(*http.Request) nativeTokenReply { return nativeTokenReply{code, nil, `{}`} }
	}
	for _, tc := range []struct {
		name     string
		codes    []int
		accepted []int
		call     func(*tokens.API) error
	}{
		{"Create", []int{200, 204, 401}, []int{201, 202}, func(api *tokens.API) error {
			_, err := api.Create(ctx, &tokens.AuthOptions{TokenID: "old"})
			return err
		}},
		{"Get", []int{201, 204, 404}, []int{200, 203}, func(api *tokens.API) error { _, err := api.Get(ctx, "s"); return err }},
		{"Revoke", []int{200, 201, 404}, []int{202, 204}, func(api *tokens.API) error { _, err := api.Revoke(ctx, "s"); return err }},
	} {
		for _, code := range tc.codes {
			var calls []nativeTokenCall
			err := tc.call(tokens.New(nativeTokenClient(t, &calls, status(code))))
			nativeTokenOperation(t, err, tc.name, "tokens")
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, tc.accepted) {
				t.Fatal(tc.name, code, err)
			}
		}
	}
	t.Run("validate errors are native", func(t *testing.T) {
		var calls []nativeTokenCall
		valid, err := tokens.New(nativeTokenClient(t, &calls, status(403))).Validate(ctx, "s")
		var native gophercloud.ErrUnexpectedResponseCode
		var wrapped *resource.OperationError
		if valid || !errors.As(err, &native) || native.Actual != 403 || errors.As(err, &wrapped) {
			t.Fatal(valid, err)
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeTokenCall
		api := tokens.New(nativeTokenClient(t, &calls, status(201)))
		for name, err := range map[string]error{
			"nil options": func() error { _, err := api.Create(ctx, nil); return err }(),
			"no method":   func() error { _, err := api.Create(ctx, &tokens.AuthOptions{}); return err }(),
			"bad header": func() error {
				_, err := api.Create(ctx, &tokens.AuthOptions{TokenID: "old"}, tokens.WithCreateHeader("bad header", "x"))
				return err
			}(),
		} {
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeTokenOperation(t, err, "Create", "tokens")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}

func TestNativeCatalogList(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTokenCall
	api := catalog.New(nativeTokenClient(t, &calls, func(*http.Request) nativeTokenReply {
		return nativeTokenReply{200, nil, `{"catalog":[{"id":"s1","type":"compute","name":"nova","endpoints":[{"id":"e1","interface":"public","region":"r1","region_id":"r1","url":"https://nova"}]}],"links":{"self":"x","next":null}}`}
	}))
	var services []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		services = append(services, value.Type+"="+value.Endpoints[0].URL)
	}
	if !reflect.DeepEqual(services, []string{"compute=https://nova"}) || !reflect.DeepEqual(calls, []nativeTokenCall{{http.MethodGet, "/keystone/v3/auth/catalog", "", "test-token", ""}}) {
		t.Fatal(services, calls)
	}
	for _, tc := range []nativeTokenReply{{404, nil, `{}`}, {200, nil, `{"catalog":[]}`}} {
		var calls []nativeTokenCall
		var errs []error
		for _, err := range catalog.New(nativeTokenClient(t, &calls, func(*http.Request) nativeTokenReply { return tc })).List(ctx) {
			errs = append(errs, err)
		}
		if (tc.code == 404) != (len(errs) == 1) {
			t.Fatal(tc.code, errs)
		}
	}
}
