package rules_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type ruleCall struct{ method, path, query, body, accept string }

const ruleBase = "/neutron/v2.0/metering/metering-label-rules"

func ruleAPI(t *testing.T, calls *[]ruleCall, reply func(*http.Request) (int, string)) *rules.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, ruleCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return rules.New(client)
}

func ruleFailure(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "meteringlabelrules" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if code != 0 && (!errors.As(err, &native) || native.Actual != code) {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const ruleRow = `{"id":"r1","direction":"egress","excluded":false,"metering_label_id":"ml1","tenant_id":"p1","remote_ip_prefix":null,"source_ip_prefix":"10.0.0.0/24","destination_ip_prefix":"0.0.0.0/0"}`

func ruleCheck(t *testing.T, v *rules.MeteringLabelRule) {
	t.Helper()
	if v == nil || v.ID != "r1" || v.Name != nil || *v.Direction != "egress" || *v.IsExcluded || *v.MeteringLabelID != "ml1" || *v.ProjectID != "p1" || *v.TenantID != "p1" || v.RemoteIPPrefix != nil || *v.SourceIPPrefix != "10.0.0.0/24" || *v.DestinationIPPrefix != "0.0.0.0/0" {
		t.Fatalf("decoded metering label rule: %#v", v)
	}
	if string(v.Resource.Body["is_excluded"]) != "false" || string(v.Resource.Body["remote_ip_prefix"]) != "null" || string(v.Wire.Body["excluded"]) != "false" {
		t.Fatalf("raw views: %#v %#v", v.Resource, v.Wire)
	}
}

func TestPythonMeteringLabelRuleCreatePostsAttrsWithWireNames(t *testing.T) {
	var calls []ruleCall
	api := ruleAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"metering_label_rule":` + ruleRow + `}` })
	v, err := api.Create(context.Background(),
		rules.WithCreateField("metering_label_id", "ml1"),
		rules.WithCreateField("direction", "egress"),
		rules.WithCreateField("is_excluded", false),
		rules.WithCreateField("source_ip_prefix", "10.0.0.0/24"),
		rules.WithCreateField("destination_ip_prefix", "0.0.0.0/0"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"metering_label_rule":{"destination_ip_prefix":"0.0.0.0/0","direction":"egress","excluded":false,"metering_label_id":"ml1","source_ip_prefix":"10.0.0.0/24"}}`
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != ruleBase || calls[0].body != want {
		t.Fatal(calls)
	}
	ruleCheck(t, v)
	for _, code := range []int{400, 404, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []ruleCall
			api := ruleAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Create(context.Background(), rules.WithCreateField("metering_label_id", "x"))
			ruleFailure(t, err, "Create", code)
		})
	}
}

func TestPythonMeteringLabelRuleGetMapsFieldsAndReportsMissing(t *testing.T) {
	var calls []ruleCall
	api := ruleAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"metering_label_rule":` + ruleRow + `}` })
	v, err := api.Get(context.Background(), "r1", rules.WithGetFields("is_excluded", "direction"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != ruleBase+"/r1" || calls[0].query != "fields=excluded&fields=direction" {
		t.Fatal(calls)
	}
	ruleCheck(t, v)
	api = ruleAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{}` })
	_, err = api.Get(context.Background(), "gone")
	ruleFailure(t, err, "Get", 404)
}

func TestPythonMeteringLabelRuleUpdatePutsOnlyGivenAttrs(t *testing.T) {
	var calls []ruleCall
	api := ruleAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"metering_label_rule":` + ruleRow + `}` })
	v, err := api.Update(context.Background(), "r1", rules.WithUpdateField("excluded", false))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != ruleBase+"/r1" || calls[0].body != `{"metering_label_rule":{"excluded":false}}` {
		t.Fatal(calls)
	}
	ruleCheck(t, v)
	calls = nil
	if v, err = api.Update(context.Background(), "r1"); err != nil || len(calls) != 0 || v.ID != "r1" {
		t.Fatal(v, err, calls)
	}
	if _, err = api.Update(context.Background(), "r1", rules.WithUpdateField("metering_label_rule", "x")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonMeteringLabelRuleDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, tc := range []struct {
		code    int
		strict  bool
		ignored bool
		fails   bool
	}{{204, false, false, false}, {404, false, true, false}, {404, true, false, true}, {500, false, false, true}} {
		t.Run(fmt.Sprint(tc.code, tc.strict), func(t *testing.T) {
			var calls []ruleCall
			api := ruleAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, "" })
			var options []rules.DeleteOption
			if tc.strict {
				options = append(options, rules.WithDeleteIgnoreMissing(false))
			}
			result, err := api.Delete(context.Background(), "r1", options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != ruleBase+"/r1" {
				t.Fatal(calls)
			}
			if tc.fails {
				ruleFailure(t, err, "Delete", tc.code)
				return
			}
			if err != nil || result.StatusCode != tc.code || result.IgnoredMissing != tc.ignored {
				t.Fatal(result, err)
			}
		})
	}
}

func TestPythonMeteringLabelRuleListMapsQueryAndFiltersBodyLocally(t *testing.T) {
	var calls []ruleCall
	api := ruleAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "b" {
			return 200, `{"metering_label_rules":[{"id":"c","excluded":true}]}`
		}
		return 200, `{"metering_label_rules":[{"id":"a","excluded":false},{"id":"b","excluded":true}],"metering_label_rules_links":[{"rel":"next","href":"?marker=b"}]}`
	})
	values, err := api.All(context.Background(),
		rules.WithListQuery("metering_label_id", "ml1"),
		rules.WithListQuery("direction", "egress"),
		rules.WithListQuery("remote_ip_prefix", "10.0.0.0/8"),
		rules.WithListQuery("is_excluded", true),
		rules.WithListQuery("excluded", "ignored"))
	if err != nil {
		t.Fatal(err)
	}
	want := "direction=egress&metering_label_id=ml1&remote_ip_prefix=10.0.0.0%2F8"
	if len(calls) != 2 || calls[0].path != ruleBase || calls[0].query != want || calls[0].accept != "application/json" || calls[1].query != "direction=egress&marker=b&metering_label_id=ml1&remote_ip_prefix=10.0.0.0%2F8" {
		t.Fatal(calls)
	}
	if len(values) != 2 || values[0].ID != "b" || values[1].ID != "c" {
		t.Fatalf("%#v", values)
	}
}

func TestPythonMeteringLabelRuleFindListsWithoutNameHint(t *testing.T) {
	var calls []ruleCall
	api := ruleAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path != ruleBase {
			return 400, ""
		}
		return 200, `{"metering_label_rules":[{"id":"x"},` + ruleRow + `]}`
	})
	v, err := api.Find(context.Background(), "r1", rules.WithFindQuery("metering_label_id", "ml1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].path != ruleBase+"/r1" || calls[0].query != "metering_label_id=ml1" || calls[1].path != ruleBase || calls[1].query != "metering_label_id=ml1" {
		t.Fatal(calls)
	}
	ruleCheck(t, v)
	for _, strict := range []bool{false, true} {
		calls = nil
		api = ruleAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Path != ruleBase {
				return 404, ""
			}
			return 200, `{"metering_label_rules":[]}`
		})
		v, err = api.Find(context.Background(), "none", rules.WithFindIgnoreMissing(!strict))
		if v != nil || len(calls) != 2 || strict != errors.Is(err, resource.ErrNotFound) {
			t.Fatal(strict, v, err, calls)
		}
	}
}
