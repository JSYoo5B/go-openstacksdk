package v1

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func signingTestInput() FormSignatureInput {
	return FormSignatureInput{ObjectPrefix: "prefix", MaxFileSize: 10, MaxUploadCount: 2, Timeout: 60}
}
func signingTestTimestamp() time.Time { return time.Unix(1700000000, 987654321) }

func TestFormSignatureCoreVectorsAndLiteralPaths(t *testing.T) {
	expected := map[TempURLDigest]string{
		TempURLDigestSHA1:   "5a79abff527cd8824ba5301b174b7aad968df0d7",
		TempURLDigestSHA256: "511e8473a22cedf2f460993c97bc87afcce1d11efac1e50c04dfb9fbcf976901",
		TempURLDigestSHA512: "ed6eb6fd14a9aa739c4aaa6f2a133c86a3272314432e1c1ca1bc6f6a6f5e6e33a4349f009802acc909d386646fc86319e1b5cc80bc860b49d02ae2ab0d91b3cc",
	}
	for digest, signature := range expected {
		t.Run(string(digest), func(t *testing.T) {
			c := tempURLKeyTestClient()
			c.ResourceBase = "unused invalid URL"
			calls := 0
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			input := FormSignatureInput{ObjectPrefix: "prefix/책%2F?#", RedirectURL: "https://app.example/result?x=%2F#ok", MaxFileSize: 1048576, MaxUploadCount: 3, Timeout: 123}
			result, err := New(c).GenerateFormSignature(context.Background(), "백업 %2F?#", input,
				WithGenerateFormSignatureKey([]byte("\xffsecret")), WithGenerateFormSignatureDigest(digest), WithGenerateFormSignatureTimestamp(signingTestTimestamp().In(time.FixedZone("local", 3600))))
			if err != nil || result == nil || result.Signature != signature || result.Expires != 1700000123 || result.Digest != digest || result.Discovery != nil || calls != 0 {
				t.Fatalf("vector/proof %#v %v calls=%d", result, err, calls)
			}
			parsed, err := url.Parse(result.URL)
			if err != nil || parsed.Path != result.Path || result.Path != "/reverse proxy/v1/AUTH_x/백업 %2F?#/prefix/책%2F?#" || parsed.Scheme != "http" || parsed.Host != "cloud.invalid" || parsed.RawQuery != "" || parsed.Fragment != "" || !strings.Contains(result.URL, "%252F%3F%23") {
				t.Fatalf("literal path escaped incorrectly %#v %v", result, err)
			}
		})
	}
	for _, endpoint := range []string{"http://cloud.invalid/v1/a/", "http://cloud.invalid/v1/a"} {
		c := tempURLKeyTestClient()
		c.Endpoint = endpoint
		result, err := New(c).GenerateFormSignature(context.Background(), "box", FormSignatureInput{MaxFileSize: 1, MaxUploadCount: 1, Timeout: 1}, WithGenerateFormSignatureKey([]byte("key")), WithGenerateFormSignatureTimestamp(signingTestTimestamp()))
		if err != nil || result == nil || result.Path != "/v1/a/box/" {
			t.Fatal(result, err)
		}
	}
}

func TestFormSignatureCoreDiscoveryAndPhaseProof(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "container secondary", true: "captured fallback"}[fallback], func(t *testing.T) {
			c := tempURLKeyTestClient()
			c.MoreHeaders = map[string]string{"X-Capture": "original"}
			calls := 0
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "HEAD" || r.Body != nil || r.URL.RawQuery != "" || r.Header.Get("X-Capture") != "original" || r.Header.Get("X-Trace") != "owned" || r.Header.Get("X-Newest") != "false" {
					t.Fatal("wrong metadata read", r.URL, r.Header)
				}
				h := http.Header{"X-Proof": {"kept"}}
				body := &tempURLKeyTestBody{Reader: strings.NewReader("container raw")}
				if calls == 1 {
					if r.URL.EscapedPath() != "/data%25/v1/AUTH_x/box" || r.Header.Get("X-Auth-Token") != "token" {
						t.Fatal("wrong first route/auth", r.URL, r.Header)
					}
					if fallback {
						body.onClose = func() { c.MoreHeaders = map[string]string{"X-Capture": "later valid"}; c.SetToken("fresh") }
					} else {
						h.Set("X-Container-Meta-Temp-URL-Key", "primary")
						h.Set("X-Container-Meta-Temp-URL-Key-2", "container-secondary")
					}
				} else {
					if r.URL.String() != c.Endpoint || r.Header.Get("X-Auth-Token") != "fresh" {
						t.Fatal("wrong fallback route/auth", r.URL, r.Header)
					}
					body.Reader = strings.NewReader("account raw")
					h.Set("X-Account-Meta-Temp-URL-Key-2", "account-secondary")
				}
				return tempURLKeyTestWire(r, 204, h, body), nil
			})
			result, err := New(c).GenerateFormSignature(context.Background(), "box", signingTestInput(), WithGenerateFormSignatureTimestamp(signingTestTimestamp()), WithGenerateFormSignatureHeader("X-Trace", "owned"), WithGenerateFormSignatureNewest(false))
			want, callsWant := "088f8af8829bd0f68cdfeb13eabb3a2a8609039b", 1
			if fallback {
				want, callsWant = "03b9a415c089df6c06c47768a2c1114f1c837470", 2
			}
			if err != nil || result == nil || result.Signature != want || calls != callsWant || result.Discovery == nil || result.Discovery.Container == nil || result.Discovery.Container.Metadata == nil || !result.Discovery.Secondary || result.Discovery.FromContainer == fallback {
				t.Fatal(result, err, calls)
			}
			if fallback && (result.Discovery.Account == nil || string(result.Discovery.Account.Body) != "account raw" || string(result.Discovery.Container.Body) != "container raw") {
				t.Fatal("lost distinct phases", result.Discovery)
			}
		})
	}
	t.Run("missing key retains both actual phases", func(t *testing.T) {
		c := tempURLKeyTestClient()
		calls := 0
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			return tempURLKeyTestWire(r, 204, http.Header{"X-Account-Meta-Temp-URL-Key": {""}}, io.NopCloser(strings.NewReader("raw"))), nil
		})
		result, err := New(c).GenerateFormSignature(context.Background(), "box", signingTestInput())
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrNotFound) || errors.As(err, &proof) || result == nil || result.Signature != "" || result.URL != "" || result.Discovery == nil || result.Discovery.Container == nil || result.Discovery.Account == nil || result.Discovery.Key != nil || calls != 2 {
			t.Fatal(result, err, calls)
		}
	})
	t.Run("unexpected fallback preserves container only", func(t *testing.T) {
		c := tempURLKeyTestClient()
		calls := 0
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			code := 204
			if calls == 2 {
				code = 403
			}
			return tempURLKeyTestWire(r, code, http.Header{}, io.NopCloser(strings.NewReader("raw"))), nil
		})
		result, err := New(c).GenerateFormSignature(context.Background(), "box", signingTestInput())
		if err == nil || result == nil || result.Signature != "" || result.Discovery == nil || result.Discovery.Container == nil || result.Discovery.Account != nil || calls != 2 {
			t.Fatal(result, err, calls)
		}
	})
}

