package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func identityQueryFixtureSource(spec identityCollectionSpec) string {
	getArgs, listArgs := "id string", ",opts ListOptsBuilder"
	if spec.parents != 0 {
		getArgs = "parentID string," + getArgs
		listArgs = ",parentID string,opts ListOptsBuilder"
	}
	opts := "type ListOpts struct{Name string `q:\"name\"`}\ntype ListOptsBuilder interface{ToListQuery()(string,error)}\nfunc(ListOpts)ToListQuery()(string,error){return \"\",nil}"
	source := controlledFixtureSource(spec.model, "ID string;Name string", spec.getter, getArgs, spec.lister, listArgs, opts)
	if spec.path == "image/v2/images" {
		source = strings.Replace(source, "type Image struct{ID string;Name string}", "type Image struct{ID string;Name string;Hidden bool `json:\"os_hidden\"`}", 1)
		source = strings.Replace(source, "type ListOpts struct{Name string `q:\"name\"`}", "type ListOpts struct{Name string `q:\"name\"`;Hidden bool `q:\"os_hidden\"`}", 1)
		source = strings.ReplaceAll(source, "ModelPage", "ImagePage")
		source = strings.Replace(source, "type ImagePage struct{}", "type ImagePage struct{serviceURL string;pagination.LinkedPageBase}", 1)
		source = strings.ReplaceAll(source, "ExtractModels", "ExtractImages")
		source += "\nfunc(ImagePage)IsEmpty()(bool,error){return false,nil}\nfunc(ImagePage)NextPageURL()(string,error){return \"\",nil}\n"
	}
	if spec.path == "compute/v2/flavors" {
		source = strings.Replace(source, "type Flavor struct{ID string;Name string}", "type Flavor struct{ID string;Name string;ExtraSpecs map[string]string `json:\"extra_specs\"`}", 1)
		source = strings.Replace(source, "type ListOpts struct{Name string `q:\"name\"`}", "type AccessType string\ntype ListOpts struct{AccessType AccessType `q:\"is_public\"`}", 1)
		source = strings.ReplaceAll(source, "ModelPage", "FlavorPage")
		source = strings.Replace(source, "type FlavorPage struct{}", "type FlavorPage struct{pagination.LinkedPageBase}", 1)
		source = strings.ReplaceAll(source, "ExtractModels", "ExtractFlavors")
		source += "\nfunc(FlavorPage)IsEmpty()(bool,error){return false,nil}\nfunc(FlavorPage)NextPageURL()(string,error){return \"\",nil}\n"
		source += "type ListExtraSpecsResult struct{Body any;Header http.Header;Err error}\nfunc(ListExtraSpecsResult)Extract()(map[string]string,error){return nil,nil}\nfunc ListExtraSpecs(ctx context.Context,client *gophercloud.ServiceClient,id string)ListExtraSpecsResult{return ListExtraSpecsResult{}}\n"
	}
	// Member uses GetMemberResult upstream; use its actual signature instead of
	// allowing the emitter to assume every native getter returns GetResult.
	source = strings.ReplaceAll(source, "GetResult", spec.getter+"Result")
	for name, value := range identityNativeURLConstants[spec.path] {
		source += fmt.Sprintf("\nconst %s=%q\n", name, value)
	}
	return source
}

func identityQueryFixture(t *testing.T, spec identityCollectionSpec, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	path := spec.path
	if strings.HasPrefix(path, "network/") {
		path = "networking/" + strings.TrimPrefix(path, "network/")
	}
	pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/"+path, source)
	plan := identifyNamedCollection(pkg, decls, extractorsByPage(pkg, decls), spec.getter, []string{spec.lister}, "Delete", spec.parents)
	return pkg, plan
}

func identityAuditPlans(spec identityCollectionSpec, plan *collectionPlan) (*collectionPlan, []scopePlan) {
	if spec.parents == 0 {
		return plan, nil
	}
	return nil, []scopePlan{{collection: plan}}
}

