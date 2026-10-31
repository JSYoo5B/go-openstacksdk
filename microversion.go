package openstack

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/gophercloud/gophercloud/v2"
)

type microversionRange struct{ minimum, maximum string }

// WithMicroversionRange selects the highest microversion shared by the cloud
// and the caller's inclusive range. Empty minimum means the cloud minimum;
// empty maximum or "latest" means the cloud maximum. Discovery is deferred
// until the service is first used, and uses that call's context.
// An exact WithMicroversion always takes precedence, independently of order.
func WithMicroversionRange(service Service, minimum, maximum string) ConnectionOption {
	return func(o *connectionOptions) error {
		major := serviceDefinitions[service].microversionMajor
		if major == "" {
			return unsupported(string(service), "microversions")
		}
		if err := microversions.ValidateRange(string(service), major, minimum, maximum); err != nil {
			return err
		}
		o.microversionRanges[service] = microversionRange{minimum, maximum}
		return nil
	}
}

// WithLatestMicroversion selects the highest microversion advertised by the
// service. Pin an upper bound with WithMicroversionRange when an application's
// response contract must remain stable across cloud upgrades.
func WithLatestMicroversion(service Service) ConnectionOption {
	return WithMicroversionRange(service, "", "latest")
}

// MicroversionSelection separates configuration from the selected API version.
// Supported bounds and DiscoveryURL are populated only after negotiation.
// Exact requests bypass discovery, even when a range was also supplied.
type MicroversionSelection struct {
	Service          Service
	RequestedExact   string
	RequestedMinimum string
	RequestedMaximum string
	SupportedMinimum string
	SupportedMaximum string
	Selected         string
	DiscoveryURL     string
	Negotiated       bool
}

// Microversion returns the selection for a service's default API version,
// lazily constructing its shared client if needed. The returned value is a
// copy; changing it cannot change the cached client or configuration.
func (c *Connection) Microversion(ctx context.Context, service Service) (MicroversionSelection, error) {
	if err := ctx.Err(); err != nil {
		return MicroversionSelection{}, err
	}
	definition, ok := serviceDefinitions[service]
	if !ok || definition.microversionMajor == "" {
		return MicroversionSelection{}, unsupported(string(service), "microversions")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.serviceClientVersion(ctx, service, definition.version); err != nil {
		return MicroversionSelection{}, err
	}
	return c.options.microversionSelections[serviceKey(service, definition.version)], nil
}

// selectMicroversion runs before a successfully configured client enters the
// connection cache. Its caller holds the connection mutex.
func (c *Connection) selectMicroversion(ctx context.Context, service Service, client *gophercloud.ServiceClient) (MicroversionSelection, error) {
	request, negotiate := c.options.microversionRanges[service]
	selection := MicroversionSelection{
		Service: service, RequestedExact: c.options.microversions[service],
		RequestedMinimum: request.minimum, RequestedMaximum: request.maximum,
		Selected: c.options.microversions[service],
	}
	if selection.Selected != "" || !negotiate {
		return selection, nil
	}
	low, high, discoveryURL, err := microversions.Discover(ctx, microversionTarget(service), client)
	if err != nil {
		return MicroversionSelection{}, fmt.Errorf("discover %s microversions: %w", service, err)
	}
	selection.SupportedMinimum, selection.SupportedMaximum = low.String(), high.String()
	selection.DiscoveryURL = discoveryURL
	selected, ok := microversions.Clamp(low, high, request.minimum, request.maximum)
	if !ok {
		return MicroversionSelection{}, unsupported(string(service), fmt.Sprintf("microversion range [%s, %s]; cloud supports [%s, %s]", request.minimum, request.maximum, selection.SupportedMinimum, selection.SupportedMaximum))
	}
	selection.Selected, selection.Negotiated = selected.String(), true
	return selection, nil
}

// microversionTarget marks the services whose project-scoped catalog URLs
// point beneath the version discovery document.
func microversionTarget(service Service) microversions.Target {
	target := microversions.Target{Service: string(service), Major: serviceDefinitions[service].microversionMajor}
	switch service {
	case Compute, BlockStorage, SharedFileSystem, InstanceHA:
		target.StripProject = true
	}
	return target
}
