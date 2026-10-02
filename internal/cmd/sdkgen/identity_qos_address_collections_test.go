package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func qosAddressIdentitySpecs() []identityCollectionSpec {
	result := []identityCollectionSpec{}
	for _, spec := range identityCollectionSpecs {
		if spec.path == "network/v2/extensions/qos/policies" || spec.path == "network/v2/extensions/security/addressgroups" {
			result = append(result, spec)
		}
	}
	return result
}

func pinnedQoSAddressIdentityDeclarations(t *testing.T, spec identityCollectionSpec) map[string]*ast.FuncDecl {
	t.Helper()
	source := pinnedAddressGroupIdentitySource
	if spec.model == "Policy" {
		source = pinnedQoSIdentitySource
	}
	file, err := parser.ParseFile(token.NewFileSet(), "native.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok {
			result[identityDeclarationKey(fn)] = fn
		}
	}
	return result
}

func TestIdentityQoSAddressNativeSchemaRejectsModelPagerAndExtractorDrift(t *testing.T) {
	if len(qosAddressIdentitySpecs()) != 2 {
		t.Fatal("missing audited QoS/address-group bindings")
	}
	for _, spec := range qosAddressIdentitySpecs() {
		t.Run(spec.model, func(t *testing.T) {
			base := identityQueryFixtureSource(spec)
			pkg, plan := identityQueryFixture(t, spec, base)
			if !identityQoSAddressSchema(pkg, plan) {
				t.Fatal("native model or page rejected")
			}
			decls := pinnedQoSAddressIdentityDeclarations(t, spec)
			if err := validateIdentityCollectionContracts(pkg, decls, plan, nil, identityQuerySourceConstants(t, base)); err != nil {
				t.Fatal(err)
			}
			page, extractor, builder := "AddressGroupPage", "ExtractGroups", "ListOptsBuilder"
			if spec.model == "Policy" {
				page, extractor, builder = "PolicyPage", "ExtractPolicies", "PolicyListOptsBuilder"
			}
			mutations := map[string]func(string) string{
				"changed ID wire key": func(s string) string {
					return strings.Replace(s, "ID string `json:\"id\"`", "ID string `json:\"uuid\"`", 1)
				},
				"changed name wire key": func(s string) string {
					return strings.Replace(s, "Name string `json:\"name\"`", "Name string `json:\"label\"`", 1)
				},
				"nullable project": func(s string) string { return strings.Replace(s, "ProjectID string `", "ProjectID *string `", 1) },
				"additional model state": func(s string) string {
					return strings.Replace(s, "type "+spec.model+" struct{", "type "+spec.model+" struct{Changed bool;", 1)
				},
				"new model decoder": func(s string) string {
					return s + "\nfunc(*" + spec.model + ")UnmarshalJSON([]byte)error{return nil}\n"
				},
				"named page base": func(s string) string {
					return strings.Replace(s, page+" struct{pagination.LinkedPageBase}", page+" struct{base pagination.LinkedPageBase}", 1)
				},
				"additional page state": func(s string) string {
					return strings.Replace(s, page+" struct{pagination.LinkedPageBase}", page+" struct{extra bool;pagination.LinkedPageBase}", 1)
				},
				"inherited continuation instead of own": func(s string) string {
					return strings.Replace(s, "func("+page+")NextPageURL()(string,error){return \"\",nil}", "", 1)
				},
				"wrong continuation argument": func(s string) string { return strings.Replace(s, "NextPageURL()", "NextPageURL(bool)", 1) },
				"wrong empty result": func(s string) string {
					return strings.Replace(s, "IsEmpty()(bool,error){return false,nil}", "IsEmpty()(int,error){return 0,nil}", 1)
				},
				"wrong extractor result":           func(s string) string { return strings.Replace(s, "([]"+spec.model+",error)", "([]string,error)", 1) },
				"renamed extractor":                func(s string) string { return strings.Replace(s, "func "+extractor+"(", "func ExtractOther(", 1) },
				"concrete list instead of builder": func(s string) string { return strings.Replace(s, "opts "+builder+")", "opts ListOpts)", 1) },
			}
			if spec.model == "Policy" {
				mutations["typed rule map"] = func(s string) string {
					return strings.Replace(s, "Rules []map[string]any", "Rules []map[string]string", 1)
				}
				mutations["nullable shared"] = func(s string) string { return strings.Replace(s, "Shared bool `", "Shared *bool `", 1) }
				mutations["numeric tags"] = func(s string) string { return strings.Replace(s, "Tags []string", "Tags []int", 1) }
				mutations["string created timestamp"] = func(s string) string { return strings.Replace(s, "CreatedAt time.Time", "CreatedAt string", 1) }
				mutations["string updated timestamp"] = func(s string) string { return strings.Replace(s, "UpdatedAt time.Time", "UpdatedAt string", 1) }
				mutations["nullable revision"] = func(s string) string { return strings.Replace(s, "RevisionNumber int `", "RevisionNumber *int `", 1) }
				mutations["renamed native typo helper"] = func(s string) string {
					return strings.Replace(s, "func ExtractPolicysInto(", "func ExtractPoliciesInto(", 1)
				}
				mutations["typed helper destination"] = func(s string) string { return strings.Replace(s, "v any)error", "v []Policy)error", 1) }
			} else {
				mutations["arbitrary address values"] = func(s string) string { return strings.Replace(s, "Addresses []string", "Addresses []any", 1) }
				mutations["changed address wire key"] = func(s string) string { return strings.Replace(s, "json:\"addresses\"", "json:\"address\"", 1) }
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

func TestIdentityQoSAddressPinnedOwnLinksAndEnvelopeExtraction(t *testing.T) {
	for _, spec := range qosAddressIdentitySpecs() {
		t.Run(spec.model, func(t *testing.T) {
			base := identityQueryFixtureSource(spec)
			pkg, plan := identityQueryFixture(t, spec, base)
			validate := func(decls map[string]*ast.FuncDecl) error {
				return validateIdentityCollectionContracts(pkg, decls, plan, nil, identityQuerySourceConstants(t, base))
			}
			page := "AddressGroupPage"
			if spec.model == "Policy" {
				page = "PolicyPage"
			}
			for name, key := range map[string]string{"member envelope": "commonResult.Extract", "plural link envelope": page + ".NextPageURL"} {
				t.Run(name, func(t *testing.T) {
					decls := pinnedQoSAddressIdentityDeclarations(t, spec)
					changed := false
					ast.Inspect(decls[key], func(node ast.Node) bool {
						field, ok := node.(*ast.Field)
						if ok && field.Tag != nil {
							field.Tag.Value = "`json:\"different_envelope\"`"
							changed = true
						}
						return true
					})
					if !changed {
						t.Fatal("native envelope field not found")
					}
					if err := validate(decls); err == nil || !strings.Contains(err.Error(), "declaration "+key+" changed") {
						t.Fatal("native envelope drift ignored", err)
					}
				})
			}
			decls := pinnedQoSAddressIdentityDeclarations(t, spec)
			decls[page+".NextPageURL"].Recv.List[0].Type = ast.NewIdent("ChangedPage")
			if err := validate(decls); err == nil || !strings.Contains(err.Error(), "declaration "+page+".NextPageURL changed") {
				t.Fatal("receiver drift ignored", err)
			}
			decls = pinnedQoSAddressIdentityDeclarations(t, spec)
			decls[spec.model+".UnmarshalJSON"] = decls["commonResult.Extract"]
			if err := validate(decls); err == nil || !strings.Contains(err.Error(), "decoder") {
				t.Fatal("new source decoder escaped type/source guard", err)
			}
			// Both pages still own continuation. AddressGroup's separate raw
			// Body lane now requires number-preserving page construction, not
			// Trunk's inherited NextPageURL implementation.
			sdkGenerator := generator{}
			dependency, err := sdkGenerator.identityPaginationDeclarations(pkg.Path())
			if spec.model == "AddressGroup" {
				if err == nil || !strings.Contains(err.Error(), "AddressGroup body collection: native pagination dependency metadata missing") || len(dependency) != 0 {
					t.Fatal("raw Body dependency silently omitted", dependency, err)
				}
			} else if err != nil || len(dependency) != 0 {
				t.Fatal("inherited continuation dependency leaked", dependency, err)
			}
		})
	}
}

func TestIdentityQoSAddressEmissionKeepsRawQueriesAndBoundedCapabilities(t *testing.T) {
	for _, spec := range qosAddressIdentitySpecs() {
		t.Run(spec.model, func(t *testing.T) {
			pkg, plan := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
			e := emitter{pkg: pkg, imports: map[string]string{}}
			e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", spec.model)
			emitCollectionAdapter(&e, plan, "a", nil)
			e.printf("}\n")
			data, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			body := string(data)
			for _, want := range []string{"IdentityFind: true", "GetIdentityQuery:", "NameQuery:", "config.Query[key] = append([]string(nil), values...)", "a.listWithControl(ctx, control, options...)", "return result.Extract()"} {
				if !strings.Contains(body, want) {
					t.Fatal("owned native identity contract lost", want, body)
				}
			}
			for _, bad := range []string{`q.Del("status")`, "request.QueryOptions", "IterateSecurityGroups(", "IterateIdentity:", "IdentityAllProjectsQuery:", "IdentityExtraSpecs:", "IdentityMissingListQuery:", "IdentityListQueryDefaults:", "LocalStatus:", "Status:"} {
				if strings.Contains(body, bad) {
					t.Fatal("statusless collection gained coercion or unsupported capability", bad, body)
				}
			}
		})
	}
	if len(identityCollectionSpecs) != 20 || len(identityNativeDeclarations) != 20 {
		t.Fatal("audited identity count changed", len(identityCollectionSpecs), len(identityNativeDeclarations))
	}
}

// Pinned native request, plural-link, and result declarations are parsed, not
// executed. The default time and arbitrary rule decoding remains upstream.
const pinnedQoSIdentitySource = `package fixture
func (opts ListOpts) ToPolicyListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}

func List(c *gophercloud.ServiceClient, opts PolicyListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToPolicyListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return PolicyPage{pagination.LinkedPageBase{PageResult: r}}

	})
}

func Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := c.Get(ctx, getURL(c, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}

func (r commonResult) Extract() (*Policy, error) {
	var s struct {
		Policy *Policy ` + "`" + `json:"policy"` + "`" + `
	}
	err := r.ExtractInto(&s)
	return s.Policy, err
}

func (r PolicyPage) NextPageURL() (string, error) {
	var s struct {
		Links []gophercloud.Link ` + "`" + `json:"policies_links"` + "`" + `
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return gophercloud.ExtractNextURL(s.Links)
}

func (r PolicyPage) IsEmpty() (bool, error) {
	if r.StatusCode == 204 {
		return true, nil
	}

	is, err := ExtractPolicies(r)
	return len(is) == 0, err
}

func ExtractPolicies(r pagination.Page) ([]Policy, error) {
	var s []Policy
	err := ExtractPolicysInto(r, &s)
	return s, err
}

func ExtractPolicysInto(r pagination.Page, v any) error {
	return r.(PolicyPage).ExtractIntoSlicePtr(v, "policies")
}

func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL(resourcePath)
}

func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(resourcePath, id)
}

func listURL(c *gophercloud.ServiceClient) string {
	return rootURL(c)
}

func getURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}
`

const pinnedAddressGroupIdentitySource = `package fixture
func (opts ListOpts) ToAddressGroupListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}

func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := rootURL(c)
	if opts != nil {
		query, err := opts.ToAddressGroupListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return AddressGroupPage{pagination.LinkedPageBase{PageResult: r}}
	})
}

func Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := c.Get(ctx, resourceURL(c, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}

func (r AddressGroupPage) NextPageURL() (string, error) {
	var s struct {
		Links []gophercloud.Link ` + "`" + `json:"address_groups_links"` + "`" + `
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return gophercloud.ExtractNextURL(s.Links)
}

func (r AddressGroupPage) IsEmpty() (bool, error) {
	if r.StatusCode == 204 {
		return true, nil
	}

	is, err := ExtractGroups(r)
	return len(is) == 0, err
}

func ExtractGroups(r pagination.Page) ([]AddressGroup, error) {
	var s struct {
		AddressGroups []AddressGroup ` + "`" + `json:"address_groups"` + "`" + `
	}
	err := (r.(AddressGroupPage)).ExtractInto(&s)
	return s.AddressGroups, err
}

func (r commonResult) Extract() (*AddressGroup, error) {
	var s struct {
		AddressGroup *AddressGroup ` + "`" + `json:"address_group"` + "`" + `
	}
	err := r.ExtractInto(&s)
	return s.AddressGroup, err
}

func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL(rootPath)
}

func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(rootPath, id)
}
`
