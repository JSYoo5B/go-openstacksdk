package containers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

type lifecycleTransport func(*http.Request) (*http.Response, error)

func (f lifecycleTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func lifecycleAPI(transport http.RoundTripper) (*API, *gophercloud.ServiceClient) {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: transport}, TokenID: "live-token"}
	provider.UseTokenLock()
	client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: "https://swift.invalid/v1/AUTH_account/", Type: "object-store"}
	return New(client), client
}
func lifecycleWire(r *http.Request, code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"X-Proof": {"actual"}}, Body: body, Request: r}
}
func lifecycleStatic(code int, body string, calls *atomic.Int32) lifecycleTransport {
	return func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return lifecycleWire(r, code, io.NopCloser(strings.NewReader(body))), nil
	}
}

type lifecycleBody struct {
	data              string
	readErr, closeErr error
	onRead, onClose   func()
	read              bool
	closes            int
}

func (b *lifecycleBody) Read(p []byte) (int, error) {
	if b.read {
		return 0, io.EOF
	}
	b.read = true
	if b.onRead != nil {
		b.onRead()
	}
	return copy(p, b.data), b.readErr
}
func (b *lifecycleBody) Close() error {
	b.closes++
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}
func lifecycleProof(t *testing.T, result *ContainerResponse, err error, code int, text string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if result == nil || result.StatusCode != code || string(result.Body) != text || result.IgnoredMissing || !errors.As(err, &proof) {
		t.Fatalf("accepted proof: result=%+v err=%v", result, err)
	}
	if proof.StatusCode != code || string(proof.Body) != text || proof.Header.Get("X-Proof") != "actual" {
		t.Fatalf("error proof=%+v", proof)
	}
	result.Header.Set("X-Proof", "caller")
	if len(result.Body) > 0 {
		result.Body[0] = '!'
	}
	if string(proof.Body) != text || proof.Header.Get("X-Proof") != "actual" {
		t.Fatal("result aliases error evidence")
	}
	return proof
}

func TestContainerLifecycleCoreWireIdentity(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if r.RequestURI != "/reverse%20prefix/v1/AUTH_account/%EB%8C%80%25%20%3F%23" || r.URL.Path != "/reverse prefix/v1/AUTH_account/대% ?#" {
			t.Errorf("literal route=%q path=%q", r.RequestURI, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live-token" {
			t.Error("unexpected body/query/auth")
		}
		w.Header().Set("X-Proof", "actual")
		if n < 3 {
			if r.Method != "PUT" || r.Header.Get("X-Container-Meta-Keep") != "literal" || r.Header.Get("X-Container-Read") != ".r:*" || r.Header.Get("X-Account-Meta-Keep") != "" {
				t.Error("wrong PUT headers or method")
			}
			if _, ok := r.Header["X-Container-Meta-Remove"]; !ok {
				t.Error("empty metadata omitted")
			}
			w.WriteHeader(200 + int(n))
			_, _ = w.Write([]byte("opaque"))
		} else {
			if r.Method != "DELETE" || r.Header.Get("X-Container-Meta-Keep") != "" {
				t.Error("DELETE inherited create metadata")
			}
			w.WriteHeader(204)
		}
	}))
	defer server.Close()
	a, client := lifecycleAPI(server.Client().Transport)
	client.Endpoint, client.ResourceBase = server.URL+"/endpoint/v1/AUTH_account/", server.URL+"/reverse%20prefix/v1/AUTH_account/"
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	for _, code := range []int{201, 202} {
		r, err := a.CreateContainer(context.Background(), "대% ?#", WithCreateContainerMetadata(map[string]string{"Keep": "literal", "Remove": ""}), WithCreateContainerHeader("X-Container-Read", ".r:*"))
		if err != nil || r == nil || r.StatusCode != code || string(r.Body) != "opaque" || r.IgnoredMissing {
			t.Fatalf("PUT%d: %+v %v", code, r, err)
		}
	}
	r, err := a.DeleteContainer(context.Background(), "대% ?#")
	if err != nil || r == nil || r.StatusCode != 204 || r.IgnoredMissing || calls.Load() != 3 {
		t.Fatalf("DELETE: %+v %v calls%d", r, err, calls.Load())
	}
	if len(client.MoreHeaders) != 1 || client.MoreHeaders["X-Source"] != "captured" {
		t.Fatal("shared source headers mutated")
	}
}

