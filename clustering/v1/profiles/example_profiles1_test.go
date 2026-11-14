package profiles_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FindProfileStrictProfiles1(ctx context.Context, conn *sdk.Connection) (*profiles.Profile, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	return service.Profiles.FindIdentity(ctx, "worker_template",
		request.IgnoreMissing(false))
}
