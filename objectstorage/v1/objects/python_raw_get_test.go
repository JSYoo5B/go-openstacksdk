package objects_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonObjectRawRead(t *testing.T, result *objects.RawObjectResponse, err error) string {
	t.Helper()
	if err != nil || result == nil || result.Body == nil {
		t.Fatal(result, err)
	}
	defer result.Body.Close()
	data, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// get_object_raw(container, obj, query_string) returns the requests.Response
// of the object GET for every status because the proxy uses raise_exc=False.
func TestPythonObjectCloudGetObjectRawReturnsEveryStatus(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
		w.Header().Set("X-Trans-Id", "tx-"+call.query)
		switch call.path {
		case "/swift/v1/AUTH_p/box/missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("<html>Not Found</html>"))
		case "/swift/v1/AUTH_p/box/broken":
			w.WriteHeader(http.StatusInternalServerError)
		case "/swift/v1/AUTH_p/box/secret":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("denied"))
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("payload"))
		}
	})
	// get_object_raw("box", "dir/file.bin", objects.WithRawObjectQueryString("multipart-manifest=get&format=raw"))
	got, err := api.GetObjectRaw(ctx, "box", "dir/file.bin", objects.WithRawObjectQueryString("multipart-manifest=get&format=raw"))
	if body := pythonObjectRawRead(t, got, err); body != "payload" || got.StatusCode != 200 || got.Header.Get("X-Trans-Id") != "tx-multipart-manifest=get&format=raw" {
		t.Fatal(got, body)
	}
	// get_object_raw("box", "dir/file.bin") sends no query.
	got, err = api.GetObjectRaw(ctx, "box", "dir/file.bin")
	if body := pythonObjectRawRead(t, got, err); body != "payload" {
		t.Fatal(body)
	}
	got, err = api.GetObjectRaw(ctx, "box", "missing", objects.WithRawObjectQueryString("symlink=get"))
	if body := pythonObjectRawRead(t, got, err); body != "<html>Not Found</html>" || got.StatusCode != 404 {
		t.Fatal(got, body)
	}
	got, err = api.GetObjectRaw(ctx, "box", "broken")
	if body := pythonObjectRawRead(t, got, err); body != "" || got.StatusCode != 500 {
		t.Fatal(got, body)
	}
	// Without a reauthentication function the 401 response is returned raw.
	got, err = api.GetObjectRaw(ctx, "box", "secret")
	if body := pythonObjectRawRead(t, got, err); body != "denied" || got.StatusCode != 401 || got.Header.Get("X-Trans-Id") != "tx-" {
		t.Fatal(got, body)
	}
	want := []struct{ path, query string }{
		{"/swift/v1/AUTH_p/box/dir%2Ffile.bin", "multipart-manifest=get&format=raw"},
		{"/swift/v1/AUTH_p/box/dir%2Ffile.bin", ""},
		{"/swift/v1/AUTH_p/box/missing", "symlink=get"},
		{"/swift/v1/AUTH_p/box/broken", ""},
		{"/swift/v1/AUTH_p/box/secret", ""},
	}
	recorded := calls()
	if len(recorded) != len(want) {
		t.Fatal(recorded)
	}
	for index, call := range recorded {
		if call.method != http.MethodGet || call.path != want[index].path || call.query != want[index].query || call.body != "" {
			t.Fatal(index, call)
		}
	}
}

