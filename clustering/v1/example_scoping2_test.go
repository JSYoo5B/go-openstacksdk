package v1_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateScopedNodeScoping2(ctx context.Context, conn *sdk.Connection, collectionPath, nodeID string) (*nodes.Node, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := service.Nodes.AtBasePath(collectionPath)
	if err != nil {
		return nil, err
	}
	tracked, err := scope.Load(ctx, resource.ID(nodeID))
	if err != nil {
		return nil, err
	}
	if err := tracked.Edit(nodes.UpdateOpts{}, request.Custom(func(c *request.Config[nodes.UpdateOpts]) error {
		c.Options.Role = request.Present("worker")
		return nil
	})); err != nil {
		return nil, err
	}
	accepted, err := tracked.Commit(ctx)
	if err != nil {
		return nil, err
	}
	// Later tracked.Refresh(ctx) uses the same collection and nodeID.
	return accepted, nil
}
