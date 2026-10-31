// Package endpointdiscovery resolves versioned service endpoints the way
// keystoneauth's get_endpoint does with min_version and max_version: version
// argument normalization including service type aliases, discovery URL
// candidates, version document fetching with one authenticated retry, and
// selection of the newest usable version within the bounds.
package endpointdiscovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Dependencies are the Connection facilities discovery needs. Every field is
// required.
type Dependencies struct {
	// CatalogEndpoint returns the catalog endpoint of the service type
	// without any request; false means no endpoint exists.
	CatalogEndpoint func() (string, bool, error)
	// ProjectID returns the project of the recorded token, as keystoneauth's
	// get_project_id; it is empty without a recorded token result.
	ProjectID func() string
	// UserAgent returns the User-Agent header value of a request.
	UserAgent func() string
	// AuthenticatedHeaders returns the token headers of the retried request.
	AuthenticatedHeaders func() map[string]string
	// Do sends one version document request.
	Do func(*http.Request) (*http.Response, error)
}

// Resolve maps the get_session_endpoint part after the override lookup:
// minimum and maximum are normalized for serviceType, the catalog endpoint is
// looked up, and with version bounds version discovery runs within ctx.
// Without bounds the catalog result is returned unchanged.
func Resolve(ctx context.Context, deps Dependencies, serviceType, minimum, maximum string) (string, bool, error) {
	minVersion, maxVersion, err := normalizeEndpointVersionArgs(minimum, maximum, serviceType)
	if err != nil {
		return "", false, err
	}
	catalogURL, found, err := deps.CatalogEndpoint()
	if err != nil || !found || (minVersion == nil && maxVersion == nil) {
		return catalogURL, found, err
	}
	return discoverVersionedEndpoint(ctx, deps, serviceType, catalogURL, minVersion, maxVersion)
}

