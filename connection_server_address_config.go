package gophercloudsdk

import (
	"os"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/compute"
)

func configuredServerAddresses(settings, client map[string]any) (compute.ServerAddressPolicy, error) {
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
