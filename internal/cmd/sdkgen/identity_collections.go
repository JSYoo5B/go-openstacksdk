package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// These collections have an audited string member route and a name-capable
// native pager. Identity fallback is an opt-in common policy, not evidence for
// every Python proxy option or every generated Get/List binding.
type identityCollectionSpec struct {
	path, model, getter, lister string
	parents                     int
	getSegments                 []string
	getCodes                    []int
	missingListKey              string
	missingListValue            string
	noNameQuery                 bool
	listDefaultKey              string
	listDefaultValue            string
	extraSpecs                  bool
	rawListIterator             string
}

var identityCollectionSpecs = []identityCollectionSpec{
	{path: "compute/v2/servers", model: "Server", getter: "Get", lister: "List", getSegments: []string{"servers", "$id"}, getCodes: []int{200, 203}},
	{path: "blockstorage/v3/volumes", model: "Volume", getter: "Get", lister: "List", getSegments: []string{"volumes", "$id"}, getCodes: []int{200}},
	{path: "network/v2/ports", model: "Port", getter: "Get", lister: "List", getSegments: []string{"ports", "$id"}, getCodes: []int{200}},
	{path: "network/v2/networks", model: "Network", getter: "Get", lister: "List", getSegments: []string{"networks", "$id"}, getCodes: []int{200}},
	{path: "network/v2/subnets", model: "Subnet", getter: "Get", lister: "List", getSegments: []string{"subnets", "$id"}, getCodes: []int{200}},
	{path: "identity/v3/projects", model: "Project", getter: "Get", lister: "List", getSegments: []string{"projects", "$id"}, getCodes: []int{200}},
	{path: "identity/v3/users", model: "User", getter: "Get", lister: "List", getSegments: []string{"users", "$id"}, getCodes: []int{200}},
	{path: "identity/v3/groups", model: "Group", getter: "Get", lister: "List", getSegments: []string{"groups", "$id"}, getCodes: []int{200}},
	{path: "identity/v3/domains", model: "Domain", getter: "Get", lister: "List", getSegments: []string{"domains", "$id"}, getCodes: []int{200}},
	{path: "identity/v3/roles", model: "Role", getter: "Get", lister: "List", getSegments: []string{"roles", "$id"}, getCodes: []int{200}},
	{path: "dns/v2/recordsets", model: "RecordSet", getter: "Get", lister: "ListByZone", parents: 1, getSegments: []string{"zones", "$parent", "recordsets", "$id"}, getCodes: []int{200}},
	{path: "loadbalancer/v2/pools", model: "Member", getter: "GetMember", lister: "ListMembers", parents: 1, getSegments: []string{"lbaas", "pools", "$parent", "members", "$id"}, getCodes: []int{200}},
	{path: "image/v2/images", model: "Image", getter: "Get", lister: "List", getSegments: []string{"images", "$id"}, getCodes: []int{200}, missingListKey: "os_hidden", missingListValue: "true"},
	{path: "compute/v2/flavors", model: "Flavor", getter: "Get", lister: "ListDetail", getSegments: []string{"flavors", "$id"}, getCodes: []int{200}, noNameQuery: true, listDefaultKey: "is_public", listDefaultValue: "None", extraSpecs: true},
	{path: "network/v2/extensions/layer3/routers", model: "Router", getter: "Get", lister: "List", getSegments: []string{"routers", "$id"}, getCodes: []int{200}},
	{path: "network/v2/extensions/security/groups", model: "SecGroup", getter: "Get", lister: "List", getSegments: []string{"security-groups", "$id"}, getCodes: []int{200}, rawListIterator: "IterateSecurityGroups"},
	{path: "network/v2/extensions/subnetpools", model: "SubnetPool", getter: "Get", lister: "List", getSegments: []string{"subnetpools", "$id"}, getCodes: []int{200}},
	{path: "network/v2/extensions/trunks", model: "Trunk", getter: "Get", lister: "List", getSegments: []string{"trunks", "$id"}, getCodes: []int{200}},
	{path: "network/v2/extensions/qos/policies", model: "Policy", getter: "Get", lister: "List", getSegments: []string{"qos", "policies", "$id"}, getCodes: []int{200}},
	{path: "network/v2/extensions/security/addressgroups", model: "AddressGroup", getter: "Get", lister: "List", getSegments: []string{"address-groups", "$id"}, getCodes: []int{200}},
}

func identityCollectionEnabled(pkg *types.Package, plan *collectionPlan, parents int) bool {
	_, ok := identityCollectionContract(pkg, plan, parents)
	return ok
}

func identityCollectionContract(pkg *types.Package, plan *collectionPlan, parents int) (identityCollectionSpec, bool) {
	if plan == nil || plan.getter == nil || plan.lister == nil || plan.id != "ID" || plan.name != "Name" || plan.idIsURL || !isString(plan.getIDType) || plan.listInput == nil {
		return identityCollectionSpec{}, false
	}
	identifier, _, _ := types.LookupFieldOrMethod(plan.model, true, nil, plan.id)
	field, ok := identifier.(*types.Var)
	if !ok || !field.IsField() || !isString(field.Type()) {
		return identityCollectionSpec{}, false
	}
	name, _, _ := types.LookupFieldOrMethod(plan.model, true, nil, plan.name)
	nameField, ok := name.(*types.Var)
	if !ok || !nameField.IsField() || !isString(nameField.Type()) {
		return identityCollectionSpec{}, false
	}
	for _, spec := range identityCollectionSpecs {
		if sdkPath(pkg.Path()) == spec.path && plan.modelName == spec.model && plan.getter.Name() == spec.getter && plan.lister.Name() == spec.lister && parents == spec.parents && identityGetResult(pkg, plan, parents) != nil {
			if !identityMissingListMetadataValid(spec) || !identityFlavorMetadataValid(spec) || !identityRawListMetadataValid(spec) || plan.nameQuery != identityExpectedNameQuery(spec) {
				return identityCollectionSpec{}, false
			}
			if plan.listQueryBuilder != (spec.rawListIterator == "") {
				return identityCollectionSpec{}, false
			}
			return spec, true
		}
	}
	return identityCollectionSpec{}, false
}

func identityExpectedNameQuery(spec identityCollectionSpec) string {
	if spec.noNameQuery {
		return ""
	}
	return "name"
}

// SecurityGroup alone has an audited concrete native ListOpts signature. The
// owned raw-query iterator preserves its native pager without exposing new
// details or all-projects capabilities or changing the public native List API.
func identityRawListMetadataValid(spec identityCollectionSpec) bool {
	if spec.path != "network/v2/extensions/security/groups" {
		return spec.rawListIterator == ""
	}
	return spec.model == "SecGroup" && spec.getter == "Get" && spec.lister == "List" && spec.parents == 0 && len(spec.getSegments) == 2 && spec.getSegments[0] == "security-groups" && spec.getSegments[1] == "$id" && len(spec.getCodes) == 1 && spec.getCodes[0] == 200 && !spec.noNameQuery && spec.rawListIterator == "IterateSecurityGroups"
}

