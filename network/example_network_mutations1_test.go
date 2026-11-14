package network_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_networkMutations1() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := runNetworkMutations1(ctx); err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			log.Printf("accepted response: status=%d", response.StatusCode)
		}
		log.Fatal(err)
	}
}

func runNetworkMutations1(ctx context.Context) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	created, err := service.CreateNetwork(ctx,
		network.CreateNetworkRequest{Name: "private"},
		network.WithNetworkPortSecurity(false))
	if err != nil {
		if created != nil {
			log.Printf("returned network: id=%q", created.ID)
		}
		return err
	}
	fmt.Println(created.ID, created.Name)
	updated, err := service.UpdateNetwork(ctx, resource.ID(created.ID),
		network.WithNetworkName("private-renamed"),
		network.WithNetworkAdminStateUp(false),
		network.WithNetworkDNSDomain(""))
	if err != nil {
		return err
	}
	fmt.Println(updated.ID, updated.Name)
	deleted, err := service.DeleteNetwork(ctx, resource.ID(created.ID))
	fmt.Println("deleted", deleted)
	return err
}
