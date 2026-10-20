package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/l7policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const octaviaL7PolicyBody = `{"l7policy":{"id":"pol-1","name":"redir","listener_id":"lis-1","action":"REDIRECT_TO_URL","position":1,"redirect_url":"https://example.test/","redirect_http_code":302,"admin_state_up":false,"provisioning_status":"PENDING_CREATE","operating_status":"OFFLINE","rules":[{"id":"rule-1"}],"vendor_extra":1}}`
const octaviaL7RuleBody = `{"rule":{"id":"rule-1","type":"HEADER","compare_type":"EQUAL_TO","value":"v","key":"X-A","invert":true,"admin_state_up":false,"provisioning_status":"PENDING_CREATE","operating_status":"OFFLINE","tags":["blue"],"vendor_extra":1}}`

func newOctaviaL7Policies(t *testing.T, cloud *testcloud.Cloud) *l7policies.API {
	t.Helper()
	return l7policies.New(cloud.Client("load-balancer", "/v2"))
}

func octaviaL7Bool(value bool) *bool       { return &value }
func octaviaL7String(value string) *string { return &value }

func octaviaL7Request(t *testing.T, r *http.Request, envelope string) map[string]string {
	t.Helper()
	var body map[string]map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("request body: %v", err)
	}
	if len(body) != 1 || body[envelope] == nil {
		t.Errorf("envelope %q missing: %v", envelope, body)
	}
	got := map[string]string{}
	for key, value := range body[envelope] {
		got[key] = string(value)
	}
	return got
}

func octaviaL7Unexpected(t *testing.T, err error, status int) {
	t.Helper()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &unexpected) || unexpected.Actual != status {
		t.Fatalf("want unexpected status %d, got %v", status, err)
	}
}

func TestOctaviaL7PolicyCreateKeepsFalseNestedRulesAndExtensionFields(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/lbaas/l7policies", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := map[string]string{"listener_id": `"lis-1"`, "action": `"REDIRECT_TO_URL"`, "redirect_url": `"https://example.test/"`, "admin_state_up": `false`,
			"rules": `[{"admin_state_up":false,"compare_type":"EQUAL_TO","key":"X-A","type":"HEADER","value":"v"}]`, "tags": `["blue"]`, "vendor": `{"enabled":false}`}
		if got := octaviaL7Request(t, r, "l7policy"); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 201, octaviaL7PolicyBody)
	})
	value, err := newOctaviaL7Policies(t, cloud).Create(context.Background(), l7policies.CreateOpts{
		ListenerID: "lis-1", Action: l7policies.ActionRedirectToURL, RedirectURL: "https://example.test/", AdminStateUp: octaviaL7Bool(false), Tags: []string{"blue"},
		Rules: []l7policies.CreateRuleOpts{{RuleType: l7policies.TypeHeader, CompareType: l7policies.CompareTypeEqual, Value: "v", Key: "X-A", AdminStateUp: octaviaL7Bool(false)}},
	}, l7policies.WithCreateField("vendor", map[string]bool{"enabled": false}))
	if err != nil || value == nil || value.ID != "pol-1" || value.AdminStateUp || value.RedirectHttpCode != 302 || len(value.Rules) != 1 || value.Rules[0].ID != "rule-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestOctaviaL7PolicyCreateAcceptsOnly201Or202(t *testing.T) {
	for _, status := range []int{201, 202, 200} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/lbaas/l7policies", func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, status, octaviaL7PolicyBody)
		})
		value, err := newOctaviaL7Policies(t, cloud).Create(context.Background(), l7policies.CreateOpts{Action: l7policies.ActionReject})
		if (status != 200) != (err == nil) || (err == nil && value.ID != "pol-1") {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if status == 200 {
			octaviaL7Unexpected(t, err, 200)
		}
	}
}

