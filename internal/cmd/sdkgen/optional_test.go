package main

import (
	"go/types"
	"testing"
)

func TestOptionalBuildersFollowNativeAssertionsAndConcreteInputs(t *testing.T) {
	pkg, decls := fixture(t, `package fixture
type Options struct{}
type OptionsBuilder interface { ToBody() (map[string]any,error) }
type HeadersBuilder interface { ToHeaders() (map[string]string,error) }
type QueryBuilder interface { ToQuery() (string,error) }
type UnknownBuilder interface { ToUnknown() error }
func(Options)ToBody()(map[string]any,error){return nil,nil}
func(Options)ToHeaders()(map[string]string,error){return nil,nil}
func(Options)ToQuery()(string,error){return "",nil}
func Request(opts OptionsBuilder, other OptionsBuilder){
 if h,ok:=opts.(HeadersBuilder);ok{h.ToHeaders()}
 if _,ok:=opts.(UnknownBuilder);ok{}
 if q,ok:=other.(QueryBuilder);ok{q.ToQuery()}
}
`)
	iface, _ := ifaceOf(pkg.Scope().Lookup("OptionsBuilder").Type())
	base := pkg.Scope().Lookup("Options").Type()
	b := withOptionalBuilders(pkg, builder{name: "opts", base: base, iface: iface}, decls["Request"], nil)
	caps := capabilities(pkg, b)
	if b.iface.NumMethods() != 2 || !caps.body || !caps.headers || caps.query {
		t.Fatalf("methods=%d caps=%+v", b.iface.NumMethods(), caps)
	}
	if !types.Implements(base, b.iface) {
		t.Fatal("generated contract exceeds its concrete input")
	}
}
