package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
)

func containerBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{{key: "consumers"}, {key: "container_id"}, {key: "container_ref"}, {key: "created_at"}, {key: "id"}, {key: "name"}, {key: "secret_refs"}, {key: "status"}, {key: "type"}, {key: "updated_at"}}
}

func containerBodyCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	return spec.path == "keymanager/v1/containers" && spec.model == "Container" && spec.rawRecord && reflect.DeepEqual(spec.fields, containerBodyCollectionFields())
}

func containerBodyNativeSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "Container" || !plan.listQueryBuilder || plan.nameQuery != "name" || plan.statusQuery != "" || plan.id != "ContainerRef" || !plan.idIsURL || plan.name != "Name" || plan.status != "Status" || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || !isString(plan.getIDType) {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok || !rawBodyNativeDecoder(model) || !rawBodyFieldsMatch(model, "json", map[string][2]string{
		"Consumers": {"[]" + pkg.Path() + ".ConsumerRef", "consumers"}, "ContainerRef": {"string", "container_ref"}, "Created": {"time.Time", "-"}, "CreatorID": {"string", "creator_id"}, "Name": {"string", "name"},
		"SecretRefs": {"[]" + pkg.Path() + ".SecretRef", "secret_refs"}, "Status": {"string", "status"}, "Type": {"string", "type"}, "Updated": {"time.Time", "-"},
	}) || !rawBodyFieldsMatch(plan.listInput, "q", map[string][2]string{"Limit": {"int", "limit"}, "Name": {"string", "name"}, "Offset": {"int", "offset"}}) {
		return false
	}
	for name, fields := range map[string]map[string][2]string{
		"ConsumerRef": {"Name": {"string", "name"}, "URL": {"string", "url"}},
		"SecretRef":   {"Name": {"string", "name"}, "SecretRef": {"string", "secret_ref"}},
	} {
		object := pkg.Scope().Lookup(name)
		if object == nil {
			return false
		}
		nested, ok := object.Type().(*types.Named)
		if !ok || !rawBodyFieldsMatch(nested, "json", fields) {
			return false
		}
		if name == "ConsumerRef" {
			if nested.NumMethods() != 0 {
				return false
			}
			continue
		}
		// SecretRef is also the native create/delete reference request builder.
		// Its one ordinary builder method does not replace JSON row decoding.
		if nested.NumMethods() != 1 || nested.Method(0).Name() != "ToContainerSecretRefMap" {
			return false
		}
		sig := nested.Method(0).Type().(*types.Signature)
		if !types.Identical(sig.Recv().Type(), nested) || sig.Variadic() || sig.Params().Len() != 0 || sig.Results().Len() != 2 || !isError(sig.Results().At(1).Type()) {
			return false
		}
		body := types.NewMap(types.Typ[types.String], types.NewInterfaceType(nil, nil).Complete())
		if !types.Identical(sig.Results().At(0).Type(), body) {
			return false
		}
	}
	return rawBodyNativePage(pkg, plan.model, "ContainerPage", "ExtractContainers")
}

var containerBodyNativeDeclarations = map[string]string{
	"ListOpts.ToContainerListQuery":     "93479112439a3ea672cb5a093f14f20dd430ae148caf73e663d250810b3743af",
	"List":                              "2d4370124eff6fbfdee66aa3ba27c01ee709509f7e9e0620839ff03a8bde5c4c",
	"listURL":                           "e6fc58e355e51548ad3c15fffb55ba8f2d9bf67598dc29ee7f5d91b03f1f192a",
	"Container.UnmarshalJSON":           "719857cb89ce0f4749f704e637231c48257d8055a8a07b2ea52f2b7e452fd475",
	"ContainerPage.IsEmpty":             "2ae1cd50c5dbb4ff98e55468b8261a65b8fb2b8f0b238b21b03f3796cc1c0f61",
	"ContainerPage.NextPageURL":         "1337656907bfaf9f5f7470a347241d4f30c8ad6217fd99e4537934f28d253db9",
	"ExtractContainers":                 "48497f576be0ed6592bc5fba7457dcb8132558ba77330fe4dd13a20a473429d8",
	"Get":                               "aa6bf83ccd6b2db8e3c0d09e73ca2cd3d7d4e69d9ff8d604080177e38eb40232",
	"getURL":                            "b5c32254b3d75aa05c681b914c143cbc4141962365254f296106108bacbb0e7f",
	"commonResult.Extract":              "8acf6ec1f0282cf300ceb3a549ff51d9995f8d37acfb6fb9c6954a186e0a6b70",
	"pagination.PageResultFrom":         "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9",
	"pagination.PageResultFromParsed":   "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089",
	"pagination.LinkedPageBase.GetBody": "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284",
}

func validateContainerBodyNativeDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) != "keymanager/v1/containers" {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited Container body collection: native model/list schema changed")
	}
	for name, want := range containerBodyNativeDeclarations {
		got, err := requestDeclarationHash(decls[name])
		if err != nil || got != want {
			return fmt.Errorf("audited Container body collection: pinned native declaration %s changed", name)
		}
	}
	return nil
}
