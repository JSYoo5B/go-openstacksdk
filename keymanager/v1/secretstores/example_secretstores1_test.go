package secretstores_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
	"github.com/gophercloud/gophercloud/v2"
)

func ReadBackendsSecretstores1(ctx context.Context, client *gophercloud.ServiceClient) (*secretstores.SecretStore, *secretstores.SecretStore, error) {
	api := secretstores.New(client)
	global, err := api.GetGlobalDefault(ctx)
	if err != nil {
		return nil, nil, err
	}
	preferred, err := api.GetPreferred(ctx)
	return global, preferred, err
}
