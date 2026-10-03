package v1_test

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	nativeobjects "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects"
	"gophercloudsdk/objectstorage/v1"
	"gophercloudsdk/objectstorage/v1/accounts"
	"gophercloudsdk/objectstorage/v1/containers"
	"gophercloudsdk/objectstorage/v1/objects"
	"gophercloudsdk/objectstorage/v1/swauth"
	"gophercloudsdk/resource"
)

var signingInstant = time.Unix(1700000000, 0).UTC()

const signingPath = "/v1/AUTH_demo/photos/report.txt"
const signingDefaultMAC = "fae90b1cb89365ab777417932d5650ed019558f7"

func signingClientNoHTTP(t *testing.T) *gophercloud.ServiceClient {
	t.Helper()
	return tempKeyClient(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected HTTP %s %s", r.Method, r.URL)
		return nil, errors.New("unexpected signing HTTP")
	})
}
func signingInput() v1.FormSignatureInput {
	return v1.FormSignatureInput{MaxFileSize: 1024, MaxUploadCount: 3, Timeout: 60}
}
func signingUnsignedTemp(t *testing.T, result *v1.TempURLResult) {
	t.Helper()
	if result != nil && (result.Path != "" || result.URL != "" || result.Signature != "" || result.Expires != 0 || result.Digest != "") {
		t.Fatalf("failed signing exposed signed fields: %+v", result)
	}
}
func signingUnsignedForm(t *testing.T, result *v1.FormSignatureResult) {
	t.Helper()
	if result != nil && (result.Path != "" || result.URL != "" || result.Signature != "" || result.Expires != 0 || result.Digest != "") {
		t.Fatalf("failed form exposed signed fields: %+v", result)
	}
}
func signingTempOK(t *testing.T, result *v1.TempURLResult, err error, signature string) {
	t.Helper()
	if err != nil || result == nil || result.Signature != signature || result.Expires != 1700000060 || result.Path != signingPath {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	parsed, parseErr := url.Parse(result.URL)
	if parseErr != nil || parsed.Host != "" || parsed.Path != signingPath || parsed.Query().Get("temp_url_sig") != signature || parsed.Query().Get("temp_url_expires") != "1700000060" {
		t.Fatalf("URL=%q parse=%v", result.URL, parseErr)
	}
}

func TestTempURLSigningContractsKnownVectorsAndPaths(t *testing.T) {
	// Constants are independent Python stdlib HMAC vectors over pinned Swift payloads.
	t.Run("form binary key all digests", func(t *testing.T) {
		client := signingClientNoHTTP(t)
		client.Endpoint = "https://swift.invalid/v1/AUTH_a/"
		input := v1.FormSignatureInput{ObjectPrefix: "dir/% ?#한", RedirectURL: "https://example.test/done?a=1&b=한", MaxFileSize: 123, MaxUploadCount: 3, Timeout: 60}
		for _, tc := range []struct {
			digest v1.TempURLDigest
			want   string
		}{
			{v1.TempURLDigestSHA1, "b5dc36eddaa76fe3f5feceec2b83f3c8d7359972"},
			{v1.TempURLDigestSHA256, "36ce76380345887f4892b5afb2ca209882ae1a8d1c9cd2ebce9b641c1c98efe1"},
			{v1.TempURLDigestSHA512, "9f1572282761aac5f55f8fbaee1800790667e8bc57daf522b14a41f27ede94d4ccb51a3cfb0fc3d86e54c3c27cbe6b824b6675b38d29987ad719bd7e70fe973e"},
		} {
			t.Run(string(tc.digest), func(t *testing.T) {
				result, err := v1.New(client).GenerateFormSignature(context.Background(), "bucket", input,
					v1.WithGenerateFormSignatureKey([]byte{'k', 0xff, 'y'}), v1.WithGenerateFormSignatureTimestamp(signingInstant), v1.WithGenerateFormSignatureDigest(tc.digest))
				if err != nil || result == nil || result.Path != "/v1/AUTH_a/bucket/dir/% ?#한" || result.Signature != tc.want || result.Expires != 1700000060 || result.Digest != tc.digest || result.Discovery != nil {
					t.Fatalf("result=%+v err=%v", result, err)
				}
				u, e := url.Parse(result.URL)
				if e != nil || u.Path != result.Path || u.Host != "swift.invalid" || u.RawQuery != "" || !strings.Contains(result.URL, "%25") || !strings.Contains(result.URL, "%3F%23") {
					t.Fatalf("URL=%q err=%v", result.URL, e)
				}
			})
		}
	})
	t.Run("temp prefix IP binary key all digests", func(t *testing.T) {
		path := "/v1/AUTH_a/bucket/dir/% ?#한"
		for _, tc := range []struct {
			digest v1.TempURLDigest
			want   string
		}{
			{v1.TempURLDigestSHA1, "fe579be4f6d595d50430586538cca3cc5d0461ad"},
			{v1.TempURLDigestSHA256, "bab606b29aefce7582995faba46cb56dcc3691e133445eb0a2fe3a0f7db3beca"},
			{v1.TempURLDigestSHA512, "95e8665a675d7c534abdf4a499d1817a0645b763cfaa553c0f2042d03ea8b60ed01099f7daef78d1034b484fc04bbe2962dad40017f45cf7df0d6ad79e3d46a3"},
		} {
			t.Run(string(tc.digest), func(t *testing.T) {
				result, err := v1.New(signingClientNoHTTP(t)).GenerateTempURL(context.Background(), path, 60, "get",
					v1.WithGenerateTempURLKey([]byte{'k', 0xff, 'y'}), v1.WithGenerateTempURLTimestamp(signingInstant), v1.WithGenerateTempURLDigest(tc.digest), v1.WithGenerateTempURLPrefix(true), v1.WithGenerateTempURLIPRange("2001:db8::/64"))
				if err != nil || result == nil || result.Signature != tc.want || result.Path != path || result.Digest != tc.digest || result.Discovery != nil {
					t.Fatalf("result=%+v err=%v", result, err)
				}
				u, e := url.Parse(result.URL)
				if e != nil || u.Path != path || u.Host != "" || u.Fragment != "" || u.Query().Get("temp_url_prefix") != "dir/% ?#한" || u.Query().Get("temp_url_ip_range") != "2001:db8::/64" || u.Query().Get("temp_url_sig") != tc.want || strings.Contains(tc.want, ":") {
					t.Fatalf("URL=%q err=%v", result.URL, e)
				}
			})
		}
	})
	t.Run("decoded endpoint and literal percent paths", func(t *testing.T) {
		client := signingClientNoHTTP(t)
		client.Endpoint = "https://cloud.invalid/proxy%20x/v1/AUTH%25/"
		client.ResourceBase = "not a usable resource base"
		input := signingInput()
		input.ObjectPrefix = "사진/obj %2F?#"
		input.RedirectURL = "https://example.test/done?ok=1&value=%2F"
		result, err := v1.New(client).GenerateFormSignature(context.Background(), "상자 %2F?#", input,
			v1.WithGenerateFormSignatureKey([]byte("test-secret")), v1.WithGenerateFormSignatureTimestamp(signingInstant))
		if err != nil || result == nil || result.Path != "/proxy x/v1/AUTH%/상자 %2F?#/사진/obj %2F?#" || result.Signature != "cbbe6f01d8d16c3eeb12898fd9666535e106483f" {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		u, e := url.Parse(result.URL)
		if e != nil || u.Path != result.Path || u.Host != "cloud.invalid" || !strings.Contains(result.URL, "%252F") {
			t.Fatalf("URL=%q err=%v", result.URL, e)
		}
		prefix, err := v1.New(client).GenerateTempURL(context.Background(), "/v1/a/c/", 60, "GET",
			v1.WithGenerateTempURLKey([]byte("test-secret")), v1.WithGenerateTempURLTimestamp(signingInstant), v1.WithGenerateTempURLPrefix(true))
		if err != nil || prefix == nil || prefix.Signature != "9be5f5e07e32bee069bb55fe089ad6518863447d" {
			t.Fatalf("prefix=%+v err=%v", prefix, err)
		}
		p, e := url.Parse(prefix.URL)
		if e != nil {
			t.Fatal(e)
		}
		if values, ok := p.Query()["temp_url_prefix"]; !ok || !reflect.DeepEqual(values, []string{""}) {
			t.Fatalf("prefix query=%v", p.Query())
		}
	})
	t.Run("default relative absolute ISO and custom method", func(t *testing.T) {
		service := v1.New(signingClientNoHTTP(t))
		result, err := service.GenerateTempURL(context.Background(), signingPath, 60, "get", v1.WithGenerateTempURLKey([]byte("test-secret")), v1.WithGenerateTempURLTimestamp(signingInstant))
		signingTempOK(t, result, err, signingDefaultMAC)
		if result.Digest != v1.TempURLDigestSHA1 {
			t.Fatalf("default digest=%q", result.Digest)
		}
		iso, err := service.GenerateTempURL(context.Background(), signingPath, 1700000060, "GET", v1.WithGenerateTempURLKey([]byte("test-secret")), v1.WithGenerateTempURLTimestamp(signingInstant), v1.WithGenerateTempURLAbsolute(true), v1.WithGenerateTempURLISO8601(true))
		if err != nil || iso == nil || iso.Signature != signingDefaultMAC || iso.Expires != 1700000060 {
			t.Fatalf("iso=%+v err=%v", iso, err)
		}
		u, e := url.Parse(iso.URL)
		if e != nil || u.Query().Get("temp_url_expires") != "2023-11-14T22:14:20Z" {
			t.Fatalf("isoURL=%q err=%v", iso.URL, e)
		}
		zero, err := service.GenerateTempURL(context.Background(), "/v1/a/c/o", 0, "propfind", v1.WithGenerateTempURLKey([]byte("test-secret")), v1.WithGenerateTempURLTimestamp(time.Unix(0, 0)), v1.WithGenerateTempURLAbsolute(true))
		if err != nil || zero == nil || zero.Expires != 0 || zero.Signature != "dc7a77d0a77c03566e28b817d374fee98fe8cb6e" {
			t.Fatalf("zero=%+v err=%v", zero, err)
		}
	})
}

func TestTempURLSigningContractsDiscoveryAndEvidence(t *testing.T) {
	t.Run("form secondary primary and fallback routes", func(t *testing.T) {
		for _, tc := range []struct {
			name               string
			container, account http.Header
			key                string
			from, secondary    bool
			calls              int
		}{
			{"container secondary", http.Header{"X-Container-Meta-Temp-Url-Key-2": {"container-secondary"}, "X-Container-Meta-Temp-Url-Key": {"container-primary"}}, nil, "container-secondary", true, true, 1},
			{"container primary", http.Header{"X-Container-Meta-Temp-Url-Key-2": {""}, "X-Container-Meta-Temp-Url-Key": {"container-primary"}}, nil, "container-primary", true, false, 1},
			{"account secondary", http.Header{"X-Container-Meta-Temp-Url-Key": {""}}, http.Header{"X-Account-Meta-Temp-Url-Key": {"account-primary"}, "X-Account-Meta-Temp-Url-Key-2": {"account-secondary"}}, "account-secondary", false, true, 2},
			{"account primary", nil, http.Header{"X-Account-Meta-Temp-Url-Key": {"account-primary"}, "X-Account-Meta-Temp-Url-Key-2": {""}}, "account-primary", false, false, 2},
		} {
			t.Run(tc.name, func(t *testing.T) {
				calls := 0
				client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
					calls++
					tempKeyNoBody(t, r)
					want := tempKeyBase + url.PathEscape(tempKeyName)
					headers := tc.container
					if calls == 2 {
						want = tempKeyAccount
						headers = tc.account
					}
					if r.Method != http.MethodHead || r.URL.String() != want || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Call") != "final" {
						t.Errorf("phase%d %s %s headers=%v", calls, r.Method, r.URL, r.Header)
					}
					return tempKeyWire(204, "opaque-phase", headers), nil
				})
				result, err := v1.New(client).GenerateFormSignature(context.Background(), tempKeyName, signingInput(),
					v1.WithGenerateFormSignatureTimestamp(signingInstant), v1.WithGenerateFormSignatureHeaders(map[string]string{"x-call": "initial"}), v1.WithGenerateFormSignatureHeader("X-Call", "final"), v1.WithGenerateFormSignatureNewest(false))
				if err != nil || result == nil || result.Discovery == nil || string(result.Discovery.Key) != tc.key || result.Discovery.FromContainer != tc.from || result.Discovery.Secondary != tc.secondary || calls != tc.calls {
					t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
				}
				if result.Discovery.Container == nil || result.Discovery.Container.StatusCode != 204 || string(result.Discovery.Container.Body) != "opaque-phase" {
					t.Fatalf("container=%+v", result.Discovery.Container)
				}
				if tc.calls == 2 && (result.Discovery.Account == nil || result.Discovery.Account.StatusCode != 204) {
					t.Fatalf("account=%+v", result.Discovery.Account)
				}
			})
		}
	})
	t.Run("temp discovers account only with newest reset", func(t *testing.T) {
		calls := 0
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			tempKeyNoBody(t, r)
			if r.Method != "HEAD" || r.URL.String() != tempKeyAccount || r.Header.Get("X-Newest") != "" || r.Header.Get("X-Call") != "final" || r.Header.Get("X-Extra") != "kept" {
				t.Errorf("request=%s %s headers=%v", r.Method, r.URL, r.Header)
			}
			return tempKeyWire(204, "account-proof", http.Header{"X-Account-Meta-Temp-Url-Key-2": {"account-secondary"}}), nil
		})
		client.ResourceBase = "unused and invalid"
		result, err := v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET",
			v1.WithGenerateTempURLTimestamp(signingInstant), v1.WithGenerateTempURLHeader("x-call", "initial"), v1.WithGenerateTempURLHeaders(map[string]string{"X-Call": "final", "X-Extra": "kept"}), v1.WithGenerateTempURLNewest(true), v1.WithoutGenerateTempURLNewest())
		signingTempOK(t, result, err, "b8c34dc228e9eb17451dae9cc7c5a870dea377a9")
		if calls != 1 || result.Discovery.Container != nil || result.Discovery.Account == nil || !result.Discovery.Secondary || result.Discovery.FromContainer {
			t.Fatalf("discovery=%+v calls=%d", result.Discovery, calls)
		}
	})
	t.Run("missing key retains real proof without fabricated HTTP error", func(t *testing.T) {
		calls := 0
		body := &tempKeyBody{Reader: strings.NewReader("empty-account")}
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			return tempKeyResponse(204, body, http.Header{"X-Account-Meta-Temp-Url-Key": {""}, "X-Account-Meta-Other": {"literal 한"}}), nil
		})
		result, err := v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLTimestamp(signingInstant))
		signingUnsignedTemp(t, result)
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrNotFound) || errors.As(err, &proof) || result == nil || result.Discovery == nil || result.Discovery.Key != nil || result.Discovery.Account == nil || string(result.Discovery.Account.Body) != "empty-account" || result.Discovery.Account.Metadata.Values["other"] != "literal 한" || calls != 1 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v calls=%d closes=%d", result, err, calls, body.closes.Load())
		}
		formCalls := 0
		formClient := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			formCalls++
			return tempKeyWire(204, "both-empty", nil), nil
		})
		form, err := v1.New(formClient).GenerateFormSignature(context.Background(), "bucket", signingInput(), v1.WithGenerateFormSignatureTimestamp(signingInstant), v1.WithGenerateFormSignatureNewest(true), v1.WithoutGenerateFormSignatureNewest())
		signingUnsignedForm(t, form)
		if !errors.Is(err, resource.ErrNotFound) || form == nil || form.Discovery.Container == nil || form.Discovery.Account == nil || formCalls != 2 {
			t.Fatalf("form=%+v err=%v calls=%d", form, err, formCalls)
		}
	})
	t.Run("native first failure later failure and atomic projection", func(t *testing.T) {
		t.Run("first unexpected", func(t *testing.T) {
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) { return tempKeyWire(200, "native-200", nil), nil })
			result, err := v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET")
			var native gophercloud.ErrUnexpectedResponseCode
			if result != nil || !errors.As(err, &native) || native.Actual != 200 || string(native.Body) != "native-200" {
				t.Fatalf("result=%+v native=%+v err=%v", result, native, err)
			}
		})
		t.Run("account error retains container", func(t *testing.T) {
			calls := 0
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return tempKeyWire(204, "container-before", nil), nil
				}
				return tempKeyWire(403, "account-denied", nil), nil
			})
			result, err := v1.New(client).GenerateFormSignature(context.Background(), "bucket", signingInput())
			signingUnsignedForm(t, result)
			var native gophercloud.ErrUnexpectedResponseCode
			if result == nil || result.Discovery.Container == nil || result.Discovery.Account != nil || !errors.As(err, &native) || native.Actual != 403 || calls != 2 {
				t.Fatalf("result=%+v native=%+v err=%v", result, native, err)
			}
		})
		t.Run("unrelated malformed header blocks key selection", func(t *testing.T) {
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				return tempKeyWire(204, "malformed-count", http.Header{"X-Account-Meta-Temp-Url-Key": {"test-secret"}, "X-Account-Object-Count": {"2.5"}}), nil
			})
			result, err := v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET")
			signingUnsignedTemp(t, result)
			tempKeyProof(t, err, 204, "malformed-count")
			if result == nil || result.Discovery.Account.Metadata != nil || result.Discovery.Key != nil {
				t.Fatalf("result=%+v", result)
			}
		})
	})
}

