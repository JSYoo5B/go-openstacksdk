package tokens_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonTokenValidateQueriesHeadersAndOwnedRecord(t *testing.T) {
	cases := []struct {
		name, query string
		options     []tokens.ValidateTokenOption
	}{
		{"defaults", "", nil},
		{"false", "", []tokens.ValidateTokenOption{tokens.WithValidateTokenNoCatalog(false), tokens.WithValidateTokenAllowExpired(false)}},
		{"catalog", "nocatalog=True", []tokens.ValidateTokenOption{tokens.WithValidateTokenNoCatalog(true)}},
		{"expired", "allow_expired=True", []tokens.ValidateTokenOption{tokens.WithValidateTokenAllowExpired(true)}},
		{"both", "allow_expired=True&nocatalog=True", []tokens.ValidateTokenOption{tokens.WithValidateTokenOpts(tokens.ValidateTokenOpts{NoCatalog: true, AllowExpired: true})}},
		{"replace", "", []tokens.ValidateTokenOption{tokens.WithValidateTokenNoCatalog(true), tokens.WithValidateTokenOpts(tokens.ValidateTokenOpts{})}},
		{"last wins", "allow_expired=True", []tokens.ValidateTokenOption{tokens.WithValidateTokenNoCatalog(true), tokens.WithValidateTokenNoCatalog(false), tokens.WithValidateTokenAllowExpired(true)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []nativeTokenCall
			client := nativeTokenClient(t, &calls, func(req *http.Request) nativeTokenReply {
				if req.URL.RawQuery != tc.query {
					t.Errorf("query=%q want %q", req.URL.RawQuery, tc.query)
				}
				return nativeTokenReply{203, http.Header{"X-Subject-Token": {"response-subject"}, "X-Request-Id": {"actual"}}, `{"token":{"id":"body-id","expires_at":"non-RFC3339 literal","issued_at":9007199254740993,"user":{"id":"u"},"x-extension":9007199254740993}}`}
			})
			client.MoreHeaders = map[string]string{"x-subject-token": "configured-other", "X-Ordinary": "stable"}
			before := maps.Clone(client.MoreHeaders)
			value, err := tokens.New(client).ValidateToken(context.Background(), "submitted / token", tc.options...)
			if err != nil || value == nil || value.SubjectToken == nil || *value.SubjectToken != "response-subject" {
				t.Fatal(value, err)
			}
			if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != "/keystone/v3/auth/tokens" || calls[0].subjectToken != "submitted / token" || calls[0].authToken != "test-token" || calls[0].body != "" {
				t.Fatal(calls)
			}
			if value.Resource.Body["id"] == nil || string(value.Resource.Body["id"]) != `"body-id"` || string(value.Resource.Body["expires_at"]) != `"non-RFC3339 literal"` || string(value.Resource.Body["issued_at"]) != "9007199254740993" {
				t.Fatal(value.Resource.Body)
			}
			if string(value.Resource.Body["location"]) != "null" || string(value.Resource.Body["catalog"]) != "null" || string(value.Resource.Body["subject_token"]) != `"response-subject"` {
				t.Fatal(value.Resource.Body)
			}
			if _, exists := value.Resource.Body["x-extension"]; exists {
				t.Fatal("unknown field entered declared view")
			}
			if string(value.Wire.Body["x-extension"]) != "9007199254740993" || value.StatusCode != 203 || value.Header.Get("X-Request-Id") != "actual" {
				t.Fatal(value)
			}
			value.Resource.Body["id"][1] = 'X'
			value.Resource.Header.Set("X-Request-Id", "changed")
			value.Wire.Body["x-extension"][0] = '1'
			value.Wire.Header.Set("X-Request-Id", "changed wire")
			value.Envelope[0] = 'X'
			if string(value.Wire.Body["id"]) != `"body-id"` || value.Header.Get("X-Request-Id") != "actual" || !bytes.HasPrefix(value.Resource.Body["issued_at"], []byte("9007")) {
				t.Fatal("response channels share bytes", value)
			}
			if !reflect.DeepEqual(client.MoreHeaders, before) {
				t.Fatal("changed shared source headers", client.MoreHeaders)
			}
		})
	}
}

