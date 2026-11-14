package v1_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateTrackedClusterAsync1(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	tracked, err := service.Clusters.Load(ctx, resource.Name("workers"))
	if err != nil {
		return nil, err
	}
	if err := tracked.Edit(clusters.UpdateOpts{},
		request.Custom(func(c *request.Config[clusters.UpdateOpts]) error {
			c.Options.Name = request.Present("workers_renamed")
			return nil
		}),
		clusters.WithUpdateTimeout(0),
		clusters.WithUpdateMetadata(map[string]any{"team": "platform"}),
		clusters.WithUpdateProfileOnlyNull()); err != nil {
		return nil, err
	}
	accepted, err := tracked.Commit(ctx, request.Header("X-Audit-Tag", "rename"))
	if err != nil {
		return nil, err
	}
	if accepted.Operation != nil {
		fmt.Println(accepted.Operation.ActionID, accepted.Operation.Location)
	}
	fmt.Println(tracked.Dirty(), tracked.Response().Body, accepted.UserMetadata)
	return accepted, nil
}
