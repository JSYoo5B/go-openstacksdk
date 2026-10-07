package rest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type policyTransport func(*http.Request) (*http.Response, error)

func (value policyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return value(request)
}

type policyReader func([]byte) (int, error)

func (value policyReader) Read(buffer []byte) (int, error) { return value(buffer) }

type policyBody struct {
	reader   io.Reader
	closes   int
	closeErr error
}

func (value *policyBody) Read(buffer []byte) (int, error) { return value.reader.Read(buffer) }
func (value *policyBody) Close() error                    { value.closes++; return value.closeErr }

type policyMarshaler func() ([]byte, error)

func (value policyMarshaler) MarshalJSON() ([]byte, error) { return value() }

func policyClient(transport policyTransport) *gophercloud.ServiceClient {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: transport}}
	provider.UseTokenLock()
	provider.SetToken("before")
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: "test", Endpoint: "https://example.test/", ResourceBase: "https://example.test/reverse/v1/"}
}

func policyHTTP(status int, body io.ReadCloser, header http.Header) *http.Response {
	return &http.Response{StatusCode: status, Body: body, Header: header}
}

func TestResponsePolicyContextPreflightAndSingleCauseIdentity(t *testing.T) {
	var requests, marshals int
	client := policyClient(func(*http.Request) (*http.Response, error) {
		requests++
		return policyHTTP(200, http.NoBody, http.Header{}), nil
	})
	body := policyMarshaler(func() ([]byte, error) { marshals++; return []byte(`{}`), nil })
	if result, err := DoJSON(nil, client, http.MethodPost, client.ServiceURL("objects"), body, nil, 200); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil context result=%+v err=%v", result, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := DoJSON(ctx, client, http.MethodPost, client.ServiceURL("objects"), body, nil, 200); err != context.Canceled {
		t.Fatalf("single canceled cause changed identity: %v", err)
	}
	custom := errors.New("caller cancellation")
	ctx, cancelCause := context.WithCancelCause(context.Background())
	cancelCause(custom)
	if _, err := DoJSON(ctx, client, http.MethodPost, client.ServiceURL("objects"), body, nil, 200); !errors.Is(err, context.Canceled) || !errors.Is(err, custom) {
		t.Fatalf("custom context cause lost: %v", err)
	}
	if marshals != 0 || requests != 0 {
		t.Fatalf("preflight invoked marshal=%d HTTP=%d", marshals, requests)
	}
}

func TestResponsePolicyAcceptedReadCloseContextOwnEvidence(t *testing.T) {
	readCause, closeCause, customCause := errors.New("read failed"), errors.New("close failed"), errors.New("caller canceled")
	for _, test := range []struct {
		name              string
		status            int
		readErr, closeErr error
		canceled          bool
	}{
		{name: "read only", status: 200, readErr: readCause},
		{name: "close only", status: 201, closeErr: closeCause},
		{name: "read close canceled", status: 204, readErr: readCause, closeErr: closeCause, canceled: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			header := http.Header{"X-Actual": {"before"}}
			body := &policyBody{reader: policyReader(func(buffer []byte) (int, error) {
				header.Set("X-Actual", "changed during read")
				if test.canceled {
					cancel(customCause)
				}
				err := test.readErr
				if err == nil {
					err = io.EOF
				}
				return copy(buffer, "raw partial bytes"), err
			}), closeErr: test.closeErr}
			var requests, retries int
			client := policyClient(func(*http.Request) (*http.Response, error) {
				requests++
				return policyHTTP(test.status, body, header), nil
			})
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return errors.New("accepted body must not reach retry")
			}
			result, err := DoJSON(ctx, client, http.MethodPost, client.ServiceURL("objects"), nil, nil, test.status)
			var accepted *resource.ResponseError
			if result == nil || result.StatusCode != test.status || string(result.Body) != "raw partial bytes" || result.Header.Get("X-Actual") != "before" || !errors.As(err, &accepted) || accepted.StatusCode != test.status || string(accepted.Body) != "raw partial bytes" || accepted.Header.Get("X-Actual") != "before" || requests != 1 || retries != 0 || body.closes != 1 {
				t.Fatalf("result=%+v accepted=%+v err=%v requests=%d retries=%d closes=%d", result, accepted, err, requests, retries, body.closes)
			}
			for _, cause := range []error{test.readErr, test.closeErr} {
				if cause != nil && !errors.Is(err, cause) {
					t.Fatalf("cause %v lost: %v", cause, err)
				}
			}
			if !test.canceled && ((test.readErr != nil && accepted.Cause != test.readErr) || (test.closeErr != nil && accepted.Cause != test.closeErr)) {
				t.Fatal("single accepted cause changed identity")
			}
			if test.canceled && (!errors.Is(err, context.Canceled) || !errors.Is(err, customCause)) {
				t.Fatal("context causes lost", err)
			}
			result.Body[0] = '!'
			result.Header.Set("X-Actual", "caller mutation")
			if string(accepted.Body) != "raw partial bytes" || accepted.Header.Get("X-Actual") != "before" {
				t.Fatal("accepted error aliases returned response")
			}
		})
	}
}