// knownServiceTypeAliases lists the aliases of the same data that end in a
// version digit. keystoneauth derives an implied major version from them.
var knownServiceTypeAliases = map[string]bool{
	"volumev2": true, "volumev3": true, "sharev2": true, "workflowv2": true,
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// apiVersion is a keystoneauth normalized version tuple; latestVersion stands
// for LATEST.
type apiVersion []int

const latestVersion = math.MaxInt

func (v apiVersion) compare(other apiVersion) int {
	for i := 0; i < len(v) && i < len(other); i++ {
		if v[i] != other[i] {
			if v[i] < other[i] {
				return -1
			}
			return 1
		}
	}
	return len(v) - len(other)
}

func (v apiVersion) String() string {
	parts := make([]string, len(v))
	for i, part := range v {
		if part == latestVersion {
			parts[i] = "latest"
		} else {
			parts[i] = strconv.Itoa(part)
		}
	}
	return strings.Join(parts, ".")
}

// normalizeAPIVersion mirrors keystoneauth normalize_version_number for string
// input.
func normalizeAPIVersion(version string) (apiVersion, error) {
	text := strings.TrimLeft(version, "v")
	if number, err := strconv.Atoi(text); err == nil {
		text = strconv.Itoa(number) + ".0"
	}
	parts := strings.Split(text, ".")
	if len(parts) == 1 && parts[0] == "latest" {
		return apiVersion{latestVersion, latestVersion}, nil
	}
	out := make(apiVersion, len(parts))
	for i, part := range parts {
		if part == "latest" {
			out[i] = latestVersion
			continue
		}
		number, err := strconv.Atoi(part)
		if err != nil {
			return nil, invalid("invalid version specified: %q", version)
		}
		out[i] = number
	}
	return out, nil
}

// normalizeEndpointVersionArgs mirrors keystoneauth _normalize_version_args
// for min_version/max_version strings and a service type.
func normalizeEndpointVersionArgs(minimum, maximum, serviceType string) (apiVersion, apiVersion, error) {
	var implied apiVersion
	if knownServiceTypeAliases[serviceType] {
		implied, _ = normalizeAPIVersion(serviceType[len(serviceType)-1:])
	}
	if minimum == "latest" {
		if maximum != "" && maximum != "latest" {
			return nil, nil, invalid("min_version is 'latest' and max_version is %q but is only allowed to be 'latest' or empty", maximum)
		}
		maximum = "latest"
	}
	var minVersion, maxVersion apiVersion
	var err error
	if minimum != "" {
		if minVersion, err = normalizeAPIVersion(minimum); err != nil {
			return nil, nil, err
		}
		if maximum == "" {
			maxVersion = apiVersion{latestVersion, latestVersion}
		}
	}
	if maximum != "" {
		if maxVersion, err = normalizeAPIVersion(maximum); err != nil {
			return nil, nil, err
		}
	}
	if minVersion != nil && maxVersion != nil && maxVersion.compare(minVersion) < 0 {
		return nil, nil, invalid("min_version cannot be greater than max_version")
	}
	if implied != nil {
		if minVersion != nil {
			if minVersion[0] != implied[0] {
				return nil, nil, invalid("service type %s implies major version %d but min_version %s was given", serviceType, implied[0], minVersion)
			}
		} else {
			minVersion = implied
		}
		if maxVersion != nil && maxVersion[0] != latestVersion {
			if maxVersion[0] != implied[0] {
				return nil, nil, invalid("service type %s implies major version %d but max_version %s was given", serviceType, implied[0], maxVersion)
			}
		} else {
			maxVersion = apiVersion{implied[0], latestVersion}
		}
	}
	return minVersion, maxVersion, nil
}

func apiVersionBetween(minimum, maximum, candidate apiVersion) bool {
	if minimum != nil && candidate.compare(minimum) < 0 {
		return false
	}
	return maximum == nil || candidate.compare(maximum) <= 0
}

func apiVersionMatch(required, candidate apiVersion) bool {
	return candidate[0] == required[0] && candidate.compare(required) >= 0
}

func apiLatestSoftMatch(required, candidate apiVersion) bool {
	if required == nil || !slices.Contains(required, latestVersion) {
		return false
	}
	if !slices.ContainsFunc(required, func(part int) bool { return part != latestVersion }) {
		return true
	}
	return required[0] == candidate[0] && required[1] == latestVersion
}

var identityDiscoveryHack = regexp.MustCompile(`/v2.0/?$`)

// discoveryChoices mirrors EndpointData._get_discovery_url_choices with the
// version hack allowed. exact reports that the catalog URL already satisfies
// the request, in which case keystoneauth performs no discovery.
type discoveryChoices struct {
	urls           []string
	savedProjectID string
	exact          bool
}

func endpointDiscoveryChoices(serviceType, catalogURL, projectID string, minimum, maximum apiVersion) (discoveryChoices, error) {
	var out discoveryChoices
	u, err := url.Parse(strings.TrimRight(catalogURL, "/"))
	if err != nil {
		return out, fmt.Errorf("parse catalog endpoint: %w", err)
	}
	parts := strings.Split(u.Path, "/")
	if projectID != "" && parts[len(parts)-1] == projectID {
		out.savedProjectID, parts = parts[len(parts)-1], parts[:len(parts)-1]
	} else if projectID == "" && len(parts) >= 2 {
		if _, err := normalizeAPIVersion(parts[len(parts)-2]); err == nil {
			out.savedProjectID, parts = parts[len(parts)-1], parts[:len(parts)-1]
		}
	}
	build := func(parts []string) string {
		copyURL := *u
		copyURL.Path, copyURL.RawPath = strings.Join(parts, "/"), ""
		return copyURL.String()
	}
	catalogDiscovery := ""
	if len(parts) > 0 {
		if urlVersion, err := normalizeAPIVersion(parts[len(parts)-1]); err == nil {
			between := minimum != nil && maximum != nil && apiVersionBetween(minimum, maximum, urlVersion)
			exact := between && maximum[0] == urlVersion[0]
			high := between && maximum[1] != latestVersion && apiVersionMatch(maximum, urlVersion)
			if between {
				out.exact = exact
				catalogDiscovery = strings.TrimRight(build(parts), "/") + "/"
			}
			if catalogDiscovery != "" && (high || exact) {
				out.urls = append(out.urls, catalogDiscovery)
				catalogDiscovery = ""
			}
			parts = parts[:len(parts)-1]
		}
	}
	hacked := build(parts)
	if hacked != catalogURL {
		hacked = strings.Trim(hacked, "/") + "/"
	}
	out.urls = append(out.urls, hacked)
	if catalogDiscovery != "" {
		out.urls = append(out.urls, catalogDiscovery)
	}
	if serviceType == "identity" && identityDiscoveryHack.MatchString(catalogURL) {
		out.urls = append(out.urls, identityDiscoveryHack.ReplaceAllString(catalogURL, "/"))
	} else {
		out.urls = append(out.urls, catalogURL)
	}
	out.urls = append(out.urls, catalogURL)
	return out, nil
}

func discoverVersionedEndpoint(ctx context.Context, deps Dependencies, serviceType, catalogURL string, minimum, maximum apiVersion) (string, bool, error) {
	choices, err := endpointDiscoveryChoices(serviceType, catalogURL, deps.ProjectID(), minimum, maximum)
	if err != nil {
		return "", false, err
	}
	if choices.exact {
		return catalogURL, true, nil
	}
	tried := make(map[string]bool)
	for _, candidate := range choices.urls {
		if tried[candidate] {
			continue
		}
		tried[candidate] = true
		entries, err := fetchVersionDocument(ctx, deps, candidate)
		if err != nil {
			if cause := ctx.Err(); cause != nil {
				return "", false, errors.Join(err, cause)
			}
			continue
		}
		versions, err := discoveredVersions(candidate, entries)
		if err != nil {
			return "", false, fmt.Errorf("version discovery at %s: %w", candidate, err)
		}
		selected, ok, err := selectDiscoveredVersion(versions, minimum, maximum)
		if err != nil || !ok {
			return "", false, err
		}
		endpoint := selected.url
		if choices.savedProjectID != "" {
			base, err := url.Parse(strings.TrimRight(endpoint, "/") + "/")
			if err != nil {
				return "", false, err
			}
			endpoint = base.ResolveReference(&url.URL{Path: choices.savedProjectID}).String()
		}
		return endpoint, true, nil
	}
	// No document anywhere: keystoneauth falls back to the catalog URL.
	return catalogURL, true, nil
}

// errNoVersionDocument marks a response that is not a version document, which
// keystoneauth treats like an unreachable URL.
var errNoVersionDocument = errors.New("response is not a version discovery document")

// fetchVersionDocument mirrors keystoneauth get_version_data: an
// unauthenticated GET, retried once with the token after a 401. Any status
// of 400 or more, a transport failure or a body that is not a version
// document lets discovery continue with the next URL.
func fetchVersionDocument(ctx context.Context, deps Dependencies, target string) (any, error) {
	response, err := versionDocumentRequest(ctx, deps, target, false)
	if err == nil && response.StatusCode == http.StatusUnauthorized {
		response.Body.Close()
		response, err = versionDocumentRequest(ctx, deps, target, true)
	}
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("version discovery at %s: HTTP %d", target, response.StatusCode)
	}
	var body any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if decoder.Decode(&body) != nil {
		return nil, errNoVersionDocument
	}
	object, ok := body.(map[string]any)
	if !ok {
		return nil, errNoVersionDocument
	}
	if versions, ok := object["versions"].(map[string]any); ok {
		if values, present := versions["values"]; present {
			return values, nil
		}
	}
	if versions, present := object["versions"]; present {
		return versions, nil
	}
	if version, present := object["version"]; present {
		return []any{version}, nil
	}
	if _, present := object["id"]; present {
		// Older Ironic: a bare version object with microversions in headers.
		object["status"] = "CURRENT"
		for name, values := range response.Header {
			lower := strings.ToLower(name)
			if !strings.HasPrefix(lower, "x-openstack") || len(values) == 0 {
				continue
			}
			if _, set := object["min_version"]; !set && strings.HasSuffix(lower, "api-minimum-version") {
				object["min_version"] = values[0]
			}
			if _, set := object["version"]; !set && strings.HasSuffix(lower, "api-maximum-version") {
				object["version"] = values[0]
			}
		}
		return []any{object}, nil
	}
	return nil, errNoVersionDocument
}

