package v1

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestTempURLOptionsSnapshotsAndReplacement(t *testing.T) {
	t.Run("full config and callback copies preserve flags key and time", func(t *testing.T) {
		c := tempURLKeyTestClient()
		key := []byte("\xffsecret")
		stamp := signingTestTimestamp()
		newest := true
		headers := map[string]string{"x-trace": "factory"}
		full := WithGenerateTempURLOpts(GenerateTempURLOpts{Key: key, Digest: TempURLDigestSHA256, Timestamp: &stamp, Headers: headers, Newest: &newest, Prefix: true, ISO8601: true, IPRange: "192.0.2.0/24"})
		key[0] = '!'
		stamp = stamp.AddDate(1, 0, 0)
		newest = false
		headers["x-trace"] = "outside"
		var retained *GenerateTempURLOpts
		inspect := func(cfg *GenerateTempURLOpts) error {
			if string(cfg.Key) != "\xffsecret" || cfg.Timestamp.Unix() != 1700000000 || cfg.Newest == nil || !*cfg.Newest || cfg.Headers["x-trace"] != "factory" {
				t.Fatal("factory aliases", cfg)
			}
			retained = cfg
			return nil
		}
		mutate := func(cfg *GenerateTempURLOpts) error {
			retained.Key[0] = '!'
			*retained.Timestamp = stamp
			retained.Prefix = false
			retained.IPRange = "invalid"
			retained.Headers["x-trace"] = "retained"
			if string(cfg.Key) != "\xffsecret" || cfg.Timestamp.Unix() != 1700000000 || !cfg.Prefix || cfg.IPRange != "192.0.2.0/24" {
				t.Fatal("callback config aliases", cfg)
			}
			return nil
		}
		for i := 0; i < 2; i++ {
			result, err := New(c).GenerateTempURL(context.Background(), "/v1/AUTH_x/box/", 60, "put", full, inspect, mutate)
			if err != nil || result == nil || result.Signature != "5caa6c493122e21b63ad7beb4eacf3f88e173002f8427736f9302ffc54fd7bcb" || result.Expires != 1700000060 {
				t.Fatal(result, err)
			}
			parsed, _ := url.Parse(result.URL)
			if parsed.Query().Get("temp_url_expires") != "2023-11-14T22:14:20Z" {
				t.Fatal(parsed.Query())
			}
		}
		result, err := New(c).GenerateTempURL(context.Background(), "/v1/AUTH_x/box/한국 %2F?#/tail", 60, "GET", full, WithGenerateTempURLOpts(GenerateTempURLOpts{Key: []byte("\xffsecret")}), WithGenerateTempURLTimestamp(signingTestTimestamp()), WithGenerateTempURLAbsolute(false), WithGenerateTempURLPrefix(false), WithGenerateTempURLISO8601(false), WithGenerateTempURLIPRange(""))
		if err != nil || result == nil || result.Signature != "fe5b8b5bb2a81f4ec0ece3bd7ea3665ca0677869" || result.Digest != TempURLDigestSHA1 || strings.Contains(result.URL, "temp_url_prefix") || strings.Contains(result.URL, "temp_url_ip_range") {
			t.Fatal("full replacement lost", result, err)
		}
	})
	t.Run("nil key discovery newest clear and last header wins", func(t *testing.T) {
		c := tempURLKeyTestClient()
		headers := map[string]string{"X-Trace": "plural"}
		plural := WithGenerateTempURLHeaders(headers)
		headers["X-Trace"] = "outside"
		calls := 0
		var retained *GenerateTempURLOpts
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Trace") != "last" || len(r.Header.Values("X-Trace")) != 1 || r.Header.Get("X-Newest") != "false" || r.URL.String() != c.Endpoint {
				t.Fatal("read snapshot lost", r.Header, r.URL)
			}
			body := &tempURLKeyTestBody{Reader: strings.NewReader("owned"), onClose: func() {
				retained.Headers["x-trace"] = "retained"
				*retained.Timestamp = signingTestTimestamp().AddDate(1, 0, 0)
				*retained.Newest = true
			}}
			return tempURLKeyTestWire(r, 204, http.Header{"X-Account-Meta-Temp-URL-Key-2": {"account-secondary"}}, body), nil
		})
		full := WithGenerateTempURLOpts(GenerateTempURLOpts{Headers: map[string]string{"x-trace": "old"}})
		result, err := New(c).GenerateTempURL(context.Background(), "/v1/AUTH_x/box/file", 60, "get", WithGenerateTempURLKey([]byte("discard")), full, WithGenerateTempURLKey(nil), WithGenerateTempURLTimestamp(signingTestTimestamp()), WithGenerateTempURLNewest(true), WithoutGenerateTempURLNewest(), WithGenerateTempURLNewest(false), func(cfg *GenerateTempURLOpts) error { retained = cfg; return nil }, plural, WithGenerateTempURLHeader("X-Trace", "last"))
		if err != nil || result == nil || result.Signature != "98649f6963fef8b113423393b6c77b9fda62b2ad" || calls != 1 || result.Discovery == nil || result.Discovery.Account == nil || result.Discovery.Account.Metadata == nil {
			t.Fatal(result, err, calls)
		}
	})
}

func TestTempURLOptionsParallelReuseAndCallbackCauses(t *testing.T) {
	c := tempURLKeyTestClient()
	key := []byte("\xffsecret")
	stamp := signingTestTimestamp()
	full := WithGenerateTempURLOpts(GenerateTempURLOpts{Key: key, Timestamp: &stamp, Digest: TempURLDigestSHA256, Headers: map[string]string{"X-Trace": "stable"}})
	key[0] = '!'
	stamp = stamp.AddDate(1, 0, 0)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			result, err := New(c).GenerateTempURL(context.Background(), "/v1/AUTH_x/box/한국 %2F?#/tail", 60, "get", full)
			if err != nil || result == nil || result.Signature != "a2e233170f7ba06d49aff7b8a5bc67979ffb41989a333ad2d9d4a94e6c8f1c38" || result.Discovery != nil {
				t.Errorf("parallel signing failed %#v %v", result, err)
				return
			}
		}()
	}
	workers.Wait()
	for _, option := range []GenerateTempURLOption{nil, WithGenerateTempURLKey(make([]byte, 0)), WithGenerateTempURLOpts(GenerateTempURLOpts{Key: []byte{}}), WithGenerateTempURLHeaders(map[string]string{"x-trace": "one", "X-Trace": "two"})} {
		client := tempURLKeyTestClient()
		calls := 0
		client.HTTPClient.Transport = tempURLKeyTestTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("HTTP") })
		result, err := New(client).GenerateTempURL(context.Background(), "/v1/a/c/o", 1, "GET", option)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(result, err, calls)
		}
	}
	t.Run("callback error retains guard and custom cancel", func(t *testing.T) {
		client := tempURLKeyTestClient()
		s := New(client)
		cause, optionErr := errors.New("cause"), errors.New("option")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		callbacks := 0
		result, err := s.GenerateTempURL(ctx, "/v1/a/c/o", 1, "GET", func(cfg *GenerateTempURLOpts) error {
			callbacks++
			if cfg.Headers == nil {
				t.Fatal("nil callback Headers")
			}
			s.Accounts = nil
			cancel(cause)
			return optionErr
		})
		if result != nil || callbacks != 1 {
			t.Fatal(result, err, callbacks)
		}
		for _, want := range []error{optionErr, cause, context.Canceled, resource.ErrInvalidOption} {
			if !errors.Is(err, want) {
				t.Fatal("lost callback cause", want, err)
			}
		}
	})
}
