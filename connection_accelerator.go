package gophercloudsdk

import (
	"context"
	accelerator "github.com/JSYoo5B/gophercloudsdk/accelerator/v2"
	"github.com/gophercloud/gophercloud/v2"
	"strings"
)

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
