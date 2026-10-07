package v1

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestTempURLCoreVectorsAndEscapedOutput(t *testing.T) {
	for _, tc := range []struct {
		name, path, method, signature, expires string
		seconds                                int64
		options                                []GenerateTempURLOption
	}{
		{"SHA1 literal", "/v1/AUTH_x/box/한국 %2F?#/tail", "get", "fe5b8b5bb2a81f4ec0ece3bd7ea3665ca0677869", "1700000060", 60, nil},
		{"SHA256 empty prefix and IP", "/v1/AUTH_x/box/", "put", "5caa6c493122e21b63ad7beb4eacf3f88e173002f8427736f9302ffc54fd7bcb", "1700000060", 60, []GenerateTempURLOption{WithGenerateTempURLDigest(TempURLDigestSHA256), WithGenerateTempURLPrefix(true), WithGenerateTempURLIPRange("192.0.2.0/24")}},
		{"SHA512 absolute UTC", "/v1/AUTH_x/box/file", "HEAD", "6ef0670805d0ade3cc63f573d6627ab33b15cec28a63aca412fccafa7a0f3066edba8454d0d4fa89576571029aa3715dfaaa4b72f7c173464ee249becd82d442", "2023-11-14T22:30:00Z", 1700001000, []GenerateTempURLOption{WithGenerateTempURLDigest(TempURLDigestSHA512), WithGenerateTempURLAbsolute(true), WithGenerateTempURLISO8601(true)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := tempURLKeyTestClient()
			c.ResourceBase = "invalid unused URL"
			calls := 0
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("HTTP") })
			options := []GenerateTempURLOption{WithGenerateTempURLKey([]byte("\xffsecret")), WithGenerateTempURLTimestamp(signingTestTimestamp().In(time.FixedZone("nonUTC", -3600)))}
			options = append(options, tc.options...)
			result, err := New(c).GenerateTempURL(context.Background(), tc.path, tc.seconds, tc.method, options...)
			if err != nil || result == nil || result.Signature != tc.signature || result.Path != tc.path || result.Discovery != nil || calls != 0 {
				t.Fatal(result, err, calls)
			}
			parsed, err := url.Parse(result.URL)
			if err != nil || parsed.Path != tc.path || parsed.Scheme != "" || parsed.Host != "" || parsed.Fragment != "" || parsed.Query().Get("temp_url_sig") != tc.signature || parsed.Query().Get("temp_url_expires") != tc.expires {
				t.Fatal("wrong output", result, err)
			}
			if strings.Contains(tc.path, "%2F") && !strings.Contains(result.URL, "%252F%3F%23") {
				t.Fatal("not escaped once", result.URL)
			}
			if tc.name == "SHA256 empty prefix and IP" {
				if values, ok := parsed.Query()["temp_url_prefix"]; !ok || len(values) != 1 || values[0] != "" || parsed.Query().Get("temp_url_ip_range") != "192.0.2.0/24" {
					t.Fatal("prefix/IP presence lost", parsed.Query())
				}
			}
		})
	}
}

func TestTempURLCorePreflightAndTimeBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, path, method string
		seconds            int64
		options            []GenerateTempURLOption
	}{
		{"negative seconds", "/v1/a/c/o", "GET", -1, nil}, {"wrong version", "/v2/a/c/o", "GET", 1, nil}, {"empty account", "/v1//c/o", "GET", 1, nil}, {"empty container", "/v1/a//o", "GET", 1, nil}, {"empty object", "/v1/a/c/", "GET", 1, nil}, {"dot component", "/v1/a/c/x/../y", "GET", 1, nil}, {"backslash", "/v1/a/c/a\\b", "GET", 1, nil}, {"invalid UTF8", "/v1/a/c/" + string([]byte{255}), "GET", 1, nil}, {"method token", "/v1/a/c/o", "GET POST", 1, nil},
		{"non-nil empty key", "/v1/a/c/o", "GET", 1, []GenerateTempURLOption{WithGenerateTempURLKey([]byte{})}}, {"unsupported digest", "/v1/a/c/o", "GET", 1, []GenerateTempURLOption{WithGenerateTempURLDigest("md5")}},
		{"relative overflow", "/v1/a/c/o", "GET", math.MaxInt64, nil}, {"negative timestamp absolute", "/v1/a/c/o", "GET", 1, []GenerateTempURLOption{WithGenerateTempURLAbsolute(true), WithGenerateTempURLTimestamp(time.Unix(-1, 0))}},
		{"ISO year overflow", "/v1/a/c/o", "GET", 253402300800, []GenerateTempURLOption{WithGenerateTempURLAbsolute(true), WithGenerateTempURLISO8601(true)}},
		{"CIDR host bits", "/v1/a/c/o", "GET", 1, []GenerateTempURLOption{WithGenerateTempURLIPRange("192.0.2.7/24")}}, {"scoped IPv6", "/v1/a/c/o", "GET", 1, []GenerateTempURLOption{WithGenerateTempURLIPRange("fe80::1%eth0")}}, {"padded IP", "/v1/a/c/o", "GET", 1, []GenerateTempURLOption{WithGenerateTempURLIPRange(" 192.0.2.1")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := tempURLKeyTestClient()
			calls := 0
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("HTTP") })
			options := []GenerateTempURLOption{WithGenerateTempURLKey([]byte("key")), WithGenerateTempURLTimestamp(signingTestTimestamp())}
			options = append(options, tc.options...)
			result, err := New(c).GenerateTempURL(context.Background(), tc.path, tc.seconds, tc.method, options...)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(result, err, calls)
			}
		})
	}
	t.Run("absolute int64 without ISO and nonstandard token", func(t *testing.T) {
		c := tempURLKeyTestClient()
		result, err := New(c).GenerateTempURL(context.Background(), "/v1/a/c/o", math.MaxInt64, "x-CUSTOM", WithGenerateTempURLKey([]byte("key")), WithGenerateTempURLAbsolute(true))
		if err != nil || result == nil || result.Expires != math.MaxInt64 || result.Signature == "" {
			t.Fatal(result, err)
		}
	})
	t.Run("source preflight before callback", func(t *testing.T) {
		c := tempURLKeyTestClient()
		c.MoreHeaders = map[string]string{"X-K": "bad"}
		callbacks := 0
		result, err := New(c).GenerateTempURL(context.Background(), "/v1/a/c/o", 1, "GET", func(*GenerateTempURLOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal(result, err, callbacks)
		}
	})
}

