package v1_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateScopedClusterScoping1(ctx context.Context, conn *sdk.Connection, collectionPath string) (*clusters.Cluster, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := service.Clusters.AtBasePath(collectionPath)
	if err != nil {
		return nil, err
	}
	return scope.Update(ctx, resource.Name("workers"), clusters.UpdateOpts{},
		request.Custom(func(c *request.Config[clusters.UpdateOpts]) error {
			c.Options.Name = request.Present("workers_renamed")
			return nil
		}),
		request.Header("X-Audit-Tag", "rename"))
}
