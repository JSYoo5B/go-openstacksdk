package accounts

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

type accountMetadataTransport func(*http.Request) (*http.Response, error)

func (f accountMetadataTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type accountMetadataBody struct {
	io.Reader
	closeErr error
	closes   atomic.Int32
	onClose  func()
}

func (b *accountMetadataBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type accountMetadataReader func([]byte) (int, error)

func (f accountMetadataReader) Read(p []byte) (int, error) { return f(p) }
func accountMetadataClient(endpoint string) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{TokenID: "token"}
	p.UseTokenLock()
	return &gophercloud.ServiceClient{ProviderClient: p, Endpoint: endpoint, Type: "object-store"}
}
func accountMetadataWire(r *http.Request, code int, header http.Header, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: header, Body: body, Request: r}
}
func accountMetadataProof(t *testing.T, err error, code int, body string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body || proof.Header.Get("X-Proof") != "kept" {
		t.Fatalf("missing response evidence: %v %#v", err, proof)
	}
}

func TestAccountMetadataCoreRoutesAndEvidence(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if r.URL.EscapedPath() != "/reverse%20proxy/v1/AUTH_account/" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "token" {
			t.Errorf("wrong account route/header %s %#v", r.RequestURI, r.Header)
		}
		if body, _ := io.ReadAll(r.Body); len(body) != 0 {
			t.Errorf("unexpected body %q", body)
		}
		if n == 1 {
			if r.Method != "HEAD" || r.Header.Get("X-Newest") != "false" {
				t.Errorf("bad HEAD %#v", r)
			}
			w.Header().Set("X-Account-Bytes-Used", "+0")
			w.Header().Set("X-Account-Object-Count", "-2")
			w.Header().Set("X-Timestamp", "")
			w.Header().Set("X-Account-Meta-Temp-URL-Key", "literal%20secret")
			w.Header().Set("Date", "opaque-date")
		} else {
			if r.Method != "POST" {
				t.Errorf("mutation used %s", r.Method)
			}
			if n == 2 && (r.Header.Get("X-Account-Meta-Book") != "책%20\tvalue" || r.Header.Get("X-Account-Meta-Empty") != "") {
				t.Errorf("metadata transformed %#v", r.Header)
			}
			if n == 3 {
				if _, ok := r.Header["X-Account-Meta-Book"]; !ok {
					t.Errorf("missing delete header")
				}
				if r.Header.Get("X-Remove-Account-Meta-Book") != "" {
					t.Errorf("wrong remove syntax")
				}
			}
		}
		w.Header().Set("X-Proof", "kept")
		w.WriteHeader(204)
	}))
	defer s.Close()
	c := accountMetadataClient(s.URL + "/reverse%20proxy/v1/AUTH_account/")
	c.ResourceBase = "https://other.invalid/object/"
	c.Type = ""
	a := New(c)
	get, err := a.GetMetadata(context.Background(), WithGetMetadataNewest(false))
	if err != nil || get.StatusCode != 204 || get.Metadata.BytesUsed == nil || *get.Metadata.BytesUsed != 0 || get.Metadata.ContainerCount != nil || *get.Metadata.ObjectCount != -2 || get.Metadata.Timestamp == nil || get.Metadata.Values["temp-url-key"] != "literal%20secret" {
		t.Fatalf("GET %+v %v", get, err)
	}
	get.Header.Set("X-Account-Meta-Temp-URL-Key", "changed")
	if get.Metadata.Values["temp-url-key"] != "literal%20secret" {
		t.Fatal("projection aliases Header")
	}
	set, err := a.SetMetadata(context.Background(), map[string]string{"Book": "책%20\tvalue", "Empty": ""})
	if err != nil || set.StatusCode != 204 {
		t.Fatalf("set %+v %v", set, err)
	}
	if _, err := a.DeleteMetadata(context.Background(), []string{"Book"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMetadata(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := a.DeleteMetadata(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 5 {
		t.Fatalf("hidden request: %d", calls.Load())
	}
}

func TestAccountMetadataCoreAtomicHeaderProjection(t *testing.T) {
	bad := []http.Header{
		{"X-Account-Meta-K": {"alias"}},
		{"X-Account-Bytes-Used": {"9223372036854775808"}},
		{"X-Account-Object-Count": {" 1"}},
		{"X-Account-Container-Count": {"1.0"}},
		{"X-Account-Bytes-Used": {"1"}, "x-account-bytes-used": {"1"}},
		{"X-Account-Meta-Key": {"a", "b"}},
		{"X-Account-Meta-Key": {}},
		{"X-Account-Meta-Key": {"a"}, "x-account-meta-key": {"a"}},
		{"X-Account-Meta-": {"a"}},
		{"X-Timestamp": {"one", "two"}},
	}
	for i, header := range bad {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			body := &accountMetadataBody{Reader: strings.NewReader("wire")}
			header.Set("X-Proof", "kept")
			c := accountMetadataClient("http://account.invalid/v1/AUTH_x/")
			c.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) { return accountMetadataWire(r, 204, header, body), nil })
			got, err := New(c).GetMetadata(context.Background())
			if got == nil || got.Metadata != nil || got.StatusCode != 204 || string(got.Body) != "wire" || body.closes.Load() != 1 {
				t.Fatalf("non-atomic result %+v %v", got, err)
			}
			accountMetadataProof(t, err, 204, "wire")
			if i == 1 {
				var number *strconv.NumError
				if !errors.As(err, &number) {
					t.Fatalf("lost numeric cause %v", err)
				}
			}
		})
	}
	info, err := projectMetadata(http.Header{"Date": {"not a date"}, "X-Account-Meta-Quota-Bytes": {"999999999999999999999"}, "X-Timestamp": {""}})
	if err != nil || info.Values == nil || info.BytesUsed != nil || info.Timestamp == nil || *info.Timestamp != "" || info.Values["quota-bytes"] != "999999999999999999999" {
		t.Fatalf("passive fields %+v %v", info, err)
	}
}

