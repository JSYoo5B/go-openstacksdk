package events_test

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/events"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FilterEventsEvents1(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for event, err := range service.Events.List(ctx,
		events.WithListFilter("status", "ERROR"),
		events.WithListFilter("meta_data", json.RawMessage(`{"count":9007199254740993}`)),
		request.MaxItems(50), request.Paginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(event.ID, event.Level, event.Status)
	}
	return nil
}
