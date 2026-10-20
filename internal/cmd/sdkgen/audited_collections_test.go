package main

import (
	"fmt"
	"strings"
	"testing"
)

func auditedFixtureSource(spec auditedCollectionSpec) string {
	fields := spec.identifier + " string"
	if spec.completion != "" {
		fields += ";" + spec.completion + " bool;" + spec.failure + " string;State string"
	}
	return fmt.Sprintf(`package fixture
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
import "github.com/gophercloud/gophercloud/v2/pagination"
var _ context.Context
type %s struct{%s}
type Other struct{Name string}
type GetResult struct{}
func(GetResult)Extract()(*%s,error){return nil,nil}
type ListOpts struct{}
type ListOptsBuilder interface{ToListQuery()(string,error)}
func(ListOpts)ToListQuery()(string,error){return "",nil}
type ModelPage struct{}
func %s(ctx context.Context,client *gophercloud.ServiceClient,id string)GetResult{return GetResult{}}
func %s(client *gophercloud.ServiceClient,opts ListOptsBuilder)pagination.Pager{_ = ModelPage{};return pagination.Pager{}}
func ExtractModels(p pagination.Page)([]%s,error){_ = p.(ModelPage);return nil,nil}
`, spec.model, fields, spec.model, spec.getter, spec.lister, spec.model)
}

func TestAuditedNamedCollectionsUseInspectedIdentityAndReadOnlyPolicies(t *testing.T) {
	for _, spec := range auditedCollections {
		t.Run(spec.path, func(t *testing.T) {
			pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/"+spec.path, auditedFixtureSource(spec))
			plan, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls))
			if err != nil || plan == nil {
				t.Fatalf("plan=%v err=%v", plan, err)
			}
			if plan.id != spec.identifier || plan.name != spec.name || plan.getter.Name() != spec.getter || plan.lister.Name() != spec.lister || plan.deleter != nil || plan.status != "" {
				t.Fatalf("plan=%+v", plan)
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			emitCollectionAdapter(&e, plan, "a", nil)
			if !strings.Contains(e.body.String(), "v."+spec.identifier) || !strings.Contains(e.body.String(), "a."+spec.getter) || !strings.Contains(e.body.String(), "a."+controlledListName(spec.lister)) {
				t.Fatalf("identity and native names were not emitted: %s", e.body.String())
			}
		})
	}
}

func TestAuditedCollectionsRejectUpstreamShapeChanges(t *testing.T) {
	for _, spec := range auditedCollections {
		for name, mutate := range map[string]func(string) string{
			"missing getter": func(source string) string {
				return strings.Replace(source, "func "+spec.getter+"(", "func RenamedGetter(", 1)
			},
			"missing lister": func(source string) string {
				return strings.Replace(source, "func "+spec.lister+"(", "func RenamedLister(", 1)
			},
			"missing context": func(source string) string {
				return strings.Replace(source, "ctx context.Context,", "", 1)
			},
			"numeric identity": func(source string) string {
				return strings.Replace(source, spec.identifier+" string", spec.identifier+" int", 1)
			},
			"scoped getter": func(source string) string {
				return strings.Replace(source, "id string)GetResult", "parentID,id string)GetResult", 1)
			},
			"required getter options": func(source string) string {
				return strings.Replace(source, "id string)GetResult", "id string,opts ListOptsBuilder)GetResult", 1)
			},
			"different list model": func(source string) string {
				return strings.Replace(source, "([]"+spec.model+",error)", "([]Other,error)", 1)
			},
			"new inferred wait": func(source string) string {
				return strings.Replace(source, spec.identifier+" string", spec.identifier+" string;Status string", 1)
			},
		} {
			t.Run(spec.path+"/"+name, func(t *testing.T) {
				pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/"+spec.path, mutate(auditedFixtureSource(spec)))
				plan, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls))
				if err == nil || plan != nil || !strings.Contains(err.Error(), "audited collection "+spec.path) {
					t.Fatalf("plan=%v err=%v", plan, err)
				}
			})
		}
	}
}

func TestAuditedIntrospectionRequiresBooleanCompletionAndStringFailure(t *testing.T) {
	spec := auditedCollections[2]
	for name, source := range map[string]string{
		"string completion": strings.Replace(auditedFixtureSource(spec), "Finished bool", "Finished string", 1),
		"boolean error":     strings.Replace(auditedFixtureSource(spec), "Error string", "Error bool", 1),
	} {
		t.Run(name, func(t *testing.T) {
			pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/"+spec.path, source)
			if plan, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls)); plan != nil || err == nil {
				t.Fatalf("plan=%v err=%v", plan, err)
			}
		})
	}
}
