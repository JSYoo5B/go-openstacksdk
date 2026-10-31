package users_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonUserFindRecordFetchControlsCollectionAndEnvelope(t *testing.T) {
	var calls []pythonUserCall
	body := `{"selected":{"id":"actual","enabled":0,"federated":"scalar"},"user":{"id":"wrong"},"x_proof":true}`
	api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.Path != "/keystone/v3/alternate-users" || r.URL.RawQuery != "domain_id=d" {
			t.Fatal(r.URL)
		}
		if r.Header.Get("X-Typed") != "yes" || r.Header.Get("Cache-Control") != "" {
			t.Fatal(r.Header)
		}
		reply := recordUserWire(203, body)
		reply.Header.Set("X-Request-Id", "controlled-get")
		return reply
	})
	row, err := api.FindRecord(context.Background(), "alice",
		sdk.WithRecordFindQuery("requires_id", false), sdk.WithRecordFindQuery("base_path", "/alternate-users"),
		sdk.WithRecordFindQuery("skip_cache", true), sdk.WithRecordFindQuery("error_message", "custom missing"),
		sdk.WithRecordFindQuery("resource_response_key", "selected"), sdk.WithRecordFindQuery("domain_id", "d"),
		sdk.WithRecordFindHeaders(map[string]string{"X-Typed": "yes"}))
	if err != nil || row == nil || len(calls) != 1 || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "controlled-get" || string(row.Envelope) != body {
		t.Fatal(row, err, calls)
	}
	recordField(t, row, "id", `"actual"`)
	recordField(t, row, "is_enabled", `false`)
	recordField(t, row, "federated", `["scalar"]`)
	if row.Resource.Body["skip_cache"] != nil || row.Resource.Body["resource_response_key"] != nil {
		t.Fatal(row.Resource)
	}
}

func TestPythonUserFindRecordFetchControlsHeadersAreStageSpecific(t *testing.T) {
	var calls []pythonUserCall
	api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Typed") != "kept" {
			t.Fatal(r.Header)
		}
		if len(calls) == 1 {
			if r.URL.Path != "/keystone/v3/alternate-users/alice" || r.URL.Query().Get("headers") != "X-Raw" || r.Header.Get("X-Raw") != "typed" {
				t.Fatal(r.URL, r.Header)
			}
			return recordUserWire(404, `{}`)
		}
		if r.URL.Path != "/keystone/v3/list-users" || r.URL.Query().Get("name") != "alice" || r.URL.Query().Has("headers") || r.Header.Get("X-Raw") != "raw" {
			t.Fatal(r.URL, r.Header)
		}
		for _, key := range []string{"requires_id", "base_path", "skip_cache", "error_message", "resource_response_key"} {
			if r.URL.Query().Has(key) {
				t.Fatal(r.URL)
			}
		}
		return recordUserWire(200, `{"users":[{"id":"u","name":"alice"}]}`)
	})
	row, err := api.FindRecord(context.Background(), "alice", sdk.WithRecordFindQuery("base_path", "/alternate-users"), sdk.WithRecordFindListBasePath("/list-users"), sdk.WithRecordFindQuery("headers", map[string]string{"X-Raw": "raw"}), sdk.WithRecordFindHeaders(map[string]string{"X-Raw": "typed", "X-Typed": "kept"}), sdk.WithRecordFindQuery("requires_id", true), sdk.WithRecordFindQuery("skip_cache", true), sdk.WithRecordFindQuery("error_message", "custom"), sdk.WithRecordFindQuery("resource_response_key", "selected"))
	if err != nil || row == nil || len(calls) != 2 {
		t.Fatal(row, err, calls)
	}
	recordField(t, row, "id", `"u"`)
}

