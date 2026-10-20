package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
)

const subnetPoolSDKPath = "network/v2/extensions/subnetpools"

func subnetPoolBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{
		{key: "created_at", member: "CreatedAt"}, {key: "default_prefixlen", member: "DefaultPrefixLen", aliases: []string{"default_prefix_length"}}, {key: "default_quota", member: "DefaultQuota"}, {key: "id", member: "ID"}, {key: "max_prefixlen", member: "MaxPrefixLen", aliases: []string{"maximum_prefix_length"}}, {key: "min_prefixlen", member: "MinPrefixLen", aliases: []string{"minimum_prefix_length"}}, {key: "prefixes", member: "Prefixes"}, {key: "revision_number", member: "RevisionNumber"}, {key: "tenant_id", member: "TenantID"}, {key: "updated_at", member: "UpdatedAt"}}
}
func subnetPoolBodyCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	return spec.path == subnetPoolSDKPath && spec.model == "SubnetPool" && spec.rawRecord && reflect.DeepEqual(spec.fields, subnetPoolBodyCollectionFields())
}
func subnetPoolPointerDecoder(model *types.Named) bool {
	if !rawBodyNativeDecoder(model) {
		return false
	}
	sig := model.Method(0).Type().(*types.Signature)
	return model.Method(0).Name() == "UnmarshalJSON" && types.Identical(sig.Recv().Type(), types.NewPointer(model))
}
func subnetPoolBodyNativeSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "SubnetPool" || plan.id != "ID" || plan.idIsURL || plan.name != "Name" || plan.nameQuery != "name" || plan.status != "" || plan.statusQuery != "" || !plan.listQueryBuilder || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || !types.Identical(plan.getIDType, types.Typ[types.String]) {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok || !subnetPoolPointerDecoder(model) || !rawBodyFieldsMatch(model, "json", map[string][2]string{"ID": {"string", "id"}, "Name": {"string", "name"}, "DefaultQuota": {"int", "default_quota"}, "TenantID": {"string", "tenant_id"}, "ProjectID": {"string", "project_id"}, "CreatedAt": {"time.Time", "-"}, "UpdatedAt": {"time.Time", "-"}, "Prefixes": {"[]string", "prefixes"}, "DefaultPrefixLen": {"int", "-"}, "MinPrefixLen": {"int", "-"}, "MaxPrefixLen": {"int", "-"}, "AddressScopeID": {"string", "address_scope_id"}, "IPversion": {"int", "ip_version"}, "Shared": {"bool", "shared"}, "Description": {"string", "description"}, "IsDefault": {"bool", "is_default"}, "RevisionNumber": {"int", "revision_number"}, "Tags": {"[]string", "tags"}}) || !rawBodyFieldsMatch(plan.listInput, "q", map[string][2]string{"ID": {"string", "id"}, "Name": {"string", "name"}, "DefaultQuota": {"int", "default_quota"}, "TenantID": {"string", "tenant_id"}, "ProjectID": {"string", "project_id"}, "DefaultPrefixLen": {"int", "default_prefixlen"}, "MinPrefixLen": {"int", "min_prefixlen"}, "MaxPrefixLen": {"int", "max_prefixlen"}, "AddressScopeID": {"string", "address_scope_id"}, "IPVersion": {"int", "ip_version"}, "Shared": {"*bool", "shared"}, "Description": {"string", "description"}, "IsDefault": {"*bool", "is_default"}, "Limit": {"int", "limit"}, "Marker": {"string", "marker"}, "SortKey": {"string", "sort_key"}, "SortDir": {"string", "sort_dir"}, "Tags": {"string", "tags"}, "TagsAny": {"string", "tags-any"}, "NotTags": {"string", "not-tags"}, "NotTagsAny": {"string", "not-tags-any"}, "RevisionNumber": {"int", "revision_number"}}) {
		return false
	}
	list, ok := plan.listInput.(*types.Named)
	if !ok || list.NumMethods() != 1 || list.Method(0).Name() != "ToSubnetPoolListQuery" {
		return false
	}
	query := list.Method(0).Type().(*types.Signature)
	if !types.Identical(query.Recv().Type(), list) || !subnetPoolListQuerySignature(query) {
		return false
	}
	object := pkg.Scope().Lookup("ListOptsBuilder")
	if object == nil {
		return false
	}
	builder, ok := object.Type().Underlying().(*types.Interface)
	if !ok || builder.NumEmbeddeds() != 0 || builder.NumMethods() != 1 || builder.Method(0).Name() != "ToSubnetPoolListQuery" || !subnetPoolListQuerySignature(builder.Method(0).Type().(*types.Signature)) {
		return false
	}
	if !rawBodyNativePage(pkg, model, "SubnetPoolPage", "ExtractSubnetPools") {
		return false
	}
	result := identityGetResult(pkg, plan, 0)
	if result == nil || result.Obj().Name() != "GetResult" || result.NumMethods() != 0 {
		return false
	}
	fields, ok := result.Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 1 || !fields.Field(0).Embedded() || types.TypeString(fields.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != pkg.Path()+".commonResult" {
		return false
	}
	common, ok := fields.Field(0).Type().(*types.Named)
	if !ok || common.NumMethods() != 1 || common.Method(0).Name() != "Extract" {
		return false
	}
	sig := common.Method(0).Type().(*types.Signature)
	if !types.Identical(sig.Recv().Type(), common) {
		return false
	}
	for _, dependency := range pkg.Imports() {
		if dependency.Path() != upstreamModule {
			continue
		}
		linkObj := dependency.Scope().Lookup("Link")
		timeObj := dependency.Scope().Lookup("JSONRFC3339NoZ")
		if linkObj == nil || timeObj == nil {
			return false
		}
		link, ok := linkObj.Type().(*types.Named)
		if !ok || link.NumMethods() != 0 || !rawBodyFieldsMatch(link, "json", map[string][2]string{"Href": {"string", "href"}, "Rel": {"string", "rel"}}) {
			return false
		}
		wrapper, ok := timeObj.Type().(*types.Named)
		if !ok || !subnetPoolPointerDecoder(wrapper) {
			return false
		}
		for _, timePkg := range dependency.Imports() {
			if timePkg.Path() == "time" {
				timeType := timePkg.Scope().Lookup("Time")
				return timeType != nil && types.Identical(wrapper.Underlying(), timeType.Type().Underlying())
			}
		}
		return false
	}
	return false
}
func subnetPoolListQuerySignature(sig *types.Signature) bool {
	return !sig.Variadic() && sig.Params().Len() == 0 && sig.Results().Len() == 2 && types.Identical(sig.Results().At(0).Type(), types.Typ[types.String]) && isError(sig.Results().At(1).Type())
}

var subnetPoolBodyNativeDeclarations = map[string]string{
	"Get":                            "f2e21c88e17a8ad49060e84dad42244e17c64453bae932aadebe34ad0e7ebc9b",
	"List":                           "b5685bd0990f632e5bc6b3824520185616669b675d28c83168502946acf8d242",
	"ListOpts.ToSubnetPoolListQuery": "c6af8790e25021db5d472bfef7cfe34e4c692bcacce1eaf79c2451e9e4343779",
	"rootURL":                        "4f185db1078eefd1b5f5b38bd1c601bd6fccaa3c20fa99a7b85371ef7f7b8104",
	"resourceURL":                    "ca0246cf0e192c0133b2d43b9c5b577badf51c51344d3c5fc499d01a97ef54c8",
	"listURL":                        "0a55ee851552d789ddd3c12304e6d0d209cae0cfd477d71b138196681486bf9d",
	"getURL":                         "4e7e71e4b36e374a2fc6830f4f621ab3dd904cb2e1fa10b99e43a5590b7b4b36",
	"commonResult.Extract":           "e70226430ded7b75197a644df50e5484cbb78fcfc8174beba0bb7add0dea5310",
	"SubnetPool.UnmarshalJSON":       "7166552c0b959e3559a7f81f10bec4072c454530187c73331faf174efce743ef",
	"SubnetPoolPage.NextPageURL":     "5002bfbc328292413a3aaa785d5fad4a23dcd0835581440ff41497deeb0b7d57",
	"SubnetPoolPage.IsEmpty":         "4843c1ad38084a3f8fa778644a55267a8cdc9f052735cb111f4299e8285a978f",
	"ExtractSubnetPools":             "d55581a7189bdae0d033052889b6db3043ecfb67892fc537809b4cec522de3d8",
	"gophercloud.ExtractNextURL":     "e0ce1875e9f11ea6812ed64aa5f42e095fa53dace0a482f099c05ffb99322dc2",
	"gophercloud.Result.ExtractInto": "a978ec69fee3b64d971a9a749c20845d5419425d1ef304d24e82c118bcf53a57",
	"gophercloud.JSONRFC3339NoZ.UnmarshalJSON": "cfc66b819ff596fb73c3430ca00f08acfe0fc8aa7a3667fe0bbcda77626ceab4",
	"pagination.PageResultFrom":                "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9",
	"pagination.PageResultFromParsed":          "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089",
	"pagination.LinkedPageBase.GetBody":        "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284",
	"pagination.Request":                       "fc14f4b1bc17be1eec2b4d8bdc5773ddaab7290e0f68678eb24e0d2efa69de9b",
	"pagination.NewPager":                      "0ae3e28e02baab6177a09b15c34800d28693195df04a760eacc8706b8054de3b",
	"pagination.Pager.EachPage":                "2242dc6337f973ebe8515e684f9cd8d055c801a52ce4657d08986ee11942c4db",
	"pagination.Pager.fetchNextPage":           "7a785acf0d677bd3fc14c586f72a3a1a5268812568e500e8a9cc11598bbbcc90",
}

func validateSubnetPoolBodyNativeDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) != subnetPoolSDKPath {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited SubnetPool body collection: native model/list schema changed")
	}
	for name, want := range subnetPoolBodyNativeDeclarations {
		got, err := requestDeclarationHash(decls[name])
		if err != nil || got != want {
			return fmt.Errorf("audited SubnetPool body collection: pinned native declaration %s changed", name)
		}
	}
	return nil
}
func emitSubnetPoolBodyRecordAdapter(e *emitter, plan *collectionPlan) {
	emitNeutronBodyRecordAdapter(e, plan, "subnetPoolBodyFilterValue")
}
func emitSubnetPoolBodyFilterList(e *emitter, plan *collectionPlan) {
	emitNeutronBodyFilterList(e, plan, "subnetpools", "subnetpools", "ExtractSubnetPools")
}
