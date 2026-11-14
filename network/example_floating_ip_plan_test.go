package network_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_floatingIpPlan() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	serverID := flag.String("server-id", "", "existing Nova server ID")
	external := flag.String("network", "", "exact external network name; empty uses roles/router")
	project := flag.String("project-id", "", "explicit reuse/allocation owner; empty uses recorded scope at execution")
	reuse := flag.Bool("reuse", true, "reuse an attached/free floating IP before allocating")
	execute := flag.Bool("execute", false, "execute the prepared assignment")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := runFloatingIpPlan(ctx, *cloud, *serverID, *external, *project, *reuse, *execute); err != nil {
		log.Fatal(err)
	}
}

func runFloatingIpPlan(ctx context.Context, cloud, serverID, external, project string, reuse, execute bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	request := network.EnsureFloatingIPRequest{Server: resource.ID(serverID)}
	if external != "" {
		request.Network = resource.Name(external)
	}
	options := []network.EnsureFloatingIPOption{
		network.WithEnsureReuse(reuse),
		network.WithEnsureWait(resource.WithTimeout(time.Minute)),
	}
	if project != "" {
		options = append(options, network.WithEnsureProject(project))
	}
	plan, err := service.FloatingIPs.PrepareEnsure(ctx, request, options...)
	if err != nil {
		var absence *network.FloatingIPPlanUnavailableError
		if errors.As(err, &absence) {
			log.Printf("no prepared target: %s", absence.Reason)
		}
		return fmt.Errorf("prepare floating IP: %w", err)
	}
	selection := plan.Selection()
	fmt.Printf("server=%s external=%s port-network=%s port=%s fixed4=%s\n",
		selection.ServerID, selection.NetworkID, selection.PortNetworkID,
		selection.PortID, selection.FixedIPv4)
	if !execute {
		return nil
	}

	assignment, err := service.FloatingIPs.EnsurePrepared(ctx, plan)
	if assignment != nil {
		fmt.Printf("reused=%t allocated=%t\n", assignment.Reused, assignment.Allocated)
		if assignment.FloatingIP != nil {
			ip := assignment.FloatingIP
			fmt.Printf("ip-id=%s floating4=%s port=%s fixed4=%s status=%s\n",
				ip.ID, ip.FloatingIP, ip.PortID, ip.FixedIP, ip.Status)
		}
	}
	if err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			log.Printf("response evidence: status=%d", response.StatusCode)
		}
		return fmt.Errorf("execute floating IP: %w", err)
	}
	return nil
}
