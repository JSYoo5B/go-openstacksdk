// These tests use source fixtures adapted from Gophercloud v2.15.0.
// Copyright Gophercloud authors
// SPDX-License-Identifier: Apache-2.0
//
// Local modifications select and combine upstream declarations, add test stubs,
// and create altered source variants to check generator behavior.
// Upstream scope (github.com/gophercloud/gophercloud/v2):
//   openstack/messaging/v2/queues/requests.go
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

// This is the inspected v2.15.0 declaration: it hands the builder value itself
// to the request body instead of calling ToQueueUpdateMap.
const pinnedQueueUpdateDeclaration = `func Update(ctx context.Context, client *gophercloud.ServiceClient, queueName string, opts UpdateOptsBuilder) (r UpdateResult) {
	resp, err := client.Patch(ctx, updateURL(client, queueName), opts, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201, 204},
		MoreHeaders: map[string]string{
			"Content-Type": "application/openstack-messaging-v2.0-json-patch"},
	})
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`

const queueUpdateFixtureSource = `package queues
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
type UpdateOptsBuilder interface{ToQueueUpdateMap()([]map[string]any,error)}
type UpdateOp string
type UpdateOpts struct{
	Op UpdateOp ` + "`json:\"op\" required:\"true\"`" + `
	Path string ` + "`json:\"path\" required:\"true\"`" + `
	Value any ` + "`json:\"value\" required:\"true\"`" + `
}
type BatchUpdateOpts []UpdateOpts
func(BatchUpdateOpts)ToQueueUpdateMap()([]map[string]any,error){return nil,nil}
type QueueDetails struct{ID string}
type UpdateResult struct{gophercloud.Result}
func(UpdateResult)Extract()(QueueDetails,error){return QueueDetails{},nil}
func Update(ctx context.Context,client *gophercloud.ServiceClient,queueName string,opts UpdateOptsBuilder)(r UpdateResult){return}
func Get(ctx context.Context,client *gophercloud.ServiceClient,queueName string)(r UpdateResult){return}
`

func queueUpdateFixture(t *testing.T, source, declaration string) (*types.Package, map[string]*ast.FuncDecl) {
	t.Helper()
	cloud := types.NewPackage(upstreamModule, "gophercloud")
	client := types.NewNamed(types.NewTypeName(token.NoPos, cloud, "ServiceClient", nil), types.NewStruct(nil, nil), nil)
	cloud.Scope().Insert(client.Obj())
	cloud.Scope().Insert(types.NewNamed(types.NewTypeName(token.NoPos, cloud, "Result", nil), types.NewStruct(nil, nil), nil).Obj())
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
	pkg, err := config.Check(upstreamModule+"/openstack/messaging/v2/queues", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, node := range file.Decls {
		if decl, ok := node.(*ast.FuncDecl); ok && decl.Recv == nil {
			decls[decl.Name.Name] = decl
		}
	}
	native, err := parser.ParseFile(token.NewFileSet(), "native.go", "package queues\n"+declaration, 0)
	if err != nil {
		t.Fatal(err)
	}
	decls["Update"] = native.Decls[0].(*ast.FuncDecl)
	return pkg, decls
}

func TestAuditedQueueUpdateUsesBuilderBodyHelper(t *testing.T) {
	pkg, decls := queueUpdateFixture(t, queueUpdateFixtureSource, pinnedQueueUpdateDeclaration)
	if err := validateAuditedRequestCalls(pkg, decls); err != nil {
		t.Fatal(err)
	}
	override := requestCallOverride(pkg, "Update")
	if override == nil || override.helper != "updateQueue" || override.policy != "sdk_builder_body_update" || requestCallOverride(pkg, "Get") != nil {
		t.Fatalf("override=%+v", override)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}}
	fn := pkg.Scope().Lookup("Update").(*types.Func)
	if err := emitOperation(&e, fn, decls["Update"], nil); err != nil {
		t.Fatal(err)
	}
	generated := e.body.String()
	if !strings.Contains(generated, "updateQueue(ctx,a.client,queueName,_opts)") || strings.Contains(generated, "upstream.Update(") {
		t.Fatalf("generated=%s", generated)
	}
}

func TestAuditedQueueUpdateRejectsDrift(t *testing.T) {
	for name, mutate := range map[string]func(string) string{
		"extra option field": func(s string) string {
			return strings.Replace(s, "type UpdateOpts struct{", "type UpdateOpts struct{Extra string;", 1)
		},
		"json tag": func(s string) string { return strings.Replace(s, "json:\"path\"", "json:\"pointer\"", 1) },
		"concrete body": func(s string) string {
			return strings.Replace(s, "opts UpdateOptsBuilder)", "opts BatchUpdateOpts)", 1)
		},
		"extra builder method": func(s string) string {
			return strings.Replace(s, "interface{To", "interface{ToExtra()(string,error);To", 1) + "\nfunc(BatchUpdateOpts)ToExtra()(string,error){return \"\",nil}\n"
		},
	} {
		t.Run(name, func(t *testing.T) {
			pkg, decls := queueUpdateFixture(t, mutate(queueUpdateFixtureSource), pinnedQueueUpdateDeclaration)
			if err := validateAuditedRequestCalls(pkg, decls); err == nil || !strings.Contains(err.Error(), "audited request call") {
				t.Fatalf("err=%v", err)
			}
		})
	}
	for name, declaration := range map[string]string{
		"different success code": strings.Replace(pinnedQueueUpdateDeclaration, "[]int{200, 201, 204}", "[]int{200}", 1),
		"native map fix":         strings.Replace(pinnedQueueUpdateDeclaration, "updateURL(client, queueName), opts,", "updateURL(client, queueName), mustMap(opts),", 1),
	} {
		t.Run(name, func(t *testing.T) {
			pkg, decls := queueUpdateFixture(t, queueUpdateFixtureSource, declaration)
			if err := validateAuditedRequestCalls(pkg, decls); err == nil || !strings.Contains(err.Error(), "pinned native declaration changed") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
