package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
)

const qosPolicySDKPath = "network/v2/extensions/qos/policies"

func qosPolicyBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{{key: "rules", member: "Rules"}, {key: "tenant_id", member: "TenantID"}}
}

func qosPolicyBodyCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	return spec.path == qosPolicySDKPath && spec.model == "Policy" && spec.rawRecord && reflect.DeepEqual(spec.fields, qosPolicyBodyCollectionFields())
}

func qosPolicyBodyNativeSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "Policy" || plan.id != "ID" || plan.idIsURL || plan.name != "Name" || plan.nameQuery != "name" || plan.status != "" || plan.statusQuery != "" || !plan.listQueryBuilder || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || !types.Identical(plan.getIDType, types.Typ[types.String]) || !identityQoSAddressSchema(pkg, plan) {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok || model.NumMethods() != 0 || !rawBodyFieldsMatch(plan.listInput, "q", map[string][2]string{"ID": {"string", "id"}, "TenantID": {"string", "tenant_id"}, "ProjectID": {"string", "project_id"}, "Name": {"string", "name"}, "Description": {"string", "description"}, "IsDefault": {"*bool", "is_default"}, "Shared": {"*bool", "shared"}, "Limit": {"int", "limit"}, "Marker": {"string", "marker"}, "SortKey": {"string", "sort_key"}, "SortDir": {"string", "sort_dir"}, "Tags": {"string", "tags"}, "TagsAny": {"string", "tags-any"}, "NotTags": {"string", "not-tags"}, "NotTagsAny": {"string", "not-tags-any"}, "RevisionNumber": {"*int", "revision_number"}}) {
		return false
	}
	list, ok := plan.listInput.(*types.Named)
	if !ok || list.NumMethods() != 1 || list.Method(0).Name() != "ToPolicyListQuery" {
		return false
	}
	query := list.Method(0).Type().(*types.Signature)
	if !types.Identical(query.Recv().Type(), list) || query.Variadic() || query.Params().Len() != 0 || query.Results().Len() != 2 || !types.Identical(query.Results().At(0).Type(), types.Typ[types.String]) || !isError(query.Results().At(1).Type()) {
		return false
	}
	builder := pkg.Scope().Lookup("PolicyListOptsBuilder")
	if builder == nil {
		return false
	}
	builderType, ok := builder.Type().Underlying().(*types.Interface)
	if !ok || builderType.NumEmbeddeds() != 0 || builderType.NumMethods() != 1 || builderType.Method(0).Name() != "ToPolicyListQuery" {
		return false
	}
	interfaceQuery := builderType.Method(0).Type().(*types.Signature)
	if interfaceQuery.Variadic() || interfaceQuery.Params().Len() != 0 || interfaceQuery.Results().Len() != 2 || !types.Identical(interfaceQuery.Results().At(0).Type(), types.Typ[types.String]) || !isError(interfaceQuery.Results().At(1).Type()) {
		return false
	}
	if !rawBodyNativePage(pkg, model, "PolicyPage", "ExtractPolicies") || identityGetResult(pkg, plan, 0) == nil {
		return false
	}
	for _, dependency := range pkg.Imports() {
		if dependency.Path() != upstreamModule {
			continue
		}
		object := dependency.Scope().Lookup("Link")
		if object == nil {
			return false
		}
		link, ok := object.Type().(*types.Named)
		return ok && link.NumMethods() == 0 && rawBodyFieldsMatch(link, "json", map[string][2]string{"Href": {"string", "href"}, "Rel": {"string", "rel"}})
	}
	return false
}

var qosPolicyBodyNativeDeclarations = map[string]string{
	"Get":                                    "f2e21c88e17a8ad49060e84dad42244e17c64453bae932aadebe34ad0e7ebc9b",
	"List":                                   "04d9ad4cd55b330c83fc0c5e013b75491d40a00b0c1c3615414bc1bd8095f5e7",
	"ListOpts.ToPolicyListQuery":             "d6253365fadc272a3df97e2569174347bd27f82ea52eac33b26b5fbe7c2bbcfe",
	"rootURL":                                "4f185db1078eefd1b5f5b38bd1c601bd6fccaa3c20fa99a7b85371ef7f7b8104",
	"resourceURL":                            "ca0246cf0e192c0133b2d43b9c5b577badf51c51344d3c5fc499d01a97ef54c8",
	"listURL":                                "0a55ee851552d789ddd3c12304e6d0d209cae0cfd477d71b138196681486bf9d",
	"getURL":                                 "4e7e71e4b36e374a2fc6830f4f621ab3dd904cb2e1fa10b99e43a5590b7b4b36",
	"PolicyPage.NextPageURL":                 "7b97a390dc18f3874d6a0bcca3e030ed8ad60129e7a6d8310227785a8d6602ca",
	"PolicyPage.IsEmpty":                     "88fa833c9862dd1c989f138f4614c917e32483879512f3880ef9ea9dc3aeebd7",
	"ExtractPolicies":                        "9be7267032ad765ebed4e8f5deb34f80b6804da02b42288ff362106f314789b6",
	"ExtractPolicysInto":                     "8ac0e33d92c10eabc76680aa76b5dd845589c8c45adf242b0ec59ad14cba50de",
	"commonResult.Extract":                   "9c467d68ba024143e8a8ba5a4a69db0e027a035507c38e1aa0387a13d1c191c9",
	"pagination.PageResultFrom":              "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9",
	"pagination.PageResultFromParsed":        "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089",
	"pagination.LinkedPageBase.GetBody":      "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284",
	"gophercloud.ExtractNextURL":             "e0ce1875e9f11ea6812ed64aa5f42e095fa53dace0a482f099c05ffb99322dc2",
	"gophercloud.Result.ExtractInto":         "a978ec69fee3b64d971a9a749c20845d5419425d1ef304d24e82c118bcf53a57",
	"gophercloud.Result.ExtractIntoSlicePtr": "eed0d552518f57bf120ea36fd9a2ef6bc17288ab9be239754a205f9d6130befc",
	"gophercloud.Result.extractIntoPtr":      "418acd501c229b1cf45b844a5948aa59f06c62c04e0ac99ee2a94f14f0d8d4d2",
}

func validateQoSPolicyBodyNativeDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) != qosPolicySDKPath {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited QoSPolicy body collection: native model/list schema changed")
	}
	for name, want := range qosPolicyBodyNativeDeclarations {
		got, err := requestDeclarationHash(decls[name])
		if err != nil || got != want {
			return fmt.Errorf("audited QoSPolicy body collection: pinned native declaration %s changed", name)
		}
	}
	return nil
}

func emitQoSPolicyBodyRecordAdapter(e *emitter, plan *collectionPlan) {
	emitNeutronBodyRecordAdapter(e, plan, "qosPolicyBodyFilterValue")
}
func emitQoSPolicyBodyFilterList(e *emitter, plan *collectionPlan) {
	emitNeutronBodyFilterList(e, plan, "policies", "policies", "ExtractPolicies")
}
