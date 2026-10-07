package rest_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

func TestJSONRetryPreservesSDKOwnedHeaders(t *testing.T) {
	for _, mutation := range []string{"remove", "change", "add"} {
		t.Run(mutation, func(t *testing.T) {
			var attempts atomic.Int32
			client := limitsGuardedClient(func(r *http.Request) (*http.Response, error) {
				attempts.Add(1)
				if r.Header.Get("If-Match") != "revision_number=0" {
					t.Error(r.Header)
				}
				return limitsGuardedWire(503, io.NopCloser(strings.NewReader("original denied retry"))), nil
			})
			client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				switch mutation {
				case "remove":
					delete(opts.MoreHeaders, "If-Match")
				case "change":
					opts.MoreHeaders["If-Match"] = "revision_number=5"
				case "add":
					opts.MoreHeaders["if-match"] = "conflicting case"
				}
				return nil
			}
			_, err := rest.DoJSONGuardedHeaders(context.Background(), client, nil, http.MethodPut, limitsGuardedTarget,
				map[string]string{"target": "original"}, map[string]string{"If-Match": "revision_number=0"}, 200)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "original denied retry" || attempts.Load() != 1 {
				t.Fatalf("err=%v attempts=%d", err, attempts.Load())
			}
		})
	}
}
