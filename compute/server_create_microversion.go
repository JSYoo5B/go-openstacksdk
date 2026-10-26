package compute

import (
	"context"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/gophercloud/gophercloud/v2"
)

// serverCreateAdvertised caches one Compute version discovery per create.
type serverCreateAdvertised struct {
	read             bool
	maximum, minimum string
}

// supportsCreateMicroversion follows openstack/utils.py supports_microversion
// for the version-gated fields of the cloud create_server: the advertised
// range must contain required and, when the client selected a microversion,
// that selection must be at least required. A selected version below required
// is decided without discovery, as the outcome cannot change. Bounds that
// Connection already negotiated are reused without another request. A clean
// 404/405 discovery rejection or a missing bound means unsupported; other
// discovery failures are returned before the server POST. When the client has no selected
// microversion, a supported gate raises only this create request to required,
// leaving the shared client unchanged.
func (s *Servers) supportsCreateMicroversion(ctx context.Context, o *createServerOptions, required string) (bool, error) {
	selected := s.client.Microversion
	if selected != "" {
		matches, err := microversions.Matches(selected, required)
		if err != nil || !matches {
			return false, err
		}
	}
	if !o.advertised.read && s.dependencies.AdvertisedMinimum != "" && s.dependencies.AdvertisedMaximum != "" {
		o.advertised = serverCreateAdvertised{read: true, maximum: s.dependencies.AdvertisedMaximum, minimum: s.dependencies.AdvertisedMinimum}
	}
	if !o.advertised.read {
		if err := checkServerCreation(ctx); err != nil {
			return false, err
		}
		source, err := cloudread.Capture(ctx, s.client, "compute")
		if err != nil {
			return false, s.wrap("discover compute microversions", err)
		}
		discovery, err := microversions.Read(ctx, source, microversions.NovaProfile)
		err = errors.Join(err, checkServerCreation(ctx))
		if err != nil {
			return false, s.wrap("discover compute microversions", err)
		}
		o.advertised = serverCreateAdvertised{read: true}
		if discovery.Found {
			o.advertised.maximum, o.advertised.minimum = discovery.Maximum, discovery.Minimum
		}
	}
	supported, err := microversions.Supports(o.advertised.maximum, o.advertised.minimum, selected, required)
	if err != nil {
		return false, s.wrap("discover compute microversions", err)
	}
	if supported && selected == "" {
		if raised, _ := microversions.Matches(o.requestMicroversion, required); o.requestMicroversion == "" || !raised {
			o.requestMicroversion = required
		}
	}
	return supported, nil
}

// createClient returns the client for a create POST: the shared client, or a
// copy carrying the request microversion raised by an advertised gate.
func createClient(client *gophercloud.ServiceClient, microversion string) *gophercloud.ServiceClient {
	if microversion == "" || client == nil {
		return client
	}
	copied := *client
	copied.Microversion = microversion
	return &copied
}
