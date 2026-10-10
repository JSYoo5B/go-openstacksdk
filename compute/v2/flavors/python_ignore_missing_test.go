package flavors_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// resource.IgnoreMissing gives these direct deletes the Proxy default
// ignore_missing=True: a 404 becomes success and other failures stay errors.
func TestPythonDeleteIgnoreMissingFlavorExtraSpec(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*flavors.API) error
		want string
	}{
		{"delete_flavor_extra_specs_property", func(api *flavors.API) error { return api.DeleteExtraSpec(context.Background(), "f-1", "hw:cpu") }, "DELETE /svc/flavors/f-1/os-extra_specs/hw:cpu?"},
	} {
		// Nova accepts only 200 for this DELETE.
		for _, code := range []int{200, 404, 409} {
			var calls []string
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
				w.WriteHeader(code)
			})
			err := resource.IgnoreMissing(tc.call(flavors.New(cloud.Client("compute", "/svc/"))))
			if (code == 409) != (err != nil) || !reflect.DeepEqual(calls, []string{tc.want}) {
				t.Fatal(tc.name, code, err, calls)
			}
		}
	}
}
