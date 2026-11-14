package v1_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func ListServicesWithCallOptionsListing2(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for engine, err := range service.Services.List(ctx,
		request.Microversion("1.7"),
		request.Header("X-Trace-ID", "engine-list"),
		request.Paginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(engine.ID, engine.State)
	}
	return nil
}
