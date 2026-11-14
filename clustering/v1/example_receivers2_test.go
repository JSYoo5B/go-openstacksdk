package v1_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateReceiverAtReceivers2(ctx context.Context, conn *sdk.Connection, collectionPath, receiverID string) (*receivers.Receiver, error) {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := service.Receivers.AtBasePath(collectionPath)
	if err != nil {
		return nil, err
	}
	tracked, err := scope.Load(ctx, resource.ID(receiverID))
	if err != nil {
		return nil, err
	}
	if err := tracked.Edit(receivers.UpdateOpts{Action: request.Null[string]()}); err != nil {
		return nil, err
	}
	return tracked.Commit(ctx)
}
