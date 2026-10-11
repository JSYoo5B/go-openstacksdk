package waits_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func nativeSecretStore(t *testing.T, body string) *secretstores.SecretStore {
	t.Helper()
	wireValue := raw(t, body)
	// Only metadata supplies the Source seed. Convenience fields deliberately do
	// not encode a member request identity or replace original nullable values.
	return &secretstores.SecretStore{Metadata: wireValue.Metadata, ID: "convenience-id-must-not-route", Status: "typed-status-must-not-seed"}
}
func TestPythonKeymanagerSecretStoreWaitLiteralRouteRawProjectionAndDefaults(t *testing.T) {
	body := `{"status":"ACTIVE","id":null,"name":false,"created":0,"updated":{"precision":900719925474099312345},"secret_store_ref":"https://foreign.invalid/stores/path%2Ftail?query=x#frag","global_default":"False","crypto_plugin":[],"secret_store_plugin":false,"location":{"foreign":true},"self":"wire-only","x_new":9007199254740993}`
	api, client, calls := fixture(t, func(req *http.Request, _ int) *http.Response {
		if req.URL.EscapedPath() != "/barbican/v1/secret-stores/fixed%2Fid%20%252F" || req.Header.Get("OpenStack-API-Version") != "key-manager 1.1" || req.Header.Get("X-Selected") != "owned" {
			t.Fatal(req.URL, req.Header)
		}
		return wire(203, body)
	})
	value, err := api.WaitForStatus(context.Background(), waits.SecretStore("fixed/id %2F"), str("active"), waits.WithMicroversion("1.1"), waits.WithHeader("X-Selected", "owned"))
	if err != nil || value == nil || len(*calls) != 1 || value.RequestID != "fixed/id %2F" || value.Kind != "secret-stores" || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "wait-proof" || client.Microversion != "" || client.MoreHeaders != nil {
		t.Fatal(err, value, *calls, client)
	}
	for key, want := range map[string]string{"id": "null", "name": "false", "status": `"ACTIVE"`, "created_at": "0", "updated_at": `{"precision":900719925474099312345}`, "secret_store_id": `"path%2Ftail"`, "global_default": "true", "crypto_plugin": "[]", "secret_store_plugin": "false", "location": "null"} {
		field(t, value, key, want)
	}
	if len(value.Resource.Body) != 11 || value.Resource.Body["x_new"] != nil || value.Resource.Body["self"] != nil || string(value.Wire.Body["location"]) != `{"foreign":true}` || string(value.Wire.Body["global_default"]) != `"False"` {
		t.Fatal(value)
	}
	clone := value.Clone()
	clone.Resource.Body["status"][1] = 'x'
	clone.Wire.Body["name"][0] = 'x'
	clone.Envelope[0] = 'x'
	clone.Header.Set("X-Request-Id", "changed")
	if string(value.Resource.Body["status"]) != `"ACTIVE"` || string(value.Wire.Body["name"]) != "false" || string(value.Envelope) != body || value.Resource.Header.Get("X-Request-Id") != "wait-proof" || value.Wire.Header.Get("X-Request-Id") != "wait-proof" {
		t.Fatal("receipt channels alias", value)
	}
	api, _, calls = fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":"ACTIVE"}`) })
	value, err = api.WaitForStatus(context.Background(), waits.SecretStore("one"), str("active"))
	if err != nil || len(*calls) != 1 {
		t.Fatal(err, value, *calls)
	}
	for _, key := range []string{"name", "created_at", "updated_at", "secret_store_ref", "secret_store_id", "global_default", "crypto_plugin", "secret_store_plugin", "location"} {
		field(t, value, key, "null")
	}
	field(t, value, "id", `"one"`)
}

