package blockstorage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestGetVolumeCanonicalEmptyMemberDoesNotAdoptRequestIdentityAndRejectsNoncanonicalEnvelopes(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		accepted   bool
	}{
		{"empty object", `{"volume":{}}`, true},
		{"missing envelope", `{}`, false},
		{"case alias", `{"Volume":{}}`, false},
		{"null envelope", `{"volume":null}`, false},
		{"array envelope", `{"volume":[]}`, false},
		{"malformed JSON", `{"volume":`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) != 1 || r.Method != http.MethodGet || r.URL.Path != volumeSearchTestBase+"volumes/requested" || r.URL.RawQuery != "" {
					t.Error("member response initiated fallback/replay", r.Method, r.URL)
				}
				w.Header().Set("X-Proof", "canonical-member")
				testcloud.JSON(w, 200, tc.body)
			})
			result, err := blockstorage.GetVolume(context.Background(), client, blockstorage.GetVolumeRequest{NameOrID: "requested"})
			if result == nil || calls.Load() != 1 || result.Observed == nil || result.Observed.StatusCode != 200 || result.Observed.Header.Get("X-Proof") != "canonical-member" || string(result.Observed.Body) != tc.body || len(result.Pages) != 0 {
				t.Fatal(result, err, calls.Load())
			}
			if !tc.accepted {
				var response *resource.ResponseError
				if err == nil || !errors.As(err, &response) || response.StatusCode != 200 || string(response.Body) != tc.body || response.Header.Get("X-Proof") != "canonical-member" || result.Value != nil || result.Volume != nil {
					t.Fatal(result, err)
				}
				return
			}
			if err != nil || result.Volume == nil || len(result.Volume.Body) != 0 || result.Volume.StatusCode != 200 {
				t.Fatal(result, err)
			}
			fields := volumeSearchTestFields(t, result.Value)
			if len(fields) != 38 || string(fields["id"]) != "null" || string(fields["name"]) != "null" {
				t.Fatal("request identity fabricated into accepted normalized resource", string(result.Value))
			}
			if _, present := result.Volume.Body["id"]; present {
				t.Fatal("request identity fabricated into physical raw resource", result.Volume.Body)
			}
			result.Observed.Body[0] = '!'
			result.Observed.Header.Set("X-Proof", "caller")
			if result.Volume.Header.Get("X-Proof") != "canonical-member" || string(fields["id"]) != "null" {
				t.Fatal("physical and normalized evidence aliases", result)
			}
		})
	}
}

