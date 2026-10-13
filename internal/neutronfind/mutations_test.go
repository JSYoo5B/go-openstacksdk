package neutronfind_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func mutationProfile() neutronfind.Profile {
	profile := policyProfile(true)
	profile.StopFaultRetryCallbacks = true
	profile.Headers[0].Kind = neutronfind.List
	profile.Body = append(profile.Body,
		neutronfind.Descriptor{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"},
		neutronfind.Descriptor{Name: "revision_number", Wire: "revision", Kind: neutronfind.Integer, Missing: "null"},
		neutronfind.Descriptor{Name: "tags", Wire: "tags", Kind: neutronfind.List, Missing: "[]"})
	return profile
}
func mutationEngine(t *testing.T, reply func(*http.Request) (*http.Response, error)) (*neutronfind.Engine, *gophercloud.ServiceClient) {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("network", "/catalog/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	cloud.Provider.HTTPClient.Transport = policyTransport(reply)
	engine, err := neutronfind.New(context.Background(), client, mutationProfile(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return engine, client
}
func mutationRequestBody(t *testing.T, req *http.Request) string {
	t.Helper()
	if req.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := req.Body.Close(); err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
func mutationResponse(code int, raw string) *http.Response {
	response := policyWire(code, raw)
	response.Header.Set("X-Request-Id", "mutation-actual")
	return response
}
func mutationProof(t *testing.T, err error, code int, raw string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != raw || proof.Header.Get("X-Request-Id") != "mutation-actual" {
		t.Fatal(err, proof)
	}
	return proof
}

func TestSourceMutateOwnsBodySeedPolicyAndActualProjection(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			bodyRaw, seedRaw := json.RawMessage(`{"nested":[1,false,null]}`), json.RawMessage(`{"owned":[2]}`)
			body := map[string]json.RawMessage{"shared": json.RawMessage("false"), "x_body": bodyRaw}
			seed := map[string]json.RawMessage{"id": json.RawMessage(`"seed-id"`), "name": json.RawMessage(`"seed-name"`), "seed_only": seedRaw, "tenant_id": json.RawMessage(`"seed-tenant"`), "if_match": json.RawMessage(`"seed-condition"`), "microversion": json.RawMessage(`"seed-version"`)}
			version := "2.1"
			headers := map[string]string{"X-Probe": "owned", "If-Match": "selected"}
			raw := `{"record":{"id":null,"name":"actual","shared":true,"project_id":null,"tenant_id":"actual-tenant","revision":12.9,"if_match":"body-unknown","location":{"project":"forged"},"microversion":"foreign","x_actual":900719925474099312345},"proof":"whole-envelope"}`
			calls := 0
			var client *gophercloud.ServiceClient
			cloud := testcloud.New(t)
			client = cloud.Client("network", "/catalog/")
			client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
			client.Microversion = "2.0"
			client.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0", "X-Source": "original"}
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				path := "/neutron/v2.0/records"
				if method == http.MethodPut {
					path += "/folder%2Fname%25"
				}
				if req.Method != method || req.URL.EscapedPath() != path || req.URL.RawQuery != "" || req.Header.Get("OpenStack-API-Version") != "network 2.1" || req.Header.Get("X-Probe") != "owned" || req.Header.Get("If-Match") != "selected" || req.Header.Get("X-Auth-Token") == "" {
					t.Fatal(req.Method, req.URL, req.Header)
				}
				if got := mutationRequestBody(t, req); got != `{"record":{"shared":false,"x_body":{"nested":[1,false,null]}}}` {
					t.Fatal(got)
				}
				bodyRaw[1] = 'X'
				seedRaw[1] = 'X'
				body["late"] = json.RawMessage("true")
				seed["late"] = json.RawMessage("true")
				version = "bad\n"
				headers["X-Probe"] = "late"
				response := mutationResponse(203, raw)
				response.Header.Set("If-Match", "actual-condition")
				return response, nil
			})
			engine, err := neutronfind.New(context.Background(), client, mutationProfile(), nil)
			if err != nil {
				t.Fatal(err)
			}
			identity := ""
			if method == http.MethodPut {
				identity = "folder/name%"
			}
			value, err := engine.Mutate(context.Background(), method, identity, body, seed, neutronfind.Options{Microversion: &version, Headers: headers})
			if err != nil || value == nil || value.Resource == nil || value.Wire == nil || calls != 1 || value.StatusCode != 203 || string(value.Envelope) != raw {
				t.Fatal(value, err, calls)
			}
			for key, want := range map[string]string{"id": "null", "name": `"actual"`, "is_shared": "true", "project_id": "null", "tenant_id": `"actual-tenant"`, "revision_number": "12", "tags": "[]", "if_match": `["actual-condition"]`, "location": "null", "microversion": `"2.1"`, "seed_only": `{"owned":[2]}`} {
				if string(value.Resource.Body[key]) != want {
					t.Fatal(key, want, value.Resource.Body)
				}
			}
			if value.Resource.Body["late"] != nil || value.Wire.Body["seed_only"] != nil || string(value.Wire.Body["revision"]) != "12.9" || string(value.Wire.Body["microversion"]) != `"foreign"` || string(value.Wire.Body["location"]) != `{"project":"forged"}` || client.Microversion != "2.0" || client.MoreHeaders["Openstack-Api-Version"] != "network 2.0" {
				t.Fatal(value, client)
			}
			value.Resource.Body["name"][1] = 'X'
			value.Resource.Body["seed_only"][1] = 'X'
			value.Header.Set("X-Request-Id", "consumer")
			if string(value.Wire.Body["name"]) != `"actual"` || string(value.Envelope) != raw || value.Wire.Header.Get("X-Request-Id") != "mutation-actual" || value.Resource.Header.Get("X-Request-Id") != "mutation-actual" {
				t.Fatal("receipt ownership", value)
			}
		})
	}
}

