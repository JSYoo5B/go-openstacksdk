package qos_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// resource.IgnoreMissing gives these direct deletes the Proxy default
// ignore_missing=True: a 404 becomes success and other failures stay errors.
func TestPythonDeleteIgnoreMissingQoSSpec(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*qos.API) error
		want string
	}{
		{"delete_qos_spec force", func(api *qos.API) error {
			return api.Delete(context.Background(), "q-1", qos.WithDeleteOptions(qos.DeleteOpts{Force: true}))
		}, "DELETE /svc/qos-specs/q-1?force=true"},
	} {
		for _, code := range []int{204, 404, 409} {
			var calls []string
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
				w.WriteHeader(code)
			})
			err := resource.IgnoreMissing(tc.call(qos.New(cloud.Client("block-storage", "/svc/"))))
			if (code == 409) != (err != nil) || !reflect.DeepEqual(calls, []string{tc.want}) {
				t.Fatal(tc.name, code, err, calls)
			}
		}
	}
}
