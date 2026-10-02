package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Native prefix decoding and default inherited continuation are pinned
// separately. These source declarations are parsed, not executed.
const pinnedSubnetPoolIdentitySource = `package fixture
func (opts ListOpts) ToSubnetPoolListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToSubnetPoolListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return SubnetPoolPage{pagination.LinkedPageBase{PageResult: r}}
	})
}
func Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := c.Get(ctx, getURL(c, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
func (r commonResult) Extract() (*SubnetPool, error) {
	var s struct {
		SubnetPool *SubnetPool ` + "`" + `json:"subnetpool"` + "`" + `
	}
	err := r.ExtractInto(&s)
	return s.SubnetPool, err
}
func (r *SubnetPool) UnmarshalJSON(b []byte) error {
	type tmp SubnetPool

	// Support for older neutron time format
	var s1 struct {
		tmp
		DefaultPrefixLen any ` + "`" + `json:"default_prefixlen"` + "`" + `
		MinPrefixLen     any ` + "`" + `json:"min_prefixlen"` + "`" + `
		MaxPrefixLen     any ` + "`" + `json:"max_prefixlen"` + "`" + `

		CreatedAt gophercloud.JSONRFC3339NoZ ` + "`" + `json:"created_at"` + "`" + `
		UpdatedAt gophercloud.JSONRFC3339NoZ ` + "`" + `json:"updated_at"` + "`" + `
	}

	err := json.Unmarshal(b, &s1)
	if err == nil {
		*r = SubnetPool(s1.tmp)

		r.CreatedAt = time.Time(s1.CreatedAt)
		r.UpdatedAt = time.Time(s1.UpdatedAt)

		switch t := s1.DefaultPrefixLen.(type) {
		case string:
			if r.DefaultPrefixLen, err = strconv.Atoi(t); err != nil {
				return err
			}
		case float64:
			r.DefaultPrefixLen = int(t)
		default:
			return fmt.Errorf("DefaultPrefixLen has unexpected type: %T", t)
		}

		switch t := s1.MinPrefixLen.(type) {
		case string:
			if r.MinPrefixLen, err = strconv.Atoi(t); err != nil {
				return err
			}
		case float64:
			r.MinPrefixLen = int(t)
		default:
			return fmt.Errorf("MinPrefixLen has unexpected type: %T", t)
		}

		switch t := s1.MaxPrefixLen.(type) {
		case string:
			if r.MaxPrefixLen, err = strconv.Atoi(t); err != nil {
				return err
			}
		case float64:
			r.MaxPrefixLen = int(t)
		default:
			return fmt.Errorf("MaxPrefixLen has unexpected type: %T", t)
		}

		return nil
	}

	// Support for newer neutron time format
	var s2 struct {
		tmp
		DefaultPrefixLen any ` + "`" + `json:"default_prefixlen"` + "`" + `
		MinPrefixLen     any ` + "`" + `json:"min_prefixlen"` + "`" + `
		MaxPrefixLen     any ` + "`" + `json:"max_prefixlen"` + "`" + `

		CreatedAt time.Time ` + "`" + `json:"created_at"` + "`" + `
		UpdatedAt time.Time ` + "`" + `json:"updated_at"` + "`" + `
	}

	err = json.Unmarshal(b, &s2)
	if err != nil {
		return err
	}

	*r = SubnetPool(s2.tmp)

	r.CreatedAt = time.Time(s2.CreatedAt)
	r.UpdatedAt = time.Time(s2.UpdatedAt)

	switch t := s2.DefaultPrefixLen.(type) {
	case string:
		if r.DefaultPrefixLen, err = strconv.Atoi(t); err != nil {
			return err
		}
	case float64:
		r.DefaultPrefixLen = int(t)
	default:
		return fmt.Errorf("DefaultPrefixLen has unexpected type: %T", t)
	}

	switch t := s2.MinPrefixLen.(type) {
	case string:
		if r.MinPrefixLen, err = strconv.Atoi(t); err != nil {
			return err
		}
	case float64:
		r.MinPrefixLen = int(t)
	default:
		return fmt.Errorf("MinPrefixLen has unexpected type: %T", t)
	}

	switch t := s2.MaxPrefixLen.(type) {
	case string:
		if r.MaxPrefixLen, err = strconv.Atoi(t); err != nil {
			return err
		}
	case float64:
		r.MaxPrefixLen = int(t)
	default:
		return fmt.Errorf("MaxPrefixLen has unexpected type: %T", t)
	}

	return nil
}
func (r SubnetPoolPage) NextPageURL() (string, error) {
	var s struct {
		Links []gophercloud.Link ` + "`" + `json:"subnetpools_links"` + "`" + `
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return gophercloud.ExtractNextURL(s.Links)
}
func (r SubnetPoolPage) IsEmpty() (bool, error) {
	if r.StatusCode == 204 {
		return true, nil
	}

	subnetpools, err := ExtractSubnetPools(r)
	return len(subnetpools) == 0, err
}
func ExtractSubnetPools(r pagination.Page) ([]SubnetPool, error) {
	var s struct {
		SubnetPools []SubnetPool ` + "`" + `json:"subnetpools"` + "`" + `
	}
	err := (r.(SubnetPoolPage)).ExtractInto(&s)
	return s.SubnetPools, err
}
func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(resourcePath, id)
}
func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL(resourcePath)
}
func listURL(c *gophercloud.ServiceClient) string {
	return rootURL(c)
}
func getURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}`
const pinnedTrunkIdentitySource = `package fixture
func (opts ListOpts) ToTrunkListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToTrunkListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return TrunkPage{pagination.LinkedPageBase{PageResult: r}}
	})
}
func Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := c.Get(ctx, getURL(c, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
func (r commonResult) Extract() (*Trunk, error) {
	var s struct {
		Trunk *Trunk ` + "`" + `json:"trunk"` + "`" + `
	}
	err := r.ExtractInto(&s)
	return s.Trunk, err
}
func (page TrunkPage) IsEmpty() (bool, error) {
	if page.StatusCode == 204 {
		return true, nil
	}

	trunks, err := ExtractTrunks(page)
	return len(trunks) == 0, err
}
func ExtractTrunks(page pagination.Page) ([]Trunk, error) {
	var a struct {
		Trunks []Trunk ` + "`" + `json:"trunks"` + "`" + `
	}
	err := (page.(TrunkPage)).ExtractInto(&a)
	return a.Trunks, err
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
}`
const pinnedLinkedIdentitySource = `package fixture
func (current LinkedPageBase) NextPageURL() (string, error) {
	var path []string
	var key string

	if current.LinkPath == nil {
		path = []string{"links", "next"}
	} else {
		path = current.LinkPath
	}

	submap, ok := current.Body.(map[string]any)
	if !ok {
		err := gophercloud.ErrUnexpectedType{}
		err.Expected = "map[string]any"
		err.Actual = fmt.Sprintf("%v", reflect.TypeOf(current.Body))
		return "", err
	}

	for {
		key, path = path[0], path[1:]

		value, ok := submap[key]
		if !ok {
			return "", nil
		}

		if len(path) > 0 {
			submap, ok = value.(map[string]any)
			if !ok {
				err := gophercloud.ErrUnexpectedType{}
				err.Expected = "map[string]any"
				err.Actual = fmt.Sprintf("%v", reflect.TypeOf(value))
				return "", err
			}
		} else {
			if value == nil {
				// Actual null element.
				return "", nil
			}

			url, ok := value.(string)
			if !ok {
				err := gophercloud.ErrUnexpectedType{}
				err.Expected = "string"
				err.Actual = fmt.Sprintf("%v", reflect.TypeOf(value))
				return "", err
			}

			return url, nil
		}
	}
}`

