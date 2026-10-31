package cloudconfig

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCloudImageConfigurationMergesDefaultsAndFreezesSourceValues(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "clouds.yaml")
	writeCloudConfigFile(t, "clouds-public.yaml", "public-clouds:\n  provider:\n    image_format: qcow2\n    image_api_use_tasks: false\n    disable_vendor_agent: {provider: true}\n")
	writeCloudConfigFile(t, path, "clouds:\n  dev:\n    profile: provider\n    image_format: vhd\n    disable_vendor_agent: {base: 1}\n")
	writeCloudConfigFile(t, filepath.Join(filepath.Dir(path), "secure.yaml"), "clouds:\n  dev:\n    image_format: raw\n    image_api_use_tasks: null\n    has_object_store: false\n    disable_vendor_agent: {secure: true}\n")
	configuration, err := Load("dev", []string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	policy := configuration.ImageCreatePolicy
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
	if string(configuration.ImageCreatePolicy.ImageFormat) != `"raw"` || string(configuration.ImageCreatePolicy.RawVendorAgent) != before || string(configuration.ImageCreatePolicy.RawObjectStoreEnabled) != "false" {
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