func TestFormSignatureCorePreflightAndAcceptedFailures(t *testing.T) {
	t.Run("invalid direct inputs run no callback", func(t *testing.T) {
		for _, tc := range []struct {
			name, container, endpoint string
			input                     FormSignatureInput
		}{
			{"empty container", "", "", signingTestInput()}, {"dot container", "..", "", signingTestInput()}, {"slash container", "a/b", "", signingTestInput()},
			{"empty account slot", "box", "http://cloud.invalid/v1/a//", signingTestInput()}, {"empty proxy slot", "box", "http://cloud.invalid/proxy//v1/a/", signingTestInput()}, {"decoded invalid UTF8", "box", "http://cloud.invalid/v1/%FF/", signingTestInput()},
			{"redirect trailing hyphen", "box", "", FormSignatureInput{MaxFileSize: 1, MaxUploadCount: 1, Timeout: 1, RedirectURL: "https://example/ends-"}},
			{"redirect read limit", "box", "", FormSignatureInput{MaxFileSize: 1, MaxUploadCount: 1, Timeout: 1, RedirectURL: strings.Repeat("책", 1366)}},
			{"prefix dot segment", "box", "", FormSignatureInput{MaxFileSize: 1, MaxUploadCount: 1, Timeout: 1, ObjectPrefix: "a/../b"}},
			{"zero limits", "box", "", FormSignatureInput{Timeout: 1}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				c := tempURLKeyTestClient()
				if tc.endpoint != "" {
					c.Endpoint = tc.endpoint
				}
				calls, callbacks := 0, 0
				c.HTTPClient.Transport = tempURLKeyTestTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("HTTP") })
				result, err := New(c).GenerateFormSignature(context.Background(), tc.container, tc.input, func(*GenerateFormSignatureOpts) error { callbacks++; return nil })
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
					t.Fatal(result, err, calls, callbacks)
				}
			})
		}
	})
	t.Run("joined callback identity and cancellation", func(t *testing.T) {
		c := tempURLKeyTestClient()
		s := New(c)
		cause, optionErr := errors.New("cancel cause"), errors.New("option cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		result, err := s.GenerateFormSignature(ctx, "box", signingTestInput(), func(*GenerateFormSignatureOpts) error { s.Objects = nil; cancel(cause); return optionErr })
		if result != nil {
			t.Fatal("fabricated discovery", result)
		}
		for _, want := range []error{cause, context.Canceled, optionErr, resource.ErrInvalidOption} {
			if !errors.Is(err, want) {
				t.Fatal("lost cause", want, err)
			}
		}
	})
	t.Run("accepted read close cancel retarget stops fallback", func(t *testing.T) {
		c := tempURLKeyTestClient()
		s := New(c)
		readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		body := &tempURLKeyTestBody{Reader: tempURLKeyTestReader(func(p []byte) (int, error) { return copy(p, "x"), readErr }), closeErr: closeErr, onClose: func() { s.Swauth = nil; cancel(cause) }}
		calls := 0
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			return tempURLKeyTestWire(r, 204, http.Header{"X-Proof": {"kept"}, "X-Container-Meta-Temp-URL-Key-2": {"usable"}}, body), nil
		})
		result, err := s.GenerateFormSignature(ctx, "box", signingTestInput())
		if result == nil || result.Signature != "" || result.URL != "" || result.Discovery == nil || result.Discovery.Container == nil || result.Discovery.Container.Metadata != nil || result.Discovery.Account != nil || result.Discovery.Key != nil || calls != 1 || body.closes.Load() != 1 {
			t.Fatal(result, err, calls)
		}
		proof := tempURLKeyTestProof(t, err, "x")
		for _, want := range []error{readErr, closeErr, cause, context.Canceled, resource.ErrInvalidOption} {
			if !errors.Is(err, want) {
				t.Fatal("lost cause", want, err)
			}
		}
		result.Discovery.Container.Body[0] = '!'
		result.Discovery.Container.Header.Set("X-Proof", "changed")
		if string(proof.Body) != "x" || proof.Header.Get("X-Proof") != "kept" {
			t.Fatal("proof aliases result")
		}
	})
}
