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

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretacls"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func aclState(t *testing.T, id, body string) *secretacls.SecretACL {
	t.Helper()
	wireValue := raw(t, body)
	return &secretacls.SecretACL{Metadata: wireValue.Metadata, SecretID: id}
}

func TestPythonKeymanagerACLWaitOwnsParentRouteAndSourceDictDefaults(t *testing.T) {
	body := `{"acl_ref":"https://foreign.invalid/acl/future","read":false,"id":null,"name":0,"secret_id":"must-not-route","status":"must-not-wait","location":{"foreign":true},"extension":900719925474099312345}`
	api, client, calls := fixture(t, func(req *http.Request, _ int) *http.Response {
		if req.URL.EscapedPath() != "/barbican/v1/secrets/fixed%2Fid%20%252F/acl" || req.Header.Get("OpenStack-API-Version") != "key-manager 1.1" || req.Header.Get("X-Selected") != "owned" {
			t.Fatal(req.URL, req.Header)
		}
		return wire(203, body)
	})
	value, err := api.WaitForStatus(context.Background(), waits.SecretACL("fixed/id %2F"), str("https://foreign.invalid/acl/future"), waits.WithAttribute("acl_ref"), waits.WithHeader("X-Selected", "owned"), waits.WithMicroversion("1.1"))
	if err != nil || value == nil || value.Kind != "secret-acls" || value.RequestID != "fixed/id %2F" || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "wait-proof" || len(*calls) != 1 || client.MoreHeaders != nil || client.Microversion != "" {
		t.Fatal(err, value, *calls)
	}
	for key, want := range map[string]string{"acl_ref": `"https://foreign.invalid/acl/future"`, "id": "null", "name": "0", "read": "{}", "location": "null"} {
		field(t, value, key, want)
	}
	if len(value.Resource.Body) != 5 || value.Resource.Body["status"] != nil || value.Resource.Body["secret_id"] != nil || string(value.Wire.Body["read"]) != "false" || string(value.Wire.Body["extension"]) != "900719925474099312345" {
		t.Fatal(value)
	}
	clone := value.Clone()
	clone.Resource.Body["read"][0] = 'x'
	clone.Wire.Body["read"][0] = 'x'
	clone.Envelope[0] = 'x'
	clone.Header.Set("X-Request-Id", "changed")
	if string(value.Resource.Body["read"]) != "{}" || string(value.Wire.Body["read"]) != "false" || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "wait-proof" {
		t.Fatal("receipt aliases", value)
	}
}

func TestPythonKeymanagerACLWaitSnapshotsCachedNativeAndURIWithoutFakeEnvelope(t *testing.T) {
	seed := aclState(t, "fixed/id", `{"acl_ref":"ACTIVE","read":{"users":["one"]},"id":false,"name":null}`)
	seed.Header = http.Header{"X-Original": []string{"owned"}}
	seed.StatusCode = 203
	target := waits.ForSecretACL(seed)
	seed.SecretID = "changed"
	seed.Body["acl_ref"][1] = 'x'
	seed.Body["read"][0] = 'x'
	seed.Header.Set("X-Original", "changed")
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Fatal("cached HTTP"); return nil })
	value, err := api.WaitForStatus(context.Background(), target, str("active"), waits.WithAttribute("acl_ref"), waits.WithTimeout(0), waits.WithInterval(-time.Second))
	if err != nil || value == nil || len(*calls) != 0 || value.RequestID != "fixed/id" || value.Envelope != nil || value.StatusCode != 203 || value.Header.Get("X-Original") != "owned" {
		t.Fatal(err, value, *calls)
	}
	field(t, value, "read", `{"users":["one"]}`)
	field(t, value, "id", "false")
	value.Resource.Body["acl_ref"][1] = 'x'
	value.Wire.Body["read"][0] = 'x'
	value.Header.Set("X-Original", "consumer")
	value, err = api.WaitForStatus(context.Background(), target, str("active"), waits.WithAttribute("acl_ref"))
	if err != nil || value.Header.Get("X-Original") != "owned" {
		t.Fatal(err, value)
	}
	field(t, value, "acl_ref", `"ACTIVE"`)
	// URI is declared for getattr, while to_dict omits URI fields.
	value, err = api.WaitForStatus(context.Background(), waits.SecretACL("fixed/id"), str("fixed/id"), waits.WithAttribute("secret_id"), waits.WithTimeout(0))
	if err != nil || value.StatusCode != 0 || value.Wire != nil || value.Envelope != nil || value.Resource.Body["secret_id"] != nil || len(*calls) != 0 {
		t.Fatal(err, value, *calls)
	}
	field(t, value, "id", "null")
	// No native Body means only actual cached typed channels, not fake Wire.
	value, err = api.WaitForStatus(context.Background(), waits.ForSecretACL(&secretacls.SecretACL{SecretID: "one", Read: []byte(`null`), ACLRef: []byte(`"cached"`)}), str("cached"), waits.WithAttribute("acl_ref"))
	if err != nil || value.Wire != nil || value.Envelope != nil || value.StatusCode != 0 || len(*calls) != 0 {
		t.Fatal(err, value, *calls)
	}
}

