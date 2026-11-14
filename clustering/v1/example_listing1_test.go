package v1_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListProfilesListing1(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for profile, err := range service.Profiles.List(ctx,
		profiles.WithListOptions(profiles.ListOpts{Limit: 20}),
		request.MaxItems(50), request.Paginated(true),
		profiles.WithListFilter("metadata", map[string]any{"team": "infra"})) {
		if err != nil {
			return err
		}
		fmt.Println(profile.ID)
	}
	return nil
}

func FirstProfilesPageListing1(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for profile, err := range service.Profiles.Resources.List(ctx,
		resource.WithMaxItems(5), resource.WithPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(profile.ID)
	}
	return nil
}

func FirstClusterPoliciesPageListing1(ctx context.Context, conn *sdk.Connection,
	cluster resource.Ref) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	scope, err := service.ClusterPolicies.InCluster(ctx, cluster)
	if err != nil {
		return err
	}
	for binding, err := range scope.List(ctx,
		request.MaxItems(5), request.Paginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(binding.ID, binding.PolicyID)
	}
	return nil
}
