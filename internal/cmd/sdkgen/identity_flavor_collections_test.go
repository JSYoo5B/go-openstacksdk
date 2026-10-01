package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// These declarations retain the pinned native detail route, Swap decoder,
// envelope extraction, and separate extra-specs GET. They are never executed.
const pinnedFlavorIdentitySource = `package fixture
func ListDetail(client *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(client)
	if opts != nil {
		query, err := opts.ToFlavorListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(client, url, func(r pagination.PageResult) pagination.Page {
		return FlavorPage{pagination.LinkedPageBase{PageResult: r}}
	})
}
func Get(ctx context.Context, client *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := client.Get(ctx, getURL(client, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
func ListExtraSpecs(ctx context.Context, client *gophercloud.ServiceClient, flavorID string) (r ListExtraSpecsResult) {
	resp, err := client.Get(ctx, extraSpecsListURL(client, flavorID), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
func getURL(client *gophercloud.ServiceClient, id string) string {
	return client.ServiceURL("flavors", id)
}
func listURL(client *gophercloud.ServiceClient) string {
	return client.ServiceURL("flavors", "detail")
}
func extraSpecsListURL(client *gophercloud.ServiceClient, id string) string {
	return client.ServiceURL("flavors", id, "os-extra_specs")
}
func (r commonResult) Extract() (*Flavor, error) {
	var s struct {
		Flavor *Flavor ` + "`" + `json:"flavor"` + "`" + `
	}
	err := r.ExtractInto(&s)
	return s.Flavor, err
}
func (r *Flavor) UnmarshalJSON(b []byte) error {
	type tmp Flavor
	var s struct {
		tmp
		Swap any ` + "`" + `json:"swap"` + "`" + `
	}
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	*r = Flavor(s.tmp)

	switch t := s.Swap.(type) {
	case float64:
		r.Swap = int(t)
	case string:
		switch t {
		case "":
			r.Swap = 0
		default:
			swap, err := strconv.ParseFloat(t, 64)
			if err != nil {
				return err
			}
			r.Swap = int(swap)
		}
	}

	return nil
}
func (page FlavorPage) IsEmpty() (bool, error) {
	if page.StatusCode == 204 {
		return true, nil
	}

	flavors, err := ExtractFlavors(page)
	return len(flavors) == 0, err
}
func (page FlavorPage) NextPageURL() (string, error) {
	var s struct {
		Links []gophercloud.Link ` + "`" + `json:"flavors_links"` + "`" + `
	}
	err := page.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return gophercloud.ExtractNextURL(s.Links)
}
func ExtractFlavors(r pagination.Page) ([]Flavor, error) {
	var s struct {
		Flavors []Flavor ` + "`" + `json:"flavors"` + "`" + `
	}
	err := (r.(FlavorPage)).ExtractInto(&s)
	return s.Flavors, err
}
func (r extraSpecsResult) Extract() (map[string]string, error) {
	var s struct {
		ExtraSpecs map[string]string ` + "`" + `json:"extra_specs"` + "`" + `
	}
	err := r.ExtractInto(&s)
	return s.ExtraSpecs, err
}`

func pinnedFlavorIdentityDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "native-flavor.go", pinnedFlavorIdentitySource, 0)
	if err != nil {
		t.Fatal(err)
	}
	declarations := map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok {
			declarations[identityDeclarationKey(fn)] = fn
		}
	}
	return declarations
}

func flavorIdentitySpec(t *testing.T) identityCollectionSpec {
	t.Helper()
	for _, spec := range identityCollectionSpecs {
		if spec.path == "compute/v2/flavors" {
			return spec
		}
	}
	t.Fatal("Flavor identity contract is missing")
	return identityCollectionSpec{}
}

