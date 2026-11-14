package clusters_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func RenameTrackedClusterClusters2(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	tracked, err := service.Clusters.Load(ctx, resource.ID("CLUSTER_ID"))
	if err != nil {
		return nil, err
	}
	if err := tracked.Edit(clusters.UpdateOpts{},
		request.Custom(func(c *request.Config[clusters.UpdateOpts]) error {
			c.Options.Name = request.Present("renamed")
			return nil
		})); err != nil {
		return nil, err
	}
	return tracked.Commit(ctx)
}
