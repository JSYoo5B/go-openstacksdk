package main

import (
	"fmt"
	"go/ast"
	"go/types"
)

// Some native resources identify themselves by a hostname or expose Get/List
// under longer names. Register only inspected endpoint contracts; an upstream
// shape change must fail generation rather than silently remove SDK policies.
type auditedCollectionSpec struct {
	path, model, getter, lister, identifier, name string
}

var auditedCollections = []auditedCollectionSpec{
	{"baremetal/v1/conductors", "Conductor", "Get", "List", "Hostname", "Hostname"},
	{"baremetal/v1/drivers", "Driver", "GetDriverDetails", "ListDrivers", "Name", "Name"},
}

func identifyCollectionBinding(pkg *types.Package, decls map[string]*ast.FuncDecl, extractors map[string]string) (*collectionPlan, error) {
	for _, spec := range auditedCollections {
		if sdkPath(pkg.Path()) == spec.path {
			return identifyAuditedCollection(pkg, decls, extractors, spec)
		}
	}
	return identifyCollection(pkg, decls, extractors), nil
}

func identifyAuditedCollection(pkg *types.Package, decls map[string]*ast.FuncDecl, extractors map[string]string, spec auditedCollectionSpec) (*collectionPlan, error) {
	plan := identifyNamedCollection(pkg, decls, extractors, spec.getter, []string{spec.lister}, "", 0, spec.identifier)
	mismatch := func() (*collectionPlan, error) {
		return nil, fmt.Errorf("audited collection %s requires %s.%s/%s with string %s identity and %s name", spec.path, spec.model, spec.getter, spec.lister, spec.identifier, spec.name)
	}
	if plan == nil || plan.modelName != spec.model || plan.id != spec.identifier || plan.name != spec.name || plan.status != "" || !isString(plan.getIDType) {
		return mismatch()
	}
	identifier, _, _ := types.LookupFieldOrMethod(plan.model, true, nil, plan.id)
	if identifier == nil || !isString(identifier.Type()) {
		return mismatch()
	}
	get := plan.getter.Type().(*types.Signature)
	list := plan.lister.Type().(*types.Signature)
	if options, ok := simpleInput(plan.getter, 1); !ok || options != nil || get.Params().Len() != 3 || !isContext(get.Params().At(0).Type()) || clientParam(get) != 1 || clientParam(list) != 0 {
		return mismatch()
	}
	return plan, nil
}
