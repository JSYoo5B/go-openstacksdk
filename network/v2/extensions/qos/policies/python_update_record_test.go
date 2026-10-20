package policies_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/gophercloud/gophercloud/v2"
)

func TestQoSPolicyUpdateRecordRawWireAliasesAndDescriptorView(t *testing.T) {
	var calls []nativePolicyCall
	api, _ := nativePolicyAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Method != "PUT" || r.URL.EscapedPath() != "/neutron/v2.0/qos/policies/policy%2Fid" || r.URL.RawQuery != "" {
			t.Fatal(r.Method, r.URL)
		}
		return nativePolicyWire(203, `{"policy":{"id":7,"description":"actual","x_reply":9007199254740993}}`)
	})
	row, err := api.UpdateRecord(context.Background(), "policy/id", policies.WithRecordUpdateAttribute("is_shared", true), policies.WithRecordUpdateAttribute("shared", false), policies.WithRecordUpdateAttribute("is_default", "false"), policies.WithRecordUpdateAttribute("rules", map[string]any{"type": "unknown"}), policies.WithRecordUpdateAttribute("tags", "red"), policies.WithRecordUpdateAttribute("tenant_id", "owner"), policies.WithRecordUpdateAttribute("x_unknown", json.RawMessage(`9007199254740993`)))
	if err != nil || row == nil || len(calls) != 1 {
		t.Fatal(row, err, calls)
	}
	var payload map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(calls[0].body), &payload); err != nil {
		t.Fatal(err)
	}
	body := payload["policy"]
	if len(body) != 6 || string(body["shared"]) != "false" || string(body["is_default"]) != `"false"` || string(body["rules"]) != `{"type":"unknown"}` || string(body["tags"]) != `"red"` || body["is_shared"] != nil || body["id"] != nil {
		t.Fatal(payload)
	}
	for k, v := range map[string]string{"id": "7", "is_shared": "false", "is_default": "true", "rules": `[{"type":"unknown"}]`, "tags": `["red"]`, "project_id": `"owner"`, "x_unknown": "9007199254740993", "x_reply": "9007199254740993"} {
		pythonFindField(t, row, k, v)
	}
	if row.StatusCode != 203 || string(row.Wire.Body["x_reply"]) != "9007199254740993" {
		t.Fatal(row)
	}
}

func TestQoSPolicyUpdateRecordEmptyAndIgnoredRevisionDoNotSend(t *testing.T) {
	for _, revision := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "ignored if_revision"}[revision], func(t *testing.T) {
			var calls []nativePolicyCall
			api, _ := nativePolicyAPI(t, &calls, func(r *http.Request) *http.Response { return nativePolicyWire(200, `{}`) })
			opts := []policies.RecordUpdateOption{}
			if revision {
				opts = append(opts, policies.WithRecordUpdateAttribute("if_revision", nil))
			}
			row, err := api.UpdateRecord(context.Background(), "id", opts...)
			if err != nil || row == nil || len(calls) != 0 || row.StatusCode != 0 || row.Wire != nil || len(row.Envelope) != 0 {
				t.Fatal(row, err, calls)
			}
			pythonFindField(t, row, "id", `"id"`)
			pythonFindField(t, row, "tags", `[]`)
		})
	}
}

func TestQoSPolicyUpdateRecordUnknownAndNullRemainDirtyWithoutConditionalHeader(t *testing.T) {
	var calls []nativePolicyCall
	api, _ := nativePolicyAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("If-Match") != "" {
			t.Fatal(r.Header)
		}
		return nativePolicyWire(200, `{"policy":{"shared":null}}`)
	})
	row, err := api.UpdateRecord(context.Background(), "id", policies.WithRecordUpdateAttribute("shared", nil), policies.WithRecordUpdateAttribute("name", nil), policies.WithRecordUpdateAttribute("rules", nil), policies.WithRecordUpdateAttribute("x_unknown", nil), policies.WithRecordUpdateAttribute("if_match", "body-extension"), policies.WithRecordUpdateAttribute("if_revision", 42))
	if err != nil || row == nil || len(calls) != 1 {
		t.Fatal(row, err, calls)
	}
	var payload map[string]map[string]json.RawMessage
	_ = json.Unmarshal([]byte(calls[0].body), &payload)
	body := payload["policy"]
	if len(body) != 5 || string(body["shared"]) != "null" || string(body["name"]) != "null" || string(body["rules"]) != "null" || string(body["x_unknown"]) != "null" || string(body["if_match"]) != `"body-extension"` {
		t.Fatal(payload)
	}
	pythonFindField(t, row, "is_shared", "null")
	pythonFindField(t, row, "rules", "null")
	pythonFindField(t, row, "x_unknown", "null")
}

func TestQoSPolicyUpdateRecordResponseWireGetterPrecedenceAndEvidence(t *testing.T) {
	var calls []nativePolicyCall
	body := `{"policy":{"is_shared":false,"project_id":null,"tenant_id":"response","x_response":1}}`
	api, _ := nativePolicyAPI(t, &calls, func(r *http.Request) *http.Response {
		response := nativePolicyWire(199, body)
		response.Header.Set("X-Request-Id", "actual-update")
		return response
	})
	row, err := api.UpdateRecord(context.Background(), "id", policies.WithRecordUpdateAttribute("shared", true), policies.WithRecordUpdateAttribute("tenant_id", "seed"))
	if err != nil || row == nil || len(calls) != 1 || row.StatusCode != 199 || string(row.Envelope) != body || row.Header.Get("X-Request-Id") != "actual-update" {
		t.Fatal(row, err, calls)
	}
	pythonFindField(t, row, "is_shared", "true")
	pythonFindField(t, row, "project_id", "null")
	if string(row.Wire.Body["is_shared"]) != "false" {
		t.Fatal(row)
	}
	clone := row.Clone()
	clone.Resource.Body["is_shared"][0] = 'X'
	clone.Header.Set("X-Request-Id", "caller")
	clone.Envelope[0] = '['
	if string(row.Resource.Body["is_shared"]) != "true" || row.Header.Get("X-Request-Id") != "actual-update" || string(row.Envelope) != body {
		t.Fatal(row)
	}
}

