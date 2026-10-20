package v1_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const tempKeyAccount = "https://swift.invalid/reverse/v1/AUTH_account/"
const tempKeyBase = "https://swift.invalid/reverse/a%20b/v1/AUTH_account/"
const tempKeyName = "한글 space:%2F?#"

type tempKeyTransport func(*http.Request) (*http.Response, error)

func (f tempKeyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	if response != nil && response.Request == nil {
		response.Request = r
	}
	return response, err
}

type tempKeyBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *tempKeyBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type tempKeyReader func([]byte) (int, error)

func (f tempKeyReader) Read(p []byte) (int, error) { return f(p) }

func tempKeyClient(f tempKeyTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("token-one")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "object-store", Endpoint: tempKeyAccount, ResourceBase: tempKeyBase}
}

func tempKeyResponse(status int, body *tempKeyBody, headers http.Header) *http.Response {
	h := headers.Clone()
	if h == nil {
		h = make(http.Header)
	}
	h.Set("Content-Type", "application/json")
	h.Set("X-Trans-Id", "actual-key-phase")
	return &http.Response{StatusCode: status, Header: h, Body: body}
}

func tempKeyWire(status int, body string, headers http.Header) *http.Response {
	return tempKeyResponse(status, &tempKeyBody{Reader: strings.NewReader(body)}, headers)
}

func tempKeyNoBody(t *testing.T, r *http.Request) {
	t.Helper()
	if r.URL.RawQuery != "" {
		t.Errorf("query=%q", r.URL.RawQuery)
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Errorf("request body=%q err=%v", body, err)
		}
	}
}

func tempKeyProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Trans-Id") != "actual-key-phase" {
		t.Fatalf("proof=%+v err=%v", proof, err)
	}
	return proof
}

func tempKeySelected(t *testing.T, result *v1.TempURLKeyResult, err error, key string, from, secondary bool) {
	t.Helper()
	if err != nil || result == nil || string(result.Key) != key || result.FromContainer != from || result.Secondary != secondary {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if key == "" && result.Key != nil {
		t.Fatalf("no usable key=%q", result.Key)
	}
}

func TestTempURLKeyContractsWireAndSelection(t *testing.T) {
	t.Run("account setter owns one selected header and only POST204", func(t *testing.T) {
		for _, tc := range []struct {
			key       string
			secondary bool
		}{{"primary", false}, {"secondary", true}, {"", false}, {" literal key ", true}} {
			calls := 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				tempKeyNoBody(t, r)
				name := "X-Account-Meta-Temp-URL-Key"
				if tc.secondary {
					name += "-2"
				}
				other := "X-Account-Meta-Temp-URL-Key-2"
				if tc.secondary {
					other = "X-Account-Meta-Temp-URL-Key"
				}
				if r.Method != "POST" || r.URL.String() != tempKeyAccount || !reflect.DeepEqual(r.Header.Values(name), []string{tc.key}) || len(r.Header.Values(other)) != 0 {
					t.Errorf("wire=%s %s headers=%v", r.Method, r.URL, r.Header)
				}
				return tempKeyWire(204, "opaque acknowledgement", nil), nil
			})
			result, err := accounts.New(client).SetTempURLKey(context.Background(), tc.key, accounts.WithSetTempURLKeySecondary(tc.secondary))
			if err != nil || result == nil || result.StatusCode != 204 || string(result.Body) != "opaque acknowledgement" || calls != 1 {
				t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
			}
		}
	})
	t.Run("container setter literal encoded target and separate prefix", func(t *testing.T) {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			tempKeyNoBody(t, r)
			want := "/reverse/a%20b/v1/AUTH_account/" + url.PathEscape(tempKeyName)
			if r.Method != "POST" || r.URL.EscapedPath() != want || r.RequestURI != want || r.Header.Get("X-Container-Meta-Temp-URL-Key-2") != "literal key" {
				t.Errorf("method=%s URI=%q headers=%v", r.Method, r.RequestURI, r.Header)
			}
			for key := range r.Header {
				if strings.HasPrefix(strings.ToLower(key), "x-account-meta-") {
					t.Errorf("account prefix=%s", key)
				}
			}
			w.Header().Set("X-Trans-Id", "real-wire")
			w.WriteHeader(204)
		}))
		defer server.Close()
		client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{HTTPClient: *server.Client()}, Type: "object-store", Endpoint: server.URL + "/unused/", ResourceBase: server.URL + "/reverse/a%20b/v1/AUTH_account/"}
		result, err := containers.New(client).SetTempURLKey(context.Background(), tempKeyName, " literal key ", containers.WithSetTempURLKeySecondary(true))
		if err != nil || result == nil || result.StatusCode != 204 || result.Header.Get("X-Trans-Id") != "real-wire" || requests != 1 {
			t.Fatalf("result=%+v err=%v requests=%d", result, err, requests)
		}
	})
	t.Run("fresh secondary first within container then account", func(t *testing.T) {
		for _, tc := range []struct {
			name, c1, c2, a1, a2, want string
			from, secondary            bool
			calls                      int
		}{
			{"container", "cp", "cs", "ap", "as", "cs", true, true, 1},
			{"container", "cp", "", "ap", "as", "cp", true, false, 1},
			{"container", "", "", "ap", "as", "as", false, true, 2},
			{"container", "", "", "ap", "", "ap", false, false, 2},
			{"", "unused", "unused", "ap", "as", "as", false, true, 1},
			{"container", "", "", "", "", "", false, false, 2},
		} {
			calls := 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				tempKeyNoBody(t, r)
				h := make(http.Header)
				if r.URL.String() == tempKeyAccount {
					h.Set("X-Account-Meta-Temp-URL-Key", tc.a1)
					h.Set("X-Account-Meta-Temp-URL-Key-2", tc.a2)
				} else {
					if r.URL.String() != tempKeyBase+tc.name {
						t.Errorf("container route=%s", r.URL)
					}
					h.Set("X-Container-Meta-Temp-URL-Key", tc.c1)
					h.Set("X-Container-Meta-Temp-URL-Key-2", tc.c2)
				}
				if r.Method != "HEAD" {
					t.Errorf("method=%s", r.Method)
				}
				return tempKeyWire(204, "raw phase", h), nil
			})
			result, err := v1.New(client).GetTempURLKey(context.Background(), v1.WithGetTempURLKeyContainer(tc.name))
			tempKeySelected(t, result, err, tc.want, tc.from, tc.secondary)
			if calls != tc.calls || (tc.name != "" && (result.Container == nil || result.Container.StatusCode != 204)) || (!tc.from && result.Account == nil) {
				t.Fatalf("proof=%+v calls=%d want=%d", result, calls, tc.calls)
			}
		}
	})
	t.Run("strict actual HEAD and POST204 reject native broad successes", func(t *testing.T) {
		for _, status := range []int{200, 201, 202, 404} {
			for _, op := range []string{"get", "account", "container"} {
				calls := 0
				client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
					calls++
					return tempKeyWire(status, "rejected", nil), nil
				})
				var err error
				switch op {
				case "get":
					result, e := v1.New(client).GetTempURLKey(context.Background(), v1.WithGetTempURLKeyContainer("c"))
					err = e
					if result != nil {
						t.Fatalf("unexpected getter=%+v", result)
					}
				case "account":
					result, e := accounts.New(client).SetTempURLKey(context.Background(), "k")
					err = e
					if result != nil {
						t.Fatalf("unexpected setter=%+v", result)
					}
				case "container":
					result, e := containers.New(client).SetTempURLKey(context.Background(), "c", "k")
					err = e
					if result != nil {
						t.Fatalf("unexpected setter=%+v", result)
					}
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{204}) || calls != 1 {
					t.Fatalf("op=%s err=%v native=%+v calls=%d", op, err, native, calls)
				}
			}
		}
	})
}

