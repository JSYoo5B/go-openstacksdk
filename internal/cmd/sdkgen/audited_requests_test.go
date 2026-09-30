package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// This is the inspected v2.15.0 declaration, including the discarded query.
// A future native fix must trigger a new audit rather than retain this override.
const pinnedIntrospectionStartDeclaration = `func StartIntrospection(ctx context.Context, client *gophercloud.ServiceClient, nodeID string, opts StartOptsBuilder) (r StartResult) {
	_, err := opts.ToStartIntrospectionQuery()
	if err != nil {
		r.Err = err
		return
	}
	resp, err := client.Post(ctx, introspectionURL(client, nodeID), nil, nil, &gophercloud.RequestOpts{OkCodes: []int{202}})
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`

const introspectionStartFixtureSource = `package introspection
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
type StartOpts struct{ManageBoot *bool ` + "`q:\"manage_boot\"`" + `}
type StartOptsBuilder interface{ToStartIntrospectionQuery()(string,error)}
func(StartOpts)ToStartIntrospectionQuery()(string,error){return "",nil}
type StartResult struct{gophercloud.ErrResult}
func StartIntrospection(ctx context.Context,client *gophercloud.ServiceClient,nodeID string,opts StartOptsBuilder)(r StartResult){return}
func AbortIntrospection(ctx context.Context,client *gophercloud.ServiceClient,nodeID string)(r StartResult){return}
`

func introspectionStartFixture(t *testing.T, source, declaration string) (*types.Package, map[string]*ast.FuncDecl) {
	t.Helper()
	cloud := types.NewPackage(upstreamModule, "gophercloud")
	client := types.NewNamed(types.NewTypeName(token.NoPos, cloud, "ServiceClient", nil), types.NewStruct(nil, nil), nil)
	cloud.Scope().Insert(client.Obj())
	result := types.NewNamed(types.NewTypeName(token.NoPos, cloud, "ErrResult", nil), types.NewStruct(nil, nil), nil)
	result.AddMethod(types.NewFunc(token.NoPos, cloud, "ExtractErr", types.NewSignatureType(types.NewVar(token.NoPos, cloud, "r", result), nil, nil, types.NewTuple(), types.NewTuple(types.NewVar(token.NoPos, cloud, "", types.Universe.Lookup("error").Type())), false)))
	cloud.Scope().Insert(result.Obj())
	cloud.MarkComplete()
	contexts := types.NewPackage("context", "context")
	contextType := types.NewNamed(types.NewTypeName(token.NoPos, contexts, "Context", nil), types.NewInterfaceType(nil, nil).Complete(), nil)
	contexts.Scope().Insert(contextType.Obj())
	contexts.MarkComplete()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{Importer: packageImports{upstreamModule: cloud, "context": contexts}}
	pkg, err := config.Check(upstreamModule+"/openstack/baremetalintrospection/v1/introspection", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, node := range file.Decls {
		if decl, ok := node.(*ast.FuncDecl); ok && decl.Recv == nil {
			decls[decl.Name.Name] = decl
		}
	}
	if decls["StartIntrospection"] != nil {
		native, err := parser.ParseFile(token.NewFileSet(), "native.go", "package introspection\n"+declaration, 0)
		if err != nil {
			t.Fatal(err)
		}
		decls["StartIntrospection"] = native.Decls[0].(*ast.FuncDecl)
	}
	return pkg, decls
}

func TestAuditedStartCallPreservesOnlyTheInspectedRequestPolicy(t *testing.T) {
	pkg, decls := introspectionStartFixture(t, introspectionStartFixtureSource, pinnedIntrospectionStartDeclaration)
	if err := validateAuditedRequestCalls(pkg, decls); err != nil {
		t.Fatal(err)
	}
	override := requestCallOverride(pkg, "StartIntrospection")
	if override == nil || override.helper != "startIntrospection" || override.policy != "sdk_query_preserving_start" || requestCallOverride(pkg, "AbortIntrospection") != nil {
		t.Fatalf("override=%+v", override)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}}
	for _, name := range []string{"StartIntrospection", "AbortIntrospection"} {
		fn := pkg.Scope().Lookup(name).(*types.Func)
		if err := emitOperation(&e, fn, decls[name], nil); err != nil {
			t.Fatal(err)
		}
	}
	generated := e.body.String()
	for _, expected := range []string{
		"startIntrospection(ctx,a.client,nodeID,_opts).ExtractErr()",
		"upstream.AbortIntrospection(ctx,a.client,nodeID).ExtractErr()",
		"ValidateCapabilities(cfg,false,true,false)",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("missing %q in %s", expected, generated)
		}
	}
	if strings.Contains(generated, "upstream.StartIntrospection(") {
		t.Fatalf("native call retained: %s", generated)
	}
}

