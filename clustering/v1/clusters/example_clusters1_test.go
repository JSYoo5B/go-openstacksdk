package clusters_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FindClusterStrictClusters1(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	return service.Clusters.FindIdentity(ctx, "workers",
		request.IgnoreMissing(false))
}
