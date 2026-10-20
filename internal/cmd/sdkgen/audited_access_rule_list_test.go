// These tests use source fixtures adapted from Gophercloud v2.15.0.
// Copyright Gophercloud authors
// SPDX-License-Identifier: Apache-2.0
//
// Local modifications select and combine upstream declarations, add test stubs,
// and create altered source variants to check generator behavior.
// Upstream scope (github.com/gophercloud/gophercloud/v2):
//   openstack/sharedfilesystems/v2/shareaccessrules/requests.go
// See the root THIRD_PARTY_NOTICES.md and licenses/gophercloud-v2.15.0-LICENSE.

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// This is the inspected v2.15.0 declaration: the URL builder it calls appends
// the raw share ID to the share_id query without escaping.
const pinnedAccessRuleListDeclaration = `func List(ctx context.Context, client *gophercloud.ServiceClient, shareID string) (r ListResult) {
	resp, err := client.Get(ctx, listURL(client, shareID), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`

const accessRuleListFixtureSource = `package shareaccessrules
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
type ShareAccess struct{ID string}
type GetResult struct{gophercloud.Result}
type ListResult struct{gophercloud.Result}
func(ListResult)Extract()([]ShareAccess,error){return nil,nil}
func(GetResult)Extract()(*ShareAccess,error){return nil,nil}
func Get(ctx context.Context,client *gophercloud.ServiceClient,accessID string)(r GetResult){return}
func List(ctx context.Context,client *gophercloud.ServiceClient,shareID string)(r ListResult){return}
`

func accessRuleListFixture(t *testing.T, source, declaration string) (*types.Package, map[string]*ast.FuncDecl) {
	t.Helper()
	cloud := types.NewPackage(upstreamModule, "gophercloud")
	client := types.NewNamed(types.NewTypeName(token.NoPos, cloud, "ServiceClient", nil), types.NewStruct(nil, nil), nil)
	cloud.Scope().Insert(client.Obj())
	for _, name := range []string{"Result", "ErrResult"} {
		cloud.Scope().Insert(types.NewNamed(types.NewTypeName(token.NoPos, cloud, name, nil), types.NewStruct(nil, nil), nil).Obj())
	}
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
	pkg, err := config.Check(upstreamModule+"/openstack/sharedfilesystems/v2/shareaccessrules", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, node := range file.Decls {
		if decl, ok := node.(*ast.FuncDecl); ok && decl.Recv == nil {
			decls[decl.Name.Name] = decl
		}
	}
	native, err := parser.ParseFile(token.NewFileSet(), "native.go", "package shareaccessrules\n"+declaration, 0)
	if err != nil {
		t.Fatal(err)
	}
	decls["List"] = native.Decls[0].(*ast.FuncDecl)
	return pkg, decls
}

func TestAuditedAccessRuleListUsesEscapingHelper(t *testing.T) {
	pkg, decls := accessRuleListFixture(t, accessRuleListFixtureSource, pinnedAccessRuleListDeclaration)
	if err := validateAuditedRequestCalls(pkg, decls); err != nil {
		t.Fatal(err)
	}
	override := requestCallOverride(pkg, "List")
	if override == nil || override.helper != "listAccessRules" || override.policy != "sdk_escaped_query_list" || requestCallOverride(pkg, "Get") != nil {
		t.Fatalf("override=%+v", override)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}}
	fn := pkg.Scope().Lookup("List").(*types.Func)
	if err := emitOperation(&e, fn, decls["List"], nil); err != nil {
		t.Fatal(err)
	}
	generated := e.body.String()
	if !strings.Contains(generated, "listAccessRules(ctx,a.client,shareID)") || strings.Contains(generated, "upstream.List(") {
		t.Fatalf("generated=%s", generated)
	}
}

func TestAuditedAccessRuleListRejectsDrift(t *testing.T) {
	for name, mutate := range map[string]func(string) string{
		"extra parameter": func(s string) string {
			return strings.Replace(s, "shareID string)(r ListResult)", "shareID, extra string)(r ListResult)", 1)
		},
		"foreign result": func(s string) string {
			return strings.Replace(s, "(r ListResult){return}", "(r gophercloud.Result){return}", 1)
		},
		"no extractor": func(s string) string {
			return strings.Replace(s, "func(ListResult)Extract()", "func(ListResult)Other()", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			pkg, decls := accessRuleListFixture(t, mutate(accessRuleListFixtureSource), pinnedAccessRuleListDeclaration)
			if err := validateAuditedRequestCalls(pkg, decls); err == nil || !strings.Contains(err.Error(), "audited request call") {
				t.Fatalf("err=%v", err)
			}
		})
	}
	for name, declaration := range map[string]string{
		"native escape fix": strings.Replace(pinnedAccessRuleListDeclaration, "listURL(client, shareID)", "listURL(client, url.QueryEscape(shareID))", 1),
		"different body":    strings.Replace(pinnedAccessRuleListDeclaration, "&r.Body, nil", "&r.Body, &gophercloud.RequestOpts{OkCodes: []int{200}}", 1),
	} {
		t.Run(name, func(t *testing.T) {
			pkg, decls := accessRuleListFixture(t, accessRuleListFixtureSource, declaration)
			if err := validateAuditedRequestCalls(pkg, decls); err == nil || !strings.Contains(err.Error(), "pinned native declaration changed") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
