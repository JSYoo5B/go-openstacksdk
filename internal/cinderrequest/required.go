package cinderrequest

import (
	"fmt"

	"gophercloudsdk/resource"
)

// requireSupport follows supports_microversion and Keystoneauth's separately
// audited version_between/version_match rules, not cap-based negotiation.
func requireSupport(maximum, minimum, selected, required string) error {
	if maximum == "" || minimum == "" {
		return fmt.Errorf("%w: Cinder must advertise both bounds for microversion %s", resource.ErrUnsupported, required)
	}
	want, err := ParseVersion(required)
	if err != nil {
		return err
	}
	low, err := ParseVersion(minimum)
	if err != nil {
		return err
	}
	high, err := ParseVersion(maximum)
	if err != nil {
		return err
	}
	if want.Compare(low) < 0 || want.Compare(high) > 0 {
		return fmt.Errorf("%w: Cinder range %s..%s does not support %s", resource.ErrUnsupported, minimum, maximum, required)
	}
	if selected == "" {
		return nil
	}
	candidate, err := ParseVersion(selected)
	if err != nil {
		return err
	}
	// A global latest major differs from the required finite major. A finite
	// major with latest minor remains eligible, as in version_match.
	if candidate[0] == nil || want[0] == nil || candidate[0].Cmp(want[0]) != 0 || candidate.Compare(want) < 0 {
		return fmt.Errorf("%w: selected Cinder microversion %s does not support %s", resource.ErrUnsupported, selected, required)
	}
	return nil
}
