package containers_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listExactContainerReferencesListing2(ctx context.Context, client *gophercloud.ServiceClient) ([]*containers.Container, error) {
	return containers.New(client).Resources.All(ctx,
		resource.WithFilter("secret_refs", []any{
			map[string]any{
				"name":       "certificate",
				"secret_ref": "https://barbican.example/v1/secrets/secret-id",
			},
		}),
		resource.WithBodyFilter("status", "ACTIVE"),
		resource.WithPaginated(false))
}
