package microversions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/utils"
)

// Connection-level negotiation uses a strict major.minor form. It is kept
// separate from Version, whose Python-compatible parsing accepts a leading v,
// latest components and arbitrarily long integer tuples.

// numberPattern is the accepted Connection microversion spelling.
var numberPattern = regexp.MustCompile(`^[1-9][0-9]*\.[0-9]+$`)

// Number is a finite major.minor microversion.
type Number struct{ Major, Minor int }

// ParseNumber accepts only major.minor values whose components fit an int.
func ParseNumber(value string) (Number, error) {
	if !numberPattern.MatchString(value) {
		return Number{}, invalid("invalid microversion %q", value)
	}
	parts := strings.Split(value, ".")
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	if majorErr != nil || minorErr != nil {
		return Number{}, invalid("microversion %q is out of range", value)
	}
	return Number{major, minor}, nil
}

func (v Number) String() string { return fmt.Sprintf("%d.%d", v.Major, v.Minor) }

// Less reports whether v orders strictly below other.
func (v Number) Less(other Number) bool {
	return v.Major < other.Major || (v.Major == other.Major && v.Minor < other.Minor)
}

// ValidateRange checks an inclusive requested range for a service whose
// microversions use the given API major. Empty minimum means the cloud
// minimum; empty maximum or "latest" means the cloud maximum.
func ValidateRange(service, major, minimum, maximum string) error {
	var low, high Number
	for _, bound := range []struct {
		value  string
		target *Number
	}{{minimum, &low}, {maximum, &high}} {
		if bound.value == "" || (bound.target == &high && bound.value == "latest") {
			continue
		}
		parsed, err := ParseNumber(bound.value)
		if err != nil {
			return err
		}
		if strconv.Itoa(parsed.Major) != major {
			return invalid("microversion %q does not belong to %s", bound.value, service)
		}
		*bound.target = parsed
	}
	if minimum != "" && maximum != "" && maximum != "latest" && high.Less(low) {
		return invalid("minimum microversion %s exceeds maximum %s", minimum, maximum)
	}
	return nil
}

// Clamp narrows advertised bounds to a range accepted by ValidateRange and
// returns the highest shared version. It reports false when they are disjoint.
func Clamp(low, high Number, minimum, maximum string) (Number, bool) {
	if minimum != "" {
		requested, _ := ParseNumber(minimum) // validated at construction
		if low.Less(requested) {
			low = requested
		}
	}
	if maximum != "" && maximum != "latest" {
		requested, _ := ParseNumber(maximum)
		if requested.Less(high) {
			high = requested
		}
	}
	if high.Less(low) {
		return Number{}, false
	}
	return high, true
}

// Target identifies the service whose shared client is negotiated.
// StripProject applies to services whose project-scoped catalog URLs point
// beneath the version discovery document.
type Target struct {
	Service      string
	Major        string
	StripProject bool
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

var negotiationPathVersion = regexp.MustCompile(`^v[0-9]+(?:\.[0-9]+)?$`)

// IsPathVersion reports whether an escaped catalog path segment is a vN or
// vN.M API version segment.
func IsPathVersion(segment string) bool { return negotiationPathVersion.MatchString(segment) }

// Project-scoped Nova, Cinder, Manila and Masakari catalog URLs point beneath the
// version discovery document. Remove only the optional last project segment
// and the version segment, keeping reverse-proxy prefixes intact. Other
// services publish discovery directly at a versioned or unversioned endpoint.
func negotiationURLs(target Target, endpoint string) ([]string, *Number, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, nil, invalid("discovery endpoint must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	parts := strings.Split(strings.TrimRight(u.EscapedPath(), "/"), "/")
	index := len(parts) - 1
	if index >= 1 && !negotiationPathVersion.MatchString(parts[index]) && negotiationPathVersion.MatchString(parts[index-1]) {
		if target.StripProject {
			index--
		}
	}
	if index < 0 || !negotiationPathVersion.MatchString(parts[index]) {
		return []string{gophercloud.NormalizeURL(endpoint)}, nil, nil
	}
	major, minor, err := utils.ParseVersion(parts[index])
	if err != nil {
		return nil, nil, err
	}
	if strconv.Itoa(major) != target.Major {
		return nil, nil, unsupported(target.Service, "microversions for endpoint "+endpoint)
	}
	endpointVersion := &Number{major, minor}
	build := func(length int) string {
		copyURL := *u
		copyURL.RawPath = strings.Join(parts[:length], "/") + "/"
		copyURL.Path, _ = url.PathUnescape(copyURL.RawPath)
		return copyURL.String()
	}
	return []string{build(index + 1), build(index)}, endpointVersion, nil
}

// Discover reads advertised bounds through the shared client's provider.
// Only 404 or 405 at the version endpoint permits the root fallback.
func Discover(ctx context.Context, target Target, client *gophercloud.ServiceClient) (Number, Number, string, error) {
	urls, endpointVersion, err := negotiationURLs(target, client.Endpoint)
	if err != nil {
		return Number{}, Number{}, "", err
	}
	major, _ := strconv.Atoi(target.Major)
	var lastErr error
	for _, discoveryURL := range urls {
		if err := ctx.Err(); err != nil {
			return Number{}, Number{}, "", err
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
			return Number{}, Number{}, "", err
		}
		low, high, found, err := document.bounds(major, endpointVersion)
		if err != nil {
			return Number{}, Number{}, "", err
		}
		if found {
			return low, high, discoveryURL, nil
		}
		lastErr = unsupported(target.Service, "advertised microversion bounds at "+discoveryURL)
	}
	return Number{}, Number{}, "", lastErr
}

func (d discoveryDocument) bounds(major int, endpointVersion *Number) (Number, Number, bool, error) {
	var low, high Number
	found := false
	for _, version := range d.versions {
		apiMajor, apiMinor, err := utils.ParseVersion(version.ID)
		if err != nil {
			return low, high, false, fmt.Errorf("invalid discovery version ID: %w", err)
		}
		if apiMajor != major || (endpointVersion != nil && apiMinor != endpointVersion.Minor) {
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
		minimumNumber, err := ParseNumber(version.Minimum)
		if err != nil {
			return low, high, false, fmt.Errorf("invalid advertised minimum: %w", err)
		}
		maximumNumber, err := ParseNumber(maximum)
		if err != nil {
			return low, high, false, fmt.Errorf("invalid advertised maximum: %w", err)
		}
		if minimumNumber.Major != major || maximumNumber.Major != major || maximumNumber.Less(minimumNumber) {
			return low, high, false, fmt.Errorf("invalid advertised microversion range [%s, %s] for API major %d", version.Minimum, maximum, major)
		}
		if found && (low != minimumNumber || high != maximumNumber) {
			return low, high, false, fmt.Errorf("conflicting advertised microversion ranges for API major %d", major)
		}
		low, high, found = minimumNumber, maximumNumber, true
	}
	return low, high, found, nil
}

func unsupported(service, feature string) error {
	return fmt.Errorf("%w: %s does not support %s", resource.ErrUnsupported, service, feature)
}
