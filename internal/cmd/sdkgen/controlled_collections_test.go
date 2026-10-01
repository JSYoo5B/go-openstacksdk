package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func controlledFixtureSource(model, fields, getter, getArgs, lister, listArgs, opts string) string {
	return fmt.Sprintf(`package fixture
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
import "github.com/gophercloud/gophercloud/v2/pagination"
type %s struct{%s}
type GetResult struct{}
func(GetResult)Extract()(*%s,error){return nil,nil}
%s
type ModelPage struct{}
func %s(ctx context.Context,client *gophercloud.ServiceClient,%s)GetResult{return GetResult{}}
func %s(client *gophercloud.ServiceClient%s)pagination.Pager{_ = ModelPage{};return pagination.Pager{}}
func ExtractModels(p pagination.Page)([]%s,error){_ = p.(ModelPage);return nil,nil}
`, model, fields, model, opts, getter, getArgs, lister, listArgs, model)
}

func controlledEmittedMethod(t *testing.T, source []byte, name string) *ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "emitted.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Recv != nil && fn.Name.Name == name {
			return fn
		}
	}
	t.Fatalf("missing method %s:\n%s", name, source)
	return nil
}

func controlledCalls(fn *ast.FuncDecl) []string {
	var calls []string
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			calls = append(calls, nodeText(call))
		}
		return true
	})
	return calls
}

func requireControlledCalls(t *testing.T, fn *ast.FuncDecl, fragments ...string) {
	t.Helper()
	calls := strings.Join(controlledCalls(fn), "\n")
	for _, fragment := range fragments {
		if !strings.Contains(calls, fragment) {
			t.Fatalf("%s missing call %q:\n%s", fn.Name.Name, fragment, calls)
		}
	}
}

func emitControlledListFixture(t *testing.T, pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan, scopes []scopePlan, receiver string, parents []string) ([]byte, []byte) {
	t.Helper()
	extractors := extractorsByPage(pkg, decls)
	selected := collectionControlledLists(plan, scopes)
	var binding *collectionPlan
	if plan != nil {
		binding = plan
	} else {
		binding = scopes[0].collection
	}
	e := emitter{pkg: pkg, imports: map[string]string{}, controlledLists: selected}
	if err := emitOperation(&e, binding.lister, decls[binding.lister.Name()], extractors); err != nil {
		t.Fatal(err)
	}
	source, err := e.source()
	if err != nil {
		t.Fatal(err)
	}
	adapter := emitter{pkg: pkg, imports: map[string]string{}}
	adapter.printf("func(a *API)newResources()*resource.Collection[%s]{return ", binding.modelName)
	emitCollectionAdapter(&adapter, binding, receiver, parents)
	adapter.printf("}\n")
	adapterSource, err := adapter.source()
	if err != nil {
		t.Fatal(err)
	}
	return source, adapterSource
}

func TestControlledNativeNovaListPreservesOptionsExtractorAndRegexLookup(t *testing.T) {
	options := `type ListOpts struct{Name string ` + "`q:\"name\"`" + `;Status string ` + "`q:\"status\"`" + `}
type ListOptsBuilder interface{ToListQuery()(string,error)}
func(ListOpts)ToListQuery()(string,error){return "",nil}`
	source := controlledFixtureSource("Server", "ID string;Name string;Status string", "Get", "id string", "List", ",opts ListOptsBuilder", options)
	pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/compute/v2/servers", source)
	plan, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls))
	if err != nil || plan == nil {
		t.Fatalf("plan=%v err=%v", plan, err)
	}
	emitted, adapter := emitControlledListFixture(t, pkg, decls, plan, nil, "a", nil)
	public := controlledEmittedMethod(t, emitted, "List")
	if len(public.Body.List) != 1 || len(public.Type.Params.List) != 2 || nodeText(public.Type.Params.List[1].Type) != "...ListOption" {
		t.Fatalf("public List signature or body changed: %s", emitted)
	}
	requireControlledCalls(t, public, "a.listWithControl(ctx, resource.ListControl{}, options...)")
	private := controlledEmittedMethod(t, emitted, "listWithControl")
	requireControlledCalls(t, private, "request.Apply(opts, options...)", "request.ValidateCapabilities(cfg, false, true, false)", `request.Wrap("List", "fixture", err)`, "upstream.List(a.client, _opts)", "upstream.ExtractModels(page)", "resource.StreamWithControl(ctx,")
	if strings.Count(string(emitted), "request.Apply(") != 1 || nodeText(private.Type.Params.List[1].Type) != "resource.ListControl" {
		t.Fatalf("options duplicated or control signature changed: %s", emitted)
	}
	for _, fragment := range []string{`regexp.QuoteMeta(name)`, `IterateControlled:`, `WithListQuery(key, value)`, `a.listWithControl(ctx, control, options...)`, `q = maps.Clone(q)`} {
		if !strings.Contains(string(adapter), fragment) {
			t.Fatalf("Nova binding lost %q:\n%s", fragment, adapter)
		}
	}
}