func TestTempURLCoreDiscoveryAndNativePolicy(t *testing.T) {
	t.Run("account-only ignores unused ResourceBase", func(t *testing.T) {
		c := tempURLKeyTestClient()
		c.ResourceBase = "bad unused"
		calls := 0
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "HEAD" || r.URL.String() != c.Endpoint {
				t.Fatal("not account-only", r.URL)
			}
			return tempURLKeyTestWire(r, 204, http.Header{"X-Account-Meta-Temp-URL-Key": {"primary"}, "X-Account-Meta-Temp-URL-Key-2": {"account-secondary"}}, io.NopCloser(strings.NewReader("account proof"))), nil
		})
		result, err := New(c).GenerateTempURL(context.Background(), "/v1/AUTH_x/box/file", 60, "GET", WithGenerateTempURLTimestamp(signingTestTimestamp()))
		if err != nil || result == nil || result.Signature != "98649f6963fef8b113423393b6c77b9fda62b2ad" || result.Discovery == nil || result.Discovery.Account == nil || result.Discovery.Container != nil || !result.Discovery.Secondary || result.Discovery.FromContainer || string(result.Discovery.Account.Body) != "account proof" || calls != 1 {
			t.Fatal(result, err, calls)
		}
	})
	t.Run("accepted outer drift clears current metadata", func(t *testing.T) {
		c := tempURLKeyTestClient()
		body := &tempURLKeyTestBody{Reader: strings.NewReader("accepted"), onClose: func() { c.ResourceBase += "changed" }}
		calls := 0
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			return tempURLKeyTestWire(r, 204, http.Header{"X-Proof": {"kept"}, "X-Account-Meta-Temp-URL-Key-2": {"key"}}, body), nil
		})
		result, err := New(c).GenerateTempURL(context.Background(), "/v1/a/c/o", 1, "GET")
		if result == nil || result.Signature != "" || result.URL != "" || result.Discovery == nil || result.Discovery.Account == nil || result.Discovery.Account.Metadata != nil || result.Discovery.Key != nil || calls != 1 || body.closes.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err, calls)
		}
		tempURLKeyTestProof(t, err, "accepted")
	})
	t.Run("atomic projection rejects original invalid suffix", func(t *testing.T) {
		c := tempURLKeyTestClient()
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			return tempURLKeyTestWire(r, 204, http.Header{"X-Proof": {"kept"}, "X-Account-Meta-Temp-URL-Key-2": {"key"}, "X-Account-Meta-K": {"invalid"}}, io.NopCloser(strings.NewReader("raw"))), nil
		})
		result, err := New(c).GenerateTempURL(context.Background(), "/v1/a/c/o", 1, "GET")
		if result == nil || result.Discovery == nil || result.Discovery.Account == nil || result.Discovery.Account.Metadata != nil || result.Discovery.Key != nil || result.Signature != "" || result.URL != "" || result.Expires != 0 || result.Digest != "" || err == nil || !strings.Contains(err.Error(), "invalid account header name") {
			t.Fatal(result, err)
		}
		tempURLKeyTestProof(t, err, "raw")
	})
	t.Run("native terminal retry cause never fabricates discovery", func(t *testing.T) {
		c := tempURLKeyTestClient()
		calls, hooks := 0, 0
		cause := errors.New("hook stop")
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			return tempURLKeyTestWire(r, 503, http.Header{}, io.NopCloser(strings.NewReader("rejected"))), nil
		})
		c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return cause
		}
		result, err := New(c).GenerateTempURL(context.Background(), "/v1/a/c/o", 1, "GET")
		if result != nil || !errors.Is(err, cause) || !gophercloud.ResponseCodeIs(err, 503) || calls != 1 || hooks != 1 {
			t.Fatal(result, err, calls, hooks)
		}
	})
	t.Run("expanded native accepted code remains rejected", func(t *testing.T) {
		c := tempURLKeyTestClient()
		calls := 0
		closeErr := errors.New("wrong status close")
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			code := 503
			if calls == 2 {
				code = 200
			}
			return tempURLKeyTestWire(r, code, http.Header{}, &tempURLKeyTestBody{Reader: strings.NewReader("wrong raw"), closeErr: closeErr}), nil
		})
		c.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			opts.OkCodes = append(opts.OkCodes, 200)
			return nil
		}
		result, err := New(c).GenerateTempURL(context.Background(), "/v1/a/c/o", 1, "GET")
		if result != nil || !gophercloud.ResponseCodeIs(err, 200) || !errors.Is(err, closeErr) || calls != 2 {
			t.Fatal(result, err, calls)
		}
	})
}
