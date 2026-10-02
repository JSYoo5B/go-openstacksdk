package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func checkedSecretFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../..", secretFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func secretFilterNativeFixture(t *testing.T, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	return identityQueryFixture(t, identityCollectionSpec{path: "keymanager/v1/secrets", model: "Secret", getter: "Get", lister: "List"}, source)
}

func secretBodyPinnedDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "native_secret.go", pinnedSecretBodySource, 0)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			result[identityDeclarationKey(fn)] = fn
		}
	}
	for name, fn := range pinnedSubnetBodyIdentityDeclarations(t) {
		if strings.HasPrefix(name, "pagination.") {
			result[name] = fn
		}
	}
	return result
}

func TestSecretPythonFilterManifestKeepsDistinctPropertyAccessors(t *testing.T) {
	manifest := checkedSecretFilterManifest(t)
	if !secretPythonFilterMetadataValid(manifest) || len(manifest.Proof.Nodes) != 23 || len(manifest.Proof.Files) != 7 || len(manifest.Reserved) != 9 {
		t.Fatal(manifest)
	}
	if !reflect.DeepEqual(pythonFilterBodyFields(manifest), map[string]string{"id": "id", "bit_length": "bit_length", "content_types": "content_types", "expires_at": "expires_at", "created_at": "created_at", "updated_at": "updated_at", "secret_ref": "secret_ref", "secret_id": "secret_id", "status": "status", "payload": "payload", "payload_content_type": "payload_content_type", "payload_content_encoding": "payload_content_encoding"}) {
		t.Fatal("Body property keys collapsed onto raw wire fields")
	}
	if manifest.Body["id"].ResponseAccessor != "resource_id" || manifest.Body["secret_id"].Formatter != secretPythonFormatter || manifest.Body["secret_id"].Field != "secret_ref" || manifest.Query["algorithm"] != "alg" {
		t.Fatal(manifest.Body, manifest.Query)
	}
	for _, name := range []string{"created", "updated", "expiration", "name", "algorithm", "mode", "bits", "secret_type"} {
		if _, ok := manifest.Body[name]; ok {
			t.Fatal("wire/query field incorrectly classified as local Body", name)
		}
	}
	for _, data := range []string{`{"body":{"id":{"field":"id","response_type":null,"response_accessor":false}}}`, `{"body":{"secret_id":{"field":"secret_ref","formatter":"x","extra":true}}}`, `{} {}`} {
		if _, err := decodePythonFilterManifest([]byte(data)); err == nil {
			t.Fatal("unknown/ill-typed schema accepted", data)
		}
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"literal-id-accessor":   func(m *pythonFilterManifest) { f := m.Body["id"]; f.ResponseAccessor = ""; m.Body["id"] = f },
		"id-ref-collapse":       func(m *pythonFilterManifest) { m.Body["id"] = m.Body["secret_ref"] },
		"formatted-id-collapse": func(m *pythonFilterManifest) { m.Body["secret_id"] = m.Body["secret_ref"] },
		"unknown-formatter": func(m *pythonFilterManifest) {
			f := m.Body["secret_id"]
			f.Formatter = "other"
			m.Body["secret_id"] = f
		},
		"coercing-untyped-bits": func(m *pythonFilterManifest) {
			f := m.Body["bit_length"]
			typ := "int"
			f.ResponseType = &typ
			m.Body["bit_length"] = f
		},
		"date-wire-key": func(m *pythonFilterManifest) {
			m.Body["expiration"] = m.Body["expires_at"]
			delete(m.Body, "expires_at")
		},
		"query-alias":    func(m *pythonFilterManifest) { m.Query["algorithm"] = "algorithm" },
		"query-format":   func(m *pythonFilterManifest) { m.QueryFormats["name"] = "csv" },
		"inheritance":    func(m *pythonFilterManifest) { m.MRO = m.MRO[1:] },
		"unknown-policy": func(m *pythonFilterManifest) { m.UnknownFilters = "wire" },
		"count":          func(m *pythonFilterManifest) { m.Counts.AcceptedQuery = 12 },
	} {
		t.Run(name, func(t *testing.T) {
			changed := clonePythonFilterManifest(t, manifest)
			mutate(changed)
			if secretPythonFilterMetadataValid(changed) {
				t.Fatal("unaudited accessor/classification accepted")
			}
		})
	}
}

