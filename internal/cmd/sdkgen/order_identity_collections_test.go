package main

import (
	"encoding/json"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const orderIdentityFixtureSource = `package orders
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
import "github.com/gophercloud/gophercloud/v2/pagination"
var _ context.Context
type Meta struct{Name string}
type Order struct{OrderRef string ` + "`json:\"order_ref\"`" + `;SecretRef string ` + "`json:\"secret_ref\"`" + `;ContainerRef string ` + "`json:\"container_ref\"`" + `;Meta Meta;Status string ` + "`json:\"status\"`" + `}
type Other struct{OrderRef string}
type GetResult struct{}
func(GetResult)Extract()(*Order,error){return nil,nil}
type DeleteResult struct{}
func(DeleteResult)ExtractErr()error{return nil}
type ListOpts struct{Limit int ` + "`q:\"limit\"`" + `;Offset int ` + "`q:\"offset\"`" + `}
type ListOptsBuilder interface{ToOrderListQuery()(string,error)}
func(ListOpts)ToOrderListQuery()(string,error){return "",nil}
type OrderPage struct{}
func Get(ctx context.Context,client *gophercloud.ServiceClient,id string)GetResult{return GetResult{}}
func List(client *gophercloud.ServiceClient,opts ListOptsBuilder)pagination.Pager{_ = OrderPage{};return pagination.Pager{}}
func ExtractOrders(p pagination.Page)([]Order,error){_ = p.(OrderPage);return nil,nil}
func Delete(ctx context.Context,client *gophercloud.ServiceClient,id string)DeleteResult{return DeleteResult{}}
`

func TestOrderCollectionIdentityUsesOrderRefWithoutChangingGenericPriority(t *testing.T) {
	path := upstreamModule + "/openstack/keymanager/v1/orders"
	pkg, decls := typedCollectionFixture(t, path, orderIdentityFixtureSource)
	plan, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls))
	if err != nil || plan == nil || plan.id != "OrderRef" || !plan.idIsURL || plan.name != "" || plan.status != "Status" || plan.deleter == nil {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}}
	emitCollectionAdapter(&e, plan, "a", nil)
	source := e.body.String()
	if !strings.Contains(source, "url.Parse(v.OrderRef)") || strings.Contains(source, "v.SecretRef") || strings.Contains(source, "v.Meta") || strings.Contains(source, "Name:") || strings.Contains(source, "IdentityFind:") {
		t.Fatalf("order identity or capabilities changed: %s", source)
	}
	for _, unchanged := range []string{"a.Get(ctx,string(id))", "a.deleteOwned(ctx,id)", `q.Del("status")`, "a.listWithControl(ctx,control,options...)"} {
		if !strings.Contains(source, unchanged) {
			t.Errorf("missing existing native behavior %q in %s", unchanged, source)
		}
	}
	// The collection exception must not globally reorder SecretRef and OrderRef.
	for _, otherPath := range []string{upstreamModule + "/openstack/keymanager/v1/secrets", "example/other/orders"} {
		pkg, decls := typedCollectionFixture(t, otherPath, orderIdentityFixtureSource)
		generic, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls))
		if err != nil || generic == nil || generic.id != "SecretRef" {
			t.Fatalf("generic path=%s plan=%+v err=%v", otherPath, generic, err)
		}
	}
}