func TestPythonObjectCloudGetObjectRawRejectsInputsBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) { w.WriteHeader(http.StatusOK) })
	for _, input := range []struct{ container, object, query string }{
		{"box", "file", "a=b c"},
		{"box", "file", "a=b#frag"},
		{"box", "file", "a=\n"},
		{"box", "file", "a=\xff"},

		{"", "file", ""},
	} {
		var operation *resource.OperationError
		if got, err := api.GetObjectRaw(ctx, input.container, input.object, objects.WithRawObjectQueryString(input.query)); got != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "GetObjectRaw" {
			t.Fatal(input, got, err)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := api.GetObjectRaw(canceled, "box", "file"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var nilAPI *objects.API
	if _, err := nilAPI.GetObjectRaw(ctx, "box", "file"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(calls()) != 0 {
		t.Fatal(calls())
	}
}

func TestPythonObjectRawBufferedDefaultOwnsOpaqueBytesAndAllowsContainerEndpoint(t *testing.T) {
	for _, status := range []int{200, 206, 304, 400, 401, 404, 429, 498, 503, 777} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			payload := string([]byte{0, 0xff, 'x'})
			body := &objectMetadataBody{Reader: strings.NewReader(payload)}
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || r.URL.String() != objectMetadataBase+"box?prefix=a%2Fb&same=1&same=2" || r.Header.Get("Accept") != "*/*" || r.Body != nil && r.Body != http.NoBody {
					t.Errorf("wrong raw request %s %v", r.URL, r.Header)
				}
				return objectMetadataWire(status, body), nil
			})
			result, err := objects.New(client).GetObjectRaw(context.Background(), "box", "", objects.WithRawObjectQueryString("prefix=a%2Fb&same=1&same=2"))
			if err != nil || result == nil || result.StatusCode != status || result.Header.Get("X-Trans-Id") != "actual-object" || calls != 1 || body.closes.Load() != 1 {
				t.Fatal(result, err, calls, body.closes.Load())
			}
			data, readErr := io.ReadAll(result.Body)
			closeErr := result.Body.Close()
			if readErr != nil || closeErr != nil || string(data) != payload || body.closes.Load() != 1 {
				t.Fatal(readErr, closeErr, data, body.closes.Load())
			}
		})
	}
}

func TestPythonObjectRawPhysicalFaultsPreserveActualPayloadAndNative401WithoutHooks(t *testing.T) {
	for _, status := range []int{200, 206, 401, 404, 429, 498, 503} {
		for _, mode := range []string{"read", "close", "both", "restored source and cancellation"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("raw physical read"), errors.New("raw physical close"), errors.New("raw custom cancel")
				payload := string([]byte{0, 0xff, 'x'})
				var client *gophercloud.ServiceClient
				requests, reads, retries, reauths, backoffs := 0, 0, 0, 0, 0
				saved := objectMetadataBase
				body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) {
					reads++
					if reads > 1 {
						return 0, io.EOF
					}
					n := copy(dst, payload)
					if mode == "restored source and cancellation" {
						client.Endpoint = saved + "changed/"
					}
					if mode == "read" || mode == "both" {
						return n, readCause
					}
					return n, io.EOF
				})}
				if mode == "close" || mode == "both" {
					body.closeErr = closeCause
				}
				if mode == "restored source and cancellation" {
					body.onClose = func() { client.Endpoint = saved; cancel(cancelCause) }
				}
				client = objectMetadataClient(func(*http.Request) (*http.Response, error) { requests++; return objectMetadataWire(status, body), nil })
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				client.ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
				client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoffs++
					return nil
				}
				result, err := objects.New(client).GetObjectRaw(ctx, "box", "file", objects.WithRawObjectQueryString("opaque=%2F&opaque=%252F"))
				if err == nil || result == nil || result.StatusCode != status || result.Header.Get("X-Trans-Id") != "actual-object" || requests != 1 || body.closes.Load() != 1 || retries != 0 || reauths != 0 || backoffs != 0 {
					t.Fatal(result, err, requests, body.closes.Load(), retries, reauths, backoffs)
				}
				data, _ := io.ReadAll(result.Body)
				_ = result.Body.Close()
				if string(data) != payload {
					t.Fatal(data)
				}
				if (mode == "read" || mode == "both") && !errors.Is(err, readCause) {
					t.Fatal(err)
				}
				if (mode == "close" || mode == "both") && !errors.Is(err, closeCause) {
					t.Fatal(err)
				}
				if mode == "restored source and cancellation" && (!errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) {
					t.Fatal(err)
				}
				if status == 401 {
					var native *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &native) || native.ErrOriginal == nil || native.ErrReauth == nil {
						t.Fatal(err, native)
					}
					var receipt gophercloud.ErrUnexpectedResponseCode
					if !errors.As(native.ErrOriginal, &receipt) || receipt.Actual != 401 || string(receipt.Body) != payload || receipt.URL != objectMetadataBase+"box/file?opaque=%2F&opaque=%252F" {
						t.Fatal(err, receipt)
					}
					if (mode == "read" || mode == "both") && !errors.Is(native.ErrReauth, readCause) {
						t.Fatal(native.ErrReauth)
					}
					if (mode == "close" || mode == "both") && !errors.Is(native.ErrReauth, closeCause) {
						t.Fatal(native.ErrReauth)
					}
				} else if status == 429 || status == 498 {
					var receipt gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &receipt) || receipt.Actual != status || string(receipt.Body) != payload || receipt.ResponseHeader.Get("X-Trans-Id") != "actual-object" {
						t.Fatal(err, receipt)
					}
				} else {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != payload || proof.Header.Get("X-Trans-Id") != "actual-object" {
						t.Fatal(err, proof)
					}
				}
			})
		}
	}
}

