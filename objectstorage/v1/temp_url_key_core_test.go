package v1

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	nativeobjects "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects"
)

type tempURLKeyTestTransport func(*http.Request) (*http.Response, error)

func (f tempURLKeyTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type tempURLKeyTestBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *tempURLKeyTestBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type tempURLKeyTestReader func([]byte) (int, error)

func (f tempURLKeyTestReader) Read(p []byte) (int, error) { return f(p) }
func tempURLKeyTestClient() *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{TokenID: "token"}
	p.UseTokenLock()
	return &gophercloud.ServiceClient{ProviderClient: p, Endpoint: "http://cloud.invalid/reverse%20proxy/v1/AUTH_x/", ResourceBase: "http://cloud.invalid/data%25/v1/AUTH_x/", Type: "object-store"}
}
func tempURLKeyTestWire(r *http.Request, code int, h http.Header, body io.ReadCloser) *http.Response {
	return &http.Response{Request: r, StatusCode: code, Header: h, Body: body}
}
func tempURLKeyTestProof(t *testing.T, err error, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != body || proof.Header.Get("X-Proof") != "kept" {
		t.Fatalf("lost accepted evidence: %v %#v", err, proof)
	}
	return proof
}

func TestTempURLKeyCoreSelectionAndProof(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		container                  bool
		containerKeys, accountKeys map[string]string
		want                       string
		secondary, fromContainer   bool
		calls                      int
	}{
		{"account primary", false, nil, map[string]string{"Temp-URL-Key": "primary%20책"}, "primary%20책", false, false, 1},
		{"account secondary", false, nil, map[string]string{"Temp-URL-Key": "primary", "Temp-URL-Key-2": "secondary"}, "secondary", true, false, 1},
		{"container primary", true, map[string]string{"Temp-URL-Key": "container"}, nil, "container", false, true, 1},
		{"container secondary literal", true, map[string]string{"Temp-URL-Key": "primary", "Temp-URL-Key-2": "  책%20\tkey  "}, nil, "  책%20\tkey  ", true, true, 1},
		{"empty container fallback", true, map[string]string{"Temp-URL-Key": "", "Temp-URL-Key-2": ""}, map[string]string{"Temp-URL-Key": "primary", "Temp-URL-Key-2": "secondary"}, "secondary", true, false, 2},
		{"absent container fallback", true, nil, map[string]string{"Temp-URL-Key": "primary"}, "primary", false, false, 2},
		{"no usable key", true, map[string]string{"Temp-URL-Key": ""}, map[string]string{"Temp-URL-Key": ""}, "", false, false, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := tempURLKeyTestClient()
			calls := 0
			name := "백업 %2F?#"
			if !tc.container {
				c.ResourceBase = "not a URL: unused"
			}
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				phase, prefix, keys := "account", "X-Account-Meta-", tc.accountKeys
				wantPath := "/reverse%20proxy/v1/AUTH_x/"
				if tc.container && calls == 1 {
					phase, prefix, keys = "container", "X-Container-Meta-", tc.containerKeys
					wantPath = "/data%25/v1/AUTH_x/" + url.PathEscape(name)
				}
				if r.Method != "HEAD" || r.URL.EscapedPath() != wantPath || r.URL.RawQuery != "" || r.Body != nil {
					t.Fatalf("wrong read %s %s", r.Method, r.URL)
				}
				h := http.Header{"X-Proof": {"kept"}}
				for k, v := range keys {
					h[prefix+k] = []string{v}
				}
				return tempURLKeyTestWire(r, 204, h, io.NopCloser(strings.NewReader(phase+" raw"))), nil
			})
			var options []GetTempURLKeyOption
			if tc.container {
				options = []GetTempURLKeyOption{WithGetTempURLKeyContainer(name)}
			}
			result, err := New(c).GetTempURLKey(context.Background(), options...)
			if err != nil || result == nil || string(result.Key) != tc.want || result.Secondary != tc.secondary || result.FromContainer != tc.fromContainer || calls != tc.calls {
				t.Fatalf("wrong selection %#v %v calls=%d", result, err, calls)
			}
			if tc.want == "" && result.Key != nil {
				t.Fatal("empty selected key must remain nil")
			}
			if tc.container && (result.Container == nil || result.Container.StatusCode != 204 || string(result.Container.Body) != "container raw" || result.Container.Metadata == nil) {
				t.Fatal("lost container proof", result)
			}
			if !tc.fromContainer && (result.Account == nil || result.Account.StatusCode != 204 || string(result.Account.Body) != "account raw" || result.Account.Metadata == nil) {
				t.Fatal("lost account proof", result)
			}
			if tc.fromContainer && result.Account != nil {
				t.Fatal("fabricated account read", result)
			}
			if tc.name == "empty container fallback" {
				value, present := result.Container.Metadata.Values["temp-url-key-2"]
				if !present || value != "" {
					t.Fatal("explicit empty metadata lost")
				}
			}
			if result.Account != nil {
				result.Account.Header.Set("X-Proof", "changed")
				if result.Account.Metadata.Values["temp-url-key"] != tc.accountKeys["Temp-URL-Key"] {
					t.Fatal("header and projection alias")
				}
			}
			if len(result.Key) > 0 {
				result.Key[0] = '!'
				var value string
				if tc.fromContainer {
					value = result.Container.Metadata.Values["temp-url-key"]
					if tc.secondary {
						value = result.Container.Metadata.Values["temp-url-key-2"]
					}
				} else {
					value = result.Account.Metadata.Values["temp-url-key"]
					if tc.secondary {
						value = result.Account.Metadata.Values["temp-url-key-2"]
					}
				}
				if value != tc.want {
					t.Fatal("key bytes alias metadata")
				}
			}
		})
	}
}