func TestContainerLifecycleCoreStrictStatusesAndMissing(t *testing.T) {
	for _, tc := range []struct {
		code           int
		create, strict bool
		ignored        bool
	}{{201, true, false, false}, {202, true, false, false}, {204, true, false, false}, {202, false, false, false}, {204, false, false, false}, {404, false, false, true}, {404, false, true, false}, {409, false, false, false}} {
		t.Run(strings.Join([]string{http.StatusText(tc.code), map[bool]string{true: "create", false: "delete"}[tc.create], map[bool]string{true: "strict", false: "default"}[tc.strict]}, "/"), func(t *testing.T) {
			var calls atomic.Int32
			a, _ := lifecycleAPI(lifecycleStatic(tc.code, "raw status", &calls))
			var r *ContainerResponse
			var err error
			if tc.create {
				r, err = a.CreateContainer(context.Background(), "name")
			} else if tc.strict {
				r, err = a.DeleteContainer(context.Background(), "name", WithDeleteContainerIgnoreMissing(false))
			} else {
				r, err = a.DeleteContainer(context.Background(), "name")
			}
			accepted := tc.create && (tc.code == 201 || tc.code == 202) || !tc.create && (tc.code == 204 || tc.code == 404 && !tc.strict)
			if accepted {
				if err != nil || r == nil || r.StatusCode != tc.code || r.IgnoredMissing != tc.ignored || string(r.Body) != "raw status" {
					t.Fatalf("accepted %+v %v", r, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if r != nil || !errors.As(err, &native) || native.Actual != tc.code || string(native.Body) != "raw status" {
					t.Fatalf("unexpected %+v %v", r, err)
				}
			}
			if calls.Load() != 1 {
				t.Fatal("extra request")
			}
		})
	}
}

func TestContainerLifecycleCoreCompletePreflight(t *testing.T) {
	var calls atomic.Int32
	a, client := lifecycleAPI(lifecycleStatic(201, "", &calls))
	for _, name := range []string{"", "a/b", ".", "..", "a\\b", "a\n", "a\x7f", string([]byte{255})} {
		callbacks := 0
		_, err := a.CreateContainer(context.Background(), name, func(*CreateContainerOpts) error { callbacks++; return nil })
		if !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatalf("name%q err%v callbacks%d", name, err, callbacks)
		}
	}
	for _, mutate := range []func(){func() { client.ResourceBase = "https://foreign.invalid/v1/" }, func() { client.ResourceBase = client.Endpoint + "?bad=1" }, func() { client.ResourceBase = strings.TrimSuffix(client.Endpoint, "/") }} {
		mutate()
		_, err := a.DeleteContainer(context.Background(), "name")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("source: %v", err)
		}
		client.ResourceBase = ""
	}
	_, err := a.CreateContainer(nil, "name")
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil context%v", err)
	}
	var nilAPI *API
	_, err = nilAPI.DeleteContainer(context.Background(), "name")
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil API%v", err)
	}
	client.Type = "image"
	_, err = a.DeleteContainer(context.Background(), "name")
	if !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("type%v", err)
	}
	client.Type = "object-store"
	cause := errors.New("canceled before wire")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cancel(cause)
	_, err = a.CreateContainer(ctx, "name")
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatalf("context%v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("preflight reached HTTP")
	}
}

