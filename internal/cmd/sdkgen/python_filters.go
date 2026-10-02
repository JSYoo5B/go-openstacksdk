package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

//go:embed python_filter_manifest.py
var pythonFilterExtractor string

const pythonFilterPin = "ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe"
const subnetFilterManifestPath = "api/openstacksdk/resources/network/v2/subnet.json"

// Source hashes are independent of Python's AST representation. Changing a
// reviewed implementation requires another source audit, not just regenerating
// the descriptor from whatever checkout happens to be passed to sdkgen.
var subnetFilterSourceHashes = map[string]string{
	"openstack/network/v2/subnet.py": "a23a33b8f8baedf74a44fab67827c1f02b8941eed9eae44d0beb5050166b6c00",
	"openstack/network/v2/_base.py":  "70981b01c16f24f656291068f808e97a5f9f309cfcda535a03056eccb54acc00",
	"openstack/common/tag.py":        "c86f1d41fc1ed139366f163e4d5dfb7a37b15380dda05c3382c47dd7bafaa59e",
	"openstack/resource.py":          "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0",
	"openstack/fields.py":            "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd",
	"openstack/proxy.py":             "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46",
	"openstack/network/v2/_proxy.py": "e409ee081f8b9c44903a1b81a4d9f9590c59f9f947aa686f960a5171ab1a6b58",
}

type pythonFilterField struct {
	Field            string  `json:"field"`
	ResponseType     *string `json:"response_type"`
	ResponseAccessor string  `json:"response_accessor,omitempty"`
	Formatter        string  `json:"formatter,omitempty"`
}

type pythonFilterAnchor struct {
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Line      int    `json:"line"`
	EndLine   int    `json:"end_line"`
	ASTSHA256 string `json:"ast_sha256"`
}

type pythonFilterManifest struct {
	SchemaVersion  int                          `json:"schema_version"`
	SourcePin      string                       `json:"source_pin"`
	Resource       string                       `json:"resource"`
	SDKPackage     string                       `json:"sdk_package"`
	BasePath       string                       `json:"base_path"`
	Envelope       string                       `json:"envelope"`
	ClassBases     []string                     `json:"class_bases"`
	MRO            []string                     `json:"mro"`
	Query          map[string]string            `json:"query"`
	QueryFormats   map[string]string            `json:"query_formats"`
	Body           map[string]pythonFilterField `json:"body"`
	URI            map[string]pythonFilterField `json:"uri"`
	UnknownFilters string                       `json:"unknown_filters"`
	QueryCollision string                       `json:"query_collision"`
	Counts         struct {
		CanonicalQuery int `json:"canonical_query"`
		AcceptedQuery  int `json:"accepted_query"`
		LocalBody      int `json:"local_body"`
	} `json:"counts"`
	SourceControls struct {
		ResourceList []string `json:"resource_list"`
		ProxyList    []string `json:"proxy_list"`
	} `json:"source_controls"`
	Reserved []string `json:"reserved"`
	Proof    struct {
		ASTAlgorithm string               `json:"ast_algorithm"`
		PythonParser string               `json:"python_parser"`
		Nodes        []pythonFilterAnchor `json:"nodes"`
		Files        map[string]string    `json:"files"`
	} `json:"proof"`
}

func decodePythonFilterManifest(data []byte) (*pythonFilterManifest, error) {
	var manifest pythonFilterManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("Python filter manifest: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("Python filter manifest: unexpected trailing JSON")
	}
	return &manifest, nil
}

func extractPythonFilterManifest(source string) (*pythonFilterManifest, error) {
	return extractPythonFilterManifestTarget(source, "")
}

func extractPythonFilterManifestTarget(source, resource string) (*pythonFilterManifest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	arguments := []string{"-I", "-", "--source", source}
	if resource != "" {
		arguments = append(arguments, "--resource", resource)
	}
	command := exec.CommandContext(ctx, "python3", arguments...)
	command.Stdin = strings.NewReader(pythonFilterExtractor)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	data, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("Python filter extraction: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return decodePythonFilterManifest(data)
}

// Verification always reads the live source checkout. A valid checked-in JSON
// file alone cannot grant a generated binding semantic filter capabilities.
func verifyPythonFilterManifest(source string, manifest *pythonFilterManifest) error {
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("-openstacksdk-source is required for audited Subnet semantic filters")
	}
	if manifest == nil || manifest.SchemaVersion != 1 || manifest.SourcePin != pythonFilterPin || manifest.Resource != "openstack.network.v2.subnet.Subnet" || manifest.SDKPackage != "gophercloudsdk/network/v2/subnets" || manifest.BasePath != "/subnets" || manifest.Envelope != "subnets" {
		return fmt.Errorf("audited Subnet Python filter identity changed")
	}
	if !reflect.DeepEqual(manifest.Proof.Files, subnetFilterSourceHashes) {
		return fmt.Errorf("audited Subnet Python source proof changed")
	}
	for path, expected := range subnetFilterSourceHashes {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			return fmt.Errorf("Python filter source %s: %w", path, err)
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != expected {
			return fmt.Errorf("audited Python filter source %s changed", path)
		}
	}
	fresh, err := extractPythonFilterManifest(source)
	if err != nil {
		return err
	}
	return comparePythonFilterManifests(manifest, fresh)
}

