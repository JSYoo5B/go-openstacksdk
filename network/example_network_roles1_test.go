package network_test

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func Example_networkRoles1() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runNetworkRoles1(ctx); err != nil {
		log.Fatal(err)
	}
}

func runNetworkRoles1(ctx context.Context) error {
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud("dev"),
		sdk.WithNetworkRoles(
			network.WithConfiguredNetworks(
				network.ConfiguredNetwork{
					Name:                 "public",
					RoutesIPv4Externally: true,
					RoutesIPv6Externally: true,
					NATSource:            true,
				},
				network.ConfiguredNetwork{
					Name:             "private",
					NATDestination:   true,
					DefaultInterface: true,
				},
			),
		),
	)
	if err != nil {
		return err
	}
	fmt.Printf("external=%t internal=%t\n",
		conn.UseExternalNetwork(), conn.UseInternalNetwork())

	roles, err := conn.GetNetworkRoles(ctx)
	if err != nil {
		return err
	}
	for _, net := range roles.ExternalIPv4 {
		fmt.Printf("external IPv4: %s %s router_external=%t provider=%q\n",
			net.ID, net.Name, net.RouterExternal, net.ProviderPhysicalNetwork)
	}
	printNetworkNetworkRoles1("NAT source", roles.NATSource)
	printNetworkNetworkRoles1("NAT destination", roles.NATDestination)

	defaultNetwork, err := conn.GetDefaultNetwork(ctx)
	if err != nil {
		return err
	}
	printNetworkNetworkRoles1("default", defaultNetwork)

	conn.ResetNetworkRoles()
	fresh, err := conn.GetNetworkRoles(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("fresh external IPv4 networks: %d\n", len(fresh.ExternalIPv4))
	return nil
}

func printNetworkNetworkRoles1(label string, net *network.RoleNetwork) {
	if net == nil {
		fmt.Printf("%s: none\n", label)
		return
	}
	fmt.Printf("%s: %s %s\n", label, net.ID, net.Name)
}