func TestPythonKeymanagerACLWaitRawPresenceMergeAndReadDictCoercion(t *testing.T) {
	bodies := []string{`{"name":"BUILD","acl_ref":"https://foreign.invalid/one","read":false}`, `not JSON`, `{"name":"ACTIVE","id":null,"read":{"project-access":false,"users":[]}}`}
	api, _, calls := fixture(t, func(req *http.Request, n int) *http.Response {
		if req.URL.EscapedPath() != "/barbican/v1/secrets/one/acl" {
			t.Fatal(req.URL)
		}
		return wire(202, bodies[n-1])
	})
	seed := aclState(t, "one", `{"name":"BUILD","acl_ref":"old","read":{"old":true}}`)
	var progress []int
	value, err := api.WaitForStatus(context.Background(), waits.ForSecretACL(seed), str("active"), waits.WithAttribute("name"), waits.WithInterval(time.Nanosecond), waits.WithProgressCallback(func(v int) error { progress = append(progress, v); return nil }))
	if err != nil || len(*calls) != 3 || !reflect.DeepEqual(progress, []int{0, 0}) || value.StatusCode != 202 || string(value.Envelope) != bodies[2] {
		t.Fatal(err, value, *calls, progress)
	}
	field(t, value, "acl_ref", `"https://foreign.invalid/one"`)
	field(t, value, "id", "null")
	field(t, value, "read", `{"project-access":false,"users":[]}`)
	if value.Wire.Body["acl_ref"] != nil {
		t.Fatal(value)
	}
	for _, tc := range []struct{ read, want string }{{"null", "null"}, {"true", "{}"}, {"0", "{}"}, {`"value"`, "{}"}, {"[]", "{}"}, {`{"users":[]}`, `{"users":[]}`}} {
		api, _, calls = fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"acl_ref":"ACTIVE","read":`+tc.read+`}`) })
		value, err = api.WaitForStatus(context.Background(), waits.SecretACL("one"), str("active"), waits.WithAttribute("acl_ref"))
		if err != nil || len(*calls) != 1 || string(value.Wire.Body["read"]) != tc.read {
			t.Fatal(err, value, *calls)
		}
		field(t, value, "read", tc.want)
	}
}

func TestPythonKeymanagerACLWaitMissingStatusAndTypedAttributePreflight(t *testing.T) {
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Fatal("preflight HTTP"); return nil })
	for _, target := range []waits.Target{waits.SecretACL("one"), waits.ForSecretACL(aclState(t, "one", `{"status":"ACTIVE"}`))} {
		value, err := api.WaitForStatus(context.Background(), target, nil)
		if value != nil || !errors.Is(err, resource.ErrUnsupported) || len(*calls) != 0 {
			t.Fatal(err, value, *calls)
		}
	}
	for _, target := range []waits.Target{waits.SecretACL(""), waits.SecretACL("bad\n"), waits.ForSecretACL(nil)} {
		value, err := api.WaitForStatus(context.Background(), target, str("active"), waits.WithAttribute("acl_ref"))
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(err, value, *calls)
		}
	}
	value, err := api.WaitForStatus(context.Background(), waits.ForSecretACL(aclState(t, "one", `{"read":{"users":[]}}`)), nil, waits.WithAttribute("read"))
	if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(err, value, *calls)
	}
	value, err = api.WaitForStatus(context.Background(), waits.SecretACL("one"), nil, waits.WithAttribute("read"))
	if err != nil || len(*calls) != 0 || value.StatusCode != 0 {
		t.Fatal(err, value, *calls)
	}
	field(t, value, "read", "null")
	value, err = api.WaitForDelete(context.Background(), waits.SecretACL("one"), waits.WithAttribute("acl_ref"))
	if value != nil || !errors.Is(err, resource.ErrUnsupported) || len(*calls) != 0 {
		t.Fatal(err, value, *calls)
	}
}

func TestPythonKeymanagerACLQuotaWaitNonobjectAndSelectedValueErrorsKeepActualProof(t *testing.T) {
	for _, kind := range []string{"acl", "quota"} {
		for _, body := range []string{`null`, `[]`, `false`, `{"name":false}`, `{"name":0}`, string([]byte{255})} {
			t.Run(kind+"/"+body, func(t *testing.T) {
				api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(203, body) })
				target := waits.SecretACL("one")
				if kind == "quota" {
					target = waits.Quota()
				}
				value, err := api.WaitForStatus(context.Background(), target, str("active"), waits.WithAttribute("name"))
				var proof *resource.ResponseError
				if value == nil || value.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "wait-proof" || len(*calls) != 1 {
					t.Fatal(err, value, proof, *calls)
				}
			})
		}
		for _, body := range []string{"", "invalid JSON"} {
			api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(204, body) })
			target := waits.SecretACL("one")
			if kind == "quota" {
				target = waits.Quota()
			}
			stopped := errors.New("callback stop")
			value, err := api.WaitForStatus(context.Background(), target, str("active"), waits.WithAttribute("name"), waits.WithProgressCallback(func(v int) error {
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
}

func TestPythonKeymanagerACLQuotaDeleteSkipsUndeclaredStatusAndRetainsOriginalAt404(t *testing.T) {
	for _, kind := range []string{"acl", "quota"} {
		api, _, calls := fixture(t, func(_ *http.Request, n int) *http.Response {
			if n == 1 {
				return wire(203, `{"name":"fresh","status":"deleted","id":"foreign"}`)
			}
			return wire(404, `{"missing":true}`)
		})
		target := waits.ForSecretACL(aclState(t, "one", `{"name":"original","acl_ref":"cached"}`))
		if kind == "quota" {
			target = quotaTarget(t, `{"name":"original","secrets":900719925474099312345}`)
		}
		var progress []int
		value, err := api.WaitForDelete(context.Background(), target, waits.WithInterval(time.Nanosecond), waits.WithProgressCallback(func(v int) error { progress = append(progress, v); return nil }))
		if err != nil || value == nil || value.StatusCode != 404 || string(value.Envelope) != `{"missing":true}` || len(*calls) != 2 || !reflect.DeepEqual(progress, []int{0}) {
			t.Fatal(err, value, *calls, progress)
		}
		field(t, value, "name", `"original"`)
		if value.Resource.Body["status"] != nil {
			t.Fatal(value)
		}
		// Clean status404 remains an error, without delete missing conversion.
		api, _, calls = fixture(t, func(*http.Request, int) *http.Response { return wire(404, `{"missing":true}`) })
		value, err = api.WaitForStatus(context.Background(), target, str("active"), waits.WithAttribute("name"))
		if value == nil || value.Resource != nil || !gophercloud.ResponseCodeIs(err, 404) || len(*calls) != 1 {
			t.Fatal(err, value, *calls)
		}
	}
}

func TestPythonKeymanagerACLQuotaPhysicalFailuresNeverBecomeAbsenceOrRetry(t *testing.T) {
	for _, kind := range []string{"acl", "quota"} {
		for _, lane := range []string{"status", "delete"} {
			for _, status := range []int{200, 401, 404, 429, 503} {
				t.Run(fmt.Sprintf("%s/%s/%d", kind, lane, status), func(t *testing.T) {
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					readCause, closeCause, cancelCause := errors.New("physical read"), errors.New("physical close"), errors.New("custom cancellation")
					closes, retries, reauth := 0, 0, 0
					var client *gophercloud.ServiceClient
					originalBase := ""
					body := `{"name":"ACTIVE","read":{},"status":"deleted"}`
					api, selected, calls := fixture(t, func(*http.Request, int) *http.Response {
						response := wire(status, body)
						response.Body = &hookedBody{ReadCloser: &closingBody{Reader: strings.NewReader(body), close: func() error { closes++; client.ResourceBase = originalBase; cancel(cancelCause); return closeCause }}, after: func() { client.ResourceBase = client.Endpoint + "changed/" }, terminal: readCause}
						return response
					})
					client = selected
					originalBase = client.ResourceBase
					target := waits.SecretACL("one")
					originalURL := client.ServiceURL("secrets", "one", "acl")
					if kind == "quota" {
						target = waits.Quota()
						originalURL = client.ServiceURL("quotas")
					}
					client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						retries++
						return nil
					}
					client.ReauthFunc = func(context.Context) error { reauth++; return nil }
					var value *waits.Record
					var err error
					if lane == "status" {
						value, err = api.WaitForStatus(ctx, target, str("active"), waits.WithAttribute("name"))
					} else {
						value, err = api.WaitForDelete(ctx, target)
					}
					if value == nil || value.Resource != nil || value.StatusCode != status || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "wait-proof" || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || len(*calls) != 1 || closes != 1 || retries != 0 || reauth != 0 {
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
}

func TestPythonKeymanagerACLQuotaCleanRetryReauthPreservesStockRouteAndOwnedHeaders(t *testing.T) {
	for _, kind := range []string{"acl", "quota"} {
		for _, status := range []int{401, 503} {
			t.Run(fmt.Sprintf("%s/%d", kind, status), func(t *testing.T) {
				path := "/barbican/v1/secrets/one/acl"
				if kind == "quota" {
					path = "/barbican/v1/quotas"
				}
				api, client, calls := fixture(t, func(req *http.Request, n int) *http.Response {
					if req.URL.EscapedPath() != path || req.Header.Get("OpenStack-API-Version") != "key-manager 1.1" || req.Header.Get("X-Selected") != "owned" {
						t.Fatal(req.URL, req.Header)
					}
					if n == 1 {
						return wire(status, `{"temporary":true}`)
					}
					if req.Header.Get("X-Auth-Token") != "renewed" {
						t.Fatal(req.Header)
					}
					return wire(200, `{"name":"ACTIVE","id":"must-not-route"}`)
				})
				retries, reauth := 0, 0
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					client.SetToken("renewed")
					return nil
				}
				client.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("renewed"); return nil }
				target := waits.SecretACL("one")
				if kind == "quota" {
					target = waits.Quota()
				}
				value, err := api.WaitForStatus(context.Background(), target, str("active"), waits.WithAttribute("name"), waits.WithHeader("X-Selected", "owned"), waits.WithMicroversion("1.1"))
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
}
