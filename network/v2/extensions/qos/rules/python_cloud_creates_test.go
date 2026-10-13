package rules_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type qosCloudCreateTestReceipt struct {
	view, wire *resource.RawResource
	body       json.RawMessage
	header     http.Header
	status     int
}
type qosCloudCreateTestKind struct {
	kind, key, required, operation string
	create                         func(context.Context, *rules.API, string, map[string]json.RawMessage) (*qosCloudCreateTestReceipt, error)
}

func qosCloudCreateTestKinds() []qosCloudCreateTestKind {
	return []qosCloudCreateTestKind{
		{"bandwidth_limit_rules", "bandwidth_limit_rule", "max_kbps", "CreateCloudBandwidthLimitRecord", func(c context.Context, a *rules.API, p string, v map[string]json.RawMessage) (*qosCloudCreateTestReceipt, error) {
			r, e := a.CreateCloudBandwidthLimitRecord(c, p, v)
			if r == nil {
				return nil, e
			}
			return &qosCloudCreateTestReceipt{r.Resource, r.Wire, r.Envelope, r.Header, r.StatusCode}, e
		}},
		{"dscp_marking_rules", "dscp_marking_rule", "dscp_mark", "CreateCloudDSCPMarkingRecord", func(c context.Context, a *rules.API, p string, v map[string]json.RawMessage) (*qosCloudCreateTestReceipt, error) {
			r, e := a.CreateCloudDSCPMarkingRecord(c, p, v)
			if r == nil {
				return nil, e
			}
			return &qosCloudCreateTestReceipt{r.Resource, r.Wire, r.Envelope, r.Header, r.StatusCode}, e
		}},
		{"minimum_bandwidth_rules", "minimum_bandwidth_rule", "min_kbps", "CreateCloudMinimumBandwidthRecord", func(c context.Context, a *rules.API, p string, v map[string]json.RawMessage) (*qosCloudCreateTestReceipt, error) {
			r, e := a.CreateCloudMinimumBandwidthRecord(c, p, v)
			if r == nil {
				return nil, e
			}
			return &qosCloudCreateTestReceipt{r.Resource, r.Wire, r.Envelope, r.Header, r.StatusCode}, e
		}},
	}
}
func TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts(t *testing.T) {
	for _, kind := range qosCloudCreateTestKinds() {
		for _, status := range []int{199, 201, 302, 399} {
			t.Run(fmt.Sprint(kind.kind, status), func(t *testing.T) {
				var calls []nativeRuleCall
				body := `{"` + kind.key + `":{"id":null,"qos_policy_id":"foreign","microversion":"foreign","location":{"forged":true},"x_new":9007199254740993}}`
				api, _ := nativeRuleAPI(t, &calls, func(r *http.Request) *http.Response {
					if r.Method != http.MethodPost || r.URL.RawQuery != "" || r.URL.EscapedPath() != "/neutron/v2.0/qos/policies/p%2F1/"+kind.kind || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
						t.Error(r.Method, r.URL, r.Header)
					}
					v := nativeRuleWire(status, body)
					v.Header.Set("X-Request-Id", "create-leaf-actual")
					return v
				})
				api.RawClient().Microversion = "2.1"
				attrs := map[string]json.RawMessage{kind.required: json.RawMessage("0")}
				if kind.kind != "dscp_marking_rules" {
					attrs["direction"] = json.RawMessage("false")
				}
				if kind.kind == "bandwidth_limit_rules" {
					attrs["max_burst_kbps"] = nil
				}
				result, err := kind.create(context.Background(), api, "p/1", attrs)
				if err != nil || result == nil || result.view == nil || result.wire == nil || result.status != status || string(result.body) != body || result.header.Get("X-Request-Id") != "create-leaf-actual" || len(calls) != 1 || calls[0].path != "/neutron/v2.0/qos/policies/p/1/"+kind.kind {
					t.Fatal(result, err, calls)
				}
				for key, want := range map[string]string{kind.required: "0", "id": "null", "name": "null", "qos_policy_id": `"p/1"`, "location": "null", "microversion": `"2.1"`, "x_new": "9007199254740993"} {
					if string(result.view.Body[key]) != want {
						t.Fatal(key, string(result.view.Body[key]), want)
					}
				}
				if result.wire.Body[kind.required] != nil || string(result.wire.Body["qos_policy_id"]) != `"foreign"` || string(result.wire.Body["microversion"]) != `"foreign"` {
					t.Fatal(result.wire)
				}
				if kind.kind != "dscp_marking_rules" && string(result.view.Body["direction"]) != "false" {
					t.Fatal(result.view)
				}
				var posted map[string]map[string]json.RawMessage
				if err := json.Unmarshal([]byte(calls[0].body), &posted); err != nil || string(posted[kind.key][kind.required]) != "0" || posted[kind.key]["qos_policy_id"] != nil || posted[kind.key]["name"] != nil {
					t.Fatal(calls, err)
				}
				if kind.kind == "bandwidth_limit_rules" && string(posted[kind.key]["max_burst_kbps"]) != "null" {
					t.Fatal(posted)
				}
			})
		}
	}
}
func TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections(t *testing.T) {
	for _, kind := range qosCloudCreateTestKinds() {
		var calls []nativeRuleCall
		api, _ := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(201, `{"`+kind.key+`":{"id":"new"}}`) })
		for _, attrs := range []map[string]json.RawMessage{nil, {kind.required: json.RawMessage("0"), "x_new": nil}, {kind.required: json.RawMessage(`{bad`)}, {kind.required: json.RawMessage("0"), "qos_policy_id": json.RawMessage(`"other"`)}} {
			result, err := kind.create(context.Background(), api, "parent", attrs)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
				t.Fatal(result, err, calls)
			}
		}
		result, err := kind.create(context.Background(), api, "parent", map[string]json.RawMessage{kind.required: nil})
		if err != nil || result == nil || string(result.view.Body[kind.required]) != "null" || len(calls) != 1 {
			t.Fatal(result, err, calls)
		}
		for _, status := range []int{400, 403, 404, 409, 503} {
			calls = nil
			body := `{"error":"create rejected"}`
			api, _ := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response {
				v := nativeRuleWire(status, body)
				v.Header.Set("X-Request-Id", "rejected-create")
				return v
			})
			result, err := kind.create(context.Background(), api, "parent", map[string]json.RawMessage{kind.required: json.RawMessage("0")})
			var native gophercloud.ErrUnexpectedResponseCode
			if result == nil || result.view != nil || result.wire != nil || result.status != status || string(result.body) != body || result.header.Get("X-Request-Id") != "rejected-create" || !errors.As(err, &native) || native.Actual != status || native.Method != http.MethodPost || len(calls) != 1 {
				t.Fatal(result, err, native, calls)
			}
			nativeRuleOperation(t, err, kind.operation)
		}
	}
}