func pinnedPoolTrunkIdentityDeclarations(t *testing.T, spec identityCollectionSpec) map[string]*ast.FuncDecl {
	t.Helper()
	source := pinnedSubnetPoolIdentitySource
	if spec.model == "Trunk" {
		source = pinnedTrunkIdentitySource
	}
	file, err := parser.ParseFile(token.NewFileSet(), "pool-trunk-native.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			result[identityDeclarationKey(fn)] = fn
		}
	}
	if spec.model == "Trunk" {
		file, err = parser.ParseFile(token.NewFileSet(), "linked-native.go", pinnedLinkedIdentitySource, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				result["pagination."+identityDeclarationKey(fn)] = fn
			}
		}
	}
	return result
}

func poolTrunkIdentitySpecs() []identityCollectionSpec {
	var result []identityCollectionSpec
	for _, spec := range identityCollectionSpecs {
		if spec.path == "network/v2/extensions/subnetpools" || spec.path == "network/v2/extensions/trunks" {
			result = append(result, spec)
		}
	}
	return result
}

func TestIdentityPoolTrunkNativeSchemaAndPrefixDecoderDriftFailGeneration(t *testing.T) {
	if len(poolTrunkIdentitySpecs()) != 2 {
		t.Fatal("missing audited pool/trunk bindings")
	}
	for _, spec := range poolTrunkIdentitySpecs() {
		t.Run(spec.model, func(t *testing.T) {
			base := identityQueryFixtureSource(spec)
			pkg, plan := identityQueryFixture(t, spec, base)
			if !identityPoolTrunkSchema(pkg, plan) {
				t.Fatal("native schema rejected")
			}
			decls := pinnedPoolTrunkIdentityDeclarations(t, spec)
			if err := validateIdentityCollectionContracts(pkg, decls, plan, nil, identityQuerySourceConstants(t, base)); err != nil {
				t.Fatal(err)
			}
			page, extractor := "SubnetPoolPage", "ExtractSubnetPools"
			if spec.model == "Trunk" {
				page, extractor = "TrunkPage", "ExtractTrunks"
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
				"wrong list extractor":     func(s string) string { return strings.Replace(s, "([]"+spec.model+",error)", "([]string,error)", 1) },
				"missing named extractor":  func(s string) string { return strings.Replace(s, "func "+extractor+"(", "func ExtractOther(", 1) },
				"unexpected concrete list": func(s string) string { return strings.Replace(s, "opts ListOptsBuilder)", "opts ListOpts)", 1) },
			}
			if spec.model == "SubnetPool" {
				for _, field := range []string{"DefaultPrefixLen", "MinPrefixLen", "MaxPrefixLen"} {
					field := field
					mutations[field+" pointer"] = func(s string) string { return strings.Replace(s, field+" int `", field+" *int `", 1) }
					mutations[field+" json input tag"] = func(s string) string {
						return strings.Replace(s, field+" int `json:\"-\"`", field+" int `json:\"changed\"`", 1)
					}
					mutations[field+" missing"] = func(s string) string { return strings.Replace(s, ";"+field+" int `json:\"-\"`", "", 1) }
				}
				mutations["wrong continuation argument"] = func(s string) string { return strings.Replace(s, "NextPageURL()", "NextPageURL(bool)", 1) }
			} else {
				mutations["new own continuation override"] = func(s string) string { return s + "\nfunc(TrunkPage)NextPageURL()(string,error){return \"\",nil}\n" }
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

func TestIdentityTrunkInheritedContinuationDependencyIsNarrowAndPinned(t *testing.T) {
	var spec identityCollectionSpec
	for _, candidate := range poolTrunkIdentitySpecs() {
		if candidate.model == "Trunk" {
			spec = candidate
		}
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "linked.go")
	_, _, _, nativeBody := trunkActualNative(t)
	// The raw Body lane additionally uses the pinned native fetch/decoding and
	// handler-order declarations. Identity's continuation guard remains exact.
	pageSource := "package pagination\n" + securityGroupFunctionSource(t, nativeBody, "pagination.")
	if err := os.WriteFile(path, []byte(pageSource), 0600); err != nil {
		t.Fatal(err)
	}
	sdkGenerator := generator{meta: map[string]metadata{upstreamModule + "/pagination": {Dir: dir, GoFiles: []string{"linked.go"}}}}
	dependency, err := sdkGenerator.identityPaginationDeclarations(upstreamModule + "/openstack/networking/v2/extensions/trunks")
	if err != nil || len(dependency) != 8 {
		t.Fatal(dependency, err)
	}
	key := "pagination.LinkedPageBase.NextPageURL"
	hash, err := requestDeclarationHash(dependency[key])
	if err != nil || hash != identityNativeDeclarations[spec.path][key] {
		t.Fatal(hash, err)
	}
	source := identityQueryFixtureSource(spec)
	pkg, plan := identityQueryFixture(t, spec, source)
	validate := func(decls map[string]*ast.FuncDecl) error {
		return validateIdentityCollectionContracts(pkg, decls, plan, nil, identityQuerySourceConstants(t, source))
	}
	decls := pinnedPoolTrunkIdentityDeclarations(t, spec)
	decls[key] = dependency[key]
	if err := validate(decls); err != nil {
		t.Fatal(err)
	}
	missing := pinnedPoolTrunkIdentityDeclarations(t, spec)
	delete(missing, key)
	if err := validate(missing); err == nil || !strings.Contains(err.Error(), "declaration "+key+" changed") {
		t.Fatal("missing dependency ignored", err)
	}
	changed := pinnedPoolTrunkIdentityDeclarations(t, spec)
	changed[key].Body.List = append(changed[key].Body.List, &ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("differentDefaultLinkPath")}})
	if err := validate(changed); err == nil || !strings.Contains(err.Error(), "declaration "+key+" changed") {
		t.Fatal("dependency wire policy drift ignored", err)
	}
	override := pinnedPoolTrunkIdentityDeclarations(t, spec)
	override["TrunkPage.NextPageURL"] = override[key]
	if err := validate(override); err == nil || !strings.Contains(err.Error(), "continuation override changed") {
		t.Fatal("new own page override ignored", err)
	}
	// Protect LinkPath nil in the pinned native List constructor itself.
	changed = pinnedPoolTrunkIdentityDeclarations(t, spec)
	replacement, err := parser.ParseExpr(`[]string{"trunks_links","next"}`)
	if err != nil {
		t.Fatal(err)
	}
	mutated := false
	ast.Inspect(changed["List"], func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		if selector, ok := literal.Type.(*ast.SelectorExpr); ok && selector.Sel.Name == "LinkedPageBase" {
			literal.Elts = append(literal.Elts, &ast.KeyValueExpr{Key: ast.NewIdent("LinkPath"), Value: replacement})
			mutated = true
		}
		return true
	})
	if !mutated {
		t.Fatal("missing native LinkedPageBase construction")
	}
	if err := validate(changed); err == nil || !strings.Contains(err.Error(), "declaration List changed") {
		t.Fatal("new native LinkPath policy ignored", err)
	}
	empty := generator{meta: map[string]metadata{}}
	for _, path := range []string{upstreamModule + "/openstack/compute/v2/servers"} {
		result, err := empty.identityPaginationDeclarations(path)
		if err != nil || len(result) != 0 {
			t.Fatal("dependency guard leaked to unrelated native pagers", path, result, err)
		}
	}
	if _, err := empty.identityPaginationDeclarations(upstreamModule + "/openstack/networking/v2/extensions/trunks"); err == nil {
		t.Fatal("missing metadata accepted")
	}
	invalid := generator{meta: map[string]metadata{upstreamModule + "/pagination": {Dir: dir, GoFiles: []string{"missing.go"}}}}
	if _, err := invalid.identityPaginationDeclarations(upstreamModule + "/openstack/networking/v2/extensions/trunks"); err == nil {
		t.Fatal("missing source accepted")
	}
	if err := os.WriteFile(path, []byte("package fixture\nfunc unrelated(){}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	dependency, err = sdkGenerator.identityPaginationDeclarations(upstreamModule + "/openstack/networking/v2/extensions/trunks")
	if err != nil || len(dependency) != 0 {
		t.Fatal(dependency, err)
	}
	// Missing declaration reaches the normal contract guard; it is not replaced
	// with a guessed implementation or silently removed identity capability.
	decls = pinnedPoolTrunkIdentityDeclarations(t, spec)
	delete(decls, key)
	for name, decl := range dependency {
		decls[name] = decl
	}
	if err := validate(decls); err == nil {
		t.Fatal("absent dependency implementation accepted")
	}
}

func TestIdentityPoolTrunkEmissionKeepsWholeQueriesAndExistingNativeCapabilities(t *testing.T) {
	for _, spec := range poolTrunkIdentitySpecs() {
		t.Run(spec.model, func(t *testing.T) {
			source := identityQueryFixtureSource(spec)
			if spec.model == "Trunk" {
				source = strings.Replace(source, "type Trunk struct{ID string;Name string}", "type Trunk struct{ID string;Name string;Status string}", 1)
				source = strings.Replace(source, "type ListOpts struct{Name string `q:\"name\"`}", "type ListOpts struct{Name string `q:\"name\"`;Status string `q:\"status\"`}", 1)
			}
			pkg, plan := identityQueryFixture(t, spec, source)
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
					t.Fatal("native owned identity contract lost", want, body)
				}
			}
			for _, bad := range []string{`q.Del("status")`, "request.QueryOptions", "IterateSecurityGroups(", "IterateIdentity:", "IdentityAllProjectsQuery:", "IdentityExtraSpecs:", "IdentityMissingListQuery:", "IdentityListQueryDefaults:", "LocalStatus:"} {
				if strings.Contains(body, bad) {
					t.Fatal("pool/trunk gained coercion or unproved capability", bad, body)
				}
			}
			if spec.model == "Trunk" && !strings.Contains(body, "v.Status") {
				t.Fatal("native trunk status was removed", body)
			}
		})
	}
	if len(identityCollectionSpecs) != 20 {
		t.Fatal("audit opt-in count changed", len(identityCollectionSpecs))
	}
}