func TestPythonUserFindRecordFetchControlsBasePathAndFalseyEnvelope(t *testing.T) {
	for _, key := range []any{nil, false, "", 0} {
		t.Run(string(mustUserFetchJSON(t, key)), func(t *testing.T) {
			var calls []pythonUserCall
			api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.URL.Path != "/keystone/v3/alternate-users" {
					t.Fatal(r.URL)
				}
				if len(calls) == 1 {
					return recordUserWire(403, `{}`)
				}
				return recordUserWire(200, `{"users":[{"id":"u","name":"alice"}]}`)
			})
			row, err := api.FindRecord(context.Background(), "alice", sdk.WithRecordFindQuery("base_path", "/alternate-users"), sdk.WithRecordFindQuery("requires_id", false), sdk.WithRecordFindQuery("resource_response_key", key))
			if err != nil || row == nil || len(calls) != 2 {
				t.Fatal(row, err, calls)
			}
		})
	}
}
func mustUserFetchJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestPythonUserFindRecordFetchControlsHeadersValidateAtListStage(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "successful-get", true: "fallback"}[fallback], func(t *testing.T) {
			var calls []pythonUserCall
			api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
				if len(calls) != 1 || r.URL.Query().Get("headers") != "not-a-mapping" {
					t.Fatal(r.URL, calls)
				}
				if fallback {
					return recordUserWire(404, `{}`)
				}
				return recordUserWire(200, `{"user":{"id":"u"}}`)
			})
			row, err := api.FindRecord(context.Background(), "alice", sdk.WithRecordFindQuery("headers", "not-a-mapping"))
			if fallback {
				if row != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(row, err)
				}
			} else if err != nil || row == nil {
				t.Fatal(row, err)
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		})
	}
}

func TestPythonUserFindRecordFetchControlsOwnRawOptions(t *testing.T) {
	var calls []pythonUserCall
	query := map[string]json.RawMessage{"base_path": json.RawMessage(`"/owned-users"`), "resource_response_key": json.RawMessage(`"selected"`), "headers": json.RawMessage(`{"X-Owned":"original"}`)}
	option := sdk.WithRecordFindOpts(sdk.RecordFindOpts{Query: query})
	copy(query["base_path"], []byte(`"/other-users"`))
	query["resource_response_key"] = json.RawMessage(`"other"`)
	query["headers"] = json.RawMessage(`{"X-Owned":"mutated"}`)
	api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
		if !strings.HasPrefix(r.URL.Path, "/keystone/v3/owned-users") {
			t.Fatal(r.URL)
		}
		if len(calls) == 1 {
			return recordUserWire(404, `{}`)
		}
		if r.Header.Get("X-Owned") != "original" {
			t.Fatal(r.Header)
		}
		return recordUserWire(200, `{"users":[{"id":"u","name":"alice"}]}`)
	})
	row, err := api.FindRecord(context.Background(), "alice", option)
	if err != nil || row == nil || len(calls) != 2 {
		t.Fatal(row, err, calls)
	}
}

func TestPythonUserFindRecordFetchControlsPhysicalFailuresNeverFallback(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			var calls []pythonUserCall
			cause := errors.New("controlled physical " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			closed := 0
			api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
				reply := recordUserWire(404, `{"error":{"message":"not clean"}}`)
				body := &testcloud.Body{Data: `{"error":{"message":"not clean"}}`, OnClose: func() {
					closed++
					if mode == "cancel" {
						cancel(cause)
					}
				}}
				if mode == "read" {
					body.ReadErr = cause
				}
				if mode == "close" {
					body.CloseErr = cause
				}
				reply.Body = body
				return reply
			})
			row, err := api.FindRecord(ctx, "alice", sdk.WithRecordFindQuery("requires_id", false), sdk.WithRecordFindQuery("skip_cache", true), sdk.WithRecordFindQuery("error_message", "controlled failure"))
			if row != nil || !errors.Is(err, cause) || len(calls) != 1 || closed != 1 {
				t.Fatal(row, err, calls, closed)
			}
		})
	}
}