func TestTempURLKeyContractsPresenceAndOptions(t *testing.T) {
	t.Run("literal empty whitespace unicode and absent raw presence", func(t *testing.T) {
		for _, value := range []string{"", " ", "\t", "한글: key"} {
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				return tempKeyWire(204, "", http.Header{"X-Account-Meta-Temp-Url-Key": {value}}), nil
			})
			result, err := v1.New(client).GetTempURLKey(context.Background())
			tempKeySelected(t, result, err, value, false, false)
			if result.Account.Metadata == nil || result.Account.Metadata.Values["temp-url-key"] != value {
				t.Fatalf("observed values=%+v", result.Account)
			}
			if _, exists := result.Account.Metadata.Values["temp-url-key"]; !exists {
				t.Fatal("explicit empty/other value presence lost")
			}
		}
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) { return tempKeyWire(204, "", nil), nil })
		result, err := v1.New(client).GetTempURLKey(context.Background())
		tempKeySelected(t, result, err, "", false, false)
		if _, exists := result.Account.Metadata.Values["temp-url-key"]; exists {
			t.Fatal("absent key synthesized")
		}
	})
	t.Run("setter full replacement header override and factory snapshots", func(t *testing.T) {
		for _, account := range []bool{true, false} {
			factoryHeaders := map[string]string{"x-call": "factory"}
			calls := 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				prefix := "X-Container-Meta-Temp-URL-Key"
				if account {
					prefix = "X-Account-Meta-Temp-URL-Key"
				}
				if r.Header.Get(prefix) != "key" || len(r.Header.Values(prefix+"-2")) != 0 || r.Header.Get("X-Call") != "final" || r.Header.Get("X-Second") != "snapshot" || r.Header.Get("X-Discarded") != "" {
					t.Errorf("final headers=%v", r.Header)
				}
				return tempKeyWire(204, "ack", nil), nil
			})
			if account {
				full := accounts.WithSetTempURLKeyOpts(accounts.SetTempURLKeyOpts{Headers: factoryHeaders})
				extra := map[string]string{"X-Second": "snapshot"}
				headers := accounts.WithSetTempURLKeyHeaders(extra)
				factoryHeaders["x-call"] = "mutated"
				extra["X-Second"] = "mutated"
				var retained *accounts.SetTempURLKeyOpts
				result, err := accounts.New(client).SetTempURLKey(context.Background(), "key", accounts.WithSetTempURLKeySecondary(true), accounts.WithSetTempURLKeyHeader("X-Discarded", "yes"), full, headers, accounts.WithSetTempURLKeyHeader("X-Call", "final"), func(cfg *accounts.SetTempURLKeyOpts) error { retained = cfg; return nil }, func(cfg *accounts.SetTempURLKeyOpts) error { retained.Headers["X-Call"] = "late mutation"; return nil })
				if err != nil || result == nil {
					t.Fatalf("result=%+v err=%v", result, err)
				}
			} else {
				full := containers.WithSetTempURLKeyOpts(containers.SetTempURLKeyOpts{Headers: factoryHeaders})
				extra := map[string]string{"X-Second": "snapshot"}
				headers := containers.WithSetTempURLKeyHeaders(extra)
				factoryHeaders["x-call"] = "mutated"
				extra["X-Second"] = "mutated"
				var retained *containers.SetTempURLKeyOpts
				result, err := containers.New(client).SetTempURLKey(context.Background(), "c", "key", containers.WithSetTempURLKeySecondary(true), containers.WithSetTempURLKeyHeader("X-Discarded", "yes"), full, headers, containers.WithSetTempURLKeyHeader("X-Call", "final"), func(cfg *containers.SetTempURLKeyOpts) error { retained = cfg; return nil }, func(cfg *containers.SetTempURLKeyOpts) error {
					retained.Headers["X-Call"] = "late mutation"
					return nil
				})
				if err != nil || result == nil {
					t.Fatalf("result=%+v err=%v", result, err)
				}
			}
			if calls != 1 {
				t.Fatalf("requests=%d", calls)
			}
		}
	})
	t.Run("getter full configuration nil empty newest and copied pointers", func(t *testing.T) {
		for _, tc := range []struct {
			newest *bool
			clear  bool
			want   []string
		}{{nil, false, nil}, {new(bool), false, []string{"false"}}, {func() *bool { b := true; return &b }(), false, []string{"true"}}, {new(bool), true, nil}} {
			headers := map[string]string{"x-call": "factory"}
			full := v1.WithGetTempURLKeyOpts(v1.GetTempURLKeyOpts{Container: "c", Headers: headers, Newest: tc.newest})
			if tc.newest != nil {
				*tc.newest = !*tc.newest
			}
			headers["x-call"] = "mutated"
			extra := map[string]string{"X-Extra": "snapshot"}
			overlay := v1.WithGetTempURLKeyHeaders(extra)
			extra["X-Extra"] = "mutated"
			calls := 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if !reflect.DeepEqual(r.Header.Values("X-Newest"), tc.want) || r.Header.Get("X-Call") != "final" || r.Header.Get("X-Extra") != "snapshot" {
					t.Errorf("headers=%v want newest=%v", r.Header, tc.want)
				}
				prefix := "X-Container-Meta-Temp-URL-Key"
				if r.URL.String() == tempKeyAccount {
					prefix = "X-Account-Meta-Temp-URL-Key"
				}
				value := ""
				if prefix == "X-Account-Meta-Temp-URL-Key" {
					value = "account"
				}
				return tempKeyWire(204, "", http.Header{http.CanonicalHeaderKey(prefix): {value}}), nil
			})
			opts := []v1.GetTempURLKeyOption{v1.WithGetTempURLKeyNewest(true), full, overlay, v1.WithGetTempURLKeyHeader("X-Call", "final")}
			var retained *v1.GetTempURLKeyOpts
			opts = append(opts, func(cfg *v1.GetTempURLKeyOpts) error { retained = cfg; return nil }, func(cfg *v1.GetTempURLKeyOpts) error {
				retained.Headers["X-Call"] = "late mutation"
				if retained.Newest != nil {
					*retained.Newest = !*retained.Newest
				}
				return nil
			})
			if tc.clear {
				opts = append(opts, v1.WithoutGetTempURLKeyNewest())
			}
			result, err := v1.New(client).GetTempURLKey(context.Background(), opts...)
			tempKeySelected(t, result, err, "account", false, false)
			if calls != 2 {
				t.Fatalf("requests=%d", calls)
			}
		}
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != tempKeyAccount {
				t.Errorf("unexpected target=%s", r.URL)
			}
			return tempKeyWire(204, "", nil), nil
		})
		client.ResourceBase = "not a usable container URL"
		result, err := v1.New(client).GetTempURLKey(context.Background(), v1.WithGetTempURLKeyContainer("prior"), v1.WithGetTempURLKeyContainer(""))
		tempKeySelected(t, result, err, "", false, false)
	})
	t.Run("result headers key bytes and concurrent option reuse independent", func(t *testing.T) {
		raw := http.Header{"X-Account-Meta-Temp-Url-Key-2": {"selected"}, "X-Account-Meta-Other": {"observed"}}
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) { return tempKeyWire(204, "opaque", raw), nil })
		result, err := v1.New(client).GetTempURLKey(context.Background())
		tempKeySelected(t, result, err, "selected", false, true)
		raw.Set("X-Account-Meta-Temp-Url-Key-2", "changed")
		result.Key[0] = 'X'
		result.Account.Metadata.Values["other"] = "changed"
		if result.Account.Header.Get("X-Account-Meta-Temp-Url-Key-2") != "selected" || result.Account.Header.Get("X-Account-Meta-Other") != "observed" || result.Account.Metadata.Values["temp-url-key-2"] != "selected" {
			t.Fatal("independent owned evidence aliased")
		}
		options := []v1.GetTempURLKeyOption{v1.WithGetTempURLKeyHeaders(map[string]string{"X-Reuse": "safe"}), v1.WithGetTempURLKeyNewest(false)}
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c := tempKeyClient(func(r *http.Request) (*http.Response, error) {
					if r.Header.Get("X-Reuse") != "safe" || r.Header.Get("X-Newest") != "false" {
						t.Errorf("shared headers=%v", r.Header)
					}
					return tempKeyWire(204, "", http.Header{"X-Account-Meta-Temp-Url-Key": {"key"}}), nil
				})
				got, e := v1.New(c).GetTempURLKey(context.Background(), options...)
				if e != nil || got == nil || string(got.Key) != "key" || got.FromContainer || got.Secondary {
					t.Errorf("concurrent result=%+v err=%v", got, e)
					return
				}
			}()
		}
		wg.Wait()
	})
}