func TestOrderCollectionIdentityRejectsAmbiguousNativeShapeDrift(t *testing.T) {
	for name, change := range map[string][2]string{
		"missing order ref":        {"OrderRef string `json:\"order_ref\"`;", ""},
		"numeric order ref":        {"OrderRef string `json:\"order_ref\"`", "OrderRef int `json:\"order_ref\"`"},
		"named string order ref":   {"OrderRef string `json:\"order_ref\"`", "OrderRef NamedString `json:\"order_ref\"`"},
		"wrong order ref tag":      {"OrderRef string `json:\"order_ref\"`", "OrderRef string `json:\"secret_ref\"`"},
		"promoted order ref":       {"OrderRef string `json:\"order_ref\"`;", "Embedded;"},
		"different model":          {"type Order struct", "type Renamed struct"},
		"new top-level name":       {"type Order struct{", "type Order struct{Name string;"},
		"new nonstring name":       {"type Order struct{", "type Order struct{Name bool;"},
		"missing status":           {";Status string `json:\"status\"`", ""},
		"wrong status tag":         {"Status string `json:\"status\"`", "Status string `json:\"state\"`"},
		"missing get":              {"func Get(", "func Fetch("},
		"get without context":      {"func Get(ctx context.Context,", "func Get("},
		"get numeric id":           {"func Get(ctx context.Context,client *gophercloud.ServiceClient,id string)", "func Get(ctx context.Context,client *gophercloud.ServiceClient,id int)"},
		"get named string id":      {"func Get(ctx context.Context,client *gophercloud.ServiceClient,id string)", "func Get(ctx context.Context,client *gophercloud.ServiceClient,id NamedString)"},
		"get with extra options":   {"func Get(ctx context.Context,client *gophercloud.ServiceClient,id string)", "func Get(ctx context.Context,client *gophercloud.ServiceClient,id string,opts ListOptsBuilder)"},
		"get different model":      {"func(GetResult)Extract()(*Order,error)", "func(GetResult)Extract()(*Other,error)"},
		"get wrong result name":    {"GetResult", "FetchResult"},
		"missing list":             {"func List(", "func ListOrders("},
		"list with parent":         {"func List(client *gophercloud.ServiceClient,opts", "func List(client *gophercloud.ServiceClient,parentID string,opts"},
		"list with context":        {"func List(client *gophercloud.ServiceClient,opts", "func List(ctx context.Context,client *gophercloud.ServiceClient,opts"},
		"list without builder":     {"func List(client *gophercloud.ServiceClient,opts ListOptsBuilder)", "func List(client *gophercloud.ServiceClient,opts ListOpts)"},
		"renamed list builder":     {"ListOptsBuilder", "OtherListOptsBuilder"},
		"renamed query method":     {"ToOrderListQuery", "ToOtherListQuery"},
		"list different model":     {"func ExtractOrders(p pagination.Page)([]Order,error)", "func ExtractOrders(p pagination.Page)([]Other,error)"},
		"missing delete":           {"func Delete(", "func DeleteOrder("},
		"delete without context":   {"func Delete(ctx context.Context,", "func Delete("},
		"delete numeric id":        {"func Delete(ctx context.Context,client *gophercloud.ServiceClient,id string)", "func Delete(ctx context.Context,client *gophercloud.ServiceClient,id int)"},
		"delete wrong result name": {"DeleteResult", "OtherDeleteResult"},
	} {
		t.Run(name, func(t *testing.T) {
			source := orderIdentityFixtureSource
			if name == "different model" {
				source = strings.ReplaceAll(source, "Order", "Renamed")
			} else {
				source = strings.ReplaceAll(source, change[0], change[1])
			}
			source += "\ntype NamedString string\ntype Embedded struct{OrderRef string `json:\"order_ref\"`}\n"
			pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/keymanager/v1/orders", source)
			plan, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls))
			if err == nil || plan != nil || !strings.Contains(err.Error(), "audited order collection") {
				t.Fatalf("drift %s plan=%+v err=%v", name, plan, err)
			}
		})
	}
}

func TestOrderCollectionIdentityAcceptsActualPinnedCompiledTypeGraph(t *testing.T) {
	path := os.Getenv("GOPHERCLOUD_METADATA")
	if path == "" {
		path = "/private/tmp/gophercloudsdk-upstream-packages.json"
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		t.Skip("compiled pinned native metadata not present; pass GOPHERCLOUD_METADATA")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	meta := map[string]metadata{}
	decoder := json.NewDecoder(file)
	for {
		var entry metadata
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		meta[entry.ImportPath] = entry
	}
	path = upstreamModule + "/openstack/keymanager/v1/orders"
	native, ok := meta[path]
	if !ok || native.Dir == "" || native.Export == "" {
		t.Fatal("native Order source/export metadata missing")
	}
	compiled := importer.ForCompiler(token.NewFileSet(), "gc", func(path string) (io.ReadCloser, error) {
		entry, ok := meta[path]
		if !ok || entry.Export == "" {
			return nil, os.ErrNotExist
		}
		return os.Open(entry.Export)
	})
	pkg, err := compiled.Import(path)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, name := range native.GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(native.Dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Recv == nil && ast.IsExported(fn.Name.Name) {
				decls[fn.Name.Name] = fn
			}
		}
	}
	plan, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls))
	if err != nil || plan == nil || plan.id != "OrderRef" || plan.name != "" || plan.status != "Status" || !plan.idIsURL || plan.deleter == nil {
		t.Fatalf("actual plan=%+v err=%v", plan, err)
	}
	for name, count := range map[string]int{"Order": 1, "Meta": 1, "ListOpts": 1, "OrderPage": 2} {
		value, ok := pkg.Scope().Lookup(name).Type().(*types.Named)
		if !ok || value.NumMethods() != count {
			t.Fatalf("actual %s own methods=%v, want %d", name, value, count)
		}
	}
	metaType := pkg.Scope().Lookup("Meta").Type()
	if field, _, _ := types.LookupFieldOrMethod(metaType, true, nil, "Name"); field == nil {
		t.Fatal("actual nested Meta.Name was omitted")
	}
	e := emitter{pkg: pkg, imports: map[string]string{}}
	emitCollectionAdapter(&e, plan, "a", nil)
	if source := e.body.String(); !strings.Contains(source, "url.Parse(v.OrderRef)") || strings.Contains(source, "v.SecretRef") || strings.Contains(source, "Name:") {
		t.Fatalf("actual native emission=%s", source)
	}
}
