package rules_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type qosUpdateTestKind struct {
	kind, key, field string
	get              func(context.Context, *rules.API, json.RawMessage, string) (*rules.QoSRuleUpdateRecord, error)
	update           func(context.Context, *rules.API, *rules.QoSRuleUpdateRecord, map[string]json.RawMessage) (*rules.QoSRuleUpdateRecord, bool, error)
}

func qosUpdateTestKinds() []qosUpdateTestKind {
	return []qosUpdateTestKind{
		{"bandwidth_limit_rules", "bandwidth_limit_rule", "max_kbps", func(c context.Context, a *rules.API, p json.RawMessage, id string) (*rules.QoSRuleUpdateRecord, error) {
			return a.GetCloudBandwidthLimitUpdateRecord(c, p, id)
		}, func(c context.Context, a *rules.API, r *rules.QoSRuleUpdateRecord, v map[string]json.RawMessage) (*rules.QoSRuleUpdateRecord, bool, error) {
			return a.UpdateCloudBandwidthLimitRecord(c, r, v)
		}},
		{"dscp_marking_rules", "dscp_marking_rule", "dscp_mark", func(c context.Context, a *rules.API, p json.RawMessage, id string) (*rules.QoSRuleUpdateRecord, error) {
			return a.GetCloudDSCPMarkingUpdateRecord(c, p, id)
		}, func(c context.Context, a *rules.API, r *rules.QoSRuleUpdateRecord, v map[string]json.RawMessage) (*rules.QoSRuleUpdateRecord, bool, error) {
			return a.UpdateCloudDSCPMarkingRecord(c, r, v)
		}},
		{"minimum_bandwidth_rules", "minimum_bandwidth_rule", "min_kbps", func(c context.Context, a *rules.API, p json.RawMessage, id string) (*rules.QoSRuleUpdateRecord, error) {
			return a.GetCloudMinimumBandwidthUpdateRecord(c, p, id)
		}, func(c context.Context, a *rules.API, r *rules.QoSRuleUpdateRecord, v map[string]json.RawMessage) (*rules.QoSRuleUpdateRecord, bool, error) {
			return a.UpdateCloudMinimumBandwidthRecord(c, r, v)
		}},
	}
}
func TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI(t *testing.T) {
	for _, kind := range qosUpdateTestKinds() {
		var calls []nativeRuleCall
		api, _ := nativeRuleAPI(t, &calls, func(r *http.Request) *http.Response {
			if r.Method != http.MethodGet || r.URL.Path != "/neutron/v2.0/qos/policies/None/"+kind.kind+"/requested" {
				t.Fatal(r.Method, r.URL)
			}
			v := nativeRuleWire(203, `{"`+kind.key+`":{"id":null,"`+kind.field+`":true,"qos_policy_id":"foreign","microversion":"foreign"}}`)
			v.Header.Set("X-Request-Id", "borrowed")
			return v
		})
		current, err := kind.get(context.Background(), api, nil, "requested")
		if err != nil || current == nil {
			t.Fatal(current, err)
		}
		clone := current.Clone()
		current.Resource.Body[kind.field] = json.RawMessage("false")
		current.Wire.Body[kind.field] = json.RawMessage("false")
		current.QoSPolicyID = json.RawMessage(`"evil"`)
		current.Envelope[0] = 'X'
		current.Header.Set("X-Request-Id", "evil")
		clone.Resource.Body["id"] = json.RawMessage(`"evil"`)
		for _, seed := range []*rules.QoSRuleUpdateRecord{current, clone} {
			result, changed, err := kind.update(context.Background(), api, seed, map[string]json.RawMessage{kind.field: json.RawMessage("1")})
			if err != nil || changed || result == nil || string(result.Resource.Body[kind.field]) != "true" || string(result.Resource.Body["id"]) != "null" || string(result.Resource.Body["qos_policy_id"]) != "null" || string(result.QoSPolicyID) != "null" || result.Header.Get("X-Request-Id") != "borrowed" || string(result.Resource.Body["microversion"]) != "null" || len(calls) != 1 {
				t.Fatal(result, changed, err, calls)
			}
		}
		foreign := rules.New(api.RawClient())
		if r, changed, e := kind.update(context.Background(), foreign, clone, map[string]json.RawMessage{kind.field: json.RawMessage("2")}); r != nil || changed || !errors.Is(e, resource.ErrInvalidOption) || len(calls) != 1 {
			t.Fatal(r, changed, e, calls)
		}
		api.RawClient().ResourceBase += "changed/"
		if r, changed, e := kind.update(context.Background(), api, clone, map[string]json.RawMessage{kind.field: json.RawMessage("2")}); r != nil || changed || !errors.Is(e, resource.ErrInvalidOption) || len(calls) != 1 {
			t.Fatal(r, changed, e, calls)
		}
	}
}
func TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire(t *testing.T) {
	for _, kind := range qosUpdateTestKinds() {
		for _, tc := range []struct {
			before, after string
			put           bool
		}{{"missing", "null", true}, {`"false"`, "true", true}, {"false", "0", false}, {"9007199254740992.0", "9007199254740993.0", false}, {"1e-4000", "0.0", false}, {"1", "2", true}} {
			var calls []nativeRuleCall
			api, _ := nativeRuleAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.Header.Get("OpenStack-API-Version") != "network 2.1" || r.URL.EscapedPath() != "/neutron/v2.0/qos/policies/p%2F1/"+kind.kind+"/actual" {
					t.Error(r.Method, r.URL, r.Header)
				}
				if r.Method == http.MethodGet {
					field := ""
					if tc.before != "missing" {
						field = `,"` + kind.field + `":` + tc.before
					}
					return nativeRuleWire(200, `{"`+kind.key+`":{"id":"actual","microversion":"foreign","qos_policy_id":"foreign"`+field+`}}`)
				}
				return nativeRuleWire(200, `{"`+kind.key+`":{"id":"actual","microversion":"foreign","qos_policy_id":"foreign","x_new":9007199254740993}}`)
			})
			api.RawClient().Microversion = "2.1"
			current, err := kind.get(context.Background(), api, json.RawMessage(`"p/1"`), "actual")
			if err != nil {
				t.Fatal(err)
			}
			result, changed, err := kind.update(context.Background(), api, current, map[string]json.RawMessage{kind.field: json.RawMessage(tc.after)})
			wantCalls := 1
			wantVersion := "null"
			wantValue := tc.before
			if tc.put {
				wantCalls = 2
				wantVersion = `"2.1"`
				wantValue = tc.after
			}
			if err != nil || changed != tc.put || result == nil || len(calls) != wantCalls || string(result.Resource.Body[kind.field]) != wantValue || string(result.Resource.Body["microversion"]) != wantVersion || string(result.Resource.Body["qos_policy_id"]) != `"p/1"` {
				t.Fatal(tc, result, changed, err, calls)
			}
			if tc.put {
				var payload map[string]map[string]json.RawMessage
				json.Unmarshal([]byte(calls[1].body), &payload)
				if len(payload[kind.key]) != 1 || string(payload[kind.key][kind.field]) != tc.after || result.Wire.Body[kind.field] != nil {
					t.Fatal(payload, result)
				}
			}
		}
	}
}
func TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT(t *testing.T) {
	for _, kind := range qosUpdateTestKinds() {
		for _, rawID := range []string{"null"} {
			var calls []nativeRuleCall
			api, _ := nativeRuleAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.Method != http.MethodGet {
					t.Fatal("invalid dirty ID PUT", r.URL)
				}
				return nativeRuleWire(200, `{"`+kind.key+`":{"id":`+rawID+`,"`+kind.field+`":1}}`)
			})
			current, err := kind.get(context.Background(), api, json.RawMessage(`"parent"`), "requested")
			if err != nil {
				t.Fatal(err)
			}
			result, changed, err := kind.update(context.Background(), api, current, map[string]json.RawMessage{kind.field: json.RawMessage("2")})
			if result != nil || !changed || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
				t.Fatal(result, changed, err, calls)
			}
			if result, changed, err := kind.update(context.Background(), api, &rules.QoSRuleUpdateRecord{}, nil); result != nil || changed || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
				t.Fatal(result, changed, err, calls)
			}
			if result, changed, err := kind.update(context.Background(), api, current, map[string]json.RawMessage{"qos_policy_id": json.RawMessage(`"other"`)}); result != nil || changed || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
				t.Fatal(result, changed, err, calls)
			}
		}
	}
}

func TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView(t *testing.T) {
	for _, kind := range qosUpdateTestKinds() {
		for _, id := range []string{"0", "false", `""`, "0.0", "1e-4000", "[]", "{}"} {
			var calls []nativeRuleCall
			api, _ := nativeRuleAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.Method == http.MethodGet {
					return nativeRuleWire(200, `{"`+kind.key+`":{"id":`+id+`,"`+kind.field+`":1,"qos_policy_id":"foreign"}}`)
				}
				if r.Method != http.MethodPut || r.URL.EscapedPath() != "/neutron/v2.0/qos/policies/None/"+kind.kind+"/" {
					t.Error(r.Method, r.URL.EscapedPath())
				}
				return nativeRuleWire(200, `{"`+kind.key+`":{"id":`+id+`,"qos_policy_id":"foreign"}}`)
			})
			current, err := kind.get(context.Background(), api, nil, "requested")
			if err != nil {
				t.Fatal(err)
			}
			result, changed, err := kind.update(context.Background(), api, current, map[string]json.RawMessage{kind.field: json.RawMessage("2")})
			if err != nil || !changed || result == nil || len(calls) != 2 || string(result.Resource.Body["qos_policy_id"]) != "null" || string(result.Resource.Body[kind.field]) != "2" || string(result.Resource.Body["id"]) != id || result.Wire.Body[kind.field] != nil {
				t.Fatal(id, result, changed, err, calls)
			}
			var payload map[string]map[string]json.RawMessage
			json.Unmarshal([]byte(calls[1].body), &payload)
			if len(payload[kind.key]) != 1 || string(payload[kind.key][kind.field]) != "2" {
				t.Fatal(payload)
			}
		}
	}
}
