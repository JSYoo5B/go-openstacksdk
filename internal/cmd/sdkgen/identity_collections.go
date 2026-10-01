package main

import "go/types"

// These collections have an audited string member route and a name-capable
// native pager. Identity fallback is an opt-in common policy, not evidence for
// every Python proxy option or every generated Get/List binding.
type identityCollectionSpec struct {
	path, model, getter, lister string
	parents                     int
}

var identityCollectionSpecs = []identityCollectionSpec{
	{"compute/v2/servers", "Server", "Get", "List", 0},
	{"blockstorage/v3/volumes", "Volume", "Get", "List", 0},
	{"network/v2/ports", "Port", "Get", "List", 0},
	{"dns/v2/recordsets", "RecordSet", "Get", "ListByZone", 1},
	{"loadbalancer/v2/pools", "Member", "GetMember", "ListMembers", 1},
}

func identityCollectionEnabled(pkg *types.Package, plan *collectionPlan, parents int) bool {
	if plan == nil || plan.getter == nil || plan.lister == nil || plan.id != "ID" || plan.name != "Name" || plan.idIsURL || !isString(plan.getIDType) || plan.listInput == nil || !plan.listQueryBuilder || plan.nameQuery != "name" {
		return false
	}
	identifier, _, _ := types.LookupFieldOrMethod(plan.model, true, nil, plan.id)
	field, ok := identifier.(*types.Var)
	if !ok || !field.IsField() || !isString(field.Type()) {
		return false
	}
	name, _, _ := types.LookupFieldOrMethod(plan.model, true, nil, plan.name)
	nameField, ok := name.(*types.Var)
	if !ok || !nameField.IsField() || !isString(nameField.Type()) {
		return false
	}
	for _, spec := range identityCollectionSpecs {
		if sdkPath(pkg.Path()) == spec.path && plan.modelName == spec.model && plan.getter.Name() == spec.getter && plan.lister.Name() == spec.lister && parents == spec.parents {
			return true
		}
	}
	return false
}

func emitIdentityFind(e *emitter, receiver, target, model string) {
	e.printf("// FindIdentity tries an ID request before exact ID/name fallback within this collection.\nfunc(%s)FindIdentity(ctx context.Context,identity string,options ...resource.IdentityFindOption)(*%s,error){return %s.FindIdentity(ctx,identity,options...)}\n", receiver, model, target)
}
