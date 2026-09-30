package main

import (
	"go/types"
	"testing"
)

func TestHeaderAndQueryDetectionKeepsExtraSpecKeysOutOfQueries(t *testing.T) {
	pkg, _ := fixture(t, `package fixture
type Options struct{}
func(Options)ToObjectDownloadParams()(map[string]string,string,error){return nil,"",nil}
func(Options)ToFlavorExtraSpecUpdateMap()(map[string]string,string,error){return nil,"",nil}
func(Options)ToTokenV3HeadersMap()(map[string]string,error){return nil,nil}
func(Options)ToListQuery()(string,error){return "",nil}
func(Options)ToServiceListMap()(string,error){return "",nil}
func(Options)ToClaimCreateRequest()(map[string]any,string,error){return nil,"",nil}
`)
	typ := pkg.Scope().Lookup("Options").Type().(*types.Named)
	for i := 0; i < typ.NumMethods(); i++ {
		method := typ.Method(i)
		caps := methodCapabilities(pkg, method)
		switch method.Name() {
		case "ToObjectDownloadParams":
			if !caps.query {
				t.Fatal(caps)
			}
		case "ToFlavorExtraSpecUpdateMap":
			if caps.query || caps.headers || caps.body {
				t.Fatal(caps)
			}
		case "ToTokenV3HeadersMap":
			if !caps.headers {
				t.Fatal(caps)
			}
		case "ToListQuery", "ToServiceListMap":
			if !caps.query {
				t.Fatal(caps)
			}
		case "ToClaimCreateRequest":
			if !caps.query || !caps.body {
				t.Fatal(caps)
			}
		}
	}
}
