package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/federation"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/policies"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func identitySharedCallsClient(t *testing.T, reply func(*http.Request) (int, string)) (*testcloud.Cloud, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	return cloud, &calls
}

// Shared calls that a category admits but an identity operation never
// supported fail before HTTP.
func TestIdentitySharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud, calls := identitySharedCallsClient(t, func(*http.Request) (int, string) { return 500, `{}` })
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"

	// OS-FEDERATION records: replacing headers only, no microversion.
	idp := identityproviders.New(client)
	_, err := idp.Get(ctx, "x", request.Microversion("3.0"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = idp.Get(ctx, "x", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = idp.All(ctx, request.Microversion("3.0"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = idp.Find(ctx, "x", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	err = federation.New(client).DeleteMappingRecord(ctx, "m", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = federation.New(client).FindSourceMappingRecord(ctx, "m", request.Query("k", "v"))
	requireUnsupportedCall(t, err, "Query")

	// Endpoint groups, application credentials and trusts.
	_, err = endpointgroups.New(client).Get(ctx, "id", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = applicationcredentials.New(client).CreateCredentialRecord(ctx, "u", "n", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	for _, err := range applicationcredentials.New(client).ListCredentialRecords(ctx, "u", request.Query("name", "x")) {
		requireUnsupportedCall(t, err, "Query")
	}
	_, err = trusts.New(client).CreateRecord(ctx, request.Microversion("3.0"))
	requireUnsupportedCall(t, err, "Microversion")

	// Record lists, lookups and writes.
	_, err = regions.New(client).AllRecords(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	// Query options that check the key before marshaling the value stay
	// operation-specific (WithRecordListQuery), so request.Query is rejected.
	_, err = regions.New(client).AllRecords(ctx, request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = users.New(client).AllRecords(ctx, request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = domains.New(client).CreateRecord(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = projects.New(client).ResolveRecord(ctx, "p", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	for _, call := range []policies.RecordListOption{request.Headers(nil), request.Query("type", "x"), request.Microversion("3.0")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		_, err = policies.New(client).AllRecords(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
	}
	for _, call := range []users.ListProjectRecordsOption{request.Limit(1), request.Marker("m"), request.Headers(nil)} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		for _, err := range users.New(client).ListProjectRecords(ctx, "u", call) {
			requireUnsupportedCall(t, err, entry.Kind.String())
		}
	}

	// Cloud-layer reads: Headers and Microversion only.
	_, err = identity.NewDomainReadCloud(domains.New(client)).GetDomain(ctx, identity.WithDomainGetID("d"), request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = identity.NewUserReadCloud(users.New(client)).ListUsers(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Replacing headers and paging calls apply in caller order: the later call
// wins.
func TestIdentitySharedCallsApplyInCallerOrder(t *testing.T) {
	var header http.Header
	var query string
	cloud, _ := identitySharedCallsClient(t, func(r *http.Request) (int, string) {
		header, query = r.Header.Clone(), r.URL.RawQuery
		return 200, `{"identity_providers":[]}`
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"

	_, err := identityproviders.New(client).All(context.Background(),
		request.Headers(map[string]string{"X-First": "1"}), request.Limit(1), request.Marker("a"),
		request.Headers(map[string]string{"X-Second": "2"}), request.Limit(2), request.Marker("b"))
	if err != nil {
		t.Fatal(err)
	}
	if header.Get("X-First") != "" || header.Get("X-Second") != "2" || query != "limit=2&marker=b" {
		t.Fatal(header, query)
	}
}
