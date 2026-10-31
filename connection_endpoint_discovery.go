package openstack

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/endpointdiscovery"
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
	return endpointdiscovery.Resolve(ctx, c.endpointDiscoveryDependencies(serviceType, q), serviceType, q.minVersion, q.maxVersion)
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

// endpointDiscoveryDependencies binds version discovery to the catalog lookup
// of one query and to the shared provider's token, user agent and HTTP client.
func (c *Connection) endpointDiscoveryDependencies(serviceType string, q endpointQuery) endpointdiscovery.Dependencies {
	return endpointdiscovery.Dependencies{
		CatalogEndpoint:      func() (string, bool, error) { return c.catalogEndpoint(serviceType, q) },
		ProjectID:            c.tokenProjectID,
		UserAgent:            func() string { return c.provider.UserAgent.Join() },
		AuthenticatedHeaders: c.provider.AuthenticatedHeaders,
		Do: func(request *http.Request) (*http.Response, error) {
			client := c.provider.HTTPClient
			return client.Do(request)
		},
	}
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