func TestIdentityFlavorMetadataLimitsDefaultsAndEnrichmentToFlavor(t *testing.T) {
	flavor := flavorIdentitySpec(t)
	if !identityFlavorMetadataValid(flavor) {
		t.Fatal("audited Flavor metadata rejected")
	}
	mutations := map[string]func(*identityCollectionSpec){
		"model":              func(s *identityCollectionSpec) { s.model = "Server" },
		"GET method":         func(s *identityCollectionSpec) { s.getter = "Fetch" },
		"LIST method":        func(s *identityCollectionSpec) { s.lister = "List" },
		"parent":             func(s *identityCollectionSpec) { s.parents = 1 },
		"route":              func(s *identityCollectionSpec) { s.getSegments = []string{"servers", "$id"} },
		"literal ID":         func(s *identityCollectionSpec) { s.getSegments = []string{"flavors", "fixed"} },
		"extra route":        func(s *identityCollectionSpec) { s.getSegments = []string{"flavors", "$id", "os-extra_specs"} },
		"code":               func(s *identityCollectionSpec) { s.getCodes = []int{203} },
		"extra code":         func(s *identityCollectionSpec) { s.getCodes = []int{200, 203} },
		"name hint":          func(s *identityCollectionSpec) { s.noNameQuery = false },
		"default key":        func(s *identityCollectionSpec) { s.listDefaultKey = "isPublic" },
		"default value":      func(s *identityCollectionSpec) { s.listDefaultValue = "true" },
		"missing default":    func(s *identityCollectionSpec) { s.listDefaultKey = ""; s.listDefaultValue = "" },
		"missing enrichment": func(s *identityCollectionSpec) { s.extraSpecs = false },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			altered := flavor
			mutate(&altered)
			if identityFlavorMetadataValid(altered) {
				t.Fatal("unreviewed Flavor metadata accepted", altered)
			}
		})
	}
	for _, spec := range identityCollectionSpecs {
		if spec.path == flavor.path {
			continue
		}
		if !identityFlavorMetadataValid(spec) {
			t.Fatal("existing binding rejected", spec)
		}
		for _, mutate := range []func(*identityCollectionSpec){
			func(s *identityCollectionSpec) { s.noNameQuery = true },
			func(s *identityCollectionSpec) { s.listDefaultKey = "is_public"; s.listDefaultValue = "None" },
			func(s *identityCollectionSpec) { s.extraSpecs = true },
		} {
			altered := spec
			mutate(&altered)
			if identityFlavorMetadataValid(altered) {
				t.Fatal("Flavor capability leaked", altered)
			}
		}
	}
}

func TestIdentityFlavorSchemaAndExtractionDriftFailGeneration(t *testing.T) {
	spec := flavorIdentitySpec(t)
	base := identityQueryFixtureSource(spec)
	pkg, plan := identityQueryFixture(t, spec, base)
	if !identityFlavorSchema(pkg, plan) {
		t.Fatal("native Flavor schema rejected")
	}
	if err := validateIdentityCollectionContracts(pkg, pinnedFlavorIdentityDeclarations(t), plan, nil, nil); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(string) string{
		"name query": func(s string) string {
			return strings.Replace(s, "type ListOpts struct{AccessType", "type ListOpts struct{Name string `q:\"name\"`;AccessType", 1)
		},
		"access pointer": func(s string) string {
			return strings.Replace(s, "AccessType AccessType `q:", "AccessType *AccessType `q:", 1)
		},
		"access key": func(s string) string { return strings.Replace(s, "q:\"is_public\"", "q:\"isPublic\"", 1) },
		"access field": func(s string) string {
			return strings.Replace(s, "AccessType AccessType `q:", "Public AccessType `q:", 1)
		},
		"extra map values": func(s string) string {
			return strings.Replace(s, "ExtraSpecs map[string]string", "ExtraSpecs map[string]bool", 1)
		},
		"extra map keys": func(s string) string {
			return strings.Replace(s, "ExtraSpecs map[string]string", "ExtraSpecs map[int]string", 1)
		},
		"extra field": func(s string) string {
			return strings.Replace(s, "ExtraSpecs map[string]string", "Specs map[string]string", 1)
		},
		"extra key": func(s string) string { return strings.Replace(s, "json:\"extra_specs\"", "json:\"specs\"", 1) },
		"page state": func(s string) string {
			return strings.Replace(s, "FlavorPage struct{pagination.LinkedPageBase}", "FlavorPage struct{state bool;pagination.LinkedPageBase}", 1)
		},
		"nonembedded page": func(s string) string {
			return strings.Replace(s, "FlavorPage struct{pagination.LinkedPageBase}", "FlavorPage struct{page pagination.LinkedPageBase}", 1)
		},
		"empty result": func(s string) string {
			return strings.Replace(s, "IsEmpty()(bool,error){return false,nil}", "IsEmpty()(int,error){return 0,nil}", 1)
		},
		"continuation argument":  func(s string) string { return strings.Replace(s, "NextPageURL()", "NextPageURL(bool)", 1) },
		"list extractor":         func(s string) string { return strings.Replace(s, "([]Flavor,error)", "([]string,error)", 1) },
		"missing list extractor": func(s string) string { return strings.Replace(s, "func ExtractFlavors(", "func ExtractOther(", 1) },
		"extra GET argument": func(s string) string {
			return strings.Replace(s, "id string)ListExtraSpecsResult", "id string,opts ListOptsBuilder)ListExtraSpecsResult", 1)
		},
		"extra GET result": func(s string) string { return strings.ReplaceAll(s, "ListExtraSpecsResult", "OtherSpecsResult") },
		"extra extractor result": func(s string) string {
			return strings.Replace(s, "Extract()(map[string]string,error)", "Extract()(map[string]bool,error)", 1)
		},
		"extra extractor name": func(s string) string {
			return strings.Replace(s, "func(ListExtraSpecsResult)Extract()", "func(ListExtraSpecsResult)ExtractOther()", 1)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			source := mutate(base)
			if source == base {
				t.Fatal("mutation did not apply")
			}
			badPkg, badPlan := identityQueryFixture(t, spec, source)
			if err := validateIdentityCollectionContracts(badPkg, pinnedFlavorIdentityDeclarations(t), badPlan, nil, nil); err == nil || !strings.Contains(err.Error(), "schema changed") {
				t.Fatal("schema drift ignored", err)
			}
		})
	}
}

