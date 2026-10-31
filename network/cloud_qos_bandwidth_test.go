package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func bandwidthCloudReplies(body string) map[string][]pythonCloudBReply {
	return map[string][]pythonCloudBReply{
		"GET /v2.0/extensions":                                {{200, `{"extensions":[{"alias":"qos"}]}`}},
		"GET /v2.0/qos/policies/parent":                       {{200, `{"policy":{"id":"parent","name":"policy"}}`}},
		"GET /v2.0/qos/policies/parent/bandwidth_limit_rules": {{200, body}},
	}
}

func TestCloudQoSBandwidthListOwnsGateParentPagingAndRawFields(t *testing.T) {
	replies := bandwidthCloudReplies(`{"bandwidth_limit_rules":[{"id":"a","max_kbps":1000,"direction":"egress","x_ext":{"keep":1},"location":{"fake":true}}],"bandwidth_limit_rules_links":[{"rel":"next","href":"?marker=next"}]}`)
	replies["GET /v2.0/qos/policies/parent/bandwidth_limit_rules"] = append(replies["GET /v2.0/qos/policies/parent/bandwidth_limit_rules"], pythonCloudBReply{200, `{"bandwidth_limit_rules":[{"id":"b","max_kbps":20,"direction":"ingress"}]}`})
	client, calls := pythonCloudBClient(t, replies)
	service := network.New(client)
	result, err := service.ListQoSBandwidthLimitRules(context.Background(), "parent", network.WithQoSBandwidthLimitFilters(json.RawMessage(`{"direction":"egress","max_kbps":1000,"fields":["id"]}`)))
	if err != nil || result == nil || len(result.Extensions) != 1 || result.Policy == nil || len(result.Rules) != 1 || len(result.Inventory) != 1 || result.Rules[0].QoSPolicyID != "parent" {
		t.Fatal(result, err, calls())
	}
	row := result.Rules[0]
	for key, want := range map[string]string{"id": `"a"`, "max_kbps": "1000", "max_burst_kbps": "null", "name": "null", "location": "null", "x_ext": `{"keep":1}`} {
		if string(row.Resource.Body[key]) != want {
			t.Fatal(key, row.Resource.Body)
		}
	}
	if string(row.Wire.Body["location"]) != `{"fake":true}` || row.StatusCode != 200 {
		t.Fatal(row)
	}
	pythonCloudBRequests(t, calls(), "GET /v2.0/extensions", "GET /v2.0/qos/policies/parent", "GET /v2.0/qos/policies/parent/bandwidth_limit_rules", "GET /v2.0/qos/policies/parent/bandwidth_limit_rules?marker=next")
}

func TestCloudQoSBandwidthSearchKeepsRuleIDPhaseAndFilterPresence(t *testing.T) {
	for _, id := range []string{"a*", "missing", ""} {
		client, calls := pythonCloudBClient(t, bandwidthCloudReplies(`{"bandwidth_limit_rules":[{"id":"a","max_kbps":true},{"id":"b","max_kbps":1},{"id":"c","max_kbps":2}]}`))
		callbacks := 0
		result, err := network.New(client).SearchQoSBandwidthLimitRules(context.Background(), "parent", id, network.WithQoSBandwidthLimitFilters(json.RawMessage(`{"max_kbps":1}`)), func(*network.QoSBandwidthLimitOpts) error { callbacks++; return nil })
		want := 2
		if id == "a*" {
			want = 1
		}
		if id == "missing" {
			want = 0
		}
		if err != nil || result == nil || len(result.Rules) != want || len(result.Inventory) != 2 || result.Value == nil || callbacks != 1 || len(calls()) != 3 {
			t.Fatal(id, result, err, callbacks, calls())
		}
	}
	client, calls := pythonCloudBClient(t, bandwidthCloudReplies(`{"bandwidth_limit_rules":[{"id":"a"}]}`))
	result, err := network.New(client).SearchQoSBandwidthLimitRules(context.Background(), "parent", "", network.WithQoSBandwidthLimitFilters(json.RawMessage(`{"unknown":{"x":1}}`)))
	var proof *resource.ResponseError
	if result == nil || len(result.Inventory) != 1 || result.Value != nil || result.Rules != nil || !errors.As(err, &proof) || len(calls()) != 3 {
		t.Fatal(result, err, proof, calls())
	}
}

