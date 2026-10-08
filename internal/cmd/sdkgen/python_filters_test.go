package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func pythonFilterFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for path, source := range pythonFilterFixtureSources {
		target := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func checkedPythonFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../..", subnetFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func clonePythonFilterManifest(t *testing.T, manifest *pythonFilterManifest) *pythonFilterManifest {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	copy, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return copy
}

func TestPythonFilterExtractionIndependentMROQueryAndBody(t *testing.T) {
	root := pythonFilterFixture(t)
	manifest, err := extractPythonFilterManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	// Both fixtures contain top-level raises and unavailable imports. Extraction
	// must only parse them; no OpenStack package or source code may be executed.
	expected := checkedPythonFilterManifest(t)
	if !reflect.DeepEqual(manifest.Query, expected.Query) || !reflect.DeepEqual(manifest.Body, expected.Body) || !reflect.DeepEqual(manifest.Reserved, expected.Reserved) || !reflect.DeepEqual(manifest.MRO, expected.MRO) {
		t.Fatal("independent fixture classification differs", manifest.Query, manifest.Body, manifest.Reserved, manifest.MRO)
	}
	if manifest.Counts.CanonicalQuery != 24 || manifest.Counts.AcceptedQuery != 30 || manifest.Counts.LocalBody != 9 || len(manifest.URI) != 0 || len(manifest.QueryFormats) != 0 || len(manifest.Proof.Nodes) != 15 || len(manifest.Proof.Files) != 7 {
		t.Fatal("partial source contract", manifest)
	}
	if manifest.Body["prefix_length"].ResponseType != nil || *manifest.Body["revision_number"].ResponseType != "int" || manifest.Body["prefix_length"].Field != "prefixlen" {
		t.Fatal("untyped prefix or inherited revision was coerced", manifest.Body)
	}
	for _, notBody := range []string{"id", "name", "tags", "is_dhcp_enabled", "project_id", "prefixlen"} {
		if _, exists := manifest.Body[notBody]; exists {
			t.Fatal("query/remote name incorrectly became a local Body attr", notBody)
		}
	}
	if manifest.UnknownFilters != "discard" || manifest.QueryCollision != "canonical_client_name_wins" {
		t.Fatal("source query policies not extracted", manifest)
	}
}

func TestPythonFilterExtractionDetectsInheritedAndImplementationDrift(t *testing.T) {
	cases := []struct {
		name, path, from, to string
		invalid              bool
		check                func(*testing.T, *pythonFilterManifest)
	}{
		{"tag-alias", "openstack/common/tag.py", "'tags-any'", "'any-tag'", false, func(t *testing.T, m *pythonFilterManifest) {
			if m.Query["any_tags"] != "any-tag" {
				t.Fatal(m.Query)
			}
		}},
		{"inherited-body", "openstack/network/v2/_base.py", "revision_number =", "extra = resource.Body('extra')\n    revision_number =", false, func(t *testing.T, m *pythonFilterManifest) {
			if m.Body["extra"].Field != "extra" || m.Counts.LocalBody != 10 {
				t.Fatal(m.Body)
			}
		}},
		{"pagination-default", "openstack/resource.py", "'limit': 'limit'", "'page_size': 'page_size'", false, func(t *testing.T, m *pythonFilterManifest) {
			if m.Query["page_size"] != "page_size" || m.Query["limit"] != "" {
				t.Fatal(m.Query)
			}
		}},
		{"untyped-prefix", "openstack/network/v2/subnet.py", "Body('prefixlen')", "Body('prefixlen', type=int)", false, func(t *testing.T, m *pythonFilterManifest) {
			if m.Body["prefix_length"].ResponseType == nil || *m.Body["prefix_length"].ResponseType != "int" {
				t.Fatal(m.Body)
			}
		}},
		{"query-format", "openstack/network/v2/subnet.py", "is_dhcp_enabled='enable_dhcp'", "is_dhcp_enabled={'name': 'enable_dhcp', 'format': 'csv'}", false, func(t *testing.T, m *pythonFilterManifest) {
			if m.QueryFormats["is_dhcp_enabled"] != "csv" {
				t.Fatal(m.QueryFormats)
			}
		}},
		{"class-base", "openstack/network/v2/subnet.py", "_base.NetworkResource,", "_base.UnknownResource,", true, nil},
		{"unknown-policy", "openstack/resource.py", "allow_unknown_params=True", "allow_unknown_params=False", true, nil},
		{"canonical-precedence", "openstack/resource.py", "if client_side in query:", "if name in query:", true, nil},
		{"query-expression", "openstack/network/v2/subnet.py", "**_base.TagMixinNetwork._tag_query_parameters", "**make_query_mapping()", true, nil},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root := pythonFilterFixture(t)
			before, err := extractPythonFilterManifest(root)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, test.path)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), test.from) {
				t.Fatal("fixture drift target missing")
			}
			if err := os.WriteFile(path, []byte(strings.Replace(string(data), test.from, test.to, 1)), 0644); err != nil {
				t.Fatal(err)
			}
			after, err := extractPythonFilterManifest(root)
			if test.invalid {
				if err == nil {
					t.Fatal("unsupported source implementation was guessed", after)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			test.check(t, after)
			if before.Proof.Files[test.path] == after.Proof.Files[test.path] || comparePythonFilterManifests(before, after) == nil {
				t.Fatal("source mutation retained the previous proof")
			}
		})
	}
}

