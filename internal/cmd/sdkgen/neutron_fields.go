package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/format"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

// Field selection is based on all Body descriptors in the pinned MRO, not the
// ordinary-list manifest's local Body subset or its independent query aliases.
// The extractor reuses the existing AST-only Source resolver. Neither the SDK
// checkout nor third-party imports are executed.
type neutronFieldSpec struct {
	SDKPath, PythonResource, Model, Route string
	BodyCount                             int
}

var neutronFieldSpecs = []neutronFieldSpec{
	{"network/v2/networks", "openstack.network.v2.network.Network", "Network", "networks", 29},
	{"network/v2/subnets", "openstack.network.v2.subnet.Subnet", "Subnet", "subnets", 25},
	{"network/v2/ports", "openstack.network.v2.port.Port", "Port", "ports", 41},
	{"network/v2/extensions/layer3/floatingips", "openstack.network.v2.floating_ip.FloatingIP", "FloatingIP", "floatingips", 20},
	{"network/v2/extensions/layer3/routers", "openstack.network.v2.router.Router", "Router", "routers", 20},
	{"network/v2/extensions/security/groups", "openstack.network.v2.security_group.SecurityGroup", "SecGroup", "security-groups", 12},
	{"network/v2/extensions/security/rules", "openstack.network.v2.security_group_rule.SecurityGroupRule", "SecGroupRule", "security-group-rules", 18},
	{"network/v2/extensions/subnetpools", "openstack.network.v2.subnet_pool.SubnetPool", "SubnetPool", "subnetpools", 18},
	{"network/v2/extensions/trunks", "openstack.network.v2.trunk.Trunk", "Trunk", "trunks", 10},
	{"network/v2/extensions/qos/policies", "openstack.network.v2.qos_policy.QoSPolicy", "Policy", "qos/policies", 9},
	{"network/v2/extensions/security/addressgroups", "openstack.network.v2.address_group.AddressGroup", "AddressGroup", "address-groups", 6},
}

var neutronFieldSourceHashes = map[string]string{
	"openstack/common/tag.py":                     "c86f1d41fc1ed139366f163e4d5dfb7a37b15380dda05c3382c47dd7bafaa59e",
	"openstack/fields.py":                         "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd",
	"openstack/network/v2/_base.py":               "70981b01c16f24f656291068f808e97a5f9f309cfcda535a03056eccb54acc00",
	"openstack/network/v2/_proxy.py":              "e409ee081f8b9c44903a1b81a4d9f9590c59f9f947aa686f960a5171ab1a6b58",
	"openstack/network/v2/address_group.py":       "ba4bacb24761eb2b9e4dad0ac3f664e65c0ecb231b13813020cc5eac5f6ec54c",
	"openstack/network/v2/floating_ip.py":         "146929276c7f0940c79717a84e43af1a7384accb1fa00d2d6a5f871cedd0754d",
	"openstack/network/v2/network.py":             "ad17c5dadc94a28f35d6f0c95fa95b2a15165b12ab652c1be188ba30aeca7500",
	"openstack/network/v2/port.py":                "634255090e9ee0711a1e51cca99819d490ced82779746c850302215c66b10b25",
	"openstack/network/v2/qos_policy.py":          "d82e98910ae942c725542fbc9ddd59b088fb3f60e3cf4e6631b72e2384e45dd3",
	"openstack/network/v2/router.py":              "fdb6589856a606bbd173f52eb1d549e2028eca9dd6bd9dddd3c76d8efe3ae1f8",
	"openstack/network/v2/security_group.py":      "1fde1184e845682d237ee89235ec87f554232433a6f74e6af362e0e8ba6904dc",
	"openstack/network/v2/security_group_rule.py": "3d1582cc2b76c66ccab473d7d04ac6cff5a2df7a3a0af521f5a284777ad487ba",
	"openstack/network/v2/subnet.py":              "a23a33b8f8baedf74a44fab67827c1f02b8941eed9eae44d0beb5050166b6c00",
	"openstack/network/v2/subnet_pool.py":         "4028662600878ae10d80f64d97faafa48e3c0b4e603be48530f6488517a74fd0",
	"openstack/network/v2/trunk.py":               "0cfb0b32805ae8b5f70f97e06ac64240af69296c15b8c06a82741badad2d2028",
	"openstack/proxy.py":                          "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46",
	"openstack/resource.py":                       "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0",
}

type neutronFieldManifest struct {
	Files     map[string]string            `json:"files"`
	Resources map[string]map[string]string `json:"resources"`
}

