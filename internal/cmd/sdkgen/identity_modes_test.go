package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func identityModeFixtureSource(spec identityListModeSpec) string {
	opts := "type ListOpts struct{Name string `q:\"name\"`}\ntype ListOptsBuilder interface{ToListQuery()(string,error)}\nfunc(ListOpts)ToListQuery()(string,error){return \"\",nil}"
	source := controlledFixtureSource(spec.model, "ID string;Name string", "Get", "id string", "List", ",opts ListOptsBuilder", opts)
	source = strings.ReplaceAll(source, "ModelPage", spec.page)
	source = strings.ReplaceAll(source, "ExtractModels", spec.extractor)
	source = strings.Replace(source, "type "+spec.page+" struct{}", "type "+spec.page+" struct{pagination.LinkedPageBase}", 1)
	source += fmt.Sprintf("\nfunc(%s)IsEmpty()(bool,error){return false,nil}\nfunc(%s)NextPageURL()(string,error){return \"\",nil}\n", spec.page, spec.page)
	if spec.model == "Server" {
		source += "\nfunc ListSimple(client *gophercloud.ServiceClient,opts ListOptsBuilder)pagination.Pager{return pagination.Pager{}}\n"
	}
	return source
}

func identityModeFixture(t *testing.T, spec identityListModeSpec, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/"+spec.path, source)
	return pkg, identifyCollection(pkg, decls, extractorsByPage(pkg, decls))
}

func TestIdentityListModesOnlyEnableAuditedConcretePagerContracts(t *testing.T) {
	if len(identityListModeSpecs) != 2 {
		t.Fatal("unexpected fallback modes", len(identityListModeSpecs))
	}
	for _, spec := range identityListModeSpecs {
		t.Run(spec.path, func(t *testing.T) {
			base := identityModeFixtureSource(spec)
			pkg, plan := identityModeFixture(t, spec, base)
			if !identityListModeEnabled(pkg, plan) {
				t.Fatal("audited native page/extractor shape disabled", spec)
			}
			mutations := map[string]func(string) string{
				"missing page": func(s string) string { return strings.ReplaceAll(s, spec.page, "OtherPage") },
				"new page field": func(s string) string {
					return strings.Replace(s, "struct{pagination.LinkedPageBase}", "struct{pagination.LinkedPageBase;Extra string}", 1)
				},
				"wrong page base": func(s string) string {
					return strings.Replace(s, "struct{pagination.LinkedPageBase}", "struct{pagination.Pager}", 1)
				},
				"missing extractor": func(s string) string { return strings.Replace(s, "func "+spec.extractor+"(", "func ExtractOther(", 1) },
				"pointer element extraction": func(s string) string {
					return strings.Replace(s, "([]"+spec.model+",error)", "([]*"+spec.model+",error)", 1)
				},
				"extractor parameter": func(s string) string {
					return strings.Replace(s, spec.extractor+"(p pagination.Page)", spec.extractor+"(p pagination.Page,extra string)", 1)
				},
				"next page result": func(s string) string {
					return strings.Replace(s, "NextPageURL()(string,error){return \"\",nil}", "NextPageURL()(int,error){return 0,nil}", 1)
				},
				"empty result": func(s string) string {
					return strings.Replace(s, "IsEmpty()(bool,error){return false,nil}", "IsEmpty()(string,error){return \"\",nil}", 1)
				},
				"new list parameter": func(s string) string {
					return strings.Replace(s, "func List(client *gophercloud.ServiceClient,opts ListOptsBuilder)", "func List(client *gophercloud.ServiceClient,opts ListOptsBuilder,extra string)", 1)
				},
			}
			if spec.model == "Server" {
				mutations["missing summary list"] = func(s string) string { return strings.Replace(s, "func ListSimple(", "func OtherList(", 1) }
				mutations["different summary builder"] = func(s string) string {
					return strings.Replace(s, "func ListSimple(client *gophercloud.ServiceClient,opts ListOptsBuilder)", "func ListSimple(client *gophercloud.ServiceClient,opts ListOpts)", 1)
				}
			}
			for name, mutate := range mutations {
				t.Run(name, func(t *testing.T) {
					badPkg, badPlan := identityModeFixture(t, spec, mutate(base))
					if identityListModeEnabled(badPkg, badPlan) {
						t.Fatal("native shape drift survived", name)
					}
					if err := validateIdentityListModeContracts(badPkg, nil, badPlan); err == nil || !strings.Contains(err.Error(), "signature changed") {
						t.Fatal("shape drift silently disabled modes", err)
					}
				})
			}
		})
	}
	for _, spec := range identityCollectionSpecs {
		if spec.model == "Server" || spec.model == "Volume" {
			continue
		}
		t.Run("not-mode/"+spec.path, func(t *testing.T) {
			pkg, plan := identityCollectionFixture(t, spec, "ID string;Name string", "string", "name")
			if identityListModeEnabled(pkg, plan) {
				t.Fatal("identity capability leaked into list modes", spec)
			}
		})
	}
}