func TestPythonObjectRawCleanNativeRenewalAndMixedCallbackErrorsStayDistinct(t *testing.T) {
	for _, mode := range []string{"reauth", "backoff", "callback"} {
		t.Run(mode, func(t *testing.T) {
			requests, hooks := 0, 0
			callbackCause := errors.New("callback failed")
			var client *gophercloud.ServiceClient
			client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.String() != objectMetadataBase+"box/file?a=1&a=2" || r.Header.Get("Accept") != "application/custom" || r.Header.Get("OpenStack-API-Version") != "object-store 1.2" || r.Header.Get("If-None-Match") != "selected" {
					t.Errorf("raw retry changed %s %v", r.URL, r.Header)
				}
				if requests == 1 {
					code := 401
					if mode == "backoff" {
						code = 429
					}
					return objectMetadataResponse(code, "clean initial"), nil
				}
				if r.Header.Get("X-Auth-Token") != "renewed" {
					t.Error(r.Header)
				}
				return objectMetadataResponse(503, "raw final"), nil
			})
			client.Microversion = "1.2"
			client.MoreHeaders = map[string]string{"Accept": "application/custom", "If-None-Match": "selected"}
			switch mode {
			case "reauth":
				client.ProviderClient.ReauthFunc = func(context.Context) error { hooks++; client.ProviderClient.SetToken("renewed"); return nil }
			case "backoff":
				client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					hooks++
					client.ProviderClient.SetToken("renewed")
					return nil
				}
			case "callback":
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return callbackCause
				}
			}
			result, err := objects.New(client).GetObjectRaw(context.Background(), "box", "file", objects.WithRawObjectQueryString("a=1&a=2"))
			if mode == "callback" {
				if !errors.Is(err, callbackCause) || result == nil || result.StatusCode != 401 || requests != 1 || hooks != 1 {
					t.Fatal(result, err, requests, hooks)
				}
			} else {
				if err != nil || result == nil || result.StatusCode != 503 || requests != 2 || hooks != 1 {
					t.Fatal(result, err, requests, hooks)
				}
				data, _ := io.ReadAll(result.Body)
				_ = result.Body.Close()
				if string(data) != "raw final" {
					t.Fatal(data)
				}
			}
		})
	}
}

func TestPythonObjectRawOptionBoundaryCapturesSourceBeforeCallbacks(t *testing.T) {
	api, calls := pythonObjectAPI(t, func(_ pythonObjectCall, w http.ResponseWriter) { w.WriteHeader(200) })
	result, err := api.GetObjectRaw(context.Background(), "box", "file", nil)
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 0 {
		t.Fatal(result, err, calls())
	}
	source := api.RawClient()
	saved := source.ResourceBase
	result, err = api.GetObjectRaw(context.Background(), "box", "file", func(*objects.RawObjectOpts) error { source.ResourceBase = source.Endpoint + "changed/"; return nil })
	source.ResourceBase = saved
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 0 {
		t.Fatal(result, err, calls())
	}
}