func TestControlledNativeDesignateListPreservesOptionalHeaderBuilder(t *testing.T) {
	options := `type ListOpts struct{Name string ` + "`q:\"name\"`" + `;AllProjects bool ` + "`h:\"X-Auth-All-Projects\"`" + `;SudoTenantID string ` + "`h:\"X-Auth-Sudo-Tenant-ID\"`" + `}
type ListOptsBuilder interface{ToZoneListQuery()(string,error)}
type ListOptsHeadersBuilder interface{ToZoneListHeaders()(map[string]string,error)}
func(ListOpts)ToZoneListQuery()(string,error){return "",nil}
func(ListOpts)ToZoneListHeaders()(map[string]string,error){return nil,nil}`
	source := controlledFixtureSource("Zone", "ID string;Name string;Status string", "Get", "id string", "List", ",opts ListOptsBuilder", options)
	source = strings.Replace(source, "_ = ModelPage{};", "_, _ = opts.(ListOptsHeadersBuilder);_ = ModelPage{};", 1)
	pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/dns/v2/zones", source)
	plan := identifyCollection(pkg, decls, extractorsByPage(pkg, decls))
	if plan == nil {
		t.Fatal("missing Designate collection")
	}
	emitted, adapter := emitControlledListFixture(t, pkg, decls, plan, nil, "a", nil)
	private := controlledEmittedMethod(t, emitted, "listWithControl")
	requireControlledCalls(t, private, "request.ValidateCapabilities(cfg, false, true, true)", "upstream.List(a.client, _opts)")
	for _, fragment := range []string{"func WithListHeader(", "ToZoneListHeaders()", "b.base.ToZoneListHeaders()", "request.MergeHeadersFor(value0, b.config.Headers, b.base)", "request.ExtendQuery(value0, b.config.Query)"} {
		if !strings.Contains(string(emitted), fragment) {
			t.Fatalf("Designate lost %q:\n%s", fragment, emitted)
		}
	}
	if !strings.Contains(string(adapter), "a.listWithControl(ctx, control, options...)") || !strings.Contains(string(adapter), `q.Del("status")`) {
		t.Fatalf("Designate binding lost query/local status semantics:\n%s", adapter)
	}
}

func TestControlledNativeOctaviaMembersKeepFixedParentAndLocalStatus(t *testing.T) {
	options := `type ListMembersOpts struct{Name string ` + "`q:\"name\"`" + `}
type ListMembersOptsBuilder interface{ToMembersListQuery()(string,error)}
func(ListMembersOpts)ToMembersListQuery()(string,error){return "",nil}`
	source := controlledFixtureSource("Member", "ID string;Name string;ProvisioningStatus string", "GetMember", "poolID,id string", "ListMembers", ",poolID string,opts ListMembersOptsBuilder", options)
	pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/loadbalancer/v2/pools", source)
	scopes, err := identifyScopes(pkg, decls, extractorsByPage(pkg, decls))
	if err != nil || len(scopes) != 1 {
		t.Fatalf("scopes=%v err=%v", scopes, err)
	}
	emitted, adapter := emitControlledListFixture(t, pkg, decls, nil, scopes, "s.api", []string{"s.parentID"})
	public := controlledEmittedMethod(t, emitted, "ListMembers")
	requireControlledCalls(t, public, "a.listMembersWithControl(ctx, poolID, resource.ListControl{}, options...)")
	private := controlledEmittedMethod(t, emitted, "listMembersWithControl")
	requireControlledCalls(t, private, "request.Apply(opts, options...)", "upstream.ListMembers(a.client, poolID, _opts)", "resource.StreamWithControl(ctx,")
	if nodeText(private.Type.Params.List[1].Type) != "string" || nodeText(private.Type.Params.List[2].Type) != "resource.ListControl" {
		t.Fatalf("parent/control signature changed:\n%s", emitted)
	}
	for _, fragment := range []string{`s.api.GetMember(ctx, s.parentID, string(id))`, `s.api.listMembersWithControl(ctx, s.parentID, control, options...)`, `q.Del("status")`, `v.ProvisioningStatus`} {
		if !strings.Contains(string(adapter), fragment) {
			t.Fatalf("member scope lost %q:\n%s", fragment, adapter)
		}
	}
	if strings.Contains(string(adapter), "s.api.ListMembers(") {
		t.Fatalf("member scope bypasses controls:\n%s", adapter)
	}
}

