// Package masakari implements shared preflight for SDK-owned Masakari APIs.
package masakari

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

var uuidPattern = regexp.MustCompile(`^[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{12}$`)

func UUID(id string) error {
	if !uuidPattern.MatchString(id) {
		return fmt.Errorf("%w: Masakari route identifiers must be canonical UUIDs", resource.ErrInvalidOption)
	}
	return nil
}

// Require preserves the selected numeric version and verifies that source
// headers cannot replace it. A symbolic latest cannot prove a feature gate.
func Require(ctx context.Context, client *gophercloud.ServiceClient, minimum int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: Masakari service client is required", resource.ErrInvalidOption)
	}
	selected := client.Microversion
	if selected == "" {
		selected = "1.0"
	}
	if selected == "latest" {
		return fmt.Errorf("%w: select a numeric microversion for Masakari feature validation", resource.ErrUnsupported)
	}
	parts := strings.Split(selected, ".")
	if len(parts) != 2 || parts[0] != "1" || parts[1] == "" {
		return fmt.Errorf("%w: Masakari microversion must be 1.N", resource.ErrInvalidOption)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 || strconv.Itoa(minor) != parts[1] {
		return fmt.Errorf("%w: Masakari microversion must be canonical 1.N", resource.ErrInvalidOption)
	}
	explicit := false
	for key, value := range client.MoreHeaders {
		if !strings.EqualFold(key, "OpenStack-API-Version") {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) != 2 || fields[0] != "instance-ha" || fields[1] != selected {
			return fmt.Errorf("%w: Masakari version header conflicts with selected %s", resource.ErrInvalidOption, selected)
		}
		explicit = true
	}
	if client.Type != "instance-ha" && client.Type != "" {
		return fmt.Errorf("%w: Masakari client type must be instance-ha", resource.ErrInvalidOption)
	}
	if client.Type == "" && client.Microversion != "" && !explicit {
		return fmt.Errorf("%w: a manual Masakari client requires a matching explicit version header", resource.ErrInvalidOption)
	}
	if minor < minimum {
		return fmt.Errorf("%w: Masakari operation requires microversion 1.%d (selected %s)", resource.ErrUnsupported, minimum, selected)
	}
	return nil
}
