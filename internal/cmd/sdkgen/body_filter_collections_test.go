package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func emittedBodyFilterAdapter(t *testing.T, spec identityCollectionSpec) ([]byte, map[string]ast.Expr) {
	t.Helper()
	pkg, plan := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
	e := emitter{pkg: pkg, imports: map[string]string{}}
	e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", spec.model)
	parents := []string(nil)
	if spec.parents != 0 {
		parents = []string{"fixedParent"}
	}
	emitCollectionAdapter(&e, plan, "a", parents)
	e.printf("}\n")
	if spec.parents == 0 {
		emitBodyFilterList(&e, plan)
	}
	source, err := e.source()
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "emitted.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]ast.Expr{}
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		index, ok := literal.Type.(*ast.IndexExpr)
		if !ok {
			return true
		}
		selector, ok := index.X.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Adapter" {
			return true
		}
		for _, entry := range literal.Elts {
			pair, ok := entry.(*ast.KeyValueExpr)
			if !ok {
				t.Fatal("non-keyed adapter field")
			}
			fields[pair.Key.(*ast.Ident).Name] = pair.Value
		}
		return true
	})
	return source, fields
}

func pinnedNetworkBodyIdentityDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "native_network.go", pinnedNetworkBodyIdentitySource, 0)
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

func pinnedSubnetBodyIdentityDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "native_subnet.go", pinnedSubnetBodyIdentitySource, 0)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok {
			key := identityDeclarationKey(fn)
			if key == "PageResultFrom" || key == "PageResultFromParsed" || key == "LinkedPageBase.GetBody" {
				key = "pagination." + key
			}
			result[key] = fn
		}
	}
	return result
}

