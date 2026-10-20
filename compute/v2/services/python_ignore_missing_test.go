package services_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// resource.IgnoreMissing gives these direct deletes the Proxy default
// ignore_missing=True: a 404 becomes success and other failures stay errors.
func TestPythonDeleteIgnoreMissingService(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*services.API) error
		want string
	}{
		{"delete_service", func(api *services.API) error { return api.Delete(context.Background(), "s-1") }, "DELETE /svc/os-services/s-1?"},
	} {
		for _, code := range []int{204, 404, 409} {
			var calls []string
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
				w.WriteHeader(code)
			})
			err := resource.IgnoreMissing(tc.call(services.New(cloud.Client("compute", "/svc/"))))
			if (code == 409) != (err != nil) || !reflect.DeepEqual(calls, []string{tc.want}) {
				t.Fatal(tc.name, code, err, calls)
			}
		}
	}
}
