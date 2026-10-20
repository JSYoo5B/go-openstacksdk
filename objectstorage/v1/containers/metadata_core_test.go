package containers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
)

const containerMetadataName = "한 %?#: 컨테이너"

type containerMetadataTransport func(*http.Request) (*http.Response, error)

func (f containerMetadataTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type containerMetadataBody struct {
	io.Reader
	closeErr error
	closes   atomic.Int32
	onClose  func()
}

func (b *containerMetadataBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type containerMetadataReader func([]byte) (int, error)

func (f containerMetadataReader) Read(p []byte) (int, error) { return f(p) }
func containerMetadataClient(endpoint string) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{TokenID: "token"}
	p.UseTokenLock()
	return &gophercloud.ServiceClient{ProviderClient: p, Endpoint: endpoint, Type: "object-store"}
}
func containerMetadataWire(r *http.Request, code int, header http.Header, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: header, Body: body, Request: r}
}
func containerMetadataProof(t *testing.T, err error, code int, body string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body || proof.Header.Get("X-Proof") != "kept" {
		t.Fatalf("missing response evidence: %v %#v", err, proof)
	}
}

func TestContainerMetadataCoreRoutesAndEvidence(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if r.URL.EscapedPath() != "/reverse%20proxy/v1/AUTH_account/"+url.PathEscape(containerMetadataName) || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "token" {
			t.Errorf("wrong container route/header %s %#v", r.RequestURI, r.Header)
		}
		for key := range r.Header {
			if strings.HasPrefix(strings.ToLower(key), "x-account-meta-") {
				t.Errorf("foreign account prefix %s", key)
			}
		}
		if body, _ := io.ReadAll(r.Body); len(body) != 0 {
			t.Errorf("unexpected body %q", body)
		}
		if n == 1 {
			if r.Method != "HEAD" || r.Header.Get("X-Newest") != "false" {
				t.Errorf("bad HEAD %#v", r)
			}
			w.Header().Set("X-Container-Bytes-Used", "+0")
			w.Header().Set("X-Container-Object-Count", "-2")
			w.Header().Set("X-Timestamp", "")
			w.Header().Set("X-Container-Meta-Temp-URL-Key", "literal%20secret")
			w.Header().Set("Date", "opaque-date")
			w.Header().Set("Last-Modified", "literal-not-time")
			w.Header().Set("X-Container-Read", "raw ACL")
		} else {
			if r.Method != "POST" {
				t.Errorf("mutation used %s", r.Method)
			}
			if n == 2 && (r.Header.Get("X-Container-Meta-Book") != "책%20\tvalue" || r.Header.Get("X-Container-Meta-Empty") != "" || len(r.Header.Values("X-Container-Meta-Empty")) != 1) {
				t.Errorf("metadata transformed %#v", r.Header)
			}
			if n == 2 && (r.Header.Get("X-Container-Sync-Key") != "literal%20" || len(r.Header.Values("X-Container-Read")) != 1 || r.Header.Get("X-Container-Read") != "") {
				t.Errorf("system headers %#v", r.Header)
			}
			if n == 3 {
				if _, ok := r.Header["X-Container-Meta-Book"]; !ok {
					t.Errorf("missing delete header")
				}
				if _, present := r.Header["X-Remove-Container-Meta-Book"]; present {
					t.Errorf("wrong remove syntax")
				}
			}
		}
		w.Header().Set("X-Proof", "kept")
		w.WriteHeader(204)
	}))
	defer s.Close()
	c := containerMetadataClient(s.URL + "/endpoint/ignored/")
	c.ResourceBase = s.URL + "/reverse%20proxy/v1/AUTH_account/"
	c.Type = ""
	a := New(c)
	get, err := a.GetMetadata(context.Background(), containerMetadataName, WithGetMetadataNewest(false))
	if err != nil || get.StatusCode != 204 || get.Metadata.BytesUsed == nil || *get.Metadata.BytesUsed != 0 || *get.Metadata.ObjectCount != -2 || get.Metadata.Timestamp == nil || get.Metadata.LastModified == nil || *get.Metadata.LastModified != "literal-not-time" || get.Header.Get("X-Container-Read") != "raw ACL" || get.Metadata.Values["temp-url-key"] != "literal%20secret" {
		t.Fatalf("GET %+v %v", get, err)
	}
	get.Header.Set("X-Container-Meta-Temp-URL-Key", "changed")
	if get.Metadata.Values["temp-url-key"] != "literal%20secret" {
		t.Fatal("projection aliases Header")
	}
	set, err := a.SetMetadata(context.Background(), containerMetadataName, map[string]string{"Book": "책%20\tvalue", "Empty": ""}, WithMetadataHeaders(map[string]string{"X-Container-Sync-Key": "literal%20", "X-Container-Read": ""}))
	if err != nil || set.StatusCode != 204 {
		t.Fatalf("set %+v %v", set, err)
	}
	if _, err := a.DeleteMetadata(context.Background(), containerMetadataName, []string{"Book"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMetadata(context.Background(), containerMetadataName, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := a.DeleteMetadata(context.Background(), containerMetadataName, nil); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 5 {
		t.Fatalf("hidden request: %d", calls.Load())
	}
}

func TestContainerMetadataCoreAtomicHeaderProjection(t *testing.T) {
	bad := []http.Header{
		{"X-Container-Meta-K": {"alias"}},
		{"X-Container-Bytes-Used": {"9223372036854775808"}},
		{"X-Container-Object-Count": {" 1"}},
		{"Last-Modified": {"one", "two"}},
		{"Last-Modified": {"\x00"}},
		{"X-Container-Bytes-Used": {"1"}, "x-container-bytes-used": {"1"}},
		{"X-Container-Meta-Key": {"a", "b"}},
		{"X-Container-Meta-Key": {}},
		{"X-Container-Meta-Key": {"a"}, "x-container-meta-key": {"a"}},
		{"X-Container-Meta-": {"a"}},
		{"X-Timestamp": {"one", "two"}},
	}
	for i, header := range bad {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			body := &containerMetadataBody{Reader: strings.NewReader("wire")}
			header.Set("X-Proof", "kept")
			c := containerMetadataClient("http://account.invalid/v1/AUTH_x/")
			c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) { return containerMetadataWire(r, 204, header, body), nil })
			got, err := New(c).GetMetadata(context.Background(), containerMetadataName)
			if got == nil || got.Metadata != nil || got.StatusCode != 204 || string(got.Body) != "wire" || body.closes.Load() != 1 {
				t.Fatalf("non-atomic result %+v %v", got, err)
			}
			containerMetadataProof(t, err, 204, "wire")
			if i == 1 {
				var number *strconv.NumError
				if !errors.As(err, &number) {
					t.Fatalf("lost numeric cause %v", err)
				}
			}
		})
	}
	info, err := projectMetadata(http.Header{"Date": {"not a date"}, "X-Container-Meta-Quota-Bytes": {"999999999999999999999"}, "X-Timestamp": {""}, "Last-Modified": {"invalid date %20"}, "X-Container-Read": {"first", "second"}})
	if err != nil || info.Values == nil || info.BytesUsed != nil || info.Timestamp == nil || *info.Timestamp != "" || info.Values["quota-bytes"] != "999999999999999999999" || info.LastModified == nil || *info.LastModified != "invalid date %20" {
		t.Fatalf("passive fields %+v %v", info, err)
	}
}

