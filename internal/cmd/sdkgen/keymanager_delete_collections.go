package main

import (
	"fmt"
	"go/ast"
	"go/types"
)

// Redirect only the three inspected collection mutations. Generated native
// Delete remains a direct upstream alias; native Get/List/identity audits stay.
func keyManagerOwnedDeleteCollection(pkg *types.Package, plan *collectionPlan, parents int) bool {
	if parents != 0 || plan == nil {
		return false
	}
	var model, id, name string
	switch sdkPath(pkg.Path()) {
	case "keymanager/v1/containers":
		model, id, name = "Container", "ContainerRef", "Name"
	case "keymanager/v1/orders":
		model, id = "Order", "OrderRef"
	case "keymanager/v1/secrets":
		model, id, name = "Secret", "SecretRef", "Name"
	default:
		return false
	}
	if plan.modelName != model || plan.id != id || !plan.idIsURL || plan.name != name || plan.status != "Status" || !plan.listQueryBuilder || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || plan.deleter == nil || plan.deleter.Name() != "Delete" || !orderIdentityIDOperation(plan.getter, "GetResult") || !orderIdentityIDOperation(plan.deleter, "DeleteResult") {
		return false
	}
	result := plan.deleter.Type().(*types.Signature).Results().At(0).Type()
	extract, _, _ := types.LookupFieldOrMethod(result, true, nil, "ExtractErr")
	method, ok := extract.(*types.Func)
	if !ok {
		return false
	}
	sig := method.Type().(*types.Signature)
	return !sig.Variadic() && sig.Params().Len() == 0 && sig.Results().Len() == 1 && isError(sig.Results().At(0).Type())
}

func validateKeyManagerDeleteDeclarations(pkg *types.Package, declarations map[string]*ast.FuncDecl, plan *collectionPlan) error {
	var urlHash string
	switch sdkPath(pkg.Path()) {
	case "keymanager/v1/containers":
		urlHash = "8d51409b2ff452f91c16dabba7a75a6749aac41ecebc77e593ad3f1b22c67529"
	case "keymanager/v1/orders":
		urlHash = "3d5338d139d0ea6ceeb3d2ed1df1eb1bc2788b9c1cf2b49d17f62cfacb8053c1"
	case "keymanager/v1/secrets":
		urlHash = "c64e15f066c3c0b04939798e25b79b2ba75843cf8c23a761a0034ffe397ec93b"
	default:
		return nil
	}
	if !keyManagerOwnedDeleteCollection(pkg, plan, 0) {
		return fmt.Errorf("audited key-manager delete collection %s: native Get/List/Delete signature or identity changed", sdkPath(pkg.Path()))
	}
	for name, want := range map[string]string{
		"Delete":    "44c20607c63b4051a4336ed53fbd685315d4deaef21b6e12cb1a95ddb46b7275",
		"deleteURL": urlHash,
	} {
		got, err := requestDeclarationHash(declarations[name])
		if err != nil || got != want {
			return fmt.Errorf("audited key-manager delete collection %s: pinned native declaration %s changed", sdkPath(pkg.Path()), name)
		}
	}
	return nil
}