func TestContainerLifecycleCoreAcceptedResponseFailures(t *testing.T) {
	for _, code := range []int{201, 202, 204, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			readCause, closeCause, cancelCause := errors.New("read failed"), errors.New("close failed"), errors.New("caller cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &lifecycleBody{data: "partial", readErr: readCause, closeErr: closeCause, onRead: func() { cancel(cancelCause) }}
			a, _ := lifecycleAPI(lifecycleTransport(func(r *http.Request) (*http.Response, error) { return lifecycleWire(r, code, body), nil }))
			var result *ContainerResponse
			var err error
			if code < 204 {
				result, err = a.CreateContainer(ctx, "name")
			} else {
				result, err = a.DeleteContainer(ctx, "name")
			}
			lifecycleProof(t, result, err, code, "partial")
			for _, cause := range []error{readCause, closeCause, context.Canceled, cancelCause} {
				if !errors.Is(err, cause) {
					t.Fatalf("lost cause%v: %v", cause, err)
				}
			}
			if body.closes != 1 {
				t.Fatalf("body closes%d", body.closes)
			}
		})
	}
	body := &lifecycleBody{data: "complete", closeErr: errors.New("close only")}
	a, _ := lifecycleAPI(lifecycleTransport(func(r *http.Request) (*http.Response, error) { return lifecycleWire(r, 404, body), nil }))
	r, err := a.DeleteContainer(context.Background(), "name")
	lifecycleProof(t, r, err, 404, "complete")
	if !errors.Is(err, body.closeErr) || body.closes != 1 {
		t.Fatal("404 close failure lost")
	}
}

func TestContainerLifecycleCoreSourceAndContext(t *testing.T) {
	var calls atomic.Int32
	a, client := lifecycleAPI(lifecycleStatic(201, "", &calls))
	optionCause, cancelCause := errors.New("option failed"), errors.New("option canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	_, err := a.CreateContainer(ctx, "name", func(*CreateContainerOpts) error {
		client.Microversion = "changed"
		cancel(cancelCause)
		return optionCause
	})
	for _, cause := range []error{optionCause, cancelCause, context.Canceled, resource.ErrInvalidOption} {
		if !errors.Is(err, cause) {
			t.Fatalf("lost%v: %v", cause, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("retargeted callback sent HTTP")
	}
	body := &lifecycleBody{data: "actual"}
	a, client = lifecycleAPI(lifecycleTransport(func(r *http.Request) (*http.Response, error) { return lifecycleWire(r, 404, body), nil }))
	body.onClose = func() { client.ResourceBase = client.Endpoint }
	r, err := a.DeleteContainer(context.Background(), "name")
	lifecycleProof(t, r, err, 404, "actual")
	if !errors.Is(err, resource.ErrInvalidOption) || body.closes != 1 {
		t.Fatalf("postresponse source%v", err)
	}
}

func TestContainerLifecycleCoreNativePolicy(t *testing.T) {
	var calls atomic.Int32
	a, client := lifecycleAPI(lifecycleStatic(404, "physical", &calls))
	hooks := 0
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		hooks++
		return errors.New("should not run")
	}
	r, err := a.DeleteContainer(context.Background(), "name")
	if err != nil || r == nil || !r.IgnoredMissing || hooks != 0 {
		t.Fatalf("handled404 %+v %v hooks%d", r, err, hooks)
	}
	transportCause := gophercloud.ErrUnexpectedResponseCode{Actual: 404}
	a, _ = lifecycleAPI(lifecycleTransport(func(*http.Request) (*http.Response, error) { return nil, transportCause }))
	r, err = a.DeleteContainer(context.Background(), "name")
	if r != nil || err == nil || !errors.As(err, &transportCause) {
		t.Fatalf("transport404 %+v %v", r, err)
	}
	calls.Store(0)
	a, client = lifecycleAPI(lifecycleTransport(func(r *http.Request) (*http.Response, error) {
		n := calls.Add(1)
		code := 503
		if n > 1 {
			code = 202
		}
		return lifecycleWire(r, code, io.NopCloser(strings.NewReader("native"))), nil
	}))
	client.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		opts.OkCodes = []int{202}
		return nil
	}
	r, err = a.DeleteContainer(context.Background(), "name")
	var native gophercloud.ErrUnexpectedResponseCode
	if r != nil || !errors.As(err, &native) || native.Actual != 202 || len(native.Expected) != 2 || native.Expected[0] != 204 || native.Expected[1] != 404 || calls.Load() != 2 {
		t.Fatalf("status guard %+v %v requests%d", r, err, calls.Load())
	}
	calls.Store(0)
	a, client = lifecycleAPI(lifecycleStatic(503, "native", &calls))
	client.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		opts.KeepResponseBody = false
		return nil
	}
	r, err = a.CreateContainer(context.Background(), "name")
	if r != nil || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 1 {
		t.Fatalf("body guard %+v %v", r, err)
	}
}
