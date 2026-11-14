package secretconsumers_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretconsumers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func AssociateSecretconsumers1(ctx context.Context, client *gophercloud.ServiceClient, secretID string) error {
	scope, err := secretconsumers.New(client).InSecret(ctx, resource.ID(secretID))
	if err != nil {
		return err
	}
	response, err := scope.Create(ctx, secretconsumers.ConsumerOpts{
		Service: "image", ResourceType: "image", ResourceID: "image-id",
	})
	if err != nil {
		return err
	}
	_ = response.SecretRef // HTTP evidence; never automatically followed.
	return nil
}
