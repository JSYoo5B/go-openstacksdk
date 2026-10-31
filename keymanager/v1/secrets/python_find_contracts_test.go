package secrets_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func secretFindRecordResponse(status int, body string) *http.Response {
	return testcloud.Response(status, body, http.Header{"X-Receipt": []string{"actual"}})
}
func secretFindRecordClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("key-manager", "/catalog/v1/")
	client.ResourceBase = cloud.Server.URL + "/barbican/v1/"
	return client
}
func secretFindRecordField(t *testing.T, value *resource.RawResource, key, want string) {
	t.Helper()
	if value == nil || string(value.Body[key]) != want {
		t.Fatalf("%s wanted %s: %#v", key, want, value)
	}
}

func TestPythonSecretFindRecordDirectDescriptorsAndOwnedReceipts(t *testing.T) {
	cloud := testcloud.New(t)
	client := secretFindRecordClient(cloud)
	client.MoreHeaders = map[string]string{"X-Project-Id": "project", "aCcEpT": "configured"}
	calls := 0
	metadata := `{"name":false,"id":null,"secret_ref":"https://foreign.example/secrets/passive","status":0,"bit_length":9007199254740993,"algorithm":[false],"created":null,"created_at":"alias loses","content_types":{"default":"text/plain"},"payload_content_type":"application/octet-stream","unknown":{"n":9007199254740993},"connection":null,"microversion":true,"_synchronized":false,"cls":null}`
	cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet || req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("X-Project-Id") != "project" {
			t.Fatal(req)
		}
		switch calls {
		case 1:
			if req.URL.Path != "/barbican/v1/secrets/request-id" || req.URL.RawQuery != "" {
				t.Fatal(req.URL)
			}
			return secretFindRecordResponse(201, metadata), nil
		case 2:
			if req.URL.Path != "/barbican/v1/secrets/request-id/payload" || req.Header.Get("Accept") != "text/plain" {
				t.Fatal(req.URL, req.Header)
			}
			return secretFindRecordResponse(202, "비밀\x00"), nil
		default:
			t.Fatal("followed response ref", req.URL)
			return nil, errors.New("extra HTTP")
		}
	}))
	value, err := secrets.New(client).FindRecord(context.Background(), "request-id")
	if err != nil || value == nil || value.RequestID != "request-id" || value.StatusCode != 201 || calls != 2 {
		t.Fatal(value, err, calls)
	}
	for key, want := range map[string]string{"name": "false", "id": "null", "status": "0", "bit_length": "9007199254740993", "algorithm": "[false]", "created_at": "null", "updated_at": "null", "expires_at": "null", "secret_id": `"passive"`, "location": "null", "payload": `"비밀\u0000"`} {
		secretFindRecordField(t, value.Resource, key, want)
	}
	if value.Wire.Body["payload"] != nil || value.Resource.Body["unknown"] != nil || string(value.Wire.Body["unknown"]) != `{"n":9007199254740993}` || string(value.Envelope) != metadata || value.Payload.Text == nil || *value.Payload.Text != "비밀\x00" || value.Payload.StatusCode != 202 {
		t.Fatal(value)
	}
	// The direct custom Body updater accepts and ignores constructor controls;
	// only Resource.list's existing(...) call treats their presence as an error.
	for key, want := range map[string]string{"connection": "null", "microversion": "true", "_synchronized": "false", "cls": "null"} {
		if value.Resource.Body[key] != nil || string(value.Wire.Body[key]) != want {
			t.Fatalf("direct constructor control %s changed response semantics: %#v", key, value)
		}
	}
	value.Resource.Body["status"][0] = '9'
	value.Envelope[0] = 'x'
	value.Header.Set("X-Receipt", "changed")
	value.Payload.Body[0] = 'x'
	if string(value.Wire.Body["status"]) != "0" || value.Wire.Header.Get("X-Receipt") != "actual" || *value.Payload.Text != "비밀\x00" || client.MoreHeaders["aCcEpT"] != "configured" {
		t.Fatal("receipt/config alias", value, client)
	}
}

