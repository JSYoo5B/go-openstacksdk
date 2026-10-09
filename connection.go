package openstack

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/nativefind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
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
		configuration, loadErr := loadCloudConfiguration(cloudName, o.cloudFiles)
		if loadErr != nil {
			return nil, fmt.Errorf("load cloud configuration: %w", loadErr)
		}
		auth, eo, o.tlsConfig, err = clouds.Parse(configuration.parseOptions...)
		if err != nil {
			return nil, fmt.Errorf("load cloud configuration: %w", err)
		}
		o.configuredDefaultNetwork = configuration.defaultNetwork
		if !o.imageCreatePolicySet {
			o.imageCreatePolicy = configuration.imageCreatePolicy
		}
		if !o.networkRolesSet {
			o.networkRoles = configuration.networkRoles
		}
		if !o.serverAddressesSet {
			o.serverAddresses = configuration.serverAddresses
		}
	} else {
		auth, err = openstack.AuthOptionsFromEnv()
		if err != nil {
			return nil, fmt.Errorf("load environment authentication: %w", err)
		}
	}
	if cloudName == "" && !o.serverAddressesSet {
		o.serverAddresses, err = configuredServerAddresses(nil, nil)
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
	provider, err := config.NewProviderClient(ctx, auth, config.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	if _, present := os.LookupEnv("OS_REGION_NAME"); present || cloudName != "" || eo.Region != "" {
		o.locationFacts.RegionName = locationString(eo.Region)
	}
	return newConnection(provider, o, eo)
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
		c.compute = compute.New(client, compute.Dependencies{
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
