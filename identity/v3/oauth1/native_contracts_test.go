package oauth1_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/oauth1"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeOAuth1Transport func(*http.Request) (*http.Response, error)

func (transport nativeOAuth1Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeOAuth1Call struct{ method, path, query, body string }

type nativeOAuth1Reply struct {
	code   int
	header http.Header
	body   string
}

// nativeOAuth1JSON replies with a JSON body; token calls set their own Content-Type.
func nativeOAuth1JSON(code int, body string) nativeOAuth1Reply {
	return nativeOAuth1Reply{code, http.Header{"Content-Type": {"application/json"}}, body}
}

func nativeOAuth1API(t *testing.T, calls *[]nativeOAuth1Call, headers *[]http.Header, reply func(*http.Request) nativeOAuth1Reply) (*oauth1.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeOAuth1Transport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeOAuth1Call{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		if headers != nil {
			*headers = append(*headers, req.Header.Clone())
		}
		r := reply(req)
		return &http.Response{StatusCode: r.code, Body: io.NopCloser(strings.NewReader(r.body)), Header: r.header}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return oauth1.New(client), cloud
}

func nativeOAuth1Operation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "oauth1" {
		t.Fatal("generated oauth1 context", err, wrapped)
	}
}

// nativeOAuth1Params parses an `OAuth k="v", ...` Authorization header.
func nativeOAuth1Params(t *testing.T, header string) map[string]string {
	t.Helper()
	rest, ok := strings.CutPrefix(header, "OAuth ")
	if !ok {
		t.Fatal("not an OAuth header", header)
	}
	params := map[string]string{}
	for _, pair := range strings.Split(rest, ", ") {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
			t.Fatal("malformed OAuth pair", pair)
		}
		params[key] = value[1 : len(value)-1]
	}
	return params
}

// nativeOAuth1Sign derives the RFC 5849 HMAC-SHA1 signature from the sorted
// oauth_* parameters, independently of the upstream helpers.
func nativeOAuth1Sign(method, rawURL string, params url.Values, secrets ...string) string {
	base := method + "&" + url.QueryEscape(rawURL) + "&" + url.QueryEscape(params.Encode())
	key := url.QueryEscape(secrets[0]) + "&"
	if len(secrets) > 1 {
		key += url.QueryEscape(secrets[1])
	}
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(base))
	return url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
}

const nativeOAuth1Consumer = `{"id":"c1","secret":"cs","description":"d"}`
const nativeOAuth1AccessToken = `{"id":"at1","consumer_id":"c1","project_id":"p1","authorizing_user_id":"u1","expires_at":"2026-10-11T01:02:03.000000Z"}`