func TestPythonKeymanagerSecretStoreWaitNativeCachedSnapshotKeepsExplicitIdentityAndNoFakeEnvelope(t *testing.T) {
	seed := nativeSecretStore(t, `{"status":"ACTIVE","id":null,"name":"cached","secret_store_ref":"https://foreign.invalid/store/raw%2Fid","global_default":false}`)
	seed.Header = http.Header{"X-Original": []string{"owned"}}
	seed.StatusCode = 203
	target := waits.ForSecretStore("literal/selector", seed)
	seed.Body["status"][1] = 'x'
	seed.Body["name"][1] = 'x'
	seed.Header.Set("X-Original", "changed")
	seed.ID = "foreign"
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Fatal("cached state fetched"); return nil })
	value, err := api.WaitForStatus(context.Background(), target, str("active"), waits.WithTimeout(0), waits.WithInterval(-time.Second))
	if err != nil || value == nil || len(*calls) != 0 || value.RequestID != "literal/selector" || value.StatusCode != 203 || value.Header.Get("X-Original") != "owned" || value.Envelope != nil {
		t.Fatal(err, value, *calls)
	}
	field(t, value, "id", "null")
	field(t, value, "name", `"cached"`)
	field(t, value, "secret_store_id", `"raw%2Fid"`)
	field(t, value, "global_default", "false")
	value.Resource.Body["status"][1] = 'x'
	value.Wire.Body["name"][1] = 'x'
	value.Header.Set("X-Original", "consumer")
	value, err = api.WaitForStatus(context.Background(), target, str("active"))
	if err != nil || value.Header.Get("X-Original") != "owned" || string(value.Resource.Body["status"]) != `"ACTIVE"` || string(value.Wire.Body["name"]) != `"cached"` {
		t.Fatal(err, value)
	}
	api, _, calls = fixture(t, func(*http.Request, int) *http.Response { t.Fatal("literal null default fetched"); return nil })
	value, err = api.WaitForStatus(context.Background(), waits.SecretStore("one"), nil)
	if err != nil || value.StatusCode != 0 || value.Envelope != nil || value.Wire != nil || len(*calls) != 0 {
		t.Fatal(err, value, *calls)
	}
	field(t, value, "status", "null")
}

func TestPythonKeymanagerSecretStoreWaitOrderedAliasesAndSynchronizedMerge(t *testing.T) {
	bodies := []string{`{"status":"BUILD","secret_store_ref":"https://passive.invalid/first","secret_store_id":"https://passive.invalid/last","created":null,"global_default":false}`, `not JSON`, `{"status":"ACTIVE","updated_at":false,"updated":0,"id":false}`}
	api, _, calls := fixture(t, func(req *http.Request, n int) *http.Response {
		if req.URL.EscapedPath() != "/barbican/v1/secret-stores/fixed%2Fid" {
			t.Fatal(req.URL)
		}
		return wire(202, bodies[n-1])
	})
	seed := nativeSecretStore(t, `{"status":"BUILD","name":"retained","secret_store_ref":"https://passive.invalid/seed","created":"old","global_default":true}`)
	var progress []int
	value, err := api.WaitForStatus(context.Background(), waits.ForSecretStore("fixed/id", seed), str("active"), waits.WithInterval(time.Nanosecond), waits.WithProgressCallback(func(v int) error { progress = append(progress, v); return nil }))
	if err != nil || len(*calls) != 3 || !reflect.DeepEqual(progress, []int{0, 0}) || value.StatusCode != 202 || string(value.Envelope) != bodies[2] {
		t.Fatal(err, value, *calls, progress)
	}
	for key, want := range map[string]string{"name": `"retained"`, "secret_store_ref": `"https://passive.invalid/last"`, "secret_store_id": `"last"`, "created_at": "null", "updated_at": "0", "global_default": "false", "id": "false"} {
		field(t, value, key, want)
	}
	if value.Wire.Body["name"] != nil || value.Wire.Body["secret_store_ref"] != nil {
		t.Fatal(value)
	}
	api, _, calls = fixture(t, func(*http.Request, int) *http.Response {
		return wire(200, `{"status":"ACTIVE","secret_store_id":"https://passive.invalid/first","secret_store_ref":"https://passive.invalid/last"}`)
	})
	value, err = api.WaitForStatus(context.Background(), waits.SecretStore("fixed"), str("active"))
	if err != nil || len(*calls) != 1 {
		t.Fatal(err, value, *calls)
	}
	field(t, value, "secret_store_id", `"last"`)
}

func TestPythonKeymanagerSecretStoreWaitBooleanDescriptorUsesSourceFloatTruthiness(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{{"null", "null"}, {"false", "false"}, {"0", "false"}, {"1e-400", "false"}, {"1e309", "true"}, {`"False"`, "true"}, {"[]", "false"}, {"[false]", "true"}, {"{}", "false"}, {`{"x":false}`, "true"}} {
		t.Run(tc.raw, func(t *testing.T) {
			body := `{"status":"ACTIVE","global_default":` + tc.raw + `}`
			api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, body) })
			value, err := api.WaitForStatus(context.Background(), waits.SecretStore("one"), str("active"))
			if err != nil || len(*calls) != 1 || string(value.Wire.Body["global_default"]) != tc.raw {
				t.Fatal(err, value, *calls)
			}
			field(t, value, "global_default", tc.want)
		})
	}
}