func TestIdentityGetQueryUsesAuditedNativeRoutesCodesAndResult(t *testing.T) {
	routes := map[string]string{
		"compute/v2/servers":      `[]string{"servers", id}, q, []int{200, 203}`,
		"compute/v2/flavors":      `[]string{"flavors", id}, q, []int{200}`,
		"blockstorage/v3/volumes": `[]string{"volumes", id}, q, []int{200}`,
		"network/v2/ports":        `[]string{"ports", id}, q, []int{200}`,
		"network/v2/networks":     `[]string{"networks", id}, q, []int{200}`,
		"network/v2/subnets":      `[]string{"subnets", id}, q, []int{200}`,
		"identity/v3/projects":    `[]string{"projects", id}, q, []int{200}`,
		"identity/v3/users":       `[]string{"users", id}, q, []int{200}`,
		"identity/v3/groups":      `[]string{"groups", id}, q, []int{200}`,
		"identity/v3/domains":     `[]string{"domains", id}, q, []int{200}`,
		"identity/v3/roles":       `[]string{"roles", id}, q, []int{200}`,
		"dns/v2/recordsets":       `[]string{"zones", s.parentID, "recordsets", id}, q, []int{200}`,
		"loadbalancer/v2/pools":   `[]string{"lbaas", "pools", s.parentID, "members", id}, q, []int{200}`,
		"image/v2/images":         `[]string{"images", id}, q, []int{200}`,
	}
	if len(identityCollectionSpecs) != 14 || len(identityNativeDeclarations) != 14 {
		t.Fatal("identity opt-in inventory must remain explicit", len(identityCollectionSpecs), len(identityNativeDeclarations))
	}
	for _, spec := range identityCollectionSpecs {
		t.Run(spec.path, func(t *testing.T) {
			pkg, plan := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
			if !identityCollectionEnabled(pkg, plan, spec.parents) {
				t.Fatal("native-shaped fixture unexpectedly disabled", spec)
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			receiver, parents := "a", []string(nil)
			if spec.parents != 0 {
				receiver, parents = "s.api", []string{"s.parentID"}
			}
			e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", spec.model)
			emitCollectionAdapter(&e, plan, receiver, parents)
			e.printf("}\n")
			source, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			body := string(source)
			for _, want := range []string{
				"GetIdentityQuery: func(ctx context.Context, id string, q url.Values)",
				"var result upstream." + spec.getter + "Result",
				"result.Header, result.Err = nativefind.Get(ctx, " + receiver + ".RawClient(), " + routes[spec.path] + ", &result.Body)",
				"return result.Extract()",
				"config.Query[key] = append([]string(nil), values...)",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("lost native result/routing/query contract %q:\n%s", want, body)
				}
			}
			if strings.Contains(body, "WithListQuery(") || strings.Contains(body, "WithListMembersQuery(") || strings.Contains(body, "WithListByZoneQuery(") {
				t.Fatal("audited query map collapsed to scalar options", body)
			}
			get := controlledEmittedMethod(t, source, "newResources")
			if spec.parents == 0 {
				requireControlledCalls(t, get, "a.Get(ctx, string(id))")
			} else {
				requireControlledCalls(t, get, "s.api."+spec.getter+"(ctx, s.parentID, string(id))")
			}
		})
	}
}

