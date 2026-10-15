package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestCloudRouterInterfaceRemoveBothFalseyFirstAndHTTPZero(t *testing.T) {
	falsey := []json.RawMessage{json.RawMessage("null"), json.RawMessage("false"), json.RawMessage("0"), json.RawMessage(`""`), json.RawMessage("[]"), json.RawMessage("{}"), json.RawMessage("1e-400")}
	var missing *network.Service
	for _, subnet := range falsey {
		for _, port := range falsey {
			result, err := missing.RemoveRouterInterface(context.Background(), "", network.WithRouterInterfaceRemoveSubnetID(subnet), network.WithRouterInterfaceRemovePortID(port))
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "at least one") {
				t.Fatal(subnet, port, result, err)
			}
		}
	}
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected request", r.URL)
		testcloud.JSON(w, 500, `{}`)
	})
	result, err := service.RemoveRouterInterface(context.Background(), "unresolved-name")
	if result != nil || !strings.Contains(err.Error(), "at least one") || count.Load() != 0 {
		t.Fatal(result, err, count.Load())
	}
	result, err = missing.RemoveRouterInterface(context.Background(), "router", network.WithRouterInterfaceRemovePortID("p"))
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || strings.Contains(err.Error(), "at least one") {
		t.Fatal(result, err)
	}
}
func TestCloudRouterInterfaceRemovePortFirstRawAndCachedResourceReferences(t *testing.T) {
	cachedNull := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage("null")}}}
	cachedFalse := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage("false")}}}
	for _, tc := range []struct {
		name         string
		subnet, port any
		body         string
	}{
		{"subnet", "s", nil, `{"subnet_id":"s"}`}, {"port wins", "s", "p", `{"port_id":"p"}`}, {"falsey port", "s", false, `{"subnet_id":"s"}`}, {"underflow port", "s", json.RawMessage("1e-400"), `{"subnet_id":"s"}`}, {"port scalar", "s", 42, `{"port_id":42}`}, {"port dict", "s", json.RawMessage(`{"id":null}`), `{"port_id":{"id":null}}`}, {"subnet list", json.RawMessage(`["a","b"]`), nil, `{"subnet_id":["a","b"]}`}, {"resource port null", "s", cachedNull, `{"port_id":null}`}, {"resource subnet false", cachedFalse, nil, `{"subnet_id":false}`}, {"owned port record", "s", &ports.PortRecord{Resource: cachedNull}, `{"port_id":null}`}, {"owned subnet record", &subnets.SubnetRecord{Resource: cachedFalse}, nil, `{"subnet_id":false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"PUT /v2.0/routers/router-name/remove_router_interface": {{200, `{"ack":"actual"}`}}})
			result, err := network.New(client).RemoveRouterInterface(context.Background(), "router-name", network.WithRouterInterfaceRemoveSubnetID(tc.subnet), network.WithRouterInterfaceRemovePortID(tc.port))
			if err != nil || result == nil || string(result.Value) != "null" || len(calls()) != 1 || strings.TrimSpace(calls()[0].body) != tc.body || calls()[0].query.Encode() != "" {
				t.Fatal(result, err, calls())
			}
			if calls()[0].method != http.MethodPut {
				t.Fatal(calls())
			}
		})
	}
}
func TestCloudRouterInterfaceRemoveSourceNoneAfterArbitraryJSONAndActualCopies(t *testing.T) {
	for _, body := range []string{`null`, `false`, `0`, `"ack"`, `[]`, `{"id":"server","extension":{"n":9007199254740993}}`} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 203, body) })
		result, err := service.RemoveRouterInterface(context.Background(), "r", network.WithRouterInterfaceRemoveSubnetID("s"))
		if err != nil || result == nil || string(result.Value) != "null" || string(result.ResponseJSON) != body || string(result.Envelope) != body || result.StatusCode != 203 || result.Header.Get("X-Request-Id") != "qos-mutation-actual" || count.Load() != 1 {
			t.Fatal(body, result, err)
		}
		if strings.HasPrefix(body, "{") != (result.Wire != nil) {
			t.Fatal(body, result)
		}
		result.ResponseJSON[0] = 'X'
		result.Header.Set("X-Request-Id", "consumer")
		if string(result.Envelope) != body {
			t.Fatal(result)
		}
		if result.Wire != nil {
			if result.Wire.Header.Get("X-Request-Id") != "qos-mutation-actual" || result.Wire.Body["tags"] != nil {
				t.Fatal(result)
			}
		}
	}
}
func TestCloudRouterInterfaceRemoveAcceptedDecodeAndCleanHTTPErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{{204, ""}, {200, "bad"}, {403, `{"error":"forbidden"}`}, {404, `{"error":"not found"}`}, {409, `{"error":"conflict"}`}} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if tc.body == "" {
				w.WriteHeader(tc.status)
			} else {
				testcloud.JSON(w, tc.status, tc.body)
			}
		})
		result, err := service.RemoveRouterInterface(context.Background(), "r", network.WithRouterInterfaceRemovePortID("p"))
		if err == nil || result == nil || result.Value != nil || result.ResponseJSON != nil || result.StatusCode != tc.status || string(result.Envelope) != tc.body || result.Header.Get("X-Request-Id") != "qos-mutation-actual" || count.Load() != 1 {
			t.Fatal(tc, result, err, count.Load())
		}
		if tc.status >= 400 {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != tc.status || native.Method != http.MethodPut || string(native.Body) != tc.body {
				t.Fatal(err, native)
			}
		} else {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != tc.status || string(proof.Body) != tc.body {
				t.Fatal(err, proof)
			}
		}
	}
}
func TestCloudRouterInterfaceRemoveOwnsOptionRefsHeadersAndEscapedState(t *testing.T) {
	var requestHeader http.Header
	var posted map[string]json.RawMessage
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		requestHeader = r.Header.Clone()
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &posted)
		testcloud.JSON(w, 200, `{"ack":true}`)
	})
	raw := json.RawMessage(`"owned-port"`)
	headers := map[string]string{"X-Trace": "owned"}
	ref := &routers.RouterInterfaceRef{Value: raw}
	option := network.WithRouterInterfaceRemoveOpts(network.RouterInterfaceRemoveOpts{PortID: ref, Headers: headers})
	raw[1] = 'X'
	headers["X-Trace"] = "changed"
	var escaped *network.RouterInterfaceRemoveOpts
	result, err := service.RemoveRouterInterface(context.Background(), "r", option, func(c *network.RouterInterfaceRemoveOpts) error { escaped = c; return nil }, func(*network.RouterInterfaceRemoveOpts) error {
		escaped.PortID.Value[1] = 'Y'
		escaped.Headers["X-Trace"] = "escaped"
		return nil
	})
	if err != nil || result == nil || count.Load() != 1 || string(posted["port_id"]) != `"owned-port"` || requestHeader.Get("X-Trace") != "owned" {
		t.Fatal(result, err, posted, requestHeader)
	}
	cached := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"cached"`)}}}
	captured := network.WithRouterInterfaceRemovePortID(cached)
	cached.Body["id"] = json.RawMessage(`"changed"`)
	result, err = service.RemoveRouterInterface(context.Background(), "r", captured)
	if err != nil || string(posted["port_id"]) != `"cached"` {
		t.Fatal(result, err, posted)
	}

}
func TestCloudRouterInterfaceRemoveCancellationNilOptionsAndStickySourceStopCallbacks(t *testing.T) {
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `null`) })
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller cancellation")
	cancel(cause)
	result, err := service.RemoveRouterInterface(ctx, "r", network.WithRouterInterfaceRemovePortID("p"))
	if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || count.Load() != 0 {
		t.Fatal(result, err)
	}
	for _, opts := range [][]network.RouterInterfaceRemoveOption{{nil}, {network.WithRouterInterfaceRemovePortID(make(chan int))}} {
		result, err := service.RemoveRouterInterface(context.Background(), "r", opts...)
		if result != nil || err == nil || count.Load() != 0 {
			t.Fatal(result, err)
		}
	}
	nextCalls := 0
	result, err = service.RemoveRouterInterface(context.Background(), "r", network.WithRouterInterfaceRemovePortID("p"), func(*network.RouterInterfaceRemoveOpts) error {
		service.RawClient().ResourceBase += "drift/"
		return nil
	}, func(*network.RouterInterfaceRemoveOpts) error { nextCalls++; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || nextCalls != 0 || count.Load() != 0 {
		t.Fatal(result, err, nextCalls, count.Load())
	}
}

// Only request selection differs; reuse the QoS real body observer and the
// public Gophercloud testhelper-backed testcloud rather than another server.
type routerInterfaceRemoveObservedTransport struct {
	base              http.RoundTripper
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
}

func (o routerInterfaceRemoveObservedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := o.base.RoundTrip(r)
	if response != nil && r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/remove_router_interface") {
		response.Body = &qosPolicyActionObservedBody{ReadCloser: response.Body, readErr: o.readErr, closeErr: o.closeErr, closed: o.closed, onClose: o.onClose}
	}
	return response, err
}
func TestCloudRouterInterfaceRemovePhysicalFaultsKeepActualProofAndPreventReplay(t *testing.T) {
	for _, status := range []int{200, 401, 503} {
		for _, lane := range []string{"read-close", "cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", status, lane), func(t *testing.T) {
				body := `{"ack":"actual"}`
				if status >= 400 {
					body = `{"error":"actual"}`
				}
				service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, status, body) })
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("actual read"), errors.New("actual close"), errors.New("actual cancellation")
				closed := &atomic.Int32{}
				base := service.RawClient().HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				observer := routerInterfaceRemoveObservedTransport{base: base, closed: closed}
				switch lane {
				case "read-close":
					observer.readErr, observer.closeErr = readCause, closeCause
				case "cancel":
					observer.onClose = func() { cancel(cancelCause) }
				case "source":
					observer.onClose = func() { service.RawClient().ResourceBase += "drift/" }
				}
				service.RawClient().HTTPClient.Transport = observer
				retries, backoff, reauth := 0, 0, 0
				service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return errors.New("unexpected replay")
				}
				service.RawClient().ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoff++
					return errors.New("unexpected backoff")
				}
				service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauth++; return errors.New("unexpected auth") }
				result, err := service.RemoveRouterInterface(ctx, "r", network.WithRouterInterfaceRemovePortID("p"))
				if err == nil || result == nil || result.Value != nil || result.ResponseJSON != nil || result.StatusCode != status || string(result.Envelope) != body || result.Header.Get("X-Request-Id") != "qos-mutation-actual" || count.Load() != 1 || closed.Load() != 1 || retries != 0 || backoff != 0 || reauth != 0 {
					t.Fatal(result, err, count.Load(), closed.Load(), retries, backoff, reauth)
				}
				switch lane {
				case "read-close":
					if !errors.Is(err, readCause) || !errors.Is(err, closeCause) {
						t.Fatal(err)
					}
				case "cancel":
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
						t.Fatal(err)
					}
				case "source":
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body {
						t.Fatal(err, proof)
					}
				} else {
					original := err
					if status == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || auth == nil || auth.ErrReauth == nil {
							t.Fatal(err, auth)
						}
						original = auth.ErrOriginal
						if lane == "read-close" && (!errors.Is(auth.ErrReauth, readCause) || !errors.Is(auth.ErrReauth, closeCause)) {
							t.Fatal(err, auth)
						}
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(original, &native) || native.Actual != status || native.Method != http.MethodPut || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "qos-mutation-actual" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}
func TestCloudRouterInterfaceRemoveCleanRetryAndTokenRenewalKeepCapturedBody(t *testing.T) {
	for _, status := range []int{401, 503} {
		var bodies, headers []string
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			data, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(data))
			headers = append(headers, r.Header.Get("X-Trace"))
			if (status == 401 && r.Header.Get("X-Auth-Token") != "renewed") || (status == 503 && r.Header.Get("X-Attempt") == "") {
				w.Header().Set("X-Request-Id", "first")
				testcloud.JSON(w, status, `{"error":"first"}`)
				return
			}
			testcloud.JSON(w, 200, `null`)
		})
		attempts := 0
		client := service.RawClient()
		client.MoreHeaders = map[string]string{"X-Trace": "captured"}
		if status == 401 {
			client.ProviderClient.ReauthFunc = func(context.Context) error { attempts++; client.ProviderClient.SetToken("renewed"); return nil }
		} else {
			client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				attempts++
				if opts.MoreHeaders == nil {
					opts.MoreHeaders = map[string]string{}
				}
				opts.MoreHeaders["X-Attempt"] = "second"
				return nil
			}
		}
		result, err := service.RemoveRouterInterface(context.Background(), "r", network.WithRouterInterfaceRemovePortID("p"))
		if err != nil || result == nil || string(result.Value) != "null" || count.Load() != 2 || attempts != 1 || !reflect.DeepEqual(bodies, []string{`{"port_id":"p"}`, `{"port_id":"p"}`}) || !reflect.DeepEqual(headers, []string{"captured", "captured"}) {
			t.Fatal(result, err, count.Load(), attempts, bodies, headers)
		}
	}
}