func TestTempURLKeyCoreStopsOnErrors(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		second, native, transport bool
		bad                       http.Header
	}{
		{name: "first HTTP404", native: true}, {name: "first transport404", transport: true},
		{name: "malformed unrelated counter", bad: http.Header{"X-Container-Object-Count": {"bad"}}},
		{name: "duplicate custom aliases", bad: http.Header{"X-Container-Meta-Note": {"a"}, "x-container-meta-note": {"b"}}},
		{name: "nonASCII original suffix", bad: http.Header{"X-Container-Meta-K": {"value"}}},
		{name: "later HTTP404", second: true, native: true},
		{name: "later projection failure", second: true, bad: http.Header{"X-Account-Object-Count": {"9223372036854775808"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := tempURLKeyTestClient()
			calls := 0
			fault := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.second && calls == 1 {
					return tempURLKeyTestWire(r, 204, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("container"))), nil
				}
				if tc.transport {
					return nil, fault
				}
				code := 204
				if tc.native {
					code = 404
				}
				h := tc.bad.Clone()
				if h == nil {
					h = make(http.Header)
				}
				h.Set("X-Proof", "kept")
				return tempURLKeyTestWire(r, code, h, io.NopCloser(strings.NewReader("current"))), nil
			})
			result, err := New(c).GetTempURLKey(context.Background(), WithGetTempURLKeyContainer("box"))
			wantCalls := 1
			if tc.second {
				wantCalls = 2
			}
			if err == nil || calls != wantCalls {
				t.Fatalf("error did not stop fallback %#v %v calls=%d", result, err, calls)
			}
			if result != nil && (result.Key != nil || result.Secondary || result.FromContainer) {
				t.Fatal("selected key on failure", result)
			}
			if tc.native || tc.transport {
				if !tc.second && result != nil {
					t.Fatal("native failure fabricated proof", result)
				}
				if tc.second && (result == nil || result.Container == nil || result.Account != nil || string(result.Container.Body) != "container") {
					t.Fatal("earlier proof lost", result)
				}
				if tc.transport && !errors.Is(err, fault) {
					t.Fatal("transport cause lost", err)
				}
				if tc.native && !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatal("native HTTP cause lost", err)
				}
			} else {
				tempURLKeyTestProof(t, err, "current")
				if result == nil {
					t.Fatal("accepted proof missing")
				}
				if tc.second {
					if result.Account == nil || result.Account.Metadata != nil || result.Container.Metadata == nil {
						t.Fatal("projection was not atomic", result)
					}
				} else if result.Container == nil || result.Container.Metadata != nil || result.Account != nil {
					t.Fatal("projection was not atomic", result)
				}
			}
		})
	}
}

