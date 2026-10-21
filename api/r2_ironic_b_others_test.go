package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/deploytemplates"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/inspectionrules"
	"github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/rules"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const r2IronicBTemplateBody = `{"uuid":"22222222-2222-2222-2222-222222222222","name":"CUSTOM_T","extra":{},"steps":[{"interface":"deploy","step":"s","args":{},"priority":0}],"created_at":null,"new":1}`

func TestR2IronicBDeployTemplateCreateGetUpdatePatchDelete(t *testing.T) {
	cloud := testcloud.New(t)
	fake := r2IronicBFixed(t, cloud, 201, r2IronicBTemplateBody)
	api := deploytemplates.New(r2IronicBClient(cloud))
	ctx := context.Background()
	got, err := api.Create(ctx, deploytemplates.CreateOpts{Name: "CUSTOM_T", Steps: []deploytemplates.Step{{Interface: "deploy", Step: "s"}}, Extra: map[string]json.RawMessage{}})
	if err != nil || got.Steps[0].Priority != 0 || string(got.Body["new"]) != "1" || got.CreatedAt != nil {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	call := fake.only(t)
	if call.Method != "POST" || call.Path != "/v1/deploy_templates" || call.Body != `{"extra":{},"name":"CUSTOM_T","steps":[{"args":null,"interface":"deploy","priority":0,"step":"s"}]}` {
		t.Fatalf("call=%+v", call)
	}
	if _, err := api.Create(ctx, deploytemplates.CreateOpts{}, deploytemplates.WithCreateField("steps", 1)); err == nil {
		t.Fatal("collision")
	}

	cloud = testcloud.New(t)
	fake = r2IronicBFixed(t, cloud, 200, r2IronicBTemplateBody)
	api = deploytemplates.New(r2IronicBClient(cloud))
	if _, err := api.Get(ctx, "x", deploytemplates.WithGetFields("uuid")); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "x", deploytemplates.UpdateOpts{Name: r2IronicBPtr("n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Patch(ctx, "x", []deploytemplates.PatchOperation{{Op: "remove", Path: "/extra/k"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "x", deploytemplates.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	calls := fake.all()
	if len(calls) != 3 || calls[0].URI != "/v1/deploy_templates/x?fields=uuid" || calls[1].Body != `[{"op":"add","path":"/name","value":"n"}]` || calls[2].Method != "PATCH" {
		t.Fatalf("calls=%+v", calls)
	}

	cloud = testcloud.New(t)
	fake = r2IronicBFixed(t, cloud, 404, `{}`)
	api = deploytemplates.New(r2IronicBClient(cloud))
	if deleted, err := api.Delete(ctx, "x"); err != nil || deleted {
		t.Fatal(deleted, err)
	}
	if _, err := api.Delete(ctx, "x", deploytemplates.WithDeleteIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if fake.all()[0].Method != "DELETE" {
		t.Fatal("method")
	}
}

func TestR2IronicBDeployTemplateListDetailAndPaging(t *testing.T) {
	cloud := testcloud.New(t)
	fake := r2IronicBFixed(t, cloud, 200, `{"deploy_templates":[{"uuid":"a","name":"A"},{"uuid":"b","name":"B"}]}`)
	api := deploytemplates.New(r2IronicBClient(cloud))
	n := 0
	for v, err := range api.List(context.Background(), deploytemplates.WithListDetail(true)) {
		if err != nil || v.Name == "" {
			t.Fatal(v, err)
		}
		n++
	}
	if c := fake.only(t); n != 2 || c.RawQuery != "detail=true" {
		t.Fatalf("n=%d %+v", n, c)
	}
}

func r2IronicBFindCloud(t *testing.T, direct int, list string) (*deploytemplates.API, *r2IronicBFake) {
	cloud := testcloud.New(t)
	fake := r2IronicBServe(t, cloud, func(c r2IronicBCall) (int, string) {
		if strings.HasPrefix(c.Path, "/v1/deploy_templates/") {
			if direct == 200 {
				return 200, `{"uuid":"u","name":"direct"}`
			}
			return direct, `{}`
		}
		return 200, list
	})
	return deploytemplates.New(r2IronicBClient(cloud)), fake
}

func TestR2IronicBDeployTemplateFindDirectThenListFallback(t *testing.T) {
	ctx := context.Background()
	api, fake := r2IronicBFindCloud(t, 200, `{"deploy_templates":[]}`)
	if v, err := api.Find(ctx, "u"); err != nil || v.Name != "direct" || len(fake.all()) != 1 {
		t.Fatal(v, err)
	}
	for _, status := range []int{404, 400, 403} {
		api, fake = r2IronicBFindCloud(t, status, `{"deploy_templates":[{"uuid":"a","name":"X"},{"uuid":"b","name":"Y"}]}`)
		v, err := api.Find(ctx, "Y")
		calls := fake.all()
		if err != nil || v.UUID != "b" || len(calls) != 2 || calls[1].RawQuery != "detail=true" {
			t.Fatalf("status %d: %+v %v %+v", status, v, err, calls)
		}
	}
	api, fake = r2IronicBFindCloud(t, 404, `{"deploy_templates":[{"uuid":"a"}]}`)
	if v, err := api.Find(ctx, "nope", deploytemplates.WithFindDetail(false)); err != nil || v != nil || fake.all()[1].RawQuery != "" {
		t.Fatal(v, err)
	}
	if _, err := api.Find(ctx, "nope", deploytemplates.WithFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	api, _ = r2IronicBFindCloud(t, 404, `{"deploy_templates":[{"uuid":"a","name":"D"},{"uuid":"D","name":"z"}]}`)
	if _, err := api.Find(ctx, "D"); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	api, fake = r2IronicBFindCloud(t, 500, `{}`)
	if _, err := api.Find(ctx, "u"); !gophercloud.ResponseCodeIs(err, 500) || len(fake.all()) != 1 {
		t.Fatal(err)
	}
	if _, err := api.Find(ctx, ""); err == nil {
		t.Fatal("empty ref")
	}
	if _, err := api.Find(r2IronicBCanceled(), "u"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestR2IronicBInspectionRuleOperations(t *testing.T) {
	const body = `{"uuid":"r1","description":null,"phase":"main","priority":0,"sensitive":false,"conditions":[{"op":"eq","args":["%(a)s",1],"loop":[],"multiple":"any","x":1}],"actions":[{"op":"set-attribute","args":{"k":"v"}}],"links":[]}`
	cloud := testcloud.New(t)
	fake := r2IronicBFixed(t, cloud, 201, body)
	api := inspectionrules.New(r2IronicBClient(cloud))
	ctx := context.Background()
	got, err := api.Create(ctx, inspectionrules.CreateOpts{
		Description: r2IronicBPtr(""), Phase: r2IronicBPtr("main"), Priority: r2IronicBPtr(0), Sensitive: r2IronicBPtr(false),
		Conditions: []inspectionrules.Condition{{Op: "eq", Args: json.RawMessage(`["a",1]`)}},
		Actions:    []inspectionrules.Action{{Op: "fail", Args: json.RawMessage(`{}`)}},
	})
	if err != nil || got.Description != nil || *got.Priority != 0 || *got.Conditions[0].Multiple != "any" || string(got.Conditions[0].Extra["x"]) != "1" || string(got.Body["description"]) != "null" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if c := fake.only(t); c.Path != "/v1/inspection_rules" || c.Body != `{"actions":[{"args":{},"op":"fail"}],"conditions":[{"args":["a",1],"op":"eq"}],"description":"","phase":"main","priority":0,"sensitive":false}` {
		t.Fatalf("call=%+v", c)
	}

	cloud = testcloud.New(t)
	fake = r2IronicBFixed(t, cloud, 200, body)
	api = inspectionrules.New(r2IronicBClient(cloud))
	if _, err := api.Get(ctx, "r1", inspectionrules.WithGetFields("uuid", "phase")); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "r1", inspectionrules.UpdateOpts{Priority: r2IronicBPtr(0), Sensitive: r2IronicBPtr(false)}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Patch(ctx, "r1", []inspectionrules.PatchOperation{{Op: "test", Path: "/phase", Value: json.RawMessage(`"main"`)}}); err != nil {
		t.Fatal(err)
	}
	calls := fake.all()
	if calls[0].RawQuery != "fields=uuid%2Cphase" || calls[1].Body != `[{"op":"add","path":"/priority","value":0},{"op":"add","path":"/sensitive","value":false}]` || calls[2].Method != "PATCH" {
		t.Fatalf("calls=%+v", calls)
	}

	cloud = testcloud.New(t)
	fake = r2IronicBFixed(t, cloud, 200, `{"inspection_rules":[{"uuid":"a"}]}`)
	api = inspectionrules.New(r2IronicBClient(cloud))
	for _, err := range api.List(ctx, inspectionrules.WithListDetail(true)) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if c := fake.only(t); c.RawQuery != "detail=true" {
		t.Fatalf("call=%+v", c)
	}

	cloud = testcloud.New(t)
	fake = r2IronicBFixed(t, cloud, 204, "")
	api = inspectionrules.New(r2IronicBClient(cloud))
	if ok, err := api.Delete(ctx, "r1"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if c := fake.only(t); c.Method != "DELETE" || c.Path != "/v1/inspection_rules/r1" {
		t.Fatalf("call=%+v", c)
	}
	if _, err := api.Create(ctx, inspectionrules.CreateOpts{}, inspectionrules.WithCreateField("phase", "x")); err == nil {
		t.Fatal("collision")
	}
}

func TestR2IronicBIntrospectionRulesUseCollectionRoute(t *testing.T) {
	const full = `{"uuid":"r1","description":"d","scope":null,"conditions":[{"op":"eq","field":"memory_mb","value":1024,"invert":false}],"actions":[{"action":"set-attribute","path":"/x","value":1}],"links":[]}`
	cloud := testcloud.New(t)
	fake := r2IronicBFixed(t, cloud, 201, full)
	api := rules.New(r2IronicBInspectorClient(cloud))
	ctx := context.Background()
	got, err := api.Create(ctx, rules.CreateOpts{
		UUID: r2IronicBPtr("r1"), Description: r2IronicBPtr("d"),
		Conditions: []rules.Condition{{Op: "eq", Field: "memory_mb", Value: json.RawMessage(`1024`), Invert: r2IronicBPtr(false)}},
		Actions:    []rules.Action{{Action: "set-attribute", Params: map[string]json.RawMessage{"path": json.RawMessage(`"/x"`), "value": json.RawMessage(`1`)}}},
	})
	if err != nil || got.Scope != nil || got.Actions[0].Action != "set-attribute" || string(got.Actions[0].Params["value"]) != "1" || *got.Conditions[0].Invert {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	c := fake.only(t)
	if c.Method != "POST" || c.Path != "/v1/rules" || c.Header.Get("X-OpenStack-Ironic-Inspector-API-Version") != r2IronicBInspVer {
		t.Fatalf("call=%+v", c)
	}
	if c.Body != `{"actions":[{"action":"set-attribute","path":"/x","value":1}],"conditions":[{"field":"memory_mb","invert":false,"op":"eq","value":1024}],"description":"d","uuid":"r1"}` {
		t.Fatalf("body=%s", c.Body)
	}
	// no uuid needed: Python would fail locally, Go lets the service generate one
	if _, err := api.Create(ctx, rules.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Create(ctx, rules.CreateOpts{}, rules.WithCreateField("uuid", "x")); err == nil {
		t.Fatal("collision")
	}
	if _, err := api.Create(r2IronicBCanceled(), rules.CreateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(fake.all()) != 2 {
		t.Fatalf("calls=%d", len(fake.all()))
	}
	cloud = testcloud.New(t)
	r2IronicBFixed(t, cloud, 200, full)
	if _, err := rules.New(r2IronicBInspectorClient(cloud)).Create(ctx, rules.CreateOpts{}); !gophercloud.ResponseCodeIs(err, 200) {
		t.Fatalf("only 201: %v", err)
	}
}

func TestR2IronicBIntrospectionRulesGetListDelete(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	fake := r2IronicBServe(t, cloud, func(c r2IronicBCall) (int, string) {
		switch {
		case c.Method == "DELETE" && c.Path == "/v1/rules/gone":
			return 404, `{}`
		case c.Method == "DELETE":
			return 204, ""
		case c.Path == "/v1/rules":
			return 200, `{"rules":[{"uuid":"a","description":"","scope":"s","links":[]},{"uuid":"b"},{"uuid":"c"}],"next":"http://x/v1/rules?marker=c"}`
		}
		return 200, `{"uuid":"a","conditions":[],"actions":[]}`
	})
	api := rules.New(r2IronicBInspectorClient(cloud))
	if v, err := api.Get(ctx, "a b"); err != nil || v.Conditions == nil || fake.all()[0].Path != "/v1/rules/a%20b" {
		t.Fatal(v, err)
	}
	n := 0
	for v, err := range api.List(ctx, rules.WithListMaxItems(2)) {
		if err != nil || v.Conditions != nil {
			t.Fatal(v, err)
		}
		n++
	}
	calls := fake.all()
	if n != 2 || len(calls) != 2 || calls[1].RawQuery != "" {
		t.Fatalf("n=%d calls=%+v", n, calls)
	}
	n = 0
	for _, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 3 || len(fake.all()) != 3 {
		t.Fatalf("a next link must not be followed: n=%d calls=%d", n, len(fake.all()))
	}
	if ok, err := api.Delete(ctx, "a"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ok, err := api.Delete(ctx, "gone"); err != nil || ok {
		t.Fatal(ok, err)
	}
	if _, err := api.Delete(ctx, "gone", rules.WithDeleteIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	before := len(fake.all())
	for _, err := range api.List(ctx, rules.WithListMaxItems(-1)) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.Get(ctx, ""); err == nil {
		t.Fatal("empty id")
	}
	if len(fake.all()) != before {
		t.Fatal("invalid input reached HTTP")
	}
}
