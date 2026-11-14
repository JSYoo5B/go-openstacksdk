package network_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_floatingIpAttach() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	server := flag.String("server-id", "", "existing Nova server ID")
	ipID := flag.String("ip-id", "", "existing Floating IP ID")
	address := flag.String("address", "", "existing floating IPv4 address")
	fixed := flag.String("fixed", "", "exact destination fixed IPv4")
	execute := flag.Bool("execute", false, "execute the prepared attachment")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := runFloatingIpAttach(ctx, *cloud, *server, *ipID, *address, *fixed, *execute); err != nil {
		log.Fatal(err)
	}
}

func runFloatingIpAttach(ctx context.Context, cloud, server, ipID, address, fixed string, execute bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	ip := resource.Name(address)
	if ipID != "" {
		ip = resource.ID(ipID)
	}
	options := []network.AttachFloatingIPOption{
		network.WithAttachWait(resource.WithTimeout(time.Minute)),
	}
	if fixed != "" {
		options = append(options, network.WithAttachFixedAddress(fixed))
	}
	plan, err := service.FloatingIPs.PrepareAttach(ctx,
		network.AttachFloatingIPRequest{Server: resource.ID(server), IP: ip}, options...)
	if err != nil {
		return err
	}
	fmt.Printf("selection: %+v\n", plan.Selection())
	if !execute {
		return nil
	}
	result, err := service.FloatingIPs.AttachPrepared(ctx, plan)
	if result != nil && result.FloatingIP != nil {
		fmt.Printf("IP: %s address=%s port=%s reused=%t allocated=%t\n",
			result.FloatingIP.ID, result.FloatingIP.FloatingIP, result.FloatingIP.PortID,
			result.Reused, result.Allocated)
	}
	return err
}
