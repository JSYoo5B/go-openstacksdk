package v2_test

import (
	"context"
	"errors"
	"fmt"

	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/gophercloud/gophercloud/v2"
)

func renameNetworkRevisionUpdates(ctx context.Context, service *networkv2.Service, id, name string, revision int) (*networks.Network, error) {
	options := networks.UpdateOpts{
		Name:           &name,
		RevisionNumber: &revision,
	}
	updated, err := service.Networks.Update(ctx, id, networks.UpdateOpts{},
		networks.WithUpdateOptions(options))
	if err != nil {
		var response gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &response) {
			fmt.Printf("HTTP%d body=%q headers=%v\n",
				response.Actual, response.Body, response.ResponseHeader)
		}
	}
	return updated, err
}

func Example_revisionUpdates() {}