func TestSecretBodyFilterNativeContractRejectsDecoderPagerAndSourceDrift(t *testing.T) {
	pkg, plan := secretFilterNativeFixture(t, secretFilterNativeFixtureSource)
	if plan == nil || !secretBodyNativeSchema(pkg, plan) {
		t.Fatal(plan)
	}
	if identityCollectionEnabled(pkg, plan, 0) || len(identityCollectionSpecs) != 20 || len(bodyFilterCollectionSpecs) != 9 {
		t.Fatal("raw Body support opted into native identity or broadened inventory")
	}
	if err := validateSecretBodyNativeDeclarations(pkg, secretBodyPinnedDeclarations(t), plan); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"string-bitlength":       strings.Replace(secretFilterNativeFixtureSource, "BitLength int", "BitLength string", 1),
		"map-value":              strings.Replace(secretFilterNativeFixtureSource, "ContentTypes map[string]string", "ContentTypes map[string]any", 1),
		"timestamp-projection":   strings.Replace(secretFilterNativeFixtureSource, "Created time.Time `json:\"-\"`", "Created string `json:\"created\"`", 1),
		"raw-id-added":           strings.Replace(secretFilterNativeFixtureSource, "type Secret struct{", "type Secret struct{ID string;", 1),
		"creator-field-lost":     strings.Replace(secretFilterNativeFixtureSource, "CreatorID string `json:\"creator_id\"`;", "", 1),
		"decoder-removed":        strings.Replace(secretFilterNativeFixtureSource, "func(*Secret)UnmarshalJSON([]byte)error{return nil}", "", 1),
		"decoder-argument":       strings.Replace(secretFilterNativeFixtureSource, "UnmarshalJSON([]byte)", "UnmarshalJSON(string)", 1),
		"date-query-model":       strings.Replace(secretFilterNativeFixtureSource, "Date time.Time", "Date string", 1),
		"nullable-acl-lost":      strings.Replace(secretFilterNativeFixtureSource, "ACLOnly *bool", "ACLOnly bool", 1),
		"date-query-wire":        strings.Replace(secretFilterNativeFixtureSource, "CreatedQuery *DateQuery;", "CreatedQuery *DateQuery `q:\"created\"`;", 1),
		"name-query":             strings.Replace(secretFilterNativeFixtureSource, "Name string `q:\"name\"`", "Name string `q:\"pattern\"`", 1),
		"inherited-continuation": strings.Replace(secretFilterNativeFixtureSource, "func(SecretPage)NextPageURL()(string,error){return \"\",nil}", "", 1),
		"body-override":          secretFilterNativeFixtureSource + "\nfunc(SecretPage)GetBody()any{return nil}\n",
		"page-state":             strings.Replace(secretFilterNativeFixtureSource, "SecretPage struct{pagination.LinkedPageBase}", "SecretPage struct{extra bool;pagination.LinkedPageBase}", 1),
		"wrong-extractor":        strings.Replace(secretFilterNativeFixtureSource, "([]Secret,error)", "([]string,error)", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if source == secretFilterNativeFixtureSource {
				t.Fatal("mutation not applied")
			}
			changedPkg, changedPlan := secretFilterNativeFixture(t, source)
			if _, ok := bodyFilterCollectionContract(changedPkg, changedPlan, 0); ok {
				t.Fatal("native shape drift enabled")
			}
			if err := validateBodyFilterCollectionContracts(changedPkg, changedPlan); err == nil {
				t.Fatal("native shape drift silently dropped capability")
			}
		})
	}
	for name := range secretBodyNativeDeclarations {
		t.Run(name, func(t *testing.T) {
			declarations := secretBodyPinnedDeclarations(t)
			declarations[name].Body.List = append(declarations[name].Body.List, &ast.ExprStmt{X: ast.NewIdent("drift")})
			if err := validateSecretBodyNativeDeclarations(pkg, declarations, plan); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatal("pinned native implementation drift bypassed", err)
			}
		})
	}
	spec, _ := bodyFilterCollectionContract(pkg, plan, 0)
	for name, mutate := range map[string]func(*bodyFilterCollectionSpec){"typed-projection": func(s *bodyFilterCollectionSpec) { s.rawRecord = false }, "alias": func(s *bodyFilterCollectionSpec) { s.fields[0].aliases = []string{"bits"} }, "wire-collapse": func(s *bodyFilterCollectionSpec) { s.fields[2].key = "created" }, "member-projection": func(s *bodyFilterCollectionSpec) { s.fields[0].member = "BitLength" }, "extra": func(s *bodyFilterCollectionSpec) {
		s.fields = append(s.fields, bodyFilterCollectionField{key: "creator_id"})
	}} {
		t.Run(name, func(t *testing.T) {
			copy := spec
			copy.fields = append([]bodyFilterCollectionField(nil), spec.fields...)
			mutate(&copy)
			if bodyFilterCollectionMetadataValid(copy) {
				t.Fatal("unreviewed property metadata accepted")
			}
		})
	}
}

