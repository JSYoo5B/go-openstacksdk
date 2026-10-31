package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func dscpMarkingCloudReplies(body string) map[string][]pythonCloudBReply {
	return map[string][]pythonCloudBReply{
		"GET /v2.0/extensions":                             {{200, `{"extensions":[{"alias":"qos"}]}`}},
		"GET /v2.0/qos/policies/parent":                    {{200, `{"policy":{"id":"parent","name":"policy"}}`}},
		"GET /v2.0/qos/policies/parent/dscp_marking_rules": {{200, body}},
	}
}

func TestCloudQoSDSCPMarkingListOwnsGateParentPagingAndRawFields(t *testing.T) {
	replies := dscpMarkingCloudReplies(`{"dscp_marking_rules":[{"id":"a","dscp_mark":1000,"direction":"egress","x_ext":{"keep":1},"location":{"fake":true}}],"dscp_marking_rules_links":[{"rel":"next","href":"?marker=next"}]}`)
	replies["GET /v2.0/qos/policies/parent/dscp_marking_rules"] = append(replies["GET /v2.0/qos/policies/parent/dscp_marking_rules"], pythonCloudBReply{200, `{"dscp_marking_rules":[{"id":"b","dscp_mark":20,"direction":"ingress"}]}`})
	client, calls := pythonCloudBClient(t, replies)
	service := network.New(client)
	result, err := service.ListQoSDSCPMarkingRules(context.Background(), "parent", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(`{"direction":"egress","dscp_mark":1000,"fields":["id"]}`)))
	if err != nil || result == nil || len(result.Extensions) != 1 || result.Policy == nil || len(result.Rules) != 1 || len(result.Inventory) != 1 || result.Rules[0].PolicyID != "parent" {
		t.Fatal(result, err, calls())
	}
	row := result.Rules[0]
	for key, want := range map[string]string{"id": `"a"`, "dscp_mark": "1000", "name": "null", "location": "null", "x_ext": `{"keep":1}`} {
		if string(row.Resource.Body[key]) != want {
			t.Fatal(key, row.Resource.Body)
		}
	}
	if string(row.Wire.Body["location"]) != `{"fake":true}` || row.StatusCode != 200 {
		t.Fatal(row)
	}
	pythonCloudBRequests(t, calls(), "GET /v2.0/extensions", "GET /v2.0/qos/policies/parent", "GET /v2.0/qos/policies/parent/dscp_marking_rules", "GET /v2.0/qos/policies/parent/dscp_marking_rules?marker=next")
}

func TestCloudQoSDSCPMarkingSearchKeepsRuleIDPhaseAndFilterPresence(t *testing.T) {
	for _, id := range []string{"a*", "missing", ""} {
		client, calls := pythonCloudBClient(t, dscpMarkingCloudReplies(`{"dscp_marking_rules":[{"id":"a","dscp_mark":true},{"id":"b","dscp_mark":1},{"id":"c","dscp_mark":2}]}`))
		callbacks := 0
		result, err := network.New(client).SearchQoSDSCPMarkingRules(context.Background(), "parent", id, network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(`{"dscp_mark":1}`)), func(*network.DSCPMarkingCloudOpts) error { callbacks++; return nil })
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
	client, calls := pythonCloudBClient(t, dscpMarkingCloudReplies(`{"dscp_marking_rules":[{"id":"a"}]}`))
	result, err := network.New(client).SearchQoSDSCPMarkingRules(context.Background(), "parent", "", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(`{"unknown":{"x":1}}`)))
	var proof *resource.ResponseError
	if result == nil || len(result.Inventory) != 1 || result.Value != nil || result.Rules != nil || !errors.As(err, &proof) || len(calls()) != 3 {
		t.Fatal(result, err, proof, calls())
	}
}