func TestBodyFilterCollectionGateRejectsNativeShapeAndUnauditedFields(t *testing.T) {
	if len(bodyFilterCollectionSpecs) != 6 || len(identityCollectionSpecs) != 20 {
		t.Fatal("body-filter or identity inventory broadened")
	}
	enabled := 0
	for _, identity := range identityCollectionSpecs {
		pkg, plan := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		spec, ok := bodyFilterCollectionContract(pkg, plan, identity.parents)
		wanted := identity.path == "network/v2/extensions/qos/policies" || identity.path == "network/v2/extensions/security/addressgroups" || identity.path == "network/v2/extensions/subnetpools" || identity.path == "network/v2/networks" || identity.path == "network/v2/subnets"
		if ok != wanted {
			t.Fatal("body filter capability leaked or disappeared", identity.path, ok)
		}
		if !ok {
			continue
		}
		enabled++
		if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
			t.Fatal("global descriptor leaked into parent scope")
		}
		if err := validateBodyFilterCollectionContracts(pkg, plan); err != nil {
			t.Fatal(err)
		}
		base := identityQueryFixtureSource(identity)
		field := spec.fields[0]
		mutations := map[string]string{
			"wire field tag":         strings.Replace(base, `json:"`+field.key+`"`, `json:"different_body_key"`, 1),
			"missing selected field": strings.Replace(base, field.member+" ", "OtherField ", 1),
		}
		if identity.model == "Policy" {
			mutations["numeric rule decoder"] = strings.Replace(base, "Rules []map[string]any", "Rules []map[string]float64", 1)
		} else if spec.rawRecord {
			mutations["arbitrary array decoder"] = strings.Replace(base, "AllocationPools []AllocationPool", "AllocationPools []any", 1)
		} else {
			mutations["arbitrary array decoder"] = strings.Replace(base, field.member+" []string", field.member+" []any", 1)
		}
		for name, changed := range mutations {
			t.Run(identity.model+"/"+name, func(t *testing.T) {
				if changed == base {
					t.Fatal("native field mutation did not apply")
				}
				badPkg, badPlan := identityQueryFixture(t, identity, changed)
				if _, ok := bodyFilterCollectionContract(badPkg, badPlan, 0); ok {
					t.Fatal("invalid native selected field enabled")
				}
				if err := validateBodyFilterCollectionContracts(badPkg, badPlan); err == nil || !strings.Contains(err.Error(), "schema changed") {
					t.Fatal("native field drift silently dropped capability", err)
				}
				if (identity.model == "Policy" || identity.model == "AddressGroup") && validateIdentityCollectionContracts(badPkg, pinnedIdentityDeclarations(t, identity), badPlan, nil, identityQuerySourceConstants(t, changed)) == nil {
					t.Fatal("shared native model/schema gate bypassed")
				}
				if !identityCollectionEnabled(badPkg, badPlan, 0) {
					t.Fatal("body field change altered the existing identity option gate")
				}
			})
		}
		for name, mutate := range map[string]func(*bodyFilterCollectionSpec){
			"unreviewed path":  func(value *bodyFilterCollectionSpec) { value.path = "network/v2/ports" },
			"wrong model":      func(value *bodyFilterCollectionSpec) { value.model = "Port" },
			"public alias":     func(value *bodyFilterCollectionSpec) { value.fields[0].key = "local_alias" },
			"different member": func(value *bodyFilterCollectionSpec) { value.fields[0].member = "Name" },
			"additional field": func(value *bodyFilterCollectionSpec) {
				value.fields = append(value.fields, bodyFilterCollectionField{key: "name", member: "Name"})
			},
			"unreviewed alias":  func(value *bodyFilterCollectionSpec) { value.fields[0].aliases = []string{"unexpected_alias"} },
			"canonical alias":   func(value *bodyFilterCollectionSpec) { value.fields[0].aliases = []string{field.key} },
			"duplicate aliases": func(value *bodyFilterCollectionSpec) { value.fields[0].aliases = []string{"subnet_ids", "subnet_ids"} },
		} {
			t.Run(identity.model+"/"+name, func(t *testing.T) {
				copy := spec
				copy.fields = append([]bodyFilterCollectionField(nil), spec.fields...)
				for i := range copy.fields {
					copy.fields[i].aliases = append([]string(nil), spec.fields[i].aliases...)
				}
				mutate(&copy)
				if bodyFilterCollectionMetadataValid(copy) {
					t.Fatal("unreviewed field metadata accepted", copy)
				}
			})
		}
		if identity.model == "Network" {
			copy := spec
			copy.fields = append([]bodyFilterCollectionField(nil), spec.fields...)
			copy.fields[0].aliases = nil
			if bodyFilterCollectionMetadataValid(copy) {
				t.Fatal("Python subnet_ids alias was silently removed")
			}
			for name, changed := range map[string]string{
				"missing native decoder":                 strings.Replace(base, "func(*Network)UnmarshalJSON([]byte)error{return nil}", "", 1),
				"wrong native decoder signature":         strings.Replace(base, "UnmarshalJSON([]byte)", "UnmarshalJSON(string)", 1),
				"inherited rather than own continuation": strings.Replace(base, "func(NetworkPage)NextPageURL()(string,error){return \"\",nil}", "", 1),
				"additional pager state":                 strings.Replace(base, "NetworkPage struct{pagination.LinkedPageBase}", "NetworkPage struct{extra bool;pagination.LinkedPageBase}", 1),
				"wrong resource key shape":               strings.Replace(base, "ResourceKey()string{return \"networks\"}", "ResourceKey()int{return 0}", 1),
				"wrong whole page extractor":             strings.Replace(base, "([]Network,error)", "([]string,error)", 1),
				"typed slice destination":                strings.Replace(base, "ExtractNetworksInto(r pagination.Page,v any)", "ExtractNetworksInto(r pagination.Page,v []Network)", 1),
				"typed member destination":               strings.Replace(base, "func(GetResult)ExtractInto(v any)", "func(GetResult)ExtractInto(v Network)", 1),
			} {
				t.Run(identity.model+"/"+name, func(t *testing.T) {
					if changed == base {
						t.Fatal("native network mutation did not apply")
					}
					badPkg, badPlan := identityQueryFixture(t, identity, changed)
					if err := validateIdentityCollectionContracts(badPkg, pinnedIdentityDeclarations(t, identity), badPlan, nil, identityQuerySourceConstants(t, changed)); err == nil || !strings.Contains(err.Error(), "schema changed") {
						t.Fatal("network native decoder/page extraction drift ignored", err)
					}
				})
			}
		}
		if spec.rawRecord {
			for name, changed := range map[string]string{
				"native prefix field introduced":         strings.Replace(base, "type Subnet struct{", "type Subnet struct{PrefixLength int;", 1),
				"missing timestamp decoder":              strings.Replace(base, "func(*Subnet)UnmarshalJSON([]byte)error{return nil}", "", 1),
				"wrong timestamp decoder signature":      strings.Replace(base, "UnmarshalJSON([]byte)", "UnmarshalJSON(string)", 1),
				"native timestamp field projected":       strings.Replace(base, "CreatedAt time.Time `json:\"-\"`", "CreatedAt string `json:\"created_at\"`", 1),
				"raw row order extractor changed":        strings.Replace(base, "([]Subnet,error)", "([]string,error)", 1),
				"missing own continuation":               strings.Replace(base, "func(SubnetPage)NextPageURL()(string,error){return \"\",nil}", "", 1),
				"own body override":                      base + "\nfunc(SubnetPage)GetBody()any{return nil}\n",
				"nested pool shape":                      strings.Replace(base, "Start string `json:\"start\"`", "Start int `json:\"start\"`", 1),
				"nested route wire tag":                  strings.Replace(base, "NextHop string `json:\"nexthop\"`", "NextHop string `json:\"next_hop\"`", 1),
				"tenant field lost":                      strings.Replace(base, "TenantID string `json:\"tenant_id\"`;", "", 1),
				"numeric revision projected differently": strings.Replace(base, "RevisionNumber int `json:\"revision_number\"`", "RevisionNumber float64 `json:\"revision_number\"`", 1),
				"DNS array erases scalar type":           strings.Replace(base, "DNSNameservers []string", "DNSNameservers []any", 1),
				"service array erases scalar type":       strings.Replace(base, "ServiceTypes []string", "ServiceTypes []any", 1),
			} {
				t.Run(identity.model+"/"+name, func(t *testing.T) {
					if changed == base {
						t.Fatal("Subnet field mutation did not apply")
					}
					badPkg, badPlan := identityQueryFixture(t, identity, changed)
					if err := validateBodyFilterCollectionContracts(badPkg, badPlan); err == nil {
						t.Fatal("raw/native shape drift silently changed the Body policy")
					}
					if err := validateIdentityCollectionContracts(badPkg, pinnedIdentityDeclarations(t, identity), badPlan, nil, identityQuerySourceConstants(t, changed)); err == nil || !strings.Contains(err.Error(), "schema changed") {
						t.Fatal("raw row/native extraction guard bypassed", err)
					}
					if name != "raw row order extractor changed" && !identityCollectionEnabled(badPkg, badPlan, 0) {
						t.Fatal("raw Body field change altered the existing identity options")
					}
				})
			}
		}
	}
	if enabled != 5 {
		t.Fatal("missing audited body selectors", enabled)
	}
}