func TestOctaviaL7PolicyRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newOctaviaL7Policies(t, cloud)
	ctx := context.Background()
	rule := l7policies.CreateRuleOpts{RuleType: l7policies.TypePath, CompareType: l7policies.CompareTypeEqual, Value: "/"}
	checks := map[string]error{}
	_, checks["protected create field"] = api.Create(ctx, l7policies.CreateOpts{Action: l7policies.ActionReject}, l7policies.WithCreateField("action", "x"))
	_, checks["protected update field"] = api.Update(ctx, "pol-1", l7policies.UpdateOpts{}, l7policies.WithUpdateField("position", 2))
	_, checks["protected create rule field"] = api.CreateRule(ctx, "pol-1", rule, l7policies.WithCreateRuleField("type", "x"))
	_, checks["protected update rule field"] = api.UpdateRule(ctx, "pol-1", "rule-1", l7policies.UpdateRuleOpts{}, l7policies.WithUpdateRuleField("invert", true))
	_, checks["nil create option"] = api.Create(ctx, l7policies.CreateOpts{Action: l7policies.ActionReject}, nil)
	_, checks["nil update option"] = api.Update(ctx, "pol-1", l7policies.UpdateOpts{}, nil)
	_, checks["nil create rule option"] = api.CreateRule(ctx, "pol-1", rule, nil)
	_, checks["nil update rule option"] = api.UpdateRule(ctx, "pol-1", "rule-1", l7policies.UpdateRuleOpts{}, nil)
	for name, err := range checks {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("nil list option: %v", err)
		}
	}
	for _, err := range api.ListRules(ctx, "pol-1", nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("nil list rules option: %v", err)
		}
	}
	if _, err := api.Create(ctx, l7policies.CreateOpts{}); err == nil {
		t.Error("missing required action must fail locally")
	}
	if _, err := api.CreateRule(ctx, "pol-1", l7policies.CreateRuleOpts{}); err == nil {
		t.Error("missing required rule type, compare_type and value must fail locally")
	}
}

func TestOctaviaL7PolicyUpdateSendsEmptyAndNullValuesAndAcceptsOnly200(t *testing.T) {
	for _, status := range []int{200, 202, 201} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/l7policies/pol-1", func(w http.ResponseWriter, r *http.Request) {
			want := map[string]string{"name": `""`, "description": `""`, "admin_state_up": `false`, "tags": `[]`,
				"redirect_pool_id": `null`, "redirect_url": `null`, "redirect_prefix": `null`}
			if got := octaviaL7Request(t, r, "l7policy"); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			testcloud.JSON(w, status, octaviaL7PolicyBody)
		})
		empty, tags := "", []string{}
		value, err := newOctaviaL7Policies(t, cloud).Update(context.Background(), "pol-1", l7policies.UpdateOpts{
			Name: &empty, Description: &empty, AdminStateUp: octaviaL7Bool(false), Tags: &tags,
			RedirectPoolID: &empty, RedirectURL: &empty, RedirectPrefix: &empty,
		})
		if (status == 200) != (err == nil) || (err == nil && value.ID != "pol-1") {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if status != 200 {
			octaviaL7Unexpected(t, err, status)
		}
	}
}

func TestOctaviaL7PolicyUpdateOmitsZeroPositionAndCode(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]string
	cloud.Mux.HandleFunc("PUT /v2/lbaas/l7policies/pol-1", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, octaviaL7Request(t, r, "l7policy"))
		testcloud.JSON(w, 200, octaviaL7PolicyBody)
	})
	api := newOctaviaL7Policies(t, cloud)
	if _, err := api.Update(context.Background(), "pol-1", l7policies.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(context.Background(), "pol-1", l7policies.UpdateOpts{Position: 2, RedirectHttpCode: 301, Action: l7policies.ActionRedirectToPool}, l7policies.WithUpdateField("vendor", nil)); err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{{}, {"position": `2`, "redirect_http_code": `301`, "action": `"REDIRECT_TO_POOL"`, "vendor": `null`}}
	if !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies=%v", bodies)
	}
}

