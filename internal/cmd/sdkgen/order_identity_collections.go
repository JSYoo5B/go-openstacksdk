package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
)

// An order can contain references to its resulting secret and container.
// Only its own OrderRef identifies the order collection resource.
func identifyOrderCollectionIdentity(pkg *types.Package, decls map[string]*ast.FuncDecl, extractors map[string]string) (*collectionPlan, error) {
	plan := identifyNamedCollection(pkg, decls, extractors, "Get", []string{"List"}, "Delete", 0, "OrderRef")
	mismatch := func() (*collectionPlan, error) {
		return nil, fmt.Errorf("audited order collection requires Order.Get/List/Delete with string OrderRef identity, Status and no top-level Name")
	}
	if plan == nil || plan.modelName != "Order" || plan.id != "OrderRef" || !plan.idIsURL || plan.name != "" || plan.status != "Status" || plan.deleter == nil || !plan.listQueryBuilder || plan.nameQuery != "" || plan.statusQuery != "" {
		return mismatch()
	}
	model, ok := plan.model.(*types.Named)
	if !ok || model.Obj().Pkg() != pkg || model.Obj().Name() != "Order" || !orderIdentityStringField(model, "OrderRef", "order_ref") || !orderIdentityStringField(model, "Status", "status") {
		return mismatch()
	}
	if name, _, _ := types.LookupFieldOrMethod(model, true, nil, "Name"); name != nil {
		return mismatch()
	}
	if !orderIdentityIDOperation(plan.getter, "GetResult") || !orderIdentityIDOperation(plan.deleter, "DeleteResult") {
		return mismatch()
	}
	list := plan.lister.Type().(*types.Signature)
	if list.Variadic() || list.Params().Len() != 2 || !orderIdentityServiceClient(list.Params().At(0).Type()) || list.Results().Len() != 1 || !isPager(list.Results().At(0).Type()) {
		return mismatch()
	}
	builder, ok := list.Params().At(1).Type().(*types.Named)
	if !ok || builder.Obj().Pkg() != pkg || builder.Obj().Name() != "ListOptsBuilder" {
		return mismatch()
	}
	iface, ok := builder.Underlying().(*types.Interface)
	if !ok || iface.NumMethods() != 1 || iface.Method(0).Name() != "ToOrderListQuery" {
		return mismatch()
	}
	query := iface.Method(0).Type().(*types.Signature)
	if query.Variadic() || query.Params().Len() != 0 || query.Results().Len() != 2 || !types.Identical(query.Results().At(0).Type(), types.Typ[types.String]) || !isError(query.Results().At(1).Type()) {
		return mismatch()
	}
	return plan, nil
}

func orderIdentityStringField(model *types.Named, name, tag string) bool {
	fields, ok := model.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := 0; i < fields.NumFields(); i++ {
		field := fields.Field(i)
		if field.Name() == name {
			return !field.Embedded() && types.Identical(field.Type(), types.Typ[types.String]) && reflect.StructTag(fields.Tag(i)).Get("json") == tag
		}
	}
	return false
}

func orderIdentityIDOperation(operation *types.Func, resultName string) bool {
	if operation == nil {
		return false
	}
	sig := operation.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 3 || !isContext(sig.Params().At(0).Type()) || !orderIdentityServiceClient(sig.Params().At(1).Type()) || !types.Identical(sig.Params().At(2).Type(), types.Typ[types.String]) || sig.Results().Len() != 1 {
		return false
	}
	result, ok := sig.Results().At(0).Type().(*types.Named)
	return ok && result.Obj().Pkg() == operation.Pkg() && result.Obj().Name() == resultName
}

func orderIdentityServiceClient(value types.Type) bool {
	return types.TypeString(value, func(pkg *types.Package) string { return pkg.Path() }) == "*"+upstreamModule+".ServiceClient"
}
