package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// These pinned declarations protect the native routes, concrete-versus-builder
// list signatures, timestamps, singular envelopes and page/extractor policy.
const pinnedRouterIdentitySource = `package fixture
func (opts ListOpts) ToRouterListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(&opts)
	if err != nil {
		return "", err
	}
	return q.String(), nil
}
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := rootURL(c)
	if opts != nil {
		query, err := opts.ToRouterListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return RouterPage{pagination.LinkedPageBase{PageResult: r}}
	})
}
func Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := c.Get(ctx, resourceURL(c, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
func (r *Router) UnmarshalJSON(b []byte) error {
	type tmp Router

	// Support for older neutron time format
	var s1 struct {
		tmp
		CreatedAt gophercloud.JSONRFC3339NoZ ` + "`" + `json:"created_at"` + "`" + `
		UpdatedAt gophercloud.JSONRFC3339NoZ ` + "`" + `json:"updated_at"` + "`" + `
	}

	err := json.Unmarshal(b, &s1)
	if err == nil {
		*r = Router(s1.tmp)
		r.CreatedAt = time.Time(s1.CreatedAt)
		r.UpdatedAt = time.Time(s1.UpdatedAt)

		return nil
	}

	// Support for newer neutron time format
	var s2 struct {
		tmp
		CreatedAt time.Time ` + "`" + `json:"created_at"` + "`" + `
		UpdatedAt time.Time ` + "`" + `json:"updated_at"` + "`" + `
	}

	err = json.Unmarshal(b, &s2)
	if err != nil {
		return err
	}

	*r = Router(s2.tmp)
	r.CreatedAt = time.Time(s2.CreatedAt)
	r.UpdatedAt = time.Time(s2.UpdatedAt)

	return nil
}
func (r RouterPage) NextPageURL() (string, error) {
	var s struct {
		Links []gophercloud.Link ` + "`" + `json:"routers_links"` + "`" + `
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return gophercloud.ExtractNextURL(s.Links)
}
func (r RouterPage) IsEmpty() (bool, error) {
	if r.StatusCode == 204 {
		return true, nil
	}

	is, err := ExtractRouters(r)
	return len(is) == 0, err
}
func ExtractRouters(r pagination.Page) ([]Router, error) {
	var s []Router
	err := ExtractRoutersInto(r, &s)
	return s, err
}
func ExtractRoutersInto(r pagination.Page, v any) error {
	return r.(RouterPage).ExtractIntoSlicePtr(v, "routers")
}
func (r commonResult) Extract() (*Router, error) {
	var s struct {
		Router *Router ` + "`" + `json:"router"` + "`" + `
	}
	err := r.ExtractInto(&s)
	return s.Router, err
}
func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL(resourcePath)
}
func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(resourcePath, id)
}`
const pinnedSecGroupIdentitySource = `package fixture
func List(c *gophercloud.ServiceClient, opts ListOpts) pagination.Pager {
	q, err := gophercloud.BuildQueryString(&opts)
	if err != nil {
		return pagination.Pager{Err: err}
	}
	u := rootURL(c) + q.String()
	return pagination.NewPager(c, u, func(r pagination.PageResult) pagination.Page {
		return SecGroupPage{pagination.LinkedPageBase{PageResult: r}}
	})
}
func Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := c.Get(ctx, resourceURL(c, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
func (r *SecGroup) UnmarshalJSON(b []byte) error {
	type tmp SecGroup

	// Support for older neutron time format
	var s1 struct {
		tmp
		CreatedAt gophercloud.JSONRFC3339NoZ ` + "`" + `json:"created_at"` + "`" + `
		UpdatedAt gophercloud.JSONRFC3339NoZ ` + "`" + `json:"updated_at"` + "`" + `
	}

	err := json.Unmarshal(b, &s1)
	if err == nil {
		*r = SecGroup(s1.tmp)
		r.CreatedAt = time.Time(s1.CreatedAt)
		r.UpdatedAt = time.Time(s1.UpdatedAt)

		return nil
	}

	// Support for newer neutron time format
	var s2 struct {
		tmp
		CreatedAt time.Time ` + "`" + `json:"created_at"` + "`" + `
		UpdatedAt time.Time ` + "`" + `json:"updated_at"` + "`" + `
	}

	err = json.Unmarshal(b, &s2)
	if err != nil {
		return err
	}

	*r = SecGroup(s2.tmp)
	r.CreatedAt = time.Time(s2.CreatedAt)
	r.UpdatedAt = time.Time(s2.UpdatedAt)

	return nil
}
func (r SecGroupPage) NextPageURL() (string, error) {
	var s struct {
		Links []gophercloud.Link ` + "`" + `json:"security_groups_links"` + "`" + `
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}

	return gophercloud.ExtractNextURL(s.Links)
}
func (r SecGroupPage) IsEmpty() (bool, error) {
	if r.StatusCode == 204 {
		return true, nil
	}

	is, err := ExtractGroups(r)
	return len(is) == 0, err
}
func ExtractGroups(r pagination.Page) ([]SecGroup, error) {
	var s struct {
		SecGroups []SecGroup ` + "`" + `json:"security_groups"` + "`" + `
	}
	err := (r.(SecGroupPage)).ExtractInto(&s)
	return s.SecGroups, err
}
func (r commonResult) Extract() (*SecGroup, error) {
	var s struct {
		SecGroup *SecGroup ` + "`" + `json:"security_group"` + "`" + `
	}
	err := r.ExtractInto(&s)
	return s.SecGroup, err
}
func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL(rootPath)
}
func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(rootPath, id)
}`

