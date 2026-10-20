package openstack

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestCloudImageConfigurationMergesDefaultsAndFreezesSourceValues(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "clouds.yaml")
	writeCloudConfigFile(t, "clouds-public.yaml", "public-clouds:\n  provider:\n    image_format: qcow2\n    image_api_use_tasks: false\n    disable_vendor_agent: {provider: true}\n")
	writeCloudConfigFile(t, path, "clouds:\n  dev:\n    profile: provider\n    image_format: vhd\n    disable_vendor_agent: {base: 1}\n")
	writeCloudConfigFile(t, filepath.Join(filepath.Dir(path), "secure.yaml"), "clouds:\n  dev:\n    image_format: raw\n    image_api_use_tasks: null\n    has_object_store: false\n    disable_vendor_agent: {secure: true}\n")
	configuration, err := loadCloudConfiguration("dev", []string{path})
	if err != nil {
		t.Fatal(err)
	}
	policy := configuration.imageCreatePolicy
	if string(policy.ImageFormat) != `"raw"` || string(policy.UseTasks) != "null" || string(policy.RawObjectStoreEnabled) != "false" || policy.CloudName != "dev" {
		t.Fatal(policy)
	}
	var vendor map[string]any
	if err := json.Unmarshal(policy.RawVendorAgent, &vendor); err != nil {
		t.Fatal(err)
	}
	if vendor["secure"] != true {
		t.Fatal(vendor)
	}
	before := string(policy.RawVendorAgent)
	writeCloudConfigFile(t, path, "clouds: {dev: {image_format: changed}}\n")
	writeCloudConfigFile(t, filepath.Join(filepath.Dir(path), "secure.yaml"), "clouds: {dev: {has_object_store: true}}\n")
	if string(configuration.imageCreatePolicy.ImageFormat) != `"raw"` || string(configuration.imageCreatePolicy.RawVendorAgent) != before || string(configuration.imageCreatePolicy.RawObjectStoreEnabled) != "false" {
		t.Fatal("policy followed changed files")
	}
}

func TestConfiguredImageCreatePolicyDefersRawShapeAndAvailabilityChecks(t *testing.T) {
	policy, err := configuredImageCreatePolicy("dev", map[string]any{"image_format": nil, "image_api_use_tasks": []any{}, "disable_vendor_agent": []any{[]any{"vendor", 1}}, "has_object_store": "false"})
	if err != nil || string(policy.ImageFormat) != "null" || string(policy.UseTasks) != "[]" || string(policy.RawVendorAgent) != `[["vendor",1]]` || string(policy.RawObjectStoreEnabled) != `"false"` {
		t.Fatal(policy, err)
	}
	_, err = configuredImageCreatePolicy("dev", map[string]any{"disable_vendor_agent": map[any]any{1: "invalid JSON key"}})
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestConnectionImageCreatePolicyCapturesExplicitOptions(t *testing.T) {
	raw := json.RawMessage(`"raw"`)
	vendor := map[string]any{"owned": "value"}
	option := WithImageCreatePolicy(image.WithImageCreatePolicyOpts(image.ImageCreatePolicy{ImageFormat: raw, DisableVendorAgent: vendor}))
	raw[1] = 'x'
	vendor["owned"] = "changed"
	connection, err := FromProvider(&gophercloud.ProviderClient{}, option, WithEndpointFor(Image, "v2", "https://cloud.test/v2/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := connection.Image(context.Background())
	if err != nil || service == nil {
		t.Fatal(service, err)
	}
	if string(connection.options.imageCreatePolicy.ImageFormat) != `"raw"` || string(connection.options.imageCreatePolicy.DisableVendorAgent["owned"].(json.RawMessage)) != `"value"` {
		t.Fatal(connection.options.imageCreatePolicy)
	}
}