func TestOctaviaL7PolicyDeleteGetAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/l7policies/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "ok200":
			testcloud.JSON(w, 200, `{}`)
		case "ok202":
			w.WriteHeader(202)
		default:
			w.WriteHeader(204)
		}
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/l7policies/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, octaviaL7PolicyBody)
	})
	api := newOctaviaL7Policies(t, cloud)
	ctx := context.Background()
	for _, id := range []string{"pol-1", "ok202"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	octaviaL7Unexpected(t, api.Delete(ctx, "ok200"), 200)
	octaviaL7Unexpected(t, api.Delete(ctx, "missing"), 404)
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	value, err := api.Get(ctx, "pol-1")
	if err != nil || value.Name != "redir" || value.Action != "REDIRECT_TO_URL" || value.Position != 1 || value.ProvisioningStatus != "PENDING_CREATE" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(ctx, "missing")
	octaviaL7Unexpected(t, err, 404)
}

func TestOctaviaL7PolicyListFiltersFollowLinksAndTypedBoolLimit(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/l7policies", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "pol-1" {
			testcloud.JSON(w, 200, `{"l7policies":[{"id":"pol-2","name":"two"}],"l7policies_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/lbaas/l7policies?marker=pol-1&limit=1"
		testcloud.JSON(w, 200, `{"l7policies":[{"id":"pol-1","name":"one"}],"l7policies_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newOctaviaL7Policies(t, cloud)
	var ids []string
	for value, err := range api.List(context.Background(),
		l7policies.WithListOptions(l7policies.ListOpts{ListenerID: "lis-1", Action: "REJECT", Limit: 1}),
		l7policies.WithListQuery("admin_state_up", "false"), l7policies.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"pol-1", "pol-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	first := requests[0]
	if first.Get("listener_id") != "lis-1" || first.Get("action") != "REJECT" || first.Get("limit") != "1" || first.Get("admin_state_up") != "false" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "pol-1" {
		t.Fatalf("second query=%v", requests[1])
	}
	requests = nil
	for _, err := range api.List(context.Background(), l7policies.WithListOptions(l7policies.ListOpts{AdminStateUp: false})) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(requests) == 0 || requests[0].Has("admin_state_up") {
		t.Fatalf("typed ListOpts cannot express admin_state_up=false; query=%v", requests)
	}
}

func TestOctaviaL7PolicyCommonLookupFindsExactName(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/l7policies/redir", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/l7policies", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"l7policies":[{"id":"pol-1","name":"redir"},{"id":"pol-2","name":"redir-2"}]}`)
	})
	value, err := newOctaviaL7Policies(t, cloud).Find(context.Background(), resource.Name("redir"))
	if err != nil || value == nil || value.ID != "pol-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if len(queries) == 0 || queries[0].Get("name") != "redir" {
		t.Fatalf("queries=%v", queries)
	}
}

func TestOctaviaL7RuleCreateKeepsFalseAndExtensionFieldsAndAccepts201Or202(t *testing.T) {
	for _, status := range []int{201, 202, 200} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/lbaas/l7policies/pol-1/rules", func(w http.ResponseWriter, r *http.Request) {
			want := map[string]string{"type": `"HEADER"`, "compare_type": `"EQUAL_TO"`, "value": `"v"`, "key": `"X-A"`, "admin_state_up": `false`, "tags": `["blue"]`, "vendor": `false`}
			if got := octaviaL7Request(t, r, "rule"); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			testcloud.JSON(w, status, octaviaL7RuleBody)
		})
		value, err := newOctaviaL7Policies(t, cloud).CreateRule(context.Background(), "pol-1", l7policies.CreateRuleOpts{
			RuleType: l7policies.TypeHeader, CompareType: l7policies.CompareTypeEqual, Value: "v", Key: "X-A", AdminStateUp: octaviaL7Bool(false), Tags: []string{"blue"},
		}, l7policies.WithCreateRuleField("vendor", false))
		if (status != 200) != (err == nil) || (err == nil && (value.ID != "rule-1" || !value.Invert || value.AdminStateUp || value.Key != "X-A")) {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if status == 200 {
			octaviaL7Unexpected(t, err, 200)
		}
	}
}

func TestOctaviaL7RuleUpdateSendsNullKeyAndExplicitFalseAndAccepts200To202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {201, true}, {202, true}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/l7policies/pol-1/rules/rule-1", func(w http.ResponseWriter, r *http.Request) {
			want := map[string]string{"key": `null`, "invert": `false`, "admin_state_up": `false`, "tags": `[]`}
			if got := octaviaL7Request(t, r, "rule"); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaL7RuleBody)
		})
		tags := []string{}
		value, err := newOctaviaL7Policies(t, cloud).UpdateRule(context.Background(), "pol-1", "rule-1", l7policies.UpdateRuleOpts{
			Key: octaviaL7String(""), Invert: octaviaL7Bool(false), AdminStateUp: octaviaL7Bool(false), Tags: &tags,
		})
		if tc.success != (err == nil) || (tc.success && value.ID != "rule-1") {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			octaviaL7Unexpected(t, err, tc.status)
		}
	}
}

func TestOctaviaL7RuleGetDeleteAndNativeMissing(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/lbaas/l7policies/pol-1/rules/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, octaviaL7RuleBody)
	})
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/l7policies/pol-1/rules/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "ok200":
			testcloud.JSON(w, 200, `{}`)
		case "ok202":
			w.WriteHeader(202)
		default:
			w.WriteHeader(204)
		}
	})
	api := newOctaviaL7Policies(t, cloud)
	ctx := context.Background()
	value, err := api.GetRule(ctx, "pol-1", "rule-1")
	if err != nil || value.RuleType != "HEADER" || value.CompareType != "EQUAL_TO" || value.Value != "v" || !reflect.DeepEqual(value.Tags, []string{"blue"}) {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.GetRule(ctx, "pol-1", "missing")
	octaviaL7Unexpected(t, err, 404)
	for _, id := range []string{"rule-1", "ok202"} {
		if err := api.DeleteRule(ctx, "pol-1", id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	octaviaL7Unexpected(t, api.DeleteRule(ctx, "pol-1", "ok200"), 200)
	octaviaL7Unexpected(t, api.DeleteRule(ctx, "pol-1", "missing"), 404)
}

func TestOctaviaL7RuleListFiltersFollowLinksAndTypedBoolLimit(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/l7policies/pol-1/rules", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "rule-1" {
			testcloud.JSON(w, 200, `{"rules":[{"id":"rule-2"}],"rules_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/lbaas/l7policies/pol-1/rules?marker=rule-1&limit=1"
		testcloud.JSON(w, 200, `{"rules":[{"id":"rule-1"}],"rules_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newOctaviaL7Policies(t, cloud)
	var ids []string
	for value, err := range api.ListRules(context.Background(), "pol-1",
		l7policies.WithListRulesOptions(l7policies.ListRulesOpts{RuleType: l7policies.TypeHeader, Limit: 1}),
		l7policies.WithListRulesQuery("invert", "false")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"rule-1", "rule-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	if first := requests[0]; first.Get("type") != "HEADER" || first.Get("limit") != "1" || first.Get("invert") != "false" {
		t.Fatalf("first query=%v", first)
	}
	requests = nil
	for _, err := range api.ListRules(context.Background(), "pol-1", l7policies.WithListRulesOptions(l7policies.ListRulesOpts{Invert: false})) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(requests) == 0 || requests[0].Has("invert") {
		t.Fatalf("typed ListRulesOpts cannot express invert=false; query=%v", requests)
	}
}

func TestOctaviaL7RuleScopeFixesParentAndResolvesReferences(t *testing.T) {
	cloud := testcloud.New(t)
	var policyLookups, ruleQueries []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/l7policies/redir", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/l7policies", func(w http.ResponseWriter, r *http.Request) {
		policyLookups = append(policyLookups, r.URL.Query())
		testcloud.JSON(w, 200, `{"l7policies":[{"id":"pol-1","name":"redir"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/l7policies/pol-1/rules", func(w http.ResponseWriter, r *http.Request) {
		ruleQueries = append(ruleQueries, r.URL.Query())
		testcloud.JSON(w, 200, `{"rules":[{"id":"rule-1","type":"PATH","provisioning_status":"ACTIVE"},{"id":"rule-2","type":"HEADER","provisioning_status":"PENDING_UPDATE"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/l7policies/pol-1/rules/rule-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, octaviaL7RuleBody)
	})
	cloud.Mux.HandleFunc("PUT /v2/lbaas/l7policies/pol-1/rules/rule-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, octaviaL7RuleBody)
	})
	cloud.Mux.HandleFunc("POST /v2/lbaas/l7policies/pol-1/rules", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, octaviaL7RuleBody)
	})
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/l7policies/pol-1/rules/rule-1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
	api := newOctaviaL7Policies(t, cloud)
	ctx := context.Background()
	byID, err := api.Rules(ctx, resource.ID("pol-1"))
	if err != nil || len(policyLookups) != 0 {
		t.Fatalf("explicit parent ID must not need a lookup: scope=%v error=%v lookups=%v", byID, err, policyLookups)
	}
	byName, err := api.Rules(ctx, resource.Name("redir"))
	if err != nil || len(policyLookups) == 0 || policyLookups[0].Get("name") != "redir" {
		t.Fatalf("name parent lookup: error=%v lookups=%v", err, policyLookups)
	}
	if _, err := byName.Create(ctx, l7policies.CreateRuleOpts{RuleType: l7policies.TypeHeader, CompareType: l7policies.CompareTypeEqual, Value: "v"}); err != nil {
		t.Fatal(err)
	}
	if value, err := byName.Update(ctx, resource.ID("rule-1"), l7policies.UpdateRuleOpts{Value: "w"}); err != nil || value.ID != "rule-1" {
		t.Fatalf("update value=%+v error=%v", value, err)
	}
	if value, err := byID.Find(ctx, resource.ID("rule-1")); err != nil || value.ID != "rule-1" {
		t.Fatalf("find value=%+v error=%v", value, err)
	}
	if err := byID.Delete(ctx, resource.ID("rule-1")); err != nil {
		t.Fatal(err)
	}
	rows, err := byID.All(ctx, resource.WithStatus("ACTIVE"), resource.WithQuery("vendor", "x"))
	if err != nil || len(rows) != 1 || rows[0].ID != "rule-1" || len(ruleQueries) != 1 || ruleQueries[0].Get("vendor") != "x" || ruleQueries[0].Has("status") {
		t.Fatalf("rows=%v error=%v queries=%v", rows, err, ruleQueries)
	}
	// The status filter is applied client-side against provisioning_status and is never sent as a query.
	if _, err := byID.Find(ctx, resource.Name("x")); err == nil {
		t.Fatal("rules have no name lookup")
	}
}

func TestOctaviaL7CancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newOctaviaL7Policies(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	rule := l7policies.CreateRuleOpts{RuleType: l7policies.TypePath, CompareType: l7policies.CompareTypeEqual, Value: "/"}
	errs := map[string]error{}
	_, errs["get"] = api.Get(canceled, "pol-1")
	_, errs["create"] = api.Create(canceled, l7policies.CreateOpts{Action: l7policies.ActionReject})
	_, errs["update"] = api.Update(canceled, "pol-1", l7policies.UpdateOpts{})
	errs["delete"] = api.Delete(canceled, "pol-1")
	_, errs["get rule"] = api.GetRule(canceled, "pol-1", "rule-1")
	_, errs["create rule"] = api.CreateRule(canceled, "pol-1", rule)
	_, errs["update rule"] = api.UpdateRule(canceled, "pol-1", "rule-1", l7policies.UpdateRuleOpts{})
	errs["delete rule"] = api.DeleteRule(canceled, "pol-1", "rule-1")
	for _, err := range api.List(canceled) {
		errs["list"] = err
	}
	for _, err := range api.ListRules(canceled, "pol-1") {
		errs["list rules"] = err
	}
	for name, err := range errs {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled %s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.CreateRule(deadline, "pol-1", rule); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
