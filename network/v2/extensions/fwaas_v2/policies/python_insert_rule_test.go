package policies_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonFirewallPolicyInsertRuleRecordSendsNullPositions(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusOK, `{"id":"fp-1","name":"web","firewall_rules":["r1","r3","r2"],"x_extra":true}`
	var recorder pythonFWPolicyRecorder
	api := pythonFWPolicyAPI(t, &recorder, func(*http.Request) *http.Response { return pythonFWPolicyWire(status, reply) })
	// insert_rule_into_policy(policy, rule)
	got, err := api.InsertRuleRecord(ctx, "fp-1", "r3")
	if err != nil || got.StatusCode != 200 || got.Policy == nil || got.Policy.ID != "fp-1" || !reflect.DeepEqual(got.Policy.Rules, []string{"r1", "r3", "r2"}) || string(got.Fields["x_extra"]) != "true" {
		t.Fatal(got, err)
	}
	// insert_rule_into_policy(policy, rule, insert_after="r1", insert_before="")
	// with a 202 response; _put_request accepts every ok status.
	status = http.StatusAccepted
	after := "r1"
	options := policies.InsertRuleRecordOpts{InsertAfter: &after}
	option := policies.WithInsertRuleRecordOptions(options)
	after = "changed"
	if got, err := api.InsertRuleRecord(ctx, "fp-1", "r3", option, policies.WithInsertRuleRecordBefore("")); err != nil || got.StatusCode != 202 {
		t.Fatal(got, err)
	}
	// insert_rule_into_policy(policy, rule, insert_before="r2")
	if _, err := api.InsertRuleRecord(ctx, "fp-1", "r3", policies.WithInsertRuleRecordBefore("r2"), policies.WithInsertRuleRecordHeader("X-Trace", "t")); err != nil {
		t.Fatal(err)
	}
	pythonFWPolicyExpect(t, recorder.snapshot(), []pythonFWPolicyCall{
		{http.MethodPut, pythonFWPolicyBase + "/fp-1/insert_rule", "", `{"firewall_rule_id":"r3","insert_after":null,"insert_before":null}`},
		{http.MethodPut, pythonFWPolicyBase + "/fp-1/insert_rule", "", `{"firewall_rule_id":"r3","insert_after":"r1","insert_before":""}`},
		{http.MethodPut, pythonFWPolicyBase + "/fp-1/insert_rule", "", `{"firewall_rule_id":"r3","insert_after":null,"insert_before":"r2"}`},
	})
}

func TestPythonFirewallPolicyInsertRuleRecordRejectsInputsAndErrorStatuses(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusNotFound, `{"NeutronError":{"type":"FirewallPolicyNotFound","message":"missing"}}`
	var recorder pythonFWPolicyRecorder
	api := pythonFWPolicyAPI(t, &recorder, func(*http.Request) *http.Response { return pythonFWPolicyWire(status, reply) })
	for _, call := range []func() error{
		func() error { _, err := api.InsertRuleRecord(ctx, " ", "r3"); return err },
		func() error { _, err := api.InsertRuleRecord(ctx, "fp-1", "r3", nil); return err },
		func() error {
			var nilAPI *policies.API
			_, err := nilAPI.InsertRuleRecord(ctx, "fp-1", "r3")
			return err
		},
	} {
		var operation *resource.OperationError
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "InsertRuleRecord" {
			t.Fatal(err)
		}
	}
	if len(recorder.snapshot()) != 0 {
		t.Fatal(recorder.snapshot())
	}
	var operation *resource.OperationError
	if _, err := api.InsertRuleRecord(ctx, "fp-1", "r3"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) || !errors.As(err, &operation) || operation.Resource != "policies" {
		t.Fatal(err)
	}
	// resp.json() fails on a non-JSON accepted body in Python.
	status, reply = http.StatusOK, `not json`
	var receipt *resource.ResponseError
	if got, err := api.InsertRuleRecord(ctx, "fp-1", "r3"); !errors.As(err, &receipt) || receipt.StatusCode != 200 || got == nil || got.Policy != nil {
		t.Fatal(got, err)
	}
}
