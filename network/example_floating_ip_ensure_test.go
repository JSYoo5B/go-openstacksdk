package network_test

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_floatingIpEnsure() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := runFloatingIpEnsure(ctx); err != nil {
		log.Fatal(err)
	}
}

func runFloatingIpEnsure(ctx context.Context) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	assignment, err := service.FloatingIPs.Ensure(ctx, network.EnsureFloatingIPRequest{
		Server: resource.Name("web-01"), Network: resource.Name("public"),
	}, network.WithEnsureNATDestination(resource.Name("private")),
		network.WithEnsureWait(resource.WithTimeout(time.Minute)))
	if err != nil {
		if assignment != nil && assignment.FloatingIP != nil {
			log.Printf("floating IP remains: id=%s reused=%t allocated=%t",
				assignment.FloatingIP.ID, assignment.Reused, assignment.Allocated)
		}
		return err
	}
	floating := assignment.FloatingIP
	fmt.Println(floating.ID, floating.FloatingIP, floating.PortID, floating.FixedIP, floating.Status)
	fmt.Println("reused", assignment.Reused, "allocated", assignment.Allocated)
	return nil
}
