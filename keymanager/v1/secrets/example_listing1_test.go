package secrets_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listPlainSecretsListing1(ctx context.Context, client *gophercloud.ServiceClient) ([]*secrets.Secret, error) {
	return secrets.New(client).Resources.All(ctx,
		resource.WithFilter("algorithm", "aes"),
		resource.WithFilter("content_types", map[string]any{"default": "text/plain"}),
		resource.WithMaxItems(20))
}
