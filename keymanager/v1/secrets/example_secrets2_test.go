package secrets_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func fetchBinarySecretSecrets2(ctx context.Context, client *gophercloud.ServiceClient) (*secrets.FetchedSecret, error) {
	value, err := secrets.New(client).Fetch(ctx, resource.ID("secret-id"),
		request.Custom(func(c *request.Config[secrets.FetchOpts]) error {
			v := "application/octet-stream"
			c.Options.ContentType = &v
			return nil
		}),
		request.Header("X-Trace", "caller-trace"))
	// value may contain the successful metadata response even when err
	// describes a later selection, payload HTTP/read, or UTF-8 failure.
	return value, err
}
