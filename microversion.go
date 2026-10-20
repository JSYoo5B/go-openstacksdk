package openstack

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
)

type microversionNumber struct{ major, minor int }

func parseMicroversion(value string) (microversionNumber, error) {
	if !microversionPattern.MatchString(value) {
		return microversionNumber{}, invalid("invalid microversion %q", value)
	}
	parts := strings.Split(value, ".")
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	if majorErr != nil || minorErr != nil {
		return microversionNumber{}, invalid("microversion %q is out of range", value)
	}
	return microversionNumber{major, minor}, nil
}

func (v microversionNumber) String() string { return fmt.Sprintf("%d.%d", v.major, v.minor) }
func (v microversionNumber) less(other microversionNumber) bool {
	return v.major < other.major || (v.major == other.major && v.minor < other.minor)
}

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
		var low, high microversionNumber
		for _, bound := range []struct {
			value  string
			target *microversionNumber
		}{{minimum, &low}, {maximum, &high}} {
			if bound.value == "" || (bound.target == &high && bound.value == "latest") {
				continue
			}
			parsed, err := parseMicroversion(bound.value)
			if err != nil {
				return err
			}
			if strconv.Itoa(parsed.major) != major {
				return invalid("microversion %q does not belong to %s", bound.value, service)
			}
			*bound.target = parsed
		}
		if minimum != "" && maximum != "" && maximum != "latest" && high.less(low) {
			return invalid("minimum microversion %s exceeds maximum %s", minimum, maximum)
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
	low, high, discoveryURL, err := discoverMicroversions(ctx, service, client)
	if err != nil {
		return MicroversionSelection{}, fmt.Errorf("discover %s microversions: %w", service, err)
	}
	selection.SupportedMinimum, selection.SupportedMaximum = low.String(), high.String()
	selection.DiscoveryURL = discoveryURL
	if request.minimum != "" {
		requested, _ := parseMicroversion(request.minimum) // validated at construction
		if low.less(requested) {
			low = requested
		}
	}
	if request.maximum != "" && request.maximum != "latest" {
		requested, _ := parseMicroversion(request.maximum)
		if requested.less(high) {
			high = requested
		}
	}
	if high.less(low) {
		return MicroversionSelection{}, unsupported(string(service), fmt.Sprintf("microversion range [%s, %s]; cloud supports [%s, %s]", request.minimum, request.maximum, selection.SupportedMinimum, selection.SupportedMaximum))
	}
	selection.Selected, selection.Negotiated = high.String(), true
	return selection, nil
}