func TestIdentityGetQueryRejectsNativeResultAndSignatureDrift(t *testing.T) {
	spec := identityCollectionSpecs[0]
	base := identityQueryFixtureSource(spec)
	mutations := map[string]func(string) string{
		"missing body":   func(s string) string { return strings.Replace(s, "Body any;", "", 1) },
		"typed body":     func(s string) string { return strings.Replace(s, "Body any;", "Body string;", 1) },
		"missing header": func(s string) string { return strings.Replace(s, "Header http.Header;", "", 1) },
		"different header": func(s string) string {
			return strings.Replace(s, "Header http.Header;", "Header map[string]string;", 1)
		},
		"different error":    func(s string) string { return strings.Replace(s, "Err error", "Err string", 1) },
		"nonexported result": func(s string) string { return strings.ReplaceAll(s, "GetResult", "getResult") },
		"named extractor only": func(s string) string {
			return strings.Replace(s, "func(GetResult)Extract()", "func(GetResult)ExtractServer()", 1)
		},
		"new getter option": func(s string) string {
			return strings.Replace(s, "id string)GetResult", "id string,opts ListOptsBuilder)GetResult", 1)
		},
		"client/context reorder": func(s string) string {
			return strings.Replace(s, "ctx context.Context,client *gophercloud.ServiceClient,id", "client *gophercloud.ServiceClient,ctx context.Context,id", 1)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			source := mutate(base)
			// Keep imports used after deleting/changing the header field.
			source += "\nvar _ http.Header\n"
			pkg, plan := identityQueryFixture(t, spec, source)
			if identityCollectionEnabled(pkg, plan, spec.parents) {
				t.Fatal("GET query hook survived native signature/result drift", name)
			}
			if err := validateIdentityCollectionContracts(pkg, nil, plan, nil, nil); err == nil || !strings.Contains(err.Error(), "schema changed") {
				t.Fatal("audited schema drift silently removed support", err)
			}
		})
	}
	// Actual upstream result fields are promoted through embedded wrappers.
	source := strings.Replace(base, "type GetResult struct{Body any;Header http.Header;Err error}", "type NativeResult struct{Body any;Header http.Header;Err error}\ntype GetResult struct{NativeResult}", 1)
	pkg, plan := identityQueryFixture(t, spec, source)
	if !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal("valid promoted native result fields were rejected")
	}
}

func pinnedIdentityDeclarations(t *testing.T, spec identityCollectionSpec) map[string]*ast.FuncDecl {
	t.Helper()
	defaultGet := `func Get(ctx context.Context, client *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := client.Get(ctx, getURL(client, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`
	request := defaultGet
	var urls string
	switch spec.path {
	case "image/v2/images":
		return pinnedImageIdentityDeclarations(t)
	case "compute/v2/flavors":
		return pinnedFlavorIdentityDeclarations(t)
	case "compute/v2/servers":
		request = strings.Replace(defaultGet, "&r.Body, nil", "&r.Body, &gophercloud.RequestOpts{\n\t\tOkCodes: []int{200, 203},\n\t}", 1)
		urls = `func getURL(client *gophercloud.ServiceClient, id string) string { return deleteURL(client,id) }
func deleteURL(client *gophercloud.ServiceClient, id string) string { return client.ServiceURL("servers",id) }`
	case "blockstorage/v3/volumes":
		urls = `func getURL(c *gophercloud.ServiceClient, id string) string { return deleteURL(c,id) }
func deleteURL(c *gophercloud.ServiceClient, id string) string { return c.ServiceURL("volumes",id) }`
	case "network/v2/ports", "network/v2/networks", "network/v2/subnets":
		request = strings.ReplaceAll(defaultGet, "client", "c")
		urls = fmt.Sprintf(`func getURL(c *gophercloud.ServiceClient, id string) string { return resourceURL(c,id) }
func resourceURL(c *gophercloud.ServiceClient, id string) string { return c.ServiceURL(%q,id) }`, spec.getSegments[0])
	case "identity/v3/projects", "identity/v3/users", "identity/v3/groups", "identity/v3/domains":
		id := strings.ToLower(spec.model) + "ID"
		urls = fmt.Sprintf(`func getURL(client *gophercloud.ServiceClient, %s string) string { return client.ServiceURL(%q,%s) }`, id, spec.getSegments[0], id)
	case "identity/v3/roles":
		urls = `func getURL(client *gophercloud.ServiceClient, roleID string) string { return client.ServiceURL(rolePath,roleID) }`
	case "dns/v2/recordsets":
		request = `func Get(ctx context.Context, client *gophercloud.ServiceClient, zoneID string, rrsetID string) (r GetResult) {
	resp, err := client.Get(ctx, rrsetURL(client, zoneID, rrsetID), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`
		urls = `func rrsetURL(c *gophercloud.ServiceClient, zoneID string, rrsetID string) string { return c.ServiceURL("zones",zoneID,"recordsets",rrsetID) }`
	case "loadbalancer/v2/pools":
		request = `func GetMember(ctx context.Context, c *gophercloud.ServiceClient, poolID string, memberID string) (r GetMemberResult) {
	resp, err := c.Get(ctx, memberResourceURL(c, poolID, memberID), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`
		urls = `func memberResourceURL(c *gophercloud.ServiceClient, poolID string, memberID string) string { return c.ServiceURL(rootPath,resourcePath,poolID,memberPath,memberID) }`
	default:
		t.Fatal("missing pinned request fixture", spec)
	}
	urls = strings.ReplaceAll(urls, " { return", " {\n\treturn")
	urls = strings.ReplaceAll(urls, " }", "\n}")
	file, err := parser.ParseFile(token.NewFileSet(), "native.go", "package fixture\n"+request+"\n"+urls, 0)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			decls[fn.Name.Name] = fn
		}
	}
	return decls
}