func TestPythonKeymanagerSecretStoreWaitDescriptorAndShapeErrorsKeepActualPage(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `false`, `{"status":false}`, `{"status":0}`, `{"status":"ACTIVE","secret_store_ref":"relative"}`, `{"status":"ACTIVE","secret_store_ref":false}`, `{"status":"ACTIVE","secret_store_ref":""}`, string([]byte{255})} {
		t.Run(body, func(t *testing.T) {
			api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, body) })
			value, err := api.WaitForStatus(context.Background(), waits.SecretStore("one"), str("active"))
			var proof *resource.ResponseError
			if value == nil || value.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "wait-proof" || len(*calls) != 1 {
				t.Fatal(err, value, proof, *calls)
			}
		})
	}
	for _, body := range []string{"", "malformed JSON"} {
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(204, body) })
		stopped := errors.New("callback stop")
		value, err := api.WaitForStatus(context.Background(), waits.SecretStore("one"), str("active"), waits.WithProgressCallback(func(v int) error {
			if v != 0 {
				t.Fatal(v)
			}
			return stopped
		}))
		var proof *resource.ResponseError
		if value == nil || value.Resource != nil || !errors.Is(err, stopped) || !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatal(err, value, proof, *calls)
		}
	}
}

func TestPythonKeymanagerSecretStoreWaitDeleteCleanAbsenceAndDisabledBackendLimitation(t *testing.T) {
	for _, body := range []string{`{"title":"Not Found"}`, `{"title":"Multiple backends support is not enabled in service configuration."}`} {
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(404, body) })
		seed := nativeSecretStore(t, `{"status":"ACTIVE","name":"original","secret_store_ref":"https://passive.invalid/store"}`)
		value, err := api.WaitForDelete(context.Background(), waits.ForSecretStore("fixed", seed))
		if err != nil || value == nil || value.StatusCode != 404 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "wait-proof" || len(*calls) != 1 {
			t.Fatal(err, value, *calls)
		}
		field(t, value, "name", `"original"`)
		field(t, value, "status", `"ACTIVE"`)
	}
	for _, body := range []string{`{"status":"DELETED"}`, `{"status":null}`, `{"status":false}`} {
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(203, body) })
		value, err := api.WaitForDelete(context.Background(), waits.SecretStore("one"))
		if len(*calls) != 1 || value == nil || value.StatusCode != 203 {
			t.Fatal(err, value, *calls)
		}
		if strings.Contains(body, "DELETED") {
			if err != nil {
				t.Fatal(err)
			}
			field(t, value, "status", `"DELETED"`)
		} else {
			var proof *resource.ResponseError
			if value.Resource != nil || !errors.As(err, &proof) || string(proof.Body) != body {
				t.Fatal(err, value, proof)
			}
		}
	}
	// A status wait does not reinterpret metadata404 as deletion/missing success.
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(404, `{"missing":true}`) })
	value, err := api.WaitForStatus(context.Background(), waits.SecretStore("one"), str("active"))
	if value == nil || value.Resource != nil || !gophercloud.ResponseCodeIs(err, 404) || len(*calls) != 1 {
		t.Fatal(err, value, *calls)
	}
}

func TestPythonKeymanagerSecretStoreWaitPreflightFailureAndCallbackPolicies(t *testing.T) {
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Fatal("preflight HTTP"); return nil })
	for _, target := range []waits.Target{waits.SecretStore(""), waits.ForSecretStore("one", nil), waits.ForSecretStore("one", &secretstores.SecretStore{})} {
		value, err := api.WaitForStatus(context.Background(), target, str("active"))
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(err, value, *calls)
		}
	}
	for _, option := range []waits.Option{waits.WithAttribute("progress"), waits.WithTimeout(0), waits.WithInterval(-time.Second)} {
		value, err := api.WaitForStatus(context.Background(), waits.SecretStore("one"), str("active"), option)
		if value != nil || err == nil || len(*calls) != 0 {
			t.Fatal(err, value, *calls)
		}
	}
	api, _, calls = fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":"ERROR"}`) })
	value, err := api.WaitForStatus(context.Background(), waits.SecretStore("one"), str("active"))
	var failed *resource.FailedStateError
	if value == nil || value.Resource != nil || !errors.As(err, &failed) || failed.Status != "ERROR" || len(*calls) != 1 {
		t.Fatal(err, value, failed, *calls)
	}
	api, _, calls = fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":"ERROR","progress":900}`) })
	stop := errors.New("explicit no failures callback")
	value, err = api.WaitForStatus(context.Background(), waits.SecretStore("one"), str("active"), waits.WithFailures(), waits.WithProgressCallback(func(v int) error {
		if v != 0 {
			t.Fatal(v)
		}
		return stop
	}))
	if value == nil || value.Resource != nil || !errors.Is(err, stop) || errors.As(err, &failed) || len(*calls) != 1 {
		t.Fatal(err, value, *calls)
	}
}

