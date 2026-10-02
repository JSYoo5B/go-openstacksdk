package main

import (
	"encoding/json"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type collectionRecord struct {
	Package              string            `json:"package"`
	Source               string            `json:"source,omitempty"`
	Model                string            `json:"model,omitempty"`
	UpstreamModel        string            `json:"upstream_model,omitempty"`
	Kind                 string            `json:"kind,omitempty"`
	Find                 bool              `json:"find"`
	IdentityFind         bool              `json:"identity_find,omitempty"`
	IdentityGetQuery     bool              `json:"identity_get_query,omitempty"`
	IdentityMissingList  bool              `json:"identity_missing_list,omitempty"`
	IdentityListDefaults bool              `json:"identity_list_defaults,omitempty"`
	IdentityExtraSpecs   bool              `json:"identity_extra_specs,omitempty"`
	IdentityDetails      bool              `json:"identity_details,omitempty"`
	IdentityAllProjects  bool              `json:"identity_all_projects,omitempty"`
	BodyFilterFields     []string          `json:"body_filter_fields,omitempty"`
	SemanticQueryFilters map[string]string `json:"semantic_query_filters,omitempty"`
	SemanticBodyFilters  map[string]string `json:"semantic_body_filters,omitempty"`
	SemanticReserved     []string          `json:"semantic_reserved,omitempty"`
	Delete               bool              `json:"delete"`
	Wait                 bool              `json:"wait"`
	Scope                string            `json:"scope,omitempty"`
	Parent               string            `json:"parent,omitempty"`
	Issue                string            `json:"issue,omitempty"`
}

type collectionPlan struct {
	model                       types.Type
	modelName, id, name, status string
	getter, lister, deleter     *types.Func
	listInput                   types.Type
	listQueryBuilder            bool
	nameQuery, statusQuery      string
	idIsURL                     bool
	getIDType, deleteIDType     types.Type
}

// Only collection-planned typed slice pagers expose row/page controls. Other
// native streams, including value streams and raw pages, retain their API.
func collectionControlledLists(plan *collectionPlan, scopes []scopePlan) map[string]bool {
	result := make(map[string]bool)
	if plan != nil {
		result[plan.lister.Name()] = true
	}
	for _, scope := range scopes {
		result[scope.collection.lister.Name()] = true
	}
	return result
}

func controlledListName(operation string) string { return lower(operation) + "WithControl" }

func simpleInput(fn *types.Func, ids int) (types.Type, bool) {
	if fn == nil {
		return nil, false
	}
	sig := fn.Type().(*types.Signature)
	count := 0
	var options types.Type
	for i := 0; i < sig.Params().Len(); i++ {
		if i == clientParam(sig) || isContext(sig.Params().At(i).Type()) {
			continue
		}
		t := sig.Params().At(i).Type()
		if isString(t) || isInteger(t) {
			count++
			continue
		}
		if options != nil {
			return nil, false
		}
		if iface, ok := ifaceOf(t); ok {
			builder := false
			for j := 0; j < iface.NumMethods(); j++ {
				if strings.HasPrefix(iface.Method(j).Name(), "To") {
					builder = true
				}
			}
			if !builder {
				return nil, false
			}
			options = t
		} else {
			base := t
			if pointer, ok := base.(*types.Pointer); ok {
				base = pointer.Elem()
			}
			named, ok := base.(*types.Named)
			if !ok || !strings.HasSuffix(named.Obj().Name(), "Opts") {
				return nil, false
			}
			options = t
		}
	}
	return options, count == ids
}

func isInteger(t types.Type) bool {
	base, ok := t.Underlying().(*types.Basic)
	return ok && base.Info()&types.IsInteger != 0
}
func identifierType(fn *types.Func) types.Type {
	sig := fn.Type().(*types.Signature)
	for i := sig.Params().Len() - 1; i >= 0; i-- {
		t := sig.Params().At(i).Type()
		if isString(t) || isInteger(t) {
			return t
		}
	}
	return nil
}

func field(t types.Type, names ...string) string {
	for _, name := range names {
		obj, _, _ := types.LookupFieldOrMethod(t, true, nil, name)
		variable, ok := obj.(*types.Var)
		if !ok || !variable.IsField() {
			continue
		}
		if basic, ok := variable.Type().Underlying().(*types.Basic); ok && basic.Info()&(types.IsString|types.IsInteger) != 0 {
			return name
		}
	}
	return ""
}

func queryTag(t types.Type, name string) string {
	if pointer, ok := t.(*types.Pointer); ok {
		t = pointer.Elem()
	}
	structure, ok := t.Underlying().(*types.Struct)
	if !ok {
		return ""
	}
	for i := 0; i < structure.NumFields(); i++ {
		if structure.Field(i).Name() == name {
			return strings.Split(reflect.StructTag(structure.Tag(i)).Get("q"), ",")[0]
		}
		if structure.Field(i).Anonymous() {
			if tag := queryTag(structure.Field(i).Type(), name); tag != "" {
				return tag
			}
		}
	}
	return ""
}

func identifyCollection(pkg *types.Package, decls map[string]*ast.FuncDecl, extractors map[string]string) *collectionPlan {
	return identifyNamedCollection(pkg, decls, extractors, "Get", []string{"ListDetail", "List"}, "Delete", 0)
}

func identifyNamedCollection(pkg *types.Package, decls map[string]*ast.FuncDecl, extractors map[string]string, getter string, listers []string, deleter string, parents int, identifiers ...string) *collectionPlan {
	get, ok := pkg.Scope().Lookup(getter).(*types.Func)
	if !ok {
		return nil
	}
	if _, ok := simpleInput(get, parents+1); !ok {
		return nil
	}
	getSig := get.Type().(*types.Signature)
	if getSig.Results().Len() != 1 {
		return nil
	}
	_, extract := operationExtractor(get)
	if extract == nil || extract.Results().Len() != 2 {
		return nil
	}
	pointer, ok := extract.Results().At(0).Type().(*types.Pointer)
	if !ok {
		return nil
	}
	model := pointer.Elem()
	modelName := ""
	for _, name := range pkg.Scope().Names() {
		if obj, ok := pkg.Scope().Lookup(name).(*types.TypeName); ok && obj.Exported() && types.Identical(obj.Type(), model) {
			modelName = name
			break
		}
	}
	if modelName == "" {
		return nil
	}
	if len(identifiers) == 0 {
		identifiers = []string{"ID", "UUID", "SecretRef", "OrderRef", "ContainerRef", "MemberID", "PortID", "Access", "Name"}
	}
	id := field(model, identifiers...)
	if id == "ID" && isString(identifierType(get)) && field(model, "UUID") != "" {
		member, _, _ := types.LookupFieldOrMethod(model, true, nil, "ID")
		if isInteger(member.Type()) {
			id = "UUID"
		}
	}
	if id == "" {
		return nil
	}
	plan := &collectionPlan{model: model, modelName: modelName, id: id, name: field(model, "Name", "Hostname"), status: field(model, "ProvisioningStatus", "Status", "ProvisionState", "PortState"), getter: get, idIsURL: strings.HasSuffix(id, "Ref")}
	plan.getIDType = identifierType(get)
	for _, name := range listers {
		list, ok := pkg.Scope().Lookup(name).(*types.Func)
		if !ok {
			continue
		}
		options, ok := simpleInput(list, parents)
		if !ok {
			continue
		}
		sig := list.Type().(*types.Signature)
		if sig.Results().Len() != 1 || !isPager(sig.Results().At(0).Type()) {
			continue
		}
		_, typ := findExtractor(pkg, name, decls[name], extractors)
		if typ == nil {
			continue
		}
		slice, ok := typ.Underlying().(*types.Slice)
		if !ok || !types.Identical(slice.Elem(), model) {
			continue
		}
		plan.lister = list
		if options != nil {
			if iface, ok := ifaceOf(options); ok {
				base, err := concrete(pkg, options)
				if err != nil {
					continue
				}
				plan.listInput = base
				plan.listQueryBuilder = capabilities(pkg, builder{iface: iface}).query
			} else {
				plan.listInput = options
			}
			plan.nameQuery = queryTag(plan.listInput, "Name")
			plan.statusQuery = queryTag(plan.listInput, "Status")
			if plan.statusQuery == "" {
				plan.statusQuery = queryTag(plan.listInput, "ProvisioningStatus")
			}
		}
		break
	}
	if plan.lister == nil {
		return nil
	}
	if del, ok := pkg.Scope().Lookup(deleter).(*types.Func); ok {
		if _, ok := simpleInput(del, parents+1); ok {
			policy := returnPolicy(del.Type().(*types.Signature))
			if policy == "error" || policy == "extract" || policy == "direct-error" {
				plan.deleter = del
				plan.deleteIDType = identifierType(del)
			}
		}
	}
	return plan
}

func (g *generator) emitCollection(pkg *types.Package, plan *collectionPlan) error {
	e := emitter{pkg: pkg, imports: map[string]string{}, pythonFilters: g.pythonFilterFor(pkg, plan)}
	e.printf("// Resources applies the SDK's shared lookup, missing-resource and wait policies.\nfunc(a *API)newResources()*resource.Collection[%s]{return ", plan.modelName)
	emitCollectionAdapter(&e, plan, "a", nil)
	e.printf("}\n")
	emitBodyFilterList(&e, plan)
	e.printf("func(a *API)Find(ctx context.Context,ref resource.Ref,options ...resource.LookupOption)(*%s,error){return a.Resources.Find(ctx,ref,options...)}\n", plan.modelName)
	if identityCollectionEnabled(pkg, plan, 0) {
		emitIdentityFind(&e, "a *API", "a.Resources", plan.modelName)
	}
	e.printf("func(a *API)All(ctx context.Context,options ...resource.ListOption)([]*%s,error){return a.Resources.All(ctx,options...)}\n", plan.modelName)
	e.printf("func(a *API)Remove(ctx context.Context,ref resource.Ref,options ...resource.LookupOption)error{return a.Resources.Delete(ctx,ref,options...)}\n")
	e.printf("func(a *API)WaitFor(ctx context.Context,ref resource.Ref,status string,options ...resource.WaitOption)(*%s,error){return a.Resources.Wait(ctx,ref,status,options...)}\n", plan.modelName)
	e.printf("func(a *API)WaitForDeletion(ctx context.Context,ref resource.Ref,options ...resource.WaitOption)error{return a.Resources.WaitDeleted(ctx,ref,options...)}\n")
	source, err := e.source()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(g.root, sdkPath(pkg.Path()), "resources_generated.go"), source, 0644)
}