func TestControlledNativeCollectionsPreserveConcreteAndOptionlessLists(t *testing.T) {
	for _, options := range []bool{false, true} {
		t.Run(fmt.Sprint("concrete-options-", options), func(t *testing.T) {
			listArgs, opts := "", ""
			if options {
				listArgs = ",opts ListOpts"
				opts = `type ListOpts struct{Name string ` + "`q:\"display_name\"`" + `;Status string ` + "`q:\"state\"`" + `}`
			}
			source := controlledFixtureSource("Thing", "ID string;Name string;Status string", "Get", "id string", "List", listArgs, opts)
			pkg, decls := typedCollectionFixture(t, "fixture", source)
			plan := identifyCollection(pkg, decls, extractorsByPage(pkg, decls))
			if plan == nil || plan.deleter != nil {
				t.Fatalf("readonly plan=%+v", plan)
			}
			emitted, adapter := emitControlledListFixture(t, pkg, decls, plan, nil, "a", nil)
			private := controlledEmittedMethod(t, emitted, "listWithControl")
			if options {
				for _, fragment := range []string{`q.Set("display_name", value)`, `q.Set("state", value)`, "request.QueryOptions[ListOpts](q)", "a.listWithControl(ctx, control, WithListOptions(input))"} {
					if !strings.Contains(string(adapter), fragment) {
						t.Fatalf("concrete options lost %q:\n%s", fragment, adapter)
					}
				}
				requireControlledCalls(t, private, "request.Apply(opts, options...)", "upstream.List(a.client, cfg.Options)")
			} else {
				requireControlledCalls(t, controlledEmittedMethod(t, emitted, "List"), "a.listWithControl(ctx, resource.ListControl{})")
				requireControlledCalls(t, private, "upstream.List(a.client)")
				if !strings.Contains(string(adapter), "if len(q) != 0") || !strings.Contains(string(adapter), "resource.ErrUnsupported") || !strings.Contains(string(adapter), "a.listWithControl(ctx, control)") {
					t.Fatalf("optionless query rejection/control lost:\n%s", adapter)
				}
			}
			if strings.Contains(string(adapter), "Delete:") || strings.Contains(string(adapter), "Iterate:") {
				t.Fatalf("read-only capabilities changed or adapter duplicated:\n%s", adapter)
			}
		})
	}
}

func TestControlledNativePlannerDoesNotChangeUnboundValueOrPageStreams(t *testing.T) {
	source := controlledFixtureSource("Thing", "ID string", "Get", "id string", "List", "", "") + `
type ValuePage struct{}
type RawPage struct{}
func ExtractValues(p pagination.Page)(map[string]string,error){_ = p.(ValuePage);return nil,nil}
func ListUnbound(client *gophercloud.ServiceClient)pagination.Pager{_ = ModelPage{};return pagination.Pager{}}
func ListValues(client *gophercloud.ServiceClient)pagination.Pager{_ = ValuePage{};return pagination.Pager{}}
func ListRaw(client *gophercloud.ServiceClient)pagination.Pager{_ = RawPage{};return pagination.Pager{}}
`
	pkg, decls := typedCollectionFixture(t, "fixture", source)
	extractors := extractorsByPage(pkg, decls)
	plan := identifyCollection(pkg, decls, extractors)
	selected := collectionControlledLists(plan, nil)
	if len(selected) != 1 || !selected["List"] {
		t.Fatalf("controlled=%v", selected)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}, controlledLists: selected}
	for _, name := range []string{"List", "ListUnbound", "ListValues", "ListRaw"} {
		if err := emitOperation(&e, pkg.Scope().Lookup(name).(*types.Func), decls[name], extractors); err != nil {
			t.Fatal(err)
		}
	}
	emitted, err := e.source()
	if err != nil {
		t.Fatal(err)
	}
	requireControlledCalls(t, controlledEmittedMethod(t, emitted, "ListUnbound"), "resource.Stream(ctx,")
	requireControlledCalls(t, controlledEmittedMethod(t, emitted, "ListValues"), "resource.StreamValues(ctx,")
	requireControlledCalls(t, controlledEmittedMethod(t, emitted, "ListRaw"), "resource.Pages(ctx,")
	if strings.Count(string(emitted), "func (a *API) listWithControl(") != 1 || strings.Contains(string(emitted), "listUnboundWithControl") || strings.Contains(string(emitted), "listValuesWithControl") || strings.Contains(string(emitted), "listRawWithControl") {
		t.Fatalf("controlled methods leaked beyond planned slice pager:\n%s", emitted)
	}
}