func TestTempURLKeyContractsPreflightAndSource(t *testing.T) {
	t.Run("context source and literal identity before callback or HTTP", func(t *testing.T) {
		for _, service := range []*v1.Service{nil, {}, v1.New(nil)} {
			callbacks := 0
			result, err := service.GetTempURLKey(context.Background(), func(cfg *v1.GetTempURLKeyOpts) error { callbacks++; return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
				t.Fatalf("nil source result=%+v err=%v callbacks=%d", result, err, callbacks)
			}
		}
		cause := errors.New("caller cancelled")
		cancelled, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cancel(cause)
		for _, ctx := range []context.Context{nil, cancelled} {
			calls, callbacks := 0, 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) { calls++; return tempKeyWire(204, "", nil), nil })
			result, err := v1.New(client).GetTempURLKey(ctx, func(cfg *v1.GetTempURLKeyOpts) error { callbacks++; return nil })
			if result != nil || err == nil || calls != 0 || callbacks != 0 || (ctx != nil && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause))) {
				t.Fatalf("result=%+v err=%v callbacks=%d calls=%d", result, err, callbacks, calls)
			}
			ack, err := accounts.New(client).SetTempURLKey(ctx, "key", func(cfg *accounts.SetTempURLKeyOpts) error { callbacks++; return nil })
			if ack != nil || err == nil || calls != 0 || callbacks != 0 {
				t.Fatalf("setter preflight ack=%+v err=%v", ack, err)
			}
		}
		for _, mutate := range []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.ProviderClient = nil }, func(c *gophercloud.ServiceClient) { c.Type = "compute" }, func(c *gophercloud.ServiceClient) { c.Endpoint = "https://swift.invalid/account?retarget=1" }, func(c *gophercloud.ServiceClient) { c.Microversion = "bad\rvalue" }} {
			calls, callbacks := 0, 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) { calls++; return tempKeyWire(204, "", nil), nil })
			mutate(client)
			result, err := v1.New(client).GetTempURLKey(context.Background(), func(cfg *v1.GetTempURLKeyOpts) error { callbacks++; return nil })
			if result != nil || err == nil || calls != 0 || callbacks != 0 {
				t.Fatalf("source preflight result=%+v err=%v calls=%d callbacks=%d", result, err, calls, callbacks)
			}
		}
		for _, name := range []string{".", "..", "a/b", "a\\b", "\x00", "\x7f", string([]byte{0xff})} {
			calls := 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) { calls++; return tempKeyWire(204, "", nil), nil })
			result, err := v1.New(client).GetTempURLKey(context.Background(), v1.WithGetTempURLKeyContainer(name))
			ack, e := containers.New(client).SetTempURLKey(context.Background(), name, "key")
			if result != nil || ack != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(e, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("name=%q result=%+v ack=%+v errors=%v/%v calls=%d", name, result, ack, err, e, calls)
			}
		}
		for _, bad := range []string{"\r", "\n", "\x00", "\x7f", string([]byte{0xff})} {
			calls, callbacks := 0, 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) { calls++; return tempKeyWire(204, "", nil), nil })
			ack, err := containers.New(client).SetTempURLKey(context.Background(), "c", bad, func(cfg *containers.SetTempURLKeyOpts) error { callbacks++; return nil })
			if ack != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
				t.Fatalf("key=%q ack=%+v err=%v calls=%d callbacks=%d", bad, ack, err, calls, callbacks)
			}
		}
	})
	t.Run("joint owned prefixes original tokens aliases and option failures", func(t *testing.T) {
		for _, headers := range []map[string]string{
			{"X-Account-Meta-Temp-URL-Key": "override"}, {"X-Container-Meta-Other": "override"}, {"X-Remove-Account-Meta-Key": "remove"}, {"X-Remove-Container-Meta-Key": "remove"},
			{"X-Newest": "true"}, {"X-Auth-Token": "token"}, {"Host": "other"}, {"Content-Length": "3"}, {"Authorization": "bearer"}, {"X-Call": "one", "x-call": "two"}, {"X-K": "kelvin"}, {"X-Literal": "bad\nvalue"},
		} {
			for _, source := range []bool{false, true} {
				calls, callbacks := 0, 0
				client := tempKeyClient(func(r *http.Request) (*http.Response, error) { calls++; return tempKeyWire(204, "", nil), nil })
				opts := []v1.GetTempURLKeyOption{func(cfg *v1.GetTempURLKeyOpts) error {
					callbacks++
					if cfg.Headers == nil {
						t.Error("uninitialized headers")
					}
					return nil
				}}
				if source {
					client.MoreHeaders = headers
				} else {
					opts = append(opts, v1.WithGetTempURLKeyHeaders(headers))
				}
				result, err := v1.New(client).GetTempURLKey(context.Background(), opts...)
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || (source && callbacks != 0) {
					t.Fatalf("headers=%v source=%v result=%+v err=%v calls=%d callbacks=%d", headers, source, result, err, calls, callbacks)
				}
			}
		}
		custom := errors.New("callback failure")
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			t.Error("HTTP after callback error")
			return tempKeyWire(204, "", nil), nil
		})
		for _, option := range []v1.GetTempURLKeyOption{nil, func(cfg *v1.GetTempURLKeyOpts) error { return custom }} {
			result, err := v1.New(client).GetTempURLKey(context.Background(), option)
			if result != nil || err == nil || (option != nil && !errors.Is(err, custom)) {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		}
	})
	t.Run("callback guards original target provider and public API identities", func(t *testing.T) {
		for _, mutate := range []func(*v1.Service, *gophercloud.ServiceClient){
			func(s *v1.Service, c *gophercloud.ServiceClient) { c.Endpoint += "changed/" },
			func(s *v1.Service, c *gophercloud.ServiceClient) { c.ResourceBase += "changed/" },
			func(s *v1.Service, c *gophercloud.ServiceClient) { c.Type = "compute" },
			func(s *v1.Service, c *gophercloud.ServiceClient) { c.Microversion = "different" },
			func(s *v1.Service, c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} },
			func(s *v1.Service, c *gophercloud.ServiceClient) { s.Accounts = accounts.New(c) },
			func(s *v1.Service, c *gophercloud.ServiceClient) { s.Containers = containers.New(c) },
			func(s *v1.Service, c *gophercloud.ServiceClient) { s.Objects = nil },
			func(s *v1.Service, c *gophercloud.ServiceClient) { s.Swauth = nil },
		} {
			calls, callbacks := 0, 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) { calls++; return tempKeyWire(204, "", nil), nil })
			service := v1.New(client)
			result, err := service.GetTempURLKey(context.Background(), func(cfg *v1.GetTempURLKeyOpts) error { callbacks++; mutate(service, client); return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 1 {
				t.Fatalf("result=%+v err=%v callbacks=%d calls=%d", result, err, callbacks, calls)
			}
		}
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause := errors.New("callback context cause")
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			t.Error("HTTP after cancelled option")
			return tempKeyWire(204, "", nil), nil
		})
		result, err := v1.New(client).GetTempURLKey(ctx, func(cfg *v1.GetTempURLKeyOpts) error { cancel(cause); return nil })
		if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
	t.Run("late source drift blocks fallback captured headers and live token", func(t *testing.T) {
		for _, change := range []func(*v1.Service, *gophercloud.ServiceClient){func(s *v1.Service, c *gophercloud.ServiceClient) { c.ResourceBase += "changed/" }, func(s *v1.Service, c *gophercloud.ServiceClient) { s.Accounts = accounts.New(c) }, func(s *v1.Service, c *gophercloud.ServiceClient) { c.MoreHeaders["X-Container-Meta-Key"] = "invalid" }} {
			calls := 0
			var service *v1.Service
			var client *gophercloud.ServiceClient
			body := &tempKeyBody{Reader: strings.NewReader("observed container"), onClose: func() { change(service, client) }}
			client = tempKeyClient(func(r *http.Request) (*http.Response, error) { calls++; return tempKeyResponse(204, body, nil), nil })
			client.MoreHeaders = map[string]string{"X-Capture": "original"}
			service = v1.New(client)
			result, err := service.GetTempURLKey(context.Background(), v1.WithGetTempURLKeyContainer("c"))
			if result == nil || result.Container == nil || result.Container.Metadata != nil || result.Account != nil || result.Key != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || body.closes.Load() != 1 {
				t.Fatalf("result=%+v err=%v calls=%d closes=%d", result, err, calls, body.closes.Load())
			}
			tempKeyProof(t, err, 204, "observed container")
		}
		calls := 0
		var client *gophercloud.ServiceClient
		client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Capture") != "original" {
				t.Errorf("header snapshot=%v", r.Header)
			}
			if calls == 1 {
				return tempKeyResponse(204, &tempKeyBody{Reader: strings.NewReader(""), onClose: func() { client.MoreHeaders["X-Capture"] = "later valid source"; client.SetToken("token-two") }}, nil), nil
			}
			if r.Header.Get("X-Auth-Token") != "token-two" {
				t.Errorf("live token=%v", r.Header)
			}
			return tempKeyWire(204, "", http.Header{"X-Account-Meta-Temp-Url-Key": {"account"}}), nil
		})
		client.MoreHeaders = map[string]string{"X-Capture": "original"}
		result, err := v1.New(client).GetTempURLKey(context.Background(), v1.WithGetTempURLKeyContainer("c"))
		tempKeySelected(t, result, err, "account", false, false)
		if calls != 2 || client.MoreHeaders["X-Capture"] != "later valid source" {
			t.Fatalf("calls=%d source=%v", calls, client.MoreHeaders)
		}
	})
}