func comparePythonFilterManifests(manifest, fresh *pythonFilterManifest) error {
	if manifest.Proof.PythonParser != fresh.Proof.PythonParser {
		return fmt.Errorf("Python filter AST parser version %s differs from audited %s", fresh.Proof.PythonParser, manifest.Proof.PythonParser)
	}
	if !reflect.DeepEqual(manifest, fresh) {
		return fmt.Errorf("Subnet Python filter manifest differs from independent live-source extraction")
	}
	return nil
}

func loadPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	if strings.TrimSpace(source) == "" {
		return nil, fmt.Errorf("-openstacksdk-source is required for audited Subnet semantic filters")
	}
	data, err := os.ReadFile(filepath.Join(root, subnetFilterManifestPath))
	if err != nil {
		return nil, fmt.Errorf("Python filter manifest: %w", err)
	}
	manifest, err := decodePythonFilterManifest(data)
	if err != nil {
		return nil, err
	}
	if err := verifyPythonFilterManifest(source, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (g *generator) validatePythonFilterPlan(pkg *types.Package, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) == securityGroupSDKPath {
		if !securityGroupPythonFilterMetadataValid(g.securityGroupPythonFilters) {
			return fmt.Errorf("audited SecurityGroup semantic filter source proof was not verified")
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
			return fmt.Errorf("audited SecurityGroup semantic filters require the full native raw Body contract")
		}
		return nil
	}
	if sdkPath(pkg.Path()) == routerSDKPath {
		if !routerPythonFilterMetadataValid(g.routerPythonFilters) {
			return fmt.Errorf("audited Router semantic filter source proof was not verified")
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
			return fmt.Errorf("audited Router semantic filters require the full native raw Body contract")
		}
		return nil
	}
	if sdkPath(pkg.Path()) == networkSDKPath {
		if !networkPythonFilterMetadataValid(g.networkPythonFilters) {
			return fmt.Errorf("audited Network semantic filter source proof was not verified")
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
			return fmt.Errorf("audited Network semantic filters require the full native raw Body contract")
		}
		return nil
	}
	if sdkPath(pkg.Path()) == subnetPoolSDKPath {
		if !subnetPoolPythonFilterMetadataValid(g.subnetPoolPythonFilters) {
			return fmt.Errorf("audited SubnetPool semantic filter source proof was not verified")
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
			return fmt.Errorf("audited SubnetPool semantic filters require the full native raw Body contract")
		}
		return nil
	}
	if sdkPath(pkg.Path()) == qosPolicySDKPath {
		if !qosPolicyPythonFilterMetadataValid(g.qosPolicyPythonFilters) {
			return fmt.Errorf("audited QoSPolicy semantic filter source proof was not verified")
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
			return fmt.Errorf("audited QoSPolicy semantic filters require the full native raw Body contract")
		}
		return nil
	}
	if sdkPath(pkg.Path()) == addressGroupSDKPath {
		if !addressGroupPythonFilterMetadataValid(g.addressGroupPythonFilters) {
			return fmt.Errorf("audited AddressGroup semantic filter source proof was not verified")
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
			return fmt.Errorf("audited AddressGroup semantic filters require the full native raw Body contract")
		}
		return nil
	}
	if sdkPath(pkg.Path()) == "keymanager/v1/orders" {
		if !orderPythonFilterMetadataValid(g.orderPythonFilters) {
			return fmt.Errorf("audited Order semantic filter source proof was not verified")
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
			return fmt.Errorf("audited Order semantic filters require the full native raw Body contract")
		}
		return nil
	}
	if sdkPath(pkg.Path()) == "keymanager/v1/containers" {
		if !containerPythonFilterMetadataValid(g.containerPythonFilters) {
			return fmt.Errorf("audited Container semantic filter source proof was not verified")
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
			return fmt.Errorf("audited Container semantic filters require the full native raw Body contract")
		}
		return nil
	}
	if sdkPath(pkg.Path()) == "keymanager/v1/secrets" {
		if !secretPythonFilterMetadataValid(g.secretPythonFilters) {
			return fmt.Errorf("audited Secret semantic filter source proof was not verified")
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
			return fmt.Errorf("audited Secret semantic filters require the full native raw Body contract")
		}
		return nil
	}
	if sdkPath(pkg.Path()) != "network/v2/subnets" {
		return nil
	}
	if g.pythonFilters == nil {
		return fmt.Errorf("audited Subnet semantic filter source proof was not verified")
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited Subnet semantic filters require the full native raw Body contract")
	}
	return nil
}

