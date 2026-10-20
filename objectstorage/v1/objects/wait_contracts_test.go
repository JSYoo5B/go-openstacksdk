package objects_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func objectWaitExternalLast(t *testing.T, result *objects.ObjectWaitResult, polls, status int, body string) *objects.ObjectCreateResponse {
	t.Helper()
	if result == nil || result.Polls != polls || result.Last == nil || len(result.Last.Attempts) == 0 {
		t.Fatalf("result=%+v expected polls=%d", result, polls)
	}
	response := result.Last.Attempts[len(result.Last.Attempts)-1].Response
	if response == nil || response.StatusCode != status || string(response.Body) != body || response.Header.Get("X-Trans-Id") != "actual-object" {
		t.Fatalf("last=%+v expected status=%d body=%q", result.Last, status, body)
	}
	return response
}

func TestObjectWaitContractsLiteralHEADAndDeletion(t *testing.T) {
	var calls atomic.Int32
	var progress []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		path := "/reverse/a%20b/v1/AUTH_wait/" + url.PathEscape(objectMetadataContainer) + "/" + url.PathEscape(objectMetadataKey)
		body, readErr := io.ReadAll(r.Body)
		if r.Method != "HEAD" || r.RequestURI != path || r.Header.Get("X-Auth-Token") != "wait-token" || r.Header.Get("X-Wait") != "literal" || readErr != nil || len(body) != 0 || r.ContentLength != 0 {
			t.Errorf("wire=%s %s length=%d body=%v headers=%v", r.Method, r.RequestURI, r.ContentLength, r.Body, r.Header)
		}
		w.Header().Set("X-Trans-Id", "wire-wait")
		w.Header().Set("Content-Length", "0")
		switch call {
		case 1:
			w.WriteHeader(200)
		case 2:
			w.WriteHeader(204)
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	provider := &gophercloud.ProviderClient{HTTPClient: *server.Client()}
	provider.UseTokenLock()
	provider.SetToken("wait-token")
	client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "object-store", Endpoint: server.URL + "/catalogue/v1/AUTH_other/", ResourceBase: server.URL + "/reverse/a%20b/v1/AUTH_wait/"}
	result, err := objects.New(client).WaitForDelete(context.Background(), objectMetadataContainer, objectMetadataKey, objects.WithObjectWaitHeader("X-Wait", "literal"), objects.WithObjectWaitPollInterval(time.Millisecond), objects.WithObjectWaitProgressCallback(func(value int) error { progress = append(progress, value); return nil }))
	if err != nil || result == nil || !result.Complete || !result.Deleted || result.Status != nil || result.Polls != 3 || result.Last == nil || len(result.Last.Attempts) != 1 || calls.Load() != 3 || !reflect.DeepEqual(progress, []int{0, 0}) {
		t.Fatalf("result=%+v err=%v calls=%d progress=%v", result, err, calls.Load(), progress)
	}
	if response := result.Last.Attempts[0].Response; response == nil || response.StatusCode != 404 || response.Header.Get("X-Trans-Id") != "wire-wait" || len(response.Body) != 0 {
		t.Fatalf("last=%+v", result.Last)
	}
}