func versionDocumentRequest(ctx context.Context, deps Dependencies, target string, authenticated bool) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", deps.UserAgent())
	if authenticated {
		for name, value := range deps.AuthenticatedHeaders() {
			request.Header.Set(name, value)
		}
	}
	return deps.Do(request)
}

type discoveredVersion struct {
	version apiVersion
	url     string
}

// discoveredVersions mirrors Discover.version_data(reverse=True) with the
// default filters: CURRENT, STABLE, SUPPORTED and DEPRECATED entries with an
// ID, links and a self link, newest first. Malformed data that keystoneauth
// fails on outside its discovery error handling is an error.
func discoveredVersions(discoveryURL string, document any) ([]discoveredVersion, error) {
	entries, ok := document.([]any)
	if !ok {
		return nil, fmt.Errorf("version list must be a JSON list, got %T", document)
	}
	var out []discoveredVersion
	for _, raw := range entries {
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("version entry must be an object, got %T", raw)
		}
		rawStatus, present := entry["status"]
		if !present {
			continue
		}
		status, ok := rawStatus.(string)
		if !ok {
			return nil, fmt.Errorf("version status must be a string, got %T", rawStatus)
		}
		switch strings.ToLower(status) {
		case "stable", "current", "supported", "deprecated":
		default:
			continue
		}
		rawID, hasID := entry["id"]
		rawLinks, hasLinks := entry["links"]
		if !hasID || !hasLinks {
			continue
		}
		id, ok := rawID.(string)
		if !ok {
			return nil, fmt.Errorf("version id must be a string, got %T", rawID)
		}
		version, err := normalizeAPIVersion(id)
		if err != nil {
			return nil, err
		}
		links, ok := rawLinks.([]any)
		if !ok {
			return nil, fmt.Errorf("version links must be a list, got %T", rawLinks)
		}
		self := ""
		for _, rawLink := range links {
			link, ok := rawLink.(map[string]any)
			if !ok {
				continue
			}
			rel, relOK := link["rel"].(string)
			href, hrefOK := link["href"].(string)
			if !relOK || !hrefOK {
				continue
			}
			combined, err := combineDiscoveryURL(discoveryURL, href)
			if err != nil {
				continue
			}
			if strings.ToLower(rel) == "self" {
				self = combined
			}
		}
		if self == "" {
			continue
		}
		out = append(out, discoveredVersion{version: version, url: self})
	}
	slices.SortStableFunc(out, func(a, b discoveredVersion) int { return b.version.compare(a.version) })
	return out, nil
}

