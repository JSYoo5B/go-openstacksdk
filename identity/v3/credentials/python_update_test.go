package credentials_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCredentialUpdateRawBodyAndResponseOverlay(t *testing.T) {
	var calls []nativeCredCall
	api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Header.Get("X-Owned") != "original" {
			t.Error(req.Header)
		}
		return 200, `{"credential":{"id":"server-id","blob":{"count":9007199254740993},"extension":[false,null]}}`
	})
	fields := map[string]json.RawMessage{
		"id":         json.RawMessage(`"different-id"`),
		"blob":       json.RawMessage(`{"count":0}`),
		"name":       json.RawMessage(`""`),
		"project_id": json.RawMessage(`null`),
		"type":       json.RawMessage(`false`),
		"user_id":    json.RawMessage(`0`),
		"unknown":    json.RawMessage(`{broken`),
	}
	headers := map[string]string{"X-Owned": "original"}
	owned := credentials.WithRecordUpdateOpts(credentials.RecordUpdateOpts{Fields: fields, Headers: headers})
	fields["blob"][9] = '9'
	fields["user_id"] = json.RawMessage(`"mutated"`)
	headers["X-Owned"] = "mutated"
	value, err := api.UpdateRecord(context.Background(), "selected", owned,
		credentials.WithRecordUpdateField("ignored", make(chan int)))
	if err != nil {
		t.Fatal(value, err)
	}
	wantBody := `{"credential":{"blob":{"count":0},"name":"","project_id":null,"type":false,"user_id":0}}`
	if len(calls) != 1 || calls[0].method != http.MethodPatch || calls[0].path != "/keystone/v3/credentials/selected" || calls[0].query != "" || calls[0].body != wantBody {
		t.Fatal(calls)
	}
	for key, want := range map[string]string{
		"id": `"server-id"`, "blob": `{"count":9007199254740993}`, "name": `""`,
		"project_id": `null`, "type": `false`, "user_id": `0`, "location": `null`,
	} {
		credentialRecordField(t, value, key, want)
	}
	if len(value.Resource.Body) != 7 || value.StatusCode != 200 || string(value.Wire.Body["extension"]) != `[false,null]` {
		t.Fatal(value)
	}
	value.Resource.Body["blob"][9] = 'X'
	if string(value.Wire.Body["blob"]) != `{"count":9007199254740993}` || !strings.Contains(string(value.Envelope), "9007199254740993") {
		t.Fatal("response channels alias")
	}
	value.Header.Set("X-Owned", "changed")
	if value.Wire.Header.Get("X-Owned") != "" || value.Resource.Header.Get("X-Owned") != "" {
		t.Fatal("header channels alias")
	}
}

func TestPythonCredentialUpdateNoOpAndLiteralID(t *testing.T) {
	var calls []nativeCredCall
	api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 204, "" })
	cases := []struct {
		name    string
		options []credentials.RecordUpdateOption
	}{
		{"empty", nil},
		{"unknown only", []credentials.RecordUpdateOption{credentials.WithRecordUpdateField("extension", make(chan int))}},
		{"id only", []credentials.RecordUpdateOption{credentials.WithRecordUpdateField("id", "different")}},
		{"raw ignored", []credentials.RecordUpdateOption{credentials.WithRecordUpdateOpts(credentials.RecordUpdateOpts{Fields: map[string]json.RawMessage{"id": json.RawMessage(`{broken`), "unknown": json.RawMessage(`{broken`)}})}},
		{"policy header only", []credentials.RecordUpdateOption{credentials.WithRecordUpdateHeaders(map[string]string{"X-Custom": "value"})}},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			value, err := api.UpdateRecord(context.Background(), "selected", item.options...)
			if err != nil || len(calls) != 0 {
				t.Fatal(value, err, calls)
			}
			credentialRecordField(t, value, "id", `"selected"`)
			credentialRecordField(t, value, "blob", `null`)
			if value.Wire != nil || len(value.Envelope) != 0 || value.StatusCode != 0 || len(value.Header) != 0 {
				t.Fatal("no-op fabricated HTTP evidence", value)
			}
		})
	}
	value, err := api.UpdateRecord(context.Background(), "folder/name%",
		credentials.WithRecordUpdateField("blob", []any{}),
		credentials.WithRecordUpdateField("type", map[string]any{}))
	if err != nil || len(calls) != 1 || calls[0].body != `{"credential":{"blob":[],"type":{}}}` {
		t.Fatal(value, err, calls)
	}
	credentialRecordField(t, value, "id", `"folder/name%"`)
	credentialRecordField(t, value, "blob", `[]`)
}