func TestPythonObjectRawStreamingIsUnreadAndGuardsOpaqueLifecycle(t *testing.T) {
	for _, status := range []int{200, 206, 304, 400, 404, 503, 777} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			payload := string([]byte{0, 0xff, 'x'})
			reads, requests := 0, 0
			reader := strings.NewReader(payload)
			body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) { reads++; return reader.Read(dst) })}
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.String() != objectMetadataBase+"box/file?q=%2F&q=%252F" || r.Header.Get("Accept") != "*/*" {
					t.Errorf("wrong streamed request %s %v", r.URL, r.Header)
				}
				return objectMetadataWire(status, body), nil
			})
			result, err := objects.New(client).GetObjectRaw(context.Background(), "box", "file", objects.WithRawObjectQueryString("q=%2F&q=%252F"), objects.WithRawObjectStream(true))
			if err != nil || result == nil || result.StatusCode != status || reads != 0 || requests != 1 || body.closes.Load() != 0 {
				t.Fatal(result, err, reads, requests, body.closes.Load())
			}
			data, readErr := io.ReadAll(result.Body)
			firstClose, secondClose := result.Body.Close(), result.Body.Close()
			if readErr != nil || firstClose != nil || secondClose != nil || string(data) != payload || requests != 1 || body.closes.Load() != 1 {
				t.Fatal(readErr, firstClose, secondClose, data, requests, body.closes.Load())
			}
			if n, end := result.Body.Read(make([]byte, 1)); n != 0 || end != io.EOF {
				t.Fatal(n, end)
			}
		})
	}
}

func TestPythonObjectRawStreamingFaultsRetainImmutableWireEvidenceAndNoReplay(t *testing.T) {
	for _, mode := range []string{"read", "close", "both", "restored source and cancellation"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("stream physical read"), errors.New("stream physical close"), errors.New("stream custom cancel")
			payload := string([]byte{0, 0xff, 'x'})
			var client *gophercloud.ServiceClient
			requests, reads, retries, reauths, backoffs := 0, 0, 0, 0, 0
			saved := objectMetadataBase
			body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) {
				reads++
				if reads > 1 {
					return 0, io.EOF
				}
				n := copy(dst, payload)
				if mode == "restored source and cancellation" {
					client.Endpoint = saved + "changed/"
				}
				if mode == "read" || mode == "both" {
					return n, readCause
				}
				return n, io.EOF
			})}
			if mode == "close" || mode == "both" {
				body.closeErr = closeCause
			}
			if mode == "restored source and cancellation" {
				body.onClose = func() { client.Endpoint = saved; cancel(cancelCause) }
			}
			client = objectMetadataClient(func(*http.Request) (*http.Response, error) { requests++; return objectMetadataWire(203, body), nil })
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			client.ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
			client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoffs++
				return nil
			}
			result, err := objects.New(client).GetObjectRaw(ctx, "box", "file", objects.WithRawObjectStream(true))
			if err != nil || result == nil || reads != 0 || body.closes.Load() != 0 {
				t.Fatal(result, err, reads, body.closes.Load())
			}
			result.StatusCode = 299
			result.Header.Set("X-Trans-Id", "forged")
			data, readErr := io.ReadAll(result.Body)
			closeErr := result.Body.Close()
			combined := errors.Join(readErr, closeErr)
			if combined == nil || string(data) != payload || requests != 1 || body.closes.Load() != 1 || retries != 0 || reauths != 0 || backoffs != 0 {
				t.Fatal(combined, data, requests, body.closes.Load(), retries, reauths, backoffs)
			}
			if (mode == "read" || mode == "both") && !errors.Is(combined, readCause) {
				t.Fatal(combined)
			}
			if (mode == "close" || mode == "both") && !errors.Is(combined, closeCause) {
				t.Fatal(combined)
			}
			if mode == "restored source and cancellation" && (!errors.Is(combined, resource.ErrInvalidOption) || !errors.Is(combined, cancelCause) || !errors.Is(combined, context.Canceled)) {
				t.Fatal(combined)
			}
			var operation *resource.OperationError
			var proof *resource.ResponseError
			if !errors.As(readErr, &operation) || operation.Operation != "GetObjectRaw" || !errors.As(readErr, &proof) || proof.StatusCode != 203 || string(proof.Body) != payload || proof.Header.Get("X-Trans-Id") != "actual-object" {
				t.Fatal(readErr, operation, proof)
			}
		})
	}
}

