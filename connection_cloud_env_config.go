package openstack

import (
	"maps"
	"os"
	"strings"
)

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

// loadEnvironmentCloudConfiguration derives the SDK-owned role configuration
// of Python's environment-variable cloud: OS_EXTERNAL_NETWORK and
// OS_INTERNAL_NETWORK legacy roles, OS_USE_EXTERNAL_NETWORK and
// OS_USE_INTERNAL_NETWORK flags, OS_HAS_<SERVICE> flags and OS_SECGROUP_SOURCE.
// An OS_PROFILE naming a built-in vendor profile supplies settings under the
// environment values, and its regions entry is selected like a file cloud's.
// Native authentication still comes from Gophercloud's environment parser.
func loadEnvironmentCloudConfiguration(environ []string, region *string) (cloudConfiguration, bool, error) {
	var result cloudConfiguration
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
	if result.defaultNetwork, err = configuredDefaultNetwork(settings); err != nil {
		return result, true, err
	}
	if result.networkRoles, err = configuredNetworkRoles(settings); err != nil {
		return result, true, err
	}
	result.serviceFlags, result.secgroupSource, err = configuredServiceSettings(settings)
	return result, true, err
}

func osEnviron() []string { return os.Environ() }
