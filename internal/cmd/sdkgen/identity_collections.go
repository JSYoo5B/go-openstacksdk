package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
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
}

func identityCollectionEnabled(pkg *types.Package, plan *collectionPlan, parents int) bool {
	_, ok := identityCollectionContract(pkg, plan, parents)
	return ok
}

func identityCollectionContract(pkg *types.Package, plan *collectionPlan, parents int) (identityCollectionSpec, bool) {
	if plan == nil || plan.getter == nil || plan.lister == nil || plan.id != "ID" || plan.name != "Name" || plan.idIsURL || !isString(plan.getIDType) || plan.listInput == nil || !plan.listQueryBuilder || plan.nameQuery != "name" {
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
			if !identityMissingListMetadataValid(spec) {
				return identityCollectionSpec{}, false
			}
			return spec, true
		}
	}
	return identityCollectionSpec{}, false
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
	"network/v2/networks":     {"Get": identityNetworkGetSHA, "getURL": identityNetworkURLSHA, "resourceURL": "313da020f0b09244e553dac107ce73d06de65812ce24c0339473cc8befd6332a"},
	"network/v2/subnets":      {"Get": identityNetworkGetSHA, "getURL": identityNetworkURLSHA, "resourceURL": "0fffec6b477ce31fc27ed1dc2ab5c205c5290d79980b3aa259b179ee7766b263"},
	"identity/v3/projects":    {"Get": identityDefaultGetSHA, "getURL": "d5e4289c71c7a028ecb7fe6a2a47b1e4a8f6438bdaccbc17c7fe1469bc0461f6"},
	"identity/v3/users":       {"Get": identityDefaultGetSHA, "getURL": "d3b727df4f5525b08c0dc43b6cb255bb5a6bb1b53da4c3fcbad076787334998e"},
	"identity/v3/groups":      {"Get": identityDefaultGetSHA, "getURL": "0b7a41e86f5eb5dafbcb193d202162b4385462274178934a7ec20f7efbe9ddbe"},
	"identity/v3/domains":     {"Get": identityDefaultGetSHA, "getURL": "4eed19909e41e8b752adbb281428ba5a3418746dbf5e5410e20078841b9a7475"},
	"identity/v3/roles":       {"Get": identityDefaultGetSHA, "getURL": "17a433f86f71243f80bdd8826fe2bc785f8950a1c6a166fedd2a171bc2c2c95d"},
	"dns/v2/recordsets":       {"Get": "045a8befff06647b70ecbe4560f7a809c991290496b3032cebd876a412a90bd4", "rrsetURL": "71398da5ec5a9461f7e0d9ebc8e569b7c27f09d0f2389191a87e0eca75c9a348"},
	"loadbalancer/v2/pools":   {"GetMember": "9173191d1b3baa7d542d364f4efaa137fdfa84bfe9ee921feb4f09569c384be0", "memberResourceURL": "0ac8d3833de97abc76ece2b63ad0e265e7ed2407a79ed6c75307d8e8538ded9e"},
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
}

var identityNativeURLConstants = map[string]map[string]string{
	"loadbalancer/v2/pools": {"rootPath": "lbaas", "resourcePath": "pools", "memberPath": "members"},
	"identity/v3/roles":     {"rolePath": "roles"},
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
