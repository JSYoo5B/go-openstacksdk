// These tests use source fixtures adapted from Gophercloud v2.15.0.
// Copyright Gophercloud authors
// SPDX-License-Identifier: Apache-2.0
//
// Local modifications select and combine upstream declarations, add test stubs,
// and create altered source variants to check generator behavior.
// Upstream scope (github.com/gophercloud/gophercloud/v2):
//   openstack/dns/v2/zones/requests.go
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

// This is the inspected v2.15.0 declaration: it reflects over the builder
// value rather than calling its ToShareMap method.
const pinnedZoneShareDeclaration = `func Share(ctx context.Context, client *gophercloud.ServiceClient, zoneID string, opts ShareOptsBuilder) (r ZoneShareResult) {
	body, err := gophercloud.BuildRequestBody(opts, "")
	if err != nil {
		r.Err = err
		return
	}

	resp, err := client.Post(ctx, sharesBaseURL(client, zoneID), body, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{201},
	})
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`

const zoneShareFixtureSource = `package zones
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
type ShareOptsBuilder interface{ToShareMap()(map[string]interface{},error)}
type ShareZoneOpts struct{TargetProjectID string ` + "`json:\"target_project_id\" required:\"true\"`" + `}
func(ShareZoneOpts)ToShareMap()(map[string]interface{},error){return nil,nil}
type ZoneShare struct{ID string}
type ZoneShareResult struct{gophercloud.Result}
func(ZoneShareResult)Extract()(*ZoneShare,error){return nil,nil}
func Share(ctx context.Context,client *gophercloud.ServiceClient,zoneID string,opts ShareOptsBuilder)(r ZoneShareResult){return}
func Unshare(ctx context.Context,client *gophercloud.ServiceClient,zoneID,shareID string)(r gophercloud.ErrResult){return}
`

func zoneShareFixture(t *testing.T, source, declaration string) (*types.Package, map[string]*ast.FuncDecl) {
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
	pkg, err := config.Check(upstreamModule+"/openstack/dns/v2/zones", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, node := range file.Decls {
		if decl, ok := node.(*ast.FuncDecl); ok && decl.Recv == nil {
			decls[decl.Name.Name] = decl
		}
	}
	if decls["Share"] != nil {
		native, err := parser.ParseFile(token.NewFileSet(), "native.go", "package zones\n"+declaration, 0)
		if err != nil {
			t.Fatal(err)
		}
		decls["Share"] = native.Decls[0].(*ast.FuncDecl)
	}
	return pkg, decls
}

func TestAuditedZoneShareUsesBuilderBodyHelper(t *testing.T) {
	pkg, decls := zoneShareFixture(t, zoneShareFixtureSource, pinnedZoneShareDeclaration)
	if err := validateAuditedRequestCalls(pkg, decls); err != nil {
		t.Fatal(err)
	}
	override := requestCallOverride(pkg, "Share")
	if override == nil || override.helper != "shareZone" || override.policy != "sdk_builder_body_share" || requestCallOverride(pkg, "Unshare") != nil {
		t.Fatalf("override=%+v", override)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}}
	fn := pkg.Scope().Lookup("Share").(*types.Func)
	if err := emitOperation(&e, fn, decls["Share"], nil); err != nil {
		t.Fatal(err)
	}
	generated := e.body.String()
	if !strings.Contains(generated, "shareZone(ctx,a.client,zoneID,_opts)") || strings.Contains(generated, "upstream.Share(") {
		t.Fatalf("generated=%s", generated)
	}
}

func TestAuditedZoneShareRejectsDrift(t *testing.T) {
	for name, mutate := range map[string]func(string) string{
		"extra option field": func(s string) string {
			return strings.Replace(s, "type ShareZoneOpts struct{", "type ShareZoneOpts struct{Extra string;", 1)
		},
		"json tag":      func(s string) string { return strings.Replace(s, "target_project_id", "project", 1) },
		"concrete body": func(s string) string { return strings.Replace(s, "opts ShareOptsBuilder)", "opts ShareZoneOpts)", 1) },
		"extra builder method": func(s string) string {
			return strings.Replace(s, "interface{To", "interface{ToExtra()(string,error);To", 1) + "\nfunc(ShareZoneOpts)ToExtra()(string,error){return \"\",nil}\n"
		},
	} {
		t.Run(name, func(t *testing.T) {
			pkg, decls := zoneShareFixture(t, mutate(zoneShareFixtureSource), pinnedZoneShareDeclaration)
			if err := validateAuditedRequestCalls(pkg, decls); err == nil || !strings.Contains(err.Error(), "audited request call") {
				t.Fatalf("err=%v", err)
			}
		})
	}
	for name, declaration := range map[string]string{
		"different success code": strings.Replace(pinnedZoneShareDeclaration, "[]int{201}", "[]int{200}", 1),
		"native map fix":         strings.Replace(pinnedZoneShareDeclaration, "BuildRequestBody(opts, \"\")", "opts.ToShareMap()", 1),
	} {
		t.Run(name, func(t *testing.T) {
			pkg, decls := zoneShareFixture(t, zoneShareFixtureSource, declaration)
			if err := validateAuditedRequestCalls(pkg, decls); err == nil || !strings.Contains(err.Error(), "pinned native declaration changed") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
