package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const orderListPath = "/reverse/barbican/v1/orders"

type orderListFixture struct {
	name string
	open func(*testing.T, *testcloud.Cloud) *orders.API
}

func orderListFixtures() []orderListFixture {
	return []orderListFixture{
		{"leaf", func(t *testing.T, c *testcloud.Cloud) *orders.API { return orders.New(secretFetchClient(c)) }},
		{"cached connection", func(t *testing.T, c *testcloud.Cloud) *orders.API {
			conn, err := sdk.FromProvider(c.Provider, sdk.WithEndpoint(sdk.KeyManager, c.Server.URL+"/catalog/v1/"))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.KeyManagerV1(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			again, err := conn.KeyManager(context.Background())
			if err != nil || again != service || service.Orders.RawClient() != service.RawClient() || service.RawClient().ProviderClient != c.Provider {
				t.Fatal(service, again, err)
			}
			service.RawClient().ResourceBase = c.Server.URL + "/reverse/barbican/v1/"
			return service.Orders
		}},
	}
}

// Explicit ID routes already retain the caller's ID. This proves that contract;
// it does not claim a previously observed wrong HTTP route or name support.
func TestKeyManagerOrderIdentityExplicitRoutesIgnorePassiveReferences(t *testing.T) {
	for _, f := range orderListFixtures() {
		for _, operation := range []string{"Find", "Remove", "Wait", "WaitDeleted"} {
			t.Run(f.name+"/"+operation, func(t *testing.T) {
				c, foreign := testcloud.New(t), testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
				c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					n := calls.Add(1)
					if r.URL.Path != orderListPath+"/order-alpha" || len(r.URL.Query()) != 0 {
						t.Error(r.URL)
					}
					if operation == "Remove" {
						if r.Method != http.MethodDelete {
							t.Error(r.Method)
						}
						w.WriteHeader(204)
						return
					}
					if r.Method != http.MethodGet {
						t.Error(r.Method)
					}
					if operation == "WaitDeleted" && n == 2 {
						testcloud.JSON(w, 404, `{"error":"gone"}`)
						return
					}
					status := "ACTIVE"
					if operation == "WaitDeleted" || operation == "Wait" && n == 1 {
						status = "PENDING"
					}
					body := fmt.Sprintf(`{"order_ref":%q,"secret_ref":%q,"status":%q,"meta":{"name":"meta-name-only"}}`, foreign.Server.URL+fmt.Sprintf("/orders/different-%d", n), foreign.Server.URL+fmt.Sprintf("/secrets/secret-%d", n), status)
					testcloud.JSON(w, 200, body)
				})
				api := f.open(t, c)
				ref := resource.ID("order-alpha")
				wait := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second)}
				var err error
				want := int32(1)
				switch operation {
				case "Find":
					var value *orders.Order
					value, err = api.Find(context.Background(), ref)
					if value == nil || value.Meta.Name != "meta-name-only" || value.OrderRef != foreign.Server.URL+"/orders/different-1" {
						t.Fatal(value, err)
					}
				case "Remove":
					err = api.Remove(context.Background(), ref)
				case "Wait":
					want = 2
					var value *orders.Order
					value, err = api.WaitFor(context.Background(), ref, "ACTIVE", wait...)
					if value == nil || value.OrderRef != foreign.Server.URL+"/orders/different-2" || value.SecretRef != foreign.Server.URL+"/secrets/secret-2" {
						t.Fatal(value, err)
					}
				case "WaitDeleted":
					want = 2
					err = api.WaitForDeletion(context.Background(), ref, wait...)
				}
				if err != nil || calls.Load() != want || followed.Load() != 0 {
					t.Fatal(err, calls.Load(), followed.Load())
				}
			})
		}
	}
}

func TestKeyManagerOrderIdentityNameUnsupportedWithoutHTTP(t *testing.T) {
	for _, f := range orderListFixtures() {
		c := testcloud.New(t)
		var calls atomic.Int32
		c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			testcloud.JSON(w, 200, `{"order_ref":"https://foreign.invalid/orders/ref","secret_ref":"https://foreign.invalid/secrets/secret","meta":{"name":"meta-name-only"},"status":"ACTIVE"}`)
		})
		api := f.open(t, c)
		ref := resource.Name("meta-name-only")
		_, find := api.Find(context.Background(), ref)
		remove := api.Remove(context.Background(), ref)
		_, wait := api.WaitFor(context.Background(), ref, "ACTIVE")
		deleted := api.WaitForDeletion(context.Background(), ref)
		for _, err := range []error{find, remove, wait, deleted} {
			if !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(err)
			}
		}
		if calls.Load() != 0 {
			t.Fatal("Meta.Name was promoted to resource name", calls.Load())
		}
	}
}
