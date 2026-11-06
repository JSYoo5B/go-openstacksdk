package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func clusteringSharedCallsClient(t *testing.T, reply func(*http.Request) (int, string)) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.12"
	return client, &calls
}

// Shared calls that a category admits but a Senlin operation never supported
// fail before HTTP.
func TestClusteringSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	client, calls := clusteringSharedCallsClient(t, func(*http.Request) (int, string) { return 500, `{}` })

	// Lists take Header, Microversion, Query, MaxItems and Paginated only.
	for _, call := range []request.ListCall{request.Limit(1), request.Marker("m"), request.Headers(nil)} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		_, err = profiles.New(client).All(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
		_, err = services.New(client).All(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
	}

	// FindIdentity takes no query and no replacing headers.
	_, err := policies.New(client).FindIdentity(ctx, "x", request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = receivers.New(client).FindIdentity(ctx, "x", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	// Writes take Header and Field only.
	_, err = profiles.New(client).Create(ctx, profiles.CreateOpts{Name: "p", Spec: []byte(`{}`)}, request.Microversion("1.2"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = clusters.New(client).ScaleIn(ctx, resource.ID("c"), clusters.ScaleInOpts{}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = actions.New(client).Cancel(ctx, resource.ID("a"), request.Microversion("1.12"))
	requireUnsupportedCall(t, err, "Microversion")
	// Preview and metadata helpers never took extension fields.
	_, err = nodes.New(client).AdoptPreview(ctx, nodes.AdoptPreviewOpts{Identity: "i", Type: "t"}, request.Field("vendor", true))
	requireUnsupportedCall(t, err, "Field")
	_, err = clusters.New(client).SetMetadata(ctx, resource.ID("c"), map[string]any{"k": "v"}, request.Field("vendor", true))
	requireUnsupportedCall(t, err, "Field")

	// Node reads take Header and Query; node deletes Header and IgnoreMissing.
	_, err = nodes.New(client).Get(ctx, "n", request.Microversion("1.12"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = nodes.New(client).Delete(ctx, resource.ID("n"), request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Shared calls apply in caller order: the later call wins.
func TestClusteringSharedCallsApplyInCallerOrder(t *testing.T) {
	var header http.Header
	var query string
	client, calls := clusteringSharedCallsClient(t, func(r *http.Request) (int, string) {
		header, query = r.Header.Clone(), r.URL.RawQuery
		return 200, `{"profiles":[{"id":"a","name":"a"},{"id":"b","name":"b"}]}`
	})
	values, err := profiles.New(client).All(context.Background(),
		request.Header("X-Vendor", "first"), request.Query("vendor", "first"), request.MaxItems(5), request.Paginated(true), request.Microversion("1.2"),
		request.Header("X-Vendor", "second"), request.Query("vendor", "second"), request.MaxItems(1), request.Paginated(false), request.Microversion("1.7"))
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || calls.Load() != 1 || header.Get("X-Vendor") != "second" || header.Get("OpenStack-API-Version") != "clustering 1.7" || query != "limit=1&vendor=second" {
		t.Fatal(len(values), calls.Load(), header, query)
	}

	// IgnoreMissing: a later false turns the default nil result into an error.
	client, _ = clusteringSharedCallsClient(t, func(*http.Request) (int, string) { return 404, `{}` })
	value, err := nodes.New(client).FindIdentity(context.Background(), "missing", request.IgnoreMissing(true), request.IgnoreMissing(false), nodes.WithFindFallback(resource.FindFallbackNever))
	if value != nil || !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(value, err)
	}
}