func TestTempURLKeyCorePreflightAndAcceptedSourceGuard(t *testing.T) {
	t.Run("preflight", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			change func(*gophercloud.ServiceClient)
		}{
			{"wrong type", func(c *gophercloud.ServiceClient) { c.Type = "image" }}, {"provider nil", func(c *gophercloud.ServiceClient) { c.ProviderClient = nil }},
			{"endpoint", func(c *gophercloud.ServiceClient) { c.Endpoint = "http://cloud.invalid/v1/AUTH_x/?x=1" }}, {"source Unicode token", func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"X-K": "bad"} }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				c := tempURLKeyTestClient()
				tc.change(c)
				callbacks := 0
				result, err := New(c).GetTempURLKey(context.Background(), func(*GetTempURLKeyOpts) error { callbacks++; return nil })
				if result != nil || err == nil || callbacks != 0 {
					t.Fatal(result, err, callbacks)
				}
			})
		}
		callbacks := 0
		option := func(*GetTempURLKeyOpts) error { callbacks++; return nil }
		var nilService *Service
		for _, s := range []*Service{nilService, &Service{}, New(tempURLKeyTestClient())} {
			ctx := context.Background()
			if s != nil && s.client != nil {
				ctx = nil
			}
			if result, err := s.GetTempURLKey(ctx, option); result != nil || err == nil || callbacks != 0 {
				t.Fatal(result, err, callbacks)
			}
		}
		for _, name := range []string{".", "..", "slash/name", "back\\name", "bad\nname", string([]byte{255})} {
			c := tempURLKeyTestClient()
			calls := 0
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			if result, err := New(c).GetTempURLKey(context.Background(), WithGetTempURLKeyContainer(name)); result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(name, result, err, calls)
			}
		}
		c := tempURLKeyTestClient()
		c.ResourceBase = "http://foreign.invalid/v1/AUTH_x/"
		if result, err := New(c).GetTempURLKey(context.Background(), WithGetTempURLKeyContainer("box")); result != nil || err == nil {
			t.Fatal("foreign base accepted", result, err)
		}
	})
	t.Run("callback context and retarget", func(t *testing.T) {
		c := tempURLKeyTestClient()
		s := New(c)
		cause, optionErr := errors.New("cancel"), errors.New("option")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		result, err := s.GetTempURLKey(ctx, func(*GetTempURLKeyOpts) error { s.Accounts = nil; cancel(cause); return optionErr })
		if result != nil {
			t.Fatal("invented proof", result)
		}
		for _, want := range []error{optionErr, cause, context.Canceled, resource.ErrInvalidOption} {
			if !errors.Is(err, want) {
				t.Fatalf("lost %v: %v", want, err)
			}
		}
	})
	for _, tc := range []struct {
		name   string
		change func(*Service, *gophercloud.ServiceClient)
	}{
		{"Endpoint", func(_ *Service, c *gophercloud.ServiceClient) { c.Endpoint += "changed/" }}, {"ResourceBase", func(_ *Service, c *gophercloud.ServiceClient) { c.ResourceBase += "changed/" }},
		{"Type", func(_ *Service, c *gophercloud.ServiceClient) { c.Type = "image" }}, {"Microversion", func(_ *Service, c *gophercloud.ServiceClient) { c.Microversion = "2.0" }},
		{"Provider", func(_ *Service, c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }}, {"client", func(s *Service, c *gophercloud.ServiceClient) { copy := *c; s.client = &copy }},
		{"Accounts", func(s *Service, _ *gophercloud.ServiceClient) { s.Accounts = nil }}, {"Containers", func(s *Service, _ *gophercloud.ServiceClient) { s.Containers = nil }},
		{"Objects", func(s *Service, _ *gophercloud.ServiceClient) { s.Objects = nil }}, {"Swauth", func(s *Service, _ *gophercloud.ServiceClient) { s.Swauth = nil }},
		{"invalid live header", func(_ *Service, c *gophercloud.ServiceClient) {
			c.MoreHeaders = map[string]string{"X-Account-Meta-Temp-URL-Key": "changed"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := tempURLKeyTestClient()
			s := New(c)
			calls := 0
			b := &tempURLKeyTestBody{Reader: strings.NewReader("observed"), onClose: func() { tc.change(s, c) }}
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return tempURLKeyTestWire(r, 204, http.Header{"X-Proof": {"kept"}, "X-Container-Meta-Temp-URL-Key-2": {"usable"}}, b), nil
			})
			result, err := s.GetTempURLKey(context.Background(), WithGetTempURLKeyContainer("box"))
			if result == nil || result.Container == nil || result.Container.Metadata != nil || result.Account != nil || result.Key != nil || result.Secondary || result.FromContainer || calls != 1 || b.closes.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("source change bypassed guard %#v %v calls=%d", result, err, calls)
			}
			proof := tempURLKeyTestProof(t, err, "observed")
			result.Container.Body[0] = '!'
			result.Container.Header.Set("X-Proof", "changed")
			if string(proof.Body) != "observed" || proof.Header.Get("X-Proof") != "kept" {
				t.Fatal("accepted evidence aliases")
			}
		})
	}
	t.Run("read close cancel and retarget joined", func(t *testing.T) {
		c := tempURLKeyTestClient()
		readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("cancel")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		b := &tempURLKeyTestBody{Reader: tempURLKeyTestReader(func(p []byte) (int, error) { return copy(p, "x"), readErr }), closeErr: closeErr, onClose: func() { c.Endpoint += "changed/"; cancel(cause) }}
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			return tempURLKeyTestWire(r, 204, http.Header{"X-Proof": {"kept"}}, b), nil
		})
		result, err := New(c).GetTempURLKey(ctx, WithGetTempURLKeyContainer("box"))
		if result == nil || result.Container == nil || result.Container.Metadata != nil || result.Account != nil || b.closes.Load() != 1 {
			t.Fatal(result, err)
		}
		tempURLKeyTestProof(t, err, "x")
		for _, want := range []error{readErr, closeErr, cause, context.Canceled, resource.ErrInvalidOption} {
			if !errors.Is(err, want) {
				t.Fatalf("lost %v: %v", want, err)
			}
		}
	})
}