// combineDiscoveryURL mirrors keystoneauth _combine_relative_url, including its
// repair of version links whose host differs from the discovery URL.
func combineDiscoveryURL(discoveryURL, versionURL string) (string, error) {
	base, err := url.Parse(strings.TrimRight(discoveryURL, "/") + "/")
	if err != nil {
		return "", err
	}
	reference, err := url.Parse(versionURL)
	if err != nil {
		return "", err
	}
	joined := base.ResolveReference(reference)
	discovery, err := url.Parse(discoveryURL)
	if err != nil {
		return "", err
	}
	path := joined.Path
	if discovery.Host != joined.Host {
		segments := strings.Split(strings.TrimRight(versionURL, "/"), "/")
		version := segments[len(segments)-1]
		urlPath := strings.TrimRight(discovery.Path, "/")
		if !strings.HasSuffix(urlPath, version) {
			path = urlPath + "/" + version
			if strings.HasSuffix(versionURL, "/") {
				path += "/"
			}
		} else {
			path = discovery.Path
		}
	}
	result := url.URL{Scheme: discovery.Scheme, User: discovery.User, Host: discovery.Host, Path: path, RawQuery: joined.RawQuery, Fragment: joined.Fragment}
	return result.String(), nil
}

// selectDiscoveredVersion mirrors Discover.versioned_data_for with bounds.
func selectDiscoveredVersion(versions []discoveredVersion, minimum, maximum apiVersion) (discoveredVersion, bool, error) {
	latest := apiVersion{latestVersion, latestVersion}
	if maximum != nil && maximum.compare(latest) == 0 && (minimum == nil || minimum.compare(latest) == 0) {
		if len(versions) == 0 {
			return discoveredVersion{}, false, invalid("version discovery document lists no usable version")
		}
		return versions[0], true, nil
	}
	for _, candidate := range versions {
		if apiLatestSoftMatch(minimum, candidate.version) {
			return candidate, true, nil
		}
		if minimum != nil && maximum != nil && apiVersionBetween(minimum, maximum, candidate.version) {
			return candidate, true, nil
		}
	}
	return discoveredVersion{}, false, nil
}
