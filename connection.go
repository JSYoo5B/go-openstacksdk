package gophercloudsdk

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"gophercloudsdk/blockstorage"
	"gophercloudsdk/compute"
	"gophercloudsdk/image"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"

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
		parseOpts := []clouds.ParseOption{clouds.WithCloudName(cloudName)}
		if o.cloudFiles != nil {
			parseOpts = append(parseOpts, clouds.WithLocations(o.cloudFiles...))
		}
		auth, eo, o.tlsConfig, err = clouds.Parse(parseOpts...)
		if err != nil {
			return nil, fmt.Errorf("load cloud configuration: %w", err)
		}
	} else {
		auth, err = openstack.AuthOptionsFromEnv()
		if err != nil {
			return nil, fmt.Errorf("load environment authentication: %w", err)
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
	provider, err := config.NewProviderClient(ctx, auth, config.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
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
		return nil, invalid("FromProvider accepts only endpoint, region, interface and microversion options")
	}
	return newConnection(provider, o, gophercloud.EndpointOpts{})
}

func newConnection(provider *gophercloud.ProviderClient, o connectionOptions, eo gophercloud.EndpointOpts) (*Connection, error) {
	if o.region != nil {
		eo.Region = *o.region
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
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.compute == nil {
		client, err := c.serviceClient(ctx, Compute)
		if err != nil {
			return nil, err
		}
		c.compute = compute.New(client, compute.Dependencies{
			Image: func(ctx context.Context, ref resource.Ref) (string, error) {
				service, err := c.Image(ctx)
				if err != nil {
					return "", err
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
				value, err := service.Networks.Find(ctx, ref)
				if err != nil {
					return "", err
				}
				return value.ID, nil
			},
			Volume: func(ctx context.Context, ref resource.Ref) (string, error) {
				service, err := c.BlockStorage(ctx)
				if err != nil {
					return "", err
				}
				return service.Volumes.ResolveID(ctx, ref)
			},
		})
	}
	return c.compute, nil
}

func (c *Connection) Network(ctx context.Context) (*network.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.network == nil {
		client, err := c.serviceClient(ctx, Network)
		if err != nil {
			return nil, err
		}
		c.network = network.New(client)
	}
	return c.network, nil
}

func (c *Connection) Image(ctx context.Context) (*image.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.image == nil {
		client, err := c.serviceClient(ctx, Image)
		if err != nil {
			return nil, err
		}
		c.image = image.New(client)
	}
	return c.image, nil
}

func (c *Connection) BlockStorage(ctx context.Context) (*blockstorage.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
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
