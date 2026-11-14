package containers_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listCertificateContainersListing1(ctx context.Context, client *gophercloud.ServiceClient) ([]*containers.Container, error) {
	return containers.New(client).Resources.All(ctx,
		resource.WithFilters(map[string]any{
			"name": "certificate-container",
			"type": "certificate",
		}),
		resource.WithMaxItems(20))
}