func identityNeutronListSchema(pkg *types.Package, plan *collectionPlan) bool {
	pageName, extractor := "RouterPage", "ExtractRouters"
	if sdkPath(pkg.Path()) == "network/v2/extensions/security/groups" {
		pageName, extractor = "SecGroupPage", "ExtractGroups"
	}
	page := pkg.Scope().Lookup(pageName)
	if page == nil {
		return false
	}
	fields, ok := page.Type().Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 1 || !fields.Field(0).Embedded() || types.TypeString(fields.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.LinkedPageBase" {
		return false
	}
	for name, result := range map[string]types.Type{"IsEmpty": types.Typ[types.Bool], "NextPageURL": types.Typ[types.String]} {
		method := extractionMethod(page.Type(), name)
		if method == nil || !types.Identical(method.Results().At(0).Type(), result) {
			return false
		}
	}
	extract, ok := pkg.Scope().Lookup(extractor).(*types.Func)
	if !ok {
		return false
	}
	sig := extract.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 1 || sig.Results().Len() != 2 || types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || !types.Identical(sig.Results().At(0).Type(), types.NewSlice(plan.model)) || !isError(sig.Results().At(1).Type()) {
		return false
	}
	if pageName == "RouterPage" {
		into, ok := pkg.Scope().Lookup("ExtractRoutersInto").(*types.Func)
		if !ok {
			return false
		}
		sig = into.Type().(*types.Signature)
		if sig.Variadic() || sig.Params().Len() != 2 || types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || !types.Identical(sig.Params().At(1).Type(), types.NewInterfaceType(nil, nil).Complete()) || sig.Results().Len() != 1 || !isError(sig.Results().At(0).Type()) {
			return false
		}
	} else {
		// Native groups.List must continue to receive the concrete local opts.
		sig = plan.lister.Type().(*types.Signature)
		if sig.Params().Len() != 2 || clientParam(sig) != 0 {
			return false
		}
		opts, ok := sig.Params().At(1).Type().(*types.Named)
		if !ok || opts.Obj().Pkg() != pkg || opts.Obj().Name() != "ListOpts" || !types.Identical(opts, plan.listInput) {
			return false
		}
	}
	return true
}

