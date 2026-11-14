package secrets_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/gophercloud/gophercloud/v2"
)

func findSecretFinding1(ctx context.Context, client *gophercloud.ServiceClient) (*secrets.FetchedSecret, error) {
	return secrets.New(client).FindIdentity(ctx, "secret-id-or-name")
}
