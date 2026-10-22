package openstack

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/gophercloud/gophercloud/v2"
)

// globalRequestIDHeader is the header keystoneauth sends for a global request ID.
const globalRequestIDHeader = "X-OpenStack-Request-Id"

// ErrNoRecordedAuth reports that a Connection was adopted with FromProvider (or
// derived from such a Connection), so no authentication options are recorded
// and ConnectAs cannot repeat authentication with overrides.
var ErrNoRecordedAuth = errors.New("connection has no recorded authentication options")

// connectionMisc holds the state of the Connection-level helpers that are not
// part of the service proxy cache. The zero value is ready to use.
type connectionMisc struct {
	// auth is the resolved authentication request of Connect. It is written
	// once before the Connection is published and never mutated.
	auth *gophercloud.AuthOptions
	// globalRequestID is fixed at derivation time and never mutated, so
	// concurrent requests cannot observe a half-applied change.
	globalRequestID string

	mu      sync.Mutex
	custom  map[Service]*customService
	clients map[Service]*gophercloud.ServiceClient
}

type customService struct {
	factory  ServiceFactory
	endpoint string
}

// decorate applies connection-wide request headers to a freshly built client.
// It is called while the Connection mutex is held, before the client is shared.
func (m *connectionMisc) decorate(client *gophercloud.ServiceClient) {
	if m.globalRequestID == "" {
		return
	}
	headers := maps.Clone(client.MoreHeaders)
	if headers == nil {
		headers = make(map[string]string, 1)
	}
	headers[globalRequestIDHeader] = m.globalRequestID
	client.MoreHeaders = headers
}

func cloneAuthOptions(auth gophercloud.AuthOptions) gophercloud.AuthOptions {
	if auth.Scope != nil {
		scope := *auth.Scope
		auth.Scope = &scope
	}
	return auth
}

// cloneConnectionOptions deep-copies the option maps and slices and resets the
// recorded microversion selections, so a derived Connection negotiates again.
func cloneConnectionOptions(o connectionOptions) connectionOptions {
	o.cloudFiles = slices.Clone(o.cloudFiles)
	o.endpoints = maps.Clone(o.endpoints)
	o.microversions = maps.Clone(o.microversions)
	o.microversionRanges = maps.Clone(o.microversionRanges)
	o.versionedEndpoints = maps.Clone(o.versionedEndpoints)
	o.microversionSelections = make(map[string]MicroversionSelection)
	if o.auth != nil {
		auth := cloneAuthOptions(*o.auth)
		o.auth = &auth
	}
	if o.region != nil {
		region := *o.region
		o.region = &region
	}
	if o.availability != nil {
		availability := *o.availability
		o.availability = &availability
	}
	o.locationFacts = o.locationFacts.Clone()
	if o.cloudLocation != nil {
		location := o.cloudLocation.Clone()
		o.cloudLocation = &location
	}
	return o
}

// GlobalRequest returns a new Connection that shares this Connection's
// authenticated provider (token, HTTP client and reauthentication) and sends
// the X-OpenStack-Request-Id header with globalRequestID on every request made
// through services obtained from the new Connection.
//
// The request ID is fixed for the lifetime of the derived Connection; there is
// no in-place mutation of the receiver, so requests that are already running
// on it are never changed. The derived Connection has its own service cache
// and microversion negotiation, copies the receiver's options and custom
// services, and needs no Close: it owns no resources beyond the shared
// provider, which remains owned by whoever created the receiver. Services
// already obtained from the receiver keep sending no global request ID.
//
// The ID must be non-empty printable ASCII without spaces. Services such as
// Nova only honor IDs that start with "req-"; that prefix is not enforced.
// The ID is not sent to Keystone token requests of the shared provider.
func (c *Connection) GlobalRequest(globalRequestID string) (*Connection, error) {
	if c == nil || c.provider == nil {
		return nil, invalid("Connection is required")
	}
	if globalRequestID == "" {
		return nil, invalid("global request ID must not be empty")
	}
	for _, r := range globalRequestID {
		if r <= ' ' || r >= 0x7f {
			return nil, invalid("global request ID must be printable ASCII without spaces")
		}
	}
	return c.derive(func(m *connectionMisc) { m.globalRequestID = globalRequestID }), nil
}

