package resourceproviders_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// resource.IgnoreMissing gives these direct deletes the Proxy default
// ignore_missing=True: a 404 becomes success and other failures stay errors.
func TestPythonDeleteIgnoreMissingResourceProvider(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*resourceproviders.API) error
		want string
	}{
		{"delete_resource_provider_inventory", func(api *resourceproviders.API) error {
			return api.DeleteInventory(context.Background(), "rp-1", "VCPU")
		}, "DELETE /svc/resource_providers/rp-1/inventories/VCPU?"},
		{"delete_resource_provider_trait", func(api *resourceproviders.API) error { return api.DeleteTraits(context.Background(), "rp-1") }, "DELETE /svc/resource_providers/rp-1/traits?"},
	} {
		for _, code := range []int{204, 404, 409} {
			var calls []string
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
				w.WriteHeader(code)
			})
			err := resource.IgnoreMissing(tc.call(resourceproviders.New(cloud.Client("placement", "/svc/"))))
			if (code == 409) != (err != nil) || !reflect.DeepEqual(calls, []string{tc.want}) {
				t.Fatal(tc.name, code, err, calls)
			}
		}
	}
}