func TestPythonObjectRawStreamingCloseBeforeEOFAndCanceledUnreadBody(t *testing.T) {
	t.Run("early close", func(t *testing.T) {
		body := &objectMetadataBody{Reader: strings.NewReader("opaque bytes")}
		requests := 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { requests++; return objectMetadataWire(200, body), nil })
		result, err := objects.New(client).GetObjectRaw(context.Background(), "box", "file", objects.WithRawObjectStream(true))
		if err != nil || result == nil {
			t.Fatal(result, err)
		}
		dst := make([]byte, 1)
		if n, err := result.Body.Read(dst); n != 1 || err != nil {
			t.Fatal(n, err)
		}
		if err := result.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if err := result.Body.Close(); err != nil || body.closes.Load() != 1 || requests != 1 {
			t.Fatal(err, body.closes.Load(), requests)
		}
		if n, err := result.Body.Read(dst); n != 0 || !errors.Is(err, io.ErrClosedPipe) {
			t.Fatal(n, err)
		}
	})
	t.Run("caller canceled before first read", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause := errors.New("cancel after headers")
		reads, requests := 0, 0
		body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) { reads++; return copy(dst, "unexpected read"), io.EOF })}
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { requests++; return objectMetadataWire(200, body), nil })
		result, err := objects.New(client).GetObjectRaw(ctx, "box", "file", objects.WithRawObjectStream(true))
		if err != nil || result == nil {
			t.Fatal(result, err)
		}
		cancel(cause)
		_, readErr := result.Body.Read(make([]byte, 30))
		closeErr := result.Body.Close()
		if !errors.Is(errors.Join(readErr, closeErr), cause) || !errors.Is(readErr, context.Canceled) || reads != 0 || requests != 1 || body.closes.Load() != 1 {
			t.Fatal(readErr, closeErr, reads, requests, body.closes.Load())
		}
		var proof *resource.ResponseError
		if !errors.As(readErr, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Trans-Id") != "actual-object" {
			t.Fatal(readErr, proof)
		}
	})
}

func TestPythonObjectRawStreamingCleanAndFaultyNative401AndBackoff(t *testing.T) {
	for _, status := range []int{401, 429, 498} {
		for _, faulty := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%t", status, faulty), func(t *testing.T) {
				requests, renewals := 0, 0
				cause := errors.New("native rejected close failure")
				first := &objectMetadataBody{Reader: strings.NewReader("native status bytes")}
				if faulty {
					first.closeErr = cause
				}
				var client *gophercloud.ServiceClient
				client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					requests++
					if requests == 1 {
						return objectMetadataWire(status, first), nil
					}
					if r.Header.Get("X-Auth-Token") != "live" {
						t.Error(r.Header)
					}
					return objectMetadataResponse(404, "raw streamed final"), nil
				})
				if status == 401 {
					client.ProviderClient.ReauthFunc = func(context.Context) error { renewals++; client.ProviderClient.SetToken("live"); return nil }
				} else {
					client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
						renewals++
						client.ProviderClient.SetToken("live")
						return nil
					}
				}
				result, err := objects.New(client).GetObjectRaw(context.Background(), "box", "file", objects.WithRawObjectStream(true))
				if faulty {
					if result == nil || result.StatusCode != status || !errors.Is(err, cause) || requests != 1 || renewals != 0 || first.closes.Load() != 1 {
						t.Fatal(result, err, requests, renewals, first.closes.Load())
					}
					data, _ := io.ReadAll(result.Body)
					_ = result.Body.Close()
					if string(data) != "native status bytes" {
						t.Fatal(data)
					}
					if status == 401 {
						var native *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &native) || !errors.Is(native.ErrReauth, cause) {
							t.Fatal(err, native)
						}
					}
				} else {
					if err != nil || result == nil || result.StatusCode != 404 || requests != 2 || renewals != 1 || first.closes.Load() != 1 {
						t.Fatal(result, err, requests, renewals, first.closes.Load())
					}
					data, readErr := io.ReadAll(result.Body)
					closeErr := result.Body.Close()
					if readErr != nil || closeErr != nil || string(data) != "raw streamed final" {
						t.Fatal(readErr, closeErr, data)
					}
				}
			})
		}
	}
}