func TestSourceMutateEmptyEnvelopeAndSeededAcceptedResponses(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, raw := range []string{"", `not JSON`, `{"record":{}}`} {
			t.Run(method+"/"+raw, func(t *testing.T) {
				calls := 0
				engine, _ := mutationEngine(t, func(req *http.Request) (*http.Response, error) {
					calls++
					if got := mutationRequestBody(t, req); got != `{"record":{}}` {
						t.Fatal(got)
					}
					return mutationResponse(204, raw), nil
				})
				identity := ""
				if method == http.MethodPut {
					identity = "member"
				}
				value, err := engine.Mutate(context.Background(), method, identity, nil, map[string]json.RawMessage{"id": json.RawMessage(`"seed"`), "tenant_id": json.RawMessage(`"legacy"`)}, neutronfind.Options{})
				if err != nil || value == nil || value.Resource == nil || value.StatusCode != 204 || value.Resource.StatusCode != 204 || string(value.Envelope) != raw || string(value.Resource.Body["id"]) != `"seed"` || string(value.Resource.Body["project_id"]) != `"legacy"` || calls != 1 {
					t.Fatal(value, err, calls)
				}
				if raw == `{"record":{}}` {
					if value.Wire == nil || len(value.Wire.Body) != 0 {
						t.Fatal(value)
					}
				} else if value.Wire != nil {
					t.Fatal("invented decoded wire", value)
				}
			})
		}
	}
}