func TestTempURLSigningContractsOptionsAndPreflight(t *testing.T) {
	t.Run("owned factory full replacement and callback snapshots", func(t *testing.T) {
		key := []byte("test-secret")
		headers := map[string]string{"x-call": "factory"}
		stamp := signingInstant
		newest := true
		full := v1.WithGenerateTempURLOpts(v1.GenerateTempURLOpts{Key: key, Timestamp: &stamp, Headers: headers, Newest: &newest})
		key[0] = 'X'
		headers["x-call"] = "mutated"
		stamp = time.Unix(1, 0)
		newest = false
		var retained *v1.GenerateTempURLOpts
		calls := 0
		result, err := v1.New(signingClientNoHTTP(t)).GenerateTempURL(context.Background(), signingPath, 60, "GET",
			v1.WithGenerateTempURLDigest(v1.TempURLDigestSHA512), v1.WithGenerateTempURLPrefix(true), full,
			func(cfg *v1.GenerateTempURLOpts) error { calls++; retained = cfg; return nil },
			func(cfg *v1.GenerateTempURLOpts) error {
				calls++
				retained.Key[0] = 'Y'
				retained.Headers["x-call"] = "retained"
				*retained.Timestamp = time.Unix(2, 0)
				*retained.Newest = false
				if string(cfg.Key) != "test-secret" || cfg.Headers["x-call"] != "factory" || cfg.Timestamp.Unix() != 1700000000 || !*cfg.Newest || cfg.Digest != "" || cfg.Prefix {
					t.Errorf("snapshot=%+v", cfg)
				}
				return nil
			})
		signingTempOK(t, result, err, signingDefaultMAC)
		if calls != 2 {
			t.Fatalf("callbacks=%d", calls)
		}
		formKey := []byte("test-secret")
		formStamp := signingInstant
		formHeaders := map[string]string{"X-Call": "form"}
		formFull := v1.WithGenerateFormSignatureOpts(v1.GenerateFormSignatureOpts{Key: formKey, Timestamp: &formStamp, Headers: formHeaders})
		formKey[0] = 'Z'
		formStamp = time.Unix(3, 0)
		formHeaders["X-Call"] = "external"
		var first *v1.GenerateFormSignatureOpts
		form, err := v1.New(signingClientNoHTTP(t)).GenerateFormSignature(context.Background(), "bucket", signingInput(), v1.WithGenerateFormSignatureDigest(v1.TempURLDigestSHA512), formFull,
			func(cfg *v1.GenerateFormSignatureOpts) error { first = cfg; return nil }, func(cfg *v1.GenerateFormSignatureOpts) error {
				first.Key[0] = 'Q'
				first.Headers["X-Call"] = "alias"
				*first.Timestamp = time.Unix(4, 0)
				if string(cfg.Key) != "test-secret" || cfg.Timestamp.Unix() != 1700000000 || cfg.Headers["X-Call"] != "form" || cfg.Digest != "" {
					t.Errorf("form snapshot=%+v", cfg)
				}
				return nil
			})
		if err != nil || form == nil || form.Digest != v1.TempURLDigestSHA1 || form.Expires != 1700000060 {
			t.Fatalf("form=%+v err=%v", form, err)
		}
	})
	t.Run("key and header factories reused concurrently", func(t *testing.T) {
		key := []byte("test-secret")
		keyOpt := v1.WithGenerateTempURLKey(key)
		key[0] = 'X'
		formKey := []byte("test-secret")
		formKeyOpt := v1.WithGenerateFormSignatureKey(formKey)
		formKey[0] = 'Y'
		headers := map[string]string{"X-Call": "literal"}
		headerOpt := v1.WithGenerateTempURLHeaders(headers)
		formHeaderOpt := v1.WithGenerateFormSignatureHeaders(headers)
		headers["X-Call"] = "changed"
		service := v1.New(signingClientNoHTTP(t))
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result, err := service.GenerateTempURL(context.Background(), signingPath, 60, "GET", keyOpt, headerOpt, v1.WithGenerateTempURLTimestamp(signingInstant))
				if err != nil || result == nil || result.Signature != signingDefaultMAC {
					t.Errorf("concurrent temp=%+v err=%v", result, err)
					return
				}
				form, err := service.GenerateFormSignature(context.Background(), "bucket", signingInput(), formKeyOpt, formHeaderOpt, v1.WithGenerateFormSignatureTimestamp(signingInstant))
				if err != nil || form == nil || form.Expires != 1700000060 {
					t.Errorf("concurrent form=%+v err=%v", form, err)
				}
			}()
		}
		wg.Wait()
	})
	t.Run("all lexical and numeric failures precede HTTP", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			run  func(*v1.Service) error
		}{
			{"empty explicit key", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLKey([]byte{}))
				return e
			}},
			{"form empty explicit key", func(s *v1.Service) error {
				_, e := s.GenerateFormSignature(context.Background(), "bucket", signingInput(), v1.WithGenerateFormSignatureKey([]byte{}))
				return e
			}},
			{"digest", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLDigest("SHA256"))
				return e
			}},
			{"negative relative", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, -1, "GET")
				return e
			}},
			{"overflow", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, math.MaxInt64, "GET", v1.WithGenerateTempURLTimestamp(signingInstant))
				return e
			}},
			{"negative clock even absolute", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 0, "GET", v1.WithGenerateTempURLTimestamp(time.Unix(-1, 0)), v1.WithGenerateTempURLAbsolute(true))
				return e
			}},
			{"ISO out of range", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 253402300800, "GET", v1.WithGenerateTempURLAbsolute(true), v1.WithGenerateTempURLISO8601(true))
				return e
			}},
			{"IP host bits", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLIPRange("192.0.2.1/24"))
				return e
			}},
			{"IP zone", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLIPRange("fe80::1%en0"))
				return e
			}},
			{"IP surrounding space", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLIPRange(" 192.0.2.1"))
				return e
			}},
			{"method non token", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 60, "G ET")
				return e
			}},
			{"method Unicode", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 60, "한")
				return e
			}},
			{"wrong version", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), "/v2/a/c/o", 60, "GET")
				return e
			}},
			{"missing account", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), "/v1//c/o", 60, "GET")
				return e
			}},
			{"empty object without prefix", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), "/v1/a/c/", 60, "GET")
				return e
			}},
			{"dot segment", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), "/v1/a/c/dir/../o", 60, "GET")
				return e
			}},
			{"backslash", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), "/v1/a/c/dir\\o", 60, "GET")
				return e
			}},
			{"path control", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), "/v1/a/c/o\n", 60, "GET")
				return e
			}},
			{"path DEL", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), "/v1/a/c/o\x7f", 60, "GET")
				return e
			}},
			{"path UTF8", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), "/v1/a/c/\xff", 60, "GET")
				return e
			}},
			{"nil option", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 60, "GET", nil)
				return e
			}},
			{"reserved header", func(s *v1.Service) error {
				_, e := s.GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLHeader("X-Auth-Token", "caller"))
				return e
			}},
			{"invalid form container", func(s *v1.Service) error {
				_, e := s.GenerateFormSignature(context.Background(), "bucket/name", signingInput())
				return e
			}},
			{"invalid form prefix", func(s *v1.Service) error {
				x := signingInput()
				x.ObjectPrefix = "../o"
				_, e := s.GenerateFormSignature(context.Background(), "bucket", x)
				return e
			}},
			{"form limit", func(s *v1.Service) error {
				x := signingInput()
				x.MaxFileSize = 0
				_, e := s.GenerateFormSignature(context.Background(), "bucket", x)
				return e
			}},
			{"form timeout", func(s *v1.Service) error {
				x := signingInput()
				x.Timeout = 0
				_, e := s.GenerateFormSignature(context.Background(), "bucket", x)
				return e
			}},
			{"redirect trailing hyphen", func(s *v1.Service) error {
				x := signingInput()
				x.RedirectURL = "done-"
				_, e := s.GenerateFormSignature(context.Background(), "bucket", x)
				return e
			}},
			{"redirect control", func(s *v1.Service) error {
				x := signingInput()
				x.RedirectURL = "done\n"
				_, e := s.GenerateFormSignature(context.Background(), "bucket", x)
				return e
			}},
			{"redirect UTF8 bytes", func(s *v1.Service) error {
				x := signingInput()
				x.RedirectURL = strings.Repeat("한", 1366)
				_, e := s.GenerateFormSignature(context.Background(), "bucket", x)
				return e
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.run(v1.New(signingClientNoHTTP(t)))
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("err=%v", err)
				}
			})
		}
		for _, endpoint := range []string{"https://swift.invalid/v1/AUTH_a//", "https://swift.invalid/reverse//v1/AUTH_a/", "https://swift.invalid/v1/%FF/", "https://swift.invalid/v1/../AUTH_a/"} {
			t.Run(endpoint, func(t *testing.T) {
				client := signingClientNoHTTP(t)
				client.Endpoint = endpoint
				result, err := v1.New(client).GenerateFormSignature(context.Background(), "bucket", signingInput(), v1.WithGenerateFormSignatureKey([]byte("key")))
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("result=%+v err=%v", result, err)
				}
			})
		}
		boundary := signingInput()
		boundary.RedirectURL = strings.Repeat("a", 4096)
		result, err := v1.New(signingClientNoHTTP(t)).GenerateFormSignature(context.Background(), "bucket", boundary, v1.WithGenerateFormSignatureKey([]byte{'k', 0, 0xff}))
		if err != nil || result == nil {
			t.Fatalf("4096 bytes/binary key result=%+v err=%v", result, err)
		}
	})
	t.Run("source and context validated before callback", func(t *testing.T) {
		calls := 0
		cb := func(cfg *v1.GenerateTempURLOpts) error { calls++; return nil }
		result, err := v1.New(signingClientNoHTTP(t)).GenerateTempURL(nil, signingPath, 60, "GET", cb)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("nilctx result=%+v err=%v callbacks=%d", result, err, calls)
		}
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause := errors.New("custom preflight cancellation")
		cancel(cause)
		result, err = v1.New(signingClientNoHTTP(t)).GenerateTempURL(ctx, signingPath, 60, "GET", cb)
		if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 0 {
			t.Fatalf("cancel result=%+v err=%v calls=%d", result, err, calls)
		}
		client := signingClientNoHTTP(t)
		client.MoreHeaders = map[string]string{"X-Newest": "true"}
		result, err = v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET", cb)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("source result=%+v err=%v calls=%d", result, err, calls)
		}
		callbackErr := errors.New("callback stopped")
		client = signingClientNoHTTP(t)
		result, err = v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET", func(cfg *v1.GenerateTempURLOpts) error {
			client.Endpoint = "https://changed.invalid/v1/a/"
			return callbackErr
		})
		if result != nil || !errors.Is(err, callbackErr) || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("callback result=%+v err=%v", result, err)
		}
	})
}