func TestPythonSecretFindRecordPayloadSelectionAndBinaryDomain(t *testing.T) {
	for _, tc := range []struct {
		name, metadata, accept string
		payload                bool
	}{
		{"absent", `{}`, "", false},
		{"response content type is not a seed", `{"payload_content_type":"text/plain","payload":false}`, "", false},
		{"null default", `{"content_types":{"default":null}}`, "", false},
		{"empty default", `{"content_types":{"default":""}}`, "", true},
		{"bytes", `{"content_types":{"default":"application/octet-stream"}}`, "application/octet-stream", true},
		{"parameterized text is bytes", `{"content_types":{"default":"text/plain;charset=utf-8"}}`, "text/plain;charset=utf-8", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := secretFindRecordClient(cloud)
			calls := 0
			client.MoreHeaders = map[string]string{"Accept": "configured"}
			cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return secretFindRecordResponse(199, tc.metadata), nil
				}
				if !tc.payload || calls != 2 || req.Header.Get("Accept") != tc.accept {
					t.Fatal(req.Header, calls)
				}
				// Empty Accept remains an actual, explicit header, not default JSON.
				if _, exists := req.Header["Accept"]; !exists {
					t.Fatal("omitted explicit Accept")
				}
				return secretFindRecordResponse(399, string([]byte{0xff, 0, 0x42})), nil
			}))
			value, err := secrets.New(client).FindRecord(context.Background(), "s1")
			if err != nil || value == nil {
				t.Fatal(value, err)
			}
			secretFindRecordField(t, value.Resource, "id", `"s1"`)
			if tc.payload {
				if calls != 2 || value.Payload == nil || value.Payload.Text != nil || string(value.Payload.Body) != string([]byte{0xff, 0, 0x42}) {
					t.Fatal(value, calls)
				}
				secretFindRecordField(t, value.Resource, "payload", `"/wBC"`)
			} else if calls != 1 || value.Payload != nil {
				t.Fatal(value, calls)
			}
		})
	}
}

