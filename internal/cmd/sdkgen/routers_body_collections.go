package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
)

func routerBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{{key: "availability_zone_hints", member: "AvailabilityZoneHints"}, {key: "availability_zones"}, {key: "created_at", member: "CreatedAt"}, {key: "enable_ndp_proxy"}, {key: "evpn_vni"}, {key: "external_gateway_info", member: "GatewayInfo"}, {key: "revision", aliases: []string{"revision_number"}}, {key: "routes", member: "Routes"}, {key: "tenant_id", member: "TenantID"}, {key: "updated_at", member: "UpdatedAt"}}
}
func routerBodyCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	return spec.path == routerSDKPath && spec.model == "Router" && spec.rawRecord && reflect.DeepEqual(spec.fields, routerBodyCollectionFields())
}
func routerBodyPointerDecoder(model *types.Named) bool {
	if !rawBodyNativeDecoder(model) {
		return false
	}
	sig := model.Method(0).Type().(*types.Signature)
	return model.Method(0).Name() == "UnmarshalJSON" && types.Identical(sig.Recv().Type(), types.NewPointer(model))
}
func routerBodyNativeSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "Router" || plan.id != "ID" || plan.idIsURL || plan.name != "Name" || plan.nameQuery != "name" || plan.status != "Status" || plan.statusQuery != "status" || !plan.listQueryBuilder || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || !types.Identical(plan.getIDType, types.Typ[types.String]) || !rawBodyNativePage(pkg, plan.model, "RouterPage", "ExtractRouters") {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok || !routerBodyPointerDecoder(model) || !rawBodyFieldsMatch(model, "json", map[string][2]string{"Status": {"string", "status"}, "GatewayInfo": {"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers.GatewayInfo", "external_gateway_info"}, "AdminStateUp": {"bool", "admin_state_up"}, "Distributed": {"bool", "distributed"}, "Name": {"string", "name"}, "Description": {"string", "description"}, "ID": {"string", "id"}, "TenantID": {"string", "tenant_id"}, "ProjectID": {"string", "project_id"}, "Routes": {"[]github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers.Route", "routes"}, "AvailabilityZoneHints": {"[]string", "availability_zone_hints"}, "Tags": {"[]string", "tags"}, "RevisionNumber": {"int", "revision_number"}, "CreatedAt": {"time.Time", "-"}, "UpdatedAt": {"time.Time", "-"}}) || !rawBodyFieldsMatch(plan.listInput, "q", map[string][2]string{"ID": {"string", "id"}, "Name": {"string", "name"}, "Description": {"string", "description"}, "AdminStateUp": {"*bool", "admin_state_up"}, "Distributed": {"*bool", "distributed"}, "Status": {"string", "status"}, "TenantID": {"string", "tenant_id"}, "ProjectID": {"string", "project_id"}, "Limit": {"int", "limit"}, "Marker": {"string", "marker"}, "SortKey": {"string", "sort_key"}, "SortDir": {"string", "sort_dir"}, "Tags": {"string", "tags"}, "TagsAny": {"string", "tags-any"}, "NotTags": {"string", "not-tags"}, "NotTagsAny": {"string", "not-tags-any"}, "RevisionNumber": {"*int", "revision_number"}}) {
		return false
	}

	for _, name := range []string{"GatewayInfo", "ExternalFixedIP", "Route"} {
		object := pkg.Scope().Lookup(name)
		if object == nil {
			return false
		}
		nested, ok := object.Type().(*types.Named)
		if !ok || nested.NumMethods() != 0 {
			return false
		}
		switch name {
		case "GatewayInfo":
			if !rawBodyFieldsMatch(nested, "json", map[string][2]string{"NetworkID": {"string", "network_id,omitempty"}, "EnableSNAT": {"*bool", "enable_snat,omitempty"}, "ExternalFixedIPs": {"[]github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers.ExternalFixedIP", "external_fixed_ips,omitempty"}, "QoSPolicyID": {"string", "qos_policy_id,omitempty"}}) {
				return false
			}
		case "ExternalFixedIP":
			if !rawBodyFieldsMatch(nested, "json", map[string][2]string{"IPAddress": {"string", "ip_address,omitempty"}, "SubnetID": {"string", "subnet_id,omitempty"}}) {
				return false
			}
		case "Route":
			if !rawBodyFieldsMatch(nested, "json", map[string][2]string{"NextHop": {"string", "nexthop"}, "DestinationCIDR": {"string", "destination"}}) {
				return false
			}
		}
	}
	intoObject := pkg.Scope().Lookup("ExtractRoutersInto")
	if intoObject == nil {
		return false
	}
	into, ok := intoObject.(*types.Func)
	if !ok {
		return false
	}
	intoSig := into.Type().(*types.Signature)
	if intoSig.Variadic() || intoSig.Params().Len() != 2 || types.TypeString(intoSig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || !types.Identical(intoSig.Params().At(1).Type(), types.NewInterfaceType(nil, nil).Complete()) || intoSig.Results().Len() != 1 || !isError(intoSig.Results().At(0).Type()) {
		return false
	}
	list, ok := plan.listInput.(*types.Named)
	if !ok || list.NumMethods() != 1 || list.Method(0).Name() != "ToRouterListQuery" {
		return false
	}
	query := list.Method(0).Type().(*types.Signature)
	if !types.Identical(query.Recv().Type(), list) || !routerBodyListQuerySignature(query) {
		return false
	}
	object := pkg.Scope().Lookup("ListOptsBuilder")
	if object == nil {
		return false
	}
	builder, ok := object.Type().Underlying().(*types.Interface)
	if !ok || builder.NumEmbeddeds() != 0 || builder.NumMethods() != 1 || builder.Method(0).Name() != "ToRouterListQuery" || !routerBodyListQuerySignature(builder.Method(0).Type().(*types.Signature)) {
		return false
	}
	page := pkg.Scope().Lookup("RouterPage").Type().(*types.Named)
	if page.NumMethods() != 2 {
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
	if !ok || common.NumMethods() != 1 {
		return false
	}
	base, ok := common.Underlying().(*types.Struct)
	if !ok || base.NumFields() != 1 || !base.Field(0).Embedded() || types.TypeString(base.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+".Result" {
		return false
	}
	for i := 0; i < common.NumMethods(); i++ {
		method := common.Method(i)
		if method.Name() != "Extract" {
			return false
		}
		sig := method.Type().(*types.Signature)
		if !types.Identical(sig.Recv().Type(), common) || sig.Variadic() || sig.Params().Len() != 0 || sig.Results().Len() != 2 || !types.Identical(sig.Results().At(0).Type(), types.NewPointer(model)) || !isError(sig.Results().At(1).Type()) {
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
		if !ok || !routerBodyPointerDecoder(wrapper) {
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
func routerBodyListQuerySignature(sig *types.Signature) bool {
	return !sig.Variadic() && sig.Params().Len() == 0 && sig.Results().Len() == 2 && types.Identical(sig.Results().At(0).Type(), types.Typ[types.String]) && isError(sig.Results().At(1).Type())
}

var routerBodyNativeDeclarations = map[string]string{"ExtractRouters": "f4a7e6f4f0ee7722ff1d297a2fb026d843cce89d137d60dd85214f0a20f29e18", "ExtractRoutersInto": "be02017c665dfc07a48940b7c44d0adc592ab754f39044fb6e0fbe77403a759e", "Get": "c0787d9b62a2cda3caa71d9a5deeaa3b99842417646def58aea2161caf892fb9", "List": "452ffb77d4836fef164e8c731d85511442433e043be2198359a7534d2def0420", "ListOpts.ToRouterListQuery": "16630576a814d8e9f82bd02355bd7367745f3e7062990da4efe898e23b8dc12f", "Router.UnmarshalJSON": "5f6e06c6894b79e08121cd89520019211ab88760dd5247b4bf6a3697a55b5377", "RouterPage.IsEmpty": "1962b3680eb57993f0a30287f025a408eefdd5d11e55d4e28fc683a75612c3ab", "RouterPage.NextPageURL": "4323c0d1e0740e56af094d4f0df9c943acb1f2409f2fab8fb26c6ee93fb71a9c", "commonResult.Extract": "4bb6eaf3e9ca18e71322e50f621501195d9aab13ecca5a1fc7acc7f263ac6df4", "gophercloud.ExtractNextURL": "e0ce1875e9f11ea6812ed64aa5f42e095fa53dace0a482f099c05ffb99322dc2", "gophercloud.JSONRFC3339NoZ.UnmarshalJSON": "cfc66b819ff596fb73c3430ca00f08acfe0fc8aa7a3667fe0bbcda77626ceab4", "gophercloud.Result.ExtractInto": "a978ec69fee3b64d971a9a749c20845d5419425d1ef304d24e82c118bcf53a57", "gophercloud.Result.ExtractIntoSlicePtr": "eed0d552518f57bf120ea36fd9a2ef6bc17288ab9be239754a205f9d6130befc", "gophercloud.Result.extractIntoPtr": "418acd501c229b1cf45b844a5948aa59f06c62c04e0ac99ee2a94f14f0d8d4d2", "pagination.LinkedPageBase.GetBody": "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284", "pagination.NewPager": "0ae3e28e02baab6177a09b15c34800d28693195df04a760eacc8706b8054de3b", "pagination.PageResultFrom": "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9", "pagination.PageResultFromParsed": "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089", "pagination.Pager.EachPage": "2242dc6337f973ebe8515e684f9cd8d055c801a52ce4657d08986ee11942c4db", "pagination.Pager.fetchNextPage": "7a785acf0d677bd3fc14c586f72a3a1a5268812568e500e8a9cc11598bbbcc90", "pagination.Request": "fc14f4b1bc17be1eec2b4d8bdc5773ddaab7290e0f68678eb24e0d2efa69de9b", "resourceURL": "ca0246cf0e192c0133b2d43b9c5b577badf51c51344d3c5fc499d01a97ef54c8", "rootURL": "4f185db1078eefd1b5f5b38bd1c601bd6fccaa3c20fa99a7b85371ef7f7b8104"}

func validateRouterBodyNativeDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) != routerSDKPath {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited Router body collection: native model/list schema changed")
	}
	for name, want := range routerBodyNativeDeclarations {
		got, err := requestDeclarationHash(decls[name])
		if err != nil || got != want {
			return fmt.Errorf("audited Router body collection: pinned native declaration %s changed", name)
		}
	}
	return nil
}
func emitRouterBodyRecordAdapter(e *emitter, plan *collectionPlan) {
	emitNeutronBodyRecordAdapter(e, plan, "routerBodyFilterValue")
}
func emitRouterBodyFilterList(e *emitter, plan *collectionPlan) {
	emitNeutronBodyFilterList(e, plan, "routers", "routers", "ExtractRouters")
}
