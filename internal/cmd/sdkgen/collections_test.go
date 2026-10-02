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
	pkg, decls := collectionFixturePackage(t, scoped)
	return identifyCollection(pkg, decls, extractorsByPage(pkg, decls))
}

func collectionFixturePackage(t *testing.T, scoped bool) (*types.Package, map[string]*ast.FuncDecl) {
	t.Helper()
	getArgs := "id string"
	listArgs := "opts ListOptsBuilder"
	if scoped {
		getArgs = "parentID,id string"
		listArgs = "parentID string,opts ListOptsBuilder"
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
func List(client *gophercloud.ServiceClient,` + listArgs + `)pagination.Pager{_ = ThingPage{};return pagination.Pager{}}
func ExtractThings(p pagination.Page)([]Thing,error){_ = p.(ThingPage);return nil,nil}
`
	return typedCollectionFixture(t, "fixture", source)
}

func typedCollectionFixture(t *testing.T, packagePath, source string) (*types.Package, map[string]*ast.FuncDecl) {
	t.Helper()
	return typedCollectionFixtureWithRuleSource(t, packagePath, source, securityGroupRuleFixtureSource)
}
func typedCollectionFixtureWithRuleSource(t *testing.T, packagePath, source, ruleSource string) (*types.Package, map[string]*ast.FuncDecl) {
	t.Helper()
	cloud := types.NewPackage(upstreamModule, "gophercloud")
	cloud.Scope().Insert(types.NewTypeName(token.NoPos, cloud, "ServiceClient", types.NewNamed(types.NewTypeName(token.NoPos, cloud, "ServiceClient", nil), types.NewStruct(nil, nil), nil)))
	link := types.NewNamed(types.NewTypeName(token.NoPos, cloud, "Link", nil), types.NewStruct([]*types.Var{types.NewVar(token.NoPos, cloud, "Href", types.Typ[types.String]), types.NewVar(token.NoPos, cloud, "Rel", types.Typ[types.String])}, []string{`json:"href"`, `json:"rel"`}), nil)
	cloud.Scope().Insert(types.NewTypeName(token.NoPos, cloud, "Link", link))
	page := types.NewPackage(upstreamModule+"/pagination", "pagination")
	page.Scope().Insert(types.NewTypeName(token.NoPos, page, "Page", types.NewNamed(types.NewTypeName(token.NoPos, page, "Page", nil), types.NewInterfaceType(nil, nil).Complete(), nil)))
	page.Scope().Insert(types.NewTypeName(token.NoPos, page, "Pager", types.NewNamed(types.NewTypeName(token.NoPos, page, "Pager", nil), types.NewStruct(nil, nil), nil)))
	linked := types.NewNamed(types.NewTypeName(token.NoPos, page, "LinkedPageBase", nil), types.NewStruct(nil, nil), nil)
	nextResult := types.NewTuple(types.NewVar(token.NoPos, page, "", types.Typ[types.String]), types.NewVar(token.NoPos, page, "", types.Universe.Lookup("error").Type()))
	linked.AddMethod(types.NewFunc(token.NoPos, page, "NextPageURL", types.NewSignatureType(types.NewVar(token.NoPos, page, "current", linked), nil, nil, types.NewTuple(), nextResult, false)))
	page.Scope().Insert(types.NewTypeName(token.NoPos, page, "LinkedPageBase", linked))
	cloud.MarkComplete()
	page.MarkComplete()
	contexts := types.NewPackage("context", "context")
	contexts.Scope().Insert(types.NewTypeName(token.NoPos, contexts, "Context", types.NewNamed(types.NewTypeName(token.NoPos, contexts, "Context", nil), types.NewInterfaceType(nil, nil).Complete(), nil)))
	contexts.MarkComplete()
	http := types.NewPackage("net/http", "http")
	http.Scope().Insert(types.NewTypeName(token.NoPos, http, "Header", types.NewNamed(types.NewTypeName(token.NoPos, http, "Header", nil), types.NewMap(types.Typ[types.String], types.NewSlice(types.Typ[types.String])), nil)))
	http.MarkComplete()
	times := types.NewPackage("time", "time")
	times.Scope().Insert(types.NewTypeName(token.NoPos, times, "Time", types.NewNamed(types.NewTypeName(token.NoPos, times, "Time", nil), types.NewStruct(nil, nil), nil)))
	times.MarkComplete()
	// Audited SubnetPool decoder dependency: a named time.Time wrapper with
	// its sole pointer JSON decoder, without tying fixtures to private Time fields.
	noZ := types.NewNamed(types.NewTypeName(token.NoPos, cloud, "JSONRFC3339NoZ", nil), times.Scope().Lookup("Time").Type().Underlying(), nil)
	noZ.AddMethod(types.NewFunc(token.NoPos, cloud, "UnmarshalJSON", types.NewSignatureType(types.NewVar(token.NoPos, cloud, "jt", types.NewPointer(noZ)), nil, nil, types.NewTuple(types.NewVar(token.NoPos, cloud, "data", types.NewSlice(types.Typ[types.Uint8]))), types.NewTuple(types.NewVar(token.NoPos, cloud, "", types.Universe.Lookup("error").Type())), false)))
	cloud.Scope().Insert(types.NewTypeName(token.NoPos, cloud, "JSONRFC3339NoZ", noZ))
	result := types.NewNamed(types.NewTypeName(token.NoPos, cloud, "Result", nil), types.NewStruct([]*types.Var{types.NewField(token.NoPos, cloud, "Body", types.NewInterfaceType(nil, nil).Complete(), false), types.NewField(token.NoPos, cloud, "StatusCode", types.Typ[types.Int], false), types.NewField(token.NoPos, cloud, "Header", http.Scope().Lookup("Header").Type(), false), types.NewField(token.NoPos, cloud, "Err", types.Universe.Lookup("error").Type(), false)}, nil), nil)
	cloud.Scope().Insert(types.NewTypeName(token.NoPos, cloud, "Result", result))
	cloud.SetImports([]*types.Package{times})
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	ruleSet := token.NewFileSet()
	ruleFile, err := parser.ParseFile(ruleSet, "rule.go", ruleSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	ruleConfig := types.Config{Importer: packageImports{"time": times}}
	rules, err := ruleConfig.Check(securityGroupRulesNativePath, ruleSet, []*ast.File{ruleFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{Importer: packageImports{upstreamModule: cloud, upstreamModule + "/pagination": page, securityGroupRulesNativePath: rules, "context": contexts, "net/http": http, "time": times}}
	pkg, err := config.Check(packagePath, fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			decls[fn.Name.Name] = fn
		}
	}
	return pkg, decls
}

func TestScopedCollectionRequiresMatchingParentArity(t *testing.T) {
	pkg, decls := collectionFixturePackage(t, true)
	plan := identifyNamedCollection(pkg, decls, extractorsByPage(pkg, decls), "Get", []string{"List"}, "Delete", 1)
	if plan == nil || plan.modelName != "Thing" || !plan.listQueryBuilder {
		t.Fatalf("scoped plan=%+v", plan)
	}
	if _, ok := simpleInput(plan.getter, 2); !ok {
		t.Fatal("parent and target identifiers were lost")
	}
	if wrong := identifyNamedCollection(pkg, decls, extractorsByPage(pkg, decls), "Get", []string{"List"}, "Delete", 2); wrong != nil {
		t.Fatalf("binding with the wrong parent arity: %+v", wrong)
	}
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
