package compute_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_createWithFloatingIp() {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := runCreateWithFloatingIp(ctx); err != nil {
		log.Fatal(err)
	}
}

func runCreateWithFloatingIp(ctx context.Context) error {
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud("dev"),
		sdk.WithMicroversion(sdk.Compute, "2.37"),
	)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	result, err := service.Servers.CreateWithFloatingIP(ctx,
		compute.CreateServerWithFloatingIPRequest{
			Server: compute.CreateServerRequest{
				Name:   "web-01",
				Image:  resource.Name("ubuntu"),
				Flavor: resource.Name("small"),
			},
			FloatingIPNetwork: resource.Name("public"),
		},
		compute.WithWorkflowTimeout(3*time.Minute),
		compute.WithServerOptions(
			compute.WithNetworks(resource.Name("private")),
			compute.WithWait(resource.WithPollInterval(time.Second)),
		),
		compute.WithFloatingIPOptions(
			network.WithEnsureWait(resource.WithPollInterval(time.Second)),
		),
	)
	if err != nil {
		if result != nil {
			if result.Server != nil {
				fmt.Fprintf(os.Stderr, "server=%s status=%s\n",
					result.Server.ID, result.Server.Status)
			}
			if result.Assignment != nil && result.Assignment.FloatingIP != nil {
				assignment := result.Assignment
				fmt.Fprintf(os.Stderr, "floating-ip=%s address=%s reused=%t allocated=%t\n",
					assignment.FloatingIP.ID, assignment.FloatingIP.FloatingIP,
					assignment.Reused, assignment.Allocated)
			}
		}
		return err
	}
	fmt.Printf("server=%s floating-ip=%s\n",
		result.Server.ID, result.Assignment.FloatingIP.FloatingIP)
	return nil
}
