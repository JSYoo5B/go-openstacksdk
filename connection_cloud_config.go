package gophercloudsdk

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"gopkg.in/yaml.v2"
)

// Freeze the selected files once. Native authentication/TLS parsing and SDK
// network policy then consume the same bytes, even if files change meanwhile.
type cloudConfiguration struct {
	parseOptions   []clouds.ParseOption
	defaultNetwork string
}

type cloudNetworkDocument struct {
	Clouds       map[string]map[string]any `yaml:"clouds"`
	PublicClouds map[string]map[string]any `yaml:"public-clouds"`
}

func loadCloudConfiguration(name string, locations []string) (cloudConfiguration, error) {
	var result cloudConfiguration
	if len(locations) == 0 {
		if path := os.Getenv("OS_CLIENT_CONFIG_FILE"); path != "" {
			locations = []string{path}
		} else {
			var err error
			locations, err = cloudSearchPaths("clouds.yaml")
			if err != nil {
				return result, err
			}
		}
	}
	base, path, err := readFirstCloudFile(locations)
	if err != nil {
		return result, err
	}
	var baseHeaders clouds.Clouds
	if err := yaml.NewDecoder(bytes.NewReader(base)).Decode(&baseHeaders); err != nil {
		return result, err
	}
	header, ok := baseHeaders.Clouds[name]
	if !ok {
		return result, fmt.Errorf("cloud %q not found in clouds.yaml", name)
	}
	baseSettings, err := decodeCloudNetworkSettings(base, name, false)
	if err != nil {
		return result, err
	}
	result.parseOptions = []clouds.ParseOption{clouds.WithCloudName(name), clouds.WithCloudsYAML(bytes.NewReader(base))}
	settings := maps.Clone(baseSettings)
	secure, present, err := readOptionalCloudFile(filepath.Join(filepath.Dir(path), "secure.yaml"))
	if err != nil {
		return result, fmt.Errorf("read secure.yaml: %w", err)
	}
	if present {
		var secureHeaders clouds.Clouds
		if err := yaml.NewDecoder(bytes.NewReader(secure)).Decode(&secureHeaders); err != nil {
			return result, fmt.Errorf("parse secure.yaml: %w", err)
		}
		// Preserve native nonzero profile/cloud inheritance for authentication.
		if override, ok := secureHeaders.Clouds[name]; ok {
			if override.Profile != "" {
				header.Profile = override.Profile
			}
			if override.Cloud != "" {
				header.Cloud = override.Cloud
			}
		}
		overrides, err := decodeCloudNetworkSettings(secure, name, false)
		if err != nil {
			return result, fmt.Errorf("parse secure.yaml: %w", err)
		}
		if settings == nil {
			settings = make(map[string]any)
		}
		maps.Copy(settings, overrides)
		result.parseOptions = append(result.parseOptions, clouds.WithSecureYAML(bytes.NewReader(secure)))
	}
	profile := header.Profile
	if profile == "" {
		profile = header.Cloud
	}
	// Python's metadata overlay preserves null/empty profile presence. Native
	// authentication instead falls back from zero profile/cloud values. Keep
	// those established auth rules without accidentally inheriting networks.
	metadataProfile, hasProfile := settings["profile"]
	if !hasProfile {
		metadataProfile = settings["cloud"]
	}
	metadataProfileName, validProfile := metadataProfile.(string)
	if metadataProfile != nil && !validProfile {
		return result, invalid("cloud profile must be a string or null")
	}
	if profile != "" {
		paths, err := cloudSearchPaths("clouds-public.yaml")
		if err != nil {
			return result, err
		}
		public, _, err := readFirstCloudFile(paths)
		if err != nil {
			return result, err
		}
		var inherited map[string]any
		if metadataProfileName != "" {
			inherited, err = decodeCloudNetworkSettings(public, metadataProfileName, true)
			if err != nil {
				return result, fmt.Errorf("parse clouds-public.yaml: %w", err)
			}
		}
		if inherited == nil {
			inherited = make(map[string]any)
		}
		// Python replaces network lists at each layer, including an empty list.
		maps.Copy(inherited, settings)
		settings = inherited
		result.parseOptions = append(result.parseOptions, clouds.WithCloudsPublicYAML(bytes.NewReader(public)))
	}
	result.defaultNetwork, err = configuredDefaultNetwork(settings)
	return result, err
}

// Match the version-pinned native parser's paths on every supported OS.
func cloudSearchPaths(filename string) ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current working directory: %w", err)
	}
	userConfig := os.Getenv("XDG_CONFIG_HOME")
	if userConfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		userConfig = filepath.Join(home, ".config")
	}
	return []string{filepath.Join(cwd, filename), filepath.Join(userConfig, "openstack", filename), filepath.Join("/etc/openstack", filename)}, nil
}

func readOptionalCloudFile(path string) ([]byte, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		// The native parser skips every open failure, including permissions.
		return nil, false, nil
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	return data, true, err
}

func readFirstCloudFile(paths []string) ([]byte, string, error) {
	for _, path := range paths {
		data, present, err := readOptionalCloudFile(path)
		if present {
			return data, path, err
		}
	}
	return nil, "", fmt.Errorf("clouds file not found. Search locations were: %v", paths)
}

func decodeCloudNetworkSettings(data []byte, name string, public bool) (map[string]any, error) {
	var document cloudNetworkDocument
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&document); err != nil {
		return nil, err
	}
	if public {
		return document.PublicClouds[name], nil
	}
	return document.Clouds[name], nil
}

func networkConfigBoolean(value any) (bool, error) {
	switch value := value.(type) {
	case nil:
		return false, nil
	case bool:
		return value, nil
	case string:
		return strings.EqualFold(value, "true"), nil
	default:
		return false, invalid("network flags must be booleans or strings, got %T", value)
	}
}

func configuredDefaultNetwork(settings map[string]any) (string, error) {
	value, hasNetworks := settings["networks"]
	for _, key := range []string{"external_network", "internal_network"} {
		if _, hasLegacy := settings[key]; hasLegacy && hasNetworks {
			return "", invalid("%s and networks cannot be combined", key)
		}
	}
	if !hasNetworks {
		// The deprecated external-network setting selects the default too.
		if value, present := settings["external_network"]; present && value != nil && value != "" {
			name, ok := value.(string)
			if !ok || strings.TrimSpace(name) == "" {
				return "", invalid("external_network must be a nonempty string")
			}
			return name, nil
		}
		return "", nil
	}
	rows, ok := value.([]any)
	if !ok {
		return "", invalid("networks must be a list; use [] to clear inherited networks")
	}
	var selected string
	var natDestination bool
	for i, raw := range rows {
		row, ok := raw.(map[any]any)
		if !ok {
			return "", invalid("network entry %d must be an object", i)
		}
		name, ok := row["name"].(string)
		if !ok || strings.TrimSpace(name) == "" {
			return "", invalid("network entry %d requires a nonempty string name or ID", i)
		}
		for _, flag := range []string{"default_interface", "nat_destination", "nat_source", "routes_externally", "routes_ipv4_externally", "routes_ipv6_externally"} {
			enabled, err := networkConfigBoolean(row[flag])
			if err != nil {
				return "", fmt.Errorf("network entry %d %s: %w", i, flag, err)
			}
			if flag == "default_interface" && enabled {
				if selected != "" {
					return "", invalid("only one network may have default_interface enabled")
				}
				selected = name
			}
			if flag == "nat_destination" && enabled {
				if natDestination {
					return "", invalid("only one network may have nat_destination enabled")
				}
				natDestination = true
			}
		}
	}
	return selected, nil
}