func identityQuerySourceConstants(t *testing.T, source string) map[string]string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "constants.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	return identitySourceConstants(file)
}

func TestIdentityCollectionContractsRejectRouteCodeAndConstantDrift(t *testing.T) {
	for _, spec := range identityCollectionSpecs {
		t.Run(spec.path, func(t *testing.T) {
			base := identityQueryFixtureSource(spec)
			pkg, plan := identityQueryFixture(t, spec, base)
			global, scopes := identityAuditPlans(spec, plan)
			decls := pinnedIdentityDeclarations(t, spec)
			if err := validateIdentityCollectionContracts(pkg, decls, global, scopes, identityQuerySourceConstants(t, base)); err != nil {
				t.Fatal("pinned native declaration mismatch", err)
			}
			t.Run("different accepted GET code", func(t *testing.T) {
				fresh := pinnedIdentityDeclarations(t, spec)
				options, err := parser.ParseExpr("&gophercloud.RequestOpts{OkCodes:[]int{202}}")
				if err != nil {
					t.Fatal(err)
				}
				changed := false
				ast.Inspect(fresh[spec.getter], func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok || len(call.Args) != 4 {
						return true
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if ok && selector.Sel.Name == "Get" {
						call.Args[3] = options
						changed = true
					}
					return true
				})
				if !changed {
					t.Fatal("missing native GET options")
				}
				if err := validateIdentityCollectionContracts(pkg, fresh, global, scopes, identityQuerySourceConstants(t, base)); err == nil || !strings.Contains(err.Error(), "declaration "+spec.getter+" changed") {
					t.Fatal("native accepted-code drift ignored", err)
				}
			})
			for name := range identityNativeDeclarations[spec.path] {
				t.Run("missing "+name, func(t *testing.T) {
					fresh := pinnedIdentityDeclarations(t, spec)
					delete(fresh, name)
					if err := validateIdentityCollectionContracts(pkg, fresh, global, scopes, identityQuerySourceConstants(t, base)); err == nil || !strings.Contains(err.Error(), "declaration "+name+" changed") {
						t.Fatal("missing native declaration ignored", err)
					}
				})
				t.Run("changed "+name, func(t *testing.T) {
					fresh := pinnedIdentityDeclarations(t, spec)
					fresh[name].Body.List = append(fresh[name].Body.List, &ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("changedRouteOrCode")}})
					if err := validateIdentityCollectionContracts(pkg, fresh, global, scopes, identityQuerySourceConstants(t, base)); err == nil || !strings.Contains(err.Error(), "declaration "+name+" changed") {
						t.Fatal("native body/codes/URL drift ignored", err)
					}
				})
			}
			for name, value := range identityNativeURLConstants[spec.path] {
				t.Run("constant "+name, func(t *testing.T) {
					altered := strings.Replace(base, fmt.Sprintf("const %s=%q", name, value), fmt.Sprintf("const %s=%q", name, "changed"), 1)
					badPkg, badPlan := identityQueryFixture(t, spec, altered)
					badGlobal, badScopes := identityAuditPlans(spec, badPlan)
					if err := validateIdentityCollectionContracts(badPkg, decls, badGlobal, badScopes, identityQuerySourceConstants(t, altered)); err == nil || !strings.Contains(err.Error(), "URL constant "+name+" changed") {
						t.Fatal("native URL constant drift ignored", err)
					}
				})
			}
		})
	}
}