func TestQoSPolicyUpdateRecordOptionsOwnRawBytesAndTransport(t *testing.T) {
	attrs := map[string]json.RawMessage{"name": json.RawMessage(`"original"`)}
	headers := map[string]string{"X-Owned": "original"}
	version := "2.1"
	option := policies.WithRecordUpdateOpts(policies.RecordUpdateOpts{Attributes: attrs, Headers: headers, Microversion: &version})
	copy(attrs["name"], []byte(`"mutated!"`))
	headers["X-Owned"] = "changed"
	version = "2.2"
	var calls []nativePolicyCall
	api, _ := nativePolicyAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Owned") != "original" || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
			t.Fatal(r.Header)
		}
		return nativePolicyWire(200, `{"policy":{}}`)
	})
	for i := 0; i < 2; i++ {
		row, err := api.UpdateRecord(context.Background(), "id", option)
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
		pythonFindField(t, row, "name", `"original"`)
	}
	if len(calls) != 2 || calls[0].body != `{"policy":{"name":"original"}}` || calls[1].body != calls[0].body {
		t.Fatal(calls)
	}
}

func TestQoSPolicyUpdateRecordConstructorCollisionsAndSourceGuardsPreflight(t *testing.T) {
	for _, key := range []string{"id", "qos_policy", "value", "resource_type", "connection", "_synchronized", "self", "cls", "base_path"} {
		t.Run(key, func(t *testing.T) {
			var calls []nativePolicyCall
			api, _ := nativePolicyAPI(t, &calls, func(r *http.Request) *http.Response { return nativePolicyWire(200, `{}`) })
			row, err := api.UpdateRecord(context.Background(), "id", policies.WithRecordUpdateAttribute(key, nil))
			if row != nil || err == nil || len(calls) != 0 {
				t.Fatal(row, err, calls)
			}
		})
	}
	for _, mode := range []string{"nil option", "source change", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			var calls []nativePolicyCall
			later := 0
			cause := errors.New("QoS update cancelled")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			api, cloud := nativePolicyAPI(t, &calls, func(r *http.Request) *http.Response { return nativePolicyWire(200, `{}`) })
			var option policies.RecordUpdateOption
			if mode != "nil option" {
				option = func(*policies.RecordUpdateOpts) error {
					if mode == "cancel" {
						cancel(cause)
					} else {
						*api = *policies.New(cloud.Client("network", "/replacement/"))
					}
					return nil
				}
			}
			_, err := api.UpdateRecord(ctx, "id", option, func(*policies.RecordUpdateOpts) error { later++; return nil })
			if err == nil || len(calls) != 0 || later != 0 {
				t.Fatal(err, calls, later)
			}
		})
	}
}

func TestQoSPolicyUpdateRecordActualRejectionsAndPhysicalFaultsNeverReplay(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel", "clean404", "clean412"} {
		t.Run(mode, func(t *testing.T) {
			var calls []nativePolicyCall
			closed := 0
			cause := errors.New("QoS mutation " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			api, _ := nativePolicyAPI(t, &calls, func(r *http.Request) *http.Response {
				code := 404
				if mode == "clean412" {
					code = 412
				}
				response := nativePolicyWire(code, `{"error":"actual mutation"}`)
				if !strings.HasPrefix(mode, "clean") {
					body := &pythonFindFaultBody{data: `{"error":"physical"}`, onClose: func() {
						closed++
						if mode == "cancel" {
							cancel(cause)
						}
					}}
					if mode == "read" {
						body.read = cause
					}
					if mode == "close" {
						body.close = cause
					}
					response.Body = body
				}
				return response
			})
			_, err := api.UpdateRecord(ctx, "id", policies.WithRecordUpdateAttribute("name", "changed"))
			if err == nil || len(calls) != 1 {
				t.Fatal(err, calls)
			}
			if strings.HasPrefix(mode, "clean") {
				var native gophercloud.ErrUnexpectedResponseCode
				want := 404
				if mode == "clean412" {
					want = 412
				}
				if !errors.As(err, &native) || native.Actual != want || string(native.Body) != `{"error":"actual mutation"}` {
					t.Fatal(err, native)
				}
			} else if !errors.Is(err, cause) || closed != 1 {
				t.Fatal(err, closed)
			}
		})
	}
}

func TestQoSPolicyUpdateRecordBindsExistingBorrowedNoopLane(t *testing.T) {
	var calls []nativePolicyCall
	api, _ := nativePolicyAPI(t, &calls, func(r *http.Request) *http.Response {
		return nativePolicyWire(200, `{"policy":{"id":"id","name":"gold","shared":true}}`)
	})
	row, err := api.UpdateRecord(context.Background(), "id", policies.WithRecordUpdateAttribute("name", "gold"), policies.WithRecordUpdateAttribute("shared", 1))
	if err != nil || row == nil {
		t.Fatal(row, err)
	}
	row.Resource.Body["name"] = json.RawMessage(`"caller"`)
	next, dirty, err := api.UpdateCloudRecord(context.Background(), row, map[string]json.RawMessage{"name": json.RawMessage(`"gold"`), "shared": json.RawMessage(`1.0`)})
	if err != nil || dirty || next == nil || len(calls) != 1 {
		t.Fatal(next, dirty, err, calls)
	}
	pythonFindField(t, next, "name", `"gold"`)
}