func TestResponsePolicyRetryRejectsBodyOwnershipAndPreservesCauses(t *testing.T) {
	encodeCause, callbackCause := errors.New("replacement encoding failed"), errors.New("retry callback failed")
	for _, test := range []struct {
		name        string
		body        any
		mutate      func(*gophercloud.RequestOpts)
		callbackErr error
		encodeErr   bool
	}{
		{name: "response closed by native", mutate: func(value *gophercloud.RequestOpts) { value.KeepResponseBody = false }},
		{name: "response decoded by native", mutate: func(value *gophercloud.RequestOpts) { value.JSONResponse = new(any) }},
		{name: "raw payload", mutate: func(value *gophercloud.RequestOpts) { value.RawBody = strings.NewReader("replacement") }},
		{name: "nil becomes null", mutate: func(value *gophercloud.RequestOpts) { value.JSONBody = json.RawMessage("null") }},
		{name: "null becomes nil", body: json.RawMessage("null"), mutate: func(value *gophercloud.RequestOpts) { value.JSONBody = nil }},
		{name: "changed encoding", body: map[string]int{"n": 1}, mutate: func(value *gophercloud.RequestOpts) { value.JSONBody = map[string]int{"n": 2} }},
		{name: "in place RawMessage mutation", body: map[string]int{"n": 1}, mutate: func(value *gophercloud.RequestOpts) { value.JSONBody.(json.RawMessage)[5] = '2' }},
		{name: "encoding callback cause", body: map[string]int{"n": 1}, encodeErr: true, mutate: func(value *gophercloud.RequestOpts) {
			value.JSONBody = policyMarshaler(func() ([]byte, error) { return nil, encodeCause })
		}},
		{name: "all callback encoding ownership causes", body: map[string]int{"n": 1}, callbackErr: callbackCause, encodeErr: true, mutate: func(value *gophercloud.RequestOpts) {
			value.KeepResponseBody = false
			value.JSONBody = policyMarshaler(func() ([]byte, error) { return nil, encodeCause })
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			rejected := &policyBody{reader: strings.NewReader("original HTTP failure")}
			var requests, callbacks int
			client := policyClient(func(*http.Request) (*http.Response, error) {
				requests++
				return policyHTTP(503, rejected, http.Header{"X-Actual": {"rejected"}}), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, value *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks++
				test.mutate(value)
				return test.callbackErr
			}
			originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
			result, err := DoJSON(context.Background(), client, http.MethodPost, client.ServiceURL("objects"), test.body, nil, 200)
			var native gophercloud.ErrUnexpectedResponseCode
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "original HTTP failure" || native.ResponseHeader.Get("X-Actual") != "rejected" || requests != 1 || callbacks != 1 || rejected.closes != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
				t.Fatalf("result=%+v native=%+v err=%v requests=%d callbacks=%d closes=%d", result, native, err, requests, callbacks, rejected.closes)
			}
			if test.encodeErr && !errors.Is(err, encodeCause) || test.callbackErr != nil && !errors.Is(err, test.callbackErr) {
				t.Fatalf("encoding/callback cause lost: %v", err)
			}
		})
	}
}