func TestPythonTokenValidateDescriptorConversionsAndToleratedBodies(t *testing.T) {
	cases := []struct {
		name, body string
		code       int
		want       map[string]string
	}{
		{"raw root", `{"expires_at":false,"issued_at":{"raw":1},"methods":"password","audit_ids":9,"user":[1],"project":"p","domain":null,"system":true,"is_domain":"False","catalog":{"id":"one"},"roles":[null,3,{"id":"r"}],"self":"passive"}`, 200, map[string]string{"expires_at": "false", "issued_at": `{"raw":1}`, "methods": `["password"]`, "audit_ids": "[9]", "user": "{}", "project": "{}", "domain": "null", "system": "{}", "is_domain": "true", "catalog": `[{"id":"one"}]`, "roles": `[{},{},{"id":"r"}]`}},
		{"empty arrays", `{"token":{"audit_ids":[],"catalog":[],"methods":[],"roles":[],"is_domain":[],"user":{}}}`, 299, map[string]string{"audit_ids": "[]", "catalog": "[]", "methods": "[]", "roles": "[]", "is_domain": "false", "user": "{}"}},
		{"nulls", `{"token":{"audit_ids":null,"catalog":null,"is_domain":null,"methods":null,"roles":null}}`, 399, map[string]string{"audit_ids": "null", "catalog": "null", "is_domain": "null", "methods": "null", "roles": "null"}},
		{"empty", "", 204, map[string]string{"expires_at": "null", "id": "null"}},
		{"invalid JSON", "not json", 202, map[string]string{"expires_at": "null", "id": "null"}},
		{"no envelope", `{}`, 201, map[string]string{"expires_at": "null", "id": "null"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []nativeTokenCall
			api := tokens.New(nativeTokenClient(t, &calls, func(*http.Request) nativeTokenReply { return nativeTokenReply{tc.code, nil, tc.body} }))
			value, err := api.ValidateToken(context.Background(), "")
			if err != nil || value == nil || value.StatusCode != tc.code || value.SubjectToken != nil {
				t.Fatal(value, err)
			}
			for key, want := range tc.want {
				if string(value.Resource.Body[key]) != want {
					t.Fatalf("%s=%s want %s", key, value.Resource.Body[key], want)
				}
			}
			if json.Valid([]byte(tc.body)) && value.Wire == nil {
				t.Fatal("missing wire for object")
			}
			if !json.Valid([]byte(tc.body)) && value.Wire != nil {
				t.Fatal("invented wire")
			}
		})
	}
}

func TestPythonTokenValidateRepresentationAndHTTPFailures(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `"text"`, `{"token":null}`, `{"token":[]}`} {
		t.Run(body, func(t *testing.T) {
			var calls []nativeTokenCall
			api := tokens.New(nativeTokenClient(t, &calls, func(*http.Request) nativeTokenReply {
				return nativeTokenReply{200, http.Header{"X-Request-Id": {"bad-shape"}}, body}
			}))
			value, err := api.ValidateToken(context.Background(), "subject")
			var proof *resource.ResponseError
			if value == nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "bad-shape" || len(calls) != 1 {
				t.Fatal(value, err, proof, calls)
			}
			nativeTokenOperation(t, err, "ValidateToken", "tokens")
		})
	}
	for _, code := range []int{400, 401, 403, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeTokenCall
			api := tokens.New(nativeTokenClient(t, &calls, func(*http.Request) nativeTokenReply {
				return nativeTokenReply{code, http.Header{"X-Request-Id": {"rejected"}}, "rejected body"}
			}))
			value, err := api.ValidateToken(context.Background(), "subject")
			var native gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "rejected body" || native.ResponseHeader.Get("X-Request-Id") != "rejected" {
				t.Fatal(value, err, native)
			}
		})
	}
}

