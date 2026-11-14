package secrets_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func fetchSecretSecrets1(ctx context.Context, client *gophercloud.ServiceClient) (*secrets.FetchedSecret, error) {
	return secrets.New(client).Fetch(ctx, resource.ID("secret-id"))
}