func TestObjectWaitContractsSelectedStatusAndFailurePolicy(t *testing.T) {
	t.Run("selected metadata header and no unrelated projection", func(t *testing.T) {
		calls, callbacks := 0, 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "HEAD" || r.URL.String() != objectMetadataEndpoint || r.URL.RawQuery != "" || r.Body != nil {
				t.Errorf("HEAD=%s %s", r.Method, r.URL)
			}
			response := objectMetadataResponse(200, "status proof")
			response.Header.Set("X-Object-Meta-State", "RUNNING")
			if calls == 2 {
				response.Header.Set("X-Object-Meta-State", "Ready")
			}
			response.Header.Set("Content-Length", "not an integer")
			response.Header.Set("Last-Modified", "not a date")
			response.Header["X-Unrelated"] = []string{"ignored\x7f"}
			return response, nil
		})
		result, err := objects.New(client).WaitForStatus(context.Background(), objectMetadataContainer, objectMetadataKey, "ready", objects.WithObjectWaitStatusHeader("X-Object-Meta-State"), objects.WithObjectWaitPollInterval(time.Millisecond), objects.WithObjectWaitProgressCallback(func(value int) error {
			callbacks++
			if value != 0 {
				t.Errorf("progress=%d", value)
			}
			return nil
		}))
		objectWaitExternalLast(t, result, 2, 200, "status proof")
		if err != nil || !result.Complete || result.Deleted || result.Status == nil || *result.Status != "Ready" || calls != 2 || callbacks != 1 {
			t.Fatalf("result=%+v err=%v calls=%d callback=%d", result, err, calls, callbacks)
		}
	})
	t.Run("default failure and explicit empty failure list", func(t *testing.T) {
		for _, disabled := range []bool{false, true} {
			calls, callbacks := 0, 0
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				response := objectMetadataResponse(204, "failure policy")
				response.Header.Set("X-Object-Meta-State", "eRrOr")
				if calls == 2 {
					response.Header.Set("X-Object-Meta-State", "ready")
				}
				return response, nil
			})
			opts := []objects.ObjectWaitOption{objects.WithObjectWaitStatusHeader("X-Object-Meta-State"), objects.WithObjectWaitPollInterval(time.Millisecond), objects.WithObjectWaitProgressCallback(func(int) error { callbacks++; return nil })}
			if disabled {
				opts = append(opts, objects.WithObjectWaitFailureStates())
			}
			result, err := objects.New(client).WaitForStatus(context.Background(), "c", "o", "ready", opts...)
			if disabled {
				objectWaitExternalLast(t, result, 2, 204, "failure policy")
				if err != nil || !result.Complete || callbacks != 1 || calls != 2 {
					t.Fatalf("disabled=%v result=%+v err=%v callback=%d HTTP=%d", disabled, result, err, callbacks, calls)
				}
			} else {
				objectWaitExternalLast(t, result, 1, 204, "failure policy")
				if !errors.Is(err, resource.ErrFailedState) || result.Complete || callbacks != 0 || calls != 1 {
					t.Fatalf("result=%+v err=%v callback=%d HTTP=%d", result, err, callbacks, calls)
				}
			}
		}
	})
	t.Run("known native string attribute and explicit empty selected status", func(t *testing.T) {
		for _, tc := range []struct{ attribute, header, target, value string }{{"content_type", "Content-Type", "application/object", "application/object"}, {"", "X-Object-Meta-State", "", ""}} {
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				response := objectMetadataResponse(200, "selected")
				response.Header[tc.header] = []string{tc.value}
				return response, nil
			})
			opts := []objects.ObjectWaitOption{}
			if tc.attribute != "" {
				opts = append(opts, objects.WithObjectWaitStatusAttribute(tc.attribute))
			} else {
				opts = append(opts, objects.WithObjectWaitStatusHeader(tc.header))
			}
			result, err := objects.New(client).WaitForStatus(context.Background(), "c", "o", tc.target, opts...)
			objectWaitExternalLast(t, result, 1, 200, "selected")
			if err != nil || !result.Complete || result.Status == nil || *result.Status != tc.value {
				t.Fatalf("selected=%+v result=%+v err=%v", tc, result, err)
			}
		}
	})
	t.Run("absent default status rejects before HTTP", func(t *testing.T) {
		calls := 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			return objectMetadataResponse(200, "unexpected"), nil
		})
		result, err := objects.New(client).WaitForStatus(context.Background(), "c", "o", "ready")
		if !errors.Is(err, resource.ErrUnsupported) || calls != 0 || result != nil && (result.Last != nil || result.Polls != 0 || result.Complete) {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
	})
}

func TestObjectWaitContractsProjectionIsAtomic(t *testing.T) {
	for _, header := range []http.Header{
		{"X-Object-Meta-State": {"ready"}, "x-object-meta-state": {"ready"}},
		{"X-Object-Meta-State": {"ready", "ready"}},
		{"X-Object-Meta-State": {"bad\x7f"}},
		{"X-Object-Meta-State": {"bad\u0085"}},
		{"X-Object-Meta-State": {string([]byte{0xff})}},
		{},
	} {
		calls, callbacks := 0, 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			response := objectMetadataResponse(200, "projection proof")
			for key, values := range header {
				response.Header[key] = append([]string(nil), values...)
			}
			return response, nil
		})
		result, err := objects.New(client).WaitForStatus(context.Background(), "c", "o", "ready", objects.WithObjectWaitStatusHeader("X-Object-Meta-State"), objects.WithObjectWaitProgressCallback(func(int) error { callbacks++; return nil }))
		objectWaitExternalLast(t, result, 1, 200, "projection proof")
		if err == nil || result.Complete || result.Status != nil || calls != 1 || callbacks != 0 {
			t.Fatalf("header=%v result=%+v err=%v calls=%d callback=%d", header, result, err, calls, callbacks)
		}
		objectMetadataProof(t, err, 200, "projection proof")
	}
}

