package routers_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCloudRouterInterfaceRemoveClosedSourceStatusAndLegacyIsolation(t *testing.T) {
	for _, status := range []int{199, 203, 399} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
			reply := nativeRouterWire(status, `{"ack":true}`)
			reply.Header.Set("X-Request-Id", "cloud-remove-actual")
			return reply
		})
		result, err := api.RemoveCloudInterfaceRecord(context.Background(), "literal/id?", routers.WithRouterInterfacePort("p"))
		if err != nil || result == nil || result.StatusCode != status || string(result.Value) != `{"ack":true}` || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].body != `{"port_id":"p"}` {
			t.Fatal(result, err, calls)
		}
	}
	var calls []nativeRouterCall
	api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(199, `null`) })
	result, err := api.RemoveInterfaceRecord(context.Background(), "r", routers.WithRouterInterfaceSubnet("s"))
	var native gophercloud.ErrUnexpectedResponseCode
	if result != nil || !errors.As(err, &native) || native.Actual != 199 || len(calls) != 1 {
		t.Fatal(result, err, native, calls)
	}
}
func TestPythonCloudRouterInterfaceRemovePhysicalOptInNeverCallsHooks(t *testing.T) {
	for _, status := range []int{200, 401, 429} {
		var calls []nativeRouterCall
		readCause, closeCause := errors.New("read actual"), errors.New("close actual")
		body := `{"actual":true}`
		observed := &pythonRouterActionObservedBody{body: []byte(body), readErr: readCause, closeErr: closeCause}
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
			reply := nativeRouterWire(status, "")
			reply.Body = observed
			reply.Header.Set("X-Request-Id", "cloud-remove-fault")
			return reply
		})
		retries, backoff, auth := 0, 0, 0
		provider := api.RawClient().ProviderClient
		provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
			backoff++
			return nil
		}
		provider.ReauthFunc = func(context.Context) error { auth++; return nil }
		result, err := api.RemoveCloudInterfaceRecord(context.Background(), "r", routers.WithRouterInterfaceSubnet("s"))
		if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || len(calls) != 1 || observed.closed != 1 || retries != 0 || backoff != 0 || auth != 0 {
			t.Fatal(result, err, calls, observed.closed, retries, backoff, auth)
		}
		if status < 400 {
			var proof *resource.ResponseError
			if result == nil || result.Value != nil || !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "cloud-remove-fault" {
				t.Fatal(result, err, proof)
			}
		} else {
			original := err
			if status == 401 {
				var carrier *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &carrier) || carrier == nil || !errors.Is(carrier.ErrReauth, readCause) || !errors.Is(carrier.ErrReauth, closeCause) {
					t.Fatal(err, carrier)
				}
				original = carrier.ErrOriginal
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if result != nil || !errors.As(original, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "cloud-remove-fault" {
				t.Fatal(result, err, native)
			}
		}
	}
}
