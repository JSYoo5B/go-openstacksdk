package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
)

func networkBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{{key: "availability_zone_hints", member: "AvailabilityZoneHints"}, {key: "availability_zones"}, {key: "created_at", member: "CreatedAt"}, {key: "dns_domain"}, {key: "is_default"}, {key: "mtu"}, {key: "pvlan"}, {key: "qos_policy_id"}, {key: "revision_number", member: "RevisionNumber"}, {key: "segments"}, {key: "subnets", member: "Subnets", aliases: []string{"subnet_ids"}}, {key: "updated_at", member: "UpdatedAt"}, {key: "vlan_qinq", aliases: []string{"is_vlan_qinq"}}, {key: "vlan_transparent", aliases: []string{"is_vlan_transparent"}}}
}
func networkBodyCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	return spec.path == networkSDKPath && spec.model == "Network" && spec.rawRecord && reflect.DeepEqual(spec.fields, networkBodyCollectionFields())
}
func networkBodyPointerDecoder(model *types.Named) bool {
	if !rawBodyNativeDecoder(model) {
		return false
	}
	sig := model.Method(0).Type().(*types.Signature)
	return model.Method(0).Name() == "UnmarshalJSON" && types.Identical(sig.Recv().Type(), types.NewPointer(model))
}
func networkBodyNativeSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "Network" || plan.id != "ID" || plan.idIsURL || plan.name != "Name" || plan.nameQuery != "name" || plan.status != "Status" || plan.statusQuery != "status" || !plan.listQueryBuilder || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || !types.Identical(plan.getIDType, types.Typ[types.String]) || !identityNetworkListSchema(pkg, plan) {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok || !networkBodyPointerDecoder(model) || !rawBodyFieldsMatch(model, "json", map[string][2]string{"ID": {"string", "id"}, "Name": {"string", "name"}, "Description": {"string", "description"}, "AdminStateUp": {"bool", "admin_state_up"}, "Status": {"string", "status"}, "Subnets": {"[]string", "subnets"}, "TenantID": {"string", "tenant_id"}, "UpdatedAt": {"time.Time", "-"}, "CreatedAt": {"time.Time", "-"}, "ProjectID": {"string", "project_id"}, "Shared": {"bool", "shared"}, "AvailabilityZoneHints": {"[]string", "availability_zone_hints"}, "Tags": {"[]string", "tags"}, "RevisionNumber": {"int", "revision_number"}}) || !rawBodyFieldsMatch(plan.listInput, "q", map[string][2]string{"Status": {"string", "status"}, "Name": {"string", "name"}, "Description": {"string", "description"}, "AdminStateUp": {"*bool", "admin_state_up"}, "TenantID": {"string", "tenant_id"}, "ProjectID": {"string", "project_id"}, "Shared": {"*bool", "shared"}, "ID": {"string", "id"}, "Marker": {"string", "marker"}, "Limit": {"int", "limit"}, "SortKey": {"string", "sort_key"}, "SortDir": {"string", "sort_dir"}, "Tags": {"string", "tags"}, "TagsAny": {"string", "tags-any"}, "NotTags": {"string", "not-tags"}, "NotTagsAny": {"string", "not-tags-any"}, "RevisionNumber": {"*int", "revision_number"}}) {
		return false
	}
	list, ok := plan.listInput.(*types.Named)
	if !ok || list.NumMethods() != 1 || list.Method(0).Name() != "ToNetworkListQuery" {
		return false
	}
	query := list.Method(0).Type().(*types.Signature)
	if !types.Identical(query.Recv().Type(), list) || !networkBodyListQuerySignature(query) {
		return false
	}
	object := pkg.Scope().Lookup("ListOptsBuilder")
	if object == nil {
		return false
	}
	builder, ok := object.Type().Underlying().(*types.Interface)
	if !ok || builder.NumEmbeddeds() != 0 || builder.NumMethods() != 1 || builder.Method(0).Name() != "ToNetworkListQuery" || !networkBodyListQuerySignature(builder.Method(0).Type().(*types.Signature)) {
		return false
	}
	page := pkg.Scope().Lookup("NetworkPage").Type().(*types.Named)
	if page.NumMethods() != 3 {
		return false
	}
	for i := 0; i < page.NumMethods(); i++ {
		sig := page.Method(i).Type().(*types.Signature)
		if !types.Identical(sig.Recv().Type(), page) {
			return false
		}
	}
	result := identityGetResult(pkg, plan, 0)
	if result == nil || result.Obj().Name() != "GetResult" || result.NumMethods() != 0 {
		return false
	}
	structure, ok := result.Underlying().(*types.Struct)
	if !ok || structure.NumFields() != 1 || !structure.Field(0).Embedded() || types.TypeString(structure.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != pkg.Path()+".commonResult" {
		return false
	}
	common, ok := structure.Field(0).Type().(*types.Named)
	if !ok || common.NumMethods() != 2 {
		return false
	}
	base, ok := common.Underlying().(*types.Struct)
	if !ok || base.NumFields() != 1 || !base.Field(0).Embedded() || types.TypeString(base.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+".Result" {
		return false
	}
	for i := 0; i < common.NumMethods(); i++ {
		method := common.Method(i)
		if method.Name() != "Extract" && method.Name() != "ExtractInto" {
			return false
		}
		sig := method.Type().(*types.Signature)
		if !types.Identical(sig.Recv().Type(), common) {
			return false
		}
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
		if !ok || !networkBodyPointerDecoder(wrapper) {
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
func networkBodyListQuerySignature(sig *types.Signature) bool {
	return !sig.Variadic() && sig.Params().Len() == 0 && sig.Results().Len() == 2 && types.Identical(sig.Results().At(0).Type(), types.Typ[types.String]) && isError(sig.Results().At(1).Type())
}

var networkBodyNativeDeclarations = map[string]string{"ExtractNetworks": "5a2d1db17e38f0070dfdac235159c3aae4589cf69a4399e2f00a3bdd02a1448f", "ExtractNetworksInto": "808d8179a8b72552591dc81fbb2cd482cef8c07873a77cda7063d5af0fd2f512", "Get": "f2e21c88e17a8ad49060e84dad42244e17c64453bae932aadebe34ad0e7ebc9b", "List": "cc52c5ec8c677374bafc4322ae68c43f0dce73cc84000b434e15267d3b617d20", "ListOpts.ToNetworkListQuery": "786e84233300d77b73a1e7d1b2f13ec4d5a42ac7bd085a49421e6b15f3394c90", "Network.UnmarshalJSON": "f9840328e1d80cc0b32221f0acfa5c98ebe426375dca0985dc2eedf008e1e2ee", "NetworkPage.IsEmpty": "756bddbbc265e3b30dcff213d983b457559b713f7a0299588d0232a2914b1abc", "NetworkPage.NextPageURL": "e6f3f1611b0d2892145f024b0e2fae31b2d29f282a3d24dd25605c1ec4556efc", "NetworkPage.ResourceKey": "443f3b4a977a9746314eddc9b8db71d0269e53854d1b5fd9f067764f6ab060a7", "commonResult.Extract": "f958482bc76477b9059a8c3a4d921dae13668f04f28fb93340ccb9a8709f586b", "commonResult.ExtractInto": "51fd854e39753b80bd255703daeedbf5d4e279e4542dbb226aaa98247134314f", "getURL": "4e7e71e4b36e374a2fc6830f4f621ab3dd904cb2e1fa10b99e43a5590b7b4b36", "gophercloud.ExtractNextURL": "e0ce1875e9f11ea6812ed64aa5f42e095fa53dace0a482f099c05ffb99322dc2", "gophercloud.JSONRFC3339NoZ.UnmarshalJSON": "cfc66b819ff596fb73c3430ca00f08acfe0fc8aa7a3667fe0bbcda77626ceab4", "gophercloud.Result.ExtractInto": "a978ec69fee3b64d971a9a749c20845d5419425d1ef304d24e82c118bcf53a57", "gophercloud.Result.ExtractIntoSlicePtr": "eed0d552518f57bf120ea36fd9a2ef6bc17288ab9be239754a205f9d6130befc", "gophercloud.Result.ExtractIntoStructPtr": "07bafede6846c4d1f86b1b04cc53dc97046cabbe56d5f858bb68115b92f6a625", "gophercloud.Result.extractIntoPtr": "418acd501c229b1cf45b844a5948aa59f06c62c04e0ac99ee2a94f14f0d8d4d2", "listURL": "0a55ee851552d789ddd3c12304e6d0d209cae0cfd477d71b138196681486bf9d", "pagination.LinkedPageBase.GetBody": "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284", "pagination.NewPager": "0ae3e28e02baab6177a09b15c34800d28693195df04a760eacc8706b8054de3b", "pagination.PageResultFrom": "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9", "pagination.PageResultFromParsed": "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089", "pagination.Pager.EachPage": "2242dc6337f973ebe8515e684f9cd8d055c801a52ce4657d08986ee11942c4db", "pagination.Pager.fetchNextPage": "7a785acf0d677bd3fc14c586f72a3a1a5268812568e500e8a9cc11598bbbcc90", "pagination.Request": "fc14f4b1bc17be1eec2b4d8bdc5773ddaab7290e0f68678eb24e0d2efa69de9b", "resourceURL": "313da020f0b09244e553dac107ce73d06de65812ce24c0339473cc8befd6332a", "rootURL": "dfa59922daefee3ab2d5e490b69377ca7a0e58f52ef4e06b59de8837f18fb2c3"}

func validateNetworkBodyNativeDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) != networkSDKPath {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited Network body collection: native model/list schema changed")
	}
	for name, want := range networkBodyNativeDeclarations {
		got, err := requestDeclarationHash(decls[name])
		if err != nil || got != want {
			return fmt.Errorf("audited Network body collection: pinned native declaration %s changed", name)
		}
	}
	return nil
}
func emitNetworkBodyRecordAdapter(e *emitter, plan *collectionPlan) {
	emitNeutronBodyRecordAdapter(e, plan, "networkBodyFilterValue")
}
func emitNetworkBodyFilterList(e *emitter, plan *collectionPlan) {
	emitNeutronBodyFilterList(e, plan, "networks", "networks", "ExtractNetworks")
}