func TestCloudQoSBandwidthGetResolvesPolicyOnceAndKeeps404(t *testing.T) {
	for _, status := range []int{200, 404} {
		replies := bandwidthCloudReplies(`{"bandwidth_limit_rules":[]}`)
		replies["GET /v2.0/qos/policies/parent/bandwidth_limit_rules/rule"] = []pythonCloudBReply{{status, `{"bandwidth_limit_rule":{"id":"rule","max_kbps":"raw","x":null}}`}}
		client, calls := pythonCloudBClient(t, replies)
		result, err := network.New(client).GetQoSBandwidthLimitRule(context.Background(), "parent", "rule")
		if result == nil || result.Policy == nil || len(result.Extensions) != 1 || len(calls()) != 3 {
			t.Fatal(result, err, calls())
		}
		if status == 200 {
			if err != nil || result.Rule == nil || string(result.Rule.Resource.Body["max_kbps"]) != `"raw"` || string(result.Rule.Resource.Body["location"]) != "null" {
				t.Fatal(result, err)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 404 || result.Value != nil {
				t.Fatal(result, err, native)
			}
		}
		pythonCloudBRequests(t, calls(), "GET /v2.0/extensions", "GET /v2.0/qos/policies/parent", "GET /v2.0/qos/policies/parent/bandwidth_limit_rules/rule")
	}
}

func TestCloudQoSBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy(t *testing.T) {
	for _, body := range []string{`{"extensions":[{"alias":"other"}]}`, `{"extensions":[{"alias":"qos"},{"alias":[]}]}`} {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/extensions": {{200, body}}})
		callbacks := 0
		result, err := network.New(client).ListQoSBandwidthLimitRules(context.Background(), "parent", func(*network.QoSBandwidthLimitOpts) error { callbacks++; return nil })
		if result == nil || len(result.Extensions) == 0 || result.Policy != nil || result.Value != nil || err == nil || callbacks != 0 || len(calls()) != 1 {
			t.Fatal(result, err, callbacks, calls())
		}
		if len(result.Extensions) == 1 && !errors.Is(err, network.ErrUnavailableExtension) {
			t.Fatal(err)
		}
	}
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/extensions": {{200, `{"extensions":[{"alias":"qos"}],"extensions_links":[{"rel":"next","href":"?marker=later"}]}`}, {503, `{"error":"later extension failure"}`}}})
	result, err := network.New(client).ListQoSBandwidthLimitRules(context.Background(), "parent")
	if result == nil || len(result.Extensions) != 1 || result.Policy != nil || err == nil || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}

func TestCloudQoSBandwidthPolicyMissingAndAmbiguityRetainActualGate(t *testing.T) {
	for _, body := range []string{`{"policies":[]}`, `{"policies":[{"id":"a","name":"wanted"},{"id":"b","name":"wanted"}]}`} {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
			"GET /v2.0/extensions":          {{200, `{"extensions":[{"alias":"qos"}]}`}},
			"GET /v2.0/qos/policies/wanted": {{404, `{"error":"missing"}`}},
			"GET /v2.0/qos/policies":        {{200, body}},
		})
		result, err := network.New(client).ListQoSBandwidthLimitRules(context.Background(), "wanted", network.WithQoSBandwidthLimitFilters(json.RawMessage("true")))
		if result == nil || len(result.Extensions) != 1 || result.Value != nil || len(calls()) != 3 || err == nil {
			t.Fatal(result, err, calls())
		}
		if body == `{"policies":[]}` && !errors.Is(err, resource.ErrNotFound) || body != `{"policies":[]}` && !errors.Is(err, resource.ErrAmbiguous) {
			t.Fatal(err)
		}
	}
}

func TestCloudQoSBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary(t *testing.T) {
	for _, test := range []struct{ expression, want string }{{"`[1,null,{\"id\":\"literal\"}]`", `[1,null,{"id":"literal"}]`}, {"'한😀'", `["한","😀"]`}, {"`{\"z\":1,\"a\":2}`", `["a","z"]`}, {"to_array(@)", ""}, {"[*].id", "error"}, {"length(@)", "error"}} {
		replies := bandwidthCloudReplies(`{"bandwidth_limit_rules":[{"id":"actual"}]}`)
		client, calls := pythonCloudBClient(t, replies)
		filter, _ := json.Marshal(map[string]any{"jmespath_filters": test.expression})
		result, err := network.New(client).ListQoSBandwidthLimitRules(context.Background(), "parent", network.WithQoSBandwidthLimitFilters(filter))
		if result == nil || len(calls()) != 2 || len(result.Inventory) != 0 || result.Rules != nil {
			t.Fatal(test, result, err, calls())
		}
		if test.want == "error" {
			if err == nil || result.Value != nil {
				t.Fatal(result, err)
			}
			continue
		}
		if err != nil || string(result.Value) != test.want {
			t.Fatal(test, result, err)
		}
		if test.expression == "to_array(@)" {
			g, ok := result.Values[0].(*network.QoSBandwidthLimitGenerator)
			if !ok {
				t.Fatal(result)
			}
			n := 0
			for row, err := range g.Records {
				if err != nil || row == nil {
					t.Fatal(row, err)
				}
				n++
			}
			if n != 1 || len(calls()) != 3 {
				t.Fatal(n, calls())
			}
		}
	}
	replies := bandwidthCloudReplies(`{"bandwidth_limit_rules":[]}`)
	replies["GET /v2.0/alternate"] = []pythonCloudBReply{{200, `{"bandwidth_limit_rules":[{"id":"actual"}]}`}}
	client, calls := pythonCloudBClient(t, replies)
	result, err := network.New(client).ListQoSBandwidthLimitRules(context.Background(), "parent", network.WithQoSBandwidthLimitFilters(json.RawMessage(`{"base_path":"alternate","headers":{"X-Trace":"raw"}}`)), network.WithQoSBandwidthLimitHeaders(map[string]string{"X-Trace": "typed"}))
	if err != nil || len(result.Rules) != 1 || len(calls()) != 3 || calls()[2].path != "/v2.0/alternate" {
		t.Fatal(result, err, calls())
	}
	_, err = network.New(client).API.QoSRules.AllBandwidthLimitRecords(context.Background(), "parent", rules.WithBandwidthLimitListQuery("base_path", "alternate"))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 3 {
		t.Fatal(err, calls())
	}
}

func TestCloudQoSBandwidthFilterOwnershipCallbackGuardAndFalseyCases(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage("false"), json.RawMessage("1e-400"), json.RawMessage("[]"), json.RawMessage(`""`), json.RawMessage("true"), json.RawMessage(`"expression"`)} {
		client, calls := pythonCloudBClient(t, bandwidthCloudReplies(`{"bandwidth_limit_rules":[]}`))
		result, err := network.New(client).ListQoSBandwidthLimitRules(context.Background(), "parent", network.WithQoSBandwidthLimitFilters(raw))
		valid := string(raw) != "true" && string(raw) != `"expression"`
		if result == nil || valid && (err != nil || string(result.Value) != "[]" || len(calls()) != 3) || !valid && (err == nil || len(calls()) != 2) {
			t.Fatal(string(raw), result, err, calls())
		}
	}
	for _, cancelled := range []bool{false, true} {
		client, calls := pythonCloudBClient(t, bandwidthCloudReplies(`{"bandwidth_limit_rules":[]}`))
		service := network.New(client)
		ctx, cancel := context.WithCancelCause(context.Background())
		cause, optionCause := errors.New("cancelled"), errors.New("option error")
		result, err := service.ListQoSBandwidthLimitRules(ctx, "parent", func(c *network.QoSBandwidthLimitOpts) error {
			c.Headers["X-Trace"] = "initialized"
			if cancelled {
				cancel(cause)
			} else {
				client.ResourceBase += "changed/"
			}
			return optionCause
		})
		cancel(nil)
		if result == nil || result.Policy == nil || !errors.Is(err, optionCause) || len(calls()) != 2 || cancelled && !errors.Is(err, cause) || !cancelled && !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err, calls())
		}
	}
}

