package openstack

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strings"

	accelerator "github.com/JSYoo5B/go-openstacksdk/accelerator/v2"
	imageapi "github.com/JSYoo5B/go-openstacksdk/image/v2"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudconfig"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
)

// Only a single-cause chain ending in catalog absence can select an alternate
// backend. An errors.Join may also carry HTTP, source or cancellation failures.
func missingCatalogEndpoint(err error) bool {
	for err != nil {
		switch err.(type) {
		case gophercloud.ErrEndpointNotFound, *gophercloud.ErrEndpointNotFound:
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

const (
	Accelerator            Service = "accelerator"
	BareMetal              Service = "baremetal"
	BareMetalIntrospection Service = "baremetal-introspection"
	Clustering             Service = "clustering"
	Container              Service = "application-container"
	ContainerInfra         Service = "container-infrastructure-management"
	Database               Service = "database"
	DNS                    Service = "dns"
	Identity               Service = "identity"
	InstanceHA             Service = "instance-ha"
	KeyManager             Service = "key-manager"
	LoadBalancer           Service = "load-balancer"
	Messaging              Service = "message"
	Metric                 Service = "metric-storage"
	ObjectStorage          Service = "object-store"
	Orchestration          Service = "orchestration"
	Placement              Service = "placement"
	Reservation            Service = "reservation"
	SharedFileSystem       Service = "shared-file-system"
	Workflow               Service = "workflow"
)

type clientFactory func(*gophercloud.ProviderClient, gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error)

type serviceDefinition struct {
	version           string
	microversionMajor string
	factories         map[string]clientFactory
}

var serviceDefinitions = map[Service]serviceDefinition{
	Accelerator:            {"v2", "2", map[string]clientFactory{"v2": newAcceleratorV2}},
	Compute:                {"v2", "2", map[string]clientFactory{"v2": openstack.NewComputeV2}},
	Network:                {"v2", "", map[string]clientFactory{"v2": openstack.NewNetworkV2}},
	Image:                  {"v2", "", map[string]clientFactory{"v2": openstack.NewImageV2}},
	BlockStorage:           {"v3", "3", map[string]clientFactory{"v2": openstack.NewBlockStorageV2, "v3": openstack.NewBlockStorageV3}},
	BareMetal:              {"v1", "1", map[string]clientFactory{"v1": openstack.NewBareMetalV1}},
	BareMetalIntrospection: {"v1", "1", map[string]clientFactory{"v1": openstack.NewBareMetalIntrospectionV1}},
	Clustering:             {"v1", "1", map[string]clientFactory{"v1": newClusteringV1}},
	Container:              {"v1", "1", map[string]clientFactory{"v1": openstack.NewContainerV1}},
	ContainerInfra:         {"v1", "1", map[string]clientFactory{"v1": openstack.NewContainerInfraV1}},
	Database:               {"v1", "", map[string]clientFactory{"v1": openstack.NewDBV1}},
	DNS:                    {"v2", "", map[string]clientFactory{"v2": openstack.NewDNSV2}},
	Identity:               {"v3", "", map[string]clientFactory{"v2": openstack.NewIdentityV2, "v3": openstack.NewIdentityV3}},
	InstanceHA:             {"v1", "1", map[string]clientFactory{"v1": newInstanceHAV1}},
	KeyManager:             {"v1", "", map[string]clientFactory{"v1": openstack.NewKeyManagerV1}},
	LoadBalancer:           {"v2", "", map[string]clientFactory{"v2": openstack.NewLoadBalancerV2}},
	Messaging:              {"v2", "", map[string]clientFactory{}},
	Metric:                 {"v1", "", map[string]clientFactory{"v1": openstack.NewMetricV1}},
	ObjectStorage:          {"v1", "", map[string]clientFactory{"v1": openstack.NewObjectStorageV1}},
	Orchestration:          {"v1", "", map[string]clientFactory{"v1": openstack.NewOrchestrationV1}},
	Placement:              {"v1", "1", map[string]clientFactory{"v1": openstack.NewPlacementV1}},
	Reservation:            {"v1", "", map[string]clientFactory{"v1": openstack.NewReservationV1}},
	SharedFileSystem:       {"v2", "2", map[string]clientFactory{"v2": openstack.NewSharedFileSystemV2}},
	Workflow:               {"v2", "", map[string]clientFactory{"v2": openstack.NewWorkflowV2}},
}

func serviceKey(service Service, version string) string { return string(service) + "/" + version }

func cachedService[T any](ctx context.Context, c *Connection, service Service, version string, newService func(*gophercloud.ServiceClient) *T) (*T, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := serviceKey(service, version)
	if proxy, ok := c.services[key]; ok {
		return proxy.(*T), nil
	}
	client, err := c.serviceClientVersion(ctx, service, version)
	if err != nil {
		return nil, err
	}
	proxy := newService(client)
	c.services[key] = proxy
	return proxy, nil
}

// serviceClientVersion is called while the connection mutex is held.
func (c *Connection) serviceClientVersion(ctx context.Context, service Service, version string) (*gophercloud.ServiceClient, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	definition, ok := serviceDefinitions[service]
	if !ok {
		return nil, unsupported(string(service), version)
	}
	factory, ok := definition.factories[version]
	if service == Messaging && version == "v2" {
		ok = true
		factory = func(p *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
			return openstack.NewMessagingV2(p, c.options.messagingClientID, eo)
		}
	}
	if !ok {
		return nil, unsupported(string(service), version)
	}
	key := serviceKey(service, version)
	if client := c.clients[key]; client != nil {
		return client, nil
	}
	_, negotiate := c.options.microversionRanges[service]
	if (c.options.microversions[service] != "" || negotiate) && version != definition.version {
		return nil, invalid("microversion configured for %s %s cannot be used with %s", service, definition.version, version)
	}
	endpoint := c.options.versionedEndpoints[key]
	if endpoint == "" {
		endpoint = c.options.endpoints[service]
	}
	provider := c.provider
	if endpoint != "" {
		endpoint = overrideCatalogEndpoint(service, version, endpoint)
		provider = &gophercloud.ProviderClient{EndpointLocator: func(gophercloud.EndpointOpts) (string, error) { return endpoint, nil }}
	} else if provider.EndpointLocator == nil {
		return nil, invalid("provider has no endpoint locator; supply WithEndpoint(%q, ...)", service)
	}
	eo := c.endpointOptions
	if service == Identity {
		if version == "v2" {
			eo.Version = 2
		} else {
			eo.Version = 3
		}
	}
	client, err := factory(provider, eo)
	if err != nil {
		return nil, fmt.Errorf("connect %s %s: %w", service, version, err)
	}
	client.ProviderClient = c.provider
	c.misc.decorate(client)
	selection, err := c.selectMicroversion(ctx, service, client)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client.Microversion = selection.Selected
	c.options.microversionSelections[key] = selection
	c.clients[key] = client
	return client, nil
}

// Constructors add their resource prefix to catalog endpoints. Accept an
// explicit endpoint with or without that prefix, adding it exactly once.
func overrideCatalogEndpoint(service Service, version, endpoint string) string {
	prefix := ""
	switch service {
	case Image, DNS:
		prefix = "v2"
	case KeyManager:
		prefix = "v1"
	case Network, LoadBalancer:
		prefix = "v2.0"
	case Metric:
		prefix = "api/v1"
	}
	endpoint = strings.TrimRight(endpoint, "/")
	if prefix != "" {
		endpoint = strings.TrimSuffix(endpoint, "/"+prefix)
	}
	return gophercloud.NormalizeURL(endpoint)
}

func newMessagingClientID() string {
	var bytes [16]byte
	_, _ = rand.Read(bytes[:])
	bytes[6] = (bytes[6] & 15) | 64
	bytes[8] = (bytes[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:])
}

type customService struct {
	factory  ServiceFactory
	endpoint string
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

// AcceleratorV2 returns a cached Cyborg client sharing authentication and HTTP
// policies with the Connection. Explicit endpoints may include the v2 suffix.
func (c *Connection) AcceleratorV2(ctx context.Context) (*accelerator.Service, error) {
	return cachedService(ctx, c, Accelerator, "v2", accelerator.New)
}

func (c *Connection) Accelerator(ctx context.Context) (*accelerator.Service, error) {
	return c.AcceleratorV2(ctx)
}

func newAcceleratorV2(provider *gophercloud.ProviderClient, options gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
	options.ApplyDefaults("accelerator")
	endpoint, err := provider.EndpointLocator(options)
	if err != nil {
		return nil, err
	}
	endpoint = strings.TrimSuffix(strings.TrimRight(endpoint, "/"), "/v2")
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: "accelerator", Endpoint: gophercloud.NormalizeURL(endpoint + "/v2")}, nil
}

func newClusteringV1(provider *gophercloud.ProviderClient, options gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
	return newSDKOwnedV1(provider, options, Clustering)
}

func newInstanceHAV1(provider *gophercloud.ProviderClient, options gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
	return newSDKOwnedV1(provider, options, InstanceHA)
}

// Preserve a catalog's version/project suffix and reverse-proxy prefix. Senlin
// also publishes unversioned catalog endpoints, which need the v1 resource root.
func newSDKOwnedV1(provider *gophercloud.ProviderClient, options gophercloud.EndpointOpts, service Service) (*gophercloud.ServiceClient, error) {
	options.ApplyDefaults(string(service))
	endpoint, err := provider.EndpointLocator(options)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, invalid("%s endpoint must be an absolute HTTP(S) URL without credentials, query or fragment", service)
	}
	parts := strings.Split(strings.TrimRight(u.EscapedPath(), "/"), "/")
	index := len(parts) - 1
	if index > 0 && !microversions.IsPathVersion(parts[index]) && microversions.IsPathVersion(parts[index-1]) {
		index--
	}
	if microversions.IsPathVersion(parts[index]) {
		version := parts[index]
		if version != "v1" && !strings.HasPrefix(version, "v1.") {
			return nil, unsupported(string(service), "endpoint version "+version)
		}
	} else {
		u.RawPath = strings.TrimRight(u.EscapedPath(), "/") + "/v1"
		u.Path, err = url.PathUnescape(u.RawPath)
		if err != nil {
			return nil, err
		}
		endpoint = u.String()
	}
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: string(service), Endpoint: gophercloud.NormalizeURL(endpoint)}, nil
}

// The versioned and high-level image entry points share Connection facts as
// well as their authenticated service client.
func (c *Connection) newImageV2Service(client *gophercloud.ServiceClient) *imageapi.Service {
	return imageapi.NewWithDependencies(client, imageapi.Dependencies{CloudLocation: c.CurrentLocation})
}

// officialServiceTypes lists the official service types of the OpenStack
// service-types-authority data shipped with os-service-types 1.8.1 (data
// version 2024-05-08, sha 52d438fe913eecea4e14d1e83f148cbe22edef91), the
// minimum version required by the pinned openstacksdk. CloudRegion.has_service
// uses is_official(service_type) as the default of an unset has_<type> flag.
var officialServiceTypes = map[string]bool{
	"accelerator": true, "admin-logic": true, "alarm": true, "application-catalog": true,
	"application-container": true, "application-deployment": true, "backup": true,
	"baremetal": true, "baremetal-introspection": true, "block-storage": true,
	"clustering": true, "compute": true, "container-infrastructure-management": true,
	"data-processing": true, "data-protection-orchestration": true, "database": true,
	"dns": true, "ec2-api": true, "event": true, "function-engine": true, "identity": true,
	"image": true, "instance-ha": true, "key-manager": true, "load-balancer": true,
	"log-management": true, "message": true, "meter": true, "monitoring": true,
	"monitoring-events": true, "monitoring-logging": true,
	"multi-region-network-automation": true, "network": true, "nfv-orchestration": true,
	"object-store": true, "operator-policy": true, "orchestration": true, "placement": true,
	"rating": true, "reservation": true, "resource-optimization": true,
	"root-cause-analysis": true, "search": true, "shared-file-system": true, "workflow": true,
}

// serviceFlagKey mirrors CloudRegion.has_service's key normalization.
func serviceFlagKey(serviceType string) string {
	return strings.ReplaceAll(strings.ToLower(serviceType), "-", "_")
}

// WithServiceEnabled sets the has_<service> configuration flag of one service
// type, as clouds.yaml does. A disabled service makes HasService and
// HasServiceVersion report false without any lookup. The type is normalized
// like Python (lower case, '-' and '_' equivalent). It takes precedence over
// clouds.yaml for that type.
func WithServiceEnabled(serviceType string, enabled bool) ConnectionOption {
	return func(o *connectionOptions) error {
		if strings.TrimSpace(serviceType) == "" {
			return invalid("service type must not be empty")
		}
		flags := maps.Clone(o.serviceFlags)
		if flags == nil {
			flags = make(map[string]bool)
		}
		flags[serviceFlagKey(serviceType)] = enabled
		o.serviceFlags = flags
		return nil
	}
}

// WithSecurityGroupSource sets the secgroup_source configuration, replacing
// clouds.yaml. "neutron" (the default) uses Neutron security groups when the
// network service is available; "nova" (or any other casing of either value)
// uses the Nova os-security-groups API; any other value, including the empty
// string that mirrors a YAML null, makes the security-group helpers fail with
// network.ErrUnavailableSecurityGroups. As in Python, only the exact string
// "neutron" selects Neutron.
func WithSecurityGroupSource(source string) ConnectionOption {
	return func(o *connectionOptions) error {
		value := source
		o.secgroupSource, o.secgroupSourceSet = &value, true
		return nil
	}
}

func (o *connectionOptions) applyConfiguredServiceSettings(configuration cloudconfig.Configuration) {
	if len(configuration.ServiceFlags) != 0 {
		flags := maps.Clone(configuration.ServiceFlags)
		maps.Copy(flags, o.serviceFlags)
		o.serviceFlags = flags
	}
	if !o.secgroupSourceSet && configuration.SecgroupSource != nil {
		value := *configuration.SecgroupSource
		o.secgroupSource = &value
	}
}

// ServiceEnabled mirrors CloudRegion.has_service: the has_<service> flag from
// WithServiceEnabled or clouds.yaml, else whether the type is an official
// service type. Aliases such as volumev3 are not official and default to
// false. No request is made.
func (c *Connection) ServiceEnabled(serviceType string) bool {
	if c == nil {
		return false
	}
	key := serviceFlagKey(serviceType)
	if enabled, ok := c.options.serviceFlags[key]; ok {
		return enabled
	}
	return officialServiceTypes[strings.ReplaceAll(key, "_", "-")]
}

// securityGroupSource returns secgroup_source with Python's "neutron" default.
func (c *Connection) securityGroupSource() string {
	if c.options.secgroupSource == nil {
		return "neutron"
	}
	return *c.options.secgroupSource
}

// hasSecurityGroups mirrors _has_secgroups.
func (c *Connection) hasSecurityGroups() bool {
	source := strings.ToLower(c.securityGroupSource())
	return source == "neutron" || source == "nova"
}

// WithServiceLookupFailureAsUnavailable selects Python's has_service policy
// for network role discovery and the floating IP helpers' backend choice: any
// failure to look up the Network endpoint (for example a catalog or
// authentication error) makes the service unavailable, so roles are empty and
// the helpers use Nova, instead of returning the error. A missing endpoint is
// always unavailable, and context cancellation is always returned. The default
// (false) returns lookup failures so that they stay observable. Nothing is
// cached: the next call looks the endpoint up again, as Python does.
func WithServiceLookupFailureAsUnavailable(enabled bool) ConnectionOption {
	return func(o *connectionOptions) error { o.lookupFailureUnavailable = enabled; return nil }
}

// networkLookupUnavailable reports whether a Network service lookup error
// means "no Neutron" rather than a returned failure.
func (c *Connection) networkLookupUnavailable(ctx context.Context, err error) bool {
	if missingCatalogEndpoint(err) {
		return true
	}
	return c.options.lookupFailureUnavailable && ctx.Err() == nil
}
