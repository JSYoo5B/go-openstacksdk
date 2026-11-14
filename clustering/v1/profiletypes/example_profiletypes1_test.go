package profiletypes_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FilterProfileTypesProfiletypes1(ctx context.Context, conn *sdk.Connection, typeName string) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for profile, err := range service.ProfileTypes.List(ctx,
		profiletypes.WithListFilter("name", typeName),
		request.MaxItems(50), request.Paginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(profile.Name, profile.Version)
	}
	return nil
}