func TestPythonSecretFindRecordMetadataAndTextFailuresKeepProof(t *testing.T) {
	for _, tc := range []struct {
		name, metadata, payload string
		payloadHTTP             bool
	}{
		{"empty", "", "", false}, {"null", "null", "", false}, {"array", "[]", "", false}, {"invalid UTF8", string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}), "", false},
		{"null content types", `{"content_types":null}`, "", false}, {"array content types", `{"content_types":[]}`, "", false},
		{"missing default", `{"content_types":{}}`, "", false}, {"nonstring default", `{"content_types":{"default":false}}`, "", false},
		{"bad header", `{"content_types":{"default":"text/plain\r\nInjected: yes"}}`, "", false},
		{"bad text", `{"content_types":{"default":"text/plain"}}`, string([]byte{0xff}), true},
		{"invalid href", `{"secret_ref":"relative/ref"}`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := secretFindRecordClient(cloud)
			calls := 0
			cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return secretFindRecordResponse(200, tc.metadata), nil
				}
				if !tc.payloadHTTP || calls != 2 {
					t.Fatal("decode error continued HTTP", req.URL)
				}
				return secretFindRecordResponse(200, tc.payload), nil
			}))
			value, err := secrets.New(client).FindRecord(context.Background(), "s1")
			var proof *resource.ResponseError
			if err == nil || !errors.As(err, &proof) || proof.Header.Get("X-Receipt") != "actual" || value == nil || value.Resource != nil {
				t.Fatal(value, err)
			}
			want := tc.metadata
			if tc.payloadHTTP {
				want = tc.payload
			}
			if string(proof.Body) != want || string(value.Envelope) != tc.metadata || calls != 1+boolFindRecord(tc.payloadHTTP) {
				t.Fatal(value, proof, calls)
			}
		})
	}
}
func boolFindRecord(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestPythonSecretFindRecordCleanFallbackFromBothReads(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, stage := range []string{"metadata", "payload"} {
			t.Run(fmt.Sprintf("%s/%d", stage, status), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := secretFindRecordClient(cloud)
				calls := 0
				payloads := 0
				lists := 0
				cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.URL.Path == "/barbican/v1/secrets/name" {
						if stage == "metadata" {
							return secretFindRecordResponse(status, `{"error":"clean"}`), nil
						}
						return secretFindRecordResponse(200, `{"content_types":{"default":"text/plain"}}`), nil
					}
					if req.URL.Path == "/barbican/v1/secrets/name/payload" {
						payloads++
						return secretFindRecordResponse(status, `{"error":"clean payload"}`), nil
					}
					if req.URL.Path != "/barbican/v1/secrets" || req.URL.Query().Get("name") != "name" || req.Header.Get("Accept") != "application/json" {
						t.Fatal(req.URL, req.Header)
					}
					lists++
					if lists == 1 {
						return secretFindRecordResponse(302, `{"secrets":[{"id":"other","name":"other"}],"next":"?offset=1&limit=1"}`), nil
					}
					if lists != 2 || req.URL.Query().Get("offset") != "1" {
						t.Fatal(req.URL, lists)
					}
					return secretFindRecordResponse(200, `{"secrets":[{"name":"name","secret_ref":"https://passive.example/secrets/s2","content_types":false,"bit_length":0,"x":true}],"next":false}`), nil
				}))
				value, err := secrets.New(client).FindRecord(context.Background(), "name")
				if err != nil || value == nil || value.Payload != nil || value.RequestID != "" || lists != 2 || payloads != boolFindRecord(stage == "payload") || calls != 3+boolFindRecord(stage == "payload") {
					t.Fatal(value, err, calls, payloads, lists)
				}
				secretFindRecordField(t, value.Resource, "id", `"https://passive.example/secrets/s2"`)
				secretFindRecordField(t, value.Resource, "secret_id", `"s2"`)
				secretFindRecordField(t, value.Resource, "content_types", `{}`)
				if string(value.Wire.Body["content_types"]) != "false" || value.Resource.Body["x"] != nil || value.Header.Get("X-Receipt") != "actual" {
					t.Fatal(value)
				}
			})
		}
	}
}

