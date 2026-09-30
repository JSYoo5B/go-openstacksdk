package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func fixture(t *testing.T, source string) (*types.Package, map[string]*ast.FuncDecl) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := new(types.Config).Check("fixture", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			decls[fn.Name.Name] = fn
		}
	}
	return pkg, decls
}

func TestExtractorFollowsIntoHelpersAndKeepsTypedPages(t *testing.T) {
	pkg, decls := fixture(t, `package fixture
type Page interface{}
type Server struct{ ID string }
type ServerPage struct{}
type AddressPage struct{}
type Address struct{ Addr string }
func ExtractServersInto(p Page, v any) error { _ = p.(ServerPage); return nil }
func ExtractServers(p Page) ([]Server,error) { var values []Server; return values, ExtractServersInto(p,&values) }
func ExtractAddresses(p Page) (map[string][]Address,error) { _ = p.(AddressPage); return nil,nil }
func List() Page { return ServerPage{} }
func ListAddresses() Page { return AddressPage{} }
`)
	byPage := extractorsByPage(pkg, decls)
	name, typ := findExtractor(pkg, "List", decls["List"], byPage)
	if name != "ExtractServers" {
		t.Fatalf("extractor=%s", name)
	}
	if _, ok := typ.Underlying().(*types.Slice); !ok {
		t.Fatalf("type=%v", typ)
	}
	name, typ = findExtractor(pkg, "ListAddresses", decls["ListAddresses"], byPage)
	if name != "ExtractAddresses" {
		t.Fatalf("extractor=%s", name)
	}
	if _, ok := typ.Underlying().(*types.Map); !ok {
		t.Fatalf("type=%v", typ)
	}
}

func TestReturnPoliciesKeepErrorsAndPrimitiveResults(t *testing.T) {
	pkg, _ := fixture(t, `package fixture
type Empty struct{}
func (Empty) ExtractErr() error { return nil }
type Value struct{}
func (*Value) Extract() (string,error) { return "",nil }
type Unparsed struct{ Err error }
func Delete() Empty { return Empty{} }
func Get() Value { return Value{} }
func Upload() Unparsed { return Unparsed{} }
func Wait() error { return nil }
func URL() string { return "" }
`)
	for name, want := range map[string]string{"Delete": "error", "Get": "extract", "Upload": "result", "Wait": "direct-error", "URL": "direct"} {
		sig := pkg.Scope().Lookup(name).Type().(*types.Signature)
		if got := returnPolicy(sig); got != want {
			t.Errorf("%s: %s, want %s", name, got, want)
		}
	}
}

func TestGenericBatchUsesConcreteInputsAndNamedSliceRemainsTyped(t *testing.T) {
	pkg, _ := fixture(t, `package fixture
type UpdateOptsBuilder interface{ ToUpdateMap() (map[string]any,error) }
type UpdateOpts struct{ Name string }
func (UpdateOpts) ToUpdateMap() (map[string]any,error) { return nil,nil }
func Update[T UpdateOptsBuilder](options []T) error { return nil }
type Record struct{ Name string }
type Records []Record
`)
	e := emitter{pkg: pkg, imports: map[string]string{}}
	sig := pkg.Scope().Lookup("Update").Type().(*types.Signature)
	if got := e.typ(sig.Params().At(0).Type()); got != "[]UpdateOpts" {
		t.Fatalf("type=%s", got)
	}
	if got := e.typ(pkg.Scope().Lookup("Records").Type()); got != "Records" {
		t.Fatalf("type=%s", got)
	}
	if err := emitOperation(&e, pkg.Scope().Lookup("Update").(*types.Func), nil, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.body.String(), "options []UpdateOpts") {
		t.Fatalf("missing concrete batch input: %s", e.body.String())
	}
}