// derive copies the Connection configuration into a new Connection that shares
// the provider. The cache of services and clients is not shared.
func (c *Connection) derive(modify func(*connectionMisc)) *Connection {
	c.mu.Lock()
	options := cloneConnectionOptions(c.options)
	c.mu.Unlock()
	derived := &Connection{
		provider: c.provider, options: options, endpointOptions: c.endpointOptions,
		services: make(map[string]any), clients: make(map[string]*gophercloud.ServiceClient),
	}
	derived.misc.auth = c.misc.auth
	derived.misc.globalRequestID = c.misc.globalRequestID
	c.misc.mu.Lock()
	derived.misc.custom = maps.Clone(c.misc.custom)
	c.misc.mu.Unlock()
	if modify != nil {
		modify(&derived.misc)
	}
	return derived
}

// ConnectAsOption overrides one authentication setting of ConnectAs.
type ConnectAsOption func(*connectAsConfig) error

type connectAsConfig struct {
	username, userID, password                                 *string
	userDomainID, userDomainName                               *string
	projectName, projectID, projectDomainID, projectDomainName *string
	raw                                                        []func(*gophercloud.AuthOptions) error
}

func connectAsString(set func(*connectAsConfig, *string), value string) ConnectAsOption {
	return func(c *connectAsConfig) error { set(c, &value); return nil }
}

// AsUsername overrides the user name. Like Python, overriding the user name or
// the user ID discards both inherited values first.
func AsUsername(name string) ConnectAsOption {
	return connectAsString(func(c *connectAsConfig, v *string) { c.username = v }, name)
}

// AsUserID overrides the user ID; see AsUsername for the discard rule. Because
// Identity v3 rejects a user domain next to a user ID, the inherited user domain
// is dropped (after preserving the project scope it implied) unless the user
// domain is overridden too, which Keystone-side validation then reports.
func AsUserID(id string) ConnectAsOption {
	return connectAsString(func(c *connectAsConfig, v *string) { c.userID = v }, id)
}

// AsPassword overrides the password. An empty value is sent as empty.
func AsPassword(password string) ConnectAsOption {
	return connectAsString(func(c *connectAsConfig, v *string) { c.password = v }, password)
}

// AsUserDomainID overrides the user domain ID, discarding the inherited user
// domain ID and name.
func AsUserDomainID(id string) ConnectAsOption {
	return connectAsString(func(c *connectAsConfig, v *string) { c.userDomainID = v }, id)
}

// AsUserDomainName overrides the user domain name; see AsUserDomainID.
func AsUserDomainName(name string) ConnectAsOption {
	return connectAsString(func(c *connectAsConfig, v *string) { c.userDomainName = v }, name)
}

// AsProjectName scopes to a project by name, discarding the inherited project
// ID and name. The project domain comes from AsProjectDomainID/Name, else from
// the inherited project-by-name scope, else from the user domain.
func AsProjectName(name string) ConnectAsOption {
	return connectAsString(func(c *connectAsConfig, v *string) { c.projectName = v }, name)
}

// AsProjectID scopes to a project by ID; see AsProjectName for the discard
// rule. When both an ID and a name are given the ID selects the scope.
func AsProjectID(id string) ConnectAsOption {
	return connectAsString(func(c *connectAsConfig, v *string) { c.projectID = v }, id)
}

// AsProjectDomainID sets the domain ID of a project selected by name.
func AsProjectDomainID(id string) ConnectAsOption {
	return connectAsString(func(c *connectAsConfig, v *string) { c.projectDomainID = v }, id)
}

// AsProjectDomainName sets the domain name of a project selected by name.
func AsProjectDomainName(name string) ConnectAsOption {
	return connectAsString(func(c *connectAsConfig, v *string) { c.projectDomainName = v }, name)
}

// AsAuthOptions applies an arbitrary edit to a copy of the resolved
// authentication options after the typed overrides. It is the escape hatch for
// settings without a typed override (token, application credential, trust).
func AsAuthOptions(edit func(*gophercloud.AuthOptions) error) ConnectAsOption {
	return func(c *connectAsConfig) error {
		if edit == nil {
			return invalid("nil auth edit")
		}
		c.raw = append(c.raw, edit)
		return nil
	}
}