func TestSourceMutateRejectsRemoteShapesWithActualProof(t *testing.T) {
	for _, raw := range []string{`{"record":false}`, `{"record":[]}`, `{"record":null}`, `[false]`, `{"record":{"revision":"²"}}`, "{\"record\":{\"name\":\"\xff\"}}"} {
		t.Run(raw, func(t *testing.T) {
			calls := 0
			engine, _ := mutationEngine(t, func(*http.Request) (*http.Response, error) { calls++; return mutationResponse(202, raw), nil })
			value, err := engine.Mutate(context.Background(), http.MethodPost, "", nil, nil, neutronfind.Options{})
			if err == nil || value == nil || value.Resource != nil || value.StatusCode != 202 || string(value.Envelope) != raw || calls != 1 {
				t.Fatal(value, err, calls)
			}
			proof := mutationProof(t, err, 202, raw)
			if proof.Cause == nil || !errors.Is(err, proof.Cause) {
				t.Fatal(err, proof)
			}
		})
	}
}

func TestSourceMutateDeletePreservesBodylessRawReceiptAndNativeRejections(t *testing.T) {
	for _, code := range []int{199, 200, 202, 204, 299, 300, 399, 400, 403, 404, 409, 429, 500, 503, 599} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			raw := "\xffnot JSON"
			calls := 0
			target := ""
			engine, _ := mutationEngine(t, func(req *http.Request) (*http.Response, error) {
				calls++
				target = req.URL.String()
				if req.Method != http.MethodDelete || req.URL.EscapedPath() != "/neutron/v2.0/records/folder%2Fname%25" || req.Body != nil || req.URL.RawQuery != "" {
					t.Fatal(req.Method, req.URL, req.Body)
				}
				return mutationResponse(code, raw), nil
			})
			value, err := engine.Mutate(context.Background(), http.MethodDelete, "folder/name%", nil, map[string]json.RawMessage{"name": json.RawMessage(`"seed-is-not-a-receipt"`)}, neutronfind.Options{})
			if calls != 1 {
				t.Fatal(calls)
			}
			if code >= 200 && code < 400 {
				if err != nil || value == nil || value.Resource != nil || value.Wire != nil || value.StatusCode != code || string(value.Envelope) != raw || value.Header.Get("X-Request-Id") != "mutation-actual" {
					t.Fatal(value, err)
				}
				value.Header.Set("X-Request-Id", "consumer")
				if string(value.Envelope) != raw {
					t.Fatal(value)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodDelete || native.URL != target || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "mutation-actual" {
					t.Fatal(value, err, native)
				}
				var proof *resource.ResponseError
				if errors.As(err, &proof) {
					t.Fatal("rejection turned into accepted receipt", proof)
				}
			}
		})
	}
}

func TestSourceProjectSeedOwnsDefaultsAndValidatesPolicyWithoutHTTP(t *testing.T) {
	for _, version := range []string{"", "2.2"} {
		t.Run(version, func(t *testing.T) {
			calls := 0
			engine, client := mutationEngine(t, func(*http.Request) (*http.Response, error) { calls++; t.Fatal("no-op sent HTTP"); return nil, nil })
			raw := json.RawMessage(`{"nested":[false,1]}`)
			seed := map[string]json.RawMessage{"id": json.RawMessage(`"seed-id"`), "tenant_id": json.RawMessage(`"legacy"`), "is_shared": json.RawMessage(`"false"`), "revision_number": json.RawMessage(`"١٢"`), "if_match": json.RawMessage(`false`), "microversion": json.RawMessage(`"foreign"`), "location": json.RawMessage(`{"forged":true}`), "extension": raw, "explicit_null": nil}
			value, err := engine.ProjectSeed(context.Background(), seed, neutronfind.Options{Microversion: &version, Headers: map[string]string{"X-Probe": "transport-only"}})
			if err != nil || value == nil || value.Resource == nil || value.Wire != nil || value.Envelope != nil || value.Header != nil || value.StatusCode != 0 || value.Resource.Header != nil || value.Resource.StatusCode != 0 || calls != 0 {
				t.Fatal(value, err, calls)
			}
			wantVersion := "null"
			if version != "" {
				wantVersion = `"` + version + `"`
			}
			for key, want := range map[string]string{"id": `"seed-id"`, "project_id": `"legacy"`, "is_shared": "true", "revision_number": "12", "tags": "[]", "if_match": "[false]", "location": "null", "microversion": wantVersion, "extension": `{"nested":[false,1]}`, "explicit_null": "null"} {
				if string(value.Resource.Body[key]) != want {
					t.Fatal(key, want, value.Resource.Body)
				}
			}
			raw[1] = 'X'
			seed["later"] = json.RawMessage("true")
			if string(value.Resource.Body["extension"]) != `{"nested":[false,1]}` || value.Resource.Body["later"] != nil || client.Microversion != "" {
				t.Fatal(value, client)
			}
		})
	}
}