func TestCloudQoSDSCPMarkingGetResolvesPolicyOnceAndKeeps404(t *testing.T) {
	for _, status := range []int{200, 404} {
		replies := dscpMarkingCloudReplies(`{"dscp_marking_rules":[]}`)
		replies["GET /v2.0/qos/policies/parent/dscp_marking_rules/rule"] = []pythonCloudBReply{{status, `{"dscp_marking_rule":{"id":"rule","dscp_mark":"raw","x":null}}`}}
		client, calls := pythonCloudBClient(t, replies)
		result, err := network.New(client).GetQoSDSCPMarkingRule(context.Background(), "parent", "rule")
		if result == nil || result.Policy == nil || len(result.Extensions) != 1 || len(calls()) != 3 {
			t.Fatal(result, err, calls())
		}
		if status == 200 {
			if err != nil || result.Rule == nil || string(result.Rule.Resource.Body["dscp_mark"]) != `"raw"` || string(result.Rule.Resource.Body["location"]) != "null" {
				t.Fatal(result, err)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 404 || result.Value != nil {
				t.Fatal(result, err, native)
			}
		}
		pythonCloudBRequests(t, calls(), "GET /v2.0/extensions", "GET /v2.0/qos/policies/parent", "GET /v2.0/qos/policies/parent/dscp_marking_rules/rule")
	}
}

func TestCloudQoSDSCPMarkingGateConsumesAllExtensionsAndStopsBeforePolicy(t *testing.T) {
	for _, body := range []string{`{"extensions":[{"alias":"other"}]}`, `{"extensions":[{"alias":"qos"},{"alias":[]}]}`} {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/extensions": {{200, body}}})
		callbacks := 0
		result, err := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent", func(*network.DSCPMarkingCloudOpts) error { callbacks++; return nil })
		if result == nil || len(result.Extensions) == 0 || result.Policy != nil || result.Value != nil || err == nil || callbacks != 0 || len(calls()) != 1 {
			t.Fatal(result, err, callbacks, calls())
		}
		if len(result.Extensions) == 1 && !errors.Is(err, network.ErrUnavailableExtension) {
			t.Fatal(err)
		}
	}
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/extensions": {{200, `{"extensions":[{"alias":"qos"}],"extensions_links":[{"rel":"next","href":"?marker=later"}]}`}, {503, `{"error":"later extension failure"}`}}})
	result, err := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent")
	if result == nil || len(result.Extensions) != 1 || result.Policy != nil || err == nil || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}

func TestCloudQoSDSCPMarkingPolicyMissingAndAmbiguityRetainActualGate(t *testing.T) {
	for _, body := range []string{`{"policies":[]}`, `{"policies":[{"id":"a","name":"wanted"},{"id":"b","name":"wanted"}]}`} {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
			"GET /v2.0/extensions":          {{200, `{"extensions":[{"alias":"qos"}]}`}},
			"GET /v2.0/qos/policies/wanted": {{404, `{"error":"missing"}`}},
			"GET /v2.0/qos/policies":        {{200, body}},
		})
		result, err := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "wanted", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage("true")))
		if result == nil || len(result.Extensions) != 1 || result.Value != nil || len(calls()) != 3 || err == nil {
			t.Fatal(result, err, calls())
		}
		if body == `{"policies":[]}` && !errors.Is(err, resource.ErrNotFound) || body != `{"policies":[]}` && !errors.Is(err, resource.ErrAmbiguous) {
			t.Fatal(err)
		}
	}
}

