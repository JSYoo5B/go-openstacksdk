package openstack

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

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

// EndpointOption refines EndpointFor, SessionEndpoint and HasService.
type EndpointOption func(*endpointQuery) error

type endpointQuery struct {
	availability gophercloud.Availability
	region       *string
	serviceName  string
	// minVersion and maxVersion request version discovery; they are only
	// accepted by SessionEndpointContext and HasServiceVersion.
	minVersion, maxVersion string
}

// WithEndpointInterface selects the catalog interface instead of the
// Connection's interface. A trailing "URL" is ignored like in keystoneauth.
func WithEndpointInterface(availability gophercloud.Availability) EndpointOption {
	return func(q *endpointQuery) error {
		if availability == "" {
			return invalid("endpoint interface must not be empty")
		}
		q.availability = availability
		return nil
	}
}

// WithEndpointRegion selects a region instead of the Connection's region. The
// empty string matches every region.
func WithEndpointRegion(region string) EndpointOption {
	return func(q *endpointQuery) error { q.region = &region; return nil }
}

// WithEndpointServiceName additionally requires the catalog service name.
func WithEndpointServiceName(name string) EndpointOption {
	return func(q *endpointQuery) error { q.serviceName = name; return nil }
}

func (c *Connection) endpointQuery(options []EndpointOption) (endpointQuery, error) {
	q := endpointQuery{availability: c.endpointOptions.Availability}
	if q.availability == "" {
		q.availability = gophercloud.AvailabilityPublic
	}
	region := c.endpointOptions.Region
	q.region = &region
	for _, apply := range options {
		if apply == nil {
			return q, invalid("nil endpoint option")
		}
		if err := apply(&q); err != nil {
			return q, err
		}
	}
	q.availability = gophercloud.Availability(strings.TrimSuffix(string(q.availability), "URL"))
	return q, nil
}

// catalogURL mirrors keystoneauth url_for: the first endpoint, in catalog
// order, of the first matching service entry whose interface and region match.
func catalogURL(catalog []CatalogService, serviceType string, q endpointQuery) (string, bool) {
	for _, service := range catalog {
		if service.Type != serviceType || (q.serviceName != "" && service.Name != q.serviceName) {
			continue
		}
		for _, endpoint := range service.Endpoints {
			if strings.TrimSuffix(endpoint.Interface, "URL") != string(q.availability) || endpoint.URL == "" {
				continue
			}
			if *q.region != "" && endpoint.Region != *q.region && endpoint.RegionID != *q.region {
				continue
			}
			return endpoint.URL, true
		}
	}
	return "", false
}

// catalogEndpoint looks the type up in the recorded token catalog without any
// HTTP request. Without a recorded token result it falls back to the
// provider's endpoint locator, whose absence error maps to not found.
func (c *Connection) catalogEndpoint(serviceType string, q endpointQuery) (string, bool, error) {
	catalog, err := c.ServiceCatalog()
	if err == nil {
		url, ok := catalogURL(catalog, serviceType, q)
		return url, ok, nil
	}
	if !errors.Is(err, ErrNoAuthResult) || c.provider.EndpointLocator == nil {
		return "", false, err
	}
	url, locateErr := c.provider.EndpointLocator(gophercloud.EndpointOpts{Type: serviceType, Name: q.serviceName, Region: *q.region, Availability: q.availability})
	if locateErr != nil {
		if missingCatalogEndpoint(locateErr) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("endpoint lookup for %s: %w", serviceType, locateErr)
	}
	return url, url != "", nil
}

// EndpointFor returns the endpoint for a service type string, honoring a
// WithEndpoint override of that type before consulting the catalog. The bool is
// false when neither an override nor a catalog endpoint exists. No request is
// made and no context is needed. Catalog matching is exact on the type; os-service-types
// aliases (for example volumev3) are not resolved. Defaults come from the
// Connection's region and interface; options override them for one call.
func (c *Connection) EndpointFor(serviceType string, options ...EndpointOption) (string, bool, error) {
	if c == nil || c.provider == nil {
		return "", false, invalid("Connection is required")
	}
	if serviceType == "" {
		return "", false, invalid("service type must not be empty")
	}
	q, err := c.endpointQuery(options)
	if err != nil {
		return "", false, err
	}
	if q.minVersion != "" || q.maxVersion != "" {
		return "", false, invalid("endpoint version bounds require SessionEndpointContext")
	}
	c.mu.Lock()
	override := c.options.endpoints[Service(serviceType)]
	c.mu.Unlock()
	if override != "" {
		return override, true, nil
	}
	return c.catalogEndpoint(serviceType, q)
}

// SessionEndpoint is the Go counterpart of get_session_endpoint without
// version bounds: the override or the catalog endpoint of the service type,
// selected by the Connection's region and interface or the options (including
// the catalog service name). The bool is false when no endpoint exists. Version
// discovery with min_version/max_version is not performed.
func (c *Connection) SessionEndpoint(serviceType string, options ...EndpointOption) (string, bool, error) {
	return c.EndpointFor(serviceType, options...)
}

// HasService reports whether the service is enabled by configuration
// (ServiceEnabled: the has_<service> flag, else an official service type) and
// SessionEndpoint finds an endpoint. A disabled service is (false, nil) without
// a lookup and a missing endpoint is (false, nil); unlike Python, lookup
// failures such as a missing recorded catalog are returned as errors instead of
// false. It performs no version discovery; HasServiceVersion does.
func (c *Connection) HasService(serviceType string, options ...EndpointOption) (bool, error) {
	if c != nil && serviceType != "" && !c.ServiceEnabled(serviceType) {
		return false, nil
	}
	_, ok, err := c.SessionEndpoint(serviceType, options...)
	return ok, err
}