func TestSecretSemanticFilterEmissionKeepsNativePoliciesAndQueryOwnership(t *testing.T) {
	pkg, plan := secretFilterNativeFixture(t, secretFilterNativeFixtureSource)
	manifest := checkedSecretFilterManifest(t)
	if err := (&generator{}).validatePythonFilterPlan(pkg, plan); err == nil {
		t.Fatal("unchecked manifest enabled")
	}
	g := generator{secretPythonFilters: manifest}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}, pythonFilters: g.pythonFilterFor(pkg, plan)}
	e.printf("func(a *API)newResources()*resource.Collection[Secret]{return ")
	emitCollectionAdapter(&e, plan, "a", nil)
	e.printf("}\n")
	emitBodyFilterList(&e, plan)
	source, err := e.source()
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, wanted := range []string{"secretBodyFilterValue(record, key)", "resource.BodyStreamWithControl(ctx, upstream.List(a.client, _opts)", "upstream.ExtractSecrets(page)", `"secrets", control`, "return a.listWithControl(ctx, control, options...)", "return a.listBodyWithControl(ctx, control, options...)", `q.Del("status")`, `path.Base(parsed.Path)`, "NameQuery:", "Status:", "Failed:", "Delete:"} {
		if !strings.Contains(text, wanted) {
			t.Fatal("missing owned/retained policy", wanted, text)
		}
	}
	if strings.Count(text, "config.Query[key] = append([]string(nil), values...)") != 2 || strings.Count(text, `q.Del("status")`) != 2 {
		t.Fatal("both iterators must copy whole queries and retain status suppression", text)
	}
	for _, forbidden := range []string{"WithListQuery(", "IdentityFind:", "GetIdentityQuery:", "IdentityResponseID:", "IterateIdentity:", "BodyFilterValue:", "reflect.", "json.Marshal(v."} {
		if strings.Contains(text, forbidden) {
			t.Fatal("unproved behavior/projection added", forbidden, text)
		}
	}
	if strings.Contains(text, `"expires_at": "expiration"`) || strings.Contains(text, `"secret_id": "secret_ref"`) {
		t.Fatal("distinct property names collapsed", text)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "emitted.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	ast.Inspect(file, func(n ast.Node) bool {
		if pair, ok := n.(*ast.KeyValueExpr); ok {
			if key, ok := pair.Key.(*ast.Ident); ok {
				if key.Name == "FilterDescriptor" || key.Name == "BodyFilterFields" || key.Name == "IterateBodyControlled" {
					counts[key.Name]++
				}
			}
		}
		return true
	})
	if counts["FilterDescriptor"] != 1 || counts["BodyFilterFields"] != 1 || counts["IterateBodyControlled"] != 1 {
		t.Fatal(counts)
	}
	record := collectionRecord{BodyFilterFields: bodyFilterCollectionFields(pkg, plan, 0), SemanticQueryFilters: pythonFilterQueryFields(manifest), SemanticBodyFilters: pythonFilterBodyFields(manifest), SemanticReserved: pythonFilterReserved(manifest)}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.BodyFilterFields) != 12 || len(record.SemanticQueryFilters) != 12 || len(record.SemanticBodyFilters) != 12 || strings.Contains(string(data), `"identity_find":true`) || record.SemanticBodyFilters["secret_id"] != "secret_id" {
		t.Fatal(record, string(data))
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
		t.Fatal("raw global collection leaked into scoped binding")
	}
	for _, identity := range identityCollectionSpecs {
		otherPkg, otherPlan := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		if g.pythonFilterFor(otherPkg, otherPlan) != nil {
			t.Fatal("Secret semantic metadata leaked into other collection", identity.path)
		}
	}
}

