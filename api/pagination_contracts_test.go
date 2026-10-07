package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestSwiftMarkerCycleCannotLoopForever(t *testing.T) {
	for _, body := range []string{`[{"name":"same","bytes":1}]`, `[{"subdir":"folder/"}]`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /swift/container", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, body) // Ignores marker or returns an unusable one.
			})
			failures := 0
			for value, err := range objects.New(cloud.Client("object-store", "/swift")).List(context.Background(), "container") {
				if err != nil {
					failures++
					if value != nil || !errors.Is(err, resource.ErrPaginationCycle) {
						t.Fatalf("value=%v err=%v", value, err)
					}
				}
			}
			if failures != 1 || calls.Load() != 2 {
				t.Fatalf("failures=%d calls=%d", failures, calls.Load())
			}
		})
	}
}
