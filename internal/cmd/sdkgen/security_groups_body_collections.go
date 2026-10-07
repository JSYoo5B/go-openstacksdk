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

func securityGroupBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{{key: "created_at", member: "CreatedAt"}, {key: "security_group_rules", member: "Rules"}, {key: "updated_at", member: "UpdatedAt"}}
}
func securityGroupBodyCollectionMetadataValid(s bodyFilterCollectionSpec) bool {
	return s.path == securityGroupSDKPath && s.model == "SecGroup" && s.rawRecord && reflect.DeepEqual(s.fields, securityGroupBodyCollectionFields())
}
func securityGroupBodyPointerDecoder(n *types.Named) bool {
	if !rawBodyNativeDecoder(n) {
		return false
	}
	sig := n.Method(0).Type().(*types.Signature)
	return types.Identical(sig.Recv().Type(), types.NewPointer(n))
}
func securityGroupBodyNativeSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "SecGroup" || plan.id != "ID" || plan.idIsURL || plan.name != "Name" || plan.nameQuery != "name" || plan.status != "" || plan.statusQuery != "" || plan.listQueryBuilder || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || !types.Identical(plan.getIDType, types.Typ[types.String]) || !rawBodyNativePage(pkg, plan.model, "SecGroupPage", "ExtractGroups") {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok || !securityGroupBodyPointerDecoder(model) || !rawBodyFieldsMatch(model, "json", map[string][2]string{"ID": {"string", ""}, "Name": {"string", ""}, "Description": {"string", ""}, "Rules": {"[]github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules.SecGroupRule", "security_group_rules"}, "Stateful": {"bool", "stateful"}, "TenantID": {"string", "tenant_id"}, "UpdatedAt": {"time.Time", "-"}, "CreatedAt": {"time.Time", "-"}, "ProjectID": {"string", "project_id"}, "Tags": {"[]string", "tags"}, "RevisionNumber": {"int", "revision_number"}}) || !rawBodyFieldsMatch(plan.listInput, "q", map[string][2]string{"ID": {"string", "id"}, "Name": {"string", "name"}, "Description": {"string", "description"}, "Stateful": {"*bool", "stateful"}, "TenantID": {"string", "tenant_id"}, "ProjectID": {"string", "project_id"}, "Limit": {"int", "limit"}, "Marker": {"string", "marker"}, "SortKey": {"string", "sort_key"}, "SortDir": {"string", "sort_dir"}, "Tags": {"string", "tags"}, "TagsAny": {"string", "tags-any"}, "NotTags": {"string", "not-tags"}, "NotTagsAny": {"string", "not-tags-any"}, "RevisionNumber": {"*int", "revision_number"}}) {
		return false
	}
	list, ok := plan.listInput.(*types.Named)
	if !ok || list.NumMethods() != 0 || pkg.Scope().Lookup("ListOptsBuilder") != nil {
		return false
	}
	sig := plan.lister.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 2 || clientParam(sig) != 0 || !types.Identical(sig.Params().At(1).Type(), list) || sig.Results().Len() != 1 || !isPager(sig.Results().At(0).Type()) {
		return false
	}
	var rule *types.Named
	for _, dep := range pkg.Imports() {
		if dep.Path() == securityGroupRulesNativePath {
			obj := dep.Scope().Lookup("SecGroupRule")
			if obj != nil {
				rule, _ = obj.Type().(*types.Named)
			}
		}
	}
	if rule == nil || !securityGroupBodyPointerDecoder(rule) || !rawBodyFieldsMatch(rule, "json", map[string][2]string{"ID": {"string", ""}, "Direction": {"string", ""}, "Description": {"string", "description"}, "EtherType": {"string", "ethertype"}, "SecGroupID": {"string", "security_group_id"}, "PortRangeMin": {"int", "port_range_min"}, "PortRangeMax": {"int", "port_range_max"}, "Protocol": {"string", ""}, "RemoteAddressGroupID": {"string", "remote_address_group_id"}, "RemoteGroupID": {"string", "remote_group_id"}, "RemoteIPPrefix": {"string", "remote_ip_prefix"}, "TenantID": {"string", "tenant_id"}, "ProjectID": {"string", "project_id"}, "RevisionNumber": {"int", "revision_number"}, "CreatedAt": {"time.Time", "-"}, "UpdatedAt": {"time.Time", "-"}}) {
		return false
	}
	page := pkg.Scope().Lookup("SecGroupPage").Type().(*types.Named)
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
		if !ok || !securityGroupBodyPointerDecoder(wrapper) {
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

// The union also guards the unchanged concrete typed List, whose query builder
// is not invoked by either SDK-owned full-query iterator.
var securityGroupBodyNativeDeclarations = map[string]string{"ExtractGroups": "e42007fd330615fa5e88a454260fee2cfd8a1c2b604d42bd4dc4ff7db325d423", "Get": "c0787d9b62a2cda3caa71d9a5deeaa3b99842417646def58aea2161caf892fb9", "List": "5b43132cd8886b599837c8ff9268bef10f152cd644d82774471fd964d568b2e1", "SecGroup.UnmarshalJSON": "7f81c1ad2bc0435c683b7d3033de0bd391d6db90ede9478900b89ee79700f7e6", "SecGroupPage.IsEmpty": "06143a32b9aef0ab4b8bb63239b4a05a0c38bdfcf863fbcb7c9d92a3a6423f2b", "SecGroupPage.NextPageURL": "49a58e1cf6d83daa67e619f1bcd26a55428baba575a5909f51541bee417bfd6b", "commonResult.Extract": "0593db713468ed605d0dbfce265a3003901cedcf742051a00a3e1f385ab13df2", "gophercloud.BuildQueryString": "b5f912c2272dc58902c4ef0f777323c734d311dea0202a036604de3939622598", "gophercloud.ExtractNextURL": "e0ce1875e9f11ea6812ed64aa5f42e095fa53dace0a482f099c05ffb99322dc2", "gophercloud.JSONRFC3339NoZ.UnmarshalJSON": "cfc66b819ff596fb73c3430ca00f08acfe0fc8aa7a3667fe0bbcda77626ceab4", "gophercloud.Result.ExtractInto": "a978ec69fee3b64d971a9a749c20845d5419425d1ef304d24e82c118bcf53a57", "pagination.LinkedPageBase.GetBody": "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284", "pagination.NewPager": "0ae3e28e02baab6177a09b15c34800d28693195df04a760eacc8706b8054de3b", "pagination.PageResultFrom": "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9", "pagination.PageResultFromParsed": "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089", "pagination.Pager.EachPage": "2242dc6337f973ebe8515e684f9cd8d055c801a52ce4657d08986ee11942c4db", "pagination.Pager.fetchNextPage": "7a785acf0d677bd3fc14c586f72a3a1a5268812568e500e8a9cc11598bbbcc90", "pagination.Request": "fc14f4b1bc17be1eec2b4d8bdc5773ddaab7290e0f68678eb24e0d2efa69de9b", "resourceURL": "ff31c93c543d54044b2c72a7219311759861582c91db4787e04a5f2d3dfc060c", "rootURL": "5a4bf90be880f0165342169abba24c6938989c185637bae8e4c5092f33ac9f3b", "rules.SecGroupRule.UnmarshalJSON": "1a1279b8abd4d55de8c21edf1714b11ee682a4558eea597a439e8d6c6e933aa5"}

func validateSecurityGroupBodyNativeDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) != securityGroupSDKPath {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited SecurityGroup body collection: native model/list schema changed")
	}
	for name, want := range securityGroupBodyNativeDeclarations {
		got, err := requestDeclarationHash(decls[name])
		if err != nil || got != want {
			return fmt.Errorf("audited SecurityGroup body collection: pinned native declaration %s changed", name)
		}
	}
	return nil
}