func TestTempURLKeyCoreCapturedHeadersAndNativeSigner(t *testing.T) {
	t.Run("same header snapshot with live auth", func(t *testing.T) {
		c := tempURLKeyTestClient()
		c.MoreHeaders = map[string]string{"X-Source": "captured"}
		calls := 0
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			token := "token"
			if calls == 2 {
				token = "next-token"
			}
			if r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Call") != "owned" || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Auth-Token") != token {
				t.Fatal("snapshot/auth changed", r.Header)
			}
			h := http.Header{"X-Proof": {"kept"}}
			if calls == 2 {
				h["X-Account-Meta-Temp-URL-Key-2"] = []string{"key"}
			}
			b := &tempURLKeyTestBody{Reader: strings.NewReader("raw")}
			if calls == 1 {
				b.onClose = func() { c.MoreHeaders["X-Source"] = "new valid value"; c.SetToken("next-token") }
			}
			return tempURLKeyTestWire(r, 204, h, b), nil
		})
		result, err := New(c).GetTempURLKey(context.Background(), WithGetTempURLKeyContainer("box"), WithGetTempURLKeyHeader("X-Call", "owned"), WithGetTempURLKeyNewest(false))
		if err != nil || result == nil || string(result.Key) != "key" || !result.Secondary || result.FromContainer || calls != 2 || c.MoreHeaders["X-Source"] != "new valid value" {
			t.Fatal(result, err, calls, c.MoreHeaders)
		}
	})
	t.Run("native primary discovery remains separate", func(t *testing.T) {
		c := tempURLKeyTestClient()
		calls := 0
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			return tempURLKeyTestWire(r, 204, http.Header{"X-Container-Meta-Temp-Url-Key": {"primary"}, "X-Container-Meta-Temp-Url-Key-2": {"secondary"}}, io.NopCloser(strings.NewReader(""))), nil
		})
		s := New(c)
		opts := nativeobjects.CreateTempURLOpts{Method: nativeobjects.GET, TTL: 60, Timestamp: time.Unix(123456, 0)}
		actual, err := s.Objects.CreateTempURL(context.Background(), "box", "object", opts)
		if err != nil || calls != 1 {
			t.Fatal(actual, err, calls)
		}
		opts.TempURLKey = "primary"
		expected, err := s.Objects.CreateTempURL(context.Background(), "box", "object", opts)
		if err != nil || actual != expected || calls != 1 {
			t.Fatal("native primary policy changed", actual, expected, err, calls)
		}
		result, err := s.GetTempURLKey(context.Background(), WithGetTempURLKeyContainer("box"))
		if err != nil || string(result.Key) != "secondary" || !result.Secondary || !result.FromContainer || calls != 2 {
			t.Fatal("owned discovery differs incorrectly", result, err, calls)
		}
	})
}