func TestBodyFilterCollectionSelectorsMarshalOnlyTheSelectedNativeField(t *testing.T) {
	for _, identity := range identityCollectionSpecs {
		t.Run(identity.model, func(t *testing.T) {
			source, fields := emittedBodyFilterAdapter(t, identity)
			pkg, plan := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
			spec, enabled := bodyFilterCollectionContract(pkg, plan, identity.parents)
			if !enabled {
				if fields["BodyFilterFields"] != nil || fields["BodyFilterValue"] != nil || fields["BodyFilterRecordValue"] != nil || fields["IterateBodyControlled"] != nil {
					t.Fatal("unsupported binding acquired body selectors", string(source))
				}
				return
			}
			field := spec.fields[0]
			descriptor, ok := fields["BodyFilterFields"].(*ast.CompositeLit)
			keys, canonicalKeys := []string{}, []string{}
			for _, selected := range spec.fields {
				keys = append(keys, selected.key)
				canonicalKeys = append(canonicalKeys, selected.key)
				for _, alias := range selected.aliases {
					keys = append(keys, alias)
					canonicalKeys = append(canonicalKeys, selected.key)
				}
			}
			if !ok || len(descriptor.Elts) != len(keys) {
				t.Fatal("canonical and reviewed alias descriptor missing", string(source))
			}
			for i, entry := range descriptor.Elts {
				pair := entry.(*ast.KeyValueExpr)
				key, err := strconv.Unquote(pair.Key.(*ast.BasicLit).Value)
				if err != nil {
					t.Fatal(err)
				}
				value, err := strconv.Unquote(pair.Value.(*ast.BasicLit).Value)
				if err != nil || key != keys[i] || value != canonicalKeys[i] {
					t.Fatal("body descriptor exposed an unreviewed alias or model name", key, value, err)
				}
			}
			if spec.rawRecord {
				if fields["BodyFilterValue"] != nil {
					t.Fatal("Subnet raw fields projected from a native model", string(source))
				}
				callback, ok := fields["BodyFilterRecordValue"].(*ast.FuncLit)
				if !ok || nodeText(callback.Type.Params.List[0].Type) != "*resource.BodyRecord[Subnet]" || nodeText(callback.Type.Results.List[0].Type) != "json.RawMessage" {
					t.Fatal("raw Body callback type lost", fields["BodyFilterRecordValue"])
				}
				iterator, ok := fields["IterateBodyControlled"].(*ast.FuncLit)
				if !ok || len(iterator.Type.Params.List) != 3 || nodeText(iterator.Type.Params.List[2].Type) != "resource.ListControl" || nodeText(iterator.Type.Results.List[0].Type) != "iter.Seq2[*resource.BodyRecord[Subnet], error]" {
					t.Fatal("raw controlled iterator type lost", fields["IterateBodyControlled"])
				}
				body := string(source)
				for _, required := range []string{
					"if record == nil", "resource.ErrInvalidOption", `case "revision_number":`, `case "allocation_pools", "created_at", "dns_nameservers", "host_routes", "prefixlen", "service_types", "tenant_id", "updated_at":`, "resource.BodyRecordField(record.Fields, key, resource.BodyFieldInteger)", "resource.BodyRecordField(record.Fields, key, resource.BodyFieldJSON)",
					"config.Query[key] = append([]string(nil), values...)", "return a.listBodyWithControl(ctx, control, options...)",
				} {
					if !strings.Contains(body, required) {
						t.Fatal("raw body/query ownership policy missing", required, body)
					}
				}
				for _, forbidden := range []string{"json.Marshal(", "record.Value", "WithListQuery(", "IdentityExtraSpecs:", "IdentityMissingListQuery:", `q.Del("status")`} {
					if strings.Contains(body, forbidden) {
						t.Fatal("raw Body lane changed field/query/mode policy", forbidden, body)
					}
				}
				helper := controlledEmittedMethod(t, source, "listBodyWithControl")
				if len(helper.Type.Params.List) != 3 || nodeText(helper.Type.Params.List[2].Type) != "...ListOption" {
					t.Fatal("native List option signature was not reused", nodeText(helper.Type))
				}
				requireControlledCalls(t, helper, "request.Apply(opts, options...)", "request.ValidateCapabilities(cfg, false, true, false)", `request.Wrap("List", "subnets", err)`, "upstream.List(a.client, _opts)", "upstream.ExtractSubnets(page)", `resource.BodyStreamWithControl(ctx, upstream.List(a.client, _opts)`, `"subnets", control)`)
				return
			}
			if fields["BodyFilterRecordValue"] != nil || fields["IterateBodyControlled"] != nil {
				t.Fatal("raw Subnet lane changed an existing typed binding", string(source))
			}
			callback, ok := fields["BodyFilterValue"].(*ast.FuncLit)
			if !ok || len(callback.Type.Params.List) != 2 || len(callback.Type.Results.List) != 2 {
				t.Fatal("typed body selector callback missing")
			}
			if nodeText(callback.Type.Params.List[0].Type) != "*"+identity.model || nodeText(callback.Type.Params.List[1].Type) != "string" || nodeText(callback.Type.Results.List[0].Type) != "json.RawMessage" {
				t.Fatal("selector lost native model or raw JSON result", nodeText(callback))
			}
			var selection *ast.SwitchStmt
			marshals := 0
			ast.Inspect(callback.Body, func(node ast.Node) bool {
				if branch, ok := node.(*ast.SwitchStmt); ok {
					selection = branch
				}
				call, ok := node.(*ast.CallExpr)
				if ok && nodeText(call.Fun) == "json.Marshal" {
					marshals++
					if len(call.Args) != 1 || nodeText(call.Args[0]) != "v."+field.member {
						t.Fatal("selector projected a whole model or wrong body field", nodeText(call))
					}
				}
				return true
			})
			if marshals != 1 || selection == nil || nodeText(selection.Tag) != "key" || len(selection.Body.List) != 2 {
				t.Fatal("selector did not preserve a single direct marshal per canonical key", nodeText(callback))
			}
			selected := selection.Body.List[0].(*ast.CaseClause)
			if len(selected.List) != 1 || nodeText(selected.List[0]) != strconv.Quote(field.key) || len(selected.Body) != 1 {
				t.Fatal("selected field case changed", nodeText(callback))
			}
			// Direct marshal preserves native nil slices as null and nonnil
			// empty slices as []; no model omitempty/projection can erase them.
			for _, required := range []string{"if v == nil", "resource.ErrInvalidOption", "config.Query[key] = append([]string(nil), values...)", "return result.Extract()"} {
				if !strings.Contains(string(source), required) {
					t.Fatal("nil/error or unchanged wire query behavior lost", required)
				}
			}
			for _, forbidden := range []string{"reflect.", `q.Del("` + field.key + `")`, "json.Marshal(v)", "WithBodyFilter", "IdentityExtraSpecs:"} {
				if strings.Contains(string(source), forbidden) {
					t.Fatal("body selector changed request policy or used model reflection", forbidden)
				}
			}
		})
	}
}