func TestTempURLSigningContractsSourceAndContext(t *testing.T) {
	t.Run("captured ordinary headers live token and early clock", func(t *testing.T) {
		calls := 0
		stamp := signingInstant
		sourceHeaders := map[string]string{"X-Capture": "source"}
		optionHeaders := map[string]string{"X-Call": "option"}
		var client *gophercloud.ServiceClient
		client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			tempKeyNoBody(t, r)
			if r.Header.Get("X-Capture") != "source" || r.Header.Get("X-Call") != "option" {
				t.Errorf("phase%d headers=%v", calls, r.Header)
			}
			token := "token-one"
			if calls == 2 {
				token = "token-two"
			}
			if r.Header.Get("X-Auth-Token") != token {
				t.Errorf("phase%d token=%q", calls, r.Header.Get("X-Auth-Token"))
			}
			body := &tempKeyBody{Reader: strings.NewReader("clock-phase")}
			headers := http.Header{}
			if calls == 1 {
				body.onClose = func() {
					sourceHeaders["X-Capture"] = "later"
					optionHeaders["X-Call"] = "later"
					stamp = time.Unix(5, 0)
					client.ProviderClient.SetToken("token-two")
				}
			} else {
				headers.Set("X-Account-Meta-Temp-Url-Key", "test-secret")
			}
			return tempKeyResponse(204, body, headers), nil
		})
		client.MoreHeaders = sourceHeaders
		full := v1.WithGenerateFormSignatureOpts(v1.GenerateFormSignatureOpts{Timestamp: &stamp, Headers: optionHeaders})
		result, err := v1.New(client).GenerateFormSignature(context.Background(), "bucket", signingInput(), full)
		if err != nil || result == nil || result.Expires != 1700000060 || calls != 2 || result.Discovery == nil || result.Discovery.Container == nil || result.Discovery.Account == nil {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
		// Result proof owns its raw header independently of the canonical key projection.
		result.Discovery.Account.Header.Set("X-Account-Meta-Temp-Url-Key", "changed-result-header")
		if string(result.Discovery.Key) != "test-secret" || result.Discovery.Account.Metadata.Values["temp-url-key"] != "test-secret" {
			t.Fatalf("discovery aliases=%+v", result.Discovery)
		}
	})
	t.Run("all captured identity fields guarded after accepted body", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			change func(*gophercloud.ServiceClient, *v1.Service)
		}{
			{"Endpoint", func(c *gophercloud.ServiceClient, s *v1.Service) { c.Endpoint = "https://other.invalid/v1/a/" }},
			{"ResourceBase", func(c *gophercloud.ServiceClient, s *v1.Service) { c.ResourceBase = tempKeyBase + "changed/" }},
			{"Type", func(c *gophercloud.ServiceClient, s *v1.Service) { c.Type = "image" }},
			{"Microversion", func(c *gophercloud.ServiceClient, s *v1.Service) { c.Microversion = "9.9" }},
			{"provider", func(c *gophercloud.ServiceClient, s *v1.Service) {
				c.ProviderClient = signingClientNoHTTP(t).ProviderClient
			}},
			{"Accounts", func(c *gophercloud.ServiceClient, s *v1.Service) { s.Accounts = accounts.New(c) }},
			{"Containers", func(c *gophercloud.ServiceClient, s *v1.Service) { s.Containers = containers.New(c) }},
			{"Objects", func(c *gophercloud.ServiceClient, s *v1.Service) { s.Objects = objects.New(c) }},
			{"Swauth", func(c *gophercloud.ServiceClient, s *v1.Service) { s.Swauth = swauth.New(c) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				calls := 0
				var client *gophercloud.ServiceClient
				var service *v1.Service
				body := &tempKeyBody{Reader: strings.NewReader("owned-identity")}
				body.onClose = func() { tc.change(client, service) }
				client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
					calls++
					return tempKeyResponse(204, body, http.Header{"X-Account-Meta-Temp-Url-Key": {"test-secret"}}), nil
				})
				service = v1.New(client)
				result, err := service.GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLTimestamp(signingInstant))
				signingUnsignedTemp(t, result)
				tempKeyProof(t, err, 204, "owned-identity")
				if !errors.Is(err, resource.ErrInvalidOption) || result == nil || result.Discovery.Key != nil || result.Discovery.Secondary || result.Discovery.FromContainer || result.Discovery.Account.Metadata != nil || calls != 1 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v calls=%d closes=%d", result, err, calls, body.closes.Load())
				}
			})
		}
		calls := 0
		var client *gophercloud.ServiceClient
		body := &tempKeyBody{Reader: strings.NewReader("container-identity")}
		body.onClose = func() { client.ResourceBase = tempKeyBase + "drift/" }
		client = tempKeyClient(func(r *http.Request) (*http.Response, error) { calls++; return tempKeyResponse(204, body, nil), nil })
		form, err := v1.New(client).GenerateFormSignature(context.Background(), "bucket", signingInput())
		signingUnsignedForm(t, form)
		tempKeyProof(t, err, 204, "container-identity")
		if form == nil || form.Discovery.Container.Metadata != nil || form.Discovery.Account != nil || calls != 1 {
			t.Fatalf("form=%+v err=%v calls=%d", form, err, calls)
		}
	})
	t.Run("read close and custom cancellation preserve accepted evidence", func(t *testing.T) {
		readErr := errors.New("read-partial")
		closeErr := errors.New("close-error")
		cause := errors.New("custom-body-cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		reads := 0
		body := &tempKeyBody{Reader: tempKeyReader(func(p []byte) (int, error) { reads++; return copy(p, "partial-body"), readErr }), closeErr: closeErr, onClose: func() { cancel(cause) }}
		calls := 0
		hooks := 0
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			return tempKeyResponse(204, body, http.Header{"X-Account-Meta-Temp-Url-Key": {"test-secret"}}), nil
		})
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nil
		}
		result, err := v1.New(client).GenerateTempURL(ctx, signingPath, 60, "GET", v1.WithGenerateTempURLTimestamp(signingInstant))
		signingUnsignedTemp(t, result)
		tempKeyProof(t, err, 204, "partial-body")
		for _, target := range []error{readErr, closeErr, context.Canceled, cause} {
			if !errors.Is(err, target) {
				t.Errorf("missing %v in %v", target, err)
			}
		}
		if result == nil || result.Discovery.Account.Metadata != nil || result.Discovery.Key != nil || calls != 1 || hooks != 0 || reads != 1 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v calls=%d hooks=%d reads=%d closes=%d", result, err, calls, hooks, reads, body.closes.Load())
		}
		closeOnly := &tempKeyBody{Reader: strings.NewReader("full-but-close-failed"), closeErr: closeErr}
		formClient := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			return tempKeyResponse(204, closeOnly, http.Header{"X-Container-Meta-Temp-Url-Key": {"key"}}), nil
		})
		form, err := v1.New(formClient).GenerateFormSignature(context.Background(), "bucket", signingInput())
		signingUnsignedForm(t, form)
		tempKeyProof(t, err, 204, "full-but-close-failed")
		if form == nil || form.Discovery.Key != nil || form.Discovery.Container.Metadata != nil || !errors.Is(err, closeErr) || closeOnly.closes.Load() != 1 {
			t.Fatalf("form=%+v err=%v closes=%d", form, err, closeOnly.closes.Load())
		}
	})
	t.Run("no stale selection after account-phase source drift", func(t *testing.T) {
		calls := 0
		var client *gophercloud.ServiceClient
		client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			body := &tempKeyBody{Reader: strings.NewReader("phase-proof")}
			headers := http.Header{}
			if calls == 2 {
				headers.Set("X-Account-Meta-Temp-Url-Key-2", "test-secret")
				body.onClose = func() { client.Microversion = "changed" }
			}
			return tempKeyResponse(204, body, headers), nil
		})
		result, err := v1.New(client).GenerateFormSignature(context.Background(), "bucket", signingInput(), v1.WithGenerateFormSignatureTimestamp(signingInstant))
		signingUnsignedForm(t, result)
		tempKeyProof(t, err, 204, "phase-proof")
		if !errors.Is(err, resource.ErrInvalidOption) || result == nil || result.Discovery.Container == nil || result.Discovery.Container.Metadata == nil || result.Discovery.Account == nil || result.Discovery.Account.Metadata != nil || result.Discovery.Key != nil || result.Discovery.FromContainer || result.Discovery.Secondary || calls != 2 {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
	})
}