// WithEndpointMinVersion requests version discovery with a minimum API
// version, as get_session_endpoint's min_version. Versions use keystoneauth
// notation: "2", "v2.1", "3.latest" or "latest". The empty string means unset.
// Only SessionEndpointContext accepts version bounds.
func WithEndpointMinVersion(version string) EndpointOption {
	return func(q *endpointQuery) error { q.minVersion = version; return nil }
}

// WithEndpointMaxVersion requests version discovery with a maximum API version,
// as get_session_endpoint's max_version. Only SessionEndpointContext accepts it.
func WithEndpointMaxVersion(version string) EndpointOption {
	return func(q *endpointQuery) error { q.maxVersion = version; return nil }
}

// SessionEndpointContext maps get_session_endpoint including min_version and
// max_version. A WithEndpoint override is returned without discovery. Without
// version bounds (and for service types that imply none) it equals
// SessionEndpoint. With bounds, the catalog endpoint is selected as in
// SessionEndpoint and keystoneauth's version discovery runs within ctx:
//
//   - a catalog URL whose path version lies within the bounds and has the
//     maximum's major version is returned without any request;
//   - otherwise version documents are fetched without a token (a 401 is retried
//     once with the current token) from the unversioned URL, the catalog's
//     versioned URL, the identity /v2.0 hack and the catalog URL, in that
//     order, until one answers with a version document;
//   - the newest CURRENT, SUPPORTED, STABLE or DEPRECATED version within the
//     bounds is returned as its self link, with a project ID removed from the
//     catalog URL appended again; when no document matches, found is false;
//   - when no URL answers with a document, the catalog URL is returned.
//
// Aliases ending in a version (volumev2, volumev3, sharev2, workflowv2) imply
// that major version, as in keystoneauth. Invalid or contradictory versions are
// errors where Python raises. Discovery results are not cached and
// cancellation stops discovery with an error.
func (c *Connection) SessionEndpointContext(ctx context.Context, serviceType string, options ...EndpointOption) (string, bool, error) {
	if c == nil || c.provider == nil {
		return "", false, invalid("Connection is required")
	}
	if serviceType == "" {
		return "", false, invalid("service type must not be empty")
	}
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	q, err := c.endpointQuery(options)
	if err != nil {
		return "", false, err
	}
	c.mu.Lock()
	override := c.options.endpoints[Service(serviceType)]
	c.mu.Unlock()
	if override != "" {
		return override, true, nil
	}
	minVersion, maxVersion, err := normalizeEndpointVersionArgs(q.minVersion, q.maxVersion, serviceType)
	if err != nil {
		return "", false, err
	}
	catalogURL, found, err := c.catalogEndpoint(serviceType, q)
	if err != nil || !found || (minVersion == nil && maxVersion == nil) {
		return catalogURL, found, err
	}
	return c.discoverVersionedEndpoint(ctx, serviceType, catalogURL, minVersion, maxVersion)
}

// HasServiceVersion maps has_service(service_key, version). A service disabled
// by ServiceEnabled is (false, nil) without a lookup. A nonempty version asks
// SessionEndpointContext for that minimum and "<major>.latest" as maximum, as
// Python does; the empty string means no version. Lookup failures and invalid
// versions are returned as errors where Python returns False.
func (c *Connection) HasServiceVersion(ctx context.Context, serviceType, version string, options ...EndpointOption) (bool, error) {
	if c == nil || c.provider == nil {
		return false, invalid("Connection is required")
	}
	if serviceType == "" {
		return false, invalid("service type must not be empty")
	}
	if !c.ServiceEnabled(serviceType) {
		return false, nil
	}
	options = slices.Clone(options)
	if version != "" {
		major := strings.SplitN(strings.TrimLeft(version, "v"), ".", 2)[0]
		options = append(options, WithEndpointMinVersion(version), WithEndpointMaxVersion(major+".latest"))
	}
	_, found, err := c.SessionEndpointContext(ctx, serviceType, options...)
	return found, err
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

// tokenProjectID returns the project of the recorded token, as keystoneauth's
// get_project_id; it is empty without a recorded token result.
func (c *Connection) tokenProjectID() string {
	r, ok := c.provider.GetAuthResult().(interface {
		ExtractProject() (*tokens.Project, error)
	})
	if !ok {
		return ""
	}
	project, err := r.ExtractProject()
	if err != nil || project == nil {
		return ""
	}
	return project.ID
}

func (c *Connection) discoverVersionedEndpoint(ctx context.Context, serviceType, catalogURL string, minimum, maximum apiVersion) (string, bool, error) {
	choices, err := endpointDiscoveryChoices(serviceType, catalogURL, c.tokenProjectID(), minimum, maximum)
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
		entries, err := c.fetchVersionDocument(ctx, candidate)
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
func (c *Connection) fetchVersionDocument(ctx context.Context, target string) (any, error) {
	response, err := c.versionDocumentRequest(ctx, target, false)
	if err == nil && response.StatusCode == http.StatusUnauthorized {
		response.Body.Close()
		response, err = c.versionDocumentRequest(ctx, target, true)
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

func (c *Connection) versionDocumentRequest(ctx context.Context, target string, authenticated bool) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", c.provider.UserAgent.Join())
	if authenticated {
		for name, value := range c.provider.AuthenticatedHeaders() {
			request.Header.Set(name, value)
		}
	}
	client := c.provider.HTTPClient
	return client.Do(request)
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
