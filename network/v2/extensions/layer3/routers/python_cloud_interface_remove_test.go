package routers_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
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

func TestPythonCloudRouterInterfaceRemoveExplicitCharsetsKeepRawProofAndLegacy(t *testing.T) {
	for _, tc := range []struct{ name, contentType, wireHex string }{
		{"latin1", "application/json; charset=ISO-8859-1", "7b2261636b223a22636166e9227d"},
		{"utf16 BOM", "application/json; charset=UTF-16", "fffe7b002200610063006b0022003a002200630061006600e90022007d00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := hex.DecodeString(tc.wireHex)
			if err != nil {
				t.Fatal(err)
			}
			var calls []nativeRouterCall
			var originalHeader http.Header
			api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
				reply := nativeRouterWire(203, string(wire))
				reply.Header.Set("Content-Type", tc.contentType)
				reply.Header.Set("X-Request-Id", "charset-wire")
				originalHeader = reply.Header
				return reply
			})
			value, err := api.RemoveCloudInterfaceRecord(context.Background(), "r", routers.WithRouterInterfacePort("p"))
			if err != nil || value == nil || string(value.Value) != `{"ack":"café"}` || value.Wire == nil || string(value.Wire.Body["ack"]) != `"café"` || value.StatusCode != 203 || !bytes.Equal(value.Envelope, wire) || value.Header.Get("Content-Type") != tc.contentType || value.Wire.Header.Get("X-Request-Id") != "charset-wire" || value.Wire.StatusCode != 203 || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].body != `{"port_id":"p"}` {
				t.Fatal(value, err, calls)
			}
			value.Value[2] = 'X'
			value.Wire.Body["ack"][1] = 'X'
			value.Header.Set("Content-Type", "changed")
			if !bytes.Equal(value.Envelope, wire) || originalHeader.Get("Content-Type") != tc.contentType || value.Wire.Header.Get("Content-Type") != tc.contentType {
				t.Fatal("normalized views changed wire proof", value, originalHeader)
			}
			legacy, legacyErr := api.RemoveInterfaceRecord(context.Background(), "r", routers.WithRouterInterfacePort("p"))
			if legacyErr == nil || legacy == nil || legacy.Value != nil || !bytes.Equal(legacy.Envelope, wire) || len(calls) != 2 {
				t.Fatal("legacy charset policy changed", legacy, legacyErr, calls)
			}
		})
	}
}

func TestPythonCloudRouterInterfaceRemoveCharsetFailuresKeepOriginalResponse(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body string
		unsupported             bool
	}{
		{"unknown codec", "application/json; charset=x-not-a-codec", `{"ack":true}`, true},
		{"malformed charset", "application/json; charset=\"unterminated", `{"ack":true}`, true},
		{"bare UTF16 no BOM", "application/json; charset=UTF-16", string([]byte{'{', 0, '}', 0}), true},
		{"normalized invalid JSON", "application/json; charset=UTF-16LE", string([]byte{'{', 0, 'b', 0, 'a', 0, 'd', 0}), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []nativeRouterCall
			api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
				reply := nativeRouterWire(203, tc.body)
				reply.Header.Set("Content-Type", tc.contentType)
				reply.Header.Set("X-Request-Id", "charset-failed")
				return reply
			})
			value, err := api.RemoveCloudInterfaceRecord(context.Background(), "r", routers.WithRouterInterfaceSubnet("s"))
			var proof *resource.ResponseError
			if err == nil || value == nil || value.Value != nil || value.StatusCode != 203 || string(value.Envelope) != tc.body || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != tc.body || proof.Header.Get("Content-Type") != tc.contentType || proof.Header.Get("X-Request-Id") != "charset-failed" || len(calls) != 1 {
				t.Fatal(value, err, proof, calls)
			}
			if tc.unsupported {
				if !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(err)
				}
			} else {
				var syntax *json.SyntaxError
				if !errors.As(err, &syntax) {
					t.Fatal(err)
				}
			}
		})
	}
}
