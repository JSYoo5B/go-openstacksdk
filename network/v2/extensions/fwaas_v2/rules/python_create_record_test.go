package rules_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonFirewallRuleCreateRecordSendsOnlyGivenAttributes(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusCreated, `{"firewall_rule":`+pythonFWRuleRow+`}`
	var recorder pythonFWRuleRecorder
	api := pythonFWRuleAPI(t, &recorder, func(*http.Request) *http.Response { return pythonFWRuleWire(status, reply) })
	// create_firewall_rule() sends an empty object and Neutron applies defaults.
	got, err := api.CreateRecord(ctx)
	if err != nil || got.StatusCode != 201 || got.Rule == nil || got.Rule.ID != "fr-1" || got.Rule.Protocol != "tcp" || string(got.Fields["firewall_policy_id"]) != `["fp-1"]` {
		t.Fatal(got, err)
	}
	// create_firewall_rule(name="ssh", description="", protocol=None,
	// enabled=False, ip_version=4, source_firewall_group_id="g1") with a flat
	// 200 response; Python accepts every status below 400.
	status, reply = http.StatusOK, `{"id":"fr-2","protocol":null,"enabled":false}`
	got, err = api.CreateRecord(ctx,
		rules.WithCreateRecordOptions(rules.CreateRecordOpts{
			Name: request.Present("ssh"), Description: request.Present(""), Protocol: request.Null[string](),
			Enabled: request.Present(false), IPVersion: request.Present(4),
		}),
		rules.WithCreateRecordField("source_firewall_group_id", "g1"), rules.WithCreateRecordHeader("X-Trace", "t"))
	if err != nil || got.StatusCode != 200 || got.Rule.ID != "fr-2" || got.Rule.Protocol != "" || string(got.Fields["protocol"]) != "null" || len(got.Fields) != 3 {
		t.Fatal(got, err)
	}
	// create_firewall_rule(protocol=6, action="deny") sends a declared key with
	// another JSON type through the extension when the typed field is omitted,
	// and an empty 202 body leaves only the status and headers.
	status, reply = http.StatusAccepted, ``
	got, err = api.CreateRecord(ctx, rules.WithCreateRecordOptions(rules.CreateRecordOpts{Action: request.Present("deny")}), rules.WithCreateRecordField("protocol", 6))
	if err != nil || got.StatusCode != 202 || got.Rule != nil || got.Fields != nil {
		t.Fatal(got, err)
	}
	pythonFWRuleExpect(t, recorder.snapshot(), []pythonFWRuleCall{
		{http.MethodPost, pythonFWRuleBase, "", `{"firewall_rule":{}}`},
		{http.MethodPost, pythonFWRuleBase, "", `{"firewall_rule":{"description":"","enabled":false,"ip_version":4,"name":"ssh","protocol":null,"source_firewall_group_id":"g1"}}`},
		{http.MethodPost, pythonFWRuleBase, "", `{"firewall_rule":{"action":"deny","protocol":6}}`},
	})
}

func TestPythonFirewallRuleCreateRecordRejectsInputsAndErrorStatuses(t *testing.T) {
	ctx := context.Background()
	var recorder pythonFWRuleRecorder
	api := pythonFWRuleAPI(t, &recorder, func(*http.Request) *http.Response {
		return pythonFWRuleWire(http.StatusBadRequest, `{"NeutronError":{"type":"HTTPBadRequest","message":"bad"}}`)
	})
	for _, options := range [][]rules.CreateRecordOption{
		{rules.WithCreateRecordField(" ", 1)},
		{nil},
		{rules.WithCreateRecordOptions(rules.CreateRecordOpts{Protocol: request.Present("tcp")}), rules.WithCreateRecordField("protocol", 6)},
	} {
		var operation *resource.OperationError
		if _, err := api.CreateRecord(ctx, options...); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "CreateRecord" {
			t.Fatal(err)
		}
	}
	if len(recorder.snapshot()) != 0 {
		t.Fatal(recorder.snapshot())
	}
	var operation *resource.OperationError
	if _, err := api.CreateRecord(ctx, rules.WithCreateRecordOptions(rules.CreateRecordOpts{Protocol: request.Present("bogus")})); !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) || !errors.As(err, &operation) || operation.Resource != "rules" {
		t.Fatal(err)
	}
	var nilAPI *rules.API
	if _, err := nilAPI.CreateRecord(ctx); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
