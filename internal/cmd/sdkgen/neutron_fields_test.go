package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func neutronFieldSourceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"openstack/resource.py": `raise RuntimeError("checkout source must never execute")
from typing import Protocol
class ResourceMixinProtocol(Protocol):
    pass
class Resource(dict):
    id = Body("id")
    name = Body("name")
`,
		"openstack/common/tag.py": `raise RuntimeError("checkout source must never execute")
from openstack import resource
class TagMixin(resource.ResourceMixinProtocol):
    tags = resource.Body("tags")
`,
		"openstack/network/v2/_base.py": `raise RuntimeError("checkout source must never execute")
from openstack import resource
from openstack.common import tag
class NetworkResource(resource.Resource):
    revision_number = resource.Body("revision_number")
class TagMixinNetwork(tag.TagMixin):
    pass
`,
	}
	for _, spec := range neutronFieldSpecs {
		module, class, _ := strings.Cut(strings.TrimPrefix(spec.PythonResource, "openstack.network.v2."), ".")
		source := `raise RuntimeError("checkout source must never execute")
from unavailable_sdk_dependency import unavailable
from openstack import resource
from openstack.network.v2 import _base
class ` + class + `(_base.NetworkResource, _base.TagMixinNetwork):
    is_admin_state_up = resource.Body("admin_state_up")
    any_tags = resource.QueryParameters(any_tags="tags-any")
    if_match = resource.Header("if-match")
`
		if module == "network" {
			source += `    revision_number = resource.Body("custom_revision")
    name = None
`
		}
		files["openstack/network/v2/"+module+".py"] = source
	}
	for path := range neutronFieldSourceHashes {
		source := files[path]
		if source == "" {
			source = "raise RuntimeError('checkout source must never execute')\n"
		}
		target := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource(t *testing.T) {
	root := neutronFieldSourceFixture(t)
	manifest, err := extractNeutronFieldDescriptors(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Resources) != len(neutronFieldSpecs) || len(manifest.Files) != len(neutronFieldSourceHashes) {
		t.Fatal("incomplete extraction", manifest)
	}
	for _, spec := range neutronFieldSpecs {
		want := map[string]string{"id": "id", "name": "name", "tags": "tags", "revision_number": "revision_number", "is_admin_state_up": "admin_state_up"}
		if spec.SDKPath == "network/v2/networks" {
			want["revision_number"] = "custom_revision"
		}
		if !reflect.DeepEqual(manifest.Resources[spec.key()], want) {
			t.Fatalf("%s: got %v want %v", spec.key(), manifest.Resources[spec.key()], want)
		}
	}
	// Parsing a synthetic tree is permitted; it cannot authorize production
	// generation. Fresh source bytes, not the parser's output, grant that.
	if _, err := loadNeutronFieldDescriptors(root); err == nil || !strings.Contains(err.Error(), "SHA mismatch") {
		t.Fatal("unreviewed synthetic Source authorized generation", err)
	}
}

func TestNeutronFieldExtractionRejectsUnknownMROAndNonliteralBodyNames(t *testing.T) {
	for _, change := range []struct{ name, before, after string }{
		{"unknown inherited class", "class Network(_base.NetworkResource, _base.TagMixinNetwork):", "class Network(unavailable):"},
		{"nonliteral wire name", `resource.Body("admin_state_up")`, "resource.Body(unavailable())"},
	} {
		t.Run(change.name, func(t *testing.T) {
			root := neutronFieldSourceFixture(t)
			path := filepath.Join(root, "openstack/network/v2/network.py")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			changed := strings.Replace(string(data), change.before, change.after, 1)
			if changed == string(data) {
				t.Fatal("mutation did not apply")
			}
			if err := os.WriteFile(path, []byte(changed), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := extractNeutronFieldDescriptors(root); err == nil {
				t.Fatal("unreviewed Source shape accepted")
			}
		})
	}
}

const neutronFieldNativeFixture = `package networks
import "context"
import "net/http"
import gophercloud "github.com/gophercloud/gophercloud/v2"
type Network struct{ID string}
type GetResult struct{Body any; Header http.Header; Err error}
func(GetResult)Extract()(*Network,error){return nil,nil}
func Get(ctx context.Context, client *gophercloud.ServiceClient, id string)GetResult{return GetResult{}}
`

func TestNeutronFieldGenerationRejectsNativeResultAndInputDrift(t *testing.T) {
	spec := neutronFieldSpecs[0]
	path := upstreamModule + "/openstack/networking/v2/networks"
	pkg, _ := typedCollectionFixture(t, path, neutronFieldNativeFixture)
	if err := validateNeutronFieldGetter(pkg, spec); err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct{ name, before, after string }{
		{"ID type", "id string", "id int"},
		{"input shape", "id string", "id string, fields []string"},
		{"result shape", ")GetResult{return GetResult{}", ")*GetResult{return &GetResult{}"},
		{"response body", "Body any", "Body map[string]any"},
		{"response header", "Header http.Header", "Header map[string]string"},
		{"typed extraction", "Extract()(*Network,error)", "Extract()(Network,error)"},
		{"extract args", "Extract()", "Extract(id string)"},
	} {
		t.Run(change.name, func(t *testing.T) {
			changed := strings.Replace(neutronFieldNativeFixture, change.before, change.after, 1)
			if changed == neutronFieldNativeFixture {
				t.Fatal("mutation did not apply")
			}
			// A changed return statement must remain type-correct so that this
			// checks sdkgen's contract validation rather than fixture parsing.
			if change.name == "response header" {
				changed = strings.Replace(changed, "import \"net/http\"\n", "", 1)
			}
			if change.name == "typed extraction" {
				changed = strings.Replace(changed, "return nil,nil", "return Network{},nil", 1)
			}
			pkg, _ := typedCollectionFixture(t, path, changed)
			if err := validateNeutronFieldGetter(pkg, spec); err == nil {
				t.Fatal("native Get drift accepted")
			}
		})
	}
}

func TestNeutronFieldGenerationMatchesFreshPinnedSource(t *testing.T) {
	source := os.Getenv("SDKGEN_OPENSTACKSDK_SOURCE")
	if source == "" {
		source = os.Getenv("OPENSTACKSDK_SOURCE")
	}
	if source == "" {
		t.Skip("fresh pinned Source checkout not configured")
	}
	manifest, err := loadNeutronFieldDescriptors(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, specs := range neutronFieldPackages() {
		got, err := renderNeutronFieldGetters(specs, manifest.Resources)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join("../../..", filepath.FromSlash(specs[0].SDKPath), "get_fields_generated.go")
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatal("fresh Source field wrapper differs from checked-in Go", specs[0].SDKPath)
		}
	}
}

const neutronChildFieldNativeFixture = `package bgpvpns
import "context"
import "net/http"
import gophercloud "github.com/gophercloud/gophercloud/v2"
type NetworkAssociation struct{ID string}
type commonNetworkAssociationResult struct{Body any; Header http.Header; Err error}
func(commonNetworkAssociationResult)Extract()(*NetworkAssociation,error){return nil,nil}
type GetNetworkAssociationResult struct{commonNetworkAssociationResult}
func GetNetworkAssociation(ctx context.Context, client *gophercloud.ServiceClient, bgpVpnID string, id string)GetNetworkAssociationResult{return GetNetworkAssociationResult{}}
`

func neutronChildFieldSpec(t *testing.T) neutronFieldSpec {
	t.Helper()
	for _, spec := range neutronFieldSpecs {
		if spec.Getter == "GetNetworkAssociation" {
			return spec
		}
	}
	t.Fatal("network association field spec missing")
	return neutronFieldSpec{}
}

func TestNeutronChildFieldGenerationValidatesNativeParentGetter(t *testing.T) {
	spec := neutronChildFieldSpec(t)
	path := upstreamModule + "/openstack/networking/v2/extensions/bgpvpns"
	pkg, _ := typedCollectionFixture(t, path, neutronChildFieldNativeFixture)
	if err := validateNeutronFieldGetter(pkg, spec); err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct{ name, before, after string }{
		{"missing parent", "bgpVpnID string, id string", "id string"},
		{"parent type", "bgpVpnID string", "bgpVpnID int"},
		{"result type", ")GetNetworkAssociationResult{return GetNetworkAssociationResult{}}", ")commonNetworkAssociationResult{return commonNetworkAssociationResult{}}"},
		{"typed extraction", "Extract()(*NetworkAssociation,error)", "Extract()(NetworkAssociation,error)"},
	} {
		t.Run(change.name, func(t *testing.T) {
			changed := strings.Replace(neutronChildFieldNativeFixture, change.before, change.after, 1)
			if changed == neutronChildFieldNativeFixture {
				t.Fatal("mutation did not apply")
			}
			if change.name == "typed extraction" {
				changed = strings.Replace(changed, "return nil,nil", "return NetworkAssociation{},nil", 1)
			}
			pkg, _ := typedCollectionFixture(t, path, changed)
			if err := validateNeutronFieldGetter(pkg, spec); err == nil {
				t.Fatal("native child getter drift accepted")
			}
		})
	}
}

func TestNeutronFieldRenderingGroupsParentAndChildGetters(t *testing.T) {
	var specs []neutronFieldSpec
	for _, group := range neutronFieldPackages() {
		if group[0].SDKPath == "network/v2/extensions/bgpvpns" {
			specs = group
		}
	}
	if len(specs) != 4 || specs[0].Getter != "" {
		t.Fatal("bgpvpns field getters are not grouped with the top-level getter first", specs)
	}
	manifest := map[string]map[string]string{}
	for _, spec := range specs {
		manifest[spec.key()] = map[string]string{"id": "id", "project_id": "project_id"}
	}
	data, err := renderNeutronFieldGetters(specs, manifest)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{
		"package bgpvpns\n",
		"func (a *API) GetWithFields(ctx context.Context, id string, fields ...string) (*BGPVPN, error) {",
		`[]string{"bgpvpn/bgpvpns", id}, fields, neutronBodyFieldNames, &result.Body)`,
		"func (a *API) GetNetworkAssociationWithFields(ctx context.Context, parentID string, id string, fields ...string) (*NetworkAssociation, error) {",
		"var result GetNetworkAssociationResult\n",
		`[]string{"bgpvpn/bgpvpns", parentID, "network_associations", id}, fields, neutronNetworkAssociationBodyFieldNames, &result.Body)`,
		`request.Wrap("GetNetworkAssociationWithFields", "bgpvpns", err)`,
		`[]string{"bgpvpn/bgpvpns", parentID, "port_associations", id}, fields, neutronPortAssociationBodyFieldNames, &result.Body)`,
		`[]string{"bgpvpn/bgpvpns", parentID, "router_associations", id}, fields, neutronRouterAssociationBodyFieldNames, &result.Body)`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered getters miss %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "package ") != 1 || strings.Count(got, "import (") != 1 {
		t.Fatal("grouped getters must share one file header", got)
	}
	mixed := append([]neutronFieldSpec{neutronFieldSpecs[0]}, specs[1])
	if _, err := renderNeutronFieldGetters(mixed, manifest); err == nil {
		t.Fatal("getters from different packages rendered into one file")
	}
}