func pinnedNeutronExtensionIdentityDeclarations(t *testing.T, spec identityCollectionSpec) map[string]*ast.FuncDecl {
	t.Helper()
	source := pinnedRouterIdentitySource
	if spec.model == "SecGroup" {
		source = pinnedSecGroupIdentitySource
	}
	file, err := parser.ParseFile(token.NewFileSet(), "native-network-extension.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			result[identityDeclarationKey(fn)] = fn
		}
	}
	return result
}

func neutronExtensionIdentitySpecs() []identityCollectionSpec {
	var result []identityCollectionSpec
	for _, spec := range identityCollectionSpecs {
		if spec.path == "network/v2/extensions/layer3/routers" || spec.path == "network/v2/extensions/security/groups" {
			result = append(result, spec)
		}
	}
	return result
}

func TestIdentityNeutronRawListExceptionOnlyEnablesConcreteSecurityGroups(t *testing.T) {
	if len(neutronExtensionIdentitySpecs()) != 2 {
		t.Fatal("missing Neutron opt-ins")
	}
	var group identityCollectionSpec
	for _, spec := range identityCollectionSpecs {
		if spec.model == "SecGroup" {
			group = spec
		}
		if !identityRawListMetadataValid(spec) {
			t.Fatal("reviewed metadata rejected", spec)
		}
	}
	mutations := map[string]func(*identityCollectionSpec){
		"model":             func(s *identityCollectionSpec) { s.model = "SecurityGroup" },
		"get":               func(s *identityCollectionSpec) { s.getter = "Fetch" },
		"list":              func(s *identityCollectionSpec) { s.lister = "ListDetail" },
		"parent":            func(s *identityCollectionSpec) { s.parents = 1 },
		"route":             func(s *identityCollectionSpec) { s.getSegments = []string{"security_groups", "$id"} },
		"id route":          func(s *identityCollectionSpec) { s.getSegments = []string{"security-groups", "fixed"} },
		"subroute":          func(s *identityCollectionSpec) { s.getSegments = []string{"security-groups", "$id", "rules"} },
		"codes":             func(s *identityCollectionSpec) { s.getCodes = []int{200, 203} },
		"missing helper":    func(s *identityCollectionSpec) { s.rawListIterator = "" },
		"wrong helper":      func(s *identityCollectionSpec) { s.rawListIterator = "IterateServers" },
		"missing name hint": func(s *identityCollectionSpec) { s.noNameQuery = true },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			altered := group
			mutate(&altered)
			if identityRawListMetadataValid(altered) {
				t.Fatal("unsafe concrete-list exception accepted", altered)
			}
		})
	}
	for _, spec := range identityCollectionSpecs {
		if spec.path == group.path {
			continue
		}
		spec.rawListIterator = "IterateSecurityGroups"
		if identityRawListMetadataValid(spec) {
			t.Fatal("raw-list policy leaked", spec)
		}
	}
}

