package policies_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FindPolicyStrictPolicies1(ctx context.Context, conn *sdk.Connection) (*policies.Policy, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	return service.Policies.FindIdentity(ctx, "placement",
		request.IgnoreMissing(false))
}
