package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const securityGroupRuleFixtureSource = "package rules\nimport \"time\"\ntype SecGroupRule struct{ID string;Direction string;Description string `json:\"description\"`;EtherType string `json:\"ethertype\"`;SecGroupID string `json:\"security_group_id\"`;PortRangeMin int `json:\"port_range_min\"`;PortRangeMax int `json:\"port_range_max\"`;Protocol string;RemoteAddressGroupID string `json:\"remote_address_group_id\"`;RemoteGroupID string `json:\"remote_group_id\"`;RemoteIPPrefix string `json:\"remote_ip_prefix\"`;TenantID string `json:\"tenant_id\"`;ProjectID string `json:\"project_id\"`;RevisionNumber int `json:\"revision_number\"`;CreatedAt time.Time `json:\"-\"`;UpdatedAt time.Time `json:\"-\"`}\nfunc(*SecGroupRule)UnmarshalJSON([]byte)error{return nil}\n"

func checkedSecurityGroupFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	b, e := os.ReadFile(filepath.Join("../../..", securityGroupFilterManifestPath))
	if e != nil {
		t.Fatal(e)
	}
	m, e := decodePythonFilterManifest(b)
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func securityGroupFilterFixtureSource() string {
	spec := identityCollectionSpec{path: securityGroupSDKPath, model: "SecGroup", getter: "Get", lister: "List", rawListIterator: "IterateSecurityGroups"}
	return strings.Replace(identityQueryFixtureSource(spec), "package fixture", "package groups", 1) + "\nfunc Delete(ctx context.Context,client *gophercloud.ServiceClient,id string)error{return nil}\n"
}
func securityGroupFilterFixture(t *testing.T, s string) (*types.Package, *collectionPlan) {
	t.Helper()
	return identityQueryFixture(t, identityCollectionSpec{path: securityGroupSDKPath, model: "SecGroup", getter: "Get", lister: "List", rawListIterator: "IterateSecurityGroups"}, s)
}

