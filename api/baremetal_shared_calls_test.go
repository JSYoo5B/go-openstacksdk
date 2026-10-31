package api_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/chassis"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/runbooks"
	"github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/introspection"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func requireUnsupportedCall(t *testing.T, err error, kind string) {
	t.Helper()
	if !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), kind+" call option is unsupported by this operation") {
		t.Fatalf("%s: err = %v", kind, err)
	}
}

// Shared calls that a category admits but a bare metal operation never
// supported fail before HTTP.
func TestBaremetalSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	fake := r2IronicBFixed(t, cloud, 500, `{}`)
	client := r2IronicBClient(cloud)

	_, err := chassis.New(client).Create(ctx, chassis.CreateOpts{}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = chassis.New(client).Get(ctx, "c", request.Microversion("1.1"))
	requireUnsupportedCall(t, err, "Microversion")
	for _, err := range chassis.New(client).List(ctx, request.Limit(1)) {
		requireUnsupportedCall(t, err, "Limit")
	}
	_, err = runbooks.New(client).Create(ctx, runbooks.CreateOpts{Name: "n"}, request.Microversion("1.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = runbooks.New(client).Patch(ctx, "u", nil, request.Field("name", "x"))
	requireUnsupportedCall(t, err, "Field")
	_, err = nodes.New(client).RemoveTrait(ctx, "n1", "T", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = nodes.New(client).DetachVIF(ctx, "n1", "v", request.Custom(func(*int) error { return nil }))
	requireUnsupportedCall(t, err, "Custom")
	_, err = runbooks.New(client).Get(ctx, "u", request.Custom(func(*request.Config[runbooks.ListOpts]) error { return nil }))
	requireUnsupportedCall(t, err, "Custom")
	if calls := fake.all(); len(calls) != 0 {
		t.Fatalf("calls = %+v", calls)
	}

	inspector := introspection.New(r2IronicBInspectorClient(cloud))
	requireUnsupportedCall(t, inspector.AbortIntrospectionWithOptions(ctx, "n1", request.Header("X-Test", "1")), "Header")
}

// request.Paginated keeps the caller order of the removed WithListSinglePage:
// a later Paginated(true) restores paging.
func TestBaremetalRunbookListPaginatedCallerOrder(t *testing.T) {
	api, fake := r2IronicBRunbooks(t, 200, `{"runbooks":[]}`)
	for _, err := range api.List(context.Background(), request.Paginated(false), request.Paginated(true), request.Limit(1)) {
		t.Fatal(err)
	}
	call := fake.only(t)
	if call.RawQuery != "limit=1" {
		t.Fatalf("call = %+v", call)
	}
}