func TestSourceMutationsPreflightKeepsCallerCausesAndNoInventedHTTP(t *testing.T) {
	cases := []struct {
		name, method, id string
		body, seed       map[string]json.RawMessage
		policy           neutronfind.Options
	}{
		{name: "unsupported", method: http.MethodPatch, id: "member"},
		{name: "lowercase", method: "post"},
		{name: "post member", method: http.MethodPost, id: "member"},
		{name: "blank member", method: http.MethodPut},
		{name: "dot member", method: http.MethodDelete, id: "."},
		{name: "dotdot member", method: http.MethodPut, id: ".."},
		{name: "control member", method: http.MethodPut, id: "bad\n"},
		{name: "utf8 member", method: http.MethodDelete, id: "\xff"},
		{name: "query", method: http.MethodPost, policy: neutronfind.Options{Query: map[string]json.RawMessage{"unused": json.RawMessage("null")}}},
		{name: "delete body", method: http.MethodDelete, id: "member", body: map[string]json.RawMessage{"name": json.RawMessage("null")}},
		{name: "malformed body", method: http.MethodPost, body: map[string]json.RawMessage{"name": json.RawMessage("broken")}},
		{name: "malformed seed", method: http.MethodPost, seed: map[string]json.RawMessage{"name": json.RawMessage("broken")}},
		{name: "utf8 body", method: http.MethodPost, body: map[string]json.RawMessage{"name": json.RawMessage("\"\xff\"")}},
		{name: "header", method: http.MethodPost, policy: neutronfind.Options{Headers: map[string]string{"Authorization": "foreign"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			engine, _ := mutationEngine(t, func(*http.Request) (*http.Response, error) { calls++; t.Fatal("preflight sent HTTP"); return nil, nil })
			value, err := engine.Mutate(context.Background(), tc.method, tc.id, tc.body, tc.seed, tc.policy)
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(value, err, calls)
			}
			var proof *resource.ResponseError
			if errors.As(err, &proof) {
				t.Fatal("invented receipt", proof)
			}
			if strings.HasPrefix(tc.name, "malformed") {
				var syntax *json.SyntaxError
				if !errors.As(err, &syntax) {
					t.Fatal("JSON cause dropped", err)
				}
			}
		})
	}
	for _, policy := range []neutronfind.Options{{Query: map[string]json.RawMessage{"name": json.RawMessage(`"ignored?"`)}}, {Headers: map[string]string{"X-Auth-Token": "foreign"}}, {Microversion: func() *string { value := "bad\n"; return &value }()}} {
		engine, _ := mutationEngine(t, func(*http.Request) (*http.Response, error) { t.Fatal("seed sent HTTP"); return nil, nil })
		value, err := engine.ProjectSeed(context.Background(), nil, policy)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	engine, _ := mutationEngine(t, func(*http.Request) (*http.Response, error) { t.Fatal("invalid context sent HTTP"); return nil, nil })
	if _, err := engine.Mutate(nil, http.MethodPost, "", nil, nil, neutronfind.Options{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := (*neutronfind.Engine)(nil).ProjectSeed(context.Background(), nil, neutronfind.Options{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	cause := errors.New("canceled mutation")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if _, err := engine.Mutate(ctx, http.MethodPost, "", nil, nil, neutronfind.Options{}); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := engine.ProjectSeed(ctx, nil, neutronfind.Options{}); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

type mutationFaultReader func([]byte) (int, error)

func (f mutationFaultReader) Read(p []byte) (int, error) { return f(p) }

type mutationFaultBody struct {
	io.Reader
	closed     int
	closeCause error
	onClose    func()
}

func (b *mutationFaultBody) Close() error {
	b.closed++
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeCause
}
func TestSourceMutatePhysicalFaultsPreserveAllCausesAndNative401Fields(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		for _, code := range []int{200, 203, 401, 404, 429, 503} {
			t.Run(fmt.Sprintf("%s/%d", method, code), func(t *testing.T) {
				raw := `{"record":{"id":"actual"},"fault":"all-causes"}`
				readCause, closeCause, cancelCause := errors.New("mutation read"), errors.New("mutation close"), errors.New("mutation cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				calls, retries, reauth := 0, 0, 0
				target := ""
				done := false
				var client *gophercloud.ServiceClient
				original := ""
				physical := &mutationFaultBody{closeCause: closeCause}
				physical.Reader = mutationFaultReader(func(p []byte) (int, error) {
					if done {
						return 0, io.EOF
					}
					done = true
					client.ResourceBase = "https://foreign.invalid/v2/"
					cancel(cancelCause)
					return copy(p, raw), readCause
				})
				physical.onClose = func() { client.ResourceBase = original }
				engine, selected := mutationEngine(t, func(req *http.Request) (*http.Response, error) {
					calls++
					target = req.URL.String()
					res := mutationResponse(code, raw)
					res.Body = physical
					return res, nil
				})
				client = selected
				original = client.ResourceBase
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				client.ReauthFunc = func(context.Context) error { reauth++; return nil }
				identity := ""
				if method != http.MethodPost {
					identity = "member"
				}
				value, err := engine.Mutate(ctx, method, identity, nil, nil, neutronfind.Options{})
				if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || physical.closed != 1 || retries != 0 || reauth != 0 {
					t.Fatal(value, err, calls, physical.closed, retries, reauth)
				}
				if code < 400 {
					mutationProof(t, err, code, raw)
					if value == nil || value.Resource != nil || value.Wire != nil || value.StatusCode != code || string(value.Envelope) != raw || value.Header.Get("X-Request-Id") != "mutation-actual" {
						t.Fatal(value)
					}
				} else {
					if value != nil {
						t.Fatal("native rejection has successful record", value)
					}
					originalErr := err
					if code == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || !errors.Is(auth.ErrReauth, readCause) || !errors.Is(auth.ErrReauth, closeCause) || !errors.Is(auth.ErrReauth, cancelCause) || !errors.Is(auth.ErrReauth, context.Canceled) || !errors.Is(auth.ErrReauth, resource.ErrInvalidOption) {
							t.Fatal(err, auth)
						}
						originalErr = auth.ErrOriginal
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(originalErr, &native) || native.Actual != code || native.Method != method || native.URL != target || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "mutation-actual" {
						t.Fatal(err, native, target)
					}
				}
			})
		}
	}
}

func TestSourceMutateCleanRetryAndReauthenticationKeepCapturedRequests(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		for _, first := range []int{401, 503} {
			t.Run(fmt.Sprintf("%s/%d", method, first), func(t *testing.T) {
				calls, hooks := 0, 0
				var targets, bodies, tokens []string
				engine, client := mutationEngine(t, func(req *http.Request) (*http.Response, error) {
					calls++
					targets = append(targets, req.Method+" "+req.URL.String())
					bodies = append(bodies, mutationRequestBody(t, req))
					tokens = append(tokens, req.Header.Get("X-Auth-Token"))
					if calls == 1 {
						return mutationResponse(first, `{"error":"retry"}`), nil
					}
					return mutationResponse(200, `{"record":{"id":"actual"}}`), nil
				})
				if first == 401 {
					client.ReauthFunc = func(context.Context) error { hooks++; client.SetToken("renewed"); return nil }
				} else {
					client.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
						hooks++
						opts.MoreHeaders["X-Retry-Probe"] = "allowed"
						client.SetToken("renewed")
						return nil
					}
				}
				identity := ""
				if method != http.MethodPost {
					identity = "member"
				}
				value, err := engine.Mutate(context.Background(), method, identity, nil, nil, neutronfind.Options{})
				if err != nil || value == nil || calls != 2 || hooks != 1 || targets[0] != targets[1] || bodies[0] != bodies[1] || !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
					t.Fatal(value, err, calls, hooks, targets, bodies, tokens)
				}
			})
		}
	}
}

func TestSourceMutateRetryCannotChangeBodyOrConditionalPolicy(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		for _, mode := range []string{"body", "condition injection", "condition replacement", "condition omission", "version injection", "version omission", "content-type replacement", "content-type omission"} {
			t.Run(method+"/"+mode, func(t *testing.T) {
				if method == http.MethodDelete && strings.HasPrefix(mode, "content-type") {
					return
				}
				raw := `{"rejected":"actual first"}`
				calls, hooks := 0, 0
				cause := errors.New("mutation retry callback")
				engine, client := mutationEngine(t, func(req *http.Request) (*http.Response, error) { calls++; return mutationResponse(503, raw), nil })
				headers := map[string]string{}
				if mode == "condition replacement" || mode == "condition omission" {
					headers["If-Match"] = "selected"
				}
				var version *string
				if mode == "version omission" {
					v := "2.1"
					version = &v
				}
				client.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					hooks++
					switch mode {
					case "body":
						opts.JSONBody = map[string]any{"changed": true}
					case "condition injection", "condition replacement":
						opts.MoreHeaders["If-Match"] = "foreign"
					case "condition omission":
						opts.OmitHeaders = []string{"If-Match"}
					case "version injection":
						opts.MoreHeaders["OpenStack-API-Version"] = "network 2.9"
					case "version omission":
						opts.OmitHeaders = []string{"OpenStack-API-Version"}
					case "content-type replacement":
						opts.MoreHeaders["Content-Type"] = "application/xml"
					case "content-type omission":
						opts.OmitHeaders = []string{"Content-Type"}
					}
					return cause
				}
				identity := ""
				if method != http.MethodPost {
					identity = "member"
				}
				value, err := engine.Mutate(context.Background(), method, identity, nil, nil, neutronfind.Options{Headers: headers, Microversion: version})
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || native.Method != method || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "mutation-actual" || calls != 1 || hooks != 1 {
					t.Fatal(value, err, native, calls, hooks)
				}
			})
		}
	}
}

func TestSourceMutateSelectedConditionsAndExplicitContentTypeAllowCleanRetry(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		for _, condition := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"} {
			t.Run(method+"/"+condition, func(t *testing.T) {
				calls, hooks := 0, 0
				engine, client := mutationEngine(t, func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Header.Get(condition) != "selected" || req.Header.Get("OpenStack-API-Version") != "" {
						t.Fatal(req.Header)
					}
					if method != http.MethodDelete && req.Header.Get("Content-Type") != "application/vendor+json" {
						t.Fatal(req.Header)
					}
					if calls == 1 {
						return mutationResponse(503, `{"error":"normal retry"}`), nil
					}
					if req.Header.Get("X-Retry-Probe") != "allowed" {
						t.Fatal(req.Header)
					}
					return mutationResponse(203, `{"record":{}}`), nil
				})
				client.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					hooks++
					opts.MoreHeaders["X-Retry-Probe"] = "allowed"
					return nil
				}
				headers := map[string]string{condition: "selected"}
				if method != http.MethodDelete {
					headers["Content-Type"] = "application/vendor+json"
				}
				identity := ""
				if method != http.MethodPost {
					identity = "member"
				}
				value, err := engine.Mutate(context.Background(), method, identity, nil, nil, neutronfind.Options{Headers: headers})
				if err != nil || value == nil || value.StatusCode != 203 || calls != 2 || hooks != 1 || client.MoreHeaders != nil {
					t.Fatal(value, err, calls, hooks, client)
				}
			})
		}
	}
}

