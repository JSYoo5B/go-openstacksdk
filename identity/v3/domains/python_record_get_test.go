package domains_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonDomainGetRecordNullableRawOptionsSeedAndActualEvidence(t *testing.T) {
	for _, code := range []int{100, 199, 200, 203, 399} {
		calls := 0
		body := `{"domain":{"name":"wire","enabled":false,"is_enabled":true,"options":{"n":9007199254740993},"description":false,"links":{"self":"foreign"},"location":{"cloud":"forged"},"x_extension":true}}`
		api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.EscapedPath() != "/keystone/v3/domains/folder%2Fname%25" || r.URL.RawQuery != "" || r.Method != "GET" || r.Body != nil {
				t.Fatal(r.URL, r.Body)
			}
			return domainRecordWire(code, body)
		})
		row, err := api.GetRecord(context.Background(), "folder/name%")
		if err != nil || row == nil || calls != 1 || row.StatusCode != code || row.Header.Get("X-Request-Id") != "domain-proof" || string(row.Envelope) != body {
			t.Fatal(code, row, err, calls)
		}
		for key, want := range map[string]string{"id": `"folder/name%"`, "is_enabled": "true", "options": `{"n":9007199254740993}`, "description": "false", "location": "null"} {
			domainRecordField(t, row, key, want)
		}
		if string(row.Wire.Body["x_extension"]) != "true" || string(row.Wire.Body["location"]) != `{"cloud":"forged"}` {
			t.Fatal(row)
		}
		clone := row.Clone()
		clone.Resource.Body["id"][1] = 'X'
		clone.Header.Set("X-Request-Id", "caller")
		if string(row.Resource.Body["id"]) != `"folder/name%"` || row.Header.Get("X-Request-Id") != "domain-proof" {
			t.Fatal(row)
		}
	}
	for _, body := range []string{`{"domain":{"id":null,"options":null,"enabled":null}}`, `{"domain":{"options":5}}`, `not-json`, `{"description":"flat"}`} {
		api, _ := domainRecordAPI(t, func(*http.Request) *http.Response { return domainRecordWire(199, body) })
		row, err := api.GetRecord(context.Background(), "fixed")
		if err != nil || row == nil {
			t.Fatal(body, row, err)
		}
		if strings.Contains(body, `"id":null`) {
			domainRecordField(t, row, "id", "null")
			domainRecordField(t, row, "options", "null")
		} else {
			domainRecordField(t, row, "id", `"fixed"`)
		}
		if strings.Contains(body, `"options":5`) {
			domainRecordField(t, row, "options", "{}")
		}
	}
}
func TestPythonDomainGetRecordRejectedNeverFallsBackAndBadEnvelopeKeepsProof(t *testing.T) {
	for _, lane := range []struct {
		code int
		body string
	}{{404, `{"error":"missing"}`}, {403, `{"error":"forbidden"}`}, {200, `{"domain":false}`}} {
		calls := 0
		api, _ := domainRecordAPI(t, func(*http.Request) *http.Response { calls++; return domainRecordWire(lane.code, lane.body) })
		row, err := api.GetRecord(context.Background(), "fixed")
		if err == nil || calls != 1 || row != nil && row.Resource != nil {
			t.Fatal(row, err, calls)
		}
		if lane.code >= 400 {
			var proof gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &proof) || proof.Actual != lane.code || proof.Method != "GET" || !strings.Contains(proof.URL, "/domains/fixed") || string(proof.Body) != lane.body || proof.ResponseHeader.Get("X-Request-Id") != "domain-proof" {
				t.Fatal(err, proof)
			}
		} else {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != lane.body || proof.Header.Get("X-Request-Id") != "domain-proof" {
				t.Fatal(err, proof)
			}
		}
	}
}
func TestPythonDomainResolveRecordClosedCleanFallbackAbsentDuplicatesAndLateProof(t *testing.T) {
	for _, mode := range []string{"absent", "match", "duplicate", "late"} {
		for _, code := range []int{400, 403, 404} {
			t.Run(fmt.Sprintf("%s/%d", mode, code), func(t *testing.T) {
				calls := 0
				api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
					calls++
					if calls == 1 {
						return domainRecordWire(code, `{"error":"fallback"}`)
					}
					if r.URL.Query().Get("name") != "alias" || r.URL.Path != "/keystone/v3/domains" {
						t.Fatal(r.URL)
					}
					if mode == "absent" {
						return domainRecordWire(200, `{"domains":[]}`)
					}
					if calls == 2 {
						next := ""
						if mode != "match" {
							next = `,"next":"?marker=one&name=alias"`
						}
						return domainRecordWire(200, `{"domains":[{"id":"one","name":"alias","options":{"keep":1}}]`+next+`}`)
					}
					if mode == "duplicate" {
						return domainRecordWire(199, `{"domains":[{"id":"one","name":"alias"}]}`)
					}
					return domainRecordWire(503, `{"error":"late"}`)
				})
				row, err := api.ResolveRecord(context.Background(), "alias")
				switch mode {
				case "absent":
					if row != nil || err != nil || calls != 2 {
						t.Fatal(row, err, calls)
					}
				case "match":
					if err != nil || row == nil || calls != 2 {
						t.Fatal(row, err, calls)
					}
					domainRecordField(t, row, "options", `{"keep":1}`)
				case "duplicate":
					if row != nil || !errors.Is(err, resource.ErrAmbiguous) || calls != 3 {
						t.Fatal(row, err, calls)
					}
				case "late":
					var proof gophercloud.ErrUnexpectedResponseCode
					if row != nil || !errors.As(err, &proof) || proof.Actual != 503 || string(proof.Body) != `{"error":"late"}` || calls != 3 {
						t.Fatal(row, err, proof, calls)
					}
				}
			})
		}
	}
}
func TestPythonDomainGetRecordOwnedOptionsLocationAndStickyGuard(t *testing.T) {
	calls := 0
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Header.Get("X-Owned") != "owned" || r.Header.Get("OpenStack-API-Version") != "identity 3.7" {
			t.Fatal(r.Header)
		}
		return domainRecordWire(203, `{"domain":{"id":"wire","project_id":"foreign","location":{"cloud":"forged"}}}`)
	})
	name, project := "configured", "current"
	location := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &project}}
	headers := map[string]string{"X-Owned": "owned"}
	version := "3.7"
	opt := domains.WithRecordGetOpts(domains.RecordGetOpts{Headers: headers, Microversion: &version, Location: &location})
	headers["X-Owned"] = "changed"
	version = "9"
	name = "changed"
	project = "changed"
	row, err := api.GetRecord(context.Background(), "fixed", opt)
	if err != nil || calls != 1 {
		t.Fatal(row, err, calls)
	}
	var view map[string]json.RawMessage
	_ = json.Unmarshal(row.Resource.Body["location"], &view)
	var scope map[string]json.RawMessage
	_ = json.Unmarshal(view["project"], &scope)
	if string(view["cloud"]) != `"configured"` || string(scope["id"]) != `"scope"` || string(scope["name"]) != `"current"` {
		t.Fatal(row)
	}
	for _, mode := range []string{"source", "cancel", "nil"} {
		t.Run(mode, func(t *testing.T) {
			before := calls
			seen := 0
			client := api.RawClient()
			endpoint := client.Endpoint
			defer func() { client.Endpoint = endpoint }()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("domain get custom cancel")
			first := domains.RecordGetOption(func(*domains.RecordGetOpts) error {
				if mode == "source" {
					client.Endpoint += "changed"
				} else {
					cancel(cause)
				}
				return nil
			})
			if mode == "nil" {
				first = nil
			}
			_, err := api.GetRecord(ctx, "fixed", first, domains.RecordGetOption(func(*domains.RecordGetOpts) error { seen++; client.Endpoint = endpoint; return nil }))
			var proof *resource.ResponseError
			if err == nil || seen != 0 || calls != before || errors.As(err, &proof) {
				t.Fatal(err, seen, calls, proof)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}
func TestPythonDomainResolveRecordGetOnlyVersionAndProfileIsolation(t *testing.T) {
	var versions, paths []string
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		paths = append(paths, r.URL.Path)
		versions = append(versions, r.Header.Get("OpenStack-API-Version"))
		if len(paths) == 1 {
			return domainRecordWire(404, `{}`)
		}
		return domainRecordWire(200, `{"domains":[{"id":"one","name":"alias"}]}`)
	})
	api.RawClient().Microversion = "3.5"
	row, err := api.ResolveRecord(context.Background(), "alias", domains.WithRecordGetMicroversion("3.9"))
	if err != nil || row == nil || !reflect.DeepEqual(versions, []string{"identity 3.9", "identity 3.5"}) {
		t.Fatal(row, err, versions)
	}
	if _, err := api.AllRecords(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 || paths[2] != "/keystone/v3/domains" {
		t.Fatal(paths)
	}
}
func TestPythonDomainGetRecordPhysicalFaultsKeepActualProofAndNoRetry(t *testing.T) {
	for _, code := range []int{200, 400, 403, 404, 401, 429, 503} {
		for _, phase := range []string{"read", "close", "source", "cancel"} {
			t.Run(fmt.Sprintf("%d/%s", code, phase), func(t *testing.T) {
				calls, closes, hooks := 0, 0, 0
				cause := errors.New("domain physical fault")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var api *domains.API
				api, fixture := domainRecordAPI(t, func(*http.Request) *http.Response {
					calls++
					b := &domainFaultBody{closes: &closes}
					switch phase {
					case "read":
						b.readErr = cause
					case "close":
						b.closeErr = cause
					case "source":
						b.read = func() { api.RawClient().Endpoint += "changed" }
						b.close = func() { api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().Endpoint, "changed") }
					case "cancel":
						b.read = func() { cancel(cause) }
					}
					response := domainRecordWire(code, "")
					response.Body = b
					return response
				})
				fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return nil
				}
				fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
				fixture.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
				fixture.Provider.MaxBackoffRetries = 1
				row, err := api.GetRecord(ctx, "fixed")
				if err == nil || row != nil || calls != 1 || closes != 1 || hooks != 0 {
					t.Fatal(row, err, calls, closes, hooks)
				}
				if code < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != code || proof.Header.Get("X-Request-Id") != "domain-proof" || string(proof.Body) != `{"domains":` {
						t.Fatal(err, proof)
					}
				} else {
					original := err
					if code == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
							t.Fatal(err, auth)
						}
						original = auth.ErrOriginal
					}
					var proof gophercloud.ErrUnexpectedResponseCode
					if !errors.As(original, &proof) || proof.Actual != code || proof.ResponseHeader.Get("X-Request-Id") != "domain-proof" || string(proof.Body) != `{"domains":` || proof.Method != "GET" || !strings.Contains(proof.URL, "/keystone/v3/domains/fixed") {
						t.Fatal(err, proof)
					}
				}
				if code != 401 && phase != "source" && !errors.Is(err, cause) {
					t.Fatal(err, cause)
				}
				if code == 401 {
					var reauth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &reauth) || reauth.ErrReauth == nil {
						t.Fatal(err, reauth)
					}
					// The native 401 wrapper has no Unwrap: its original response
					// and failed reauthentication are independently inspectable.
					// Gophercloud stores ErrOriginal as a response-error value.
					var original gophercloud.ErrUnexpectedResponseCode
					if !errors.As(reauth.ErrOriginal, &original) || original.Actual != 401 || original.Method != "GET" || !strings.Contains(original.URL, "/domains/fixed") || string(original.Body) != `{"domains":` || original.ResponseHeader.Get("X-Request-Id") != "domain-proof" {
						t.Fatal(err, original)
					}
					if phase == "source" {
						if !errors.Is(reauth.ErrReauth, resource.ErrInvalidOption) {
							t.Fatal(err, reauth.ErrReauth)
						}
					} else if !errors.Is(reauth.ErrReauth, cause) {
						t.Fatal(err, reauth.ErrReauth, cause)
					}
					if phase == "cancel" && !errors.Is(reauth.ErrReauth, context.Canceled) {
						t.Fatal(err, reauth.ErrReauth)
					}
				}
			})
		}
	}
}
func TestPythonDomainGetRecordCleanRetryPreservesLiveAuthAndFrozenRequest(t *testing.T) {
	for _, code := range []int{401, 503, 429, 498} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, reauth, retry, backoff := 0, 0, 0, 0
			api, fixture := domainRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.Method != "GET" || r.Body != nil || r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
					t.Fatal(r.Header, r.Body)
				}
				if calls == 1 {
					if r.Header.Get("X-Auth-Token") != "test-token" {
						t.Fatal(r.Header)
					}
					return domainRecordWire(code, `{"error":"native clean"}`)
				}
				if r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Fatal(r.Header)
				}
				return domainRecordWire(203, `{"domain":{"id":"a"}}`)
			})
			api.RawClient().MoreHeaders = map[string]string{"X-Snapshot": "before"}
			update := func() {
				fixture.Provider.SetToken("refreshed")
				api.RawClient().MoreHeaders["X-Snapshot"] = "after"
				api.RawClient().MoreHeaders["X-Late"] = "after"
			}
			fixture.Provider.ReauthFunc = func(context.Context) error { reauth++; update(); return nil }
			fixture.Provider.RetryFunc = func(ctx context.Context, method, u string, opts *gophercloud.RequestOpts, _ error, count uint) error {
				retry++
				update()
				return nil
			}
			fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				update()
				return nil
			}
			fixture.Provider.MaxBackoffRetries = 1
			rows, err := api.GetRecord(context.Background(), "fixed")
			if err != nil || rows == nil || rows.StatusCode != 203 || calls != 2 {
				t.Fatal(rows, err, calls)
			}
			if code == 401 && (reauth != 1 || retry != 0 || backoff != 0) || code == 503 && (retry != 1 || reauth != 0 || backoff != 0) || (code == 429 || code == 498) && (backoff != 1 || reauth != 0 || retry != 0) {
				t.Fatal(reauth, retry, backoff)
			}
		})
	}
}