func (cfg *connectAsConfig) apply(auth *gophercloud.AuthOptions) error {
	if cfg.username != nil || cfg.userID != nil {
		auth.Username, auth.UserID = "", ""
		if cfg.username != nil {
			auth.Username = *cfg.username
		}
		if cfg.userID != nil {
			auth.UserID = *cfg.userID
		}
		if auth.UserID != "" && cfg.userDomainID == nil && cfg.userDomainName == nil {
			// Identity v3 rejects a user domain next to a user ID. Python keeps
			// the unused domain; keep the project scope it implied, then drop it.
			if auth.Scope == nil && auth.TenantID != "" {
				auth.Scope = &gophercloud.AuthScope{ProjectID: auth.TenantID}
			} else if auth.Scope == nil && auth.TenantName != "" {
				auth.Scope = &gophercloud.AuthScope{ProjectName: auth.TenantName, DomainID: auth.DomainID, DomainName: auth.DomainName}
			}
			auth.DomainID, auth.DomainName = "", ""
		}
	}
	if cfg.password != nil {
		auth.Password = *cfg.password
	}
	if cfg.userDomainID != nil || cfg.userDomainName != nil {
		auth.DomainID, auth.DomainName = "", ""
		if cfg.userDomainID != nil {
			auth.DomainID = *cfg.userDomainID
		}
		if cfg.userDomainName != nil {
			auth.DomainName = *cfg.userDomainName
		}
	}
	projectChanged := cfg.projectName != nil || cfg.projectID != nil
	domainChanged := cfg.projectDomainID != nil || cfg.projectDomainName != nil
	if projectChanged {
		var inheritedID, inheritedName string
		if auth.Scope != nil && auth.Scope.ProjectName != "" {
			inheritedID, inheritedName = auth.Scope.DomainID, auth.Scope.DomainName
		} else {
			inheritedID, inheritedName = auth.DomainID, auth.DomainName
		}
		auth.TenantID, auth.TenantName = "", ""
		scope := &gophercloud.AuthScope{}
		if cfg.projectID != nil && *cfg.projectID != "" {
			auth.TenantID = *cfg.projectID
			if cfg.projectName != nil {
				auth.TenantName = *cfg.projectName
			}
			scope.ProjectID = *cfg.projectID
		} else if cfg.projectName != nil && *cfg.projectName != "" {
			auth.TenantName = *cfg.projectName
			scope.ProjectName = *cfg.projectName
			scope.DomainID, scope.DomainName = inheritedID, inheritedName
			if domainChanged {
				scope.DomainID, scope.DomainName = "", ""
			}
		} else {
			return invalid("a project override needs a non-empty project ID or name")
		}
		auth.Scope = scope
	}
	if domainChanged {
		if auth.Scope == nil || auth.Scope.ProjectName == "" {
			return invalid("a project domain override needs a project selected by name")
		}
		auth.Scope.DomainID, auth.Scope.DomainName = "", ""
		if cfg.projectDomainID != nil {
			auth.Scope.DomainID = *cfg.projectDomainID
		}
		if cfg.projectDomainName != nil {
			auth.Scope.DomainName = *cfg.projectDomainName
		}
	}
	for _, edit := range cfg.raw {
		if err := edit(auth); err != nil {
			return err
		}
	}
	return nil
}

