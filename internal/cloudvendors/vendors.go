// Package cloudvendors embeds the built-in vendor profiles of the pinned
// OpenStackSDK (openstack/config/vendors) unchanged and resolves them like
// openstack.config.vendors.get_profile and the profile branch of
// OpenStackConfig._expand_vendor_profile. See README.md for provenance.
package cloudvendors

import (
	"embed"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"path"
	"sort"
	"sync"

	"gopkg.in/yaml.v2"
)

//go:embed data/*.json data/*.yaml
var files embed.FS

// ErrShutdown reports a profile whose status is "shutdown"; Python raises
// ConfigException for it.
var ErrShutdown = errors.New("vendor profile references a cloud that no longer exists")

// ErrRemoteProfile reports a profile (or a nested profile) given as a URL.
// Python fetches its well-known document over HTTP; this package does not.
var ErrRemoteProfile = errors.New("remote vendor profiles are not fetched")

var (
	once     sync.Once
	profiles map[string]map[string]any
	loadErr  error
)

func load() {
	profiles = make(map[string]map[string]any)
	entries, err := files.ReadDir("data")
	if err != nil {
		loadErr = err
		return
	}
	// Python reads *.yaml first and then *.json, a later file replacing an
	// earlier one of the same name.
	var yamlFiles, jsonFiles []string
	for _, entry := range entries {
		switch path.Ext(entry.Name()) {
		case ".yaml":
			yamlFiles = append(yamlFiles, entry.Name())
		case ".json":
			jsonFiles = append(jsonFiles, entry.Name())
		}
	}
	sort.Strings(yamlFiles)
	sort.Strings(jsonFiles)
	for _, name := range append(yamlFiles, jsonFiles...) {
		data, err := files.ReadFile("data/" + name)
		if err != nil {
			loadErr = err
			return
		}
		var document struct {
			Name    string         `yaml:"name"`
			Profile map[string]any `yaml:"profile"`
		}
		if err := yaml.Unmarshal(data, &document); err != nil {
			loadErr = fmt.Errorf("vendor profile %s: %w", name, err)
			return
		}
		profiles[document.Name] = document.Profile
	}
}

// Names lists the built-in profile names in sorted order.
func Names() []string {
	once.Do(load)
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Profile returns a copy of the named built-in profile settings with the
// status and message keys removed, after following a nested "profile" key as
// Python does (the nested profile replaces the outer settings when found). A
// name that is not built in returns (nil, false, nil), or ErrRemoteProfile
// when it is a URL. A shutdown profile returns ErrShutdown; a deprecated one
// is returned without a warning.
func Profile(name string) (map[string]any, bool, error) {
	once.Do(load)
	if loadErr != nil {
		return nil, false, loadErr
	}
	data, ok := profiles[name]
	if !ok {
		if isURL(name) {
			return nil, false, fmt.Errorf("%w: %s", ErrRemoteProfile, name)
		}
		return nil, false, nil
	}
	result := maps.Clone(data)
	if nested, present := result["profile"]; present {
		delete(result, "profile")
		if nestedName, ok := nested.(string); ok && nestedName != "" {
			nestedData, found, err := Profile(nestedName)
			if err != nil {
				return nil, false, err
			}
			if found {
				result = nestedData
			}
		}
	}
	status := "active"
	if value, ok := result["status"].(string); ok {
		status = value
	}
	message, _ := result["message"].(string)
	delete(result, "status")
	delete(result, "message")
	if status == "shutdown" {
		return nil, true, fmt.Errorf("%w: %s: %s", ErrShutdown, name, message)
	}
	return result, true, nil
}

func isURL(name string) bool {
	parsed, err := url.Parse(name)
	return err == nil && parsed.Host != ""
}
