// Package manilaversion validates the selected version before Manila scopes
// choose a route or resolve a name. It never upgrades or mutates a client.
package manilaversion

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

// Minor returns the configured 2.N minor version. Empty selection means 2.0;
// Symbolic latest cannot establish the 2.7 route switch before HTTP. Use a
// numeric version, optionally selected through Connection range discovery.
// Manila reads its legacy header, so a generic version header alone cannot
// establish the selected version on a manual client with an empty Type.
func Minor(client *gophercloud.ServiceClient) (int, error) {
	if client == nil {
		return 0, fmt.Errorf("%w: Manila version validation requires a service client", resource.ErrInvalidOption)
	}
	selected := client.Microversion
	expected := selected
	if expected == "" {
		expected = "2.0"
	}
	if expected == "latest" {
		return 0, fmt.Errorf("%w: Manila quota routes require a selected numeric microversion; negotiate a range before binding a scope", resource.ErrUnsupported)
	}
	parts := strings.Split(expected, ".")
	if len(parts) != 2 || parts[0] != "2" {
		return 0, fmt.Errorf("%w: Manila microversion must be 2.N", resource.ErrInvalidOption)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 || strconv.Itoa(minor) != parts[1] {
		return 0, fmt.Errorf("%w: invalid Manila microversion %q", resource.ErrInvalidOption, selected)
	}
	legacy := false
	for key, value := range client.MoreHeaders {
		switch {
		case strings.EqualFold(key, "X-OpenStack-Manila-API-Version"):
			if value != expected {
				return 0, fmt.Errorf("%w: Manila version header conflicts with selected microversion", resource.ErrInvalidOption)
			}
			legacy = true
		case strings.EqualFold(key, "OpenStack-API-Version"):
			if value != "shared-file-system "+expected && value != "sharev2 "+expected && value != "share "+expected {
				return 0, fmt.Errorf("%w: Manila version header conflicts with selected microversion", resource.ErrInvalidOption)
			}
		}
	}
	if selected != "" {
		switch client.Type {
		case "shared-file-system", "sharev2", "share":
		case "":
			if !legacy {
				return 0, fmt.Errorf("%w: Manila microversion requires a native Manila client type or matching legacy version header", resource.ErrInvalidOption)
			}
		default:
			return 0, fmt.Errorf("%w: Manila quotas require a Manila service client", resource.ErrInvalidOption)
		}
	}
	return minor, nil
}
