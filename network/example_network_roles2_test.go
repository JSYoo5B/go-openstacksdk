package network_test

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_networkRoles2() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := runNetworkRoles2(ctx); err != nil {
		log.Fatal(err)
	}
}

func runNetworkRoles2(ctx context.Context) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
	if err != nil {
		return err
	}
	roles, err := conn.GetNetworkRoles(ctx)
	if err != nil {
		return err
	}
	if roles.DefaultNetwork != nil {
		fmt.Printf("default NIC: %s\n", roles.DefaultNetwork.ID)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	result, err := service.Servers.CreateWithFloatingIP(ctx,
		compute.CreateServerWithFloatingIPRequest{
			Server: compute.CreateServerRequest{
				Name:   "web",
				Image:  resource.Name("ubuntu"),
				Flavor: resource.Name("c2"),
			},
		},
	)
	if err != nil {
		if result != nil && result.Server != nil {
			fmt.Printf("known server: %s\n", result.Server.ID)
		}
		if result != nil && result.Assignment != nil && result.Assignment.FloatingIP != nil {
			fmt.Printf("known floating IP: %s\n", result.Assignment.FloatingIP.ID)
		}
		return err
	}
	fmt.Printf("server=%s floating IPv4=%s\n",
		result.Server.ID, result.Assignment.FloatingIP.FloatingIP)
	return nil
}