func TestTempURLSigningContractsNativePolicy(t *testing.T) {
	t.Run("native CreateTempURL remains primary-first independent ABI", func(t *testing.T) {
		client := signingClientNoHTTP(t)
		client.Endpoint = "https://swift.invalid/v1/AUTH_demo/"
		client.ResourceBase = ""
		native, err := objects.New(client).CreateTempURL(context.Background(), "photos", "report.txt", nativeobjects.CreateTempURLOpts{Method: nativeobjects.GET, TTL: 60, Timestamp: signingInstant, TempURLKey: "test-secret"})
		if err != nil {
			t.Fatal(err)
		}
		u, e := url.Parse(native)
		if e != nil || u.Host != "swift.invalid" || u.Path != signingPath || u.Query().Get("temp_url_sig") != signingDefaultMAC {
			t.Fatalf("native=%q err=%v", native, e)
		}
		calls := 0
		client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.String() != tempKeyBase+"photos" {
				t.Errorf("native lookup=%s", r.URL)
			}
			return tempKeyWire(204, "", http.Header{"X-Container-Meta-Temp-Url-Key": {"test-secret"}, "X-Container-Meta-Temp-Url-Key-2": {"different-secondary"}}), nil
		})
		native, err = objects.New(client).CreateTempURL(context.Background(), "photos", "report.txt", nativeobjects.CreateTempURLOpts{Method: nativeobjects.GET, TTL: 60, Timestamp: signingInstant})
		if err != nil || native == "" || calls != 1 {
			t.Fatalf("native=%q err=%v calls=%d", native, err, calls)
		}
		var digestErr nativeobjects.ErrTempURLDigestNotValid
		_, err = objects.New(signingClientNoHTTP(t)).CreateTempURL(context.Background(), "photos", "report.txt", nativeobjects.CreateTempURLOpts{Method: nativeobjects.GET, TTL: 60, Timestamp: signingInstant, TempURLKey: "explicit", Digest: "invalid"})
		if !errors.As(err, &digestErr) {
			t.Fatalf("native digest err=%v", err)
		}
	})
	t.Run("configured prebody retries transport causes and reauth fields", func(t *testing.T) {
		calls, hooks := 0, 0
		rejected := &tempKeyBody{Reader: strings.NewReader("retry503")}
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			tempKeyNoBody(t, r)
			if calls == 1 {
				return tempKeyResponse(503, rejected, nil), nil
			}
			return tempKeyWire(204, "after-retry", http.Header{"X-Account-Meta-Temp-Url-Key": {"test-secret"}}), nil
		})
		client.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
			hooks++
			var n gophercloud.ErrUnexpectedResponseCode
			if method != "HEAD" || target != tempKeyAccount || !errors.As(original, &n) || n.Actual != 503 {
				t.Errorf("hook method=%s target=%s err=%v", method, target, original)
			}
			return nil
		}
		result, err := v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLTimestamp(signingInstant))
		signingTempOK(t, result, err, signingDefaultMAC)
		if calls != 2 || hooks != 1 || rejected.closes.Load() != 1 {
			t.Fatalf("calls=%d hooks=%d closes=%d", calls, hooks, rejected.closes.Load())
		}
		transportErr := errors.New("transport-failure")
		nested404 := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
		calls, hooks = 0, 0
		client = tempKeyClient(func(r *http.Request) (*http.Response, error) { calls++; return nil, transportErr })
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nested404
		}
		result, err = v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET")
		if result != nil || !errors.Is(err, transportErr) || !errors.Is(err, nested404) || calls != 1 || hooks != 1 {
			t.Fatalf("result=%+v err=%v calls=%d hooks=%d", result, err, calls, hooks)
		}
		reauthErr := errors.New("reauth-failed")
		calls = 0
		client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			return tempKeyWire(401, "unauthorized", nil), nil
		})
		client.ReauthFunc = func(context.Context) error { return reauthErr }
		result, err = v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET")
		var reauth *gophercloud.ErrUnableToReauthenticate
		if result != nil || !errors.As(err, &reauth) || reauth.ErrOriginal == nil || reauth.ErrReauth != reauthErr || errors.Is(err, reauthErr) || calls != 1 {
			t.Fatalf("result=%+v reauth=%+v err=%v calls=%d", result, reauth, err, calls)
		}
	})
	t.Run("bodyless ownership and actual status gates", func(t *testing.T) {
		for _, change := range []func(*gophercloud.RequestOpts){
			func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false },
			func(o *gophercloud.RequestOpts) { var v any; o.JSONResponse = &v },
			func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("injected") },
			func(o *gophercloud.RequestOpts) { var typedNil *string; o.JSONBody = typedNil },
		} {
			calls, hooks := 0, 0
			body := &tempKeyBody{Reader: strings.NewReader("original503")}
			client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				tempKeyNoBody(t, r)
				return tempKeyResponse(503, body, nil), nil
			})
			client.RetryFunc = func(ctx context.Context, method, target string, o *gophercloud.RequestOpts, original error, count uint) error {
				hooks++
				change(o)
				return nil
			}
			result, err := v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET")
			var native gophercloud.ErrUnexpectedResponseCode
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || calls != 1 || hooks != 1 || body.closes.Load() != 1 {
				t.Fatalf("result=%+v err=%v calls=%d hooks=%d closes=%d", result, err, calls, hooks, body.closes.Load())
			}
		}
		calls := 0
		body := &tempKeyBody{Reader: strings.NewReader("native-accepted-wrong-code")}
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
		result, err := v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET")
		var native gophercloud.ErrUnexpectedResponseCode
		if result != nil || !errors.As(err, &native) || native.Actual != 200 || !reflect.DeepEqual(native.Expected, []int{204}) || string(native.Body) != "native-accepted-wrong-code" || body.closes.Load() != 1 || calls != 2 {
			t.Fatalf("result=%+v native=%+v err=%v calls=%d closes=%d", result, native, err, calls, body.closes.Load())
		}
	})
	t.Run("advanced wholesale headers and fixed redirect policy", func(t *testing.T) {
		calls := 0
		client := tempKeyClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				if r.Header.Get("X-Capture") != "source" || r.Header.Get("X-Advanced") != "" {
					t.Errorf("first headers=%v", r.Header)
				}
				return tempKeyWire(503, "retry", nil), nil
			}
			if r.Header.Get("X-Capture") != "" || r.Header.Get("X-Advanced") != "native" {
				t.Errorf("advanced headers=%v", r.Header)
			}
			return tempKeyWire(204, "", http.Header{"X-Account-Meta-Temp-Url-Key": {"test-secret"}}), nil
		})
		client.MoreHeaders = map[string]string{"X-Capture": "source"}
		client.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
			opts.MoreHeaders = map[string]string{"X-Advanced": "native"}
			return nil
		}
		result, err := v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLTimestamp(signingInstant))
		signingTempOK(t, result, err, signingDefaultMAC)
		if calls != 2 || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Capture": "source"}) {
			t.Fatalf("calls=%d source=%v", calls, client.MoreHeaders)
		}
		for _, same := range []bool{true, false} {
			calls = 0
			client = tempKeyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					target := tempKeyAccount
					if !same {
						target += "retarget"
					}
					return tempKeyWire(307, "redirect", http.Header{"Location": {target}}), nil
				}
				return tempKeyWire(204, "", http.Header{"X-Account-Meta-Temp-Url-Key": {"test-secret"}}), nil
			})
			result, err = v1.New(client).GenerateTempURL(context.Background(), signingPath, 60, "GET", v1.WithGenerateTempURLTimestamp(signingInstant))
			if same {
				signingTempOK(t, result, err, signingDefaultMAC)
				if calls != 2 {
					t.Fatalf("same redirect calls=%d", calls)
				}
			} else if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatalf("retarget result=%+v err=%v calls=%d", result, err, calls)
			}
		}
	})
}
