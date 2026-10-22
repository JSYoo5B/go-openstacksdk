package openstack

import (
	"maps"
	"strings"
)

// officialServiceTypes lists the official service types of the OpenStack
// service-types-authority data shipped with os-service-types 1.8.1 (data
// version 2024-05-08, sha 52d438fe913eecea4e14d1e83f148cbe22edef91), the
// minimum version required by the pinned openstacksdk. CloudRegion.has_service
// uses is_official(service_type) as the default of an unset has_<type> flag.
var officialServiceTypes = map[string]bool{
	"accelerator": true, "admin-logic": true, "alarm": true, "application-catalog": true,
	"application-container": true, "application-deployment": true, "backup": true,
	"baremetal": true, "baremetal-introspection": true, "block-storage": true,
	"clustering": true, "compute": true, "container-infrastructure-management": true,
	"data-processing": true, "data-protection-orchestration": true, "database": true,
	"dns": true, "ec2-api": true, "event": true, "function-engine": true, "identity": true,
	"image": true, "instance-ha": true, "key-manager": true, "load-balancer": true,
	"log-management": true, "message": true, "meter": true, "monitoring": true,
	"monitoring-events": true, "monitoring-logging": true,
	"multi-region-network-automation": true, "network": true, "nfv-orchestration": true,
	"object-store": true, "operator-policy": true, "orchestration": true, "placement": true,
	"rating": true, "reservation": true, "resource-optimization": true,
	"root-cause-analysis": true, "search": true, "shared-file-system": true, "workflow": true,
}

// knownServiceTypeAliases lists the aliases of the same data that end in a
// version digit. keystoneauth derives an implied major version from them.
var knownServiceTypeAliases = map[string]bool{
	"volumev2": true, "volumev3": true, "sharev2": true, "workflowv2": true,
}

// serviceFlagKey mirrors CloudRegion.has_service's key normalization.
func serviceFlagKey(serviceType string) string {
	return strings.ReplaceAll(strings.ToLower(serviceType), "-", "_")
}

// WithServiceEnabled sets the has_<service> configuration flag of one service
// type, as clouds.yaml does. A disabled service makes HasService and
// HasServiceVersion report false without any lookup. The type is normalized
// like Python (lower case, '-' and '_' equivalent). It takes precedence over
// clouds.yaml for that type.
func WithServiceEnabled(serviceType string, enabled bool) ConnectionOption {
	return func(o *connectionOptions) error {
		if strings.TrimSpace(serviceType) == "" {
			return invalid("service type must not be empty")
		}
		flags := maps.Clone(o.serviceFlags)
		if flags == nil {
			flags = make(map[string]bool)
		}
		flags[serviceFlagKey(serviceType)] = enabled
		o.serviceFlags = flags
		return nil
	}
}

// WithSecurityGroupSource sets the secgroup_source configuration, replacing
// clouds.yaml. "neutron" (the default) uses Neutron security groups when the
// network service is available; "nova" (or any other casing of either value)
// uses the Nova os-security-groups API; any other value, including the empty
// string that mirrors a YAML null, makes the security-group helpers fail with
// network.ErrUnavailableSecurityGroups. As in Python, only the exact string
// "neutron" selects Neutron.
func WithSecurityGroupSource(source string) ConnectionOption {
	return func(o *connectionOptions) error {
		value := source
		o.secgroupSource, o.secgroupSourceSet = &value, true
		return nil
	}
}

// configuredServiceSettings reads has_<service> flags and secgroup_source from
// the merged clouds.yaml settings of one cloud.
func configuredServiceSettings(settings map[string]any) (map[string]bool, *string, error) {
	var flags map[string]bool
	for key, value := range settings {
		if !strings.HasPrefix(key, "has_") || len(key) == len("has_") {
			continue
		}
		enabled, ok := value.(bool)
		if !ok {
			// CloudRegion.has_service asserts a bool value.
			return nil, nil, invalid("%s must be a boolean, got %T", key, value)
		}
		if flags == nil {
			flags = make(map[string]bool)
		}
		// Python looks the flag up by its exact normalized key only.
		flags[key[len("has_"):]] = enabled
	}
	value, present := settings["secgroup_source"]
	if !present {
		value, present = settings["secgroup-source"]
	}
	if !present {
		return flags, nil, nil
	}
	if value == nil {
		empty := ""
		return flags, &empty, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, nil, invalid("secgroup_source must be a string or null, got %T", value)
	}
	return flags, &text, nil
}

func (o *connectionOptions) applyConfiguredServiceSettings(configuration cloudConfiguration) {
	if len(configuration.serviceFlags) != 0 {
		flags := maps.Clone(configuration.serviceFlags)
		maps.Copy(flags, o.serviceFlags)
		o.serviceFlags = flags
	}
	if !o.secgroupSourceSet && configuration.secgroupSource != nil {
		value := *configuration.secgroupSource
		o.secgroupSource = &value
	}
}

// ServiceEnabled mirrors CloudRegion.has_service: the has_<service> flag from
// WithServiceEnabled or clouds.yaml, else whether the type is an official
// service type. Aliases such as volumev3 are not official and default to
// false. No request is made.
func (c *Connection) ServiceEnabled(serviceType string) bool {
	if c == nil {
		return false
	}
	key := serviceFlagKey(serviceType)
	if enabled, ok := c.options.serviceFlags[key]; ok {
		return enabled
	}
	return officialServiceTypes[strings.ReplaceAll(key, "_", "-")]
}

// securityGroupSource returns secgroup_source with Python's "neutron" default.
func (c *Connection) securityGroupSource() string {
	if c.options.secgroupSource == nil {
		return "neutron"
	}
	return *c.options.secgroupSource
}

// hasSecurityGroups mirrors _has_secgroups.
func (c *Connection) hasSecurityGroups() bool {
	source := strings.ToLower(c.securityGroupSource())
	return source == "neutron" || source == "nova"
}
