package services_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/services"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FilterServicesServices1(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	values, err := service.Services.All(ctx,
		services.WithListFilter("state", "up"),
		services.WithListFilter("disabled_reason", nil),
		request.MaxItems(50), request.Paginated(false))
	if err != nil {
		return err
	}
	for _, value := range values {
		fmt.Println(value.ID, value.Host, value.State)
	}
	return nil
}