func (g *generator) pythonFilterFor(pkg *types.Package, plan *collectionPlan) *pythonFilterManifest {
	if sdkPath(pkg.Path()) == securityGroupSDKPath {
		if !securityGroupPythonFilterMetadataValid(g.securityGroupPythonFilters) {
			return nil
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); ok {
			return g.securityGroupPythonFilters
		}
		return nil
	}
	if sdkPath(pkg.Path()) == routerSDKPath {
		if !routerPythonFilterMetadataValid(g.routerPythonFilters) {
			return nil
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); ok {
			return g.routerPythonFilters
		}
		return nil
	}
	if sdkPath(pkg.Path()) == networkSDKPath {
		if !networkPythonFilterMetadataValid(g.networkPythonFilters) {
			return nil
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); ok {
			return g.networkPythonFilters
		}
		return nil
	}
	if sdkPath(pkg.Path()) == subnetPoolSDKPath {
		if !subnetPoolPythonFilterMetadataValid(g.subnetPoolPythonFilters) {
			return nil
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); ok {
			return g.subnetPoolPythonFilters
		}
		return nil
	}
	if sdkPath(pkg.Path()) == qosPolicySDKPath {
		if !qosPolicyPythonFilterMetadataValid(g.qosPolicyPythonFilters) {
			return nil
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); ok {
			return g.qosPolicyPythonFilters
		}
		return nil
	}
	if sdkPath(pkg.Path()) == addressGroupSDKPath {
		if !addressGroupPythonFilterMetadataValid(g.addressGroupPythonFilters) {
			return nil
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); ok {
			return g.addressGroupPythonFilters
		}
		return nil
	}
	if sdkPath(pkg.Path()) == "keymanager/v1/orders" {
		if !orderPythonFilterMetadataValid(g.orderPythonFilters) {
			return nil
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); ok {
			return g.orderPythonFilters
		}
		return nil
	}
	if sdkPath(pkg.Path()) == "keymanager/v1/containers" {
		if !containerPythonFilterMetadataValid(g.containerPythonFilters) {
			return nil
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); ok {
			return g.containerPythonFilters
		}
		return nil
	}
	if sdkPath(pkg.Path()) == "keymanager/v1/secrets" {
		if !secretPythonFilterMetadataValid(g.secretPythonFilters) {
			return nil
		}
		if _, ok := bodyFilterCollectionContract(pkg, plan, 0); ok {
			return g.secretPythonFilters
		}
		return nil
	}
	if sdkPath(pkg.Path()) != "network/v2/subnets" || g.pythonFilters == nil {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return nil
	}
	return g.pythonFilters
}

func pythonFilterBodyFields(manifest *pythonFilterManifest) map[string]string {
	if manifest == nil {
		return nil
	}
	result := make(map[string]string, len(manifest.Body))
	for name, field := range manifest.Body {
		if manifest.Resource == secretPythonResource || manifest.Resource == containerPythonResource || manifest.Resource == orderPythonResource {
			// Body properties have distinct accessors even when they share a
			// stored field: literal id, raw reference and formatted ID stay separate.
			result[name] = name
		} else {
			result[name] = field.Field
		}
	}
	return result
}

func pythonFilterQueryFields(manifest *pythonFilterManifest) map[string]string {
	if manifest == nil {
		return nil
	}
	result := make(map[string]string, len(manifest.Query))
	for key, value := range manifest.Query {
		result[key] = value
	}
	return result
}

func pythonFilterReserved(manifest *pythonFilterManifest) []string {
	if manifest == nil {
		return nil
	}
	return append([]string(nil), manifest.Reserved...)
}

func emitPythonFilterDescriptor(e *emitter, plan *collectionPlan, parents int) {
	manifest := e.pythonFilters
	if manifest == nil || parents != 0 || manifest.SDKPackage != "gophercloudsdk/"+sdkPath(e.pkg.Path()) {
		return
	}
	if _, ok := bodyFilterCollectionContract(e.pkg, plan, parents); !ok {
		return
	}
	e.printf("FilterDescriptor:&resource.FilterDescriptor{Query:")
	emitPythonFilterMap(e, manifest.Query)
	e.printf(",Body:")
	emitPythonFilterMap(e, pythonFilterBodyFields(manifest))
	e.printf(",Reserved:[]string{")
	for _, key := range manifest.Reserved {
		e.printf("%q,", key)
	}
	e.printf("}},\n")
}

func emitPythonFilterMap(e *emitter, fields map[string]string) {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	e.printf("map[string]string{")
	for _, key := range keys {
		e.printf("%q:%q,", key, fields[key])
	}
	e.printf("}")
}