func TestObjectWaitContractsDirtyMissingAndAcceptedFaults(t *testing.T) {
	for _, status := range []int{200, 404, 501} {
		for _, fault := range []string{"read", "close"} {
			t.Run(http.StatusText(status)+"/"+fault, func(t *testing.T) {
				sentinel := errors.New("wait body " + fault)
				body := &objectMetadataBody{Reader: strings.NewReader("physical proof")}
				if fault == "read" {
					body.Reader = objectMetadataReader(func(p []byte) (int, error) { return copy(p, "physical proof"), sentinel })
				} else {
					body.closeErr = sentinel
				}
				calls, callbacks := 0, 0
				client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataWire(status, body), nil })
				result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWaitProgressCallback(func(int) error { callbacks++; return nil }))
				objectWaitExternalLast(t, result, 1, status, "physical proof")
				if !errors.Is(err, sentinel) || result.Complete || result.Deleted || calls != 1 || callbacks != 0 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v calls=%d callback=%d closes=%d", result, err, calls, callbacks, body.closes.Load())
				}
				if status != 501 {
					objectMetadataProof(t, err, status, "physical proof")
				}
			})
		}
	}
	t.Run("nested transport404 is not absence", func(t *testing.T) {
		cause := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: "HEAD", URL: objectMetadataEndpoint, Body: []byte("not a received response")}
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { return nil, cause })
		result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o")
		if err == nil || result == nil || result.Complete || result.Deleted || result.Polls != 1 || result.Last == nil || len(result.Last.Attempts) != 1 || result.Last.Attempts[0].Response != nil || result.Last.Acknowledgement != nil {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
	t.Run("physical404 plus transport error retains observation without acknowledgement", func(t *testing.T) {
		sentinel := errors.New("response and transport fault")
		body := &objectMetadataBody{Reader: strings.NewReader("not consumed")}
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { return objectMetadataWire(404, body), sentinel })
		result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o")
		objectWaitExternalLast(t, result, 1, 404, "")
		if !errors.Is(err, sentinel) || result.Complete || result.Deleted || result.Last.Acknowledgement != nil || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v closes=%d", result, err, body.closes.Load())
		}
	})
	t.Run("clean501 remains rejected with native and response proof", func(t *testing.T) {
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { return objectMetadataResponse(501, "native failure"), nil })
		result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o")
		objectWaitExternalLast(t, result, 1, 501, "native failure")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 501 || native.Method != "HEAD" || string(native.Body) != "native failure" || result.Complete || result.Deleted {
			t.Fatalf("result=%+v err=%v native=%+v", result, err, native)
		}
	})
	t.Run("status404 is a native error and preserves raw proof", func(t *testing.T) {
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			return objectMetadataResponse(404, "missing status object"), nil
		})
		result, err := objects.New(client).WaitForStatus(context.Background(), "c", "o", "ready", objects.WithObjectWaitStatusHeader("X-Object-Meta-State"))
		objectWaitExternalLast(t, result, 1, 404, "missing status object")
		if !errors.Is(err, resource.ErrNotFound) || result.Complete || result.Deleted || result.Status != nil {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		objectMetadataProof(t, err, 404, "missing status object")
	})
}

