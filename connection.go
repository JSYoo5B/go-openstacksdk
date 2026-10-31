package openstack

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudconfig"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlocation"
	"github.com/JSYoo5B/go-openstacksdk/internal/nativefind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

// Connection shares authentication and caches lazily constructed service proxies.
// Construct it with Connect or FromProvider. Concurrent service access is safe.
type Connection struct {
	provider        *gophercloud.ProviderClient
	options         connectionOptions
	endpointOptions gophercloud.EndpointOpts
	mu              sync.Mutex
	compute         *compute.Service
	network         *network.Service
	image           *image.Service
	blockStorage    *blockstorage.Service
	services        map[string]any
	clients         map[string]*gophercloud.ServiceClient
	misc            connectionMisc
}

// Connect authenticates once. It uses OS_CLOUD/clouds.yaml when OS_CLOUD is set,
// otherwise Gophercloud's OS_* authentication parser. Functional options override
// these defaults. Authentication always happens within the supplied context.
func Connect(ctx context.Context, opts ...ConnectionOption) (*Connection, error) {
	o, err := parseConnection(opts)
	if err != nil {
		return nil, err
	}
	eo := gophercloud.EndpointOpts{Region: os.Getenv("OS_REGION_NAME"), Availability: gophercloud.Availability(os.Getenv("OS_INTERFACE"))}
	cloudName := o.cloud
	if cloudName == "" && o.auth == nil {
		cloudName = os.Getenv("OS_CLOUD")
	}
	var auth gophercloud.AuthOptions
	if o.auth != nil {
		auth = *o.auth
	} else if cloudName != "" {
		configuration, loadErr := cloudconfig.Load(cloudName, o.cloudFiles, o.region)
		if loadErr != nil {
			return nil, fmt.Errorf("load cloud configuration: %w", loadErr)
		}
		auth, eo, o.tlsConfig, err = clouds.Parse(configuration.ParseOptions...)
		if err != nil {
			return nil, fmt.Errorf("load cloud configuration: %w", err)
		}
		o.configuredDefaultNetwork = configuration.DefaultNetwork
		if !o.imageCreatePolicySet {
			o.imageCreatePolicy = configuration.ImageCreatePolicy
		}
		if !o.networkRolesSet {
			o.networkRoles = configuration.NetworkRoles
		}
		if !o.serverAddressesSet {
			o.serverAddresses = configuration.ServerAddresses
		}
		o.applyConfiguredServiceSettings(configuration)
	} else {
		auth, err = openstack.AuthOptionsFromEnv()
		if err != nil {
			return nil, fmt.Errorf("load environment authentication: %w", err)
		}
		configuration, present, loadErr := cloudconfig.LoadEnvironment(os.Environ(), o.region)
		if loadErr != nil {
			return nil, fmt.Errorf("load environment cloud configuration: %w", loadErr)
		}
		if present {
			o.configuredDefaultNetwork = configuration.DefaultNetwork
			if !o.networkRolesSet {
				o.networkRoles = configuration.NetworkRoles
			}
			o.applyConfiguredServiceSettings(configuration)
		}
	}
	if cloudName == "" && !o.serverAddressesSet {
		o.serverAddresses, err = cloudconfig.ServerAddresses(nil, nil)
		if err != nil {
			return nil, err
		}
	}
	client := o.httpClient
	if o.tlsConfig != nil {
		var transport *http.Transport
		switch t := client.Transport.(type) {
		case nil:
			transport = http.DefaultTransport.(*http.Transport).Clone()
		case *http.Transport:
			transport = t.Clone()
		default:
			return nil, invalid("cloud TLS settings require *http.Transport; custom RoundTrippers must be supplied via FromProvider")
		}
		transport.TLSClientConfig = o.tlsConfig.Clone()
		client.Transport = transport
	}
	// Preserve configured facts before native authentication derives/mutates
	// its internal scope. Token response names are not configuration names.
	o.locationFacts = configuredLocation(cloudName, auth)
	recordedAuth := cloneAuthOptions(auth)
	provider, err := config.NewProviderClient(ctx, auth, config.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	if _, present := os.LookupEnv("OS_REGION_NAME"); present || cloudName != "" || eo.Region != "" {
		o.locationFacts.RegionName = locationString(eo.Region)
	}
	connection, err := newConnection(provider, o, eo)
	if err == nil {
		connection.misc.auth = &recordedAuth
	}
	return connection, err
}

// FromProvider adopts an already authenticated provider without changing it.
// Region and interface default to public/unspecified and can be overridden.
// Authentication/cloud/HTTP options are rejected here to prevent silent ignores.
func FromProvider(provider *gophercloud.ProviderClient, opts ...ConnectionOption) (*Connection, error) {
	if provider == nil {
		return nil, invalid("provider must not be nil")
	}
	o, err := parseConnection(opts)
	if err != nil {
		return nil, err
	}
	if o.auth != nil || o.cloud != "" || len(o.cloudFiles) != 0 || o.httpConfigured {
		return nil, invalid("FromProvider accepts endpoint, region, interface, microversion, location, default network and network role options")
	}
	if !o.serverAddressesSet {
		o.serverAddresses, err = compute.PrepareServerAddressPolicy()
		if err != nil {
			return nil, err
		}
	}
	return newConnection(provider, o, gophercloud.EndpointOpts{})
}

func newConnection(provider *gophercloud.ProviderClient, o connectionOptions, eo gophercloud.EndpointOpts) (*Connection, error) {
	var policyErr error
	o.imageCreatePolicy, policyErr = image.PrepareImageCreatePolicy(image.WithImageCreatePolicyOpts(o.imageCreatePolicy))
	if policyErr != nil {
		return nil, policyErr
	}
	if o.networkRolesSet {
		o.configuredDefaultNetwork = o.networkRoles.DefaultNetworkSelector()
	}
	if o.region != nil {
		eo.Region = *o.region
		o.locationFacts.RegionName = locationString(*o.region)
	}
	if o.availability != nil {
		eo.Availability = *o.availability
	}
	if eo.Availability == "" {
		eo.Availability = gophercloud.AvailabilityPublic
	}
	switch eo.Availability {
	case gophercloud.AvailabilityPublic, gophercloud.AvailabilityInternal, gophercloud.AvailabilityAdmin:
	default:
		return nil, invalid("invalid endpoint interface %q", eo.Availability)
	}
	if o.messagingClientID == "" {
		o.messagingClientID = newMessagingClientID()
	}
	return &Connection{provider: provider, options: o, endpointOptions: eo, services: make(map[string]any), clients: make(map[string]*gophercloud.ServiceClient)}, nil
}

func (c *Connection) serviceClient(ctx context.Context, service Service) (*gophercloud.ServiceClient, error) {
	return c.serviceClientVersion(ctx, service, serviceDefinitions[service].version)
}

func (c *Connection) Compute(ctx context.Context) (*compute.Service, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.compute == nil {
		client, err := c.serviceClient(ctx, Compute)
		if err != nil {
			return nil, err
		}
		selection := c.options.microversionSelections[serviceKey(Compute, serviceDefinitions[Compute].version)]
		c.compute = compute.New(client, compute.Dependencies{
			AdvertisedMinimum: selection.SupportedMinimum, AdvertisedMaximum: selection.SupportedMaximum,
			BackupImages:  c.Image,
			CloudLocation: c.CurrentLocation,
			NetworkRoles:  c.GetNetworkRoles, AddressNetworks: c.addressNetworkService,
			AddressCompute: c.addressComputeClient, NetworkPolicy: c.options.networkRoles, ServerAddresses: c.options.serverAddresses,
			DefaultNetwork:          c.defaultServerNetwork,
			DefaultNetworkUsesRoles: !c.options.defaultNetworkSet && c.options.configuredDefaultNetwork != "",
			FloatingIPs: func(ctx context.Context) (*network.FloatingIPs, error) {
				service, err := c.Network(ctx)
				if err != nil {
					return nil, err
				}
				return service.FloatingIPs, nil
			},
			Image: func(ctx context.Context, ref resource.Ref) (string, error) {
				service, err := c.Image(ctx)
				if err != nil {
					return "", err
				}
				if rest.HasOperationGuard(ctx) {
					if service.API == nil || service.Images == nil || service.RawClient() == nil || service.RawClient().ProviderClient == nil {
						return "", invalid("image resolver service is required")
					}
					api, collection, raw := service.API, service.Images, service.RawClient()
					child := api.Images
					return nativefind.ResolveName(ctx, raw, "images", "image", "images", ref,
						func(value *image.Image) string { return value.ID }, func(value *image.Image) string { return value.Name }, true,
						func(context.Context) error {
							if service.API != api || service.Images != collection || service.RawClient() != raw || api.Images != child || api.RawClient() != raw {
								return invalid("image resolver source changed")
							}
							return nil
						})
				}
				value, err := service.Images.Find(ctx, ref)
				if err != nil {
					return "", err
				}
				return value.ID, nil
			},
			Network: func(ctx context.Context, ref resource.Ref) (string, error) {
				service, err := c.Network(ctx)
				if err != nil {
					return "", err
				}
				if rest.HasOperationGuard(ctx) {
					if service.API == nil || service.Networks == nil || service.RawClient() == nil || service.RawClient().ProviderClient == nil {
						return "", invalid("network resolver service is required")
					}
					api, collection, raw := service.API, service.Networks, service.RawClient()
					child := api.Networks
					return nativefind.ResolveName(ctx, raw, "networks", "network", "networks", ref,
						func(value *network.Network) string { return value.ID }, func(value *network.Network) string { return value.Name }, true,
						func(context.Context) error {
							if service.API != api || service.Networks != collection || service.RawClient() != raw || api.Networks != child || api.RawClient() != raw {
								return invalid("network resolver source changed")
							}
							return nil
						})
				}
				value, err := service.Networks.Find(ctx, ref)
				if err != nil {
					return "", err
				}
				return value.ID, nil
			},
			Port: func(ctx context.Context, ref resource.Ref) (string, error) {
				service, err := c.Network(ctx)
				if err != nil {
					return "", err
				}
				if rest.HasOperationGuard(ctx) {
					if service.API == nil || service.Ports == nil || service.RawClient() == nil || service.RawClient().ProviderClient == nil {
						return "", invalid("port resolver service is required")
					}
					api, collection, raw := service.API, service.Ports, service.RawClient()
					child := api.Ports
					return nativefind.ResolveName(ctx, raw, "ports", "port", "ports", ref,
						func(value *network.Port) string { return value.ID }, func(value *network.Port) string { return value.Name }, true,
						func(context.Context) error {
							if service.API != api || service.Ports != collection || service.RawClient() != raw || api.Ports != child || api.RawClient() != raw {
								return invalid("port resolver source changed")
							}
							return nil
						})
				}
				return service.Ports.ResolveID(ctx, ref)
			},
			Volume: func(ctx context.Context, ref resource.Ref) (string, error) {
				service, err := c.BlockStorage(ctx)
				if err != nil {
					return "", err
				}
				if rest.HasOperationGuard(ctx) {
					if service.API == nil || service.Volumes == nil || service.RawClient() == nil || service.RawClient().ProviderClient == nil {
						return "", invalid("volume resolver service is required")
					}
					api, collection, raw := service.API, service.Volumes, service.RawClient()
					child := api.Volumes
					return nativefind.ResolveName(ctx, raw, "volumes/detail", "volume", "volumes", ref,
						func(value *blockstorage.Volume) string { return value.ID }, func(value *blockstorage.Volume) string { return value.Name }, true,
						func(context.Context) error {
							if service.API != api || service.Volumes != collection || service.RawClient() != raw || api.Volumes != child || api.RawClient() != raw {
								return invalid("volume resolver source changed")
							}
							return nil
						})
				}
				return service.Volumes.ResolveID(ctx, ref)
			},
		})
	}
	return c.compute, nil
}

func (c *Connection) Network(ctx context.Context) (*network.Service, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.network == nil {
		client, err := c.serviceClient(ctx, Network)
		if err != nil {
			return nil, err
		}
		c.network = network.NewWithDependencies(client, network.Dependencies{
			NetworkRoles: c.options.networkRoles,
			Server: func(ctx context.Context, ref resource.Ref) (string, error) {
				service, err := c.Compute(ctx)
				if err != nil {
					return "", err
				}
				return service.Servers.ResolveID(ctx, ref)
			},
		})
	}
	return c.network, nil
}

func (c *Connection) Image(ctx context.Context) (*image.Service, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.image == nil {
		client, err := c.serviceClient(ctx, Image)
		if err != nil {
			return nil, err
		}
		c.image = image.NewWithDependencies(client, image.Dependencies{CloudLocation: c.CurrentLocation, CreatePolicy: c.options.imageCreatePolicy, ObjectStorage: c.imageCreateSwiftService})
	}
	return c.image, nil
}

func (c *Connection) BlockStorage(ctx context.Context) (*blockstorage.Service, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.blockStorage == nil {
		client, err := c.serviceClient(ctx, BlockStorage)
		if err != nil {
			return nil, err
		}
		c.blockStorage = blockstorage.New(client)
	}
	return c.blockStorage, nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

func unsupported(service, feature string) error {
	return fmt.Errorf("%w: %s does not support %s", resource.ErrUnsupported, service, feature)
}

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

// ErrNoAuthResult reports that the Connection has no recorded Keystone token
// response, for example because it adopted a provider whose token was set by
// hand. CurrentUserID and ServiceCatalog cannot answer in that case.
var ErrNoAuthResult = errors.New("connection has no recorded Keystone authentication result")

// CatalogEndpoint is one endpoint of a token service catalog entry.
type CatalogEndpoint struct {
	ID        string
	Interface string
	Region    string
	RegionID  string
	URL       string
}

// CatalogService is one service entry of the token service catalog.
type CatalogService struct {
	ID        string
	Name      string
	Type      string
	Endpoints []CatalogEndpoint
}

// AuthToken returns the token currently held by the shared provider without any
// HTTP request. It is empty when no token has been obtained, and unlike the
// keystoneauth accessor it never re-authenticates an expired token.
func (c *Connection) AuthToken() string {
	return c.provider.Token()
}

// Authorize returns the current token, authenticating through the provider's
// reauthentication function first when no token is held. It does not refresh a
// token that is merely expired; the failed request triggers native reauth.
// Errors wrap the native cause and context cancellation stays observable.
func (c *Connection) Authorize(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("authorize: %w", err)
	}
	if token := c.provider.Token(); token != "" {
		return token, nil
	}
	if err := c.provider.Reauthenticate(ctx, ""); err != nil {
		return "", fmt.Errorf("authorize: %w", err)
	}
	token := c.provider.Token()
	if token == "" {
		return "", invalid("authorize: provider has no token and no reauthentication function")
	}
	return token, nil
}

// CurrentUserID returns the user ID recorded in the Keystone token response.
// It makes no request and returns ErrNoAuthResult when no response is recorded.
func (c *Connection) CurrentUserID() (string, error) {
	r, ok := c.provider.GetAuthResult().(interface {
		ExtractUser() (*tokens.User, error)
	})
	if !ok {
		return "", ErrNoAuthResult
	}
	user, err := r.ExtractUser()
	if err != nil {
		return "", fmt.Errorf("decode token user: %w", err)
	}
	if user == nil {
		return "", nil
	}
	return user.ID, nil
}

// ServiceCatalog returns a copy of the service catalog recorded in the Keystone
// token response. It makes no request and returns ErrNoAuthResult when no
// response is recorded. An absent catalog yields an empty slice.
func (c *Connection) ServiceCatalog() ([]CatalogService, error) {
	r, ok := c.provider.GetAuthResult().(interface {
		ExtractServiceCatalog() (*tokens.ServiceCatalog, error)
	})
	if !ok {
		return nil, ErrNoAuthResult
	}
	catalog, err := r.ExtractServiceCatalog()
	if err != nil {
		return nil, fmt.Errorf("decode token service catalog: %w", err)
	}
	out := make([]CatalogService, 0, len(catalog.Entries))
	for _, entry := range catalog.Entries {
		service := CatalogService{ID: entry.ID, Name: entry.Name, Type: entry.Type, Endpoints: make([]CatalogEndpoint, 0, len(entry.Endpoints))}
		for _, e := range entry.Endpoints {
			service.Endpoints = append(service.Endpoints, CatalogEndpoint{ID: e.ID, Interface: e.Interface, Region: e.Region, RegionID: e.RegionID, URL: e.URL})
		}
		out = append(out, service)
	}
	return out, nil
}

// WithCloudLocation supplies owned configured facts when an adopted provider
// cannot recover cloud/authentication configuration. It replaces the complete
// location, including the project scope ID, and is accepted by FromProvider.
func WithCloudLocation(value resource.CloudLocation) ConnectionOption {
	owned := value.Clone()
	return func(options *connectionOptions) error {
		copy := owned.Clone()
		options.cloudLocation = &copy
		return nil
	}
}

// CurrentLocation snapshots cloud, region and recorded token scope without
// making an API or authentication request. Unknown facts remain null.
func (c *Connection) CurrentLocation() (resource.CloudLocation, error) {
	if c == nil || c.provider == nil {
		return resource.CloudLocation{}, invalid("Connection is required")
	}
	if c.options.cloudLocation != nil {
		return c.options.cloudLocation.Clone(), nil
	}
	value := c.options.locationFacts.Clone()
	id, err := cloudlocation.ProjectID(c.provider)
	if err != nil {
		return resource.CloudLocation{}, err
	}
	value.Project.ID = id
	return value, nil
}

func configuredLocation(cloudName string, auth gophercloud.AuthOptions) resource.CloudLocation {
	var value resource.CloudLocation
	if cloudName != "" {
		value.Cloud = locationString(cloudName)
	} else if parsed, err := url.Parse(auth.IdentityEndpoint); err == nil && parsed.Hostname() != "" {
		value.Cloud = locationString(parsed.Hostname())
	}
	value.Project.Name = locationNonempty(auth.TenantName)
	if scope := auth.Scope; scope != nil && (scope.ProjectName != "" || scope.ProjectID != "") {
		if scope.ProjectName != "" {
			value.Project.Name = locationString(scope.ProjectName)
		}
		value.Project.DomainID = locationNonempty(scope.DomainID)
		value.Project.DomainName = locationNonempty(scope.DomainName)
	} else if auth.TenantName != "" {
		value.Project.DomainID = locationNonempty(auth.DomainID)
		value.Project.DomainName = locationNonempty(auth.DomainName)
	}
	return value
}

func locationString(value string) *string { copy := value; return &copy }
func locationNonempty(value string) *string {
	if value == "" {
		return nil
	}
	return locationString(value)
}