func TestIdentityFlavorEmitsNativeDefaultsEnrichmentAndWholeQuery(t *testing.T) {
	defaults, enrichment, missing, modes := 0, 0, 0, 0
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
			emitted, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			body := string(emitted)
			flavor := spec.path == "compute/v2/flavors"
			if flavor {
				for _, want := range []string{`IdentityListQueryDefaults: url.Values{"is_public": {"None"}}`, `IdentityExtraSpecs: func(ctx context.Context, value *Flavor) (*Flavor, error)`, `return nativefind.FlavorExtraSpecs(ctx, a.RawClient(), value)`, `[]string{"flavors", id}, q, []int{200}`, "return result.Extract()", "config.Query[key] = append([]string(nil), values...)", "a.listDetailWithControl(ctx, control, options...)"} {
					if !strings.Contains(body, want) {
						t.Fatal("native Flavor behavior lost", want, body)
					}
				}
				for _, bad := range []string{"NameQuery:", `q.Del("status")`, `q.Set("",`, `WithListDetailQuery(key, value)`, "IdentityAllProjectsQuery:", "IterateIdentity:", "IdentityMissingListQuery:"} {
					if strings.Contains(body, bad) {
						t.Fatal("Flavor gained unrelated policy or query rewrite", bad, body)
					}
				}
			} else if strings.Contains(body, "IdentityListQueryDefaults:") || strings.Contains(body, "IdentityExtraSpecs:") {
				t.Fatal("Flavor-only policy leaked", body)
			}
			if strings.Contains(body, "IdentityListQueryDefaults:") {
				defaults++
			}
			if strings.Contains(body, "IdentityExtraSpecs:") {
				enrichment++
			}
			if strings.Contains(body, "IdentityMissingListQuery:") {
				missing++
			}
			if strings.Contains(body, "IdentityAllProjectsQuery:") {
				modes++
			}
			enabled := identityFlavorEnabled(pkg, plan, spec.parents)
			if enabled != flavor {
				t.Fatal("capability mismatch", spec, enabled)
			}
			data, err := json.Marshal(collectionRecord{IdentityListDefaults: enabled, IdentityExtraSpecs: enabled})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), `"identity_list_defaults":true`) != flavor || strings.Contains(string(data), `"identity_extra_specs":true`) != flavor {
				t.Fatal("inventory mismatch", string(data))
			}
		})
	}
	if len(identityCollectionSpecs) != 16 || defaults != 1 || enrichment != 1 || missing != 1 || modes != 2 {
		t.Fatal("explicit capability totals changed", len(identityCollectionSpecs), defaults, enrichment, missing, modes)
	}
	data, err := json.Marshal(collectionRecord{})
	if err != nil || strings.Contains(string(data), "identity_list_defaults") || strings.Contains(string(data), "identity_extra_specs") {
		t.Fatal("default inventory overclaims enrichment", string(data), err)
	}
}