func TestPythonTokenCheckAllHTTPStatusesAndExpiredQuery(t *testing.T) {
	for _, code := range []int{200, 201, 204, 299, 301, 399, 400, 401, 403, 404, 429, 500, 503, 599} {
		for _, expired := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%t", code, expired), func(t *testing.T) {
				var calls []nativeTokenCall
				api := tokens.New(nativeTokenClient(t, &calls, func(req *http.Request) nativeTokenReply {
					want := ""
					if expired {
						want = "allow_expired=True"
					}
					if req.URL.RawQuery != want {
						t.Errorf("query=%q want %q", req.URL.RawQuery, want)
					}
					return nativeTokenReply{code, nil, "not JSON"}
				}))
				got, err := api.CheckToken(context.Background(), "subject", tokens.WithCheckTokenAllowExpired(expired))
				if err != nil || got != (code == 200) || len(calls) != 1 || calls[0].method != http.MethodHead || calls[0].subjectToken != "subject" {
					t.Fatal(got, err, calls)
				}
			})
		}
	}
	var calls []nativeTokenCall
	api := tokens.New(nativeTokenClient(t, &calls, func(req *http.Request) nativeTokenReply {
		if req.URL.RawQuery != "" {
			t.Error(req.URL.RawQuery)
		}
		return nativeTokenReply{200, nil, ""}
	}))
	if got, err := api.CheckToken(context.Background(), "subject", tokens.WithCheckTokenAllowExpired(true), tokens.WithCheckTokenOpts(tokens.CheckTokenOpts{})); err != nil || !got {
		t.Fatal(got, err)
	}
}

func TestPythonTokenRevokeAcceptedCodesAndOpaqueBody(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 400, 403, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeTokenCall
			api := tokens.New(nativeTokenClient(t, &calls, func(req *http.Request) nativeTokenReply {
				if req.URL.RawQuery != "" {
					t.Error(req.URL.RawQuery)
				}
				return nativeTokenReply{code, nil, "opaque / not JSON"}
			}))
			err := api.RevokeToken(context.Background(), "subject")
			if code < 400 {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || string(native.Body) != "opaque / not JSON" {
					t.Fatal(err, native)
				}
				nativeTokenOperation(t, err, "RevokeToken", "tokens")
			}
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].subjectToken != "subject" || calls[0].authToken != "test-token" {
				t.Fatal(calls)
			}
		})
	}
}