func TestSecurityGroupPythonManifestKeepsQueryAliasesAndOnlyThreeJSONBodyProperties(t *testing.T) {
	m := checkedSecurityGroupFilterManifest(t)
	if !securityGroupPythonFilterMetadataValid(m) || len(m.Proof.Files) != 7 || len(m.Proof.Nodes) != 39 || len(m.Reserved) != 9 || m.Query["tenant_id"] != "tenant_id" || m.Query["project_id"] != "project_id" || m.Query["revision_number"] != "revision_number" || m.Query["is_shared"] != "shared" || m.Query["fields"] != "fields" || m.Query["stateful"] != "stateful" {
		t.Fatal(m)
	}
	for _, name := range []string{"created_at", "updated_at"} {
		if m.Body[name].ResponseType != nil || m.Body[name].Field != name {
			t.Fatal(name, m.Body[name])
		}
	}
	if m.Body["security_group_rules"].ResponseType == nil || *m.Body["security_group_rules"].ResponseType != "list" {
		t.Fatal(m.Body)
	}
	for _, name := range []string{"id", "name", "tenant_id", "project_id", "revision_number", "stateful", "is_shared", "description", "tags", "rules"} {
		if _, ok := m.Body[name]; ok {
			t.Fatal("query/wire field incorrectly local", name)
		}
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"tenant-remap":     func(c *pythonFilterManifest) { c.Query["tenant_id"] = "project_id" },
		"shared-canonical": func(c *pythonFilterManifest) { delete(c.Query, "is_shared"); c.Query["shared"] = "shared" },
		"rule-native-name": func(c *pythonFilterManifest) {
			c.Body["rules"] = c.Body["security_group_rules"]
			delete(c.Body, "security_group_rules")
		},
		"rules-coercion": func(c *pythonFilterManifest) {
			f := c.Body["security_group_rules"]
			f.ResponseType = nil
			c.Body["security_group_rules"] = f
		},
		"timestamp-type": func(c *pythonFilterManifest) {
			f := c.Body["created_at"]
			kind := "str"
			f.ResponseType = &kind
			c.Body["created_at"] = f
		},
		"NetworkResource-parent": func(c *pythonFilterManifest) { c.ClassBases = []string{"resource.Resource", "_base.TagMixinNetwork"} },
		"Protocol-MRO":           func(c *pythonFilterManifest) { c.MRO = c.MRO[:6] },
		"accepted-count":         func(c *pythonFilterManifest) { c.Counts.AcceptedQuery = 17 },
	} {
		t.Run(name, func(t *testing.T) {
			c := clonePythonFilterManifest(t, m)
			mutate(c)
			if securityGroupPythonFilterMetadataValid(c) {
				t.Fatal("source metadata drift accepted")
			}
		})
	}
	for _, node := range m.Proof.Nodes {
		t.Run(node.Symbol, func(t *testing.T) {
			c := clonePythonFilterManifest(t, m)
			for i, n := range c.Proof.Nodes {
				if n.Symbol == node.Symbol {
					c.Proof.Nodes = append(c.Proof.Nodes[:i], c.Proof.Nodes[i+1:]...)
					break
				}
			}
			if securityGroupPythonFilterMetadataValid(c) {
				t.Fatal("classification proof anchor missing")
			}
		})
	}
}
func securityGroupFixtureWithRules(t *testing.T, source, ruleSource string) (*types.Package, *collectionPlan) {
	t.Helper()
	pkg, decls := typedCollectionFixtureWithRuleSource(t, upstreamModule+"/openstack/networking/v2/extensions/security/groups", source, ruleSource)
	plan, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls))
	if err != nil {
		t.Fatal(err)
	}
	return pkg, plan
}
func TestSecurityGroupBodyNativeSchemaGuardsConcreteListAndBothTimestampDecoders(t *testing.T) {
	source := securityGroupFilterFixtureSource()
	pkg, plan := securityGroupFilterFixture(t, source)
	if !securityGroupBodyNativeSchema(pkg, plan) || len(bodyFilterCollectionSpecs) != 11 || len(identityCollectionSpecs) != 20 || !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
	for name, pair := range map[string][2]string{
		"ID-untagged":             {"ID string;Name string", "ID string `json:\"id\"`;Name string"},
		"rules-type":              {"Rules []rules.SecGroupRule", "Rules []string"},
		"rule-json-key":           {"json:\"security_group_rules\"", "json:\"rules\""},
		"time-field":              {"CreatedAt time.Time", "CreatedAt string"},
		"native-revision":         {"RevisionNumber int `json:\"revision_number\"`", "RevisionNumber float64 `json:\"revision_number\"`"},
		"group-value-decoder":     {"func(*SecGroup)UnmarshalJSON", "func(SecGroup)UnmarshalJSON"},
		"group-decoder-param":     {"UnmarshalJSON([]byte)", "UnmarshalJSON(string)"},
		"query-bool-presence":     {"Stateful *bool `q:\"stateful\"`", "Stateful bool `q:\"stateful\"`"},
		"query-revision-presence": {"RevisionNumber *int `q:\"revision_number\"`", "RevisionNumber int `q:\"revision_number\"`"},
		"no-native-fields":        {"type ListOpts struct{", "type ListOpts struct{Fields []string `q:\"fields\"`;"},
		"native-list-pointer":     {"opts ListOpts)", "opts *ListOpts)"},
		"page-extra-field":        {"SecGroupPage struct{pagination.LinkedPageBase}", "SecGroupPage struct{Extra bool;pagination.LinkedPageBase}"},
		"page-value-method":       {"func(SecGroupPage)NextPageURL", "func(*SecGroupPage)NextPageURL"},
		"whole-extractor":         {"([]SecGroup,error)", "([]string,error)"},
		"common-pointer":          {"func(commonResult)Extract", "func(*commonResult)Extract"},
		"root-result-embedding":   {"commonResult struct{gophercloud.Result}", "commonResult struct{gophercloud.Result;Extra bool}"},
	} {
		t.Run(name, func(t *testing.T) {
			changed := strings.ReplaceAll(source, pair[0], pair[1])
			if changed == source {
				t.Fatal("mutation absent")
			}
			p, pl := securityGroupFilterFixture(t, changed)
			if securityGroupBodyNativeSchema(p, pl) {
				t.Fatal("native type drift enabled")
			}
			if validateBodyFilterCollectionContracts(p, pl) == nil {
				t.Fatal("capability silently dropped")
			}
		})
	}
	for name, pair := range map[string][2]string{
		"Rule-range-integer":        {"PortRangeMin int", "PortRangeMin string"},
		"Rule-nullable-native-date": {"CreatedAt time.Time", "CreatedAt string"},
		"Rule-wire-key":             {"json:\"remote_ip_prefix\"", "json:\"remote_cidr\""},
		"Rule-pointer-decoder":      {"func(*SecGroupRule)UnmarshalJSON", "func(SecGroupRule)UnmarshalJSON"},
		"Rule-decoder-param":        {"UnmarshalJSON([]byte)", "UnmarshalJSON(string)"},
	} {
		t.Run(name, func(t *testing.T) {
			changed := strings.ReplaceAll(securityGroupRuleFixtureSource, pair[0], pair[1])
			if changed == securityGroupRuleFixtureSource {
				t.Fatal("rule mutation absent")
			}
			p, pl := securityGroupFixtureWithRules(t, source, changed)
			if securityGroupBodyNativeSchema(p, pl) {
				t.Fatal("nested rule decoder drift enabled")
			}
		})
	}
	for name, tail := range map[string]string{"group-extra-method": "\nfunc(SecGroup)Other(){}", "list-own-method": "\nfunc(ListOpts)ToListQuery()(string,error){return \"\",nil}", "invented-list-builder": "\ntype ListOptsBuilder interface{ToListQuery()(string,error)}", "own-page-body": "\nfunc(SecGroupPage)GetBody()any{return nil}", "get-own-extractor": "\nfunc(GetResult)Extract()(*SecGroup,error){return nil,nil}"} {
		t.Run(name, func(t *testing.T) {
			p, pl := securityGroupFilterFixture(t, source+tail)
			if securityGroupBodyNativeSchema(p, pl) {
				t.Fatal("new method bypassed exact graph")
			}
		})
	}
	p, pl := securityGroupFixtureWithRules(t, source, securityGroupRuleFixtureSource+"\nfunc(SecGroupRule)Other(){}")
	if securityGroupBodyNativeSchema(p, pl) {
		t.Fatal("new Rule method accepted")
	}
}