const neutronFieldExtractor = `
params = json.loads(sys.argv[2])
source = Source(pathlib.Path(sys.argv[1]), params["files"])
resources = {}
for sdk_path, resource_name in params["targets"].items():
    fields = {}
    # _get_with_fields considers each class's Body objects directly. A
    # derived non-Body member does not mask an inherited Body of that name.
    # The first Body with a given Python name in C3 MRO wins.
    for cls in source.mro(resource_name):
        for name, (module, value, node) in source.attributes.get(cls, {}).items():
            if not isinstance(value, ast.Call):
                continue
            if source.resolve(module, value.func) not in {
                "openstack.resource.Body", "openstack.fields.Body"
            }:
                continue
            if name in fields:
                continue
            if not value.args:
                raise ValueError("Body field has no wire name: " + cls + "." + name)
            wire = source.literal(module, value.args[0])
            if not isinstance(wire, str):
                raise ValueError("Body wire name is not a string: " + cls + "." + name)
            fields[name] = wire
    resources[sdk_path] = fields
print(json.dumps({
    "files": {path: hashlib.sha256(raw).hexdigest() for path, raw in source.raw.items()},
    "resources": resources,
}, sort_keys=True))
`

func extractNeutronFieldDescriptors(source string) (*neutronFieldManifest, error) {
	files := make([]string, 0, len(neutronFieldSourceHashes))
	for path := range neutronFieldSourceHashes {
		files = append(files, path)
	}
	sort.Strings(files)
	targets := make(map[string]string, len(neutronFieldSpecs))
	for _, spec := range neutronFieldSpecs {
		targets[spec.SDKPath] = spec.PythonResource
	}
	parameters, err := json.Marshal(struct {
		Files   []string          `json:"files"`
		Targets map[string]string `json:"targets"`
	}{files, targets})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "python3", "-I", "-", source, string(parameters))
	// Prevent the shared extractor's command entry point, then use its Source
	// resolver without importing or executing any file from the checkout.
	command.Stdin = strings.NewReader("__name__ = 'neutron_fields_ast'\n" + pythonFilterExtractor + "\n" + neutronFieldExtractor)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	data, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("Neutron field extraction: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var result neutronFieldManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("Neutron field descriptors: %w", err)
	}
	return &result, nil
}

func loadNeutronFieldDescriptors(source string) (*neutronFieldManifest, error) {
	if source == "" {
		return nil, fmt.Errorf("Neutron field selection requires the pinned OpenStackSDK source checkout")
	}
	for path, expected := range neutronFieldSourceHashes {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			return nil, fmt.Errorf("Neutron field source %s: %w", path, err)
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != expected {
			return nil, fmt.Errorf("Neutron field source SHA mismatch: %s", path)
		}
	}
	manifest, err := extractNeutronFieldDescriptors(source)
	if err != nil {
		return nil, err
	}
	// Check the bytes actually parsed too: the preliminary source audit alone
	// cannot authorize descriptors extracted from subsequently changed files.
	if !reflect.DeepEqual(manifest.Files, neutronFieldSourceHashes) {
		return nil, fmt.Errorf("Neutron field parsed source SHA mismatch")
	}
	if len(manifest.Resources) != len(neutronFieldSpecs) {
		return nil, fmt.Errorf("Neutron field extraction has an incomplete resource set")
	}
	for _, spec := range neutronFieldSpecs {
		if len(manifest.Resources[spec.SDKPath]) != spec.BodyCount {
			return nil, fmt.Errorf("Neutron field extraction has incomplete %s Body descriptors", spec.PythonResource)
		}
	}
	return manifest, nil
}

