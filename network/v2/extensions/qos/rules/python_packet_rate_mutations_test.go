package rules_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonRateMutationCall struct {
	method, path, query, body string
	header                    http.Header
}

func pythonRateMutationAPI(t *testing.T, calls *[]pythonRateMutationCall, reply func(*http.Request) (int, string)) *rules.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonRateMutationCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Clone()})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return rules.New(client)
}

func pythonRateMutationBody(t *testing.T, raw, key string, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil || !reflect.DeepEqual(got, map[string]any{key: want}) {
		t.Fatal(raw, want, err)
	}
}

func pythonRateMutationFailure(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "rules" {
		t.Fatal("operation context", err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatal("native status", code, err)
	}
}

const (
	pythonMinimumPacketRateBase = "/neutron/v2.0/qos/policies/p%2F1/minimum_packet_rate_rules"
	pythonPacketRateLimitBase   = "/neutron/v2.0/qos/policies/p%2F1/packet_rate_limit_rules"
)

func TestPythonMinimumPacketRateCreateUpdateDelete(t *testing.T) {
	var calls []pythonRateMutationCall
	api := pythonRateMutationAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"minimum_packet_rate_rule":{"id":"m1","min_kpps":1000,"direction":"any","qos_policy_id":"foreign"}}`
	})
	v, err := api.CreateMinimumPacketRateRecord(context.Background(), "p/1", rules.WithPacketRateRuleField("min_kpps", 1000), rules.WithPacketRateRuleField("direction", "any"), rules.WithPacketRateRuleField("x_extension", nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonMinimumPacketRateBase || calls[0].query != "" || calls[0].header.Get("Content-Type") != "application/json" {
		t.Fatal(calls)
	}
	pythonRateMutationBody(t, calls[0].body, "minimum_packet_rate_rule", map[string]any{"min_kpps": float64(1000), "direction": "any", "x_extension": nil})
	if v.QoSPolicyID != "p/1" || string(v.Resource.Body["min_kpps"]) != `1000` || string(v.Resource.Body["x_extension"]) != `null` ||
		string(v.Resource.Body["qos_policy_id"]) != `"p/1"` || string(v.Wire.Body["qos_policy_id"]) != `"foreign"` || string(v.Resource.Body["name"]) != `null` {
		t.Fatal(v.Resource.Body, v)
	}
	calls = nil
	if _, err := api.CreateMinimumPacketRateRecord(context.Background(), "p/1", rules.WithPacketRateRuleField("qos_policy_id", "x")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal("qos_policy_id duplicates the bound argument", err, calls)
	}

	api = pythonRateMutationAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"minimum_packet_rate_rule":{"id":"m1","min_kpps":2000}}`
	})
	v, err = api.UpdateMinimumPacketRateRecord(context.Background(), "p/1", "m1", rules.WithPacketRateRuleField("min_kpps", 2000), rules.WithPacketRateRuleField("if_revision", 1))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pythonMinimumPacketRateBase+"/m1" || calls[0].header.Get("If-Match") != "" {
		t.Fatal(err, calls)
	}
	pythonRateMutationBody(t, calls[0].body, "minimum_packet_rate_rule", map[string]any{"min_kpps": float64(2000)})
	if string(v.Resource.Body["min_kpps"]) != `2000` || string(v.Resource.Body["direction"]) != `null` {
		t.Fatal(v.Resource.Body)
	}
	calls = nil
	if v, err := api.UpdateMinimumPacketRateRecord(context.Background(), "p/1", "m1"); err != nil || len(calls) != 0 || string(v.Resource.Body["id"]) != `"m1"` {
		t.Fatal("no changed attribute sends nothing", v, err, calls)
	}
	if _, err := api.UpdateMinimumPacketRateRecord(context.Background(), "p/1", "m1", rules.WithPacketRateRuleField("id", "x")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}

	api = pythonRateMutationAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{}` })
	_, err = api.UpdateMinimumPacketRateRecord(context.Background(), "p/1", "m1", rules.WithPacketRateRuleField("min_kpps", 1))
	pythonRateMutationFailure(t, err, "UpdateMinimumPacketRateRecord", 404)
	calls = nil
	result, err := api.DeleteMinimumPacketRateRecord(context.Background(), "p/1", "m1")
	if err != nil || !result.IgnoredMissing || len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonMinimumPacketRateBase+"/m1" || calls[0].body != "" {
		t.Fatal(result, err, calls)
	}
	_, err = api.DeleteMinimumPacketRateRecord(context.Background(), "p/1", "m1", rules.WithPacketRateRuleDeleteIgnoreMissing(false))
	pythonRateMutationFailure(t, err, "DeleteMinimumPacketRateRecord", 404)
	api = pythonRateMutationAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
	_, err = api.CreateMinimumPacketRateRecord(context.Background(), "p/1")
	pythonRateMutationFailure(t, err, "CreateMinimumPacketRateRecord", 409)
}

func TestPythonPacketRateLimitCreateUpdateDelete(t *testing.T) {
	var calls []pythonRateMutationCall
	api := pythonRateMutationAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"packet_rate_limit_rule":{"id":"l1","max_kpps":500,"max_burst_kpps":0,"direction":"egress"}}`
	})
	v, err := api.CreatePacketRateLimitRecord(context.Background(), "p/1", rules.WithPacketRateRuleField("max_kpps", 500), rules.WithPacketRateRuleField("max_burst_kpps", 0), rules.WithPacketRateRuleField("direction", "egress"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonPacketRateLimitBase {
		t.Fatal(calls)
	}
	pythonRateMutationBody(t, calls[0].body, "packet_rate_limit_rule", map[string]any{"max_kpps": float64(500), "max_burst_kpps": float64(0), "direction": "egress"})
	if v.QoSPolicyID != "p/1" || string(v.Resource.Body["max_burst_kpps"]) != `0` || v.Resource.Body["qos_policy_id"] != nil {
		t.Fatal(v.Resource.Body)
	}

	api = pythonRateMutationAPI(t, &calls, func(*http.Request) (int, string) { return 200, `` })
	calls = nil
	v, err = api.UpdatePacketRateLimitRecord(context.Background(), "p/1", "l1", rules.WithPacketRateRuleField("max_kpps", 600))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pythonPacketRateLimitBase+"/l1" || calls[0].body != `{"packet_rate_limit_rule":{"max_kpps":600}}` {
		t.Fatal(err, calls)
	}
	if string(v.Resource.Body["max_kpps"]) != `600` || string(v.Resource.Body["id"]) != `"l1"` || v.Wire != nil {
		t.Fatal("empty accepted body keeps the seed", v)
	}
	for _, code := range []int{204, 404} {
		api = pythonRateMutationAPI(t, &calls, func(*http.Request) (int, string) { return code, `` })
		calls = nil
		result, err := api.DeletePacketRateLimitRecord(context.Background(), "p/1", "l1")
		if err != nil || result.StatusCode != code || result.IgnoredMissing != (code == 404) || len(calls) != 1 || calls[0].path != pythonPacketRateLimitBase+"/l1" {
			t.Fatal(result, err, calls)
		}
	}
	api = pythonRateMutationAPI(t, &calls, func(*http.Request) (int, string) { return 400, `{}` })
	_, err = api.CreatePacketRateLimitRecord(context.Background(), "p/1", rules.WithPacketRateRuleField("max_kpps", -1))
	pythonRateMutationFailure(t, err, "CreatePacketRateLimitRecord", 400)
	_, err = api.UpdatePacketRateLimitRecord(context.Background(), "p/1", "l1", rules.WithPacketRateRuleField("max_kpps", -1))
	pythonRateMutationFailure(t, err, "UpdatePacketRateLimitRecord", 400)
	_, err = api.DeletePacketRateLimitRecord(context.Background(), "p/1", "l1")
	pythonRateMutationFailure(t, err, "DeletePacketRateLimitRecord", 400)
	if _, err := api.CreatePacketRateLimitRecord(context.Background(), " "); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("blank policy fails before HTTP", err)
	}
}