func TestSecretPythonFilterPinnedSourceRequiresSHAAndIndependentAST(t *testing.T) {
	manifest := checkedSecretFilterManifest(t)
	if err := verifySecretPythonFilterManifest("", manifest); err == nil {
		t.Fatal("no source accepted")
	}
	if err := verifySecretPythonFilterManifest(t.TempDir(), manifest); err == nil || !strings.Contains(err.Error(), "Python filter source") {
		t.Fatal(err)
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/gophercloudsdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/key_manager/v1/secret.py")); err != nil {
		t.Skip("audited source checkout not present")
	}
	if err := verifySecretPythonFilterManifest(source, manifest); err != nil {
		t.Fatal(err)
	}
	fresh, err := extractPythonFilterManifestTarget(source, secretPythonResource)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fresh, manifest) {
		t.Fatal("independent target extraction mismatch")
	}
	root := t.TempDir()
	for path := range secretFilterSourceHashes {
		data, err := os.ReadFile(filepath.Join(source, path))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for path := range secretFilterSourceHashes {
		t.Run(path, func(t *testing.T) {
			target := filepath.Join(root, path)
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, append(data, []byte("\n# drift\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifySecretPythonFilterManifest(root, manifest); err == nil || !strings.Contains(err.Error(), path) {
				t.Fatal("source SHA guard bypassed", err)
			}
			if err := os.WriteFile(target, data, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){"reserved": func(m *pythonFilterManifest) { m.Reserved = m.Reserved[:8] }, "proof-node": func(m *pythonFilterManifest) { m.Proof.Nodes[0].ASTSHA256 = "wrong" }, "parser": func(m *pythonFilterManifest) { m.Proof.PythonParser = "other" }, "file-map": func(m *pythonFilterManifest) { m.Proof.Files["openstack/format.py"] = "wrong" }, "path-escape": func(m *pythonFilterManifest) { m.Proof.Files["../outside.py"] = "wrong" }, "source-control": func(m *pythonFilterManifest) { m.SourceControls.ResourceList = m.SourceControls.ResourceList[:6] }} {
		t.Run(name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, manifest)
			mutate(copy)
			if err := verifySecretPythonFilterManifest(root, copy); err == nil {
				t.Fatal("manifest replaced independent live proof")
			}
		})
	}
	loaded, err := loadSecretPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, manifest) {
		t.Fatal(loaded, err)
	}
	// The existing Subnet default target and its complete checked-in metadata
	// remain independent; the extra optional fields must be absent there.
	subnet, err := extractPythonFilterManifest(source)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(subnet, checkedPythonFilterManifest(t)) {
		t.Fatal("Secret target changed default Subnet extraction")
	}
}

func TestSecretRawPageDependencyLoaderRequiresOriginalNumberAndBodyContract(t *testing.T) {
	pkg, plan := secretFilterNativeFixture(t, secretFilterNativeFixtureSource)
	if _, err := (&generator{}).identityPaginationDeclarations(pkg.Path()); err == nil || !strings.Contains(err.Error(), "Secret body collection") {
		t.Fatal("missing raw-page dependency metadata accepted", err)
	}
	if declarations, err := (&generator{}).identityPaginationDeclarations(upstreamModule + "/openstack/identity/v3/projects"); err != nil || len(declarations) != 0 {
		t.Fatal("dependency requirement leaked into an unrelated binding", declarations, err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "page.go")
	if err := os.WriteFile(file, []byte(pinnedSubnetBodyIdentitySource), 0600); err != nil {
		t.Fatal(err)
	}
	g := generator{meta: map[string]metadata{upstreamModule + "/pagination": {Dir: dir, GoFiles: []string{"page.go"}}}}
	dependencies, err := g.identityPaginationDeclarations(pkg.Path())
	if err != nil || len(dependencies) != 3 {
		t.Fatal(dependencies, err)
	}
	declarations := secretBodyPinnedDeclarations(t)
	for name, declaration := range dependencies {
		declarations[name] = declaration
	}
	if err := validateSecretBodyNativeDeclarations(pkg, declarations, plan); err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(pinnedSubnetBodyIdentitySource, "dec.UseNumber()", "", 1)
	if changed == pinnedSubnetBodyIdentitySource {
		t.Fatal("native number-policy mutation did not apply")
	}
	if err := os.WriteFile(file, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	dependencies, err = g.identityPaginationDeclarations(pkg.Path())
	if err != nil {
		t.Fatal(err)
	}
	for name, declaration := range dependencies {
		declarations[name] = declaration
	}
	if err := validateSecretBodyNativeDeclarations(pkg, declarations, plan); err == nil || !strings.Contains(err.Error(), "pagination.PageResultFrom changed") {
		t.Fatal("native loss of exact JSON numbers passed the source guard", err)
	}
}

const secretFilterNativeFixtureSource = "package secrets\n" + `
import "context"
import "time"
import gophercloud "github.com/gophercloud/gophercloud/v2"
import "github.com/gophercloud/gophercloud/v2/pagination"
type Secret struct{BitLength int ` + "`json:\"bit_length\"`" + `;Algorithm string ` + "`json:\"algorithm\"`" + `;Expiration time.Time ` + "`json:\"-\"`" + `;ContentTypes map[string]string ` + "`json:\"content_types\"`" + `;Created time.Time ` + "`json:\"-\"`" + `;CreatorID string ` + "`json:\"creator_id\"`" + `;Mode string ` + "`json:\"mode\"`" + `;Name string ` + "`json:\"name\"`" + `;SecretRef string ` + "`json:\"secret_ref\"`" + `;SecretType string ` + "`json:\"secret_type\"`" + `;Status string ` + "`json:\"status\"`" + `;Updated time.Time ` + "`json:\"-\"`" + `}
func(*Secret)UnmarshalJSON([]byte)error{return nil}
type SecretType string
type DateFilter string
type DateQuery struct{Date time.Time;Filter DateFilter}
type ListOpts struct{Offset int ` + "`q:\"offset\"`" + `;Limit int ` + "`q:\"limit\"`" + `;Name string ` + "`q:\"name\"`" + `;Alg string ` + "`q:\"alg\"`" + `;Mode string ` + "`q:\"mode\"`" + `;Bits int ` + "`q:\"bits\"`" + `;SecretType SecretType ` + "`q:\"secret_type\"`" + `;ACLOnly *bool ` + "`q:\"acl_only\"`" + `;CreatedQuery *DateQuery;UpdatedQuery *DateQuery;ExpirationQuery *DateQuery;Sort string ` + "`q:\"sort\"`" + `}
type ListOptsBuilder interface{ToSecretListQuery()(string,error)}
func(ListOpts)ToSecretListQuery()(string,error){return "",nil}
type GetResult struct{}
func(GetResult)Extract()(*Secret,error){return nil,nil}
func Get(context.Context,*gophercloud.ServiceClient,string)GetResult{return GetResult{}}
func Delete(context.Context,*gophercloud.ServiceClient,string)error{return nil}
type SecretPage struct{pagination.LinkedPageBase}
func(SecretPage)IsEmpty()(bool,error){return false,nil}
func(SecretPage)NextPageURL()(string,error){return "",nil}
func List(*gophercloud.ServiceClient,ListOptsBuilder)pagination.Pager{_ = SecretPage{};return pagination.Pager{}}
func ExtractSecrets(p pagination.Page)([]Secret,error){_ = p.(SecretPage);return nil,nil}
`

const pinnedSecretBodySource = `package secrets
func (opts ListOpts) ToSecretListQuery() (string, error) {
 q, err := gophercloud.BuildQueryString(opts)
 params := q.Query()
 if opts.CreatedQuery != nil {
  created := opts.CreatedQuery.Date.Format(time.RFC3339)
  if v := opts.CreatedQuery.Filter; v != "" { created = fmt.Sprintf("%s:%s", v, created) }
  params.Add("created", created)
 }
 if opts.UpdatedQuery != nil {
  updated := opts.UpdatedQuery.Date.Format(time.RFC3339)
  if v := opts.UpdatedQuery.Filter; v != "" { updated = fmt.Sprintf("%s:%s", v, updated) }
  params.Add("updated", updated)
 }
 if opts.ExpirationQuery != nil {
  expiration := opts.ExpirationQuery.Date.Format(time.RFC3339)
  if v := opts.ExpirationQuery.Filter; v != "" { expiration = fmt.Sprintf("%s:%s", v, expiration) }
  params.Add("expiration", expiration)
 }
 q = &url.URL{RawQuery: params.Encode()}
 return q.String(), err
}
func List(client *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
 url := listURL(client)
 if opts != nil {
  query, err := opts.ToSecretListQuery()
  if err != nil { return pagination.Pager{Err: err} }
  url += query
 }
 return pagination.NewPager(client, url, func(r pagination.PageResult) pagination.Page {
  return SecretPage{pagination.LinkedPageBase{PageResult: r}}
 })
}
func Get(ctx context.Context, client *gophercloud.ServiceClient, id string) (r GetResult) {
 resp, err := client.Get(ctx, getURL(client, id), &r.Body, nil)
 _, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
 return
}
func (r *Secret) UnmarshalJSON(b []byte) error {
 type tmp Secret
 var s struct {
  tmp
  Created gophercloud.JSONRFC3339NoZ ` + "`json:\"created\"`" + `
  Updated gophercloud.JSONRFC3339NoZ ` + "`json:\"updated\"`" + `
  Expiration gophercloud.JSONRFC3339NoZ ` + "`json:\"expiration\"`" + `
 }
 err := json.Unmarshal(b, &s)
 if err != nil { return err }
 *r = Secret(s.tmp)
 r.Created = time.Time(s.Created)
 r.Updated = time.Time(s.Updated)
 r.Expiration = time.Time(s.Expiration)
 return nil
}
func (r commonResult) Extract() (*Secret, error) {
 var s *Secret
 err := r.ExtractInto(&s)
 return s, err
}
func (r SecretPage) IsEmpty() (bool, error) {
 if r.StatusCode == 204 { return true, nil }
 secrets, err := ExtractSecrets(r)
 return len(secrets) == 0, err
}
func (r SecretPage) NextPageURL() (string, error) {
 var s struct {
  Next string ` + "`json:\"next\"`" + `
  Previous string ` + "`json:\"previous\"`" + `
 }
 err := r.ExtractInto(&s)
 if err != nil { return "", err }
 return s.Next, err
}
func ExtractSecrets(r pagination.Page) ([]Secret, error) {
 var s struct { Secrets []Secret ` + "`json:\"secrets\"`" + ` }
 err := (r.(SecretPage)).ExtractInto(&s)
 return s.Secrets, err
}
func listURL(client *gophercloud.ServiceClient) string {return client.ServiceURL("secrets")}
func getURL(client *gophercloud.ServiceClient, id string) string {return client.ServiceURL("secrets",id)}
`