func TestPythonFilterManifestCannotReplaceLiveSourceProof(t *testing.T) {
	manifest := checkedPythonFilterManifest(t)
	if err := verifyPythonFilterManifest("", manifest); err == nil || !strings.Contains(err.Error(), "-openstacksdk-source") {
		t.Fatal("missing live source accepted", err)
	}
	if err := verifyPythonFilterManifest(t.TempDir(), manifest); err == nil || !strings.Contains(err.Error(), "Python filter source") {
		t.Fatal("missing source files accepted", err)
	}
	for _, mutate := range []func(*pythonFilterManifest){
		func(m *pythonFilterManifest) { m.SourcePin = "other" },
		func(m *pythonFilterManifest) { m.Proof.Files["openstack/resource.py"] = "changed" },
		func(m *pythonFilterManifest) { m.Proof.Files["../outside.py"] = "changed" },
	} {
		copy := clonePythonFilterManifest(t, manifest)
		mutate(copy)
		if err := verifyPythonFilterManifest(t.TempDir(), copy); err == nil {
			t.Fatal("altered source identity/proof accepted")
		}
	}
	fixture, err := extractPythonFilterManifest(pythonFilterFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*pythonFilterManifest)
	}{
		{"query-alias", func(m *pythonFilterManifest) { m.Query["is_dhcp_enabled"] = "dhcp" }},
		{"body-remote-name", func(m *pythonFilterManifest) {
			m.Body["prefixlen"] = m.Body["prefix_length"]
			delete(m.Body, "prefix_length")
		}},
		{"body-type", func(m *pythonFilterManifest) {
			f := m.Body["prefix_length"]
			v := "int"
			f.ResponseType = &v
			m.Body["prefix_length"] = f
		}},
		{"reserved", func(m *pythonFilterManifest) { m.Reserved = m.Reserved[:8] }},
		{"mro", func(m *pythonFilterManifest) { m.MRO = m.MRO[1:] }},
		{"ast-proof", func(m *pythonFilterManifest) { m.Proof.Nodes[0].ASTSHA256 = "changed" }},
		{"ast-parser", func(m *pythonFilterManifest) { m.Proof.PythonParser = "old" }},
		{"query-formats", func(m *pythonFilterManifest) { m.QueryFormats["fields"] = "csv" }},
		{"unknown-policy", func(m *pythonFilterManifest) { m.UnknownFilters = "wire" }},
		{"canonical-precedence", func(m *pythonFilterManifest) { m.QueryCollision = "wire_wins" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, fixture)
			test.mutate(copy)
			if comparePythonFilterManifests(copy, fixture) == nil {
				t.Fatal("altered independent manifest accepted")
			}
		})
	}
	for _, data := range []string{`{} {}`, `{"unknown_policy":true}`} {
		if _, err := decodePythonFilterManifest([]byte(data)); err == nil {
			t.Fatal("unrecognized manifest JSON accepted", data)
		}
	}
}