func TestTempURLKeyContractsPhaseEvidence(t *testing.T) {
	t.Run("accepted setter and getter Read Close context evidence no replay", func(t *testing.T) {
		readErr, closeErr, cause := errors.New("read stopped"), errors.New("close stopped"), errors.New("caller cause")
		for _, op := range []string{"account", "container", "get"} {
			ctx, cancel := context.WithCancelCause(context.Background())
			func() {
				defer cancel(nil)
				calls := 0
				body := &tempKeyBody{Reader: tempKeyReader(func(p []byte) (int, error) { return copy(p, "partial opaque"), readErr }), closeErr: closeErr, onClose: func() { cancel(cause) }}
				client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
					calls++
					return tempKeyResponse(204, body, http.Header{"X-Account-Meta-Temp-Url-Key-2": {"must not select"}}), nil
				})
				var err error
				switch op {
				case "account":
					ack, e := accounts.New(client).SetTempURLKey(ctx, "key")
					err = e
					if ack == nil || string(ack.Body) != "partial opaque" || ack.StatusCode != 204 {
						t.Fatalf("ack=%+v", ack)
					}
				case "container":
					ack, e := containers.New(client).SetTempURLKey(ctx, "c", "key")
					err = e
					if ack == nil || string(ack.Body) != "partial opaque" || ack.StatusCode != 204 {
						t.Fatalf("ack=%+v", ack)
					}
				case "get":
					result, e := v1.New(client).GetTempURLKey(ctx)
					err = e
					if result == nil || result.Account == nil || result.Account.Metadata != nil || result.Key != nil || result.Secondary || result.FromContainer {
						t.Fatalf("result=%+v", result)
					}
				}
				for _, expected := range []error{readErr, closeErr, context.Canceled, cause} {
					if !errors.Is(err, expected) {
						t.Errorf("op=%s missing=%v err=%v", op, expected, err)
					}
				}
				tempKeyProof(t, err, 204, "partial opaque")
				if calls != 1 || body.closes.Load() != 1 {
					t.Fatalf("op=%s calls=%d closes=%d", op, calls, body.closes.Load())
				}
			}()
		}
	})
	t.Run("account fallback native and accepted failures retain earlier container", func(t *testing.T) {
		for _, mode := range []string{"native", "close", "transport"} {
			cause := errors.New("later account cause")
			calls := 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return tempKeyWire(204, "container proof", http.Header{"X-Container-Meta-Unrelated": {"still observed"}}), nil
				}
				if mode == "native" {
					return tempKeyWire(404, "missing account", nil), nil
				}
				if mode == "transport" {
					return nil, cause
				}
				return tempKeyResponse(204, &tempKeyBody{Reader: strings.NewReader("account proof"), closeErr: cause}, http.Header{"X-Account-Meta-Temp-Url-Key": {"not selected"}}), nil
			})
			result, err := v1.New(client).GetTempURLKey(context.Background(), v1.WithGetTempURLKeyContainer("c"))
			if result == nil || result.Container == nil || result.Container.Metadata == nil || string(result.Container.Body) != "container proof" || result.Key != nil || result.Secondary || result.FromContainer || err == nil || calls != 2 {
				t.Fatalf("mode=%s result=%+v err=%v calls=%d", mode, result, err, calls)
			}
			if mode == "close" {
				if result.Account == nil || result.Account.Metadata != nil || !errors.Is(err, cause) {
					t.Fatalf("result=%+v err=%v", result, err)
				}
				tempKeyProof(t, err, 204, "account proof")
			} else if result.Account != nil {
				t.Fatalf("fabricated account proof=%+v", result.Account)
			}
			if mode == "native" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "missing account" {
					t.Fatalf("err=%v native=%+v", err, native)
				}
			}
			if mode == "transport" && !errors.Is(err, cause) {
				t.Fatalf("transport cause lost=%v", err)
			}
		}
	})
	t.Run("full atomic header projection stops selection and never falls back", func(t *testing.T) {
		for _, bad := range []http.Header{
			{"X-Container-Bytes-Used": {"1.5"}}, {"X-Container-Object-Count": {"9223372036854775808"}},
			{"X-Container-Meta-Other": {"one", "two"}}, {"X-Container-Meta-Other": {"one"}, "x-container-meta-other": {"alias"}},
			{"X-Container-Meta-K": {"original nonascii suffix"}}, {"X-Container-Meta-Other": {string([]byte{0xff})}},
			{"X-Timestamp": {"first", "second"}},
		} {
			calls := 0
			bad.Set("X-Container-Meta-Temp-Url-Key-2", "otherwise usable")
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				return tempKeyWire(204, "whole raw body", bad), nil
			})
			result, err := v1.New(client).GetTempURLKey(context.Background(), v1.WithGetTempURLKeyContainer("c"))
			if result == nil || result.Container == nil || result.Container.Metadata != nil || result.Account != nil || result.Key != nil || result.FromContainer || result.Secondary || err == nil || calls != 1 {
				t.Fatalf("bad=%v result=%+v err=%v calls=%d", bad, result, err, calls)
			}
			tempKeyProof(t, err, 204, "whole raw body")
		}
		for _, status := range []int{403, 404, 500} {
			calls := 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				return tempKeyWire(status, "container failure", nil), nil
			})
			result, err := v1.New(client).GetTempURLKey(context.Background(), v1.WithGetTempURLKeyContainer("c"))
			if result != nil || err == nil || calls != 1 {
				t.Fatalf("status=%d result=%+v err=%v calls=%d", status, result, err, calls)
			}
		}
	})
	t.Run("literal dates signed counters opaque body and proof dealiased", func(t *testing.T) {
		body := &tempKeyBody{Reader: strings.NewReader("\xff opaque nonJSON")}
		h := http.Header{"X-Account-Meta-Temp-Url-Key": {"selected"}, "X-Account-Bytes-Used": {"+0"}, "X-Account-Container-Count": {"-2"}, "X-Account-Object-Count": {"9223372036854775807"}, "X-Timestamp": {"literal-not-date"}}
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) { return tempKeyResponse(204, body, h), nil })
		result, err := v1.New(client).GetTempURLKey(context.Background())
		tempKeySelected(t, result, err, "selected", false, false)
		m := result.Account.Metadata
		if m.BytesUsed == nil || *m.BytesUsed != 0 || m.ContainerCount == nil || *m.ContainerCount != -2 || m.ObjectCount == nil || *m.ObjectCount != 9223372036854775807 || m.Timestamp == nil || *m.Timestamp != "literal-not-date" || string(result.Account.Body) != "\xff opaque nonJSON" || body.closes.Load() != 1 {
			t.Fatalf("metadata=%+v result=%+v", m, result)
		}
		closeErr := errors.New("owned close failure")
		client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
			return tempKeyResponse(204, &tempKeyBody{Reader: strings.NewReader("snapshot"), closeErr: closeErr}, nil), nil
		})
		ack, err := accounts.New(client).SetTempURLKey(context.Background(), "key")
		proof := tempKeyProof(t, err, 204, "snapshot")
		ack.Body[0] = 'X'
		ack.Header.Set("X-Trans-Id", "changed")
		if string(proof.Body) != "snapshot" || proof.Header.Get("X-Trans-Id") != "actual-key-phase" || !errors.Is(err, closeErr) {
			t.Fatal("returned acknowledgement aliases error proof")
		}
	})
}

