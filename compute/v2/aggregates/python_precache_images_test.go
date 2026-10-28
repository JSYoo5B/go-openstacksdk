package aggregates_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonAggregatePrecacheImagesFixedVersionLiteralOrder(t *testing.T) {
	api, calls := pythonAggregateAPI(t, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Trace") != "trace" {
			t.Error(r.Header)
		}
		return pythonAggregateWire(202, "not JSON")
	})
	api.RawClient().Microversion = "2.1"
	images := []string{"image/a", "", "image/a"}
	mutate := func(c *request.Config[aggregates.PrecacheImagesOpts]) error { images[0] = "changed"; return nil }
	if err := api.PrecacheImages(context.Background(), 7, images, mutate, aggregates.WithPrecacheImagesHeader("X-Trace", "trace")); err != nil {
		t.Fatal(err)
	}
	if calls.Len() != 1 || calls.Calls()[0].Method != "POST" || calls.Calls()[0].Path != pythonAggregateBase+"/7/images" || calls.Calls()[0].Body != `{"cache":[{"id":"image/a"},{"id":""},{"id":"image/a"}]}` || calls.Calls()[0].Header.Get("X-OpenStack-Nova-API-Version") != "2.81" {
		t.Fatal(pythonAggregateCalls(calls))
	}
	if api.RawClient().Microversion != "2.1" {
		t.Fatal("mutated selected client")
	}
}
func TestPythonAggregatePrecacheImagesEmptyStillPosts(t *testing.T) {
	api, calls := pythonAggregateAPI(t, func(*http.Request) *http.Response { return pythonAggregateWire(204, "") })
	if err := api.PrecacheImages(context.Background(), -1, nil); err != nil {
		t.Fatal(err)
	}
	if calls.Len() != 1 || calls.Calls()[0].Body != `{"cache":[]}` || calls.Calls()[0].Path != pythonAggregateBase+"/-1/images" {
		t.Fatal(pythonAggregateCalls(calls))
	}
}
func TestPythonAggregatePrecacheImagesHTTPAndAcceptedCloseFailures(t *testing.T) {
	failure := errors.New("close failed")
	for _, status := range []int{404, 202} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			api, calls := pythonAggregateAPI(t, func(*http.Request) *http.Response {
				r := pythonAggregateWire(status, "opaque")
				r.Header.Set("X-Proof", "receipt")
				r.Body = &testcloud.Body{Data: "opaque", CloseErr: failure}
				return r
			})
			err := api.PrecacheImages(context.Background(), 7, []string{"one"})
			if calls.Len() != 1 {
				t.Fatal(pythonAggregateCalls(calls))
			}
			if status == 404 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !gophercloud.ResponseCodeIs(err, 404) || !errors.Is(err, failure) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "opaque" || native.ResponseHeader.Get("X-Proof") != "receipt" {
					t.Fatal(err, native)
				}
			} else {
				var proof *resource.ResponseError
				if !errors.Is(err, failure) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != "opaque" {
					t.Fatal(err, proof)
				}
			}
		})
	}
}

func TestPythonAggregatePrecacheImagesInvalidOptionsAndSourceMutation(t *testing.T) {
	api, calls := pythonAggregateAPI(t, func(*http.Request) *http.Response { return pythonAggregateWire(202, "") })
	if err := api.PrecacheImages(context.Background(), 7, nil, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	mutate := func(*request.Config[aggregates.PrecacheImagesOpts]) error {
		api.RawClient().Endpoint = "https://foreign.invalid/"
		return nil
	}
	if err := api.PrecacheImages(context.Background(), 7, nil, mutate); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Len() != 0 {
		t.Fatal(pythonAggregateCalls(calls))
	}
	var absent *aggregates.API
	if err := absent.PrecacheImages(context.Background(), 7, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop")
	cancel(cause)
	if err := api.PrecacheImages(ctx, 7, nil); !errors.Is(err, cause) {
		t.Fatal(err)
	}
}

func TestPythonAggregatePrecacheImagesSelectedAPIReplacement(t *testing.T) {
	api, calls := pythonAggregateAPI(t, func(*http.Request) *http.Response { return pythonAggregateWire(202, "") })
	copied := *api.RawClient()
	replace := func(*request.Config[aggregates.PrecacheImagesOpts]) error {
		*api = *aggregates.New(&copied)
		return nil
	}
	if err := api.PrecacheImages(context.Background(), 7, nil, replace); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Len() != 0 {
		t.Fatal(pythonAggregateCalls(calls))
	}
}

func TestPythonAggregatePrecacheImagesRejectsRetryVersionMutation(t *testing.T) {
	api, calls := pythonAggregateAPI(t, func(*http.Request) *http.Response { return pythonAggregateWire(503, "busy") })
	hooks := 0
	api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		hooks++
		if opts.MoreHeaders == nil {
			opts.MoreHeaders = map[string]string{}
		}
		opts.MoreHeaders["OpenStack-API-Version"] = "compute 2.1"
		opts.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.1"
		return nil
	}
	if err := api.PrecacheImages(context.Background(), 7, []string{"image"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if hooks != 1 || calls.Len() != 1 || calls.Calls()[0].Header.Get("X-OpenStack-Nova-API-Version") != "2.81" {
		t.Fatal(hooks, pythonAggregateCalls(calls))
	}
}
func TestPythonAggregatePrecacheImagesReplacesCapturedVersionAliases(t *testing.T) {
	api, calls := pythonAggregateAPI(t, func(r *http.Request) *http.Response {
		if r.Header.Get("OpenStack-API-Version") != "compute 2.81" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.81" {
			t.Error(r.Header)
		}
		return pythonAggregateWire(202, "")
	})
	client := api.RawClient()
	client.Microversion = "2.1"
	client.MoreHeaders = map[string]string{"OpenStack-API-Version": "compute 2.1", "X-OpenStack-Nova-API-Version": "2.1"}
	if err := api.PrecacheImages(context.Background(), 7, nil); err != nil {
		t.Fatal(err)
	}
	if calls.Len() != 1 || client.MoreHeaders["OpenStack-API-Version"] != "compute 2.1" {
		t.Fatal(pythonAggregateCalls(calls), client.MoreHeaders)
	}
}