func TestBodyFilterCollectionStillRequiresPinnedNativeContracts(t *testing.T) {
	for _, identity := range identityCollectionSpecs {
		pkg, plan := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		if _, ok := bodyFilterCollectionContract(pkg, plan, identity.parents); !ok {
			continue
		}
		t.Run(identity.model, func(t *testing.T) {
			source := identityQueryFixtureSource(identity)
			pkg, plan := identityQueryFixture(t, identity, source)
			if err := validateBodyFilterCollectionContracts(pkg, plan); err != nil {
				t.Fatal(err)
			}
			decls := pinnedIdentityDeclarations(t, identity)
			decls["commonResult.Extract"].Body.List = append(decls["commonResult.Extract"].Body.List, &ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("differentBodyDecode")}})
			if err := validateIdentityCollectionContracts(pkg, decls, plan, nil, identityQuerySourceConstants(t, source)); err == nil || !strings.Contains(err.Error(), "declaration commonResult.Extract changed") {
				t.Fatal("body-filter binding bypassed pinned native extraction gate", err)
			}
			if identity.model == "Network" || identity.model == "Subnet" {
				for name := range identityNativeDeclarations[identity.path] {
					for _, absent := range []bool{false, true} {
						decls := pinnedIdentityDeclarations(t, identity)
						if absent {
							delete(decls, name)
						} else {
							decls[name].Body.List = append(decls[name].Body.List, &ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("differentNativeWireOrDecode")}})
						}
						if err := validateIdentityCollectionContracts(pkg, decls, plan, nil, identityQuerySourceConstants(t, source)); err == nil || !strings.Contains(err.Error(), "declaration "+name+" changed") {
							t.Fatal("network native list/decoder hash gate lost", name, absent, err)
						}
					}
				}
			}
			if identity.model == "Subnet" {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "page.go"), []byte(pinnedSubnetBodyIdentitySource), 0600); err != nil {
					t.Fatal(err)
				}
				sdkGenerator := generator{meta: map[string]metadata{upstreamModule + "/pagination": {Dir: dir, GoFiles: []string{"page.go"}}}}
				dependency, err := sdkGenerator.identityPaginationDeclarations(pkg.Path())
				if err != nil || len(dependency) != 3 {
					t.Fatal("raw-row page dependency collection changed", dependency, err)
				}
				for _, key := range []string{"pagination.PageResultFrom", "pagination.PageResultFromParsed", "pagination.LinkedPageBase.GetBody"} {
					if hash, err := requestDeclarationHash(dependency[key]); err != nil || hash != identityNativeDeclarations[identity.path][key] {
						t.Fatal("JSON-number or original body contract changed", key, hash, err)
					}
				}
				if _, err := (&generator{meta: map[string]metadata{}}).identityPaginationDeclarations(pkg.Path()); err == nil {
					t.Fatal("raw page guard accepted missing dependency metadata")
				}
				if result, err := sdkGenerator.identityPaginationDeclarations(upstreamModule + "/openstack/networking/v2/networks"); err != nil || len(result) != 0 {
					t.Fatal("raw page dependency loading leaked into typed lane", result, err)
				}
				// A new JSON parser without UseNumber or a copied/mapped GetBody
				// must fail the native contract, rather than silently erase values.
				changed := strings.Replace(pinnedSubnetBodyIdentitySource, "dec.UseNumber()", "", 1)
				if err := os.WriteFile(filepath.Join(dir, "page.go"), []byte(changed), 0600); err != nil {
					t.Fatal(err)
				}
				dependency, err = sdkGenerator.identityPaginationDeclarations(pkg.Path())
				if err != nil {
					t.Fatal(err)
				}
				decls := pinnedIdentityDeclarations(t, identity)
				for key, declaration := range dependency {
					decls[key] = declaration
				}
				if err := validateIdentityCollectionContracts(pkg, decls, plan, nil, nil); err == nil || !strings.Contains(err.Error(), "pagination.PageResultFrom changed") {
					t.Fatal("raw-number dependency drift ignored", err)
				}
			}
		})
	}
}

