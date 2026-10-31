package openstack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/utils"
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

type discoveryVersion struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Version string `json:"version"`
	Maximum string `json:"max_version"`
	Minimum string `json:"min_version"`
}

type discoveryDocument struct{ versions []discoveryVersion }

func (d *discoveryDocument) UnmarshalJSON(data []byte) error {
	var envelope struct {
		ID       string          `json:"id"`
		Version  json.RawMessage `json:"version"`
		Versions json.RawMessage `json:"versions"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	if envelope.ID != "" {
		var version discoveryVersion
		if err := json.Unmarshal(data, &version); err != nil {
			return err
		}
		d.versions = []discoveryVersion{version}
		return nil
	}
	if len(envelope.Version) > 0 && envelope.Version[0] == '{' {
		var version discoveryVersion
		if err := json.Unmarshal(envelope.Version, &version); err != nil {
			return err
		}
		d.versions = []discoveryVersion{version}
		return nil
	}
	if len(envelope.Versions) > 0 {
		if envelope.Versions[0] == '[' {
			return json.Unmarshal(envelope.Versions, &d.versions)
		}
		var nested struct {
			Values []discoveryVersion `json:"values"`
		}
		if err := json.Unmarshal(envelope.Versions, &nested); err != nil {
			return err
		}
		d.versions = nested.Values
		return nil
	}
	return fmt.Errorf("response does not contain a version discovery document")
}

var discoveryPathVersion = regexp.MustCompile(`^v[0-9]+(?:\.[0-9]+)?$`)

// Project-scoped Nova, Cinder, Manila and Masakari catalog URLs point beneath the
// version discovery document. Remove only the optional last project segment
// and the version segment, keeping reverse-proxy prefixes intact. Other
// services publish discovery directly at a versioned or unversioned endpoint.
func microversionDiscoveryURLs(service Service, endpoint string) ([]string, *microversionNumber, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, nil, invalid("discovery endpoint must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	parts := strings.Split(strings.TrimRight(u.EscapedPath(), "/"), "/")
	index := len(parts) - 1
	if index >= 1 && !discoveryPathVersion.MatchString(parts[index]) && discoveryPathVersion.MatchString(parts[index-1]) {
		switch service {
		case Compute, BlockStorage, SharedFileSystem, InstanceHA:
			index--
		}
	}
	if index < 0 || !discoveryPathVersion.MatchString(parts[index]) {
		return []string{gophercloud.NormalizeURL(endpoint)}, nil, nil
	}
	major, minor, err := utils.ParseVersion(parts[index])
	if err != nil {
		return nil, nil, err
	}
	if strconv.Itoa(major) != serviceDefinitions[service].microversionMajor {
		return nil, nil, unsupported(string(service), "microversions for endpoint "+endpoint)
	}
	endpointVersion := &microversionNumber{major, minor}
	build := func(length int) string {
		copyURL := *u
		copyURL.RawPath = strings.Join(parts[:length], "/") + "/"
		copyURL.Path, _ = url.PathUnescape(copyURL.RawPath)
		return copyURL.String()
	}
	return []string{build(index + 1), build(index)}, endpointVersion, nil
}

func discoverMicroversions(ctx context.Context, service Service, client *gophercloud.ServiceClient) (microversionNumber, microversionNumber, string, error) {
	urls, endpointVersion, err := microversionDiscoveryURLs(service, client.Endpoint)
	if err != nil {
		return microversionNumber{}, microversionNumber{}, "", err
	}
	major, _ := strconv.Atoi(serviceDefinitions[service].microversionMajor)
	var lastErr error
	for _, discoveryURL := range urls {
		if err := ctx.Err(); err != nil {
			return microversionNumber{}, microversionNumber{}, "", err
		}
		var document discoveryDocument
		_, err := client.ProviderClient.Request(ctx, "GET", discoveryURL, &gophercloud.RequestOpts{
			JSONResponse: &document, OkCodes: []int{200, 300},
		})
		if err != nil {
			var responseErr gophercloud.ErrUnexpectedResponseCode
			if errors.As(err, &responseErr) && (responseErr.Actual == 404 || responseErr.Actual == 405) {
				lastErr = err
				continue
			}
			return microversionNumber{}, microversionNumber{}, "", err
		}
		low, high, found, err := document.bounds(major, endpointVersion)
		if err != nil {
			return microversionNumber{}, microversionNumber{}, "", err
		}
		if found {
			return low, high, discoveryURL, nil
		}
		lastErr = unsupported(string(service), "advertised microversion bounds at "+discoveryURL)
	}
	return microversionNumber{}, microversionNumber{}, "", lastErr
}

func (d discoveryDocument) bounds(major int, endpointVersion *microversionNumber) (microversionNumber, microversionNumber, bool, error) {
	var low, high microversionNumber
	found := false
	for _, version := range d.versions {
		apiMajor, apiMinor, err := utils.ParseVersion(version.ID)
		if err != nil {
			return low, high, false, fmt.Errorf("invalid discovery version ID: %w", err)
		}
		if apiMajor != major || (endpointVersion != nil && apiMinor != endpointVersion.minor) {
			continue
		}
		switch strings.ToUpper(version.Status) {
		case "", "CURRENT", "SUPPORTED", "STABLE", "DEPRECATED":
		default:
			continue
		}
		maximum := version.Maximum
		if maximum == "" {
			maximum = version.Version
		}
		if maximum == "" || version.Minimum == "" {
			continue
		}
		minimumNumber, err := parseMicroversion(version.Minimum)
		if err != nil {
			return low, high, false, fmt.Errorf("invalid advertised minimum: %w", err)
		}
		maximumNumber, err := parseMicroversion(maximum)
		if err != nil {
			return low, high, false, fmt.Errorf("invalid advertised maximum: %w", err)
		}
		if minimumNumber.major != major || maximumNumber.major != major || maximumNumber.less(minimumNumber) {
			return low, high, false, fmt.Errorf("invalid advertised microversion range [%s, %s] for API major %d", version.Minimum, maximum, major)
		}
		if found && (low != minimumNumber || high != maximumNumber) {
			return low, high, false, fmt.Errorf("conflicting advertised microversion ranges for API major %d", major)
		}
		low, high, found = minimumNumber, maximumNumber, true
	}
	return low, high, found, nil
}