// Both global and parent-bound resources use exactly the same adapter policies.
func emitCollectionAdapter(e *emitter, plan *collectionPlan, receiver string, parents []string) {
	e.use("context")
	e.use("net/url")
	e.use("iter")
	e.use("gophercloudsdk/resource")
	e.use("fmt")
	if plan.status != "" {
		e.use("strings")
	}
	arguments := func(id string) string { return strings.Join(append(append([]string{"ctx"}, parents...), id), ",") }
	e.printf("resource.NewCollection(resource.Adapter[%s]{\nKind:%q,\n", plan.modelName, e.pkg.Name())
	emitBodyFilterCollection(e, plan, len(parents))
	emitPythonFilterDescriptor(e, plan, len(parents))
	if contract, ok := identityCollectionContract(e.pkg, plan, len(parents)); ok {
		e.printf("IdentityFind:true,\n")
		emitIdentityGetQuery(e, plan, receiver, parents, contract)
		if contract.missingListKey != "" {
			e.printf("IdentityMissingListQuery:url.Values{%q:{%q}},\n", contract.missingListKey, contract.missingListValue)
		}
		if contract.extraSpecs {
			e.printf("IdentityListQueryDefaults:url.Values{%q:{%q}},\nIdentityExtraSpecs:func(ctx context.Context,value *%s)(*%s,error){return nativefind.FlavorExtraSpecs(ctx,%s.RawClient(),value)},\n", contract.listDefaultKey, contract.listDefaultValue, plan.modelName, plan.modelName, receiver)
		}
		if len(parents) == 0 {
			if mode, ok := identityListModeContract(e.pkg, plan); ok {
				e.printf("IdentityAllProjectsQuery:\"all_tenants\",\nIterateIdentity:func(ctx context.Context,q url.Values,details bool)iter.Seq2[*%s,error]{return nativefind.%s(ctx,%s.RawClient(),q,details)},\n", plan.modelName, mode.iterator, receiver)
			}
		}
	}
	e.printf("Get:func(ctx context.Context,id string)(*%s,error){", plan.modelName)
	if isInteger(plan.getIDType) {
		e.use("gophercloudsdk/request")
		e.printf("parsed,err:=request.NumericID[%s](id);if err!=nil{return nil,err};return %s.%s(%s)},\n", e.typ(plan.getIDType), receiver, plan.getter.Name(), arguments("parsed"))
	} else {
		e.printf("return %s.%s(%s)},\n", receiver, plan.getter.Name(), arguments(e.typ(plan.getIDType)+"(id)"))
	}
	if plan.idIsURL {
		e.use("path")
		e.printf("ID:func(v *%s)string{parsed,err:=url.Parse(v.%s);if err!=nil{return \"\"};return path.Base(parsed.Path)},\n", plan.modelName, plan.id)
	} else {
		e.printf("ID:func(v *%s)string{return fmt.Sprint(v.%s)},\n", plan.modelName, plan.id)
	}
	if plan.name != "" {
		e.printf("Name:func(v *%s)string{return fmt.Sprint(v.%s)},\n", plan.modelName, plan.name)
		if plan.nameQuery != "" {
			if strings.HasSuffix(e.pkg.Path(), "/compute/v2/servers") {
				e.use("regexp")
				e.printf("NameQuery:func(name string)string{return \"^\"+regexp.QuoteMeta(name)+\"$\"},\n")
			} else {
				e.printf("NameQuery:func(name string)string{return name},\n")
			}
		}
	}
	if plan.status != "" {
		e.printf("Status:func(v *%s)string{return fmt.Sprint(v.%s)},\nFailed:func(status string)bool{status=strings.ToLower(status);return strings.HasPrefix(status,\"error\")||strings.HasSuffix(status,\"fail\")||strings.HasSuffix(status,\"failed\")||status==\"killed\"},\n", plan.modelName, plan.status)
		if identityCollectionEnabled(e.pkg, plan, len(parents)) && plan.statusQuery == "" {
			e.printf("LocalStatus:true,\n")
		}
	}
	if plan.deleter != nil {
		policy := returnPolicy(plan.deleter.Type().(*types.Signature))
		e.printf("Delete:func(ctx context.Context,id string)error{")
		idArg := e.typ(plan.deleteIDType) + "(id)"
		if isInteger(plan.deleteIDType) {
			e.use("gophercloudsdk/request")
			e.printf("parsed,err:=request.NumericID[%s](id);if err!=nil{return err};", e.typ(plan.deleteIDType))
			idArg = "parsed"
		}
		if policy == "extract" {
			e.printf("_,err:=%s.%s(%s);return err},\n", receiver, plan.deleter.Name(), arguments(idArg))
		} else {
			e.printf("return %s.%s(%s)},\n", receiver, plan.deleter.Name(), arguments(idArg))
		}
	}
	e.printf("IterateControlled:func(ctx context.Context,q url.Values,control resource.ListControl)iter.Seq2[*%s,error]{\n", plan.modelName)
	e.use("maps")
	e.printf("q=maps.Clone(q)\n")
	if plan.nameQuery != "" && plan.nameQuery != "name" {
		e.printf("if value:=q.Get(\"name\");value!=\"\"{q.Set(%q,value);q.Del(\"name\")}\n", plan.nameQuery)
	}
	if plan.statusQuery == "" {
		if !identityCollectionEnabled(e.pkg, plan, len(parents)) {
			e.printf("q.Del(\"status\")\n")
		}
	} else if plan.statusQuery != "status" {
		e.printf("if value:=q.Get(\"status\");value!=\"\"{q.Set(%q,value);q.Del(\"status\")}\n", plan.statusQuery)
	}
	list := plan.lister.Name()
	listArgs := strings.Join(append(append([]string{"ctx"}, parents...), "control"), ",")
	if contract, ok := identityCollectionContract(e.pkg, plan, len(parents)); ok && contract.rawListIterator != "" {
		e.use("gophercloudsdk/internal/nativefind")
		e.printf("return nativefind.%s(ctx,%s.RawClient(),q,control)\n", contract.rawListIterator, receiver)
	} else if plan.listInput == nil {
		e.printf("if len(q)!=0{return func(yield func(*%s,error)bool){yield(nil,resource.ErrUnsupported)}}\nreturn %s.%s(%s)\n", plan.modelName, receiver, controlledListName(list), listArgs)
	} else if plan.listQueryBuilder {
		if identityCollectionEnabled(e.pkg, plan, len(parents)) || sdkPath(e.pkg.Path()) == "keymanager/v1/secrets" {
			// FindIdentity can retain repeated query values and a present nil
			// name key. A sequence of WithQuery options would Set each value
			// and collapse that input before the native builder sees it.
			e.use("gophercloudsdk/request")
			e.printf("options:=[]%sOption{func(config *request.Config[%s])error{config.Query=make(url.Values,len(q));for key,values:=range q{config.Query[key]=append([]string(nil),values...)};return nil}}\nreturn %s.%s(%s,options...)\n", list, e.typ(plan.listInput), receiver, controlledListName(list), listArgs)
		} else {
			e.printf("options:=make([]%sOption,0,len(q))\nfor key,values:=range q{for _,value:=range values{options=append(options,With%sQuery(key,value))}}\nreturn %s.%s(%s,options...)\n", list, list, receiver, controlledListName(list), listArgs)
		}
	} else {
		e.use("gophercloudsdk/request")
		e.printf("input,err:=request.QueryOptions[%s](q)\nif err!=nil{return func(yield func(*%s,error)bool){yield(nil,err)}}\nreturn %s.%s(%s,With%sOptions(input))\n", e.typ(plan.listInput), plan.modelName, receiver, controlledListName(list), listArgs, list)
	}
	e.printf("},})")
}

func (g *generator) writeCollectionInventory() error {
	data, err := json.MarshalIndent(g.collections, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(g.root, "api", "resource_inventory.json"), append(data, '\n'), 0644)
}