func TestPythonCredentialUpdateTranslationAndStatuses(t *testing.T) {
	for _, item := range []struct {
		name, body string
		valid      bool
	}{
		{"singular", `{"credential":{"blob":false}}`, true},
		{"root", `{"name":"response","extension":1}`, true},
		{"empty", ``, true}, {"malformed", `{broken`, true}, {"object", `{}`, true},
		{"array", `[]`, false}, {"null", `null`, false}, {"wrapped null", `{"credential":null}`, false},
	} {
		t.Run(item.name, func(t *testing.T) {
			var calls []nativeCredCall
			api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, item.body })
			value, err := api.UpdateRecord(context.Background(), "selected", credentials.WithRecordUpdateField("blob", "requested"))
			if (err == nil) != item.valid || len(calls) != 1 || value == nil || string(value.Envelope) != item.body || value.StatusCode != 200 {
				t.Fatal(value, err, calls)
			}
			if !item.valid {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != item.body {
					t.Fatal(err, proof)
				}
				return
			}
			credentialRecordField(t, value, "id", `"selected"`)
			expectedBlob := `"requested"`
			if item.name == "singular" {
				expectedBlob = `false`
			}
			credentialRecordField(t, value, "blob", expectedBlob)
		})
	}
	for _, code := range []int{200, 201, 202, 204, 206, 299, 301, 399, 400, 403, 404, 409, 422, 429, 500, 503, 599, 600} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeCredCall
			api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			value, err := api.UpdateRecord(context.Background(), "selected", credentials.WithRecordUpdateField("blob", nil))
			if len(calls) != 1 {
				t.Fatal(calls)
			}
			if code < 400 {
				if err != nil {
					t.Fatal(value, err)
				}
				credentialRecordField(t, value, "blob", `null`)
				credentialRecordField(t, value, "id", `"selected"`)
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodPatch || string(native.Body) != `{}` {
					t.Fatal(value, err, native)
				}
				nativeCredOperation(t, err, "UpdateRecord")
			}
		})
	}
}

