package v1_test

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindClusterStrictFinding2(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return err
	}
	value, err := service.Clusters.FindIdentity(ctx, "workers",
		request.IgnoreMissing(false),
		clusters.WithFindFallback(resource.FindFallbackNotFoundOnly),
		request.Header("X-Vendor-Trace", "lookup"),
		request.Microversion("1.7"))
	if errors.Is(err, resource.ErrNotFound) {
		return fmt.Errorf("workers cluster: %w", err)
	}
	if err != nil {
		return err
	}
	fmt.Println(value.ID, value.Name, value.Header, value.StatusCode)
	return nil
}