func TestCloudQoSDSCPMarkingSourceProxyLiteralsBasePathAndOldTypedBoundary(t *testing.T) {
	for _, test := range []struct{ expression, want string }{{"`[1,null,{\"id\":\"literal\"}]`", `[1,null,{"id":"literal"}]`}, {"'한😀'", `["한","😀"]`}, {"`{\"z\":1,\"a\":2}`", `["a","z"]`}, {"to_array(@)", ""}, {"[*].id", "error"}, {"length(@)", "error"}} {
		replies := dscpMarkingCloudReplies(`{"dscp_marking_rules":[{"id":"actual"}]}`)
		client, calls := pythonCloudBClient(t, replies)
		filter, _ := json.Marshal(map[string]any{"jmespath_filters": test.expression})
		result, err := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent", network.WithDSCPMarkingCloudFiltersJSON(filter))
		if result == nil || len(calls()) != 2 || len(result.Inventory) != 0 || len(result.Rules) != 0 {
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
			g, ok := result.Values[0].(*rules.DSCPMarkingGenerator)
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
	replies := dscpMarkingCloudReplies(`{"dscp_marking_rules":[]}`)
	replies["GET /v2.0/alternate"] = []pythonCloudBReply{{200, `{"dscp_marking_rules":[{"id":"actual"}]}`}}
	client, calls := pythonCloudBClient(t, replies)
	result, err := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(`{"base_path":"alternate","headers":{"X-Trace":"raw"}}`)), func(*network.DSCPMarkingCloudOpts) error { return nil })
	if err != nil || len(result.Rules) != 1 || len(calls()) != 3 || calls()[2].path != "/v2.0/alternate" {
		t.Fatal(result, err, calls())
	}
	_, err = network.New(client).API.QoSRules.AllDSCPMarkingRecords(context.Background(), "parent", rules.WithDSCPMarkingListQuery("base_path", "alternate"))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 3 {
		t.Fatal(err, calls())
	}
}

