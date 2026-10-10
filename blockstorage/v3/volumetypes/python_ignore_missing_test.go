package volumetypes_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// resource.IgnoreMissing gives these direct deletes the Proxy default
// ignore_missing=True: a 404 becomes success and other failures stay errors.
func TestPythonDeleteIgnoreMissingTypeEncryption(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*volumetypes.API) error
		want string
	}{
		{"delete_type_encryption", func(api *volumetypes.API) error { return api.DeleteEncryption(context.Background(), "t-1", "e-1") }, "DELETE /svc/types/t-1/encryption/e-1?"},
	} {
		for _, code := range []int{204, 404, 409} {
			var calls []string
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
				w.WriteHeader(code)
			})
			err := resource.IgnoreMissing(tc.call(volumetypes.New(cloud.Client("block-storage", "/svc/"))))
			if (code == 409) != (err != nil) || !reflect.DeepEqual(calls, []string{tc.want}) {
				t.Fatal(tc.name, code, err, calls)
			}
		}
	}
}