func securityGroupAllManifests(t *testing.T) generator {
	t.Helper()
	return generator{pythonFilters: checkedPythonFilterManifest(t), secretPythonFilters: checkedSecretFilterManifest(t), containerPythonFilters: checkedContainerFilterManifest(t), orderPythonFilters: checkedOrderFilterManifest(t), addressGroupPythonFilters: checkedAddressGroupFilterManifest(t), qosPolicyPythonFilters: checkedQoSPolicyFilterManifest(t), subnetPoolPythonFilters: checkedSubnetPoolFilterManifest(t), networkPythonFilters: checkedNetworkFilterManifest(t), routerPythonFilters: checkedRouterFilterManifest(t), securityGroupPythonFilters: checkedSecurityGroupFilterManifest(t)}
}
func securityGroupPriorManifests(t *testing.T) map[string]*pythonFilterManifest {
	t.Helper()
	return map[string]*pythonFilterManifest{"": checkedPythonFilterManifest(t), secretPythonResource: checkedSecretFilterManifest(t), containerPythonResource: checkedContainerFilterManifest(t), orderPythonResource: checkedOrderFilterManifest(t), addressGroupPythonResource: checkedAddressGroupFilterManifest(t), qosPolicyPythonResource: checkedQoSPolicyFilterManifest(t), subnetPoolPythonResource: checkedSubnetPoolFilterManifest(t), networkPythonResource: checkedNetworkFilterManifest(t), routerPythonResource: checkedRouterFilterManifest(t)}
}
func securityGroupPriorFixture(t *testing.T, path string) (*types.Package, *collectionPlan) {
	t.Helper()
	switch path {
	case "keymanager/v1/secrets":
		return secretFilterNativeFixture(t, secretFilterNativeFixtureSource)
	case "keymanager/v1/containers":
		return containerFilterNativeFixture(t, containerFilterNativeFixtureSource)
	case "keymanager/v1/orders":
		return orderFilterNativeFixture(t, orderFilterNativeFixtureSource)
	case addressGroupSDKPath:
		return addressGroupFilterNativeFixture(t, addressGroupFilterFixtureSource())
	case qosPolicySDKPath:
		return qosPolicyFilterNativeFixture(t, qosPolicyFilterFixtureSource())
	case subnetPoolSDKPath:
		return subnetPoolFilterNativeFixture(t, subnetPoolFilterFixtureSource())
	case networkSDKPath:
		return networkFilterNativeFixture(t, networkFilterFixtureSource())
	case routerSDKPath:
		return routerFilterNativeFixture(t, routerFilterFixtureSource())
	case "network/v2/subnets":
		spec := identityCollectionSpec{path: path, model: "Subnet", getter: "Get", lister: "List"}
		source := strings.Replace(identityQueryFixtureSource(spec), "package fixture", "package subnets", 1) + "\nfunc Delete(ctx context.Context,client *gophercloud.ServiceClient,id string)error{return nil}\n"
		return identityQueryFixture(t, spec, source)
	}
	t.Fatal("unreviewed fixture", path)
	return nil, nil
}
func TestSecurityGroupFilterEmissionPreservesOwnedFullQueryPagersAndPriorNineBindings(t *testing.T) {
	pkg, plan := securityGroupFilterFixture(t, securityGroupFilterFixtureSource())
	if (&generator{}).validatePythonFilterPlan(pkg, plan) == nil {
		t.Fatal("source proof optional")
	}
	g := securityGroupAllManifests(t)
	g.root = t.TempDir()
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(g.root, securityGroupSDKPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := g.emitCollection(pkg, plan); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(g.root, securityGroupSDKPath, "resources_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{"securityGroupBodyFilterValue(record, key)", "nativefind.IterateSecurityGroupBodies(ctx, a.RawClient(), cfg.Query, control)", "nativefind.IterateSecurityGroups(ctx, a.RawClient(), q, control)", "config.Query[key] = append([]string(nil), values...)", `request.Wrap("List", "groups", err)`, "IdentityFind:", "GetIdentityQuery:", "Delete:", `[]string{"security-groups", id}`} {
		if !strings.Contains(text, want) {
			t.Fatal("owned path changed", want, text)
		}
	}
	for _, bad := range []string{`q.Del("status")`, "WithListQuery(", "BodyFilterValue:", "json.Marshal(v.", "upstream.List(", "listOptsBuilder", "ResourceAdapter", "IdentityAllProjectsQuery:", "IdentityExtraSpecs:", "Failed:", "Status:", `"tenant_id": "project_id"`} {
		if strings.Contains(text, bad) {
			t.Fatal("unproved policy emitted", bad, text)
		}
	}
	record := collectionRecord{BodyFilterFields: bodyFilterCollectionFields(pkg, plan, 0), SemanticQueryFilters: pythonFilterQueryFields(g.securityGroupPythonFilters), SemanticBodyFilters: pythonFilterBodyFields(g.securityGroupPythonFilters), SemanticReserved: pythonFilterReserved(g.securityGroupPythonFilters)}
	if len(record.BodyFilterFields) != 3 || len(record.SemanticQueryFilters) != 17 || len(record.SemanticBodyFilters) != 3 || len(record.SemanticReserved) != 9 || record.SemanticBodyFilters["security_group_rules"] != "security_group_rules" || record.SemanticQueryFilters["tenant_id"] != "tenant_id" {
		t.Fatal(record)
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
		t.Fatal("unscoped descriptor leaked into scope")
	}
	for _, spec := range identityCollectionSpecs {
		if spec.path == securityGroupSDKPath {
			continue
		}
		p, pl := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
		if (&generator{securityGroupPythonFilters: g.securityGroupPythonFilters}).pythonFilterFor(p, pl) != nil {
			t.Fatal("descriptor leaked", spec.path)
		}
	}
	for _, path := range []string{"network/v2/subnets", "keymanager/v1/secrets", "keymanager/v1/containers", "keymanager/v1/orders", addressGroupSDKPath, qosPolicySDKPath, subnetPoolSDKPath, networkSDKPath, routerSDKPath} {
		t.Run(path, func(t *testing.T) {
			p, pl := securityGroupPriorFixture(t, path)
			copy := securityGroupAllManifests(t)
			copy.root = t.TempDir()
			if err := os.MkdirAll(filepath.Join(copy.root, path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := copy.emitCollection(p, pl); err != nil {
				t.Fatal(err)
			}
			actual, err := os.ReadFile(filepath.Join(copy.root, path, "resources_generated.go"))
			if err != nil {
				t.Fatal(err)
			}
			previous, err := os.ReadFile(filepath.Join("../../..", path, "resources_generated.go"))
			if err != nil {
				t.Fatal(err)
			}
			names := []string{"API.newResources", "API.listBodyWithControl"}
			if path == networkSDKPath {
				names = append(names, "API.ResourceAdapter")
			}
			for _, name := range names {
				if emittedFunctionHash(t, actual, name) != emittedFunctionHash(t, previous, name) {
					t.Fatal("prior generated function changed", name)
				}
			}
		})
	}
}
func TestSecurityGroupPythonPinnedSourceRequiresCompleteLiveClassificationProof(t *testing.T) {
	m := checkedSecurityGroupFilterManifest(t)
	if verifySecurityGroupPythonFilterManifest("", m) == nil || verifySecurityGroupPythonFilterManifest(t.TempDir(), m) == nil {
		t.Fatal("source optional")
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/go-openstacksdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/network/v2/security_group.py")); err != nil {
		t.Skip("pinned source unavailable")
	}
	if err := verifySecurityGroupPythonFilterManifest(source, m); err != nil {
		t.Fatal(err)
	}
	fresh, err := extractPythonFilterManifestTarget(source, securityGroupPythonResource)
	if err != nil || !reflect.DeepEqual(fresh, m) {
		t.Fatal(fresh, err)
	}
	temp := t.TempDir()
	for path := range securityGroupFilterSourceHashes {
		b, err := os.ReadFile(filepath.Join(source, path))
		if err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(temp, path)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for path := range securityGroupFilterSourceHashes {
		t.Run(path, func(t *testing.T) {
			dest := filepath.Join(temp, path)
			b, err := os.ReadFile(dest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dest, append(b, []byte("\n# source drift\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifySecurityGroupPythonFilterManifest(temp, m); err == nil || !strings.Contains(err.Error(), path) {
				t.Fatal("source SHA guard bypassed", err)
			}
			if err := os.WriteFile(dest, b, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){"AST": func(c *pythonFilterManifest) { c.Proof.Nodes[0].ASTSHA256 = "wrong" }, "parser": func(c *pythonFilterManifest) { c.Proof.PythonParser = "different" }, "controls": func(c *pythonFilterManifest) { c.SourceControls.ResourceList = c.SourceControls.ResourceList[:6] }, "reserved": func(c *pythonFilterManifest) { c.Reserved = c.Reserved[:8] }, "foreign-proof": func(c *pythonFilterManifest) { c.Proof.Files["../foreign.py"] = "wrong" }} {
		t.Run(name, func(t *testing.T) {
			c := clonePythonFilterManifest(t, m)
			mutate(c)
			if verifySecurityGroupPythonFilterManifest(temp, c) == nil {
				t.Fatal("live source proof bypassed")
			}
		})
	}
	loaded, err := loadSecurityGroupPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, m) {
		t.Fatal(loaded, err)
	}
	for target, expected := range securityGroupPriorManifests(t) {
		actual, err := extractPythonFilterManifestTarget(source, target)
		if err != nil || !reflect.DeepEqual(actual, expected) {
			t.Fatal("prior manifest changed", target, err)
		}
	}
}
func securityGroupActualNative(t *testing.T) (generator, *types.Package, *collectionPlan, map[string]*ast.FuncDecl) {
	t.Helper()
	path := os.Getenv("GOPHERCLOUD_METADATA")
	if path == "" {
		path = "/private/tmp/go-openstacksdk-upstream-packages.json"
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		t.Skip("compiled pinned metadata unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	meta := map[string]metadata{}
	decoder := json.NewDecoder(file)
	for {
		var entry metadata
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		meta[entry.ImportPath] = entry
	}
	imp := importer.ForCompiler(token.NewFileSet(), "gc", func(path string) (io.ReadCloser, error) {
		m, ok := meta[path]
		if !ok || m.Export == "" {
			return nil, os.ErrNotExist
		}
		return os.Open(m.Export)
	})
	for _, path := range []string{upstreamModule, securityGroupRulesNativePath} {
		if _, err := imp.Import(path); err != nil {
			t.Fatal(err)
		}
	}
	path = upstreamModule + "/openstack/networking/v2/extensions/security/groups"
	pkg, err := imp.Import(path)
	if err != nil {
		t.Fatal(err)
	}
	decls, native := map[string]*ast.FuncDecl{}, map[string]*ast.FuncDecl{}
	for _, name := range meta[path].GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(meta[path].Dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range file.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok {
				native[identityDeclarationKey(fn)] = fn
				if fn.Recv == nil && ast.IsExported(fn.Name.Name) {
					decls[fn.Name.Name] = fn
				}
			}
		}
	}
	plan, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls))
	if err != nil {
		t.Fatal(err)
	}
	g := generator{meta: meta, importer: imp, securityGroupPythonFilters: checkedSecurityGroupFilterManifest(t)}
	for _, load := range []func(string) (map[string]*ast.FuncDecl, error){g.bodyRecordRootDeclarations, g.identityPaginationDeclarations, g.securityGroupBodyRuleDeclarations} {
		extra, err := load(path)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range extra {
			native[k] = v
		}
	}
	return g, pkg, plan, native
}
func securityGroupFunctionSource(t *testing.T, decls map[string]*ast.FuncDecl, prefix string) string {
	t.Helper()
	var b bytes.Buffer
	for key, fn := range decls {
		if strings.HasPrefix(key, prefix) {
			if err := format.Node(&b, token.NewFileSet(), fn); err != nil {
				t.Fatal(err)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
func TestSecurityGroupDependenciesGuardNestedRuleWholePageNumbersCodesAndConstants(t *testing.T) {
	_, pkg, plan, decls := securityGroupActualNative(t)
	if len(securityGroupBodyNativeDeclarations) != 21 {
		t.Fatal("unexpected dependency union")
	}
	if err := validateSecurityGroupBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	for name := range securityGroupBodyNativeDeclarations {
		t.Run("hash-"+name, func(t *testing.T) {
			copy := map[string]*ast.FuncDecl{}
			for k, v := range decls {
				copy[k] = v
			}
			fn := *copy[name]
			block := *fn.Body
			block.List = append(append([]ast.Stmt(nil), block.List...), &ast.ExprStmt{X: ast.NewIdent("changed")})
			fn.Body = &block
			copy[name] = &fn
			if validateSecurityGroupBodyNativeDeclarations(pkg, copy, plan) == nil {
				t.Fatal("reachable native hash drift accepted")
			}
			delete(copy, name)
			if validateSecurityGroupBodyNativeDeclarations(pkg, copy, plan) == nil {
				t.Fatal("required declaration absent")
			}
		})
	}
	dir := t.TempDir()
	sources := map[string]string{"root.go": "package gophercloud\nconst RFC3339NoZ=\"2006-01-02T15:04:05\"\n" + securityGroupFunctionSource(t, decls, "gophercloud."), "page.go": "package pagination\n" + securityGroupFunctionSource(t, decls, "pagination."), "rule.go": "package rules\n" + securityGroupFunctionSource(t, decls, "rules.")}
	leaf := "package groups\nconst rootPath=\"security-groups\"\n"
	for key, fn := range decls {
		if strings.HasPrefix(key, "gophercloud.") || strings.HasPrefix(key, "pagination.") || strings.HasPrefix(key, "rules.") {
			continue
		}
		var b bytes.Buffer
		if err := format.Node(&b, token.NewFileSet(), fn); err != nil {
			t.Fatal(err)
		}
		leaf += b.String() + "\n"
	}
	sources["leaf.go"] = leaf
	for name, source := range sources {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := pkg.Path()
	g := generator{meta: map[string]metadata{upstreamModule: {Dir: dir, GoFiles: []string{"root.go"}}, upstreamModule + "/pagination": {Dir: dir, GoFiles: []string{"page.go"}}, path: {Dir: dir, GoFiles: []string{"leaf.go"}}, securityGroupRulesNativePath: {Dir: dir, GoFiles: []string{"rule.go"}}}}
	for label, load := range map[string]func(string) (map[string]*ast.FuncDecl, error){"root": g.bodyRecordRootDeclarations, "page": g.identityPaginationDeclarations, "rules": g.securityGroupBodyRuleDeclarations} {
		got, err := load(path)
		count := map[string]int{"root": 4, "page": 7, "rules": 1}[label]
		if err != nil || len(got) != count {
			t.Fatal(label, got, err)
		}
	}
	for _, path := range []string{upstreamModule + "/openstack/networking/v2/ports", upstreamModule + "/openstack/networking/v2/extensions/agents"} {
		for _, load := range []func(string) (map[string]*ast.FuncDecl, error){g.bodyRecordRootDeclarations, g.securityGroupBodyRuleDeclarations} {
			d, err := load(path)
			if err != nil || d != nil {
				t.Fatal("bounded dependency leaked", path, d, err)
			}
		}
	}
	old, err := g.addressGroupBodyRootDeclarations(upstreamModule + "/openstack/" + addressGroupSDKPath)
	if err != nil || len(old) != 2 {
		t.Fatal("Address root boundary changed", old, err)
	}
	for name, entry := range map[string]struct {
		file, code string
		load       func(string) (map[string]*ast.FuncDecl, error)
	}{
		"NoZ-wrong":                  {"root.go", strings.Replace(sources["root.go"], "2006-01-02T15:04:05", "2006-01-02", 1), g.bodyRecordRootDeclarations},
		"NoZ-duplicate":              {"root.go", sources["root.go"] + "\nconst RFC3339NoZ=\"2006-01-02T15:04:05\"", g.bodyRecordRootDeclarations},
		"NoZ-missing":                {"root.go", strings.Replace(sources["root.go"], "const RFC3339NoZ=\"2006-01-02T15:04:05\"", "", 1), g.bodyRecordRootDeclarations},
		"root-declaration-duplicate": {"root.go", sources["root.go"] + "\nfunc ExtractNextURL(v any)string{return \"\"}", g.bodyRecordRootDeclarations},
		"route-wrong":                {"leaf.go", strings.Replace(leaf, "security-groups", "routers", 1), g.securityGroupBodyRuleDeclarations},
		"route-missing":              {"leaf.go", strings.Replace(leaf, "const rootPath=\"security-groups\"", "", 1), g.securityGroupBodyRuleDeclarations},
		"route-duplicate":            {"leaf.go", leaf + "\nconst rootPath=\"security-groups\"", g.securityGroupBodyRuleDeclarations},
		"rule-missing":               {"rule.go", "package rules", g.securityGroupBodyRuleDeclarations},
		"rule-duplicate":             {"rule.go", sources["rule.go"] + "\nfunc(*SecGroupRule)UnmarshalJSON(b []byte)error{return nil}", g.securityGroupBodyRuleDeclarations},
		"leaf-duplicate":             {"leaf.go", leaf + "\nfunc ExtractGroups(v any)error{return nil}", g.securityGroupBodyRuleDeclarations},
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(dir, entry.file), []byte(entry.code), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := entry.load(path); err == nil {
				t.Fatal("missing/duplicate dependency accepted")
			}
			if err := os.WriteFile(filepath.Join(dir, entry.file), []byte(sources[entry.file]), 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	for name, pair := range map[string][2]string{"UseNumber": {"dec.UseNumber()", ""}, "native-codes": {"[]int{200, 204, 300}", "[]int{200}"}, "IsEmpty-before-handler": {"empty, err := currentPage.IsEmpty()", "empty, err := false, error(nil)"}, "continuation-after-handler": {"currentURL, err = currentPage.NextPageURL()", "currentURL, err = \"\", error(nil)"}} {
		t.Run(name, func(t *testing.T) {
			source := strings.Replace(sources["page.go"], pair[0], pair[1], 1)
			if source == sources["page.go"] {
				t.Fatal("page mutation absent")
			}
			if err := os.WriteFile(filepath.Join(dir, "page.go"), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			pd, err := g.identityPaginationDeclarations(path)
			if err != nil {
				t.Fatal(err)
			}
			copy := map[string]*ast.FuncDecl{}
			for k, v := range decls {
				copy[k] = v
			}
			for k, v := range pd {
				copy[k] = v
			}
			if validateSecurityGroupBodyNativeDeclarations(pkg, copy, plan) == nil {
				t.Fatal("native page behavior drift ignored")
			}
			if err := os.WriteFile(filepath.Join(dir, "page.go"), []byte(sources["page.go"]), 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestSecurityGroupBodyFiltersAcceptActualCompiledGroupRuleAndConcreteRequestGraph(t *testing.T) {
	g, pkg, plan, decls := securityGroupActualNative(t)
	if err := validateBodyFilterCollectionContracts(pkg, plan); err != nil {
		t.Fatal(err)
	}
	if err := validateSecurityGroupBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	if pkg.Scope().Lookup("ListOptsBuilder") != nil || plan.listQueryBuilder {
		t.Fatal("native concrete signature turned into builder")
	}
	for name, count := range map[string]int{"SecGroup": 1, "ListOpts": 0, "SecGroupPage": 2, "commonResult": 1, "GetResult": 0, "CreateResult": 0, "UpdateResult": 0, "DeleteResult": 0, "CreateOpts": 1, "UpdateOpts": 1} {
		obj := pkg.Scope().Lookup(name)
		if obj == nil {
			t.Fatal("compiled type missing", name)
		}
		n, ok := obj.Type().(*types.Named)
		if !ok || n.NumMethods() != count {
			t.Fatal("actual own method graph", name, n)
		}
	}
	for _, name := range []string{"CreateOptsBuilder", "UpdateOptsBuilder"} {
		obj := pkg.Scope().Lookup(name)
		if obj == nil {
			t.Fatal(name)
		}
		iface, ok := obj.Type().Underlying().(*types.Interface)
		if !ok || iface.NumEmbeddeds() != 0 || iface.NumMethods() != 1 {
			t.Fatal("request-only native graph", name, iface)
		}
	}
	var rules *types.Package
	for _, dep := range pkg.Imports() {
		if dep.Path() == securityGroupRulesNativePath {
			rules = dep
		}
	}
	if rules == nil {
		t.Fatal("compiled nested rule package missing")
	}
	n := rules.Scope().Lookup("SecGroupRule").Type().(*types.Named)
	st := n.Underlying().(*types.Struct)
	if st.NumFields() != 16 || !securityGroupBodyPointerDecoder(n) {
		t.Fatal("actual rule type graph", n)
	}
	if len(bodyFilterCollectionFields(pkg, plan, 0)) != 3 || !identityCollectionEnabled(pkg, plan, 0) || plan.id != "ID" || plan.name != "Name" || plan.status != "" {
		t.Fatal(plan)
	}
}