func TestPythonSecretFindRecordMissingStrictHTTPAndFallbackPolicies(t *testing.T) {
	for _, tc := range []struct {
		name         string
		direct, list int
		options      []secrets.FindRecordOption
		wantErr      bool
		requests     int
	}{
		{"default missing", 404, 200, nil, false, 2},
		{"strict logical missing", 404, 200, []secrets.FindRecordOption{secrets.WithFindRecordIgnoreMissing(false)}, true, 2},
		{"list404 never missing", 404, 404, nil, true, 2},
		{"list403 never missing", 404, 403, nil, true, 2},
		{"401 never fallback", 401, 200, nil, true, 1},
		{"503 never fallback", 503, 200, nil, true, 1},
		{"notfoundonly403", 403, 200, []secrets.FindRecordOption{secrets.WithFindRecordFallback(resource.FindFallbackNotFoundOnly)}, true, 1},
		{"never clean404", 404, 200, []secrets.FindRecordOption{secrets.WithFindRecordFallback(resource.FindFallbackNever)}, false, 1},
		{"never strict404", 404, 200, []secrets.FindRecordOption{secrets.WithFindRecordFallback(resource.FindFallbackNever), secrets.WithFindRecordIgnoreMissing(false)}, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := secretFindRecordClient(cloud)
			calls := 0
			cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return secretFindRecordResponse(tc.direct, `{"error":"must not become a secret"}`), nil
				}
				if calls != 2 || req.URL.Query().Get("name") != "s1" {
					t.Fatal(req.URL, calls)
				}
				return secretFindRecordResponse(tc.list, `{"secrets":[]}`), nil
			}))
			value, err := secrets.New(client).FindRecord(context.Background(), "s1", tc.options...)
			if (err != nil) != tc.wantErr || err == nil && value != nil || value != nil && value.Resource != nil || calls != tc.requests {
				t.Fatal(value, err, calls)
			}
			if tc.list >= 400 && calls == 2 || calls == 1 && tc.wantErr {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.ResponseHeader.Get("X-Receipt") != "actual" {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPythonSecretFindRecordListOnlyExactMatchingAndLateFailures(t *testing.T) {
	for _, tc := range []struct {
		name, identity, first, second string
		wantErr                       bool
		duplicate                     bool
		requests                      int
	}{
		{"literal unsafe name", " name/% ", `{"secrets":{"id":false,"name":" name/% ","secret_ref":"https://elsewhere/secrets/id"},"next":null}`, "", false, false, 1},
		{"full alternate ref", "https://elsewhere/secrets/id", `{"secrets":[{"secret_ref":"https://elsewhere/secrets/id"}]}`, "", false, false, 1},
		{"literal null wins over ref", "https://elsewhere/secrets/id", `{"secrets":[{"id":null,"secret_ref":"https://elsewhere/secrets/id"}]}`, "", false, false, 1},
		{"duplicates across pages", "name with spaces", `{"secrets":[{"id":"1","name":"name with spaces"}],"next":"?offset=1"}`, `{"secrets":[{"id":"2","name":"name with spaces"}]}`, true, true, 2},
		{"late malformed row", "name with spaces", `{"secrets":[{"id":"1","name":"name with spaces"}],"next":"?offset=1"}`, `{"secrets":[null]}`, true, false, 2},
		{"constructor connection", "name with spaces", `{"secrets":[{"id":"1","name":"name with spaces","connection":null}]}`, "", true, false, 1},
		{"constructor microversion", "name with spaces", `{"secrets":[{"id":"1","name":"name with spaces","microversion":false}]}`, "", true, false, 1},
		{"constructor synchronized", "name with spaces", `{"secrets":[{"id":"1","name":"name with spaces","_synchronized":{}}]}`, "", true, false, 1},
		{"constructor cls null", "name with spaces", `{"secrets":[{"id":"1","name":"name with spaces","cls":null}]}`, "", true, false, 1},
		{"constructor cls false", "name with spaces", `{"secrets":[{"id":"1","name":"name with spaces","cls":false}]}`, "", true, false, 1},
		{"foreign next rejected", "name with spaces", `{"secrets":[{"name":"name with spaces"}],"next":"https://foreign.example/secrets?offset=1"}`, "", true, false, 1},
		{"cycle rejected", "name with spaces", `{"secrets":[{"name":"name with spaces"}],"next":"?offset=0"}`, "", true, false, 1},
		{"wrong next filter rejected", "name with spaces", `{"secrets":[{"name":"name with spaces"}],"next":"?offset=1&name=replace"}`, "", true, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := secretFindRecordClient(cloud)
			calls := 0
			cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != "/barbican/v1/secrets" || req.URL.Query().Get("name") != tc.identity {
					t.Fatal(req.URL)
				}
				if calls == 1 {
					return secretFindRecordResponse(200, tc.first), nil
				}
				if calls == 2 && tc.second != "" {
					return secretFindRecordResponse(200, tc.second), nil
				}
				t.Fatal("unexpected next", req.URL)
				return nil, errors.New("extra HTTP")
			}))
			value, err := secrets.New(client).FindRecord(context.Background(), tc.identity)
			if (err != nil) != tc.wantErr || calls != tc.requests || value != nil && err != nil && value.Resource != nil {
				t.Fatal(value, err, calls)
			}
			if strings.HasPrefix(tc.name, "constructor ") {
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != tc.first || proof.Header.Get("X-Receipt") != "actual" || value == nil || value.Resource != nil || string(value.Envelope) != tc.first || calls != 1 {
					t.Fatalf("constructor collision lost whole-page evidence: value=%#v proof=%#v error=%v requests=%d", value, proof, err, calls)
				}
			}
			if tc.duplicate && !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
			if tc.name == "literal null wins over ref" && value != nil {
				t.Fatal("null ID became alternate", value)
			}
			if tc.name == "full alternate ref" {
				secretFindRecordField(t, value.Resource, "id", `"https://elsewhere/secrets/id"`)
			}
		})
	}
}