func TestResponsePolicySafeRetryUsesOwnedSerializedBytesAndLiveAuth(t *testing.T) {
	var calls, callbacks, replacementMarshals int
	var client *gophercloud.ServiceClient
	bodies := []*policyBody{{reader: strings.NewReader("retry")}, {reader: strings.NewReader(`{"ok":true}`)}}
	client = policyClient(func(r *http.Request) (*http.Response, error) {
		calls++
		data, err := io.ReadAll(r.Body)
		if err != nil || string(data) != `{"n":1}` || r.Method != http.MethodPost || r.URL.String() != "https://example.test/reverse/v1/objects" {
			t.Fatalf("changed request: %s %s data=%s headers=%v err=%v", r.Method, r.URL, data, r.Header, err)
		}
		status := 503
		if calls == 2 {
			status = 200
			if r.Header.Get("X-Auth-Token") != "after" || r.Header.Get("X-Caller-Retry") != "allowed" || r.Header.Get("X-Source") != "" {
				t.Fatal("live auth or existing header policy lost", r.Header)
			}
		} else if r.Header.Get("X-Source") != "before" {
			t.Fatal("initial source header snapshot lost", r.Header)
		}
		return policyHTTP(status, bodies[calls-1], http.Header{}), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "before"}
	input := map[string]int{"n": 1}
	client.RetryFunc = func(_ context.Context, _, _ string, value *gophercloud.RequestOpts, _ error, _ uint) error {
		callbacks++
		input["n"] = 9
		client.MoreHeaders["X-Source"] = "changed"
		client.SetToken("after")
		value.MoreHeaders = map[string]string{"X-Caller-Retry": "allowed"}
		value.JSONBody = policyMarshaler(func() ([]byte, error) {
			replacementMarshals++
			if replacementMarshals > 1 {
				return []byte(`{"n":2}`), nil
			}
			return []byte(`{"n":1}`), nil
		})
		return nil
	}
	originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
	result, err := DoJSON(context.Background(), client, http.MethodPost, client.ServiceURL("objects"), input, nil, 200)
	if err != nil || result == nil || string(result.Body) != `{"ok":true}` || calls != 2 || callbacks != 1 || replacementMarshals != 1 || bodies[0].closes != 1 || bodies[1].closes != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
		t.Fatalf("result=%+v err=%v calls=%d callbacks=%d marshals=%d closes=%d/%d", result, err, calls, callbacks, replacementMarshals, bodies[0].closes, bodies[1].closes)
	}
}

func TestResponsePolicyActualStatusGateAndNativeFailureBoundary(t *testing.T) {
	readCause, closeCause, customCause := errors.New("unexpected body read failed"), errors.New("unexpected close failed"), errors.New("caller canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	header := http.Header{"X-Actual": {"unexpected"}}
	bodies := []*policyBody{{reader: strings.NewReader("retry")}, {reader: policyReader(func(buffer []byte) (int, error) {
		cancel(customCause)
		return copy(buffer, "unexpected partial bytes"), readCause
	}), closeErr: closeCause}}
	var calls int
	client := policyClient(func(*http.Request) (*http.Response, error) {
		calls++
		status := 503
		if calls == 2 {
			status = 201
		}
		return policyHTTP(status, bodies[calls-1], header), nil
	})
	callerCodes := []int{200}
	client.RetryFunc = func(_ context.Context, _, _ string, value *gophercloud.RequestOpts, _ error, _ uint) error {
		value.OkCodes = []int{201}
		callerCodes[0] = 201
		return nil
	}
	result, err := DoJSON(ctx, client, http.MethodGet, client.ServiceURL("objects"), nil, nil, callerCodes...)
	var native gophercloud.ErrUnexpectedResponseCode
	var accepted *resource.ResponseError
	if result != nil || !errors.As(err, &native) || native.Actual != 201 || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != http.MethodGet || native.URL != "https://example.test/reverse/v1/objects" || string(native.Body) != "unexpected partial bytes" || native.ResponseHeader.Get("X-Actual") != "unexpected" || errors.As(err, &accepted) || calls != 2 || bodies[0].closes != 1 || bodies[1].closes != 1 {
		t.Fatalf("result=%+v native=%+v err=%v calls=%d closes=%d/%d", result, native, err, calls, bodies[0].closes, bodies[1].closes)
	}
	for _, cause := range []error{readCause, closeCause, context.Canceled, customCause} {
		if !errors.Is(err, cause) {
			t.Fatalf("unexpected status cause %v lost: %v", cause, err)
		}
	}
	forbidden := &policyBody{reader: strings.NewReader("native forbidden")}
	client = policyClient(func(*http.Request) (*http.Response, error) { return policyHTTP(403, forbidden, header), nil })
	result, err = DoJSON(context.Background(), client, http.MethodGet, client.ServiceURL("objects"), nil, nil, 200)
	if result != nil || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != "native forbidden" || errors.As(err, &accepted) || forbidden.closes != 1 {
		t.Fatalf("native rejected ownership changed: result=%+v err=%v closes=%d", result, err, forbidden.closes)
	}
}