func TestNativeOAuth1ConsumerRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeOAuth1Call
	var cloud *testcloud.Cloud
	api, cloud := nativeOAuth1API(t, &calls, nil, func(req *http.Request) nativeOAuth1Reply {
		switch {
		case req.Method == http.MethodDelete:
			return nativeOAuth1JSON(204, "")
		case req.Method == http.MethodPost:
			return nativeOAuth1JSON(201, `{"consumer":`+nativeOAuth1Consumer+`}`)
		case req.URL.Path == "/keystone/v3/OS-OAUTH1/consumers":
			return nativeOAuth1JSON(200, `{"consumers":[`+nativeOAuth1Consumer+`],"links":{"next":"`+cloud.Server.URL+`/other/consumers?page=2"}}`)
		case req.URL.Path == "/other/consumers":
			return nativeOAuth1JSON(200, `{"consumers":[{"id":"c2"}],"links":{"next":null}}`)
		}
		return nativeOAuth1JSON(200, `{"consumer":`+nativeOAuth1Consumer+`}`)
	})
	created, err := api.CreateConsumer(ctx, oauth1.CreateConsumerOpts{Description: "d"}, oauth1.WithCreateConsumerField("x_extension", 1))
	if err != nil || *created != (oauth1.Consumer{ID: "c1", Secret: "cs", Description: "d"}) {
		t.Fatal(created, err)
	}
	// Description has no omitempty, so an empty description is still sent.
	if _, err := api.CreateConsumer(ctx, oauth1.CreateConsumerOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.GetConsumer(ctx, "c1")
	if err != nil || got.ID != "c1" {
		t.Fatal(got, err)
	}
	updated, err := api.UpdateConsumer(ctx, "c1", oauth1.UpdateConsumerOpts{Description: "new"}, oauth1.WithUpdateConsumerField("x_extension", "v"))
	if err != nil || updated.Secret != "cs" {
		t.Fatal(updated, err)
	}
	var ids []string
	for value, err := range api.ListConsumers(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if err := api.DeleteConsumer(ctx, "c1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"c1", "c2"}) {
		t.Fatal(ids)
	}
	base := "/keystone/v3/OS-OAUTH1/consumers"
	want := []nativeOAuth1Call{
		{http.MethodPost, base, "", `{"consumer":{"description":"d","x_extension":1}}`},
		{http.MethodPost, base, "", `{"consumer":{"description":""}}`},
		{http.MethodGet, base + "/c1", "", ""},
		{http.MethodPatch, base + "/c1", "", `{"consumer":{"description":"new","x_extension":"v"}}`},
		{http.MethodGet, base, "", ""},
		{http.MethodGet, "/other/consumers", "page=2", ""},
		{http.MethodDelete, base + "/c1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeOAuth1AccessTokenRoutesAuthorizePagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeOAuth1Call
	var cloud *testcloud.Cloud
	api, cloud := nativeOAuth1API(t, &calls, nil, func(req *http.Request) nativeOAuth1Reply {
		base := "/keystone/v3/users/u1/OS-OAUTH1/access_tokens"
		switch {
		case req.Method == http.MethodDelete:
			return nativeOAuth1JSON(204, "")
		case req.Method == http.MethodPut:
			return nativeOAuth1JSON(200, `{"token":{"oauth_verifier":"verifier"}}`)
		case req.URL.Path == base:
			return nativeOAuth1JSON(200, `{"access_tokens":[`+nativeOAuth1AccessToken+`],"links":{"next":"`+cloud.Server.URL+`/other/tokens?page=2"}}`)
		case req.URL.Path == "/other/tokens":
			return nativeOAuth1JSON(200, `{"access_tokens":[{"id":"at2"}],"links":{"next":null}}`)
		case req.URL.Path == base+"/at1/roles":
			return nativeOAuth1JSON(200, `{"roles":[{"id":"r1","name":"member","domain_id":"default"}],"links":{"next":null}}`)
		case req.URL.Path == base+"/at1/roles/r1":
			return nativeOAuth1JSON(200, `{"role":{"id":"r1","name":"member","domain_id":"default"}}`)
		}
		return nativeOAuth1JSON(200, `{"access_token":`+nativeOAuth1AccessToken+`}`)
	})
	authorized, err := api.AuthorizeToken(ctx, "rt1", oauth1.AuthorizeTokenOpts{Roles: []oauth1.Role{{ID: "r1"}, {Name: "member"}}}, oauth1.WithAuthorizeTokenField("x_extension", true))
	if err != nil || authorized.OAuthVerifier != "verifier" {
		t.Fatal(authorized, err)
	}
	// Roles has no omitempty, so an empty request sends roles as null.
	if _, err := api.AuthorizeToken(ctx, "rt1", oauth1.AuthorizeTokenOpts{}); err != nil {
		t.Fatal(err)
	}
	expires := time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)
	token, err := api.GetAccessToken(ctx, "u1", "at1")
	if err != nil || token.ID != "at1" || token.ConsumerID != "c1" || token.ProjectID != "p1" || token.AuthorizingUserID != "u1" || !token.ExpiresAt.Equal(expires) {
		t.Fatal(token, err)
	}
	var ids []string
	for value, err := range api.ListAccessTokens(ctx, "u1") {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	var roles []oauth1.AccessTokenRole
	for value, err := range api.ListAccessTokenRoles(ctx, "u1", "at1") {
		if err != nil {
			t.Fatal(err)
		}
		roles = append(roles, *value)
	}
	role, err := api.GetAccessTokenRole(ctx, "u1", "at1", "r1")
	if err != nil || *role != (oauth1.AccessTokenRole{ID: "r1", Name: "member", DomainID: "default"}) {
		t.Fatal(role, err)
	}
	if err := api.RevokeAccessToken(ctx, "u1", "at1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"at1", "at2"}) || !reflect.DeepEqual(roles, []oauth1.AccessTokenRole{*role}) {
		t.Fatal(ids, roles)
	}
	base := "/keystone/v3/users/u1/OS-OAUTH1/access_tokens"
	want := []nativeOAuth1Call{
		// The body has no envelope, so extensions land at the root next to roles.
		{http.MethodPut, "/keystone/v3/OS-OAUTH1/authorize/rt1", "", `{"roles":[{"id":"r1"},{"name":"member"}],"x_extension":true}`},
		{http.MethodPut, "/keystone/v3/OS-OAUTH1/authorize/rt1", "", `{"roles":null}`},
		{http.MethodGet, base + "/at1", "", ""},
		{http.MethodGet, base, "", ""},
		{http.MethodGet, "/other/tokens", "page=2", ""},
		{http.MethodGet, base + "/at1/roles", "", ""},
		{http.MethodGet, base + "/at1/roles/r1", "", ""},
		{http.MethodDelete, base + "/at1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeOAuth1SignedRequestAndAccessTokens(t *testing.T) {
	ctx := context.Background()
	stamp := time.Unix(1760000000, 0)
	form := http.Header{"Content-Type": {oauth1.OAuth1TokenContentType}}
	var calls []nativeOAuth1Call
	var headers []http.Header
	api, cloud := nativeOAuth1API(t, &calls, &headers, func(req *http.Request) nativeOAuth1Reply {
		return nativeOAuth1Reply{201, form, "oauth_token=tok&oauth_token_secret=sec&oauth_expires_at=2026-10-11T01%3A02%3A03.000000Z"}
	})
	expires := time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)

	requested, err := api.RequestToken(ctx, oauth1.RequestTokenOpts{
		OAuthConsumerKey: "ck", OAuthConsumerSecret: "c&s", OAuthSignatureMethod: oauth1.HMACSHA1,
		OAuthTimestamp: &stamp, OAuthNonce: "nonce", RequestedProjectID: "p1",
	}, oauth1.WithRequestTokenHeader("x-vendor", "1"))
	if err != nil || requested.OAuthToken != "tok" || requested.OAuthTokenSecret != "sec" || !requested.OAuthExpiresAt.Equal(expires) {
		t.Fatal(requested, err)
	}
	params := url.Values{
		"oauth_callback": {"oob"}, "oauth_consumer_key": {"ck"}, "oauth_nonce": {"nonce"},
		"oauth_signature_method": {"HMAC-SHA1"}, "oauth_timestamp": {"1760000000"}, "oauth_version": {"1.0"},
	}
	signature := nativeOAuth1Sign(http.MethodPost, cloud.Server.URL+"/keystone/v3/OS-OAUTH1/request_token", params, "c&s")
	wantAuth := `OAuth oauth_callback="oob", oauth_consumer_key="ck", oauth_nonce="nonce", oauth_signature_method="HMAC-SHA1", oauth_timestamp="1760000000", oauth_version="1.0", oauth_signature="` + signature + `"`
	if got := headers[0].Get("Authorization"); got != wantAuth {
		t.Fatalf("%s\n%s", got, wantAuth)
	}
	if headers[0].Get("Requested-Project-Id") != "p1" || headers[0].Get("X-Vendor") != "1" {
		t.Fatal(headers[0])
	}

	// PLAINTEXT signs with the escaped secrets joined by '&', escaped once more.
	access, err := api.CreateAccessToken(ctx, oauth1.CreateAccessTokenOpts{
		OAuthConsumerKey: "ck", OAuthConsumerSecret: "cs", OAuthToken: "tok", OAuthTokenSecret: "s/t",
		OAuthVerifier: "v", OAuthSignatureMethod: oauth1.PLAINTEXT, OAuthTimestamp: &stamp, OAuthNonce: "nonce",
	}, oauth1.WithCreateAccessTokenHeader("x-vendor", "2"))
	if err != nil || access.OAuthToken != "tok" {
		t.Fatal(access, err)
	}
	wantAuth = `OAuth oauth_consumer_key="ck", oauth_nonce="nonce", oauth_signature_method="PLAINTEXT", oauth_timestamp="1760000000", oauth_token="tok", oauth_verifier="v", oauth_version="1.0", oauth_signature="cs%26s%252Ft"`
	if got := headers[1].Get("Authorization"); got != wantAuth || headers[1].Get("X-Vendor") != "2" {
		t.Fatalf("%s\n%s", got, wantAuth)
	}

	// Without a nonce or timestamp the native code generates both; only the parameter set is stable.
	if _, err := api.RequestToken(ctx, oauth1.RequestTokenOpts{OAuthConsumerKey: "ck", OAuthSignatureMethod: oauth1.HMACSHA1}); err != nil {
		t.Fatal(err)
	}
	generated := nativeOAuth1Params(t, headers[2].Get("Authorization"))
	keys := slices.Sorted(func(yield func(string) bool) {
		for key := range generated {
			if !yield(key) {
				return
			}
		}
	})
	if !reflect.DeepEqual(keys, []string{"oauth_callback", "oauth_consumer_key", "oauth_nonce", "oauth_signature", "oauth_signature_method", "oauth_timestamp", "oauth_version"}) ||
		generated["oauth_consumer_key"] != "ck" || generated["oauth_callback"] != "oob" || generated["oauth_nonce"] == "" || generated["oauth_signature"] == "" ||
		headers[2].Get("Requested-Project-Id") != "" {
		t.Fatal(generated, headers[2])
	}

	want := []nativeOAuth1Call{
		{http.MethodPost, "/keystone/v3/OS-OAUTH1/request_token", "", ""},
		{http.MethodPost, "/keystone/v3/OS-OAUTH1/access_token", "", ""},
		{http.MethodPost, "/keystone/v3/OS-OAUTH1/request_token", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	for i, header := range headers {
		// Both calls are signed by OAuth but still carry the provider token.
		if header.Get("X-Auth-Token") != "test-token" {
			t.Fatal(i, header)
		}
	}
}

func TestNativeOAuth1TokenResponseDecodeFailures(t *testing.T) {
	ctx := context.Background()
	stamp := time.Unix(1760000000, 0)
	for name, reply := range map[string]nativeOAuth1Reply{
		// The Content-Type must match exactly, so a charset parameter is rejected.
		"json":    {201, http.Header{"Content-Type": {"application/json"}}, `{}`},
		"charset": {201, http.Header{"Content-Type": {oauth1.OAuth1TokenContentType + "; charset=utf-8"}}, "oauth_token=t"},
		"expires": {201, http.Header{"Content-Type": {oauth1.OAuth1TokenContentType}}, "oauth_token=t&oauth_expires_at=2026-10-11T01:02:03%2B09:00"},
		"query":   {201, http.Header{"Content-Type": {oauth1.OAuth1TokenContentType}}, "oauth_token=%zz"},
	} {
		t.Run(name, func(t *testing.T) {
			var calls []nativeOAuth1Call
			api, _ := nativeOAuth1API(t, &calls, nil, func(*http.Request) nativeOAuth1Reply { return reply })
			token, err := api.RequestToken(ctx, oauth1.RequestTokenOpts{OAuthConsumerKey: "ck", OAuthSignatureMethod: oauth1.HMACSHA1, OAuthTimestamp: &stamp})
			if token != nil || len(calls) != 1 {
				t.Fatal(token, calls)
			}
			nativeOAuth1Operation(t, err, "RequestToken")
			token, err = api.CreateAccessToken(ctx, oauth1.CreateAccessTokenOpts{OAuthConsumerKey: "ck", OAuthToken: "t", OAuthVerifier: "v", OAuthSignatureMethod: oauth1.HMACSHA1})
			if token != nil || len(calls) != 2 {
				t.Fatal(token, calls)
			}
			nativeOAuth1Operation(t, err, "CreateAccessToken")
		})
	}
	t.Run("empty form", func(t *testing.T) {
		var calls []nativeOAuth1Call
		api, _ := nativeOAuth1API(t, &calls, nil, func(*http.Request) nativeOAuth1Reply {
			return nativeOAuth1Reply{201, http.Header{"Content-Type": {oauth1.OAuth1TokenContentType}}, ""}
		})
		token, err := api.RequestToken(ctx, oauth1.RequestTokenOpts{OAuthConsumerKey: "ck", OAuthSignatureMethod: oauth1.PLAINTEXT})
		if err != nil || *token != (oauth1.Token{}) {
			t.Fatal(token, err)
		}
	})
}

func TestNativeOAuth1CreateAuthTokenBodyHeadersAndSubjectToken(t *testing.T) {
	ctx := context.Background()
	stamp := time.Unix(1760000000, 0)
	var calls []nativeOAuth1Call
	var headers []http.Header
	api, cloud := nativeOAuth1API(t, &calls, &headers, func(*http.Request) nativeOAuth1Reply {
		return nativeOAuth1Reply{201, http.Header{"Content-Type": {"application/json"}, "X-Subject-Token": {"issued"}},
			`{"token":{"expires_at":"2026-10-11T01:02:03.000000Z","methods":["oauth1"],"OS-OAUTH1":{"access_token_id":"at1","consumer_id":"c1"}}}`}
	})
	opts := oauth1.AuthOptions{
		OAuthConsumerKey: "ck", OAuthConsumerSecret: "cs", OAuthToken: "at1", OAuthTokenSecret: "ts",
		OAuthSignatureMethod: oauth1.HMACSHA1, OAuthTimestamp: &stamp, OAuthNonce: "nonce", AllowReauth: true,
	}
	token, err := api.Create(ctx, opts, oauth1.WithCreateField("x_extension", 1), oauth1.WithCreateHeader("x-vendor", "1"))
	// The ID comes from X-Subject-Token; the OS-OAUTH1 token extension is not part of the returned tokens.Token.
	if err != nil || token.ID != "issued" || !token.ExpiresAt.Equal(time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)) {
		t.Fatal(token, err)
	}
	// Creating a token does not touch the provider token or its reauthentication.
	if cloud.Provider.Token() != "test-token" || cloud.Provider.ReauthFunc != nil {
		t.Fatal(cloud.Provider.Token())
	}
	params := url.Values{
		"oauth_consumer_key": {"ck"}, "oauth_nonce": {"nonce"}, "oauth_signature_method": {"HMAC-SHA1"},
		"oauth_timestamp": {"1760000000"}, "oauth_token": {"at1"}, "oauth_version": {"1.0"},
	}
	signature := nativeOAuth1Sign(http.MethodPost, cloud.Server.URL+"/keystone/v3/auth/tokens", params, "cs", "ts")
	wantAuth := `OAuth oauth_consumer_key="ck", oauth_nonce="nonce", oauth_signature_method="HMAC-SHA1", oauth_timestamp="1760000000", oauth_token="at1", oauth_version="1.0", oauth_signature="` + signature + `"`
	if headers[0].Get("Authorization") != wantAuth || headers[0].Get("X-Vendor") != "1" {
		t.Fatalf("%s\n%s", headers[0].Get("Authorization"), wantAuth)
	}
	// The native code asks for an empty X-Auth-Token, but the provider token is applied afterwards.
	if !reflect.DeepEqual(headers[0].Values("X-Auth-Token"), []string{"test-token"}) {
		t.Fatal(headers[0])
	}
	cloud.Provider.SetToken("")
	if _, err := api.Create(ctx, opts); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(headers[1].Values("X-Auth-Token"), []string{""}) {
		t.Fatal(headers[1])
	}
	want := []nativeOAuth1Call{
		// Extensions land inside auth, not in identity or a scope object.
		{http.MethodPost, "/keystone/v3/auth/tokens", "", `{"auth":{"identity":{"methods":["oauth1"],"oauth1":{}},"x_extension":1}}`},
		{http.MethodPost, "/keystone/v3/auth/tokens", "", `{"auth":{"identity":{"methods":["oauth1"],"oauth1":{}}}}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeOAuth1StatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	request := oauth1.RequestTokenOpts{OAuthConsumerKey: "ck", OAuthSignatureMethod: oauth1.HMACSHA1}
	access := oauth1.CreateAccessTokenOpts{OAuthConsumerKey: "ck", OAuthToken: "t", OAuthVerifier: "v", OAuthSignatureMethod: oauth1.HMACSHA1}
	auth := oauth1.AuthOptions{OAuthConsumerKey: "ck", OAuthToken: "t", OAuthSignatureMethod: oauth1.HMACSHA1}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*oauth1.API) error
	}{
		{"CreateConsumer", []int{201}, func(api *oauth1.API) error {
			_, err := api.CreateConsumer(ctx, oauth1.CreateConsumerOpts{})
			return err
		}},
		{"GetConsumer", []int{200}, func(api *oauth1.API) error { _, err := api.GetConsumer(ctx, "c1"); return err }},
		{"UpdateConsumer", []int{200}, func(api *oauth1.API) error {
			_, err := api.UpdateConsumer(ctx, "c1", oauth1.UpdateConsumerOpts{})
			return err
		}},
		{"DeleteConsumer", []int{202, 204}, func(api *oauth1.API) error { return api.DeleteConsumer(ctx, "c1") }},
		{"RequestToken", []int{201}, func(api *oauth1.API) error { _, err := api.RequestToken(ctx, request); return err }},
		{"AuthorizeToken", []int{200}, func(api *oauth1.API) error {
			_, err := api.AuthorizeToken(ctx, "rt1", oauth1.AuthorizeTokenOpts{})
			return err
		}},
		{"CreateAccessToken", []int{201}, func(api *oauth1.API) error { _, err := api.CreateAccessToken(ctx, access); return err }},
		{"GetAccessToken", []int{200}, func(api *oauth1.API) error { _, err := api.GetAccessToken(ctx, "u1", "at1"); return err }},
		{"RevokeAccessToken", []int{202, 204}, func(api *oauth1.API) error { return api.RevokeAccessToken(ctx, "u1", "at1") }},
		{"GetAccessTokenRole", []int{200}, func(api *oauth1.API) error {
			_, err := api.GetAccessTokenRole(ctx, "u1", "at1", "r1")
			return err
		}},
		{"Create", []int{201}, func(api *oauth1.API) error { _, err := api.Create(ctx, auth); return err }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeOAuth1Call
				api, _ := nativeOAuth1API(t, &calls, nil, func(*http.Request) nativeOAuth1Reply { return nativeOAuth1JSON(code, `{}`) })
				err := call.call(api)
				nativeOAuth1Operation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, list := range []struct {
			name string
			key  string
			call func(*oauth1.API) []error
		}{
			{"ListConsumers", "consumers", func(api *oauth1.API) (errs []error) {
				for _, err := range api.ListConsumers(ctx) {
					errs = append(errs, err)
				}
				return errs
			}},
			{"ListAccessTokens", "access_tokens", func(api *oauth1.API) (errs []error) {
				for _, err := range api.ListAccessTokens(ctx, "u1") {
					errs = append(errs, err)
				}
				return errs
			}},
			{"ListAccessTokenRoles", "roles", func(api *oauth1.API) (errs []error) {
				for _, err := range api.ListAccessTokenRoles(ctx, "u1", "at1") {
					errs = append(errs, err)
				}
				return errs
			}},
		} {
			for _, tc := range []struct {
				code int
				body string
			}{{404, `{}`}, {200, `{"` + list.key + `":[]}`}, {204, ""}} {
				var calls []nativeOAuth1Call
				api, _ := nativeOAuth1API(t, &calls, nil, func(*http.Request) nativeOAuth1Reply { return nativeOAuth1JSON(tc.code, tc.body) })
				errs := list.call(api)
				var native gophercloud.ErrUnexpectedResponseCode
				switch {
				case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
				case tc.code == 200 && len(errs) == 0:
				case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
				default:
					t.Fatal(list.name, tc.code, errs)
				}
				if len(calls) != 1 {
					t.Fatal(list.name, calls)
				}
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeOAuth1Call
		api, _ := nativeOAuth1API(t, &calls, nil, func(*http.Request) nativeOAuth1Reply { return nativeOAuth1JSON(201, `{}`) })
		noKey := request
		noKey.OAuthConsumerKey = ""
		noMethod := request
		noMethod.OAuthSignatureMethod = ""
		noVerifier := access
		noVerifier.OAuthVerifier = ""
		noToken := access
		noToken.OAuthToken = ""
		noAuthToken := auth
		noAuthToken.OAuthToken = ""
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"consumer core extension": {"CreateConsumer", func() error {
				_, err := api.CreateConsumer(ctx, oauth1.CreateConsumerOpts{}, oauth1.WithCreateConsumerField("description", "x"))
				return err
			}()},
			"consumer nil option": {"CreateConsumer", func() error {
				_, err := api.CreateConsumer(ctx, oauth1.CreateConsumerOpts{}, nil)
				return err
			}()},
			"update core extension": {"UpdateConsumer", func() error {
				_, err := api.UpdateConsumer(ctx, "c1", oauth1.UpdateConsumerOpts{}, oauth1.WithUpdateConsumerField("description", "x"))
				return err
			}()},
			"request consumer key": {"RequestToken", func() error { _, err := api.RequestToken(ctx, noKey); return err }()},
			"request method":       {"RequestToken", func() error { _, err := api.RequestToken(ctx, noMethod); return err }()},
			"request authorization header": {"RequestToken", func() error {
				_, err := api.RequestToken(ctx, request, oauth1.WithRequestTokenHeader("authorization", "x"))
				return err
			}()},
			// Requested-Project-Id is reserved by its h tag even when the field is empty.
			"request project header": {"RequestToken", func() error {
				_, err := api.RequestToken(ctx, request, oauth1.WithRequestTokenHeader("Requested-Project-Id", "p"))
				return err
			}()},
			"request invalid header": {"RequestToken", func() error {
				_, err := api.RequestToken(ctx, request, oauth1.WithRequestTokenHeader("bad header", "x"))
				return err
			}()},
			"authorize empty role": {"AuthorizeToken", func() error {
				_, err := api.AuthorizeToken(ctx, "rt1", oauth1.AuthorizeTokenOpts{Roles: []oauth1.Role{{ID: "r1"}, {}}})
				return err
			}()},
			"authorize core extension": {"AuthorizeToken", func() error {
				_, err := api.AuthorizeToken(ctx, "rt1", oauth1.AuthorizeTokenOpts{}, oauth1.WithAuthorizeTokenField("roles", []string{}))
				return err
			}()},
			"access verifier": {"CreateAccessToken", func() error { _, err := api.CreateAccessToken(ctx, noVerifier); return err }()},
			"access token":    {"CreateAccessToken", func() error { _, err := api.CreateAccessToken(ctx, noToken); return err }()},
			"access authorization header": {"CreateAccessToken", func() error {
				_, err := api.CreateAccessToken(ctx, access, oauth1.WithCreateAccessTokenHeader("Authorization", "x"))
				return err
			}()},
			"create token": {"Create", func() error { _, err := api.Create(ctx, noAuthToken); return err }()},
			"create core extension": {"Create", func() error {
				_, err := api.Create(ctx, auth, oauth1.WithCreateField("identity", map[string]any{}))
				return err
			}()},
			"create auth token header": {"Create", func() error {
				_, err := api.Create(ctx, auth, oauth1.WithCreateHeader("x-auth-token", "x"))
				return err
			}()},
			"create nil option": {"Create", func() error { _, err := api.Create(ctx, auth, nil); return err }()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeOAuth1Operation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
