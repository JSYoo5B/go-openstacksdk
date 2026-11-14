package nodes_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FindNodeStrictNodes1(ctx context.Context, conn *sdk.Connection) (*nodes.Node, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	return service.Nodes.FindIdentity(ctx, "worker_1",
		request.IgnoreMissing(false))
}
