package main

import (
	"go/types"
	"testing"
)

func TestNamedResultExtractionCannotChooseAnotherResource(t *testing.T) {
	pkg, _ := fixture(t, `package fixture
type Shared struct{Err error}
type A struct{ID string}
type B struct{ID string}
func(Shared)ExtractA()(*A,error){return nil,nil}
func(Shared)ExtractB()(*B,error){return nil,nil}
func GetA()Shared{return Shared{}}
func UpdateB()Shared{return Shared{}}
func Get()Shared{return Shared{}}
type Single struct{Err error}
func(Single)ExtractImageID()(string,error){return "",nil}
func CreateImage()Single{return Single{}}
`)
	for name, want := range map[string]string{"GetA": "ExtractA", "UpdateB": "ExtractB", "Get": "", "CreateImage": "ExtractImageID"} {
		fn := pkg.Scope().Lookup(name).(*types.Func)
		got, ex := operationExtractor(fn)
		if got != want || (ex == nil) != (want == "") {
			t.Errorf("%s: got %q extractor=%v want %q", name, got, ex, want)
		}
		wantPolicy := "extract"
		if want == "" {
			wantPolicy = "result"
		}
		if got := operationReturnPolicy(fn); got != wantPolicy {
			t.Errorf("%s policy=%s want %s", name, got, wantPolicy)
		}
	}
}