func validateNeutronFieldGetter(pkg *types.Package, spec neutronFieldSpec) error {
	qualified := func(p *types.Package) string { return p.Path() }
	get, ok := pkg.Scope().Lookup("Get").(*types.Func)
	if !ok {
		return fmt.Errorf("Neutron field selection requires %s.Get", pkg.Path())
	}
	signature, ok := get.Type().(*types.Signature)
	if !ok || signature.Variadic() || signature.Params().Len() != 3 || signature.Results().Len() != 1 {
		return fmt.Errorf("Neutron field selection requires the native Get signature: %s", pkg.Path())
	}
	for i, expected := range []string{"context.Context", "*" + upstreamModule + ".ServiceClient", "string"} {
		if types.TypeString(signature.Params().At(i).Type(), qualified) != expected {
			return fmt.Errorf("Neutron field selection requires native Get parameter %d: %s", i, pkg.Path())
		}
	}
	result := pkg.Scope().Lookup("GetResult")
	model := pkg.Scope().Lookup(spec.Model)
	if result == nil || model == nil || !types.Identical(signature.Results().At(0).Type(), result.Type()) {
		return fmt.Errorf("Neutron field selection requires native GetResult/model: %s", pkg.Path())
	}
	for name, expected := range map[string]string{"Body": "any", "Header": "net/http.Header", "Err": "error"} {
		member, _, _ := types.LookupFieldOrMethod(types.NewPointer(result.Type()), true, pkg, name)
		if member == nil {
			return fmt.Errorf("Neutron field selection requires native GetResult.%s: %s", name, pkg.Path())
		}
		if name == "Body" {
			if !types.Identical(member.Type(), types.Universe.Lookup("any").Type()) {
				return fmt.Errorf("Neutron field selection requires native GetResult.Body any: %s", pkg.Path())
			}
		} else if types.TypeString(member.Type(), qualified) != expected {
			return fmt.Errorf("Neutron field selection requires native GetResult.%s %s: %s", name, expected, pkg.Path())
		}
	}
	method := types.NewMethodSet(types.NewPointer(result.Type())).Lookup(pkg, "Extract")
	if method == nil {
		return fmt.Errorf("Neutron field selection requires native GetResult.Extract: %s", pkg.Path())
	}
	extract, ok := method.Obj().Type().(*types.Signature)
	if !ok || extract.Params().Len() != 0 || extract.Results().Len() != 2 || !types.Identical(extract.Results().At(0).Type(), types.NewPointer(model.Type())) || !types.Identical(extract.Results().At(1).Type(), types.Universe.Lookup("error").Type()) {
		return fmt.Errorf("Neutron field selection requires native typed Extract: %s", pkg.Path())
	}
	return nil
}

func renderNeutronFieldGetter(spec neutronFieldSpec, names map[string]string) ([]byte, error) {
	var out bytes.Buffer
	fmt.Fprintf(&out, "// Code generated by sdkgen from Gophercloud v2.15.0 and OpenStackSDK %s; DO NOT EDIT.\npackage %s\n\n", pythonFilterPin, filepath.Base(spec.SDKPath))
	out.WriteString("import (\n context \"context\"\n request \"github.com/JSYoo5B/go-openstacksdk/request\"\n)\n\n")
	out.WriteString("// Canonical Body names from the freshly verified Python Resource MRO.\nvar neutronBodyFieldNames = map[string]string{\n")
	keys := make([]string, 0, len(names))
	for key := range names {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&out, "%q: %q,\n", key, names[key])
	}
	out.WriteString("}\n\n")
	out.WriteString("// GetWithFields fetches selected fields using canonical Python Body aliases.\n// No fields requests the ordinary representation. Unknown field names and\n// repeated fields are preserved; list-query aliases are not applied.\n// It uses native result extraction with successful HTTP statuses 200..399.\n")
	fmt.Fprintf(&out, "func (a *API) GetWithFields(ctx context.Context, id string, fields ...string) (*%s, error) {\n", spec.Model)
	out.WriteString("var result GetResult\n")
	fmt.Fprintf(&out, "result.Header, result.Err = request.NeutronFieldsGet(ctx, a.RawClient(), []string{%q, id}, fields, neutronBodyFieldNames, &result.Body)\n", spec.Route)
	if spec.SDKPath == subnetPoolSDKPath {
		out.WriteString("value, err := extractSelectedSubnetPool(result)\n")
	} else {
		out.WriteString("value, err := result.Extract()\n")
	}
	fmt.Fprintf(&out, "return value, request.Wrap(\"GetWithFields\", %q, err)\n}\n", filepath.Base(spec.SDKPath))
	return format.Source(out.Bytes())
}

// Called after the main native generation loop, once g.importer is configured.
func (g *generator) generateNeutronFieldGetters(source string) error {
	manifest, err := loadNeutronFieldDescriptors(source)
	if err != nil {
		return err
	}
	// Validate every native type before writing any wrapper from this bundle.
	for _, spec := range neutronFieldSpecs {
		path := upstreamModule + "/openstack/networking/v2/" + strings.TrimPrefix(spec.SDKPath, "network/v2/")
		pkg, err := g.importer.Import(path)
		if err != nil {
			return err
		}
		if err := validateNeutronFieldGetter(pkg, spec); err != nil {
			return err
		}
	}
	for _, spec := range neutronFieldSpecs {
		data, err := renderNeutronFieldGetter(spec, manifest.Resources[spec.SDKPath])
		if err != nil {
			return err
		}
		target := filepath.Join(g.root, filepath.FromSlash(spec.SDKPath), "get_fields_generated.go")
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
	}
	return nil
}