// ConnectAs authenticates again with the receiver's recorded authentication
// options overridden by the given settings and returns a new, independent
// Connection. Everything else is inherited: endpoint, region, interface,
// endpoint and microversion options, HTTP client and TLS settings, location
// facts of the cloud and region, and the default network, address, role and
// image policies. The derived Connection has its own token, provider and
// service cache (there is no shared discovery cache), no global request ID and
// no custom services, and needs no Close. The receiver is unchanged and its
// token is never reused.
//
// Authentication happens within ctx; failures wrap the native cause and context
// cancellation stays observable. A Connection adopted with FromProvider records
// no authentication options and fails with ErrNoRecordedAuth. Passwords are
// held in memory like for the receiver's own reauthentication.
func (c *Connection) ConnectAs(ctx context.Context, overrides ...ConnectAsOption) (*Connection, error) {
	if c == nil || c.provider == nil {
		return nil, invalid("Connection is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("connect as: %w", err)
	}
	if c.misc.auth == nil {
		return nil, ErrNoRecordedAuth
	}
	var cfg connectAsConfig
	for _, override := range overrides {
		if override == nil {
			return nil, invalid("nil connect-as option")
		}
		if err := override(&cfg); err != nil {
			return nil, err
		}
	}
	auth := cloneAuthOptions(*c.misc.auth)
	if err := cfg.apply(&auth); err != nil {
		return nil, err
	}
	c.mu.Lock()
	base := cloneConnectionOptions(c.options)
	c.mu.Unlock()
	base.auth, base.cloud = &auth, ""
	base.serverAddressesSet = true
	if c.endpointOptions.Region != "" {
		region := c.endpointOptions.Region
		base.region = &region
	}
	availability := c.endpointOptions.Availability
	base.availability = &availability
	inherit := func(o *connectionOptions) error { *o = base; return nil }
	derived, err := Connect(ctx, inherit)
	if err != nil {
		return nil, fmt.Errorf("connect as: %w", err)
	}
	// Project facts follow the new scope; the cloud and region facts stay.
	facts := c.options.locationFacts.Clone()
	derived.options.locationFacts.Cloud = facts.Cloud
	derived.options.locationFacts.RegionName = facts.RegionName
	return derived, nil
}

// ProjectRef selects a project for ConnectAsProject like the dict returned by
// list_projects: an ID, a name, or both. The domain ID applies to a project
// selected by name.
type ProjectRef struct {
	ID       string
	Name     string
	DomainID string
}

// ConnectAsProject is ConnectAs scoped to another project. At least an ID or a
// name is required. Like Python, the inherited project ID and name are
// discarded; the project domain is only overridden when DomainID is set.
func (c *Connection) ConnectAsProject(ctx context.Context, project ProjectRef) (*Connection, error) {
	if project.ID == "" && project.Name == "" {
		return nil, invalid("project reference needs an ID or a name")
	}
	overrides := []ConnectAsOption{AsProjectName(project.Name)}
	if project.ID != "" {
		overrides = append(overrides, AsProjectID(project.ID))
	}
	if project.DomainID != "" {
		overrides = append(overrides, AsProjectDomainID(project.DomainID))
	}
	return c.ConnectAs(ctx, overrides...)
}

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

// ServiceFactory builds a Gophercloud client for a custom service from a
// provider whose endpoint locator selects the catalog or override endpoint.
type ServiceFactory func(provider *gophercloud.ProviderClient, options gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error)

// AddService registers a custom service on this Connection only, so that
// CustomServiceClient can build its client lazily from the shared provider with
// the Connection's region and interface. The name must be new: built-in
// services cannot be replaced and a registration cannot be changed. An optional
// endpoint (absolute HTTP(S) URL passed to the factory through the provider's
// locator) replaces the catalog lookup. Registration performs no request.
// Unlike Python, which attaches the description to the Connection class for the
// whole process, nothing outside this Connection and its later derived
// Connections is affected.
func (c *Connection) AddService(service Service, factory ServiceFactory, endpoint ...string) error {
	if c == nil || c.provider == nil {
		return invalid("Connection is required")
	}
	if strings.TrimSpace(string(service)) == "" || factory == nil {
		return invalid("service name and factory are required")
	}
	if _, builtin := serviceDefinitions[service]; builtin {
		return invalid("service %q is built in", service)
	}
	if len(endpoint) > 1 {
		return invalid("at most one endpoint override")
	}
	override := ""
	if len(endpoint) == 1 {
		check := connectionOptions{endpoints: make(map[Service]string)}
		if err := WithEndpoint(Compute, endpoint[0])(&check); err != nil {
			return err
		}
		override = endpoint[0]
	}
	c.misc.mu.Lock()
	defer c.misc.mu.Unlock()
	if _, exists := c.misc.custom[service]; exists {
		return invalid("service %q is already registered", service)
	}
	if c.misc.custom == nil {
		c.misc.custom = make(map[Service]*customService)
	}
	c.misc.custom[service] = &customService{factory: factory, endpoint: override}
	return nil
}

// CustomServiceClient returns the cached client of a service registered with
// AddService, building it on first use. The client shares the provider and
// carries the global request ID of a GlobalRequest Connection.
func (c *Connection) CustomServiceClient(ctx context.Context, service Service) (*gophercloud.ServiceClient, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.misc.mu.Lock()
	defer c.misc.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if client := c.misc.clients[service]; client != nil {
		return client, nil
	}
	custom := c.misc.custom[service]
	if custom == nil {
		return nil, unsupported(string(service), "unregistered custom service")
	}
	provider := c.provider
	if custom.endpoint != "" {
		endpoint := custom.endpoint
		provider = &gophercloud.ProviderClient{EndpointLocator: func(gophercloud.EndpointOpts) (string, error) { return endpoint, nil }}
	} else if provider.EndpointLocator == nil {
		return nil, invalid("provider has no endpoint locator; register %q with an endpoint", service)
	}
	client, err := custom.factory(provider, c.endpointOptions)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", service, err)
	}
	if client == nil {
		return nil, invalid("service factory for %q returned no client", service)
	}
	client.ProviderClient = c.provider
	c.misc.decorate(client)
	if c.misc.clients == nil {
		c.misc.clients = make(map[Service]*gophercloud.ServiceClient)
	}
	c.misc.clients[service] = client
	return client, nil
}
