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

const addressGroupSDKPath = "network/v2/extensions/security/addressgroups"

func addressGroupBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{{key: "addresses", member: "Addresses"}, {key: "id", member: "ID"}, {key: "tenant_id"}}
}

func addressGroupBodyCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	return spec.path == addressGroupSDKPath && spec.model == "AddressGroup" && spec.rawRecord && reflect.DeepEqual(spec.fields, addressGroupBodyCollectionFields())
}

func addressGroupBodyNativeSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "AddressGroup" || plan.id != "ID" || plan.idIsURL || plan.name != "Name" || plan.nameQuery != "name" || plan.status != "" || plan.statusQuery != "" || !plan.listQueryBuilder || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || !types.Identical(plan.getIDType, types.Typ[types.String]) {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok || model.NumMethods() != 0 || !rawBodyFieldsMatch(model, "json", map[string][2]string{"ID": {"string", "id"}, "Name": {"string", "name"}, "Description": {"string", "description"}, "ProjectID": {"string", "project_id"}, "Addresses": {"[]string", "addresses"}}) || !rawBodyFieldsMatch(plan.listInput, "q", map[string][2]string{"ID": {"string", "id"}, "Name": {"string", "name"}, "Description": {"string", "description"}, "ProjectID": {"string", "project_id"}, "Addresses": {"[]string", "addresses"}, "Limit": {"int", "limit"}, "Marker": {"string", "marker"}, "SortKey": {"string", "sort_key"}, "SortDir": {"string", "sort_dir"}}) {
		return false
	}
	list, ok := plan.listInput.(*types.Named)
	if !ok || list.NumMethods() != 1 || list.Method(0).Name() != "ToAddressGroupListQuery" {
		return false
	}
	query := list.Method(0).Type().(*types.Signature)
	if !types.Identical(query.Recv().Type(), list) || query.Variadic() || query.Params().Len() != 0 || query.Results().Len() != 2 || !types.Identical(query.Results().At(0).Type(), types.Typ[types.String]) || !isError(query.Results().At(1).Type()) {
		return false
	}
	if !rawBodyNativePage(pkg, model, "AddressGroupPage", "ExtractGroups") || identityGetResult(pkg, plan, 0) == nil {
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

var addressGroupBodyNativeDeclarations = map[string]string{
	"Get":                               "c0787d9b62a2cda3caa71d9a5deeaa3b99842417646def58aea2161caf892fb9",
	"List":                              "5df8dc6e3d372ae3b098dcf493fb5a64c82ba6c1dfdd25ca0671c132ded1251b",
	"ListOpts.ToAddressGroupListQuery":  "9926b4a40d1b9c796d9a344a5935194b52681701b31be5eed66df3faa95be487",
	"rootURL":                           "5a4bf90be880f0165342169abba24c6938989c185637bae8e4c5092f33ac9f3b",
	"resourceURL":                       "ff31c93c543d54044b2c72a7219311759861582c91db4787e04a5f2d3dfc060c",
	"AddressGroupPage.NextPageURL":      "db55578ce9b76a89b003f2907d3491ba56684dafdac9c1e6fe343b5460ae92cd",
	"AddressGroupPage.IsEmpty":          "ba7bc94d31fad840c0a0c2437508ed26c1782391692a5852ef3ae2d3ed7c8054",
	"ExtractGroups":                     "d478953739195b0391c1228d9a5b5576638a3a96fbfb93fb41042920a47cfe30",
	"commonResult.Extract":              "e6cfb8e6b6b251cfcdb21e8017c976fa7cccd580d3cc8bd3319f5e1d5ee818db",
	"pagination.PageResultFrom":         "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9",
	"pagination.PageResultFromParsed":   "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089",
	"pagination.LinkedPageBase.GetBody": "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284",
	"gophercloud.ExtractNextURL":        "e0ce1875e9f11ea6812ed64aa5f42e095fa53dace0a482f099c05ffb99322dc2",
	"gophercloud.Result.ExtractInto":    "a978ec69fee3b64d971a9a749c20845d5419425d1ef304d24e82c118bcf53a57",
}

func validateAddressGroupBodyNativeDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) != addressGroupSDKPath {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited AddressGroup body collection: native model/list schema changed")
	}
	for name, want := range addressGroupBodyNativeDeclarations {
		got, err := requestDeclarationHash(decls[name])
		if err != nil || got != want {
			return fmt.Errorf("audited AddressGroup body collection: pinned native declaration %s changed", name)
		}
	}
	return nil
}