func TestPythonKeymanagerSecretStoreWaitCompoundPhysicalFaultsNeverRetryOrReauthenticate(t *testing.T) {
	for _, lane := range []string{"status", "delete"} {
		for _, status := range []int{200, 401, 404, 429, 503} {
			t.Run(fmt.Sprintf("%s/%d", lane, status), func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("physical read"), errors.New("physical close"), errors.New("custom cancellation")
				closes, retries, reauth := 0, 0, 0
				var client *gophercloud.ServiceClient
				originalBase := ""
				body := `{"status":"ACTIVE"}`
				if lane == "delete" {
					body = `{"status":"deleted"}`
				}
				api, selected, calls := fixture(t, func(*http.Request, int) *http.Response {
					response := wire(status, body)
					response.Body = &hookedBody{ReadCloser: &closingBody{Reader: strings.NewReader(body), close: func() error { closes++; client.ResourceBase = originalBase; cancel(cancelCause); return closeCause }}, after: func() { client.ResourceBase = client.Endpoint + "changed/" }, terminal: readCause}
					return response
				})
				client = selected
				originalBase = client.ResourceBase
				originalURL := client.ServiceURL("secret-stores", "one")
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				client.ReauthFunc = func(context.Context) error { reauth++; return nil }
				var value *waits.Record
				var err error
				if lane == "status" {
					value, err = api.WaitForStatus(ctx, waits.SecretStore("one"), str("active"))
				} else {
					value, err = api.WaitForDelete(ctx, waits.SecretStore("one"))
				}
				if value == nil || value.Resource != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || len(*calls) != 1 || closes != 1 || retries != 0 || reauth != 0 {
					t.Fatal(err, value, *calls, closes, retries, reauth)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "wait-proof" {
						t.Fatal(err, proof)
					}
				} else {
					originalErr := err
					if status == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil || !errors.Is(auth.ErrReauth, readCause) || !errors.Is(auth.ErrReauth, closeCause) || !errors.Is(auth.ErrReauth, cancelCause) || !errors.Is(auth.ErrReauth, resource.ErrInvalidOption) || !errors.Is(auth.ErrReauth, context.Canceled) {
							t.Fatal(err, auth)
						}
						originalErr = auth.ErrOriginal
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(originalErr, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != originalURL || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "wait-proof" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}

func TestPythonKeymanagerSecretStoreWaitCleanRetryReauthOwnHeadersAndLiveToken(t *testing.T) {
	for _, status := range []int{401, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			api, client, calls := fixture(t, func(req *http.Request, n int) *http.Response {
				if req.Header.Get("OpenStack-API-Version") != "key-manager 1.1" || req.Header.Get("X-Selected") != "owned" {
					t.Fatal(req.Header)
				}
				if n == 1 {
					return wire(status, `{"temporary":true}`)
				}
				if req.Header.Get("X-Auth-Token") != "renewed" {
					t.Fatal(req.Header)
				}
				return wire(200, `{"status":"ACTIVE"}`)
			})
			retries, reauth := 0, 0
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				client.SetToken("renewed")
				return nil
			}
			client.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("renewed"); return nil }
			value, err := api.WaitForStatus(context.Background(), waits.SecretStore("one"), str("active"), waits.WithMicroversion("1.1"), waits.WithHeader("X-Selected", "owned"))
			wantRetry, wantReauth := 1, 0
			if status == 401 {
				wantRetry, wantReauth = 0, 1
			}
			if err != nil || value == nil || len(*calls) != 2 || retries != wantRetry || reauth != wantReauth || client.Microversion != "" || client.MoreHeaders != nil {
				t.Fatal(err, value, *calls, retries, reauth, client)
			}
		})
	}
}
