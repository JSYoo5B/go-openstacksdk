package gophercloudsdk

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

// Project-scoped Nova, Cinder and Manila catalog URLs point beneath the
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
		case Compute, BlockStorage, SharedFileSystem:
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