func TestAuditedStartCallRejectsNativeSignatureAndInputResultDrift(t *testing.T) {
	for name, mutate := range map[string]func(string) string{
		"missing function": func(s string) string { return strings.Replace(s, "func StartIntrospection(", "func RenamedStart(", 1) },
		"missing context": func(s string) string {
			return strings.Replace(s, "func StartIntrospection(ctx context.Context,", "func StartIntrospection(", 1)
		},
		"numeric node": func(s string) string { return strings.Replace(s, "nodeID string,opts", "nodeID int,opts", 1) },
		"new parameter": func(s string) string {
			return strings.Replace(s, "opts StartOptsBuilder)(r", "opts StartOptsBuilder,extra string)(r", 1)
		},
		"concrete input":    func(s string) string { return strings.Replace(s, "opts StartOptsBuilder)(r", "opts StartOpts)(r", 1) },
		"non optional boot": func(s string) string { return strings.Replace(s, "ManageBoot *bool", "ManageBoot bool", 1) },
		"boot query tag":    func(s string) string { return strings.Replace(s, "manage_boot", "boot", 1) },
		"new input field": func(s string) string {
			return strings.Replace(s, "type StartOpts struct{", "type StartOpts struct{Extra string;", 1)
		},
		"new builder trait": func(s string) string {
			return strings.Replace(s, "interface{To", "interface{ToStartHeaders()(map[string]string,error);To", 1) + "\nfunc(StartOpts)ToStartHeaders()(map[string]string,error){return nil,nil}\n"
		},
		"new result field": func(s string) string {
			return strings.Replace(s, "type StartResult struct{", "type StartResult struct{Extra string;", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			pkg, decls := introspectionStartFixture(t, mutate(introspectionStartFixtureSource), pinnedIntrospectionStartDeclaration)
			if err := validateAuditedRequestCalls(pkg, decls); err == nil || !strings.Contains(err.Error(), "audited request call") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestAuditedStartCallRejectsNativeBodyFixOrMissingDeclaration(t *testing.T) {
	for name, declaration := range map[string]string{
		"different success code": strings.Replace(pinnedIntrospectionStartDeclaration, "[]int{202}", "[]int{200}", 1),
		"native query fix":       strings.Replace(strings.Replace(pinnedIntrospectionStartDeclaration, "_, err := opts", "query, err := opts", 1), "introspectionURL(client, nodeID),", "introspectionURL(client, nodeID)+query,", 1),
	} {
		t.Run(name, func(t *testing.T) {
			pkg, decls := introspectionStartFixture(t, introspectionStartFixtureSource, declaration)
			if err := validateAuditedRequestCalls(pkg, decls); err == nil || !strings.Contains(err.Error(), "pinned native declaration changed") {
				t.Fatalf("err=%v", err)
			}
		})
	}
	pkg, decls := introspectionStartFixture(t, introspectionStartFixtureSource, pinnedIntrospectionStartDeclaration)
	delete(decls, "StartIntrospection")
	if err := validateAuditedRequestCalls(pkg, decls); err == nil || !strings.Contains(err.Error(), "native declaration is missing") {
		t.Fatalf("err=%v", err)
	}
}

func TestAuditedStartCallAllowsCommentAndFormattingChanges(t *testing.T) {
	declaration := "// changed documentation\n" + strings.ReplaceAll(pinnedIntrospectionStartDeclaration, "\t", "    ")
	pkg, decls := introspectionStartFixture(t, introspectionStartFixtureSource, declaration)
	if err := validateAuditedRequestCalls(pkg, decls); err != nil {
		t.Fatal(err)
	}
}