func TestPythonSecretFindRecordOptionsCaptureAndPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	client := secretFindRecordClient(cloud)
	calls := 0
	applied := 0
	ignore := false
	bulk := secrets.WithFindRecordOptions(secrets.FindRecordOpts{IgnoreMissing: &ignore})
	ignore = true
	var retained *secrets.FindRecordOpts
	custom := func(value *secrets.FindRecordOpts) error { applied++; retained = value; return nil }
	cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		*retained.IgnoreMissing = true
		if calls == 1 {
			return secretFindRecordResponse(404, `{"error":"missing"}`), nil
		}
		return secretFindRecordResponse(200, `{"secrets":[]}`), nil
	}))
	value, err := secrets.New(client).FindRecord(context.Background(), "s1", bulk, custom)
	if value == nil || value.Resource != nil || value.Wire != nil || string(value.Envelope) != `{"secrets":[]}` || value.StatusCode != 200 || value.Header.Get("X-Receipt") != "actual" || !errors.Is(err, resource.ErrNotFound) || calls != 2 || applied != 1 {
		t.Fatal(value, err, calls, applied)
	}
	for _, tc := range []struct {
		name, identity string
		option         secrets.FindRecordOption
		canceled       bool
	}{
		{"nil option", "s1", nil, false},
		{"invalid fallback", "s1", secrets.WithFindRecordFallback(resource.FindFallbackPolicy(99)), false},
		{"bad identity", "\x00", secrets.WithFindRecordIgnoreMissing(true), false},
		{"unsafe never", "name with space", secrets.WithFindRecordFallback(resource.FindFallbackNever), false},
		{"source replaced", "s1", func(*secrets.FindRecordOpts) error { client.ResourceBase = cloud.Server.URL + "/changed/"; return nil }, false},
		{"callback error", "s1", func(*secrets.FindRecordOpts) error { return resource.ErrUnsupported }, false},
		{"canceled callback", "s1", secrets.WithFindRecordIgnoreMissing(true), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client.ResourceBase = cloud.Server.URL + "/barbican/v1/"
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("preflight cause")
			if tc.canceled {
				cancel(cause)
			}
			before := calls
			value, err := secrets.New(client).FindRecord(ctx, tc.identity, tc.option)
			if value != nil || err == nil || calls != before {
				t.Fatal(value, err, calls, before)
			}
			if tc.canceled && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
		})
	}
	var nilAPI *secrets.API
	if _, err := nilAPI.FindRecord(context.Background(), "s1"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := secrets.New(client).FindRecord(nil, "s1"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

type secretFindRecordFaultBody struct {
	data              []byte
	readErr, closeErr error
	read, close       func()
	done              bool
}

func (b *secretFindRecordFaultBody) Read(dst []byte) (int, error) {
	if b.done {
		return 0, io.EOF
	}
	b.done = true
	if b.read != nil {
		b.read()
	}
	return copy(dst, b.data), b.readErr
}
func (b *secretFindRecordFaultBody) Close() error {
	if b.close != nil {
		b.close()
	}
	return b.closeErr
}

func TestPythonSecretFindRecordPhysicalFailuresPreventFallbackAndReplay(t *testing.T) {
	for _, stage := range []string{"metadata", "payload", "list"} {
		for _, status := range []int{200, 400, 403, 404, 429, 500, 503} {
			t.Run(fmt.Sprintf("%s/%d", stage, status), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := secretFindRecordClient(cloud)
				calls := 0
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readErr, closeErr, cause := errors.New("physical Read"), errors.New("physical Close"), errors.New("physical cancellation")
				body := `{"actual":"partial receipt"}`
				cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if stage == "payload" && calls == 1 {
						return secretFindRecordResponse(200, `{"content_types":{"default":"text/plain"},"x":"metadata proof"}`), nil
					}
					if stage == "list" && req.URL.Path != "/barbican/v1/secrets" {
						t.Fatal(req.URL)
					}
					response := secretFindRecordResponse(status, body)
					response.Body = &testcloud.Body{Data: body, ReadErr: readErr, CloseErr: closeErr,
						OnRead:  func() { client.ResourceBase = cloud.Server.URL + "/transient/" },
						OnClose: func() { client.ResourceBase = cloud.Server.URL + "/barbican/v1/"; cancel(cause) }}
					return response, nil
				}))
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
				identity := "s1"
				if stage == "list" {
					identity = "name with spaces"
				}
				value, err := secrets.New(client).FindRecord(ctx, identity)
				for _, expected := range []error{readErr, closeErr, cause, context.Canceled, resource.ErrInvalidOption} {
					if !errors.Is(err, expected) {
						t.Fatalf("lost %v: %v", expected, err)
					}
				}
				if calls != 1+boolFindRecord(stage == "payload") || value != nil && value.Resource != nil {
					t.Fatal(value, err, calls)
				}
				if status >= 400 {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Receipt") != "actual" {
						t.Fatal(native, err)
					}
				} else {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Receipt") != "actual" || value == nil {
						t.Fatal(value, proof, err)
					}
				}
				if stage == "payload" && (value == nil || value.Wire == nil || string(value.Wire.Body["x"]) != `"metadata proof"`) {
					t.Fatal("lost first receipt", value)
				}
			})
		}
	}
}