func TestContainerMetadataCoreCompletePreflight(t *testing.T) {
	var calls atomic.Int32
	c := containerMetadataClient("http://account.invalid/v1/AUTH_x/")
	c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return containerMetadataWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	a := New(c)
	for _, value := range []string{"", "/", ".", "..", "bad\\name", "line\n", "\t", "\x7f", string([]byte{255})} {
		invoked := 0
		result, err := a.GetMetadata(context.Background(), value, func(*GetMetadataOpts) error { invoked++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || invoked != 0 {
			t.Fatalf("name %q accepted %+v %v callback%d", value, result, err, invoked)
		}
		if value == "" {
			var cause swift.ErrEmptyContainerName
			if !errors.As(err, &cause) {
				t.Fatalf("empty native cause %v", err)
			}
		}
		if value == "/" {
			var cause swift.ErrInvalidContainerName
			if !errors.As(err, &cause) {
				t.Fatalf("slash native cause %v", err)
			}
		}
	}
	for _, input := range []map[string]string{{"": "x"}, {"bad key": "x"}, {"x-CONTAINER-meta-Key": "x"}, {"Key": "x", "key": "x"}, {"Key": "a\r\nb"}, {"Key": string([]byte{255})}, {"Key": "a\x00b"}} {
		invoked := 0
		got, err := a.SetMetadata(context.Background(), containerMetadataName, input, func(*MetadataOpts) error { invoked++; return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || invoked != 0 {
			t.Fatalf("bad input accepted %v %v %d", input, err, invoked)
		}
	}
	if _, err := a.DeleteMetadata(context.Background(), containerMetadataName, []string{"Key", "KEY"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := a.GetMetadata(nil, containerMetadataName); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := (*API)(nil).GetMetadata(context.Background(), containerMetadataName); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := a.GetMetadata(context.Background(), containerMetadataName, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, key := range []string{"X-Newest", "x-container-meta-hidden", "X-Remove-Container-Meta-hidden", "Proxy-Authorization"} {
		c.MoreHeaders = map[string]string{key: "x"}
		if _, err := a.GetMetadata(context.Background(), containerMetadataName); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("owned source %s %v", key, err)
		}
	}
	c.MoreHeaders = nil
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop")
	cancel(cause)
	if _, err := a.GetMetadata(ctx, containerMetadataName); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
	c.Type = "image"
	if _, err := a.GetMetadata(context.Background(), containerMetadataName); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatalf("preflight sent %d", calls.Load())
	}
}

func TestContainerMetadataCoreAcceptedFailures(t *testing.T) {
	for _, get := range []bool{false, true} {
		t.Run(strconv.FormatBool(get), func(t *testing.T) {
			readErr := errors.New("read")
			closeErr := errors.New("close")
			cause := errors.New("cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &containerMetadataBody{Reader: containerMetadataReader(func(p []byte) (int, error) { cancel(cause); return copy(p, "part"), readErr }), closeErr: closeErr}
			var calls atomic.Int32
			c := containerMetadataClient("http://account.invalid/v1/AUTH_x/")
			c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return containerMetadataWire(r, 204, http.Header{"X-Proof": {"kept"}}, body), nil
			})
			var err error
			if get {
				var result *GetMetadataResult
				result, err = New(c).GetMetadata(ctx, containerMetadataName)
				if result == nil || result.Metadata != nil {
					t.Fatalf("Get lost evidence %+v", result)
				}
			} else {
				var result *MetadataResponse
				result, err = New(c).SetMetadata(ctx, containerMetadataName, nil)
				if result == nil || result.StatusCode != 204 {
					t.Fatalf("POST lost evidence %+v", result)
				}
			}
			containerMetadataProof(t, err, 204, "part")
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

func TestContainerMetadataCoreSourceAndTargetBoundaries(t *testing.T) {
	var calls atomic.Int32
	c := containerMetadataClient("http://account.invalid/v1/AUTH_x/")
	c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return containerMetadataWire(r, 204, http.Header{"X-Proof": {"kept"}}, &containerMetadataBody{Reader: strings.NewReader("ack"), onClose: func() { c.Endpoint = "http://else.invalid/" }}), nil
	})
	a := New(c)
	result, err := a.SetMetadata(context.Background(), containerMetadataName, nil)
	if result == nil || result.StatusCode != 204 || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("source drift lost %+v %v", result, err)
	}
	containerMetadataProof(t, err, 204, "ack")
	c.Endpoint = "http://account.invalid/v1/AUTH_x/"
	optionErr := errors.New("callback")
	_, err = a.GetMetadata(context.Background(), containerMetadataName, func(*GetMetadataOpts) error { c.ResourceBase = c.Endpoint; return optionErr })
	if !errors.Is(err, optionErr) || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
		t.Fatalf("callback/source %v", err)
	}
	c.ResourceBase = ""
	for _, base := range []string{"http://other.invalid/v1/", "http://account.invalid/v1?", "http://account.invalid/v1/?x=1", "http://user@account.invalid/v1/", "http://account.invalid/v1/#x", "http://account.invalid/v1", "file:///v1/"} {
		c.ResourceBase = base
		invoked := false
		got, err := a.GetMetadata(context.Background(), containerMetadataName, func(*GetMetadataOpts) error { invoked = true; return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || invoked || calls.Load() != 1 {
			t.Fatalf("base %q accepted %+v %v", base, got, err)
		}
	}
	c.ResourceBase = c.Endpoint
	for _, mutate := range []func(){func() { c.Endpoint = "http://account.invalid/v1/AUTH_x" }, func() { c.Microversion = "new" }, func() { c.ProviderClient = &gophercloud.ProviderClient{} }} {
		beforeEndpoint, beforeVersion, beforeProvider := c.Endpoint, c.Microversion, c.ProviderClient
		got, err := a.GetMetadata(context.Background(), containerMetadataName, func(*GetMetadataOpts) error { mutate(); return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
			t.Fatalf("source retarget %+v %v", got, err)
		}
		c.Endpoint, c.Microversion, c.ProviderClient = beforeEndpoint, beforeVersion, beforeProvider
	}
}

func TestContainerMetadataCoreNativePolicyAndCompatibility(t *testing.T) {
	c := containerMetadataClient("http://account.invalid/v1/AUTH_x/")
	var calls atomic.Int32
	c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
		code := 503
		if calls.Add(1) > 1 {
			code = 202
		}
		return containerMetadataWire(r, code, http.Header{}, io.NopCloser(strings.NewReader("native"))), nil
	})
	c.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		opts.OkCodes = []int{202}
		return nil
	}
	got, err := New(c).SetMetadata(context.Background(), containerMetadataName, nil)
	var code gophercloud.ErrUnexpectedResponseCode
	if got != nil || !errors.As(err, &code) || code.Actual != 202 || len(code.Expected) != 1 || code.Expected[0] != 204 || string(code.Body) != "native" || calls.Load() != 2 {
		t.Fatalf("expanded status accepted %+v %v %#v", got, err, code)
	}
	c.RetryFunc = nil
	c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
		return containerMetadataWire(r, 201, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	if _, err := New(c).Update(context.Background(), containerMetadataName, UpdateOpts{Metadata: map[string]string{"legacy": "kept"}}, WithUpdateHeader("X-Trace", "old")); err != nil {
		t.Fatalf("native Update compatibility %v", err)
	}
}