func TestBodyFilterCollectionInventoryHasOnlyOwnedCanonicalFields(t *testing.T) {
	enabled := 0
	for _, identity := range identityCollectionSpecs {
		pkg, plan := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		fields := bodyFilterCollectionFields(pkg, plan, identity.parents)
		record := collectionRecord{BodyFilterFields: fields}
		encoded, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if len(fields) == 0 {
			if strings.Contains(string(encoded), "body_filter_fields") {
				t.Fatal("unsupported body capability recorded", string(encoded))
			}
			continue
		}
		enabled++
		spec, _ := bodyFilterCollectionContract(pkg, plan, identity.parents)
		if len(fields) != len(spec.fields) || !strings.Contains(string(encoded), `"body_filter_fields":["`+fields[0]+`"`) {
			t.Fatal("canonical body field descriptor missing", string(encoded))
		}
		if strings.Contains(string(encoded), "subnet_ids") || strings.Contains(string(encoded), "prefix_length") {
			t.Fatal("inventory exposed an alias instead of the canonical field", string(encoded))
		}
		fields[0] = "mutated"
		fresh := bodyFilterCollectionFields(pkg, plan, identity.parents)
		if len(fresh) != len(spec.fields) || fresh[0] == "mutated" {
			t.Fatal("inventory caller mutated generator-owned schema")
		}
	}
	if enabled != 5 {
		t.Fatal("audited inventory body capability count changed", enabled)
	}
}

