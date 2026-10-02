package main

import (
	"strings"
	"testing"
)

func TestIdentityAuditedBindingsRetainRawStatusWithoutInferringTypedStatus(t *testing.T) {
	for _, spec := range identityCollectionSpecs {
		t.Run(spec.path, func(t *testing.T) {
			pkg, plan := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
			nativeNetworkStatus := spec.path == networkSDKPath || spec.path == routerSDKPath
			if !identityCollectionEnabled(pkg, plan, spec.parents) || (nativeNetworkStatus && (plan.status != "Status" || plan.statusQuery != "status")) || (!nativeNetworkStatus && (plan.status != "" || plan.statusQuery != "")) {
				t.Fatal("fixture must preserve audited native status ownership", plan)
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			parents := []string(nil)
			if spec.parents != 0 {
				parents = []string{"a.parentID"}
			}
			e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", spec.model)
			emitCollectionAdapter(&e, plan, "a", parents)
			e.printf("}\n")
			source, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			body := string(source)
			if strings.Contains(body, `q.Del("status")`) || strings.Contains(body, `q.Set("status"`) || strings.Contains(body, "LocalStatus:") || (!nativeNetworkStatus && strings.Contains(body, "Status:")) {
				t.Fatal("raw status was discarded or invented a typed status capability", body)
			}
			if spec.rawListIterator != "" {
				if !strings.Contains(body, "return nativefind.IterateSecurityGroups(ctx, a.RawClient(), q, control)") {
					t.Fatal(body)
				}
			} else if !strings.Contains(body, "config.Query[key] = append([]string(nil), values...)") {
				t.Fatal("whole raw query lost", body)
			}
		})
	}
}

func TestIdentityMemberTypedStatusIsLocalWhileRawStatusRetainsItsWireKey(t *testing.T) {
	for _, spec := range identityCollectionSpecs {
		if spec.model != "Member" {
			continue
		}
		source := strings.Replace(identityQueryFixtureSource(spec), "type Member struct{ID string;Name string}", "type Member struct{ID string;Name string;ProvisioningStatus string}", 1)
		pkg, plan := identityQueryFixture(t, spec, source)
		if plan.status != "ProvisioningStatus" || plan.statusQuery != "" {
			t.Fatal(plan)
		}
		e := emitter{pkg: pkg, imports: map[string]string{}}
		e.printf("func(s *MemberScope)newResources()*resource.Collection[Member]{return ")
		emitCollectionAdapter(&e, plan, "s.api", []string{"s.parentID"})
		e.printf("}\n")
		data, err := e.source()
		if err != nil {
			t.Fatal(err)
		}
		body := string(data)
		for _, want := range []string{"LocalStatus: true", "v.ProvisioningStatus", "config.Query[key] = append([]string(nil), values...)", "s.api.listMembersWithControl(ctx, s.parentID, control, options...)", `[]string{"lbaas", "pools", s.parentID, "members", id}, q, []int{200}`} {
			if !strings.Contains(body, want) {
				t.Fatal("member source/status policy lost", want, body)
			}
		}
		if strings.Count(body, "LocalStatus: true") != 1 || strings.Contains(body, `q.Del("status")`) || strings.Contains(body, `q.Set("provisioning_status"`) {
			t.Fatal("member raw status changed", body)
		}
		return
	}
	t.Fatal("audited member scope is missing")
}

func TestIdentityUnauditedBindingsKeepTheirExistingStatusQueryPolicy(t *testing.T) {
	for _, wireKey := range []string{"", "state"} {
		t.Run(wireKey, func(t *testing.T) {
			options := "type ListOpts struct{Name string `q:\"name\"`"
			if wireKey != "" {
				options += ";Status string `q:\"" + wireKey + "\"`"
			}
			options += "}\ntype ListOptsBuilder interface{ToListQuery()(string,error)}\nfunc(ListOpts)ToListQuery()(string,error){return \"\",nil}"
			source := controlledFixtureSource("Thing", "ID string;Name string;Status string", "Get", "id string", "List", ",opts ListOptsBuilder", options)
			pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/unreviewed/v1/things", source)
			plan := identifyCollection(pkg, decls, extractorsByPage(pkg, decls))
			if plan == nil || identityCollectionEnabled(pkg, plan, 0) {
				t.Fatal(plan)
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			e.printf("func(a *API)newResources()*resource.Collection[Thing]{return ")
			emitCollectionAdapter(&e, plan, "a", nil)
			e.printf("}\n")
			data, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			body := string(data)
			if !strings.Contains(body, `q.Del("status")`) || strings.Contains(body, "LocalStatus:") || strings.Contains(body, "IdentityFind:") {
				t.Fatal("unaudited policy changed", body)
			}
			if wireKey != "" && !strings.Contains(body, `q.Set("state", value)`) {
				t.Fatal("unaudited wire alias lost", body)
			}
		})
	}
}
