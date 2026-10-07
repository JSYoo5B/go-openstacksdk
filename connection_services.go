package gophercloudsdk

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

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