func TestTempURLKeyContractsNativePolicy(t *testing.T) {
	t.Run("configured prebody retry terminal causes and reauth fields", func(t *testing.T) {
		calls, hooks := 0, 0
		rejected := &tempKeyBody{Reader: strings.NewReader("retry response")}
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return tempKeyResponse(503, rejected, nil), nil
			}
			return tempKeyWire(204, "", http.Header{"X-Account-Meta-Temp-Url-Key": {"key"}}), nil
		})
		client.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
			hooks++
			if method != "HEAD" || target != tempKeyAccount || opts.JSONBody != nil || count != 1 {
				t.Errorf("hook method=%s target=%s body=%v count=%d", method, target, opts.JSONBody, count)
			}
			return nil
		}
		result, err := v1.New(client).GetTempURLKey(context.Background())
		tempKeySelected(t, result, err, "key", false, false)
		if calls != 2 || hooks != 1 || rejected.closes.Load() != 1 {
			t.Fatalf("calls=%d hooks=%d closes=%d", calls, hooks, rejected.closes.Load())
		}
		callbackErr := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte("nested callback")}
		calls = 0
		client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			return tempKeyWire(503, "original response", nil), nil
		})
		client.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
			return callbackErr
		}
		ack, err := containers.New(client).SetTempURLKey(context.Background(), "c", "key")
		if ack != nil || !errors.Is(err, callbackErr) || calls != 1 {
			t.Fatalf("ack=%+v err=%v calls=%d", ack, err, calls)
		}
		var original gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &original) || original.Actual != 503 {
			t.Fatalf("original response lost=%v", err)
		}
		reauthErr := errors.New("reauthentication failure")
		calls = 0
		client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			return tempKeyWire(401, "unauthorized", nil), nil
		})
		client.ReauthFunc = func(ctx context.Context) error { return reauthErr }
		result, err = v1.New(client).GetTempURLKey(context.Background(), v1.WithGetTempURLKeyContainer("c"))
		var reauth *gophercloud.ErrUnableToReauthenticate
		if result != nil || !errors.As(err, &reauth) || reauth.ErrOriginal == nil || reauth.ErrReauth != reauthErr || errors.Is(err, reauthErr) || calls != 1 {
			t.Fatalf("result=%+v err=%v reauth=%+v calls=%d", result, err, reauth, calls)
		}
	})
	t.Run("bodyless response ownership and actual code guard before replay", func(t *testing.T) {
		for _, change := range []func(*gophercloud.RequestOpts){func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }, func(o *gophercloud.RequestOpts) { var value any; o.JSONResponse = &value }, func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("injected") }, func(o *gophercloud.RequestOpts) { o.JSONBody = nil; var value *string; o.JSONBody = value }} {
			calls, hooks := 0, 0
			rejected := &tempKeyBody{Reader: strings.NewReader("original503")}
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				tempKeyNoBody(t, r)
				return tempKeyResponse(503, rejected, nil), nil
			})
			client.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
				hooks++
				change(opts)
				return nil
			}
			ack, err := accounts.New(client).SetTempURLKey(context.Background(), "key")
			var native gophercloud.ErrUnexpectedResponseCode
			if ack != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || calls != 1 || hooks != 1 || rejected.closes.Load() != 1 {
				t.Fatalf("ack=%+v err=%v calls=%d hooks=%d closes=%d", ack, err, calls, hooks, rejected.closes.Load())
			}
		}
		calls := 0
		body := &tempKeyBody{Reader: strings.NewReader("unexpected owned200")}
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return tempKeyWire(503, "retry", nil), nil
			}
			return tempKeyResponse(200, body, nil), nil
		})
		client.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
			opts.OkCodes = []int{200, 204}
			return nil
		}
		result, err := v1.New(client).GetTempURLKey(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		if result != nil || !errors.As(err, &native) || native.Actual != 200 || !reflect.DeepEqual(native.Expected, []int{204}) || string(native.Body) != "unexpected owned200" || body.closes.Load() != 1 || calls != 2 {
			t.Fatalf("result=%+v err=%v native=%+v calls=%d closes=%d", result, err, native, calls, body.closes.Load())
		}
	})
	t.Run("advanced native headers and configured redirect target boundary", func(t *testing.T) {
		calls := 0
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				if r.Header.Get("X-Capture") != "source" || len(r.Header.Values("X-Container-Meta-Native")) != 0 {
					t.Errorf("first captured headers=%v", r.Header)
				}
				return tempKeyWire(503, "retry", nil), nil
			}
			if r.Header.Get("X-Container-Meta-Native") != "advanced" || r.Header.Get("X-Capture") != "" {
				t.Errorf("native headers=%v", r.Header)
			}
			return tempKeyWire(204, "", http.Header{"X-Account-Meta-Temp-Url-Key": {"selected"}}), nil
		})
		client.MoreHeaders = map[string]string{"X-Capture": "source"}
		client.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
			opts.MoreHeaders = map[string]string{"X-Container-Meta-Native": "advanced"}
			return nil
		}
		result, err := v1.New(client).GetTempURLKey(context.Background())
		tempKeySelected(t, result, err, "selected", false, false)
		if calls != 2 || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Capture": "source"}) {
			t.Fatalf("calls=%d source=%v", calls, client.MoreHeaders)
		}
		for _, same := range []bool{true, false} {
			calls = 0
			client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					h := http.Header{"Location": {tempKeyAccount}}
					if !same {
						h.Set("Location", tempKeyAccount+"retarget")
					}
					return tempKeyWire(307, "redirect", h), nil
				}
				return tempKeyWire(204, "", http.Header{"X-Account-Meta-Temp-Url-Key": {"key"}}), nil
			})
			result, err = v1.New(client).GetTempURLKey(context.Background())
			if same {
				tempKeySelected(t, result, err, "key", false, false)
				if calls != 2 {
					t.Fatalf("same target calls=%d", calls)
				}
			} else if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatalf("retarget result=%+v err=%v calls=%d", result, err, calls)
			}
		}
	})
	t.Run("native metadata broad codes primary signing and explicit key ABI", func(t *testing.T) {
		calls := 0
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method == "HEAD" {
				return tempKeyWire(200, "", http.Header{"X-Container-Meta-Temp-Url-Key": {"primary"}, "X-Container-Meta-Temp-Url-Key-2": {"secondary"}}), nil
			}
			return tempKeyWire(201, "", nil), nil
		})
		_, err := accounts.New(client).Update(context.Background(), accounts.UpdateOpts{TempURLKey: "native"})
		if err != nil {
			t.Fatalf("native account Update201=%v", err)
		}
		got, err := containers.New(client).Get(context.Background(), "c")
		if err != nil || got == nil || got.TempURLKey != "primary" || got.TempURLKey2 != "secondary" {
			t.Fatalf("native Get200=%+v err=%v", got, err)
		}
		before := calls
		fixed := time.Unix(1234, 0)
		signed, err := objects.New(client).CreateTempURL(context.Background(), "c", "o", objects.CreateTempURLOpts{Method: "GET", TTL: 60, Timestamp: fixed, TempURLKey: "explicit"})
		if err != nil || !strings.Contains(signed, "temp_url_sig=") || calls != before {
			t.Fatalf("explicit signed=%q err=%v calls=%d", signed, err, calls)
		}
		primarySigned, err := objects.New(client).CreateTempURL(context.Background(), "c", "o", objects.CreateTempURLOpts{Method: "GET", TTL: 60, Timestamp: fixed, TempURLKey: "primary"})
		if err != nil || calls != before {
			t.Fatalf("explicit primary signed=%q err=%v calls=%d", primarySigned, err, calls)
		}
		signed, err = objects.New(client).CreateTempURL(context.Background(), "c", "o", objects.CreateTempURLOpts{Method: "GET", TTL: 60, Timestamp: fixed})
		if err != nil || signed != primarySigned || calls != before+1 {
			t.Fatalf("implicit signed=%q primary=%q err=%v calls=%d", signed, primarySigned, err, calls)
		}
		// Native signing has one primary-only container HEAD, whereas the new
		// selector's secondary preference is independently covered above.
		_, err = containers.New(client).Update(context.Background(), "c", containers.UpdateOpts{TempURLKey2: "native secondary"})
		if err != nil {
			t.Fatalf("native container Update201=%v", err)
		}
	})
}