func TestCloudQoSDSCPMarkingFilterOwnershipCallbackGuardAndFalseyCases(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage("false"), json.RawMessage("1e-400"), json.RawMessage("[]"), json.RawMessage(`""`), json.RawMessage("true"), json.RawMessage(`"expression"`)} {
		client, calls := pythonCloudBClient(t, dscpMarkingCloudReplies(`{"dscp_marking_rules":[]}`))
		result, err := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent", network.WithDSCPMarkingCloudFiltersJSON(raw))
		valid := string(raw) != "true" && string(raw) != `"expression"`
		if result == nil || valid && (err != nil || string(result.Value) != "[]" || len(calls()) != 3) || !valid && (err == nil || len(calls()) != 2) {
			t.Fatal(string(raw), result, err, calls())
		}
	}
	for _, cancelled := range []bool{false, true} {
		client, calls := pythonCloudBClient(t, dscpMarkingCloudReplies(`{"dscp_marking_rules":[]}`))
		service := network.New(client)
		ctx, cancel := context.WithCancelCause(context.Background())
		cause, optionCause := errors.New("cancelled"), errors.New("option error")
		result, err := service.ListQoSDSCPMarkingRules(ctx, "parent", func(c *network.DSCPMarkingCloudOpts) error {
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

func TestCloudQoSDSCPMarkingPhysical401RulePageAndLateFailurePreserveStageProof(t *testing.T) {
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
	result, err := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent")
	var native *gophercloud.ErrUnableToReauthenticate
	if result == nil || result.Policy == nil || len(result.Extensions) != 1 || !errors.As(err, &native) || !errors.Is(err, io.ErrUnexpectedEOF) || retry != 0 || reauth != 0 || len(paths) != 3 {
		t.Fatal(result, err, native, retry, reauth, paths)
	}
	var proof gophercloud.ErrUnexpectedResponseCode
	if !errors.As(native.ErrOriginal, &proof) || proof.Actual != 401 || proof.ResponseHeader.Get("X-Request-Id") != "actual-truncated" {
		t.Fatal(err, proof)
	}
	replies := dscpMarkingCloudReplies(`{"dscp_marking_rules":[{"id":"observed"}],"dscp_marking_rules_links":[{"rel":"next","href":"?marker=later"}]}`)
	replies["GET /v2.0/qos/policies/parent/dscp_marking_rules"] = append(replies["GET /v2.0/qos/policies/parent/dscp_marking_rules"], pythonCloudBReply{503, `{"error":"later"}`})
	nextClient, calls := pythonCloudBClient(t, replies)
	later, failure := network.New(nextClient).ListQoSDSCPMarkingRules(context.Background(), "parent")
	if later == nil || later.Policy == nil || len(later.Inventory) != 1 || later.Value != nil || later.Rules != nil || failure == nil || len(calls()) != 4 {
		t.Fatal(later, failure, calls())
	}
	if !reflect.DeepEqual(paths, []string{"/v2.0/extensions", "/v2.0/qos/policies/parent", "/v2.0/qos/policies/parent/dscp_marking_rules"}) {
		t.Fatal(paths)
	}
}

func TestCloudQoSDSCPMarkingFreshCompleteGatePolicyNameLookupAndSourceLinkPrecedence(t *testing.T) {
	replies := map[string][]pythonCloudBReply{
		"GET /v2.0/extensions":                             {{200, `{"extensions":[{"alias":"qos"}],"next":"?marker=tail"}`}, {203, `{"extensions":[{"alias":"later"}]}`}, {200, `{"extensions":[{"alias":"other"}]}`}},
		"GET /v2.0/qos/policies/gold":                      {{404, `{"missing":"name lookup"}`}},
		"GET /v2.0/qos/policies":                           {{203, `{"policies":[{"id":"parent","name":"gold"}]}`}},
		"GET /v2.0/qos/policies/parent/dscp_marking_rules": {{203, `{"dscp_marking_rules":[{"id":"first"}],"links":{},"dscp_marking_rules_links":[{"rel":"next","href":"?marker=unreachable"}]}`}},
	}
	client, calls := pythonCloudBClient(t, replies)
	service := network.New(client)
	result, err := service.ListQoSDSCPMarkingRules(context.Background(), "gold")
	if err != nil || len(result.Extensions) != 2 || len(result.Rules) != 1 || len(calls()) != 5 || result.Rules[0].StatusCode != 203 {
		t.Fatal(result, err, calls())
	}
	again, err := service.GetQoSDSCPMarkingRule(context.Background(), "gold", "first")
	if !errors.Is(err, network.ErrUnavailableExtension) || len(again.Extensions) != 1 || again.Policy != nil || len(calls()) != 6 {
		t.Fatal(again, err, calls())
	}
}
func TestCloudQoSDSCPMarkingOrderedNestedCloudFiltersAndLoadedFloatEquality(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		count   int
		failure bool
	}{
		{`{"x_new":0,"missing":1}`, 0, false}, {`{"missing":1,"x_new":0}`, 0, true},
		{`{"x_new":{"number":9007199254740992.0}}`, 1, false},
		{`{"dscp_mark":{}}`, 1, false}, {`{"dscp_mark":{"key":1}}`, 0, true},
	} {
		client, calls := pythonCloudBClient(t, dscpMarkingCloudReplies(`{"dscp_marking_rules":[{"id":"one","dscp_mark":true,"x_new":{"number":9007199254740993.0}}]}`))
		result, err := network.New(client).SearchQoSDSCPMarkingRules(context.Background(), "parent", "", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(tc.raw)))
		if result == nil || len(calls()) != 3 || tc.failure != (err != nil) || !tc.failure && len(result.Rules) != tc.count {
			t.Fatal(tc, result, err, calls())
		}
		if tc.failure {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || len(proof.Body) == 0 || result.Value != nil {
				t.Fatal(result, err, proof)
			}
		}
	}
}
func TestCloudQoSDSCPMarkingSourceFormalFiltersAndRestoredURIBeforeJMES(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	var headers http.Header
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
		switch r.URL.Path {
		case "/v2.0/extensions":
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}]}`)
		case "/v2.0/qos/policies/parent":
			testcloud.JSON(w, 200, `{"policy":{"id":"parent"}}`)
		case "/v2.0/qos/policies/foreign/dscp_marking_rules":
			headers = r.Header.Clone()
			testcloud.JSON(w, 203, `{"dscp_marking_rules":[{"id":"actual","qos_policy_id":"wire-other","dscp_mark":1}],"next":"?marker=unused"}`)
		default:
			t.Error(r.URL)
			testcloud.JSON(w, 500, `{}`)
		}
	})
	client := cloud.Client("network", "/")
	client.ResourceBase = cloud.Server.URL + "/v2.0/"
	filter := json.RawMessage(`{"base_path":"qos/policies/%(qos_policy_id)s/dscp_marking_rules","microversion":"2.77","headers":{"X-Trace":"raw"},"__conflicting_attrs":{"qos_policy_id":"foreign","paginated_extension":false},"paginated":false,"jmespath_filters":"@","max_items":1}`)
	result, err := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent", network.WithDSCPMarkingCloudFiltersJSON(filter))
	if err != nil || len(result.Rules) != 1 || len(paths) != 3 || result.Rules[0].PolicyID != "foreign" || headers.Get("X-Trace") != "raw" || headers.Get("OpenStack-API-Version") != "network 2.77" {
		t.Fatal(result, err, paths, headers)
	}
	if result.Rules[0].Resource.Body["qos_policy_id"] != nil || string(result.Rules[0].Wire.Body["qos_policy_id"]) != `"wire-other"` {
		t.Fatal(result.Rules[0])
	}
	for _, raw := range []string{`{"qos_policy_id":"duplicate"}`, `{"qos_policy":"duplicate"}`, `{"resource_type":null}`, `{"__conflicting_attrs":{"paginated":false}}`, `{"__conflicting_attrs":{"base_path":null}}`} {
		before := len(paths)
		failure, e := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(raw)))
		if failure == nil || failure.Policy == nil || e == nil || len(paths) != before+2 {
			t.Fatal(raw, failure, e, paths)
		}
	}
	empty, e := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(`{"base_path":""}`)))
	if e != nil || string(empty.Value) != "[]" {
		t.Fatal(empty, e)
	}
}
func TestCloudQoSDSCPMarkingRawFilterLocationAndResultCarriersOwnAllData(t *testing.T) {
	client, calls := pythonCloudBClient(t, dscpMarkingCloudReplies(`{"dscp_marking_rules":[{"id":"one","dscp_mark":false,"project_id":"wire-foreign","availability_zone":"wire-zone","location":{"fake":true}}]}`))
	filter := json.RawMessage(`{"dscp_mark":false}`)
	option := network.WithDSCPMarkingCloudFiltersJSON(filter)
	filter[2] = 'X'
	name, zone := "configured", json.RawMessage(`"configured-zone"`)
	location := resource.CloudLocation{Cloud: &name, Zone: zone, Project: resource.CloudProject{ID: json.RawMessage(`"current"`)}}
	locationOption := network.WithDSCPMarkingCloudCurrentLocation(location)
	name = "caller-mutated"
	zone[1] = 'X'
	callbacks := 0
	result, err := network.New(client).SearchQoSDSCPMarkingRules(context.Background(), "parent", "one", option, locationOption, func(*network.DSCPMarkingCloudOpts) error { callbacks++; return nil })
	if err != nil || callbacks != 1 || len(calls()) != 3 || len(result.Rules) != 1 {
		t.Fatal(result, err, calls(), callbacks)
	}
	row := result.Rules[0]
	var actual resource.CloudLocation
	if e := json.Unmarshal(row.Resource.Body["location"], &actual); e != nil || actual.Cloud == nil || *actual.Cloud != "configured" || string(actual.Zone) != `"configured-zone"` || string(actual.Project.ID) != `"current"` {
		t.Fatal(actual, e)
	}
	original := string(result.Value)
	row.Resource.Body["id"][1] = 'X'
	row.Wire.Body["id"][1] = 'Y'
	row.Envelope[0] = 'X'
	row.Header.Set("consumer", "changed")
	if string(result.Value) != original || string(result.Inventory[0].Resource.Body["id"]) != `"one"` || string(result.LastRule.Wire.Body["id"]) != `"one"` || result.Inventory[0].Header.Get("consumer") != "" {
		t.Fatal(result)
	}
}
func TestCloudQoSDSCPMarkingGetterPolicyStrictnessAndMalformedPolicyReceipt(t *testing.T) {
	for _, get := range []bool{false, true} {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
			"GET /v2.0/extensions":          {{200, `{"extensions":[{"alias":"qos"}]}`}},
			"GET /v2.0/qos/policies/wanted": {{404, `{"missing":true}`}},
			"GET /v2.0/qos/policies":        {{203, `{"policies":[]}`}},
		})
		var err error
		var result *network.DSCPMarkingCloudResult
		if get {
			result, err = network.New(client).GetQoSDSCPMarkingRule(context.Background(), "wanted", "rule")
		} else {
			result, err = network.New(client).ListQoSDSCPMarkingRules(context.Background(), "wanted")
		}
		if result == nil || len(result.Extensions) != 1 || result.Policy != nil || !errors.Is(err, resource.ErrNotFound) || len(calls()) != 3 {
			t.Fatal(result, err, calls())
		}
	}
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
		"GET /v2.0/extensions":          {{200, `{"extensions":[{"alias":"qos"}]}`}},
		"GET /v2.0/qos/policies/parent": {{203, `{"policy":{"id":true}}`}},
	})
	result, err := network.New(client).GetQoSDSCPMarkingRule(context.Background(), "parent", "rule")
	var proof *resource.ResponseError
	if result == nil || result.Policy == nil || len(calls()) != 2 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != `{"policy":{"id":true}}` {
		t.Fatal(result, err, calls(), proof)
	}
}
func TestCloudQoSDSCPMarkingPhysicalLatePagesKeepActualHeaderStatusAndNoFaultHooks(t *testing.T) {
	for _, status := range []int{200, 203, 401, 404, 429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			paths := []string{}
			faultBody := `{"error":"actual physical page"}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
				if r.URL.Path == "/v2.0/extensions" {
					testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}]}`)
					return
				}
				if r.URL.Path == "/v2.0/qos/policies/parent" {
					testcloud.JSON(w, 200, `{"policy":{"id":"parent"}}`)
					return
				}
				if r.URL.Query().Get("marker") == "later" {
					w.Header().Set("Content-Length", "1000")
					w.Header().Set("X-Request-Id", "actual-final")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(faultBody))
					return
				}
				testcloud.JSON(w, 203, `{"dscp_marking_rules":[{"id":"observed"}],"next":"?marker=later"}`)
			})
			client := cloud.Client("network", "/")
			client.ResourceBase = cloud.Server.URL + "/v2.0/"
			retries, reauths, backoffs := 0, 0, 0
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			client.ReauthFunc = func(context.Context) error { reauths++; return nil }
			client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoffs++
				return nil
			}
			result, err := network.New(client).SearchQoSDSCPMarkingRules(context.Background(), "parent", "", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(`{}`)))
			if result == nil || len(result.Inventory) != 1 || result.Value != nil || result.Rules != nil || len(paths) != 4 || !errors.Is(err, io.ErrUnexpectedEOF) || retries != 0 || reauths != 0 || backoffs != 0 {
				t.Fatal(result, err, paths, retries, reauths, backoffs)
			}
			if status < 400 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != status || proof.Header.Get("X-Request-Id") != "actual-final" || string(proof.Body) != faultBody {
					t.Fatal(proof, err)
				}
			} else {
				original := err
				if status == 401 {
					var unable *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &unable) {
						t.Fatal(err)
					}
					original = unable.ErrOriginal
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(original, &native) || native.Actual != status || native.ResponseHeader.Get("X-Request-Id") != "actual-final" || string(native.Body) != faultBody {
					t.Fatal(native, err)
				}
			}
			if status == 401 {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &native) || !errors.Is(native.ErrReauth, io.ErrUnexpectedEOF) {
					t.Fatal(native, err)
				}
			}
		})
	}
}

func TestCloudQoSDSCPMarkingFalseyMaxItemsAndTemplateOnlyJSONArguments(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `0.0`, `1e-400`, `""`, `[]`, `{}`, `true`} {
		client, calls := pythonCloudBClient(t, dscpMarkingCloudReplies(`{"dscp_marking_rules":[{"id":"actual"}]}`))
		result, err := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(`{"max_items":`+raw+`}`)))
		if err != nil || len(result.Rules) != 1 || len(calls()) != 3 {
			t.Fatal(raw, result, err, calls())
		}
		want := ""
		if raw == "true" {
			want = "1"
		}
		if calls()[2].query.Get("limit") != want {
			t.Fatal(raw, calls())
		}
	}
	replies := dscpMarkingCloudReplies(`{"dscp_marking_rules":[]}`)
	replies["GET /v2.0/alternate/hello"] = []pythonCloudBReply{{200, `{"dscp_marking_rules":[{"id":"actual"}]}`}}
	client, calls := pythonCloudBClient(t, replies)
	result, err := network.New(client).ListQoSDSCPMarkingRules(context.Background(), "parent", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(`{"base_path":"alternate/%(route_fragment)s","route_fragment":"hello"}`)))
	if err != nil || len(result.Rules) != 1 || len(calls()) != 3 || len(calls()[2].query) != 0 || calls()[2].path != "/v2.0/alternate/hello" {
		t.Fatal(result, err, calls())
	}
}
func TestCloudQoSDSCPMarkingPreflightAndInvalidLocationHaveHonestEvidence(t *testing.T) {
	var service *network.Service
	result, err := service.ListQoSDSCPMarkingRules(context.Background(), "parent")
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	client, calls := pythonCloudBClient(t, dscpMarkingCloudReplies(`{"dscp_marking_rules":[{"id":"actual"}]}`))
	result, err = network.New(client).ListQoSDSCPMarkingRules(nil, "parent")
	if result != nil || err == nil || len(calls()) != 0 {
		t.Fatal(result, err, calls())
	}
	location := resource.CloudLocation{Zone: json.RawMessage(`broken`)}
	result, err = network.New(client).GetQoSDSCPMarkingRule(context.Background(), "parent", "actual", network.WithDSCPMarkingCloudFiltersJSON(json.RawMessage(`{}`)))
	if result == nil || result.Policy == nil || err == nil || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
	// Fresh fixture: invalid JSON location is rejected with the actual member receipt.
	replies := dscpMarkingCloudReplies(`{"dscp_marking_rules":[]}`)
	replies["GET /v2.0/qos/policies/parent/dscp_marking_rules/actual"] = []pythonCloudBReply{{203, `{"dscp_marking_rule":{"id":"actual"}}`}}
	client, calls = pythonCloudBClient(t, replies)
	result, err = network.New(client).GetQoSDSCPMarkingRule(context.Background(), "parent", "actual", network.WithDSCPMarkingCloudCurrentLocation(location))
	var proof *resource.ResponseError
	if result == nil || result.LastRule == nil || result.Rule != nil || result.Value != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || len(calls()) != 3 {
		t.Fatal(result, err, proof, calls())
	}
}

func TestCloudQoSDSCPMarkingPolymorphicFalseySearchAndNestedGeneratorMembership(t *testing.T) {
	for _, tc := range []struct {
		expression, rule string
		rows             int
		failure          bool
	}{
		{"`[null,false,0,[],{}]`", "", 0, false},
		{"`[null]`", "id", 0, true},
		{"to_array(@)", "", 1, true},
		{"to_array(@)", "id", 0, true},
	} {
		client, calls := pythonCloudBClient(t, dscpMarkingCloudReplies(`{"dscp_marking_rules":[{"id":"actual"}]}`))
		raw, _ := json.Marshal(map[string]any{"jmespath_filters": tc.expression})
		result, err := network.New(client).SearchQoSDSCPMarkingRules(context.Background(), "parent", tc.rule, network.WithDSCPMarkingCloudFiltersJSON(raw))
		if result == nil || tc.failure != (err != nil) || len(result.Inventory) != tc.rows || len(calls()) != 2+tc.rows {
			t.Fatal(tc, result, err, calls())
		}
		if !tc.failure && string(result.Value) != "[]" || tc.failure && result.Value != nil {
			t.Fatal(tc, result, err)
		}
		if tc.rows == 1 {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != `{"dscp_marking_rules":[{"id":"actual"}]}` {
				t.Fatal(proof, err)
			}
		}
	}
}
