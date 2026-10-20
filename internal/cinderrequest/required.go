package cinderrequest

import (
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// requireSupport follows supports_microversion and Keystoneauth's separately
// audited version_between/version_match rules, not cap-based negotiation.
func requireSupport(maximum, minimum, selected, required string) error {
	if maximum == "" || minimum == "" {
		return fmt.Errorf("%w: Cinder must advertise both bounds for microversion %s", resource.ErrUnsupported, required)
	}
	supported, err := microversions.Supports(maximum, minimum, "", required)
	if err != nil {
		return err
	}
	if !supported {
		return fmt.Errorf("%w: Cinder range %s..%s does not support %s", resource.ErrUnsupported, minimum, maximum, required)
	}
	if selected == "" {
		return nil
	}
	supported, err = microversions.Matches(selected, required)
	if err != nil {
		return err
	}
	// A global latest major differs from the required finite major. A finite
	// major with latest minor remains eligible, as in version_match.
	if !supported {
		return fmt.Errorf("%w: selected Cinder microversion %s does not support %s", resource.ErrUnsupported, selected, required)
	}
	return nil
}