// The audited Neutron raw lanes use their reviewed root-package helpers.
// Limit source loading to their exact targets and extraction declarations.
func (g *generator) bodyRecordRootDeclarations(path string) (map[string]*ast.FuncDecl, error) {
	wanted := map[string]bool{"ExtractNextURL": true, "Result.ExtractInto": true}
	label := "audited AddressGroup body collection"
	switch sdkPath(path) {
	case addressGroupSDKPath:
	case subnetPoolSDKPath:
		label = "audited SubnetPool body collection"
		wanted["JSONRFC3339NoZ.UnmarshalJSON"] = true
	case qosPolicySDKPath:
		label = "audited QoSPolicy body collection"
		wanted["Result.ExtractIntoSlicePtr"] = true
		wanted["Result.extractIntoPtr"] = true
	default:
		return nil, nil
	}
	source, ok := g.meta[upstreamModule]
	if !ok || source.Dir == "" || len(source.GoFiles) == 0 {
		return nil, fmt.Errorf("%s: native extraction dependency metadata missing", label)
	}
	result := map[string]*ast.FuncDecl{}
	noZConstants := 0
	for _, name := range source.GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(source.Dir, name), nil, 0)
		if err != nil {
			return nil, fmt.Errorf("%s: native extraction dependency: %w", label, err)
		}
		if sdkPath(path) == subnetPoolSDKPath {
			if value, found := identitySourceConstants(file)["RFC3339NoZ"]; found {
				if value != "2006-01-02T15:04:05" {
					return nil, fmt.Errorf("%s: native RFC3339NoZ constant changed", label)
				}
				noZConstants++
			}
		}
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || !wanted[identityDeclarationKey(fn)] {
				continue
			}
			key := "gophercloud." + identityDeclarationKey(fn)
			if result[key] != nil {
				return nil, fmt.Errorf("%s: duplicate extraction declaration %s", label, key)
			}
			result[key] = fn
		}
	}
	if len(result) != len(wanted) {
		return nil, fmt.Errorf("%s: native extraction declarations missing", label)
	}
	if sdkPath(path) == subnetPoolSDKPath && noZConstants != 1 {
		return nil, fmt.Errorf("%s: native RFC3339NoZ constant missing or duplicated", label)
	}
	return result, nil
}

// Retain the AddressGroup-only helper boundary for existing callers/tests.
func (g *generator) addressGroupBodyRootDeclarations(path string) (map[string]*ast.FuncDecl, error) {
	if sdkPath(path) != addressGroupSDKPath {
		return nil, nil
	}
	return g.bodyRecordRootDeclarations(path)
}

func emitAddressGroupBodyRecordAdapter(e *emitter, plan *collectionPlan) {
	emitNeutronBodyRecordAdapter(e, plan, "addressGroupBodyFilterValue")
}

func emitNeutronBodyRecordAdapter(e *emitter, plan *collectionPlan, selector string) {
	e.use("gophercloudsdk/request")
	e.printf("},\nBodyFilterRecordValue:func(record *resource.BodyRecord[%s],key string)(json.RawMessage,error){return %s(record,key)},\n", plan.modelName, selector)
	e.printf("IterateBodyControlled:func(ctx context.Context,q url.Values,control resource.ListControl)iter.Seq2[*resource.BodyRecord[%s],error]{\n", plan.modelName)
	e.printf("options:=[]ListOption{func(config *request.Config[ListOpts])error{config.Query=make(url.Values,len(q));for key,values:=range q{config.Query[key]=append([]string(nil),values...)};return nil}}\nreturn a.listBodyWithControl(ctx,control,options...)\n},\n")
}

func emitAddressGroupBodyFilterList(e *emitter, plan *collectionPlan) {
	emitNeutronBodyFilterList(e, plan, "addressgroups", "address_groups", "ExtractGroups")
}

func emitNeutronBodyFilterList(e *emitter, plan *collectionPlan, kind, envelope, extractor string) {
	e.use("gophercloudsdk/request")
	e.use(upstreamModule + "/pagination")
	e.use(e.pkg.Path())
	e.printf("func(a *API)listBodyWithControl(ctx context.Context,control resource.ListControl,options ...ListOption)iter.Seq2[*resource.BodyRecord[%s],error]{\nvar opts ListOpts\ncfg,err:=request.Apply(opts,options...)\n", plan.modelName)
	e.printf("if err==nil{err=request.ValidateCapabilities(cfg,false,true,false)}\nif err!=nil{err=request.Wrap(\"List\",%q,err);return func(yield func(*resource.BodyRecord[%s],error)bool){yield(nil,err)}}\n", kind, plan.modelName)
	e.printf("_opts:=listOptsBuilder{base:cfg.Options,config:cfg}\nreturn resource.BodyStreamWithControl(ctx,upstream.List(a.client,_opts),func(page pagination.Page)([]%s,error){values,err:=upstream.%s(page);return []%s(values),err},%q,control)\n}\n", plan.modelName, extractor, plan.modelName, envelope)
}