// Native network decoder and both extraction envelopes are pinned source.
const pinnedNetworkBodyIdentitySource = `package fixture
func (opts ListOpts) ToNetworkListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}

func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToNetworkListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return NetworkPage{pagination.LinkedPageBase{PageResult: r}}
	})
}

func Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := c.Get(ctx, getURL(c, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}

func (r commonResult) Extract() (*Network, error) {
	var s Network
	err := r.ExtractInto(&s)
	return &s, err
}

func (r commonResult) ExtractInto(v any) error {
	return r.ExtractIntoStructPtr(v, "network")
}

func (r *Network) UnmarshalJSON(b []byte) error {
	type tmp Network

	// Support for older neutron time format
	var s1 struct {
		tmp
		CreatedAt gophercloud.JSONRFC3339NoZ ` + "`" + `json:"created_at"` + "`" + `
		UpdatedAt gophercloud.JSONRFC3339NoZ ` + "`" + `json:"updated_at"` + "`" + `
	}

	err := json.Unmarshal(b, &s1)
	if err == nil {
		*r = Network(s1.tmp)
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

	*r = Network(s2.tmp)
	r.CreatedAt = time.Time(s2.CreatedAt)
	r.UpdatedAt = time.Time(s2.UpdatedAt)

	return nil
}

func (r NetworkPage) ResourceKey() string {
	return "networks"
}

func (r NetworkPage) NextPageURL() (string, error) {
	var s struct {
		Links []gophercloud.Link ` + "`" + `json:"networks_links"` + "`" + `
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return gophercloud.ExtractNextURL(s.Links)
}

func (r NetworkPage) IsEmpty() (bool, error) {
	if r.StatusCode == 204 {
		return true, nil
	}

	is, err := ExtractNetworks(r)
	return len(is) == 0, err
}

func ExtractNetworks(r pagination.Page) ([]Network, error) {
	var s []Network
	err := ExtractNetworksInto(r, &s)
	return s, err
}

func ExtractNetworksInto(r pagination.Page, v any) error {
	return r.(NetworkPage).ExtractIntoSlicePtr(v, "networks")
}

func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL("networks", id)
}

func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("networks")
}

func getURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func listURL(c *gophercloud.ServiceClient) string {
	return rootURL(c)
}
`