func TestCloudQoSBandwidthPhysical401RulePageAndLateFailurePreserveStageProof(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v2.0/extensions":
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}]}`)
		case "/v2.0/qos/policies/parent":
			testcloud.JSON(w, 200, `{"policy":{"id":"parent"}}`)
		default:
			w.Header().Set("Content-Length", "1000")
			w.Header().Set("X-Request-Id", "actual-truncated")
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"error":"truncated"}`))
		}
	})
	client := cloud.Client("network", "/")
	client.ResourceBase = cloud.Server.URL + "/v2.0/"
	retry, reauth := 0, 0
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retry++
		return nil
	}
	client.ReauthFunc = func(context.Context) error { reauth++; return nil }
	result, err := network.New(client).ListQoSBandwidthLimitRules(context.Background(), "parent")
	var native *gophercloud.ErrUnableToReauthenticate
	if result == nil || result.Policy == nil || len(result.Extensions) != 1 || !errors.As(err, &native) || !errors.Is(err, io.ErrUnexpectedEOF) || retry != 0 || reauth != 0 || len(paths) != 3 {
		t.Fatal(result, err, native, retry, reauth, paths)
	}
	var proof gophercloud.ErrUnexpectedResponseCode
	if !errors.As(native.ErrOriginal, &proof) || proof.Actual != 401 || proof.ResponseHeader.Get("X-Request-Id") != "actual-truncated" {
		t.Fatal(err, proof)
	}
	replies := bandwidthCloudReplies(`{"bandwidth_limit_rules":[{"id":"observed"}],"bandwidth_limit_rules_links":[{"rel":"next","href":"?marker=later"}]}`)
	replies["GET /v2.0/qos/policies/parent/bandwidth_limit_rules"] = append(replies["GET /v2.0/qos/policies/parent/bandwidth_limit_rules"], pythonCloudBReply{503, `{"error":"later"}`})
	nextClient, calls := pythonCloudBClient(t, replies)
	later, failure := network.New(nextClient).ListQoSBandwidthLimitRules(context.Background(), "parent")
	if later == nil || later.Policy == nil || len(later.Inventory) != 1 || later.Value != nil || later.Rules != nil || failure == nil || len(calls()) != 4 {
		t.Fatal(later, failure, calls())
	}
	if !reflect.DeepEqual(paths, []string{"/v2.0/extensions", "/v2.0/qos/policies/parent", "/v2.0/qos/policies/parent/bandwidth_limit_rules"}) {
		t.Fatal(paths)
	}
}

func TestCloudQoSBandwidthSearchSourceFalseyLiteralsAndNestedMembership(t *testing.T) {
	for _, expression := range []string{"`[null,false,0,\"\",[],{}]`", "to_array(@)"} {
		client, calls := pythonCloudBClient(t, bandwidthCloudReplies(`{"bandwidth_limit_rules":[{"id":"observed"}]}`))
		raw, _ := json.Marshal(map[string]any{"jmespath_filters": expression})
		result, err := network.New(client).SearchQoSBandwidthLimitRules(context.Background(), "parent", "", network.WithQoSBandwidthLimitFilters(raw))
		if expression == "to_array(@)" {
			var proof *resource.ResponseError
			if result == nil || len(result.Inventory) != 1 || result.Value != nil || !errors.As(err, &proof) || len(calls()) != 3 {
				t.Fatal(result, err, proof, calls())
			}
		} else if err != nil || result == nil || string(result.Value) != "[]" || len(result.Inventory) != 0 || len(result.Rules) != 0 || len(calls()) != 2 {
			t.Fatal(result, err, calls())
		}
	}
	client, calls := pythonCloudBClient(t, bandwidthCloudReplies(`{"bandwidth_limit_rules":[]}`))
	result, err := network.New(client).SearchQoSBandwidthLimitRules(context.Background(), "parent", "id", network.WithQoSBandwidthLimitFilters(json.RawMessage(`{"jmespath_filters":"to_array(@)"}`)))
	if result == nil || result.Value != nil || err == nil || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}