func TestPythonTokenOptionsPreflightAndSourceCapture(t *testing.T) {
	var calls []nativeTokenCall
	client := nativeTokenClient(t, &calls, func(req *http.Request) nativeTokenReply {
		if req.URL.RawQuery != "nocatalog=True" {
			t.Error("retained callback changed captured options", req.URL.RawQuery)
		}
		return nativeTokenReply{200, nil, `{}`}
	})
	api := tokens.New(client)
	marker := errors.New("option failure")
	if _, err := api.ValidateToken(context.Background(), "s", func(*tokens.ValidateTokenOpts) error { return marker }); !errors.Is(err, marker) {
		t.Fatal(err)
	}
	if _, err := api.ValidateToken(context.Background(), "s", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.CheckToken(context.Background(), "s", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.ValidateToken(nil, "s"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := api.RevokeToken(context.Background(), "s\r\ninjected: yes"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller stopped")
	cancel(cause)
	if _, err := api.CheckToken(ctx, "s"); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
	var absent *tokens.API
	if err := absent.RevokeToken(context.Background(), "s"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	original := client.Endpoint
	if _, err := api.ValidateToken(context.Background(), "s", func(*tokens.ValidateTokenOpts) error { client.Endpoint = original + "changed/"; return nil }, func(*tokens.ValidateTokenOpts) error { client.Endpoint = original; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	client.Endpoint = original
	for _, operation := range []string{"validate", "check"} {
		t.Run(operation+" callback cancellation and error", func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			cause := errors.New("callback canceled caller")
			callbackErr := errors.New("callback failed after cancellation")
			invocations := 0
			apply := func() error {
				invocations++
				cancel(cause)
				return callbackErr
			}
			var err error
			if operation == "validate" {
				_, err = api.ValidateToken(ctx, "s", func(*tokens.ValidateTokenOpts) error { return apply() }, func(*tokens.ValidateTokenOpts) error { t.Error("continued after failed callback"); return nil })
			} else {
				_, err = api.CheckToken(ctx, "s", func(*tokens.CheckTokenOpts) error { return apply() }, func(*tokens.CheckTokenOpts) error { t.Error("continued after failed callback"); return nil })
			}
			if !errors.Is(err, callbackErr) || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || invocations != 1 || len(calls) != 0 {
				t.Fatal("lost callback/cancellation cause or sent HTTP", err, invocations, calls)
			}
		})
		t.Run(operation+" callback source change and error", func(t *testing.T) {
			t.Cleanup(func() { client.Endpoint = original })
			callbackErr := errors.New("callback failed after source change")
			invocations := 0
			apply := func() error {
				invocations++
				client.Endpoint = original + "changed/"
				return callbackErr
			}
			var err error
			if operation == "validate" {
				_, err = api.ValidateToken(context.Background(), "s", func(*tokens.ValidateTokenOpts) error { return apply() }, func(*tokens.ValidateTokenOpts) error { t.Error("continued after failed callback"); return nil })
			} else {
				_, err = api.CheckToken(context.Background(), "s", func(*tokens.CheckTokenOpts) error { return apply() }, func(*tokens.CheckTokenOpts) error { t.Error("continued after failed callback"); return nil })
			}
			if !errors.Is(err, callbackErr) || !errors.Is(err, resource.ErrInvalidOption) || invocations != 1 || len(calls) != 0 {
				t.Fatal("lost callback/source cause or sent HTTP", err, invocations, calls)
			}
		})
	}
	if len(calls) != 0 {
		t.Fatal("preflight sent HTTP", calls)
	}
	var retained *tokens.ValidateTokenOpts
	if _, err := api.ValidateToken(context.Background(), "s", func(config *tokens.ValidateTokenOpts) error { config.NoCatalog = true; retained = config; return nil }, func(*tokens.ValidateTokenOpts) error { retained.NoCatalog = false; return nil }); err != nil {
		t.Fatal(err)
	}
	// Mutating a retained callback pointer cannot mutate the owned next policy.
	if len(calls) != 1 {
		t.Fatal(calls)
	}
}

func TestPythonTokenTransportCancellationAndAcceptedBodyErrors(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("identity", "/v3/")
	api := tokens.New(client)
	transportCause := errors.New("transport failed")
	client.HTTPClient.Transport = nativeTokenTransport(func(*http.Request) (*http.Response, error) { return nil, transportCause })
	if got, err := api.CheckToken(context.Background(), "s"); got || !errors.Is(err, transportCause) {
		t.Fatal(got, err)
	}
	if err := api.RevokeToken(context.Background(), "s"); !errors.Is(err, transportCause) {
		t.Fatal(err)
	}
	if got, err := api.ValidateToken(context.Background(), "s"); got != nil || !errors.Is(err, transportCause) {
		t.Fatal(got, err)
	}
	readCause := errors.New("read failed")
	closeCause := errors.New("close failed")
	client.HTTPClient.Transport = nativeTokenTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"partial"}}, Body: &tokenActionFaultBody{data: []byte(`{"token":`), read: readCause, close: closeCause}}, nil
	})
	value, err := api.ValidateToken(context.Background(), "s")
	var proof *resource.ResponseError
	if value == nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(value.Envelope) != `{"token":` || proof.Header.Get("X-Request-Id") != "partial" {
		t.Fatal(value, err, proof)
	}
}

type tokenActionFaultBody struct {
	data        []byte
	read, close error
}

func (b *tokenActionFaultBody) Read(p []byte) (int, error) {
	if len(b.data) != 0 {
		n := copy(p, b.data)
		b.data = b.data[n:]
		return n, nil
	}
	return 0, b.read
}
func (b *tokenActionFaultBody) Close() error { return b.close }

func TestPythonTokenSelectedHeaderCannotChangeDuringRetry(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("identity", "/v3/")
	attempts := 0
	client.HTTPClient.Transport = nativeTokenTransport(func(req *http.Request) (*http.Response, error) {
		attempts++
		if req.Header.Get("X-Subject-Token") != "selected" {
			t.Error("subject retargeted")
		}
		return &http.Response{StatusCode: 503, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("original"))}, nil
	})
	client.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		opts.MoreHeaders["X-Subject-Token"] = "different"
		return nil
	}
	_, err := tokens.New(client).ValidateToken(context.Background(), "selected")
	if !errors.Is(err, resource.ErrInvalidOption) || attempts != 1 {
		t.Fatal(err, attempts)
	}
}