func TestSourceMutationsObserveSelectedGuardWithoutFabricatingReceipts(t *testing.T) {
	for _, noop := range []bool{false, true} {
		t.Run(fmt.Sprint(noop), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/neutron/v2.0/")
			calls := 0
			cause := errors.New("selected leaf dependency changed")
			changed := false
			cloud.Provider.HTTPClient.Transport = policyTransport(func(*http.Request) (*http.Response, error) {
				calls++
				t.Fatal("guard failure sent HTTP")
				return nil, nil
			})
			engine, err := neutronfind.New(context.Background(), client, mutationProfile(), func(context.Context) error {
				if changed {
					return cause
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			changed = true
			var value *neutronfind.Record
			if noop {
				value, err = engine.ProjectSeed(context.Background(), map[string]json.RawMessage{"id": json.RawMessage(`"member"`)}, neutronfind.Options{})
			} else {
				value, err = engine.Mutate(context.Background(), http.MethodPost, "", nil, nil, neutronfind.Options{})
			}
			if value != nil || !errors.Is(err, cause) || calls != 0 {
				t.Fatal(value, err, calls)
			}
			changed = false
			if _, again := engine.ProjectSeed(context.Background(), nil, neutronfind.Options{}); !errors.Is(again, cause) {
				t.Fatal("guard restoration erased observed cause", again)
			}
			var receipt *resource.ResponseError
			if errors.As(err, &receipt) {
				t.Fatal("invented actual receipt", receipt)
			}
		})
	}
}

// Source opts into the same status contract for reads and mutations. Profiles
// that do not opt in retain the original accepted range on every method.
func TestMutateSourceStatusOptInKeepsBaselineAndPhysicalReceipts(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		for _, sourceStatus := range []bool{false, true} {
			for _, status := range []int{199, 399, 400} {
				t.Run(fmt.Sprintf("%s/source=%t/%d", method, sourceStatus, status), func(t *testing.T) {
					cloud := testcloud.New(t)
					client := cloud.Client("network", "/v2.0/")
					raw := `{"record":{"id":"actual"},"receipt":"physical"}`
					calls := 0
					cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
						calls++
						if req.Method != method {
							t.Fatal(req.Method)
						}
						if method == http.MethodDelete && mutationRequestBody(t, req) != "" {
							t.Fatal("DELETE body")
						}
						return mutationResponse(status, raw), nil
					})
					profile := mutationProfile()
					profile.SourceStatusPolicy = sourceStatus
					engine, err := neutronfind.New(context.Background(), client, profile, nil)
					if err != nil {
						t.Fatal(err)
					}
					id := "id"
					if method == http.MethodPost {
						id = ""
					}
					row, err := engine.Mutate(context.Background(), method, id, nil, nil, neutronfind.Options{})
					accepted := status < 400 && (status >= 200 || sourceStatus)
					if calls != 1 {
						t.Fatal(calls)
					}
					if accepted {
						if err != nil || row == nil || row.StatusCode != status || string(row.Envelope) != raw || row.Header.Get("X-Request-Id") != "mutation-actual" {
							t.Fatal(row, err)
						}
						if method == http.MethodDelete && (row.Resource != nil || row.Wire != nil) {
							t.Fatal("DELETE invented Resource", row)
						}
					} else {
						var actual gophercloud.ErrUnexpectedResponseCode
						if err == nil || !errors.As(err, &actual) || actual.Actual != status || string(actual.Body) != raw || actual.ResponseHeader.Get("X-Request-Id") != "mutation-actual" {
							t.Fatal(err, actual)
						}
					}
				})
			}
		}
	}
}