func TestPythonCredentialUpdateOptionOwnershipAndPreflight(t *testing.T) {
	var calls []nativeCredCall
	api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Header.Get("X-Owned") != "old" {
			t.Error(req.Header)
		}
		return 200, `{}`
	})
	var retained *credentials.RecordUpdateOpts
	value, err := api.UpdateRecord(context.Background(), "selected",
		func(policy *credentials.RecordUpdateOpts) error {
			retained = policy
			policy.Fields = map[string]json.RawMessage{"blob": json.RawMessage(`"old"`)}
			policy.Headers = map[string]string{"X-Owned": "old"}
			return nil
		},
		func(*credentials.RecordUpdateOpts) error {
			retained.Fields["blob"] = json.RawMessage(`"changed"`)
			retained.Headers["X-Owned"] = "changed"
			return nil
		})
	if err != nil || len(calls) != 1 || calls[0].body != `{"credential":{"blob":"old"}}` {
		t.Fatal(value, err, calls)
	}
	for _, item := range []struct {
		name    string
		id      string
		options []credentials.RecordUpdateOption
	}{
		{"nil option", "a", []credentials.RecordUpdateOption{nil}},
		{"bad JSON", "a", []credentials.RecordUpdateOption{credentials.WithRecordUpdateOpts(credentials.RecordUpdateOpts{Fields: map[string]json.RawMessage{"blob": json.RawMessage(`{bad`)}})}},
		{"bad UTF8", "a", []credentials.RecordUpdateOption{credentials.WithRecordUpdateOpts(credentials.RecordUpdateOpts{Fields: map[string]json.RawMessage{"blob": json.RawMessage{'"', 0xff, '"'}}})}},
		{"protected header", "a", []credentials.RecordUpdateOption{credentials.WithRecordUpdateHeaders(map[string]string{"X-Auth-Token": "replacement"})}},
		{"dot target", "..", nil}, {"blank target", "  ", nil},
	} {
		t.Run(item.name, func(t *testing.T) {
			before := len(calls)
			_, err := api.UpdateRecord(context.Background(), item.id, item.options...)
			if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
				t.Fatal(err, calls)
			}
			nativeCredOperation(t, err, "UpdateRecord")
		})
	}
	t.Run("callback cancel and error", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		callbackCause, cancelCause := errors.New("callback"), errors.New("caller")
		before := len(calls)
		_, err := api.UpdateRecord(ctx, "a", func(*credentials.RecordUpdateOpts) error { cancel(cancelCause); return callbackCause })
		if !errors.Is(err, callbackCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || len(calls) != before {
			t.Fatal(err, calls)
		}
	})
	t.Run("callback source change and error", func(t *testing.T) {
		client := api.RawClient()
		endpoint := client.Endpoint
		defer func() { client.Endpoint = endpoint }()
		cause := errors.New("callback")
		before := len(calls)
		_, err := api.UpdateRecord(context.Background(), "a", func(*credentials.RecordUpdateOpts) error { client.Endpoint += "changed/"; return cause })
		if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(err, calls)
		}
	})
	t.Run("nil API and context", func(t *testing.T) {
		var missing *credentials.API
		_, err := missing.UpdateRecord(context.Background(), "a")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		_, err = api.UpdateRecord(nil, "a")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	})
}

func TestPythonCredentialUpdatePhysicalResponseFailures(t *testing.T) {
	for _, status := range []int{200, 404, 409, 500, 600} {
		for _, kind := range []string{"read", "close", "read and close", "cancel after read", "cancel after close", "source restored by close"} {
			t.Run(fmt.Sprintf("%d/%s", status, kind), func(t *testing.T) {
				var serverCalls []nativeCredCall
				api, cloud := nativeCredAPI(t, &serverCalls, func(*http.Request) (int, string) { return 200, `{}` })
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("physical Read"), errors.New("physical Close"), errors.New("caller cancel")
				body := &credentialRecordFaultBody{data: []byte(`{"credential":`), read: io.EOF}
				if kind == "read" || kind == "read and close" {
					body.read = readCause
				}
				if kind == "close" || kind == "read and close" {
					body.close = closeCause
				}
				if kind == "cancel after read" {
					body.onRead = func() { cancel(cancelCause) }
				}
				if kind == "cancel after close" {
					body.onClose = func() { cancel(cancelCause) }
				}
				endpoint := api.RawClient().Endpoint
				if kind == "source restored by close" {
					body.onRead = func() { api.RawClient().Endpoint = endpoint + "changed/" }
					body.onClose = func() { api.RawClient().Endpoint = endpoint }
				}
				attempts := 0
				cloud.Provider.HTTPClient.Transport = nativeCredTransport(func(req *http.Request) (*http.Response, error) {
					attempts++
					if req.Method != http.MethodPatch {
						t.Error(req.Method)
					}
					return &http.Response{StatusCode: status, Header: http.Header{"X-Request-Id": {"physical-proof"}}, Body: body}, nil
				})
				value, err := api.UpdateRecord(ctx, "a", credentials.WithRecordUpdateField("blob", "updated"))
				if err == nil || attempts != 1 || body.closes != 1 {
					t.Fatal(value, err, attempts, body.closes)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if value == nil || !errors.As(err, &proof) || proof.StatusCode != status || proof.Header.Get("X-Request-Id") != "physical-proof" || string(proof.Body) != `{"credential":` || string(value.Envelope) != `{"credential":` {
						t.Fatal(value, err, proof)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if value != nil || !errors.As(err, &native) || native.Actual != status || native.ResponseHeader.Get("X-Request-Id") != "physical-proof" || string(native.Body) != `{"credential":` {
						t.Fatal(value, err, native)
					}
				}
				if (kind == "read" || kind == "read and close") && !errors.Is(err, readCause) {
					t.Fatal(err)
				}
				if (kind == "close" || kind == "read and close") && !errors.Is(err, closeCause) {
					t.Fatal(err)
				}
				if strings.HasPrefix(kind, "cancel") && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) {
					t.Fatal(err)
				}
				if kind == "source restored by close" && (!errors.Is(err, resource.ErrInvalidOption) || api.RawClient().Endpoint != endpoint) {
					t.Fatal(err)
				}
				nativeCredOperation(t, err, "UpdateRecord")
			})
		}
	}
}