// Only the nested rule decoder participates in group row extraction. Rule CRUD
// functions and its own pager remain outside this exact dependency boundary.
func (g *generator) securityGroupBodyRuleDeclarations(path string) (map[string]*ast.FuncDecl, error) {
	if sdkPath(path) != securityGroupSDKPath {
		return nil, nil
	}
	result := map[string]*ast.FuncDecl{}
	for _, entry := range []struct{ path, prefix string }{{path, ""}, {securityGroupRulesNativePath, "rules."}} {
		m, ok := g.meta[entry.path]
		if !ok || m.Dir == "" || len(m.GoFiles) == 0 {
			return nil, fmt.Errorf("audited SecurityGroup body collection: native rule/leaf metadata missing")
		}
		seen := map[string]bool{}
		rootConstants := 0
		for _, name := range m.GoFiles {
			f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(m.Dir, name), nil, 0)
			if err != nil {
				return nil, err
			}
			for _, d := range f.Decls {
				if fn, ok := d.(*ast.FuncDecl); ok {
					key := entry.prefix + identityDeclarationKey(fn)
					if _, want := securityGroupBodyNativeDeclarations[key]; want {
						if seen[key] {
							return nil, fmt.Errorf("audited SecurityGroup body collection: duplicate native declaration %s", key)
						}
						seen[key] = true
						if entry.prefix != "" {
							result[key] = fn
						}
					}
				}
				if entry.prefix != "" {
					continue
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
						if n.Name == "rootPath" {
							rootConstants++
							if identitySourceConstants(f)[n.Name] != "security-groups" {
								return nil, fmt.Errorf("audited SecurityGroup body collection: native rootPath constant changed")
							}
						}
					}
				}
			}
		}
		if entry.prefix == "" && rootConstants != 1 {
			return nil, fmt.Errorf("audited SecurityGroup body collection: native rootPath constant missing or duplicated")
		}
	}
	if len(result) != 1 {
		return nil, fmt.Errorf("audited SecurityGroup body collection: native rule decoder missing")
	}
	return result, nil
}
func emitSecurityGroupBodyRecordAdapter(e *emitter, plan *collectionPlan) {
	emitNeutronBodyRecordAdapter(e, plan, "securityGroupBodyFilterValue")
}
func emitSecurityGroupBodyFilterList(e *emitter, plan *collectionPlan) {
	e.use("github.com/JSYoo5B/gophercloudsdk/request")
	e.use("github.com/JSYoo5B/gophercloudsdk/internal/nativefind")
	e.printf("func(a *API)listBodyWithControl(ctx context.Context,control resource.ListControl,options ...ListOption)iter.Seq2[*resource.BodyRecord[%s],error]{\nvar opts ListOpts\ncfg,err:=request.Apply(opts,options...)\n", plan.modelName)
	e.printf("if err==nil{err=request.ValidateCapabilities(cfg,false,true,false)}\nif err!=nil{err=request.Wrap(\"List\",\"groups\",err);return func(yield func(*resource.BodyRecord[%s],error)bool){yield(nil,err)}}\n", plan.modelName)
	e.printf("return nativefind.IterateSecurityGroupBodies(ctx,a.RawClient(),cfg.Query,control)\n}\n")
}
