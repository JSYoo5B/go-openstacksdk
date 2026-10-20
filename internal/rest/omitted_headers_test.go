package rest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestHeaderPolicyStopsRetryThatOmitsOwnedVersion(t *testing.T) {
	for _, withCause := range []bool{false, true} {
		name := "native retry"
		if withCause {
			name = "callback cause"
		}
		t.Run(name, func(t *testing.T) {
			fake := testcloud.New(t)
			client := fake.Client("network", "/v2.0/")
			calls, callbacks := 0, 0
			fake.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("X-Request-Id", "first503")
				testcloud.JSON(w, 503, `{"error":"temporary"}`)
			})
			callbackCause := errors.New("callback cause")
			fake.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks++
				opts.OmitHeaders = []string{"oPeNsTaCk-ApI-vErSiOn"}
				if withCause || callbacks > 1 {
					return callbackCause
				}
				return nil
			}
			_, err := DoJSONGuardedRejectionsHeaderPolicy(context.Background(), client, nil, http.MethodGet, client.ServiceURL("networks"), nil, RequestHeaderPolicy{Values: map[string]string{"OpenStack-API-Version": "network 2.0"}}, RejectionPolicy{Codes: []int{503}, PreserveCleanRetry: true}, 200)
			var native gophercloud.ErrUnexpectedResponseCode
			if calls != 1 || callbacks != 1 || !errors.Is(err, resource.ErrInvalidOption) || errors.Is(err, callbackCause) != withCause || !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Request-Id") != "first503" || string(native.Body) != `{"error":"temporary"}` {
				t.Fatal(calls, callbacks, err)
			}
		})
	}
}
