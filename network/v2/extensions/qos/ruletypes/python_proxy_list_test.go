package ruletypes_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestQoSRuleTypeProxyOwnsRawFormalsAndTypedBasePathKeepsOldRawRejection(t *testing.T) {
	var calls []nativeRuleTypeCall
	api := recordAPI(t, &calls, func(req *http.Request, _ int) *http.Response {
		if req.URL.Path != "/neutron/v2.0/qos/alternate-types" || req.Header.Get("X-Source-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.6" || req.URL.Query().Get("all_supported") != "True" {
			t.Fatal(req.URL, req.Header)
		}
		return recordWire(200, `{"rule_types":[{"type":"bandwidth_limit"}]}`)
	})
	query := map[string]json.RawMessage{"base_path": json.RawMessage(`"qos/alternate-types"`), "headers": json.RawMessage(`{"X-Source-Trace":"owned"}`), "microversion": json.RawMessage(`"2.6"`), "all_supported": json.RawMessage(`true`)}
	order := []string{"base_path", "headers", "microversion", "all_supported"}
	option := ruletypes.WithProxyListOpts(ruletypes.ProxyListOpts{Query: query, Order: order})
	query["headers"][0] = 'x'
	order[0] = "changed"
	result, err := api.ProxyListRecords(context.Background(), option)
	if err != nil || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
	generator, ok := result.Value.(*ruletypes.ProxyListGenerator)
	if !ok || result.JSON != nil {
		t.Fatal(result)
	}
	for record, err := range generator.Records {
		if err != nil || record.Wire == nil {
			t.Fatal(record, err)
		}
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
}

func TestQoSRuleTypeProxySourceFormalsStayLazyBehindConstantExpressions(t *testing.T) {
	for _, raw := range []string{`42`, `false`, `[]`, `{"wrong":null}`} {
		var calls []nativeRuleTypeCall
		api := recordAPI(t, &calls, func(*http.Request, int) *http.Response { t.Fatal("unexpected HTTP"); return nil })
		result, err := api.ProxyListRecords(context.Background(), ruletypes.WithProxyListQuery("jmespath_filters", "`[1]`"), ruletypes.WithProxyListQuery("headers", json.RawMessage(raw)))
		if err != nil || string(result.JSON) != `[1]` || len(calls) != 0 {
			t.Fatal(raw, result, err, calls)
		}
	}
	var calls []nativeRuleTypeCall
	api := recordAPI(t, &calls, func(*http.Request, int) *http.Response { t.Fatal("unexpected HTTP"); return nil })
	result, err := api.ProxyListRecords(context.Background(), ruletypes.WithProxyListQuery("headers", 42))
	if err != nil {
		t.Fatal(err)
	}
	generator := result.Value.(*ruletypes.ProxyListGenerator)
	count := 0
	for _, err := range generator.Records {
		count++
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if count != 1 || len(calls) != 0 {
		t.Fatal(count, calls)
	}
}

func TestQoSRuleTypeProxyOriginalExpressionWinsAfterOneConflictPass(t *testing.T) {
	var calls []nativeRuleTypeCall
	api := recordAPI(t, &calls, func(*http.Request, int) *http.Response { t.Fatal("unexpected HTTP"); return nil })
	result, err := api.ProxyListRecords(context.Background(), ruletypes.WithProxyListQuery("jmespath_filters", "`[1]`"), ruletypes.WithProxyListQuery("__conflicting_attrs", json.RawMessage(`{"jmespath_filters":"[0]","headers":42,"microversion":[]}`)))
	if err != nil || string(result.JSON) != `[1]` || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
	for _, field := range []string{"paginated", "base_path"} {
		_, err := api.ProxyListRecords(context.Background(), ruletypes.WithProxyListQuery("jmespath_filters", "`[1]`"), ruletypes.WithProxyListQuery("__conflicting_attrs", map[string]any{field: nil}))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(field, err, calls)
		}
	}
}

type proxyRuleTypeBadMarshal struct{ invoked *int }

func (value proxyRuleTypeBadMarshal) MarshalJSON() ([]byte, error) {
	*value.invoked++
	return nil, errors.New("must remain discarded")
}
func TestQoSRuleTypeProxyUnknownValuesAndCallbackMapsAreOwned(t *testing.T) {
	invoked := 0
	unknown := ruletypes.WithProxyListQuery("undeclared", proxyRuleTypeBadMarshal{&invoked})
	var calls []nativeRuleTypeCall
	var held map[string]json.RawMessage
	api := recordAPI(t, &calls, func(req *http.Request, _ int) *http.Response {
		held["type"] = json.RawMessage(`"changed"`)
		if req.URL.Query().Get("type") != "dscp_marking" {
			t.Fatal(req.URL)
		}
		return recordWire(200, `{"rule_types":[{"type":"dscp_marking"}]}`)
	})
	result, err := api.ProxyListRecords(context.Background(), unknown, func(c *ruletypes.ProxyListOpts) error {
		c.Query["type"] = json.RawMessage(`"dscp_marking"`)
		held = c.Query
		return nil
	})
	if err != nil || invoked != 0 {
		t.Fatal(result, err, invoked)
	}
	count := 0
	for record, err := range result.Value.(*ruletypes.ProxyListGenerator).Records {
		if err != nil || string(record.Resource.Body["type"]) != `"dscp_marking"` {
			t.Fatal(record, err)
		}
		count++
	}
	if count != 1 || !reflect.DeepEqual(calls, []nativeRuleTypeCall{{"GET", "/neutron/v2.0/qos/rule-types", "type=dscp_marking"}}) {
		t.Fatal(count, calls)
	}
}

func TestQoSRuleTypeProxyGuardsObserveEveryOptionAndOpaqueIdentity(t *testing.T) {
	var calls []nativeRuleTypeCall
	api := recordAPI(t, &calls, func(*http.Request, int) *http.Response { t.Fatal("unexpected HTTP"); return nil })
	base := api.RawClient().ResourceBase
	later := false
	_, err := api.ProxyListRecords(context.Background(), func(*ruletypes.ProxyListOpts) error { api.RawClient().ResourceBase += "changed/"; return nil }, func(*ruletypes.ProxyListOpts) error { later = true; api.RawClient().ResourceBase = base; return nil })
	if err == nil || later || len(calls) != 0 {
		t.Fatal(err, later, calls)
	}
	api.RawClient().ResourceBase = base
	result, err := api.ProxyListRecords(context.Background(), ruletypes.WithProxyListQuery("jmespath_filters", "@ == @"))
	if err != nil || string(result.JSON) != `true` || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
}
