package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
)

func trunkBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{{key: "id", member: "ID"}, {key: "tenant_id", member: "TenantID"}}
}
func trunkBodyCollectionMetadataValid(s bodyFilterCollectionSpec) bool {
	return s.path == trunkSDKPath && s.model == "Trunk" && s.rawRecord && reflect.DeepEqual(s.fields, trunkBodyCollectionFields())
}
func trunkBodyNativeSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "Trunk" || plan.id != "ID" || plan.idIsURL || plan.name != "Name" || plan.nameQuery != "name" || plan.status != "Status" || plan.statusQuery != "status" || !plan.listQueryBuilder || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || !types.Identical(plan.getIDType, types.Typ[types.String]) || !identityPoolTrunkSchema(pkg, plan) {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok || model.NumMethods() != 0 || !rawBodyFieldsMatch(model, "json", map[string][2]string{"Status": {"string", "status"}, "Subports": {"[]github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks.Subport", "sub_ports"}, "Name": {"string", "name,omitempty"}, "AdminStateUp": {"bool", "admin_state_up,omitempty"}, "ProjectID": {"string", "project_id"}, "TenantID": {"string", "tenant_id"}, "CreatedAt": {"time.Time", "created_at"}, "UpdatedAt": {"time.Time", "updated_at"}, "RevisionNumber": {"int", "revision_number"}, "PortID": {"string", "port_id"}, "ID": {"string", "id"}, "Description": {"string", "description"}, "Tags": {"[]string", "tags,omitempty"}}) || !rawBodyFieldsMatch(plan.listInput, "q", map[string][2]string{"AdminStateUp": {"*bool", "admin_state_up"}, "Description": {"string", "description"}, "ID": {"string", "id"}, "Name": {"string", "name"}, "PortID": {"string", "port_id"}, "Status": {"string", "status"}, "TenantID": {"string", "tenant_id"}, "ProjectID": {"string", "project_id"}, "SortDir": {"string", "sort_dir"}, "SortKey": {"string", "sort_key"}, "Tags": {"string", "tags"}, "TagsAny": {"string", "tags-any"}, "NotTags": {"string", "not-tags"}, "NotTagsAny": {"string", "not-tags-any"}, "RevisionNumber": {"string", "revision_number"}}) {
		return false
	}
	sub := pkg.Scope().Lookup("Subport")
	if sub == nil {
		return false
	}
	nested, ok := sub.Type().(*types.Named)
	if !ok || nested.NumMethods() != 0 || !rawBodyFieldsMatch(nested, "json", map[string][2]string{"SegmentationID": {"int", "segmentation_id"}, "SegmentationType": {"string", "segmentation_type"}, "PortID": {"string", "port_id"}}) {
		return false
	}
	fields := nested.Underlying().(*types.Struct)
	for i := 0; i < fields.NumFields(); i++ {
		if reflect.StructTag(fields.Tag(i)).Get("required") != "true" {
			return false
		}
	}
	list, ok := plan.listInput.(*types.Named)
	if !ok || list.NumMethods() != 1 || list.Method(0).Name() != "ToTrunkListQuery" {
		return false
	}
	qs := list.Method(0).Type().(*types.Signature)
	if !types.Identical(qs.Recv().Type(), list) || !routerBodyListQuerySignature(qs) {
		return false
	}
	obj := pkg.Scope().Lookup("ListOptsBuilder")
	if obj == nil {
		return false
	}
	builder, ok := obj.Type().Underlying().(*types.Interface)
	if !ok || builder.NumEmbeddeds() != 0 || builder.NumMethods() != 1 || builder.Method(0).Name() != "ToTrunkListQuery" || !routerBodyListQuerySignature(builder.Method(0).Type().(*types.Signature)) {
		return false
	}
	pageObj := pkg.Scope().Lookup("TrunkPage")
	if pageObj == nil {
		return false
	}
	page, ok := pageObj.Type().(*types.Named)
	if !ok || page.NumMethods() != 1 || page.Method(0).Name() != "IsEmpty" {
		return false
	}
	sig := page.Method(0).Type().(*types.Signature)
	if !types.Identical(sig.Recv().Type(), page) {
		return false
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
	return true
}

var trunkBodyNativeDeclarations = map[string]string{"ExtractTrunks": "95034dff77e7308bcd4c851032b735581b3c3e4e523f1b354b7255a76c95e6be", "Get": "f2e21c88e17a8ad49060e84dad42244e17c64453bae932aadebe34ad0e7ebc9b", "List": "9fcac9fb5387b699ef747b96fd8b049b6319d15e9b22ab1f0d28ae9fbbc890cc", "ListOpts.ToTrunkListQuery": "46017753bc904ae4fe4aa007cea675e1fd1378173905292c61f5cc32bd194920", "TrunkPage.IsEmpty": "594c7f1c9b29430d07dccfd9c819a288c71d3c1835075bfd1d7dcb133618121d", "commonResult.Extract": "a86967d1e97260072f2f9e8a2c1ff10644a529f65912852fe0781206d1a9c332", "getURL": "4e7e71e4b36e374a2fc6830f4f621ab3dd904cb2e1fa10b99e43a5590b7b4b36", "gophercloud.BuildQueryString": "b5f912c2272dc58902c4ef0f777323c734d311dea0202a036604de3939622598", "gophercloud.Result.ExtractInto": "a978ec69fee3b64d971a9a749c20845d5419425d1ef304d24e82c118bcf53a57", "listURL": "0a55ee851552d789ddd3c12304e6d0d209cae0cfd477d71b138196681486bf9d", "pagination.LinkedPageBase.GetBody": "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284", "pagination.LinkedPageBase.NextPageURL": "fa8678035238acb855e2d60896aa155d50de519d8545830ed52220490a02ad4e", "pagination.NewPager": "0ae3e28e02baab6177a09b15c34800d28693195df04a760eacc8706b8054de3b", "pagination.PageResultFrom": "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9", "pagination.PageResultFromParsed": "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089", "pagination.Pager.EachPage": "2242dc6337f973ebe8515e684f9cd8d055c801a52ce4657d08986ee11942c4db", "pagination.Pager.fetchNextPage": "7a785acf0d677bd3fc14c586f72a3a1a5268812568e500e8a9cc11598bbbcc90", "pagination.Request": "fc14f4b1bc17be1eec2b4d8bdc5773ddaab7290e0f68678eb24e0d2efa69de9b", "resourceURL": "ca0246cf0e192c0133b2d43b9c5b577badf51c51344d3c5fc499d01a97ef54c8", "rootURL": "4f185db1078eefd1b5f5b38bd1c601bd6fccaa3c20fa99a7b85371ef7f7b8104"}

func validateTrunkBodyNativeDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) != trunkSDKPath {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited Trunk body collection: native model/list schema changed")
	}
	for name, want := range trunkBodyNativeDeclarations {
		got, err := requestDeclarationHash(decls[name])
		if err != nil || got != want {
			return fmt.Errorf("audited Trunk body collection: pinned native declaration %s changed", name)
		}
	}
	return nil
}
func (g *generator) trunkBodyLeafDeclarations(path string) error {
	if sdkPath(path) != trunkSDKPath {
		return nil
	}
	m, ok := g.meta[path]
	if !ok || m.Dir == "" || len(m.GoFiles) == 0 {
		return fmt.Errorf("audited Trunk body collection: native leaf metadata missing")
	}
	seen := map[string]bool{}
	constants := 0
	for _, name := range m.GoFiles {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(m.Dir, name), nil, 0)
		if err != nil {
			return err
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok {
				key := identityDeclarationKey(fn)
				if _, want := trunkBodyNativeDeclarations[key]; want {
					if seen[key] {
						return fmt.Errorf("audited Trunk body collection: duplicate native declaration %s", key)
					}
					seen[key] = true
				}
			}
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, v := range gd.Specs {
				vs, ok := v.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, n := range vs.Names {
					if n.Name == "resourcePath" {
						constants++
						if identitySourceConstants(f)[n.Name] != "trunks" {
							return fmt.Errorf("audited Trunk body collection: native resourcePath constant changed")
						}
					}
				}
			}
		}
	}
	if constants != 1 {
		return fmt.Errorf("audited Trunk body collection: native resourcePath missing or duplicated")
	}
	return nil
}
func emitTrunkBodyRecordAdapter(e *emitter, plan *collectionPlan) {
	emitNeutronBodyRecordAdapter(e, plan, "trunkBodyFilterValue")
}
func emitTrunkBodyFilterList(e *emitter, plan *collectionPlan) {
	emitNeutronBodyFilterList(e, plan, "trunks", "trunks", "ExtractTrunks")
}