// SubnetPool's custom prefix-length decoder and Trunk's inherited links.next
// pager are native contracts; neither resource gains an SDK list mode.
func identityPoolTrunkSchema(pkg *types.Package, plan *collectionPlan) bool {
	pageName, extractor := "SubnetPoolPage", "ExtractSubnetPools"
	trunk := sdkPath(pkg.Path()) == "network/v2/extensions/trunks"
	if trunk {
		pageName, extractor = "TrunkPage", "ExtractTrunks"
	}
	pageObject := pkg.Scope().Lookup(pageName)
	if pageObject == nil {
		return false
	}
	page, ok := pageObject.Type().(*types.Named)
	if !ok {
		return false
	}
	fields, ok := page.Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 1 || !fields.Field(0).Embedded() || types.TypeString(fields.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.LinkedPageBase" {
		return false
	}
	for name, result := range map[string]types.Type{"IsEmpty": types.Typ[types.Bool], "NextPageURL": types.Typ[types.String]} {
		method := extractionMethod(page, name)
		if method == nil || !types.Identical(method.Results().At(0).Type(), result) {
			return false
		}
	}
	if trunk {
		for i := 0; i < page.NumMethods(); i++ {
			if page.Method(i).Name() == "NextPageURL" {
				return false
			}
		}
	}
	extract, ok := pkg.Scope().Lookup(extractor).(*types.Func)
	if !ok {
		return false
	}
	sig := extract.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 1 || sig.Results().Len() != 2 || types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || !types.Identical(sig.Results().At(0).Type(), types.NewSlice(plan.model)) || !isError(sig.Results().At(1).Type()) {
		return false
	}
	if !trunk {
		body, ok := plan.model.Underlying().(*types.Struct)
		if !ok {
			return false
		}
		for _, name := range []string{"DefaultPrefixLen", "MinPrefixLen", "MaxPrefixLen"} {
			found := false
			for i := 0; i < body.NumFields(); i++ {
				if body.Field(i).Name() == name {
					found = types.Identical(body.Field(i).Type(), types.Typ[types.Int]) && reflect.StructTag(body.Tag(i)).Get("json") == "-"
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

// These two native models use ordinary JSON decoding and their own linked
// continuation. Preserve full native fields without inferring extension gates
// or introducing a decoder, parent scope, or identity list mode.
func identityQoSAddressSchema(pkg *types.Package, plan *collectionPlan) bool {
	policy := sdkPath(pkg.Path()) == "network/v2/extensions/qos/policies"
	pageName, extractor := "AddressGroupPage", "ExtractGroups"
	fields := map[string][2]string{
		"ID": {"string", "id"}, "Name": {"string", "name"},
		"Description": {"string", "description"}, "ProjectID": {"string", "project_id"},
		"Addresses": {"[]string", "addresses"},
	}
	if policy {
		pageName, extractor = "PolicyPage", "ExtractPolicies"
		delete(fields, "Addresses")
		for name, value := range map[string][2]string{
			"TenantID": {"string", "tenant_id"}, "CreatedAt": {"time.Time", "created_at"},
			"UpdatedAt": {"time.Time", "updated_at"}, "IsDefault": {"bool", "is_default"},
			"Shared": {"bool", "shared"}, "RevisionNumber": {"int", "revision_number"},
			"Rules": {"[]map[string]any", "rules"}, "Tags": {"[]string", "tags"},
		} {
			fields[name] = value
		}
	}
	body, ok := plan.model.Underlying().(*types.Struct)
	if !ok || body.NumFields() != len(fields) {
		return false
	}
	for i := 0; i < body.NumFields(); i++ {
		field := body.Field(i)
		want, found := fields[field.Name()]
		matchesType := types.TypeString(field.Type(), func(p *types.Package) string { return p.Path() }) == want[0]
		if want[0] == "[]map[string]any" {
			matchesType = types.Identical(field.Type(), types.NewSlice(types.NewMap(types.Typ[types.String], types.NewInterfaceType(nil, nil).Complete())))
		}
		if !found || field.Embedded() || !matchesType || reflect.StructTag(body.Tag(i)).Get("json") != want[1] {
			return false
		}
	}
	model, ok := plan.model.(*types.Named)
	if !ok {
		return false
	}
	for i := 0; i < model.NumMethods(); i++ {
		if model.Method(i).Name() == "UnmarshalJSON" {
			return false
		}
	}
	pageObject := pkg.Scope().Lookup(pageName)
	if pageObject == nil {
		return false
	}
	page, ok := pageObject.Type().(*types.Named)
	if !ok {
		return false
	}
	pageFields, ok := page.Underlying().(*types.Struct)
	if !ok || pageFields.NumFields() != 1 || !pageFields.Field(0).Embedded() || types.TypeString(pageFields.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.LinkedPageBase" {
		return false
	}
	for name, result := range map[string]types.Type{"IsEmpty": types.Typ[types.Bool], "NextPageURL": types.Typ[types.String]} {
		method := extractionMethod(page, name)
		own := false
		for i := 0; i < page.NumMethods(); i++ {
			own = own || page.Method(i).Name() == name
		}
		if !own || method == nil || !types.Identical(method.Results().At(0).Type(), result) {
			return false
		}
	}
	extract, ok := pkg.Scope().Lookup(extractor).(*types.Func)
	if !ok {
		return false
	}
	sig := extract.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 1 || sig.Results().Len() != 2 || types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || !types.Identical(sig.Results().At(0).Type(), types.NewSlice(plan.model)) || !isError(sig.Results().At(1).Type()) {
		return false
	}
	if policy {
		into, ok := pkg.Scope().Lookup("ExtractPolicysInto").(*types.Func)
		if !ok {
			return false
		}
		sig = into.Type().(*types.Signature)
		if sig.Variadic() || sig.Params().Len() != 2 || types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || !types.Identical(sig.Params().At(1).Type(), types.NewInterfaceType(nil, nil).Complete()) || sig.Results().Len() != 1 || !isError(sig.Results().At(0).Type()) {
			return false
		}
	}
	return true
}

// Network body selection must retain the native timestamp decoder and the
// network envelope/extraction chain. This does not add an identity option or
// reinterpret the existing native List's wire query parameters.
func identityNetworkListSchema(pkg *types.Package, plan *collectionPlan) bool {
	model, ok := plan.model.(*types.Named)
	if !ok {
		return false
	}
	decoder, _, _ := types.LookupFieldOrMethod(types.NewPointer(model), true, nil, "UnmarshalJSON")
	decode, ok := decoder.(*types.Func)
	if !ok {
		return false
	}
	sig := decode.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 1 || !types.Identical(sig.Params().At(0).Type(), types.NewSlice(types.Typ[types.Uint8])) || sig.Results().Len() != 1 || !isError(sig.Results().At(0).Type()) {
		return false
	}
	pageObject := pkg.Scope().Lookup("NetworkPage")
	if pageObject == nil {
		return false
	}
	page, ok := pageObject.Type().(*types.Named)
	if !ok {
		return false
	}
	fields, ok := page.Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 1 || !fields.Field(0).Embedded() || types.TypeString(fields.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.LinkedPageBase" {
		return false
	}
	for name, result := range map[string]types.Type{"IsEmpty": types.Typ[types.Bool], "NextPageURL": types.Typ[types.String]} {
		method := extractionMethod(page, name)
		own := false
		for i := 0; i < page.NumMethods(); i++ {
			own = own || page.Method(i).Name() == name
		}
		if !own || method == nil || !types.Identical(method.Results().At(0).Type(), result) {
			return false
		}
	}
	resourceKey, _, _ := types.LookupFieldOrMethod(page, true, nil, "ResourceKey")
	key, ok := resourceKey.(*types.Func)
	if !ok {
		return false
	}
	sig = key.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 0 || sig.Results().Len() != 1 || !types.Identical(sig.Results().At(0).Type(), types.Typ[types.String]) {
		return false
	}
	extract, ok := pkg.Scope().Lookup("ExtractNetworks").(*types.Func)
	if !ok {
		return false
	}
	sig = extract.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 1 || types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || sig.Results().Len() != 2 || !types.Identical(sig.Results().At(0).Type(), types.NewSlice(plan.model)) || !isError(sig.Results().At(1).Type()) {
		return false
	}
	into, ok := pkg.Scope().Lookup("ExtractNetworksInto").(*types.Func)
	if !ok {
		return false
	}
	sig = into.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 2 || types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || !types.Identical(sig.Params().At(1).Type(), types.NewInterfaceType(nil, nil).Complete()) || sig.Results().Len() != 1 || !isError(sig.Results().At(0).Type()) {
		return false
	}
	getResult := identityGetResult(pkg, plan, 0)
	if getResult == nil {
		return false
	}
	resultInto, _, _ := types.LookupFieldOrMethod(getResult, true, nil, "ExtractInto")
	resultMethod, ok := resultInto.(*types.Func)
	if !ok {
		return false
	}
	sig = resultMethod.Type().(*types.Signature)
	return !sig.Variadic() && sig.Params().Len() == 1 && types.Identical(sig.Params().At(0).Type(), types.NewInterfaceType(nil, nil).Complete()) && sig.Results().Len() == 1 && isError(sig.Results().At(0).Type())
}

// Trunk relies on inherited continuation, and Subnet's raw-row Body lane
// requires the original JSON-number-aware page body. Imported type data does
// not include method bodies, so load only these audited dependency methods.
func (g *generator) identityPaginationDeclarations(path string) (map[string]*ast.FuncDecl, error) {
	wanted := map[string]bool{}
	label := "audited Trunk identity collection"
	switch sdkPath(path) {
	case "network/v2/extensions/trunks":
		wanted["LinkedPageBase.NextPageURL"] = true
	case "network/v2/subnets":
		label = "audited Subnet body collection"
		wanted["PageResultFrom"] = true
		wanted["PageResultFromParsed"] = true
		wanted["LinkedPageBase.GetBody"] = true
	case "network/v2/extensions/security/addressgroups":
		label = "audited AddressGroup body collection"
		wanted["PageResultFrom"] = true
		wanted["PageResultFromParsed"] = true
		wanted["LinkedPageBase.GetBody"] = true
	case "network/v2/extensions/qos/policies":
		label = "audited QoSPolicy body collection"
		wanted["PageResultFrom"] = true
		wanted["PageResultFromParsed"] = true
		wanted["LinkedPageBase.GetBody"] = true
	case "keymanager/v1/containers":
		label = "audited Container body collection"
		wanted["PageResultFrom"] = true
		wanted["PageResultFromParsed"] = true
		wanted["LinkedPageBase.GetBody"] = true
	case "keymanager/v1/orders":
		label = "audited Order body collection"
		wanted["PageResultFrom"] = true
		wanted["PageResultFromParsed"] = true
		wanted["LinkedPageBase.GetBody"] = true
	case "keymanager/v1/secrets":
		label = "audited Secret body collection"
		wanted["PageResultFrom"] = true
		wanted["PageResultFromParsed"] = true
		wanted["LinkedPageBase.GetBody"] = true
	default:
		return nil, nil
	}
	source, ok := g.meta[upstreamModule+"/pagination"]
	if !ok || source.Dir == "" || len(source.GoFiles) == 0 {
		return nil, fmt.Errorf("%s: native pagination dependency metadata missing", label)
	}
	result := map[string]*ast.FuncDecl{}
	for _, name := range source.GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(source.Dir, name), nil, 0)
		if err != nil {
			return nil, fmt.Errorf("%s: native pagination dependency: %w", label, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !wanted[identityDeclarationKey(fn)] {
				continue
			}
			key := "pagination." + identityDeclarationKey(fn)
			if result[key] != nil {
				return nil, fmt.Errorf("%s: duplicate pagination declaration %s", label, key)
			}
			result[key] = fn
		}
	}
	return result, nil
}

// Flavor's list default and optional enrichment are deliberately separate from
// details/all-projects modes. ListDetail is its only audited native list route.
func identityFlavorMetadataValid(spec identityCollectionSpec) bool {
	if spec.path != "compute/v2/flavors" {
		return !spec.noNameQuery && spec.listDefaultKey == "" && spec.listDefaultValue == "" && !spec.extraSpecs
	}
	return spec.model == "Flavor" && spec.getter == "Get" && spec.lister == "ListDetail" && spec.parents == 0 && len(spec.getSegments) == 2 && spec.getSegments[0] == "flavors" && spec.getSegments[1] == "$id" && len(spec.getCodes) == 1 && spec.getCodes[0] == 200 && spec.noNameQuery && spec.listDefaultKey == "is_public" && spec.listDefaultValue == "None" && spec.extraSpecs
}

func identityFlavorEnabled(pkg *types.Package, plan *collectionPlan, parents int) bool {
	spec, ok := identityCollectionContract(pkg, plan, parents)
	return ok && spec.extraSpecs
}

func identityFlavorSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan.nameQuery != "" || queryTag(plan.listInput, "AccessType") != "is_public" {
		return false
	}
	access, _, _ := types.LookupFieldOrMethod(plan.listInput, true, nil, "AccessType")
	if access == nil || !isString(access.Type()) {
		return false
	}
	fields, ok := plan.model.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	found := false
	for i := 0; i < fields.NumFields(); i++ {
		if fields.Field(i).Name() == "ExtraSpecs" {
			value, ok := fields.Field(i).Type().(*types.Map)
			if !ok || !types.Identical(value.Key(), types.Typ[types.String]) || !types.Identical(value.Elem(), types.Typ[types.String]) || reflect.StructTag(fields.Tag(i)).Get("json") != "extra_specs" {
				return false
			}
			found = true
		}
	}
	if !found {
		return false
	}
	page := pkg.Scope().Lookup("FlavorPage")
	if page == nil {
		return false
	}
	pageFields, ok := page.Type().Underlying().(*types.Struct)
	if !ok || pageFields.NumFields() != 1 || !pageFields.Field(0).Embedded() || types.TypeString(pageFields.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.LinkedPageBase" {
		return false
	}
	for name, result := range map[string]types.Type{"IsEmpty": types.Typ[types.Bool], "NextPageURL": types.Typ[types.String]} {
		method := extractionMethod(page.Type(), name)
		if method == nil || !types.Identical(method.Results().At(0).Type(), result) {
			return false
		}
	}
	extract, ok := pkg.Scope().Lookup("ExtractFlavors").(*types.Func)
	if !ok {
		return false
	}
	sig := extract.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 1 || sig.Results().Len() != 2 || types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || !types.Identical(sig.Results().At(0).Type(), types.NewSlice(plan.model)) || !isError(sig.Results().At(1).Type()) {
		return false
	}
	getter, ok := pkg.Scope().Lookup("ListExtraSpecs").(*types.Func)
	if !ok {
		return false
	}
	sig = getter.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 3 || !isContext(sig.Params().At(0).Type()) || clientParam(sig) != 1 || !isString(sig.Params().At(2).Type()) || sig.Results().Len() != 1 {
		return false
	}
	result, ok := sig.Results().At(0).Type().(*types.Named)
	if !ok || result.Obj().Pkg() != pkg || result.Obj().Name() != "ListExtraSpecsResult" {
		return false
	}
	method := extractionMethod(result, "Extract")
	if method == nil || !types.Identical(method.Results().At(0).Type(), types.NewMap(types.Typ[types.String], types.Typ[types.String])) {
		return false
	}
	return true
}

// Only Glance's audited Image.find performs one additional list after a
// successful complete absence. The fixed overlay is SDK policy, not caller
// query or an inferred capability of every name-capable pager.
func identityMissingListMetadataValid(spec identityCollectionSpec) bool {
	if spec.path != "image/v2/images" {
		return spec.missingListKey == "" && spec.missingListValue == ""
	}
	return spec.model == "Image" && spec.getter == "Get" && spec.lister == "List" && spec.parents == 0 && len(spec.getSegments) == 2 && spec.getSegments[0] == "images" && spec.getSegments[1] == "$id" && len(spec.getCodes) == 1 && spec.getCodes[0] == 200 && spec.missingListKey == "os_hidden" && spec.missingListValue == "true"
}

func identityMissingListEnabled(pkg *types.Package, plan *collectionPlan, parents int) bool {
	spec, ok := identityCollectionContract(pkg, plan, parents)
	return ok && spec.missingListKey != ""
}

func identityImageListSchema(pkg *types.Package, plan *collectionPlan) bool {
	boolField := func(value types.Type, name, tag, key string) bool {
		fields, ok := value.Underlying().(*types.Struct)
		if !ok {
			return false
		}
		for i := 0; i < fields.NumFields(); i++ {
			if fields.Field(i).Name() == name {
				return types.Identical(fields.Field(i).Type(), types.Typ[types.Bool]) && reflect.StructTag(fields.Tag(i)).Get(tag) == key
			}
		}
		return false
	}
	if !boolField(plan.listInput, "Hidden", "q", "os_hidden") || !boolField(plan.model, "Hidden", "json", "os_hidden") {
		return false
	}
	page := pkg.Scope().Lookup("ImagePage")
	if page == nil {
		return false
	}
	fields, ok := page.Type().Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 2 || fields.Field(0).Name() != "serviceURL" || !types.Identical(fields.Field(0).Type(), types.Typ[types.String]) || !fields.Field(1).Embedded() || types.TypeString(fields.Field(1).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.LinkedPageBase" {
		return false
	}
	extract, ok := pkg.Scope().Lookup("ExtractImages").(*types.Func)
	if !ok {
		return false
	}
	sig := extract.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 1 || sig.Results().Len() != 2 || types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || !types.Identical(sig.Results().At(0).Type(), types.NewSlice(plan.model)) || !isError(sig.Results().At(1).Type()) {
		return false
	}
	for name, result := range map[string]types.Type{"IsEmpty": types.Typ[types.Bool], "NextPageURL": types.Typ[types.String]} {
		method := extractionMethod(page.Type(), name)
		if method == nil || !types.Identical(method.Results().At(0).Type(), result) {
			return false
		}
	}
	return true
}

// The query GET writes into the native result before calling its own Extract.
// It must not silently manufacture a wrapper or discard an upstream option.
func identityGetResult(pkg *types.Package, plan *collectionPlan, parents int) *types.Named {
	sig := plan.getter.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != parents+3 || sig.Results().Len() != 1 || !isContext(sig.Params().At(0).Type()) || clientParam(sig) != 1 {
		return nil
	}
	for i := 2; i < sig.Params().Len(); i++ {
		if !isString(sig.Params().At(i).Type()) {
			return nil
		}
	}
	result, ok := sig.Results().At(0).Type().(*types.Named)
	if !ok || result.Obj().Pkg() != pkg || !result.Obj().Exported() {
		return nil
	}
	if _, ok := result.Underlying().(*types.Struct); !ok {
		return nil
	}
	for name, want := range map[string]string{"Body": "any", "Header": "net/http.Header", "Err": "error"} {
		object, _, _ := types.LookupFieldOrMethod(result, true, nil, name)
		field, ok := object.(*types.Var)
		if !ok || !field.IsField() {
			return nil
		}
		if want == "any" {
			iface, ok := field.Type().Underlying().(*types.Interface)
			if !ok || iface.NumMethods() != 0 {
				return nil
			}
		} else if types.TypeString(field.Type(), func(p *types.Package) string { return p.Path() }) != want {
			return nil
		}
	}
	extract := extractionMethod(result, "Extract")
	if extract == nil || !types.Identical(extract.Results().At(0).Type(), types.NewPointer(plan.model)) {
		return nil
	}
	return result
}

func emitIdentityGetQuery(e *emitter, plan *collectionPlan, receiver string, parents []string, spec identityCollectionSpec) {
	e.use("gophercloudsdk/internal/nativefind")
	result := identityGetResult(e.pkg, plan, len(parents))
	segments := make([]string, len(spec.getSegments))
	for i, segment := range spec.getSegments {
		switch segment {
		case "$id":
			segments[i] = "id"
		case "$parent":
			segments[i] = parents[0]
		default:
			segments[i] = strconv.Quote(segment)
		}
	}
	codes := make([]string, len(spec.getCodes))
	for i, code := range spec.getCodes {
		codes[i] = strconv.Itoa(code)
	}
	e.printf("GetIdentityQuery:func(ctx context.Context,id string,q url.Values)(*%s,error){var result %s.%s;result.Header,result.Err=nativefind.Get(ctx,%s.RawClient(),[]string{%s},q,[]int{%s},&result.Body);return result.Extract()},\n", plan.modelName, e.use(e.pkg.Path()), result.Obj().Name(), receiver, strings.Join(segments, ","), strings.Join(codes, ","))
}

// These declarations are audited against gophercloud v2.15.0 requests.go and
// urls.go. Nil RequestOpts means GET 200 (provider_client.go defaultOkCodes);
// Nova alone explicitly accepts 200/203. Route-helper hashes include indirect
// helpers, while the only URL constants are checked separately below.
const identityDefaultGetSHA = "aa6bf83ccd6b2db8e3c0d09e73ca2cd3d7d4e69d9ff8d604080177e38eb40232"
const identityNetworkGetSHA = "f2e21c88e17a8ad49060e84dad42244e17c64453bae932aadebe34ad0e7ebc9b"
const identityNetworkURLSHA = "4e7e71e4b36e374a2fc6830f4f621ab3dd904cb2e1fa10b99e43a5590b7b4b36"

var identityNativeDeclarations = map[string]map[string]string{
	"compute/v2/servers":      {"Get": "50ec0a3583ae35c61b1f1c3e573f5b5408421e87a2de87b0d1db1e6bcd6178e3", "getURL": "75785c38f3e330094952e6ff7b46ea79dadf61b44cb39090cf6ae3daabbb413d", "deleteURL": "8bb07ce6dfa8e67dac1b1b280aa1fe073ea3297002e2feebe3a8a653f29112ad"},
	"blockstorage/v3/volumes": {"Get": identityDefaultGetSHA, "getURL": "4194d2d5e9a4771b8f25b5acc62b52272e741d744d3ae4b8d757e6eaa8b88ec8", "deleteURL": "276fb7983145acc055ef71e5aaa7012fa5767b2aa887a7d4c99f7a1bf47c928b"},
	"network/v2/ports":        {"Get": identityNetworkGetSHA, "getURL": identityNetworkURLSHA, "resourceURL": "ac2ec66769c6a1e92e42344a62cf06bfcf1c1170f91867ef502344924b97c7ec"},
	"network/v2/networks": {
		"Get": identityNetworkGetSHA, "getURL": identityNetworkURLSHA, "resourceURL": "313da020f0b09244e553dac107ce73d06de65812ce24c0339473cc8befd6332a",
		"List":                        "cc52c5ec8c677374bafc4322ae68c43f0dce73cc84000b434e15267d3b617d20",
		"ListOpts.ToNetworkListQuery": "786e84233300d77b73a1e7d1b2f13ec4d5a42ac7bd085a49421e6b15f3394c90",
		"rootURL":                     "dfa59922daefee3ab2d5e490b69377ca7a0e58f52ef4e06b59de8837f18fb2c3",
		"listURL":                     "0a55ee851552d789ddd3c12304e6d0d209cae0cfd477d71b138196681486bf9d",
		"commonResult.Extract":        "f958482bc76477b9059a8c3a4d921dae13668f04f28fb93340ccb9a8709f586b",
		"commonResult.ExtractInto":    "51fd854e39753b80bd255703daeedbf5d4e279e4542dbb226aaa98247134314f",
		"Network.UnmarshalJSON":       "f9840328e1d80cc0b32221f0acfa5c98ebe426375dca0985dc2eedf008e1e2ee",
		"NetworkPage.ResourceKey":     "443f3b4a977a9746314eddc9b8db71d0269e53854d1b5fd9f067764f6ab060a7",
		"NetworkPage.NextPageURL":     "e6f3f1611b0d2892145f024b0e2fae31b2d29f282a3d24dd25605c1ec4556efc",
		"NetworkPage.IsEmpty":         "756bddbbc265e3b30dcff213d983b457559b713f7a0299588d0232a2914b1abc",
		"ExtractNetworks":             "5a2d1db17e38f0070dfdac235159c3aae4589cf69a4399e2f00a3bdd02a1448f",
		"ExtractNetworksInto":         "808d8179a8b72552591dc81fbb2cd482cef8c07873a77cda7063d5af0fd2f512",
	},
	"network/v2/subnets": {
		"Get": identityNetworkGetSHA, "getURL": identityNetworkURLSHA, "resourceURL": "0fffec6b477ce31fc27ed1dc2ab5c205c5290d79980b3aa259b179ee7766b263",
		"ListOpts.ToSubnetListQuery":        "96d27a9981d705c59461e14b7eec469cc7012495aef26aa86eb27f1ccb97725d",
		"List":                              "3f9c6f5492912506178bf90a8099a7fc5b20449482d4f92e3bf374426ff8feb3",
		"rootURL":                           "9065832569065787974f0229428d95df14d276dffccf043be3863252a717b09a",
		"listURL":                           "0a55ee851552d789ddd3c12304e6d0d209cae0cfd477d71b138196681486bf9d",
		"commonResult.Extract":              "a671ef3f1a4b74842dcc9f26e6fddae21b4346f77ea39bb927577f1bdf1bee72",
		"Subnet.UnmarshalJSON":              "651c7cc93b57dc51a64d179755bf27c239f1186b9d930126d98c8c5f705519b5",
		"SubnetPage.NextPageURL":            "1789418f24f1280398ccee230d5fed129fd2eda84b86cfadb7fedd0e4ccd8a2c",
		"SubnetPage.IsEmpty":                "77be080544675db1cbf4e7916f8545b1f952d4c637187346601225cd411c5c64",
		"ExtractSubnets":                    "4fde13ed5bc13c3f980d931bd6e58489d9a0c65d91757a1da44d19a948e63da4",
		"pagination.PageResultFrom":         "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9",
		"pagination.PageResultFromParsed":   "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089",
		"pagination.LinkedPageBase.GetBody": "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284",
	},
	"identity/v3/projects":  {"Get": identityDefaultGetSHA, "getURL": "d5e4289c71c7a028ecb7fe6a2a47b1e4a8f6438bdaccbc17c7fe1469bc0461f6"},
	"identity/v3/users":     {"Get": identityDefaultGetSHA, "getURL": "d3b727df4f5525b08c0dc43b6cb255bb5a6bb1b53da4c3fcbad076787334998e"},
	"identity/v3/groups":    {"Get": identityDefaultGetSHA, "getURL": "0b7a41e86f5eb5dafbcb193d202162b4385462274178934a7ec20f7efbe9ddbe"},
	"identity/v3/domains":   {"Get": identityDefaultGetSHA, "getURL": "4eed19909e41e8b752adbb281428ba5a3418746dbf5e5410e20078841b9a7475"},
	"identity/v3/roles":     {"Get": identityDefaultGetSHA, "getURL": "17a433f86f71243f80bdd8826fe2bc785f8950a1c6a166fedd2a171bc2c2c95d"},
	"dns/v2/recordsets":     {"Get": "045a8befff06647b70ecbe4560f7a809c991290496b3032cebd876a412a90bd4", "rrsetURL": "71398da5ec5a9461f7e0d9ebc8e569b7c27f09d0f2389191a87e0eca75c9a348"},
	"loadbalancer/v2/pools": {"GetMember": "9173191d1b3baa7d542d364f4efaa137fdfa84bfe9ee921feb4f09569c384be0", "memberResourceURL": "0ac8d3833de97abc76ece2b63ad0e265e7ed2407a79ed6c75307d8e8538ded9e"},
	"image/v2/images": {
		"Get":                   identityDefaultGetSHA,
		"getURL":                "355bc2abb9f6b47459672d711305c59e8752ca60544b51974e4cf17d40d32b79",
		"imageURL":              "ea231904b908b3bc2434f047f5138294a049d86e5b4b1a7c85445bfaf044a505",
		"List":                  "981709ad79bbf9aee3e1ff8a30515a1a274838c3b6eb4854dc39f661377a3ef1",
		"listURL":               "c58721a00b1c1b3dd3b75dced77e3afc2b72d2ab9610d85b8d76d05af388f3a9",
		"commonResult.Extract":  "8b6be427158a900857f0a4672e80308615a9ce9ed9a4a84af494f3d3d9ae5ae6",
		"ExtractImages":         "1de7a58440781a8e7359d5c9e4b339f00dbac3f6d3b49f3f16871830110609ed",
		"ImagePage.IsEmpty":     "b697d4a95e86389114db26bf21230180d982ed7d9534d2685f96dfb98d49367d",
		"ImagePage.NextPageURL": "8af361267917a4230d811f9938d8d4b3057096263f704c7b831db46dc0b0318e",
		"nextPageURL":           "7ac8c7e2868170732aa46e3bfa231d509eee709c214323a364e1e6e4047400f7",
	},
	"compute/v2/flavors": {
		"Get":                      identityDefaultGetSHA,
		"getURL":                   "f9856433ef837cb4589adba04a50e477e82c357dba1289e2a424ce4a88be9daf",
		"ListDetail":               "2c2031a2f9157750ca7e57591255b3297a003f675243056e6f64ae569cefd5d4",
		"listURL":                  "0f37919309f176c5f80901a6e0a815f2468803dc4d42bacbf55f25d316f2a81e",
		"commonResult.Extract":     "ca85e75b9e06417e9c7bc4f2ba3287438c3dfe6e26e2cf5f26a236c1849fc033",
		"Flavor.UnmarshalJSON":     "e90a02c1393493482f84e729304b8faa4761cadf6ad55da5b833af00a9da88ce",
		"ExtractFlavors":           "2bfbd3cf1cb46982fd8666b499165b0c5ee2efd183c3c628ee3d9b8ca11a5110",
		"FlavorPage.IsEmpty":       "6371bcf987b7f30157c5f5ea1df684d9d084715b3ff6b160576d510b3b541fc7",
		"FlavorPage.NextPageURL":   "0d9fdb462fae3de233ddd9082300c49d516833cc603414059ad63e9b9b6acfd9",
		"ListExtraSpecs":           "b18d0bb516220edefc62e35b76f98f0f7996f848ec59d7db695c14ed6a2f1625",
		"extraSpecsListURL":        "df35511fb8f504daaf787fd8b3881157d0f38de6d2a09a47e5e747904e3595fa",
		"extraSpecsResult.Extract": "22b06d0f0d4e5a25aa29d4b834f500363f04a517abcee559d5320121331b8d5e",
	},
	"network/v2/extensions/layer3/routers": {
		"Get":                        "c0787d9b62a2cda3caa71d9a5deeaa3b99842417646def58aea2161caf892fb9",
		"rootURL":                    "4f185db1078eefd1b5f5b38bd1c601bd6fccaa3c20fa99a7b85371ef7f7b8104",
		"resourceURL":                "ca0246cf0e192c0133b2d43b9c5b577badf51c51344d3c5fc499d01a97ef54c8",
		"List":                       "452ffb77d4836fef164e8c731d85511442433e043be2198359a7534d2def0420",
		"ListOpts.ToRouterListQuery": "16630576a814d8e9f82bd02355bd7367745f3e7062990da4efe898e23b8dc12f",
		"Router.UnmarshalJSON":       "5f6e06c6894b79e08121cd89520019211ab88760dd5247b4bf6a3697a55b5377",
		"RouterPage.NextPageURL":     "4323c0d1e0740e56af094d4f0df9c943acb1f2409f2fab8fb26c6ee93fb71a9c",
		"RouterPage.IsEmpty":         "1962b3680eb57993f0a30287f025a408eefdd5d11e55d4e28fc683a75612c3ab",
		"ExtractRouters":             "f4a7e6f4f0ee7722ff1d297a2fb026d843cce89d137d60dd85214f0a20f29e18",
		"ExtractRoutersInto":         "be02017c665dfc07a48940b7c44d0adc592ab754f39044fb6e0fbe77403a759e",
		"commonResult.Extract":       "4bb6eaf3e9ca18e71322e50f621501195d9aab13ecca5a1fc7acc7f263ac6df4",
	},
	"network/v2/extensions/security/groups": {
		"Get":                      "c0787d9b62a2cda3caa71d9a5deeaa3b99842417646def58aea2161caf892fb9",
		"rootURL":                  "5a4bf90be880f0165342169abba24c6938989c185637bae8e4c5092f33ac9f3b",
		"resourceURL":              "ff31c93c543d54044b2c72a7219311759861582c91db4787e04a5f2d3dfc060c",
		"List":                     "5b43132cd8886b599837c8ff9268bef10f152cd644d82774471fd964d568b2e1",
		"SecGroup.UnmarshalJSON":   "7f81c1ad2bc0435c683b7d3033de0bd391d6db90ede9478900b89ee79700f7e6",
		"SecGroupPage.NextPageURL": "49a58e1cf6d83daa67e619f1bcd26a55428baba575a5909f51541bee417bfd6b",
		"SecGroupPage.IsEmpty":     "06143a32b9aef0ab4b8bb63239b4a05a0c38bdfcf863fbcb7c9d92a3a6423f2b",
		"ExtractGroups":            "e42007fd330615fa5e88a454260fee2cfd8a1c2b604d42bd4dc4ff7db325d423",
		"commonResult.Extract":     "0593db713468ed605d0dbfce265a3003901cedcf742051a00a3e1f385ab13df2",
	},
	"network/v2/extensions/subnetpools": {
		"Get":                            "f2e21c88e17a8ad49060e84dad42244e17c64453bae932aadebe34ad0e7ebc9b",
		"List":                           "b5685bd0990f632e5bc6b3824520185616669b675d28c83168502946acf8d242",
		"ListOpts.ToSubnetPoolListQuery": "c6af8790e25021db5d472bfef7cfe34e4c692bcacce1eaf79c2451e9e4343779",
		"getURL":                         "4e7e71e4b36e374a2fc6830f4f621ab3dd904cb2e1fa10b99e43a5590b7b4b36",
		"listURL":                        "0a55ee851552d789ddd3c12304e6d0d209cae0cfd477d71b138196681486bf9d",
		"rootURL":                        "4f185db1078eefd1b5f5b38bd1c601bd6fccaa3c20fa99a7b85371ef7f7b8104",
		"resourceURL":                    "ca0246cf0e192c0133b2d43b9c5b577badf51c51344d3c5fc499d01a97ef54c8",
		"commonResult.Extract":           "e70226430ded7b75197a644df50e5484cbb78fcfc8174beba0bb7add0dea5310",
		"SubnetPool.UnmarshalJSON":       "7166552c0b959e3559a7f81f10bec4072c454530187c73331faf174efce743ef",
		"SubnetPoolPage.NextPageURL":     "5002bfbc328292413a3aaa785d5fad4a23dcd0835581440ff41497deeb0b7d57",
		"SubnetPoolPage.IsEmpty":         "4843c1ad38084a3f8fa778644a55267a8cdc9f052735cb111f4299e8285a978f",
		"ExtractSubnetPools":             "d55581a7189bdae0d033052889b6db3043ecfb67892fc537809b4cec522de3d8",
	},
	"network/v2/extensions/trunks": {
		"Get":                                   "f2e21c88e17a8ad49060e84dad42244e17c64453bae932aadebe34ad0e7ebc9b",
		"List":                                  "9fcac9fb5387b699ef747b96fd8b049b6319d15e9b22ab1f0d28ae9fbbc890cc",
		"ListOpts.ToTrunkListQuery":             "46017753bc904ae4fe4aa007cea675e1fd1378173905292c61f5cc32bd194920",
		"getURL":                                "4e7e71e4b36e374a2fc6830f4f621ab3dd904cb2e1fa10b99e43a5590b7b4b36",
		"listURL":                               "0a55ee851552d789ddd3c12304e6d0d209cae0cfd477d71b138196681486bf9d",
		"rootURL":                               "4f185db1078eefd1b5f5b38bd1c601bd6fccaa3c20fa99a7b85371ef7f7b8104",
		"resourceURL":                           "ca0246cf0e192c0133b2d43b9c5b577badf51c51344d3c5fc499d01a97ef54c8",
		"commonResult.Extract":                  "a86967d1e97260072f2f9e8a2c1ff10644a529f65912852fe0781206d1a9c332",
		"TrunkPage.IsEmpty":                     "594c7f1c9b29430d07dccfd9c819a288c71d3c1835075bfd1d7dcb133618121d",
		"ExtractTrunks":                         "95034dff77e7308bcd4c851032b735581b3c3e4e523f1b354b7255a76c95e6be",
		"pagination.LinkedPageBase.NextPageURL": "fa8678035238acb855e2d60896aa155d50de519d8545830ed52220490a02ad4e",
	},
	"network/v2/extensions/qos/policies": {
		"Get":                        "f2e21c88e17a8ad49060e84dad42244e17c64453bae932aadebe34ad0e7ebc9b",
		"List":                       "04d9ad4cd55b330c83fc0c5e013b75491d40a00b0c1c3615414bc1bd8095f5e7",
		"ListOpts.ToPolicyListQuery": "d6253365fadc272a3df97e2569174347bd27f82ea52eac33b26b5fbe7c2bbcfe",
		"getURL":                     "4e7e71e4b36e374a2fc6830f4f621ab3dd904cb2e1fa10b99e43a5590b7b4b36",
		"listURL":                    "0a55ee851552d789ddd3c12304e6d0d209cae0cfd477d71b138196681486bf9d",
		"rootURL":                    "4f185db1078eefd1b5f5b38bd1c601bd6fccaa3c20fa99a7b85371ef7f7b8104",
		"resourceURL":                "ca0246cf0e192c0133b2d43b9c5b577badf51c51344d3c5fc499d01a97ef54c8",
		"commonResult.Extract":       "9c467d68ba024143e8a8ba5a4a69db0e027a035507c38e1aa0387a13d1c191c9",
		"PolicyPage.NextPageURL":     "7b97a390dc18f3874d6a0bcca3e030ed8ad60129e7a6d8310227785a8d6602ca",
		"PolicyPage.IsEmpty":         "88fa833c9862dd1c989f138f4614c917e32483879512f3880ef9ea9dc3aeebd7",
		"ExtractPolicies":            "9be7267032ad765ebed4e8f5deb34f80b6804da02b42288ff362106f314789b6",
		"ExtractPolicysInto":         "8ac0e33d92c10eabc76680aa76b5dd845589c8c45adf242b0ec59ad14cba50de",
	},
	"network/v2/extensions/security/addressgroups": {
		"Get":                              "c0787d9b62a2cda3caa71d9a5deeaa3b99842417646def58aea2161caf892fb9",
		"List":                             "5df8dc6e3d372ae3b098dcf493fb5a64c82ba6c1dfdd25ca0671c132ded1251b",
		"ListOpts.ToAddressGroupListQuery": "9926b4a40d1b9c796d9a344a5935194b52681701b31be5eed66df3faa95be487",
		"rootURL":                          "5a4bf90be880f0165342169abba24c6938989c185637bae8e4c5092f33ac9f3b",
		"resourceURL":                      "ff31c93c543d54044b2c72a7219311759861582c91db4787e04a5f2d3dfc060c",
		"commonResult.Extract":             "e6cfb8e6b6b251cfcdb21e8017c976fa7cccd580d3cc8bd3319f5e1d5ee818db",
		"AddressGroupPage.NextPageURL":     "db55578ce9b76a89b003f2907d3491ba56684dafdac9c1e6fe343b5460ae92cd",
		"AddressGroupPage.IsEmpty":         "ba7bc94d31fad840c0a0c2437508ed26c1782391692a5852ef3ae2d3ed7c8054",
		"ExtractGroups":                    "d478953739195b0391c1228d9a5b5576638a3a96fbfb93fb41042920a47cfe30",
	},
}

var identityNativeURLConstants = map[string]map[string]string{
	"network/v2/extensions/qos/policies":           {"resourcePath": "qos/policies"},
	"network/v2/extensions/security/addressgroups": {"rootPath": "address-groups"},
	"network/v2/extensions/subnetpools":            {"resourcePath": "subnetpools"},
	"network/v2/extensions/trunks":                 {"resourcePath": "trunks"},
	"network/v2/extensions/layer3/routers":         {"resourcePath": "routers"},
	"network/v2/extensions/security/groups":        {"rootPath": "security-groups"},
	"loadbalancer/v2/pools":                        {"rootPath": "lbaas", "resourcePath": "pools", "memberPath": "members"},
	"identity/v3/roles":                            {"rolePath": "roles"},
}

// Export data omits private URL constants. Read their literal source values,
// rather than treating their absence from the imported type package as drift.
func identitySourceConstants(file *ast.File) map[string]string {
	values := map[string]string{}
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.CONST {
			continue
		}
		for _, entry := range group.Specs {
			spec, ok := entry.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range spec.Names {
				if i >= len(spec.Values) {
					continue
				}
				literal, ok := spec.Values[i].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				if value, err := strconv.Unquote(literal.Value); err == nil {
					values[name.Name] = value
				}
			}
		}
	}
	return values
}

func validateIdentityCollectionContracts(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan, scopes []scopePlan, constants map[string]string) error {
	for _, spec := range identityCollectionSpecs {
		if sdkPath(pkg.Path()) != spec.path {
			continue
		}
		selected := plan
		if spec.parents != 0 {
			selected = nil
			for _, scope := range scopes {
				if scope.collection != nil && scope.collection.modelName == spec.model {
					selected = scope.collection
				}
			}
		}
		if !identityCollectionEnabled(pkg, selected, spec.parents) {
			return fmt.Errorf("audited identity collection %s.%s: native signature, model, query, or result schema changed", spec.path, spec.model)
		}
		if spec.missingListKey != "" && !identityImageListSchema(pkg, selected) {
			return fmt.Errorf("audited identity collection %s.%s: native hidden query, image, pager, or extractor schema changed", spec.path, spec.model)
		}
		if spec.extraSpecs && !identityFlavorSchema(pkg, selected) {
			return fmt.Errorf("audited identity collection %s.%s: native access query, extra specs, pager, or extractor schema changed", spec.path, spec.model)
		}
		if (spec.path == "network/v2/extensions/layer3/routers" || spec.path == "network/v2/extensions/security/groups") && !identityNeutronListSchema(pkg, selected) {
			return fmt.Errorf("audited identity collection %s.%s: native concrete/builder list, pager, or extractor schema changed", spec.path, spec.model)
		}
		if (spec.path == "network/v2/extensions/subnetpools" || spec.path == "network/v2/extensions/trunks") && !identityPoolTrunkSchema(pkg, selected) {
			return fmt.Errorf("audited identity collection %s.%s: native prefix decoder, inherited pager, or extractor schema changed", spec.path, spec.model)
		}
		if spec.path == "network/v2/networks" && !identityNetworkListSchema(pkg, selected) {
			return fmt.Errorf("audited identity collection %s.%s: native network decoder, linked pager, or extractor schema changed", spec.path, spec.model)
		}
		if spec.path == "network/v2/subnets" && !identitySubnetBodySchema(pkg, selected) {
			return fmt.Errorf("audited identity collection %s.%s: native subnet model, decoder, linked pager, or extractor schema changed", spec.path, spec.model)
		}
		if spec.path == "network/v2/extensions/trunks" && decls["TrunkPage.NextPageURL"] != nil {
			return fmt.Errorf("audited identity collection %s.%s: native inherited continuation override changed", spec.path, spec.model)
		}
		if spec.path == "network/v2/extensions/qos/policies" || spec.path == "network/v2/extensions/security/addressgroups" {
			if !identityQoSAddressSchema(pkg, selected) || decls[spec.model+".UnmarshalJSON"] != nil {
				return fmt.Errorf("audited identity collection %s.%s: native model, decoder, linked pager, or extractor schema changed", spec.path, spec.model)
			}
		}
		for name, want := range identityNativeDeclarations[spec.path] {
			got, err := requestDeclarationHash(decls[name])
			if err != nil || got != want {
				return fmt.Errorf("audited identity collection %s.%s: pinned native declaration %s changed; review route and accepted codes", spec.path, spec.model, name)
			}
		}
		for name, want := range identityNativeURLConstants[spec.path] {
			if value, ok := constants[name]; !ok || value != want {
				return fmt.Errorf("audited identity collection %s.%s: pinned URL constant %s changed", spec.path, spec.model, name)
			}
		}
	}
	return nil
}

func emitIdentityFind(e *emitter, receiver, target, model string) {
	e.printf("// FindIdentity tries an ID request before exact ID/name fallback within this collection.\nfunc(%s)FindIdentity(ctx context.Context,identity string,options ...resource.IdentityFindOption)(*%s,error){return %s.FindIdentity(ctx,identity,options...)}\n", receiver, model, target)
}

// Only these two audited pagers expose details/all-projects identity controls.
// Other query-capable identity collections keep their original list contract.
type identityListModeSpec struct {
	path, model, iterator, page, extractor string
	lists                                  []string
	declarations                           map[string]string
}

var identityListModeSpecs = []identityListModeSpec{
	{path: "compute/v2/servers", model: "Server", iterator: "IterateServers", page: "ServerPage", extractor: "ExtractServers", lists: []string{"List", "ListSimple"}, declarations: map[string]string{
		"List":                   "debdc7a25f7ae93e59e6f2f8292a2402f8c3a7ee1b316e601365d3cf3757cafe",
		"ListSimple":             "204e88938805bcb7b556643cc82792cad58e07449ce7253467442b2f3df2de01",
		"listURL":                "b3320a7cc0d5105912c77f89d32f6c789b71dc26cef6f66149c00594686fff13",
		"listDetailURL":          "bf3e8432831c52c83f877d90c9a1370dcf8f4d15e7270c29a7753c9b7d0252ed",
		"createURL":              "0360b8ba1a61b2b662f07f10920ab4a0ad669d0162762be02d1822890d1ebf52",
		"ExtractServers":         "9f1a1e7bc37e31a989738ce59f6f3ff2a393103d8ba16c8a0d02285636bd4072",
		"ExtractServersInto":     "b46cf06bf3408c6b02b9c3610b39922ae6dda921393bb081806b6447669dfcf6",
		"ServerPage.IsEmpty":     "953b8c265c7b031f21a0c38efa5038b2bdf62b1fbf6e689ff1df10dfe80134f5",
		"ServerPage.NextPageURL": "ca478f6213cb598ee7b2d644bb6cac751c4d60dd63058626074a47615292af13",
	}},
	{path: "blockstorage/v3/volumes", model: "Volume", iterator: "IterateVolumes", page: "VolumePage", extractor: "ExtractVolumes", lists: []string{"List"}, declarations: map[string]string{
		"List":                   "416520c9ff811390c33f07194805b2f823f6ca9e7415dcf261017695b37b2209",
		"listURL":                "a173be7936d74d1d2dd2a156cf522bd73eb5a16db5b3be6830372258d37d5046",
		"createURL":              "6f9440e658f9b65c3c533e3e7d3d7913264a0bd7a7e63dc24953c44622ca79c8",
		"ExtractVolumes":         "0d4014c0aba5755840c37b5adef94262b95fe3c626ac6ba053e79ddaadb3256d",
		"ExtractVolumesInto":     "f061096c974f8c5029522f8e150fe42fcaf1b232cf49b1a3079173c97af68a65",
		"VolumePage.IsEmpty":     "8a770771b9f1951ea31737ea36c0bffefe53b02b150a93dd81919788d210f14e",
		"VolumePage.NextPageURL": "7286969be75fc92d1a471029cb95807be0373609b6b3cdd8a5cf1f68e42fa176",
	}},
}

func identityDeclarationKey(fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return fn.Name.Name
	}
	if len(fn.Recv.List) != 1 {
		return ""
	}
	receiver := fn.Recv.List[0].Type
	if pointer, ok := receiver.(*ast.StarExpr); ok {
		receiver = pointer.X
	}
	if named, ok := receiver.(*ast.Ident); ok {
		return named.Name + "." + fn.Name.Name
	}
	return ""
}

func identityListModeEnabled(pkg *types.Package, plan *collectionPlan) bool {
	_, ok := identityListModeContract(pkg, plan)
	return ok
}

func identityListModeContract(pkg *types.Package, plan *collectionPlan) (identityListModeSpec, bool) {
	if !identityCollectionEnabled(pkg, plan, 0) {
		return identityListModeSpec{}, false
	}
	for _, spec := range identityListModeSpecs {
		if spec.path != sdkPath(pkg.Path()) || plan.modelName != spec.model {
			continue
		}
		for _, name := range spec.lists {
			fn, ok := pkg.Scope().Lookup(name).(*types.Func)
			if !ok {
				return identityListModeSpec{}, false
			}
			sig := fn.Type().(*types.Signature)
			if sig.Variadic() || sig.Params().Len() != 2 || clientParam(sig) != 0 || sig.Results().Len() != 1 || !isPager(sig.Results().At(0).Type()) || !types.Identical(sig.Params().At(1).Type(), plan.lister.Type().(*types.Signature).Params().At(1).Type()) {
				return identityListModeSpec{}, false
			}
		}
		page := pkg.Scope().Lookup(spec.page)
		if page == nil {
			return identityListModeSpec{}, false
		}
		fields, ok := page.Type().Underlying().(*types.Struct)
		if !ok || fields.NumFields() != 1 || !fields.Field(0).Embedded() || types.TypeString(fields.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.LinkedPageBase" {
			return identityListModeSpec{}, false
		}
		extract, ok := pkg.Scope().Lookup(spec.extractor).(*types.Func)
		if !ok {
			return identityListModeSpec{}, false
		}
		sig := extract.Type().(*types.Signature)
		if sig.Variadic() || sig.Params().Len() != 1 || sig.Results().Len() != 2 || types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.Page" || !types.Identical(sig.Results().At(0).Type(), types.NewSlice(plan.model)) || !isError(sig.Results().At(1).Type()) {
			return identityListModeSpec{}, false
		}
		for name, result := range map[string]types.Type{"IsEmpty": types.Typ[types.Bool], "NextPageURL": types.Typ[types.String]} {
			method := extractionMethod(page.Type(), name)
			if method == nil || !types.Identical(method.Results().At(0).Type(), result) {
				return identityListModeSpec{}, false
			}
		}
		return spec, true
	}
	return identityListModeSpec{}, false
}

func validateIdentityListModeContracts(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	for _, spec := range identityListModeSpecs {
		if spec.path != sdkPath(pkg.Path()) {
			continue
		}
		if !identityListModeEnabled(pkg, plan) {
			return fmt.Errorf("audited identity list modes %s.%s: native pager, extractor, or list signature changed", spec.path, spec.model)
		}
		for name, want := range spec.declarations {
			got, err := requestDeclarationHash(decls[name])
			if err != nil || got != want {
				return fmt.Errorf("audited identity list modes %s.%s: pinned native declaration %s changed; review routes, pages and extraction", spec.path, spec.model, name)
			}
		}
	}
	return nil
}