func TestIdentityNeutronPageExtractorAndNativeListDriftFailGeneration(t *testing.T) {
	for _, spec := range neutronExtensionIdentitySpecs() {
		t.Run(spec.model, func(t *testing.T) {
			base := identityQueryFixtureSource(spec)
			pkg, plan := identityQueryFixture(t, spec, base)
			if !identityNeutronListSchema(pkg, plan) {
				t.Fatal("native schema rejected")
			}
			decls := pinnedNeutronExtensionIdentityDeclarations(t, spec)
			if err := validateIdentityCollectionContracts(pkg, decls, plan, nil, identityQuerySourceConstants(t, base)); err != nil {
				t.Fatal(err)
			}
			page, extractor := "RouterPage", "ExtractRouters"
			if spec.model == "SecGroup" {
				page, extractor = "SecGroupPage", "ExtractGroups"
			}
			mutations := map[string]func(string) string{
				"extra page state": func(s string) string {
					return strings.Replace(s, page+" struct{pagination.LinkedPageBase}", page+" struct{extra bool;pagination.LinkedPageBase}", 1)
				},
				"nonembedded page": func(s string) string {
					return strings.Replace(s, page+" struct{pagination.LinkedPageBase}", page+" struct{base pagination.LinkedPageBase}", 1)
				},
				"wrong empty result": func(s string) string {
					return strings.Replace(s, "IsEmpty()(bool,error){return false,nil}", "IsEmpty()(int,error){return 0,nil}", 1)
				},
				"continuation parameter": func(s string) string { return strings.Replace(s, "NextPageURL()", "NextPageURL(bool)", 1) },
				"wrong continuation result": func(s string) string {
					return strings.Replace(s, "NextPageURL()(string,error){return \"\",nil}", "NextPageURL()(int,error){return 0,nil}", 1)
				},
				"wrong list extractor":    func(s string) string { return strings.Replace(s, "([]"+spec.model+",error)", "([]string,error)", 1) },
				"missing named extractor": func(s string) string { return strings.Replace(s, "func "+extractor+"(", "func ExtractOther(", 1) },
			}
			if spec.model == "Router" {
				mutations["wrong into destination"] = func(s string) string { return strings.Replace(s, "v any)error", "v string)error", 1) }
				mutations["missing into extractor"] = func(s string) string {
					return strings.Replace(s, "func ExtractRoutersInto(", "func ExtractOtherInto(", 1)
				}
				mutations["unexpected concrete list"] = func(s string) string { return strings.Replace(s, "opts ListOptsBuilder)", "opts ListOpts)", 1) }
			} else {
				mutations["unexpected builder list"] = func(s string) string { return strings.Replace(s, "opts ListOpts)", "opts ListOptsBuilder)", 1) }
				mutations["unexpected renamed opts"] = func(s string) string {
					return strings.ReplaceAll(s, "type ListOpts struct", "type ConcreteOpts struct") + "\ntype ListOpts=ConcreteOpts\n"
				}
			}
			for name, mutate := range mutations {
				t.Run(name, func(t *testing.T) {
					changed := mutate(base)
					if changed == base {
						t.Fatal("mutation did not apply")
					}
					badPkg, badPlan := identityQueryFixture(t, spec, changed)
					if err := validateIdentityCollectionContracts(badPkg, decls, badPlan, nil, identityQuerySourceConstants(t, changed)); err == nil || !strings.Contains(err.Error(), "schema changed") {
						t.Fatal("native schema drift ignored", err)
					}
				})
			}
		})
	}
}

func TestIdentityNeutronEmissionPreservesRawQueriesAndExistingCapabilities(t *testing.T) {
	rawCount := 0
	for _, spec := range identityCollectionSpecs {
		t.Run(spec.path, func(t *testing.T) {
			pkg, plan := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
			for _, mode := range identityListModeSpecs {
				if mode.path == spec.path {
					pkg, plan = identityModeFixture(t, mode, identityModeFixtureSource(mode))
				}
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			parents := []string(nil)
			if spec.parents != 0 {
				parents = []string{"a.parentID"}
			}
			e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", spec.model)
			emitCollectionAdapter(&e, plan, "a", parents)
			e.printf("}\n")
			data, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			body := string(data)
			if spec.rawListIterator != "" {
				rawCount++
				for _, want := range []string{"IdentityFind: true", "GetIdentityQuery:", `[]string{"security-groups", id}, q, []int{200}`, "return result.Extract()", "NameQuery:", "return nativefind.IterateSecurityGroups(ctx, a.RawClient(), q, control)"} {
					if !strings.Contains(body, want) {
						t.Fatal("lost native SecurityGroup contract", want, body)
					}
				}
				for _, bad := range []string{"request.QueryOptions", `q.Del("status")`, "WithListOptions(input)", "IterateIdentity:", "IdentityAllProjectsQuery:", "IdentityExtraSpecs:", "IdentityMissingListQuery:", "IdentityListQueryDefaults:"} {
					if strings.Contains(body, bad) {
						t.Fatal("concrete raw-list gained coercion or unrelated policy", bad, body)
					}
				}
			} else if strings.Contains(body, "nativefind.IterateSecurityGroups(") {
				t.Fatal("raw helper leaked", body)
			}
			if spec.model == "Router" {
				for _, want := range []string{`[]string{"routers", id}, q, []int{200}`, "config.Query[key] = append([]string(nil), values...)", "a.listWithControl(ctx, control, options...)"} {
					if !strings.Contains(body, want) {
						t.Fatal("lost standard Router policy", want, body)
					}
				}
				if strings.Contains(body, "request.QueryOptions") {
					t.Fatal("Router raw query coerced", body)
				}
			}
		})
	}
	if len(identityCollectionSpecs) != 20 || rawCount != 1 {
		t.Fatal("explicit audited scope changed", len(identityCollectionSpecs), rawCount)
	}
}
