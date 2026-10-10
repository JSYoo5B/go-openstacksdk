package orders_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestOrderFindIdentityFollowsPythonFind(t *testing.T) {
	ctx := context.Background()
	run := func(reply func(n int32) *http.Response) (*orders.API, *atomic.Int32) {
		var calls atomic.Int32
		cloud := testcloud.New(t)
		cloud.Provider.HTTPClient.Transport = nativeOrderTransport(func(req *http.Request) (*http.Response, error) {
			return reply(calls.Add(1)), nil
		})
		return orders.New(nativeOrderClient(cloud)), &calls
	}
	api, calls := run(func(int32) *http.Response {
		return nativeOrderWire(200, `{"order_ref":"https://kms/v1/orders/o1","secret_ref":"https://kms/v1/secrets/s1"}`)
	})
	found, err := api.FindIdentity(ctx, "o1")
	if err != nil || found.RequestID != "o1" || *found.OrderID != "o1" || *found.SecretID != "s1" || calls.Load() != 1 {
		t.Fatal(found, err)
	}
	list := `{"orders":[{"order_ref":"https://kms/v1/orders/o2"},{"name":"named","order_ref":"https://kms/v1/orders/o3"}]}`
	api, calls = run(func(n int32) *http.Response {
		if n == 1 {
			return nativeOrderWire(404, `{}`)
		}
		return nativeOrderWire(200, list)
	})
	// Orders have no declared name; a body name still participates like Python.
	found, err = api.FindIdentity(ctx, "named")
	if err != nil || found == nil || *found.OrderID != "o3" || calls.Load() != 2 {
		t.Fatal(found, err)
	}
	api, _ = run(func(n int32) *http.Response {
		if n == 1 {
			return nativeOrderWire(404, `{}`)
		}
		return nativeOrderWire(200, list)
	})
	// The list id is the full order_ref, so a bare UUID only matches by GET.
	if found, err := api.FindIdentity(ctx, "o2", resource.WithIdentityFindIgnoreMissing(false)); found != nil || !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(found, err)
	}
}
