package secretstores_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func ListNonDefaultBackendsSecretstores2(ctx context.Context, client *gophercloud.ServiceClient) ([]*secretstores.SecretStore, error) {
	return secretstores.New(client).All(ctx,
		secretstores.WithListOptions(secretstores.ListOpts{Status: "ACTIVE", Limit: 10}),
		request.Custom(func(c *request.Config[secretstores.ListOpts]) error {
			v := false
			c.Options.GlobalDefault = &v
			return nil
		}),
		request.MaxItems(5),
		request.Paginated(false),
	)
}
