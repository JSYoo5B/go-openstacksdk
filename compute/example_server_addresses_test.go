package compute_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_serverAddresses() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	server := flag.String("server", "web-01", "exact server name, or ID with -server-id")
	serverID := flag.Bool("server-id", false, "treat -server as an ID")
	private := flag.Bool("private", false, "prefer internal IPv4 for server access")
	forceIPv4 := flag.Bool("force-ipv4", false, "omit IPv6 from calculated access fields")
	probe := flag.Bool("probe", false, "try TCP reachability among multiple candidates")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runServerAddresses(ctx, *cloud, *server, *serverID, *private, *forceIPv4, *probe); err != nil {
		log.Fatal(err)
	}
}

func runServerAddresses(ctx context.Context, cloud, serverNameOrID string, byID, private, forceIPv4, probe bool) error {
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud(cloud),
		sdk.WithServerAddressPolicy(
			compute.WithPrivateCloud(private),
			compute.WithForceIPv4(forceIPv4),
			compute.WithFloatingIPSource(compute.FloatingIPNeutron),
			compute.WithAddressReachability(probe),
		),
	)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	ref := resource.Name(serverNameOrID)
	if byID {
		ref = resource.ID(serverNameOrID)
	}
	server, err := service.Servers.Find(ctx, ref)
	if err != nil {
		return err
	}
	view, expandErr := service.ExpandServerInterfaces(ctx, server)
	if view != nil {
		fmt.Printf("server=%s public4=%q public6=%q private4=%q interface=%q access4=%q access6=%q\n",
			view.ServerID, view.PublicIPv4, view.PublicIPv6, view.PrivateIPv4,
			view.InterfaceIP, view.AccessIPv4, view.AccessIPv6)
		if view.SupplementalError != nil {
			log.Printf("supplemental address lookup: %v", view.SupplementalError)
		}
	}
	if expandErr != nil {
		return fmt.Errorf("expand server addresses: %w", expandErr)
	}

	publicIP, err := service.GetServerPublicIP(ctx, server)
	if err != nil {
		return fmt.Errorf("public address: %w", err)
	}
	privateIP, err := service.GetServerPrivateIP(ctx, server)
	if err != nil {
		return fmt.Errorf("private address: %w", err)
	}
	fmt.Printf("snapshot public4=%q private4=%q\n", publicIP, privateIP)
	return nil
}