func TestIdentityListModesEmitFallbackOnlyAndExplicitInventory(t *testing.T) {
	for _, spec := range identityListModeSpecs {
		t.Run(spec.path, func(t *testing.T) {
			pkg, plan := identityModeFixture(t, spec, identityModeFixtureSource(spec))
			e := emitter{pkg: pkg, imports: map[string]string{}}
			e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", spec.model)
			emitCollectionAdapter(&e, plan, "a", nil)
			e.printf("}\n")
			source, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			body := string(source)
			for _, want := range []string{"IdentityAllProjectsQuery: \"all_tenants\"", "func(ctx context.Context, q url.Values, details bool) iter.Seq2[*" + spec.model + ", error]", "return nativefind." + spec.iterator + "(ctx, a.RawClient(), q, details)", "a.Get(ctx, string(id))", "return result.Extract()", "IterateControlled:", "config.Query[key] = append([]string(nil), values...)"} {
				if !strings.Contains(body, want) {
					t.Fatalf("missing %q:\n%s", want, body)
				}
			}
			if strings.Contains(body, "WithListDetails") || strings.Contains(body, "WithListAllProjects") {
				t.Fatal("changed public native List options", body)
			}
			record := collectionRecord{Package: "gophercloudsdk/" + spec.path, IdentityDetails: identityListModeEnabled(pkg, plan), IdentityAllProjects: identityListModeEnabled(pkg, plan)}
			data, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{`"identity_details":true`, `"identity_all_projects":true`} {
				if !strings.Contains(string(data), field) {
					t.Fatal("unrecorded audited controls", string(data))
				}
			}
		})
	}
	data, err := json.Marshal(collectionRecord{Package: "unrelated"})
	if err != nil || strings.Contains(string(data), "identity_details") || strings.Contains(string(data), "identity_all_projects") {
		t.Fatal("inventory default falsely advertised modes", string(data), err)
	}
}

func pinnedIdentityModeDeclarations(t *testing.T, spec identityListModeSpec) map[string]*ast.FuncDecl {
	t.Helper()
	request := `func List(client *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listDetailURL(client)
	if opts != nil {
		query, err := opts.ToServerListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(client, url, func(r pagination.PageResult) pagination.Page {
		return ServerPage{pagination.LinkedPageBase{PageResult: r}}
	})
}`
	urls := `func createURL(client *gophercloud.ServiceClient) string {
	return client.ServiceURL("servers")
}
func listURL(client *gophercloud.ServiceClient) string {
	return createURL(client)
}
func listDetailURL(client *gophercloud.ServiceClient) string {
	return client.ServiceURL("servers", "detail")
}`
	pages := `func (r ServerPage) IsEmpty() (bool, error) {
	if r.StatusCode == 204 {
		return true, nil
	}

	s, err := ExtractServers(r)
	return len(s) == 0, err
}
func (r ServerPage) NextPageURL() (string, error) {
	var s struct {
		Links []gophercloud.Link ` + "`json:\"servers_links\"`" + `
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return gophercloud.ExtractNextURL(s.Links)
}
func ExtractServers(r pagination.Page) ([]Server, error) {
	var s []Server
	err := ExtractServersInto(r, &s)
	return s, err
}
func ExtractServersInto(r pagination.Page, v any) error {
	return r.(ServerPage).ExtractIntoSlicePtr(v, "servers")
}`
	if spec.model == "Server" {
		request += "\n" + strings.Replace(strings.Replace(request, "func List(", "func ListSimple(", 1), "listDetailURL(client)", "listURL(client)", 1)
	} else {
		request = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(request, "Server", "Volume"), "listDetailURL", "listURL"), "\n\treturn pagination.NewPager", "\n\n\treturn pagination.NewPager")
		urls = `func createURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("volumes")
}
func listURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("volumes", "detail")
}`
		pages = strings.ReplaceAll(strings.ReplaceAll(pages, "Server", "Volume"), "servers", "volumes")
		pages = strings.Replace(pages, "s, err := ExtractVolumes(r)\n\treturn len(s)", "volumes, err := ExtractVolumes(r)\n\treturn len(volumes)", 1)
		pages = strings.Replace(pages, "func (r VolumePage) NextPageURL()", "func (page VolumePage) NextPageURL()", 1)
		pages = strings.Replace(pages, "err := r.ExtractInto(&s)", "err := page.ExtractInto(&s)", 1)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "native-modes.go", "package fixture\n"+request+"\n"+urls+"\n"+pages, 0)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			decls[identityDeclarationKey(fn)] = fn
		}
	}
	return decls
}

func TestIdentityListModesNativeDeclarationAndReceiverGuards(t *testing.T) {
	for _, spec := range identityListModeSpecs {
		t.Run(spec.path, func(t *testing.T) {
			pkg, plan := identityModeFixture(t, spec, identityModeFixtureSource(spec))
			decls := pinnedIdentityModeDeclarations(t, spec)
			if err := validateIdentityListModeContracts(pkg, decls, plan); err != nil {
				t.Fatal("pinned native declaration mismatch", err)
			}
			for name := range spec.declarations {
				t.Run("missing/"+name, func(t *testing.T) {
					fresh := pinnedIdentityModeDeclarations(t, spec)
					delete(fresh, name)
					if err := validateIdentityListModeContracts(pkg, fresh, plan); err == nil || !strings.Contains(err.Error(), "declaration "+name+" changed") {
						t.Fatal(err)
					}
				})
				t.Run("changed/"+name, func(t *testing.T) {
					fresh := pinnedIdentityModeDeclarations(t, spec)
					fresh[name].Body.List = append(fresh[name].Body.List, &ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("changedRoutePageOrExtraction")}})
					if err := validateIdentityListModeContracts(pkg, fresh, plan); err == nil || !strings.Contains(err.Error(), "declaration "+name+" changed") {
						t.Fatal(err)
					}
				})
			}
		})
	}
	file, err := parser.ParseFile(token.NewFileSet(), "receivers.go", `package fixture
func NextPageURL(){}
func (a ServerPage) NextPageURL(){}
func (a *VolumePage) NextPageURL(){}`, 0)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		key := identityDeclarationKey(fn)
		if keys[key] {
			t.Fatal("receiver declarations collided", key)
		}
		keys[key] = true
	}
	for _, key := range []string{"NextPageURL", "ServerPage.NextPageURL", "VolumePage.NextPageURL"} {
		if !keys[key] {
			t.Fatal("receiver-qualified audit key missing", key, keys)
		}
	}
}
