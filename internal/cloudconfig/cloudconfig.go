// Package cloudconfig reads the clouds.yaml, secure.yaml and clouds-public.yaml
// files, the OS_* environment and the built-in vendor profiles, and derives the
// SDK-owned settings of one cloud from them: the frozen native parse options,
// the default network, network roles, server address policy, image create
// policy, has_<service> flags and secgroup_source. It has no Connection state;
// Connect applies the result under its own option precedence.
package cloudconfig

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvendors"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"gopkg.in/yaml.v2"
)

// Configuration freezes the selected files once. Native authentication/TLS
// parsing and SDK network policy then consume the same bytes, even if files
// change meanwhile.
type Configuration struct {
	ParseOptions      []clouds.ParseOption
	DefaultNetwork    string
	NetworkRoles      network.NetworkRolePolicy
	ServerAddresses   compute.ServerAddressPolicy
	ImageCreatePolicy image.ImageCreatePolicy
	ServiceFlags      map[string]bool
	SecgroupSource    *string
}

type cloudNetworkDocument struct {
	Clouds       map[string]map[string]any `yaml:"clouds"`
	PublicClouds map[string]map[string]any `yaml:"public-clouds"`
	Client       map[string]any            `yaml:"client"`
}

// Load reads the named cloud from the first clouds file of locations (else
// OS_CLIENT_CONFIG_FILE, else the search paths) with its secure.yaml and
// profile layers, and applies the values of the selected regions entry;
// region is the explicit WithRegion value, if any.
func Load(name string, locations []string, region *string) (Configuration, error) {
	var result Configuration
	if len(locations) == 0 {
		if path := os.Getenv("OS_CLIENT_CONFIG_FILE"); path != "" {
			locations = []string{path}
		} else {
			var err error
			locations, err = SearchPaths("clouds.yaml")
			if err != nil {
				return result, err
			}
		}
	}
	base, path, err := ReadFirstFile(locations)
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
	result.ParseOptions = []clouds.ParseOption{clouds.WithCloudName(name), clouds.WithCloudsYAML(bytes.NewReader(base))}
	settings := maps.Clone(baseSettings)
	var baseDocument cloudNetworkDocument
	if err := yaml.Unmarshal(base, &baseDocument); err != nil {
		return result, err
	}
	clientSettings := maps.Clone(baseDocument.Client)
	secure, present, err := readOptionalCloudFile(filepath.Join(filepath.Dir(path), "secure.yaml"))
	if err != nil {
		return result, fmt.Errorf("read secure.yaml: %w", err)
	}
	if present {
		var secureDocument cloudNetworkDocument
		if err := yaml.Unmarshal(secure, &secureDocument); err != nil {
			return result, err
		}
		if clientSettings == nil {
			clientSettings = make(map[string]any)
		}
		maps.Copy(clientSettings, secureDocument.Client)
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
		result.ParseOptions = append(result.ParseOptions, clouds.WithSecureYAML(bytes.NewReader(secure)))
	}
	cloudSettings := maps.Clone(settings)
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
		paths, err := SearchPaths("clouds-public.yaml")
		if err != nil {
			return result, err
		}
		public, _, publicErr := ReadFirstFile(paths)
		var inherited map[string]any
		if publicErr == nil && metadataProfileName != "" {
			inherited, err = decodeCloudNetworkSettings(public, metadataProfileName, true)
			if err != nil {
				return result, fmt.Errorf("parse clouds-public.yaml: %w", err)
			}
		}
		// Like _expand_vendor_profile, a profile missing from clouds-public.yaml
		// falls back to the built-in vendor profiles. The native parser then
		// receives a document holding that profile, so authentication sees it too.
		if inherited == nil && metadataProfileName != "" {
			builtin, found, err := builtinCloudProfile(metadataProfileName)
			if err != nil {
				return result, err
			}
			if found {
				inherited = builtin
				public, err = yaml.Marshal(map[string]any{"public-clouds": map[string]any{profile: builtin}})
				if err != nil {
					return result, err
				}
				publicErr = nil
			}
		}
		if publicErr != nil {
			return result, publicErr
		}
		if inherited == nil {
			inherited = make(map[string]any)
		}
		// Python replaces network lists at each layer, including an empty list.
		maps.Copy(inherited, settings)
		settings = inherited
		result.ParseOptions = append(result.ParseOptions, clouds.WithCloudsPublicYAML(bytes.NewReader(public)))
	}
	settings, err = applyRegionValues(cloudSettings, settings, region)
	if err != nil {
		return result, err
	}
	result.DefaultNetwork, err = configuredDefaultNetwork(settings)
	if err != nil {
		return result, err
	}
	result.NetworkRoles, err = configuredNetworkRoles(settings)
	if err != nil {
		return result, err
	}
	result.ServerAddresses, err = ServerAddresses(settings, clientSettings)
	if err != nil {
		return result, err
	}
	result.ImageCreatePolicy, err = configuredImageCreatePolicy(name, settings)
	if err != nil {
		return result, err
	}
	result.ServiceFlags, result.SecgroupSource, err = configuredServiceSettings(settings)
	return result, err
}

