package secrets_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func secretPayloadPayload(ctx context.Context, conn *openstack.Connection, secretID string) ([]byte, error) {
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		return nil, err
	}
	return service.Secrets.GetPayload(ctx, secretID,
		secrets.WithGetPayloadOptions(secrets.GetPayloadOpts{
			PayloadContentType: "application/octet-stream",
		}),
		request.Header("X-Trace", "payload"))
}