func TestPythonCredentialUpdateNativeRetryPolicyAndOwnership(t *testing.T) {
	t.Run("clean retry uses fixed body and route", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) {
			if len(calls) == 1 {
				return 503, "retry"
			}
			return 200, `{"credential":{"name":"response"}}`
		})
		retries := 0
		api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		value, err := api.UpdateRecord(context.Background(), "a", credentials.WithRecordUpdateField("blob", "new"))
		if err != nil || len(calls) != 2 || retries != 1 || calls[0] != calls[1] {
			t.Fatal(value, err, calls, retries)
		}
		credentialRecordField(t, value, "blob", `"new"`)
		credentialRecordField(t, value, "name", `"response"`)
	})
	for _, kind := range []string{"callback error", "body changed", "source changed", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			var calls []nativeCredCall
			api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 503, "retry proof" })
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("retry cause")
			api.RawClient().RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				switch kind {
				case "callback error":
					return cause
				case "body changed":
					opts.JSONBody = json.RawMessage(`{"credential":{"blob":"changed"}}`)
				case "source changed":
					api.RawClient().Endpoint += "changed/"
				case "cancel":
					cancel(cause)
				}
				return nil
			}
			value, err := api.UpdateRecord(ctx, "a", credentials.WithRecordUpdateField("blob", "new"))
			var native gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "retry proof" || len(calls) != 1 {
				t.Fatal(value, err, native, calls)
			}
			if kind == "callback error" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if (kind == "body changed" || kind == "source changed") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if kind == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
		})
	}
	t.Run("transport cause", func(t *testing.T) {
		var serverCalls []nativeCredCall
		api, cloud := nativeCredAPI(t, &serverCalls, func(*http.Request) (int, string) { return 200, `{}` })
		cause := errors.New("physical transport")
		attempts := 0
		cloud.Provider.HTTPClient.Transport = nativeCredTransport(func(*http.Request) (*http.Response, error) { attempts++; return nil, cause })
		_, err := api.UpdateRecord(context.Background(), "a", credentials.WithRecordUpdateField("blob", "new"))
		if !errors.Is(err, cause) || attempts != 1 {
			t.Fatal(err, attempts)
		}
	})
}

func TestPythonCredentialUpdateEscapesMemberExactlyOnce(t *testing.T) {
	var calls []nativeCredCall
	api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.EscapedPath() != "/keystone/v3/credentials/folder%2Fname%25" {
			t.Error(req.URL.EscapedPath())
		}
		return 200, `{}`
	})
	_, err := api.UpdateRecord(context.Background(), "folder/name%", credentials.WithRecordUpdateField("blob", "new"))
	if err != nil || len(calls) != 1 {
		t.Fatal(err, calls)
	}
}
