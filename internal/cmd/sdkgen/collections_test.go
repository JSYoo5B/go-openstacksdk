package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

type packageImports map[string]*types.Package

func (i packageImports) Import(path string) (*types.Package, error) { return i[path], nil }

func collectionFixture(t *testing.T, scoped bool) *collectionPlan {
	t.Helper()
	cloud := types.NewPackage(upstreamModule, "gophercloud")
	cloud.Scope().Insert(types.NewTypeName(token.NoPos, cloud, "ServiceClient", types.NewNamed(types.NewTypeName(token.NoPos, cloud, "ServiceClient", nil), types.NewStruct(nil, nil), nil)))
	page := types.NewPackage(upstreamModule+"/pagination", "pagination")
	page.Scope().Insert(types.NewTypeName(token.NoPos, page, "Page", types.NewInterfaceType(nil, nil).Complete()))
	page.Scope().Insert(types.NewTypeName(token.NoPos, page, "Pager", types.NewNamed(types.NewTypeName(token.NoPos, page, "Pager", nil), types.NewStruct(nil, nil), nil)))
	cloud.MarkComplete()
	page.MarkComplete()
	getArgs := "id string"
	if scoped {
		getArgs = "parentID,id string"
	}
	source := `package fixture
import gophercloud "github.com/gophercloud/gophercloud/v2"
import "github.com/gophercloud/gophercloud/v2/pagination"
type Thing struct{ID int;Name string;Status string}
type GetResult struct{}
func(GetResult)Extract()(*Thing,error){return nil,nil}
type ListOpts struct{Name string ` + "`q:\"name\"`" + `;Status string ` + "`q:\"status\"`" + `}
type ListOptsBuilder interface{ToListQuery()(string,error)}
func(ListOpts)ToListQuery()(string,error){return "",nil}
type ThingPage struct{}
func Get(client *gophercloud.ServiceClient,` + getArgs + `)GetResult{return GetResult{}}
func List(client *gophercloud.ServiceClient,opts ListOptsBuilder)pagination.Pager{_ = ThingPage{};return pagination.Pager{}}
func ExtractThings(p pagination.Page)([]Thing,error){_ = p.(ThingPage);return nil,nil}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{Importer: packageImports{upstreamModule: cloud, upstreamModule + "/pagination": page}}
	pkg, err := config.Check("fixture", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			decls[fn.Name.Name] = fn
		}
	}
	return identifyCollection(pkg, decls, extractorsByPage(pkg, decls))
}

func TestCollectionBindingRequiresMatchingTypedScope(t *testing.T) {
	plan := collectionFixture(t, false)
	if plan == nil || plan.modelName != "Thing" || plan.id != "ID" || plan.name != "Name" || plan.status != "Status" || !plan.listQueryBuilder || plan.nameQuery != "name" {
		t.Fatalf("plan=%+v", plan)
	}
	if plan := collectionFixture(t, true); plan != nil {
		t.Fatalf("unscoped binding created for scoped Get: %+v", plan)
	}
}
