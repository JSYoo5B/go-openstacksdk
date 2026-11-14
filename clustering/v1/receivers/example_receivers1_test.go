package receivers_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FindReceiverStrictReceivers1(ctx context.Context, conn *sdk.Connection) (*receivers.Receiver, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	return service.Receivers.FindIdentity(ctx, "scale_out",
		request.IgnoreMissing(false))
}
