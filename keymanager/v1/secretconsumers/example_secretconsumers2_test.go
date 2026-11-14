package secretconsumers_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretconsumers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func AssociationsSecretconsumers2(ctx context.Context, client *gophercloud.ServiceClient, secretID string) ([]*secretconsumers.Consumer, error) {
	scope, err := secretconsumers.New(client).InSecret(ctx, resource.ID(secretID))
	if err != nil {
		return nil, err
	}
	return scope.All(ctx,
		secretconsumers.WithListOptions(secretconsumers.ListOpts{Limit: 10, Offset: 0}),
		request.MaxItems(25),
	)
}