func TestPythonFilterDescriptorSelectiveEmissionAndInventory(t *testing.T) {
	manifest := checkedPythonFilterManifest(t)
	for _, spec := range identityCollectionSpecs {
		pkg, plan := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
		g := generator{pythonFilters: manifest}
		owned := g.pythonFilterFor(pkg, plan)
		if (owned != nil) != (spec.path == "network/v2/subnets") {
			t.Fatal("semantic descriptor leaked", spec.path)
		}
		if spec.path == "network/v2/subnets" {
			if err := (&generator{}).validatePythonFilterPlan(pkg, plan); err == nil {
				t.Fatal("unverified production plan accepted")
			}
			if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
				t.Fatal(err)
			}
			altered := *plan
			altered.listQueryBuilder = false
			if g.pythonFilterFor(pkg, &altered) != nil || g.validatePythonFilterPlan(pkg, &altered) == nil {
				t.Fatal("semantic descriptor bypassed the existing native raw Body contract")
			}
		}
		e := emitter{pkg: pkg, imports: map[string]string{}, pythonFilters: owned}
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
		var descriptor *ast.UnaryExpr
		ast.Inspect(file, func(n ast.Node) bool {
			if entry, ok := n.(*ast.KeyValueExpr); ok {
				if key, ok := entry.Key.(*ast.Ident); ok && key.Name == "FilterDescriptor" {
					descriptor = entry.Value.(*ast.UnaryExpr)
				}
			}
			return true
		})
		if (descriptor != nil) != (owned != nil) {
			t.Fatal("missing/extra emitted SDK descriptor", spec.path, string(source))
		}
		if owned != nil {
			literal := descriptor.X.(*ast.CompositeLit)
			counts := map[string]int{}
			for _, entry := range literal.Elts {
				pair := entry.(*ast.KeyValueExpr)
				counts[pair.Key.(*ast.Ident).Name] = len(pair.Value.(*ast.CompositeLit).Elts)
			}
			if counts["Query"] != 24 || counts["Body"] != 9 || counts["Reserved"] != 9 {
				t.Fatal("partial emitted descriptor", counts)
			}
			if !strings.Contains(string(source), `"prefix_length": "prefixlen"`) || strings.Contains(string(source), `"prefixlen": "prefixlen"`+",\n\t\t\t},\n\t\t\tReserved") {
				t.Fatal("semantic Body used remote alias", string(source))
			}
		}
		record := collectionRecord{BodyFilterFields: bodyFilterCollectionFields(pkg, plan, spec.parents), SemanticQueryFilters: pythonFilterQueryFields(owned), SemanticBodyFilters: pythonFilterBodyFields(owned), SemanticReserved: pythonFilterReserved(owned)}
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "semantic_query_filters") != (owned != nil) {
			t.Fatal("unreviewed semantic inventory", string(data))
		}
		if owned != nil {
			if len(record.BodyFilterFields) != 9 || len(record.SemanticBodyFilters) != 9 || record.SemanticBodyFilters["prefixlen"] != "" {
				t.Fatal("explicit and semantic capabilities confused", record)
			}
			record.SemanticQueryFilters["fields"] = "changed"
			record.SemanticReserved[0] = "changed"
			if manifest.Query["fields"] != "fields" || manifest.Reserved[0] == "changed" {
				t.Fatal("inventory aliases verified manifest")
			}
		}
	}
}

