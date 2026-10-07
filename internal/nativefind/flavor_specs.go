package nativefind

import (
	"context"
	"fmt"
	"maps"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
)

// FlavorExtraSpecs returns an owned copy of a resolved flavor. Nonempty inline
// extra specs need no request; nil or empty specs use Nova's native separate
// GET and result extractor. The original service/provider clients are retained,
// caller lookup query is not forwarded, and no microversion is selected here.
// Native missing/null extra_specs responses remain nil maps.
func FlavorExtraSpecs(ctx context.Context, client *gophercloud.ServiceClient, value *flavors.Flavor) (*flavors.Flavor, error) {
	invalid := func(message string) (*flavors.Flavor, error) {
		return nil, fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
	}
	if ctx == nil {
		return invalid("flavor extra specs require a context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if value == nil {
		return invalid("flavor extra specs require a resolved flavor")
	}
	owned := *value
	owned.ExtraSpecs = maps.Clone(value.ExtraSpecs)
	if !utf8.ValidString(owned.ID) || resource.ID(owned.ID).Validate() != nil {
		return invalid("flavor extra specs require a literal response ID")
	}
	for _, char := range owned.ID {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return invalid("flavor response ID cannot contain whitespace or controls")
		}
	}
	if _, err := nativeListURL(ctx, client, []string{"flavors", owned.ID, "os-extra_specs"}); err != nil {
		return nil, err
	}
	if len(owned.ExtraSpecs) != 0 {
		return &owned, nil
	}
	specs, err := flavors.ListExtraSpecs(ctx, client, owned.ID).Extract()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	owned.ExtraSpecs = maps.Clone(specs)
	return &owned, nil
}
