package v1_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateTrackedNodeAsync2(ctx context.Context, conn *sdk.Connection) (*nodes.Node, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	tracked, err := service.Nodes.Load(ctx, resource.ID("NODE_ID"))
	if err != nil {
		return nil, err
	}
	if err := tracked.Edit(nodes.UpdateOpts{},
		request.Custom(func(c *request.Config[nodes.UpdateOpts]) error { c.Options.Role = request.Present(""); return nil }),
		nodes.WithUpdateMetadata(map[string]any{}),
		request.Custom(func(c *request.Config[nodes.UpdateOpts]) error {
			c.Options.Tainted = request.Present(false)
			return nil
		})); err != nil {
		return nil, err
	}
	return tracked.Commit(ctx)
}