func TestAccountMetadataCoreCompletePreflight(t *testing.T) {
	var calls atomic.Int32
	c := accountMetadataClient("http://account.invalid/v1/AUTH_x/")
	c.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return accountMetadataWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	a := New(c)
	for _, input := range []map[string]string{{"": "x"}, {"bad key": "x"}, {"x-ACCOUNT-meta-Key": "x"}, {"Key": "x", "key": "x"}, {"Key": "a\r\nb"}, {"Key": string([]byte{255})}, {"Key": "a\x00b"}} {
		invoked := 0
		got, err := a.SetMetadata(context.Background(), input, func(*MetadataOpts) error { invoked++; return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || invoked != 0 {
			t.Fatalf("bad input accepted %v %v %d", input, err, invoked)
		}
	}
	if _, err := a.DeleteMetadata(context.Background(), []string{"Key", "KEY"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := a.GetMetadata(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := (*API)(nil).GetMetadata(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := a.GetMetadata(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, key := range []string{"X-Newest", "x-account-meta-hidden", "X-Remove-Account-Meta-hidden", "Proxy-Authorization"} {
		c.MoreHeaders = map[string]string{key: "x"}
		if _, err := a.GetMetadata(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("owned source %s %v", key, err)
		}
	}
	c.MoreHeaders = nil
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop")
	cancel(cause)
	if _, err := a.GetMetadata(ctx); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
	c.Type = "image"
	if _, err := a.GetMetadata(context.Background()); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatalf("preflight sent %d", calls.Load())
	}
}

func TestAccountMetadataCoreAcceptedFailures(t *testing.T) {
	for _, get := range []bool{false, true} {
		t.Run(strconv.FormatBool(get), func(t *testing.T) {
			readErr := errors.New("read")
			closeErr := errors.New("close")
			cause := errors.New("cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &accountMetadataBody{Reader: accountMetadataReader(func(p []byte) (int, error) { cancel(cause); return copy(p, "part"), readErr }), closeErr: closeErr}
			var calls atomic.Int32
			c := accountMetadataClient("http://account.invalid/v1/AUTH_x/")
			c.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return accountMetadataWire(r, 204, http.Header{"X-Proof": {"kept"}}, body), nil
			})
			var err error
			if get {
				var result *GetMetadataResult
				result, err = New(c).GetMetadata(ctx)
				if result == nil || result.Metadata != nil {
					t.Fatalf("Get lost evidence %+v", result)
				}
			} else {
				var result *MetadataResponse
				result, err = New(c).SetMetadata(ctx, nil)
				if result == nil || result.StatusCode != 204 {
					t.Fatalf("POST lost evidence %+v", result)
				}
			}
			accountMetadataProof(t, err, 204, "part")
			for _, want := range []error{readErr, closeErr, context.Canceled, cause} {
				if !errors.Is(err, want) {
					t.Fatalf("lost cause %v: %v", want, err)
				}
			}
			if body.closes.Load() != 1 || calls.Load() != 1 {
				t.Fatalf("close/replay %d/%d", body.closes.Load(), calls.Load())
			}
		})
	}
}

func TestAccountMetadataCoreSourceAndTargetBoundaries(t *testing.T) {
	var calls atomic.Int32
	c := accountMetadataClient("http://account.invalid/v1/AUTH_x/")
	c.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return accountMetadataWire(r, 204, http.Header{"X-Proof": {"kept"}}, &accountMetadataBody{Reader: strings.NewReader("ack"), onClose: func() { c.Endpoint = "http://else.invalid/" }}), nil
	})
	a := New(c)
	result, err := a.SetMetadata(context.Background(), nil)
	if result == nil || result.StatusCode != 204 || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("source drift lost %+v %v", result, err)
	}
	accountMetadataProof(t, err, 204, "ack")
	c.Endpoint = "http://account.invalid/v1/AUTH_x/"
	optionErr := errors.New("callback")
	_, err = a.GetMetadata(context.Background(), func(*GetMetadataOpts) error { c.Endpoint = "http://else.invalid/"; return optionErr })
	if !errors.Is(err, optionErr) || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
		t.Fatalf("callback/source %v", err)
	}
}

func TestAccountMetadataCoreNativePolicyAndCompatibility(t *testing.T) {
	c := accountMetadataClient("http://account.invalid/v1/AUTH_x/")
	var calls atomic.Int32
	c.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
		code := 503
		if calls.Add(1) > 1 {
			code = 202
		}
		return accountMetadataWire(r, code, http.Header{}, io.NopCloser(strings.NewReader("native"))), nil
	})
	c.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		opts.OkCodes = []int{202}
		return nil
	}
	got, err := New(c).SetMetadata(context.Background(), nil)
	var code gophercloud.ErrUnexpectedResponseCode
	if got != nil || !errors.As(err, &code) || code.Actual != 202 || len(code.Expected) != 1 || code.Expected[0] != 204 || string(code.Body) != "native" || calls.Load() != 2 {
		t.Fatalf("expanded status accepted %+v %v %#v", got, err, code)
	}
	c.RetryFunc = nil
	c.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
		return accountMetadataWire(r, 201, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	if _, err := New(c).Update(context.Background(), UpdateOpts{Metadata: map[string]string{"legacy": "kept"}}, WithUpdateHeader("X-Trace", "old")); err != nil {
		t.Fatalf("native Update compatibility %v", err)
	}
}