func TestPythonFilterPinnedCheckoutVerification(t *testing.T) {
	// This integration check is optional outside the audited source workspace;
	// sdkgen itself never skips its required live-source verification.
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/go-openstacksdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/resource.py")); err != nil {
		t.Skip("audited source checkout not present")
	}
	manifest := checkedPythonFilterManifest(t)
	if err := verifyPythonFilterManifest(source, manifest); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for path := range subnetFilterSourceHashes {
		data, err := os.ReadFile(filepath.Join(source, path))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"openstack/network/v2/subnet.py", "openstack/network/v2/_base.py", "openstack/common/tag.py", "openstack/resource.py", "openstack/fields.py", "openstack/proxy.py", "openstack/network/v2/_proxy.py"} {
		t.Run(path, func(t *testing.T) {
			target := filepath.Join(root, path)
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, append(data, []byte("\n# drift\n")...), 0644); err != nil {
				t.Fatal(err)
			}
			if err := verifyPythonFilterManifest(root, manifest); err == nil || !strings.Contains(err.Error(), path) {
				t.Fatal("stale live source accepted", err)
			}
			if err := os.WriteFile(target, data, 0644); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, mutate := range []func(*pythonFilterManifest){func(m *pythonFilterManifest) { m.Query["fields"] = "wrong" }, func(m *pythonFilterManifest) { m.Proof.Nodes[0].ASTSHA256 = "wrong" }} {
		copy := clonePythonFilterManifest(t, manifest)
		mutate(copy)
		if err := verifyPythonFilterManifest(root, copy); err == nil {
			t.Fatal("mutated manifest passed live extraction")
		}
	}
}

var pythonFilterFixtureSources = map[string]string{
	"openstack/network/v2/subnet.py": `from openstack.network.v2 import _base
from openstack import resource
raise RuntimeError('source must never execute')
class Subnet(_base.NetworkResource, _base.TagMixinNetwork):
    base_path='/subnets'
    resources_key='subnets'
    _query_mapping=resource.QueryParameters('cidr','description','fields','gateway_ip','id','ip_version','ipv6_address_mode','ipv6_ra_mode','name','network_id','segment_id','dns_publish_fixed_ip','project_id','sort_key','sort_dir',is_dhcp_enabled='enable_dhcp',subnet_pool_id='subnetpool_id',use_default_subnet_pool='use_default_subnetpool',**_base.TagMixinNetwork._tag_query_parameters)
    allocation_pools=resource.Body('allocation_pools',type=list)
    created_at=resource.Body('created_at')
    dns_nameservers=resource.Body('dns_nameservers',type=list)
    host_routes=resource.Body('host_routes',type=list)
    prefix_length=resource.Body('prefixlen')
    service_types=resource.Body('service_types',type=list)
    tenant_id=resource.Body('tenant_id',deprecated=True)
    updated_at=resource.Body('updated_at')
    is_dhcp_enabled=resource.Body('enable_dhcp',type=bool)
    project_id=resource.Body('project_id',alias='tenant_id')
    name=resource.Body('name')
`,
	"openstack/network/v2/_base.py": `from openstack import resource
from openstack.common import tag
class NetworkResource(resource.Resource):
    revision_number = resource.Body('revision_number',type=int)
class TagMixinNetwork(tag.TagMixin):
    pass
`,
	"openstack/common/tag.py": `from openstack import resource
class TagMixin(resource.ResourceMixinProtocol):
    _tag_query_parameters: dict = {'tags':'tags','any_tags':'tags-any','not_tags':'not-tags','not_any_tags':'not-tags-any'}
    tags=resource.Body('tags',type=list,default=[])
`,
	"openstack/resource.py": `from typing import Protocol
from openstack import fields
def Body(name,**kwargs):
    return fields.Body(name,**kwargs)
class QueryParameters:
    def __init__(self,*names,include_pagination_defaults=True,**mappings):
        self._mapping={}
        if include_pagination_defaults:
            self._mapping.update({'limit': 'limit','marker':'marker'})
        self._mapping.update({name:name for name in names})
        self._mapping.update(mappings)
    def _validate(self,query,base_path=None,allow_unknown_params=False):
        return query
    def _transpose(self,query,resource_type):
        result={}
        for client_side, name in self._mapping.items():
            if client_side in query:
                value=query[client_side]
            elif name in query:
                value=query[name]
            else:
                continue
            result[name]=value
        return result
class ResourceMixinProtocol(Protocol):
    pass
class Resource(dict):
    id=Body('id')
    name=Body('name')
    _query_mapping=QueryParameters()
    def list(cls,session,paginated=True,base_path=None,allow_unknown_params=False,*,microversion=None,headers=None,max_items=None,**params):
        api_filters=cls._query_mapping._validate(params,base_path=base_path,allow_unknown_params=True)
        return api_filters
`,
	"openstack/fields.py": `def _convert_type(value,data_type):
    return value
class _BaseComponent:
    def __get__(self,instance,owner):
        return None
`,
	"openstack/proxy.py": `class Proxy:
    def _list(self,resource_type,paginated=True,base_path=None,jmespath_filters=None,**attrs):
        return resource_type.list(self,**attrs)
`,
	"openstack/network/v2/_proxy.py": `from unavailable_package import impossible_import
class Proxy:
    def subnets(self,**query):
        return self._list(Subnet,**query)
`,
}
