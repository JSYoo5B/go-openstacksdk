package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

func TestAcceleratorRequestAcceptedBodyCancellationRetainsResponse(t *testing.T) {
	cloud := testcloud.New(t)
	const prefix = `{"arqs":`
	var calls, retries, closes atomic.Int32
	cloud.Mux.HandleFunc("/v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" {
			t.Errorf("method=%s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "accepted-before-cancel")
		w.WriteHeader(201)
		_, _ = io.WriteString(w, prefix)
		w.(http.Flusher).Flush()
		// Keep the accepted response incomplete until the client cancels its
		// body read. This is a real HTTP transfer, rather than a decode mock.
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transport := cloud.Provider.HTTPClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	cloud.Provider.HTTPClient.Transport = acceleratorResponseBoundaryTransport(func(r *http.Request) (*http.Response, error) {
		response, err := transport.RoundTrip(r)
		if err == nil {
			response.Body = &acceleratorResponseBoundaryBody{ReadCloser: response.Body, cancel: cancel, closes: &closes}
		}
		return response, err
	})
	cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries.Add(1)
		return errors.New("accepted body cancellation must not retry creation")
	}
	value, err := acceleratorrequests.New(cloud.Client("accelerator", "/v2")).Create(ctx,
		acceleratorrequests.CreateOpts{DeviceProfileName: "profile"})
	if !errors.Is(err, context.Canceled) || value == nil || value.StatusCode != 201 || value.Header.Get("X-Request-Id") != "accepted-before-cancel" || string(value.RawBody) != prefix || value.Requests == nil || len(value.Requests) != 0 {
		t.Fatalf("accepted response lost during cancellation: value=%+v error=%v", value, err)
	}
	if calls.Load() != 1 || retries.Load() != 0 || closes.Load() != 1 {
		t.Fatalf("calls/retries/body closes=%d/%d/%d", calls.Load(), retries.Load(), closes.Load())
	}
}

func TestAcceleratorRequestMalformedAcceptedJSONBypassesRetryHook(t *testing.T) {
	cloud := testcloud.New(t)
	const raw = " \n{\"arqs\":\n"
	var calls, retries atomic.Int32
	cloud.Mux.HandleFunc("/v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || r.Method != "POST" || string(body["device_profile_name"]) != `"profile"` {
			t.Errorf("request: method=%s body=%s error=%v", r.Method, body, err)
		}
		w.Header().Set("X-Request-Id", "accepted-malformed")
		testcloud.JSON(w, 201, raw)
	})
	// Permit one native retry so this fixture detects accidental JSONResponse
	// decoding inside ProviderClient.Request without entering an endless loop.
	cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		if retries.Add(1) == 1 {
			return nil
		}
		return errors.New("unexpected accepted-response retry")
	}
	value, err := acceleratorrequests.New(cloud.Client("accelerator", "/v2")).Create(context.Background(),
		acceleratorrequests.CreateOpts{DeviceProfileName: "profile"})
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) || value == nil || value.StatusCode != 201 || value.Header.Get("X-Request-Id") != "accepted-malformed" || string(value.RawBody) != raw || value.Requests == nil || len(value.Requests) != 0 {
		t.Fatalf("accepted malformed response lost: value=%+v error=%v", value, err)
	}
	if calls.Load() != 1 || retries.Load() != 0 {
		t.Fatalf("accepted creation was retried: calls=%d retries=%d", calls.Load(), retries.Load())
	}
}

type acceleratorResponseBoundaryTransport func(*http.Request) (*http.Response, error)

func (f acceleratorResponseBoundaryTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type acceleratorResponseBoundaryBody struct {
	io.ReadCloser
	cancel context.CancelFunc
	once   sync.Once
	closes *atomic.Int32
}

func (b *acceleratorResponseBoundaryBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		// Cancel only after the SDK has received response headers and read body
		// bytes; scheduling or network speed cannot turn this into a preflight
		// cancellation fixture.
		b.once.Do(b.cancel)
	}
	return n, err
}

func (b *acceleratorResponseBoundaryBody) Close() error {
	b.closes.Add(1)
	return b.ReadCloser.Close()
}
