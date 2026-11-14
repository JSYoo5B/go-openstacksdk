package v1_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func RenameTrackedReceiverReceivers1(ctx context.Context, conn *sdk.Connection) (*receivers.Receiver, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	tracked, err := service.Receivers.Load(ctx, resource.Name("scale_out"))
	if err != nil {
		return nil, err
	}
	if err := tracked.Edit(receivers.UpdateOpts{},
		request.Custom(func(c *request.Config[receivers.UpdateOpts]) error {
			c.Options.Name = request.Present("scale_in")
			return nil
		}),
		request.Custom(func(c *request.Config[receivers.UpdateOpts]) error {
			c.Options.Action = request.Present("CLUSTER_SCALE_IN")
			return nil
		}),
		receivers.WithUpdateParams(map[string]any{"team": "platform"})); err != nil {
		return nil, err
	}
	return tracked.Commit(ctx, request.Header("X-Audit-Tag", "rename"))
}