func TestObjectWaitContractsDefaultsCallbacksAndCancellation(t *testing.T) {
	stop := errors.New("stop observing")
	for _, tc := range []struct {
		name      string
		status    bool
		options   []objects.ObjectWaitOption
		deadline  bool
		remaining time.Duration
	}{
		{name: "delete default", deadline: true, remaining: 120 * time.Second},
		{name: "status unlimited", status: true},
		{name: "delete unlimited", options: []objects.ObjectWaitOption{objects.WithObjectWaitUnlimitedWait()}},
		{name: "caller timeout override", options: []objects.ObjectWaitOption{objects.WithObjectWaitTimeout(2 * time.Second)}, deadline: true, remaining: 2 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, callbacks := 0, 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				deadline, exists := r.Context().Deadline()
				if exists != tc.deadline || exists && (time.Until(deadline) > tc.remaining || time.Until(deadline) < tc.remaining-time.Second) {
					t.Errorf("deadline=%v exists=%v expected=%v remaining=%v", deadline, exists, tc.deadline, tc.remaining)
				}
				response := objectMetadataResponse(200, "callback proof")
				response.Header.Set("X-Object-Meta-State", "pending")
				return response, nil
			})
			options := append([]objects.ObjectWaitOption{}, tc.options...)
			options = append(options, objects.WithObjectWaitProgressCallback(func(progress int) error {
				callbacks++
				if progress != 0 {
					t.Errorf("progress=%d", progress)
				}
				return stop
			}))
			var result *objects.ObjectWaitResult
			var err error
			if tc.status {
				options = append(options, objects.WithObjectWaitStatusHeader("X-Object-Meta-State"))
				result, err = objects.New(client).WaitForStatus(context.Background(), "c", "o", "ready", options...)
			} else {
				result, err = objects.New(client).WaitForDelete(context.Background(), "c", "o", options...)
			}
			objectWaitExternalLast(t, result, 1, 200, "callback proof")
			if !errors.Is(err, stop) || result.Complete || calls != 1 || callbacks != 1 {
				t.Fatalf("result=%+v err=%v calls=%d callbacks=%d", result, err, calls, callbacks)
			}
		})
	}
	t.Run("callback cancellation takes priority before another poll", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			return objectMetadataResponse(204, "before cancellation"), nil
		})
		result, err := objects.New(client).WaitForDelete(ctx, "c", "o", objects.WithObjectWaitPollInterval(time.Hour), objects.WithObjectWaitProgressCallback(func(int) error { cancel(); return nil }))
		objectWaitExternalLast(t, result, 1, 204, "before cancellation")
		if !errors.Is(err, context.Canceled) || result.Complete || calls != 1 {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
	})
	t.Run("timeout interrupts pause and pre-cancellation does no HTTP", func(t *testing.T) {
		calls := 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			return objectMetadataResponse(200, "before timeout"), nil
		})
		result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWaitTimeout(20*time.Millisecond), objects.WithObjectWaitPollInterval(time.Hour))
		objectWaitExternalLast(t, result, 1, 200, "before timeout")
		if !errors.Is(err, context.DeadlineExceeded) || result.Complete || calls != 1 {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result, err = objects.New(client).WaitForDelete(ctx, "c", "o")
		if !errors.Is(err, context.Canceled) || calls != 1 || result != nil && (result.Polls != 0 || result.Last != nil) {
			t.Fatalf("pre-canceled result=%+v err=%v calls=%d", result, err, calls)
		}
	})
}