func TestPythonSecretFindRecordCleanRetryAndFixedAccept(t *testing.T) {
	for _, stage := range []string{"metadata", "payload", "list"} {
		for _, tamper := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/tamper=%t", stage, tamper), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := secretFindRecordClient(cloud)
				calls := 0
				failed := false
				cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if stage == "payload" && req.URL.Path == "/barbican/v1/secrets/s1" {
						return secretFindRecordResponse(200, `{"content_types":{"default":"text/plain"}}`), nil
					}
					if !failed {
						failed = true
						return secretFindRecordResponse(503, `{"error":"clean retry"}`), nil
					}
					if stage == "payload" {
						if req.Header.Get("Accept") != "text/plain" {
							t.Fatal(req.Header)
						}
						return secretFindRecordResponse(200, "payload"), nil
					}
					if stage == "list" {
						if req.Header.Get("Accept") != "application/json" {
							t.Fatal(req.Header)
						}
						return secretFindRecordResponse(200, `{"secrets":[{"name":"name with spaces"}]}`), nil
					}
					return secretFindRecordResponse(200, `{}`), nil
				}))
				cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					if tamper {
						if stage == "metadata" {
							opts.JSONBody = json.RawMessage(`{"replace":true}`)
						} else {
							opts.MoreHeaders["Accept"] = "tampered"
						}
					}
					return nil
				}
				identity := "s1"
				if stage == "list" {
					identity = "name with spaces"
				}
				value, err := secrets.New(client).FindRecord(context.Background(), identity)
				if tamper {
					if !errors.Is(err, resource.ErrInvalidOption) || calls != 1+boolFindRecord(stage == "payload") || value != nil && value.Resource != nil {
						t.Fatal(value, err, calls)
					}
				} else if err != nil || value == nil || calls != 2+boolFindRecord(stage == "payload") {
					t.Fatal(value, err, calls)
				}
			})
		}
	}
}

func TestPythonSecretFindRecordTransportAndRetryErrorsStayTerminal(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(fmt.Sprint(retry), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := secretFindRecordClient(cloud)
			calls := 0
			cause := errors.New("transport or retry cause")
			cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if retry {
					return secretFindRecordResponse(404, `{"error":"original"}`), nil
				}
				return nil, cause
			}))
			if retry {
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return cause }
			}
			value, err := secrets.New(client).FindRecord(context.Background(), "s1")
			if !errors.Is(err, cause) || value != nil || calls != 1 {
				t.Fatal(value, err, calls)
			}
			if retry {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `{"error":"original"}` {
					t.Fatal(err)
				}
			}
		})
	}
}