func TestGetVolumeNativeMemberAuthenticationRetriesKeepFixedBodylessRouteAndSingleAcceptedProof(t *testing.T) {
	t.Run("reauth backoff retry", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := volumeSearchTestClient(cloud)
		client.Microversion = "3.60"
		client.MoreHeaders = map[string]string{"x-owned": "entry"}
		target := cloud.Server.URL + volumeSearchTestBase + "volumes/requested"
		var calls, reauths, backoffs, retries, callbacks atomic.Int32
		cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("refreshed"); return nil }
		cloud.Provider.RetryBackoffFunc = func(_ context.Context, code *gophercloud.ErrUnexpectedResponseCode, _ error, count uint) error {
			backoffs.Add(1)
			if code.Actual != 429 || count != 1 {
				t.Error(code, count)
			}
			return nil
		}
		cloud.Provider.RetryFunc = func(_ context.Context, method, endpoint string, opts *gophercloud.RequestOpts, err error, count uint) error {
			retries.Add(1)
			if !gophercloud.ResponseCodeIs(err, 503) {
				return err
			}
			if method != http.MethodGet || endpoint != target || count != 2 || opts.JSONBody != nil || opts.RawBody != nil || opts.JSONResponse != nil || !opts.KeepResponseBody || !reflect.DeepEqual(opts.OkCodes, []int{200}) {
				t.Error(method, endpoint, count, opts)
			}
			if opts.MoreHeaders == nil {
				opts.MoreHeaders = map[string]string{}
			}
			opts.MoreHeaders["X-Retry"] = "native"
			return nil
		}
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			n := calls.Add(1)
			token := "refreshed"
			if n == 1 {
				token = "live-entry"
			}
			if r.Method != http.MethodGet || r.URL.EscapedPath() != volumeSearchTestBase+"volumes/requested" || r.URL.RawQuery != "" || r.Header.Get("X-Owned") != "entry" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != "volume 3.60" {
				t.Error(r.Method, r.URL, r.Header)
			}
			if r.Body != nil {
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 {
					t.Error("member sent a request body", body, err)
				}
			}
			switch n {
			case 1:
				testcloud.JSON(w, 401, `{"error":"expired"}`)
			case 2:
				testcloud.JSON(w, 429, `{"error":"backoff"}`)
			case 3:
				testcloud.JSON(w, 503, `{"error":"retry"}`)
			case 4:
				if r.Header.Get("X-Retry") != "native" {
					t.Error("native retry header lost", r.Header)
				}
				w.Header().Set("X-Proof", "accepted-member")
				testcloud.JSON(w, 200, `{"volume":{"id":"wire-id","unknown":[9007199254740993]}}`)
			default:
				t.Error("SDK replayed accepted member", r.URL)
				w.WriteHeader(500)
			}
		})
		result, err := blockstorage.GetVolume(context.Background(), client, blockstorage.GetVolumeRequest{NameOrID: "requested"}, func(*blockstorage.VolumeSearchOpts) error {
			callbacks.Add(1)
			client.MoreHeaders["x-owned"] = "after"
			cloud.Provider.SetToken("live-entry")
			return nil
		})
		if err != nil || result == nil || result.Volume == nil || result.Observed == nil || result.Observed.StatusCode != 200 || result.Observed.Header.Get("X-Proof") != "accepted-member" || len(result.Pages) != 0 || calls.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 || callbacks.Load() != 1 || client.MoreHeaders["x-owned"] != "after" {
			t.Fatal(result, err, calls.Load(), reauths.Load(), backoffs.Load(), retries.Load(), callbacks.Load())
		}
		if string(result.Volume.Body["id"]) != `"wire-id"` || string(result.Volume.Body["unknown"]) != "[9007199254740993]" || string(volumeSearchTestFields(t, result.Value)["id"]) != `"wire-id"` {
			t.Fatal("native replay changed accepted wire model", result)
		}
		result.Observed.Header.Set("X-Proof", "caller")
		result.Observed.Body[0] = '!'
		if result.Volume.Header.Get("X-Proof") != "accepted-member" {
			t.Fatal("accepted proof aliases raw metadata", result)
		}
	})
	for _, mode := range []string{"body", "response target", "status expansion"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var calls atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, method, endpoint string, opts *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				if method != http.MethodGet || endpoint != cloud.Server.URL+volumeSearchTestBase+"volumes/requested" {
					t.Error(method, endpoint)
				}
				switch mode {
				case "body":
					opts.JSONBody = map[string]any{"bad": true}
				case "response target":
					opts.JSONResponse = new(any)
				case "status expansion":
					opts.OkCodes = append(opts.OkCodes, 201)
				}
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != volumeSearchTestBase+"volumes/requested" {
					t.Error("native hook error fell back to list", r.URL)
				}
				w.Header().Set("X-Proof", "rejected")
				if calls.Add(1) == 1 {
					testcloud.JSON(w, 503, `{"error":"retry"}`)
				} else {
					testcloud.JSON(w, 201, `{"volume":{"id":"never-admit"}}`)
				}
			})
			result, err := blockstorage.GetVolume(context.Background(), client, blockstorage.GetVolumeRequest{NameOrID: "requested"})
			if result == nil || err == nil || result.Value != nil || result.Volume != nil || result.Observed != nil || len(result.Pages) != 0 {
				t.Fatal(result, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.ResponseHeader.Get("X-Proof") != "rejected" {
				t.Fatal("native HTTP error lost", err)
			}
			if mode == "status expansion" {
				if native.Actual != 201 || !reflect.DeepEqual(native.Expected, []int{200}) || calls.Load() != 2 {
					t.Fatal(err, calls.Load())
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || native.Actual != 503 || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
		})
	}
}