func TestObjectWaitContractsNativeReplaySnapshotsAndLiveGuards(t *testing.T) {
	t.Run("native retry keeps physical attempts and fresh auth", func(t *testing.T) {
		calls, optionCalls, retries := 0, 0, 0
		headers := map[string]string{"X-Wait": "captured"}
		factory := objects.WithObjectWaitHeaders(headers)
		headers["X-Wait"] = "mutated"
		var client *gophercloud.ServiceClient
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			token := "original-token"
			if calls == 2 {
				token = "retry-token"
			}
			if r.Method != "HEAD" || r.Body != nil || r.Header.Get("X-Wait") != "captured" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != token {
				t.Errorf("attempt=%d headers=%v", calls, r.Header)
			}
			if calls == 1 {
				return objectMetadataResponse(503, "retry proof"), nil
			}
			return objectMetadataResponse(404, "clean missing"), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 503 {
				return err
			}
			retries++
			client.SetToken("retry-token")
			return nil
		}
		result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o", factory, func(*objects.ObjectWaitOpts) error {
			optionCalls++
			client.MoreHeaders["X-Source"] = "valid drift"
			return nil
		})
		objectWaitExternalLast(t, result, 1, 404, "clean missing")
		if err != nil || !result.Complete || !result.Deleted || calls != 2 || retries != 1 || optionCalls != 1 || len(result.Last.Attempts) != 2 || result.Last.Attempts[0].Response.StatusCode != 503 || string(result.Last.Attempts[0].Response.Body) != "retry proof" || result.Last.Attempts[0].Error == nil {
			t.Fatalf("result=%+v err=%v calls=%d retries=%d options=%d", result, err, calls, retries, optionCalls)
		}
	})
	t.Run("native backoff and reauthentication retain rejected proof", func(t *testing.T) {
		for _, status := range []int{401, 429} {
			calls, hooks := 0, 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "HEAD" || r.Body != nil || calls == 2 && r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Errorf("attempt=%d headers=%v", calls, r.Header)
				}
				if calls == 1 {
					return objectMetadataResponse(status, "rejected proof"), nil
				}
				return objectMetadataResponse(404, "after native hook"), nil
			})
			if status == 401 {
				client.ReauthFunc = func(context.Context) error { hooks++; client.SetToken("refreshed"); return nil }
			} else {
				client.RetryBackoffFunc = func(_ context.Context, proof *gophercloud.ErrUnexpectedResponseCode, _ error, _ uint) error {
					hooks++
					client.SetToken("refreshed")
					proof.Body[0] = 'X'
					proof.ResponseHeader.Set("X-Trans-Id", "mutated callback proof")
					return nil
				}
			}
			result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o")
			objectWaitExternalLast(t, result, 1, 404, "after native hook")
			if err != nil || !result.Complete || calls != 2 || hooks != 1 || len(result.Last.Attempts) != 2 || result.Last.Attempts[0].Response.StatusCode != status || string(result.Last.Attempts[0].Response.Body) != "rejected proof" || result.Last.Attempts[0].Response.Header.Get("X-Trans-Id") != "actual-object" {
				t.Fatalf("status=%d result=%+v err=%v calls=%d hooks=%d", status, result, err, calls, hooks)
			}
		}
	})
	t.Run("transient forbidden source header survives restoration at Close", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		calls := 0
		body := &objectMetadataBody{Reader: strings.NewReader("source proof"), onClose: func() { delete(client.MoreHeaders, "Range") }}
		client = objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			client.MoreHeaders["Range"] = "bytes=0-0"
			return objectMetadataWire(404, body), nil
		})
		client.MoreHeaders = map[string]string{}
		result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o")
		objectWaitExternalLast(t, result, 1, 404, "source proof")
		if !errors.Is(err, resource.ErrInvalidOption) || result.Complete || result.Deleted || calls != 1 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v calls=%d closes=%d", result, err, calls, body.closes.Load())
		}
		objectMetadataProof(t, err, 404, "source proof")
	})
	t.Run("native retry cannot add conditions or broaden accepted codes", func(t *testing.T) {
		for _, mutation := range []string{"Range", "If-Match", "OkCodes", "body"} {
			calls := 0
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				return objectMetadataResponse(503, "before callback policy"), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				switch mutation {
				case "OkCodes":
					opts.OkCodes = append(opts.OkCodes, 503)
				case "body":
					opts.RawBody = strings.NewReader("injected")
				default:
					if opts.MoreHeaders == nil {
						opts.MoreHeaders = map[string]string{}
					}
					opts.MoreHeaders[mutation] = "forbidden"
				}
				return nil
			}
			result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o")
			objectWaitExternalLast(t, result, 1, 503, "before callback policy")
			if !errors.Is(err, resource.ErrInvalidOption) || result.Complete || calls != 1 || len(result.Last.Attempts) != 1 {
				t.Fatalf("mutation=%s result=%+v err=%v calls=%d", mutation, result, err, calls)
			}
		}
	})
	t.Run("redirect callback cannot add a conditional HEAD", func(t *testing.T) {
		calls, redirects := 0, 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			response := objectMetadataResponse(307, "redirect proof")
			response.Header.Set("Location", objectMetadataEndpoint)
			return response, nil
		})
		client.HTTPClient.CheckRedirect = func(r *http.Request, via []*http.Request) error {
			redirects++
			r.Header.Set("If-None-Match", "*")
			return nil
		}
		result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o")
		objectWaitExternalLast(t, result, 1, 307, "redirect proof")
		if !errors.Is(err, resource.ErrInvalidOption) || result.Complete || calls != 1 || redirects != 1 || len(result.Last.Attempts) != 1 {
			t.Fatalf("result=%+v err=%v calls=%d redirects=%d", result, err, calls, redirects)
		}
	})
}