// SearchPaths returns the search locations of filename, matching the
// version-pinned native parser's paths on every supported OS.
func SearchPaths(filename string) ([]string, error) {
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

// ReadFirstFile reads the first openable file of paths.
func ReadFirstFile(paths []string) ([]byte, string, error) {
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

// environmentCloudSettings mirrors OpenStackConfig._get_os_environ: every
// OS_* variable except the OS_TEST, OS_STD and OS_LOG families becomes a
// lower-case setting without the prefix, with its string value. Like Python,
// no settings exist when only selector variables are set.
func environmentCloudSettings(environ []string) map[string]any {
	selectors := map[string]bool{"OS_CLOUD": true, "OS_REGION_NAME": true, "OS_CLIENT_CONFIG_FILE": true, "OS_CLIENT_SECURE_FILE": true, "OS_CLOUD_NAME": true}
	settings := make(map[string]any)
	substantive := false
	for _, entry := range environ {
		key, value, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(key, "OS_") || strings.HasPrefix(key, "OS_TEST") || strings.HasPrefix(key, "OS_STD") || strings.HasPrefix(key, "OS_LOG") {
			continue
		}
		settings[strings.ToLower(key[len("OS_"):])] = value
		if !selectors[key] {
			substantive = true
		}
	}
	if !substantive {
		return nil
	}
	return settings
}

// LoadEnvironment derives the SDK-owned role configuration
// of Python's environment-variable cloud: OS_EXTERNAL_NETWORK and
// OS_INTERNAL_NETWORK legacy roles, OS_USE_EXTERNAL_NETWORK and
// OS_USE_INTERNAL_NETWORK flags, OS_HAS_<SERVICE> flags and OS_SECGROUP_SOURCE.
// An OS_PROFILE naming a built-in vendor profile supplies settings under the
// environment values, and its regions entry is selected like a file cloud's.
// Native authentication still comes from Gophercloud's environment parser.
func LoadEnvironment(environ []string, region *string) (Configuration, bool, error) {
	var result Configuration
	envSettings := environmentCloudSettings(environ)
	if envSettings == nil {
		return result, false, nil
	}
	settings := maps.Clone(envSettings)
	if name, ok := envSettings["profile"].(string); ok && name != "" {
		builtin, found, err := builtinCloudProfile(name)
		if err != nil {
			return result, true, err
		}
		if found {
			maps.Copy(builtin, envSettings)
			settings = builtin
		}
	}
	settings, err := applyRegionValues(envSettings, settings, region)
	if err != nil {
		return result, true, err
	}
	if result.DefaultNetwork, err = configuredDefaultNetwork(settings); err != nil {
		return result, true, err
	}
	if result.NetworkRoles, err = configuredNetworkRoles(settings); err != nil {
		return result, true, err
	}
	result.ServiceFlags, result.SecgroupSource, err = configuredServiceSettings(settings)
	return result, true, err
}

type cloudRegionEntry struct {
	name   string
	values map[string]any
}

// applyRegionValues mirrors the region step of the pinned
// OpenStackConfig.get_one for SDK-owned settings. The known regions are the
// cloud's own regions list; a cloud with region_name but no regions has only
// value-less entries; otherwise the regions of the merged profile are used,
// as _get_known_regions does. Entries are names or {name, values} objects.
// An explicit, nonempty WithRegion must name an entry. Without it the entry
// named by region_name is selected when present, so that the values match the
// native endpoint region, and otherwise the first entry, which is Python's
// default. Every non-null value of the selected entry replaces the merged
// setting, including whole networks lists.
func applyRegionValues(cloudSettings, settings map[string]any, explicit *string) (map[string]any, error) {
	raw, present := cloudSettings["regions"]
	if !present {
		if _, named := cloudSettings["region_name"]; named {
			return settings, nil
		}
		raw, present = settings["regions"]
	}
	if !present || raw == nil {
		return settings, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, invalid("regions must be a list")
	}
	entries := make([]cloudRegionEntry, 0, len(list))
	for i, item := range list {
		entry, err := parseRegionEntry(i, item)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return settings, nil
	}
	var selected *cloudRegionEntry
	if explicit != nil && *explicit != "" {
		for i := range entries {
			if entries[i].name == *explicit {
				selected = &entries[i]
				break
			}
		}
		if selected == nil {
			names := make([]string, len(entries))
			for i, entry := range entries {
				names[i] = entry.name
			}
			return nil, invalid("region %s is not a valid region name for this cloud; valid choices are %s (region names are case sensitive)", *explicit, strings.Join(names, ","))
		}
	} else {
		selected = &entries[0]
		if name, ok := settings["region_name"].(string); ok && name != "" {
			for i := range entries {
				if entries[i].name == name {
					selected = &entries[i]
					break
				}
			}
		}
	}
	if len(selected.values) == 0 {
		return settings, nil
	}
	result := maps.Clone(settings)
	if result == nil {
		result = make(map[string]any)
	}
	for key, value := range selected.values {
		if value != nil {
			result[key] = value
		}
	}
	return result, nil
}

func parseRegionEntry(index int, item any) (cloudRegionEntry, error) {
	switch value := item.(type) {
	case string:
		return cloudRegionEntry{name: value}, nil
	case map[any]any:
		entry := cloudRegionEntry{}
		for key, field := range value {
			switch key {
			case "name":
				name, ok := field.(string)
				if !ok {
					return entry, invalid("region entry %d name must be a string", index)
				}
				entry.name = name
			case "values":
				values, ok := field.(map[any]any)
				if !ok {
					return entry, invalid("region entry %d values must be an object", index)
				}
				entry.values = make(map[string]any, len(values))
				for k, v := range values {
					name, ok := k.(string)
					if !ok {
						return entry, invalid("region entry %d value keys must be strings", index)
					}
					entry.values[name] = v
				}
			default:
				return entry, invalid("invalid region entry %d: only name and values are allowed", index)
			}
		}
		if _, named := value["name"]; !named {
			return entry, invalid("invalid region entry %d: name is required", index)
		}
		return entry, nil
	default:
		return cloudRegionEntry{}, invalid("invalid region entry %d", index)
	}
}

// builtinCloudProfile resolves a pinned built-in vendor profile. A shutdown
// or remote profile is a configuration error.
func builtinCloudProfile(name string) (map[string]any, bool, error) {
	profile, found, err := cloudvendors.Profile(name)
	if err != nil {
		return nil, found, fmt.Errorf("%w: %w", resource.ErrInvalidOption, err)
	}
	return profile, found, nil
}

// Called after the existing default-network parser validates list/row/flag
// shapes. Secure > base > public already replaced the entire networks list.
func configuredNetworkRoles(settings map[string]any) (network.NetworkRolePolicy, error) {
	var rows []network.ConfiguredNetwork
	if raw, present := settings["networks"]; present {
		for i, rawRow := range raw.([]any) {
			row := rawRow.(map[any]any)
			base, _ := networkConfigBoolean(row["routes_externally"])
			ipv4, ipv6 := base, base
			if value, present := row["routes_ipv4_externally"]; present {
				ipv4, _ = networkConfigBoolean(value)
			}
			if value, present := row["routes_ipv6_externally"]; present {
				ipv6, _ = networkConfigBoolean(value)
			}
			source, _ := networkConfigBoolean(row["nat_source"])
			destination, _ := networkConfigBoolean(row["nat_destination"])
			defaultInterface, _ := networkConfigBoolean(row["default_interface"])
			name, ok := row["name"].(string)
			if !ok {
				return network.NetworkRolePolicy{}, invalid("network entry %d requires a string name or ID", i)
			}
			rows = append(rows, network.ConfiguredNetwork{Name: name, RoutesIPv4Externally: ipv4, RoutesIPv6Externally: ipv6, NATSource: source, NATDestination: destination, DefaultInterface: defaultInterface})
		}
	} else {
		for _, key := range []string{"external_network", "internal_network"} {
			value, present := settings[key]
			if !present || value == nil || value == "" {
				continue
			}
			name, ok := value.(string)
			if !ok || strings.TrimSpace(name) == "" {
				return network.NetworkRolePolicy{}, invalid("%s must be a nonempty string", key)
			}
			external := key == "external_network"
			rows = append(rows, network.ConfiguredNetwork{Name: name, RoutesIPv4Externally: external, RoutesIPv6Externally: external, NATDestination: !external, DefaultInterface: external})
		}
	}
	external, internal := true, true
	for _, flag := range []struct {
		name  string
		value *bool
	}{{"use_external_network", &external}, {"use_internal_network", &internal}} {
		if value, present := settings[flag.name]; present {
			parsed, err := networkConfigBoolean(value)
			if err != nil {
				return network.NetworkRolePolicy{}, fmt.Errorf("%s: %w", flag.name, err)
			}
			*flag.value = parsed
		}
	}
	return network.PrepareNetworkRoleOptions(network.WithConfiguredNetworks(rows...), network.WithExternalNetworkDiscovery(external), network.WithInternalNetworkDiscovery(internal))
}

// ServerAddresses derives the server address policy from the merged cloud
// settings, the client section and OS_PREFER_IPV6/OS_FORCE_IPV4.
func ServerAddresses(settings, client map[string]any) (compute.ServerAddressPolicy, error) {
	prefer, force := true, false
	boolean := func(settings map[string]any, key, alias string, fallback bool) (bool, error) {
		value, present := settings[key]
		if !present && alias != "" {
			value, present = settings[alias]
		}
		if !present {
			return fallback, nil
		}
		return networkConfigBoolean(value)
	}
	var err error
	prefer, err = boolean(client, "prefer_ipv6", "prefer-ipv6", prefer)
	if err != nil {
		return compute.ServerAddressPolicy{}, err
	}
	force, err = boolean(client, "force_ipv4", "broken-ipv6", force)
	if err != nil {
		return compute.ServerAddressPolicy{}, err
	}
	if value, present := os.LookupEnv("OS_PREFER_IPV6"); present {
		prefer, _ = networkConfigBoolean(value)
	}
	if value, present := os.LookupEnv("OS_FORCE_IPV4"); present {
		force, _ = networkConfigBoolean(value)
	}
	if !prefer {
		force = true
	}
	force, err = boolean(settings, "force_ipv4", "force-ipv4", force)
	if err != nil {
		return compute.ServerAddressPolicy{}, err
	}
	prefer, err = boolean(settings, "prefer_ipv6", "prefer-ipv6", true)
	if err != nil {
		return compute.ServerAddressPolicy{}, err
	}
	if !prefer {
		force = true
	}
	private, err := boolean(settings, "private", "", false)
	if err != nil {
		return compute.ServerAddressPolicy{}, err
	}
	source := compute.FloatingIPNeutron
	value, present := settings["floating_ip_source"]
	if !present {
		value, present = settings["floating-ip-source"]
	}
	if present {
		if value == nil || value == "" {
			source = compute.FloatingIPNone
		} else {
			text, ok := value.(string)
			if !ok {
				return compute.ServerAddressPolicy{}, invalid("floating_ip_source must be neutron, nova, none or null")
			}
			source = compute.FloatingIPSource(strings.ToLower(text))
		}
	}
	return compute.PrepareServerAddressPolicy(compute.WithPrivateCloud(private), compute.WithForceIPv4(force), compute.WithFloatingIPSource(source))
}
