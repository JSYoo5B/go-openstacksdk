package api_test

import (
	"context"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotaclasssets"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotasets"
	"github.com/gophercloud/gophercloud/v2"
)

// Python update_quota_set commits only the supplied attributes as
// {"quota_set": {...}} with PUT /quota-sets/{project_id}, with no force member
// unless the caller passes one. Go sends the same body for the same limits,
// with nil omitted, zero and -1 preserved and force present only when set.
func TestPyManilaBUpdateQuotaSetPutsOnlySuppliedLimits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	zero, unlimited, five := int64(0), int64(-1), int64(5)
	no := false
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"PUT /v2/quota-sets/proj-1": {status: 200, body: `{"quota_set":{"id":"proj-1","shares":5,"gigabytes":-1,"snapshots":0}}`}})
	scope, err := quotasets.New(cloud.client(pyManilaBVersion)).InProject(ctx, resource.ID("proj-1"))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := scope.Update(ctx, quotasets.UpdateOpts{Shares: &five, Gigabytes: &unlimited, Snapshots: &zero})
	if err != nil || updated.Shares == nil || *updated.Shares != 5 || updated.ProjectID != "proj-1" {
		t.Fatalf("updated=%+v error=%v", updated, err)
	}
	if _, err := scope.Update(ctx, quotasets.UpdateOpts{Shares: &five}, quotasets.WithUpdateForce(no), quotasets.WithUpdateField("vendor_limit", 3)); err != nil {
		t.Fatal(err)
	}
	cloud.expect(
		pyManilaBPut("/v2/quota-sets/proj-1", `{"quota_set":{"shares":5,"gigabytes":-1,"snapshots":0}}`),
		pyManilaBPut("/v2/quota-sets/proj-1", `{"quota_set":{"shares":5,"force":false,"vendor_limit":3}}`),
	)

	// Only HTTP 200 succeeds, and an invalid limit never reaches the server.
	odd := newPyManilaBCloud(t, map[string]pyManilaBStub{"PUT /v2/quota-sets/proj-1": {status: 202, body: `{}`}})
	oddScope, _ := quotasets.New(odd.client(pyManilaBVersion)).InProject(ctx, resource.ID("proj-1"))
	if _, err := oddScope.Update(ctx, quotasets.UpdateOpts{Shares: &five}); err == nil || !gophercloud.ResponseCodeIs(err, 202) {
		t.Fatalf("error=%v", err)
	}
	bad := int64(-2)
	if _, err := oddScope.Update(ctx, quotasets.UpdateOpts{Shares: &bad}); err == nil {
		t.Fatal("limit below -1 accepted")
	}
	if len(odd.snapshot()) != 1 {
		t.Fatalf("requests=%v", odd.snapshot())
	}
}

// Python revert_quota_set DELETEs /quota-sets/{project_id} through
// Proxy._delete with ignore_missing=True, so a 404 returns None. Go Reset
// requires 202 and reports a 404 unless WithResetIgnoreMissing(true) is set.
func TestPyManilaBRevertQuotaSetDeletesProjectOverridesAndIgnoresMissingOnRequest(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"DELETE /v2/quota-sets/proj-1": {status: 202}})
	scope, err := quotasets.New(cloud.client(pyManilaBVersion)).InProject(ctx, resource.ID("proj-1"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := scope.Reset(ctx)
	if err != nil || result == nil || result.StatusCode != 202 || result.ProjectID != "proj-1" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	cloud.expect(pyManilaBCall{method: "DELETE", path: "/v2/quota-sets/proj-1", version: pyManilaBVersion})

	missing := newPyManilaBCloud(t, map[string]pyManilaBStub{"DELETE /v2/quota-sets/proj-1": {status: 404, body: `{}`}})
	missingScope, _ := quotasets.New(missing.client(pyManilaBVersion)).InProject(ctx, resource.ID("proj-1"))
	if _, err := missingScope.Reset(ctx); err == nil || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatalf("strict reset error=%v", err)
	}
	if result, err := missingScope.Reset(ctx, quotasets.WithResetIgnoreMissing(true)); err != nil || result != nil {
		t.Fatalf("ignore-missing result=%+v error=%v", result, err)
	}
	if len(missing.snapshot()) != 2 {
		t.Fatalf("requests=%v", missing.snapshot())
	}
}

// Python update_quota_set always names the quota-sets route; Go picks the
// legacy os-quota-sets route through Manila 2.6 and quota-sets from 2.7.
func TestPyManilaBQuotaRoutesFollowTheConfiguredMicroversion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	five := int64(5)
	for version, path := range map[string]string{"2.6": "/v2/os-quota-sets/proj-1", "2.7": "/v2/quota-sets/proj-1"} {
		cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"PUT " + path: {status: 200, body: `{"quota_set":{"id":"proj-1"}}`}})
		scope, err := quotasets.New(cloud.client(version)).InProject(ctx, resource.ID("proj-1"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := scope.Update(ctx, quotasets.UpdateOpts{Shares: &five}); err != nil {
			t.Fatalf("version %s: %v", version, err)
		}
		cloud.expect(pyManilaBCall{method: "PUT", path: path, body: `{"quota_set":{"shares":5}}`, version: version})
	}
}

// Python update_quota_class_set commits {"quota_class_set": {...}} with PUT
// /quota-class-sets/{class_name}. Go InClass fixes the class name and Update
// sends the same envelope without a project lookup.
func TestPyManilaBUpdateQuotaClassSetPutsSuppliedLimitsForTheNamedClass(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	zero, unlimited := int64(0), int64(-1)
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"PUT /v2/quota-class-sets/default": {status: 200, body: `{"quota_class_set":{"id":"default","shares":0,"gigabytes":-1}}`}})
	scope, err := quotaclasssets.New(cloud.client(pyManilaBVersion)).InClass(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := scope.Update(ctx, quotaclasssets.UpdateOpts{Shares: &zero, Gigabytes: &unlimited}, quotaclasssets.WithUpdateField("vendor_limit", 3))
	if err != nil || updated.ClassName != "default" || updated.Shares == nil || *updated.Shares != 0 {
		t.Fatalf("updated=%+v error=%v", updated, err)
	}
	cloud.expect(pyManilaBPut("/v2/quota-class-sets/default", `{"quota_class_set":{"shares":0,"gigabytes":-1,"vendor_limit":3}}`))

	legacy := newPyManilaBCloud(t, map[string]pyManilaBStub{"PUT /v2/os-quota-class-sets/default": {status: 200, body: `{"quota_class_set":{"id":"default"}}`}})
	legacyScope, _ := quotaclasssets.New(legacy.client("2.6")).InClass(ctx, "default")
	if _, err := legacyScope.Update(ctx, quotaclasssets.UpdateOpts{Shares: &zero}); err != nil {
		t.Fatal(err)
	}
	legacy.expect(pyManilaBCall{method: "PUT", path: "/v2/os-quota-class-sets/default", body: `{"quota_class_set":{"shares":0}}`, version: "2.6"})
}