// Original native functions include the JSON-number-aware page dependency.
const pinnedSubnetBodyIdentitySource = `package fixture
func (opts ListOpts) ToSubnetListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}

func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToSubnetListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		return SubnetPage{pagination.LinkedPageBase{PageResult: r}}
	})
}

func Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := c.Get(ctx, getURL(c, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}

func (r commonResult) Extract() (*Subnet, error) {
	var s struct {
		Subnet *Subnet ` + "`" + `json:"subnet"` + "`" + `
	}
	err := r.ExtractInto(&s)
	return s.Subnet, err
}

func (r *Subnet) UnmarshalJSON(b []byte) error {
	type tmp Subnet

	// Support for older neutron time format
	var s1 struct {
		tmp
		CreatedAt gophercloud.JSONRFC3339NoZ ` + "`" + `json:"created_at"` + "`" + `
		UpdatedAt gophercloud.JSONRFC3339NoZ ` + "`" + `json:"updated_at"` + "`" + `
	}

	err := json.Unmarshal(b, &s1)
	if err == nil {
		*r = Subnet(s1.tmp)
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

	*r = Subnet(s2.tmp)
	r.CreatedAt = time.Time(s2.CreatedAt)
	r.UpdatedAt = time.Time(s2.UpdatedAt)

	return nil
}

func (r SubnetPage) NextPageURL() (string, error) {
	var s struct {
		Links []gophercloud.Link ` + "`" + `json:"subnets_links"` + "`" + `
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return gophercloud.ExtractNextURL(s.Links)
}

func (r SubnetPage) IsEmpty() (bool, error) {
	if r.StatusCode == 204 {
		return true, nil
	}

	is, err := ExtractSubnets(r)
	return len(is) == 0, err
}

func ExtractSubnets(r pagination.Page) ([]Subnet, error) {
	var s struct {
		Subnets []Subnet ` + "`" + `json:"subnets"` + "`" + `
	}
	err := (r.(SubnetPage)).ExtractInto(&s)
	return s.Subnets, err
}

func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL("subnets", id)
}

func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("subnets")
}

func listURL(c *gophercloud.ServiceClient) string {
	return rootURL(c)
}

func getURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func PageResultFrom(resp *http.Response) (PageResult, error) {
	var parsedBody any

	defer resp.Body.Close()
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return PageResult{}, err
	}

	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		dec := json.NewDecoder(bytes.NewReader(rawBody))
		dec.UseNumber()
		err = dec.Decode(&parsedBody)
		if err != nil {
			return PageResult{}, err
		}
	} else {
		parsedBody = rawBody
	}

	return PageResultFromParsed(resp, parsedBody), err
}

func PageResultFromParsed(resp *http.Response, body any) PageResult {
	return PageResult{
		Result: gophercloud.Result{
			Body:       body,
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
		},
		URL: *resp.Request.URL,
	}
}

func (current LinkedPageBase) GetBody() any {
	return current.Body
}
`
