package compute_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_serverAutomaticIp() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	server := flag.String("server", "web-01", "exact server name, or ID with -server-id")
	byID := flag.Bool("server-id", false, "treat -server as an ID")
	external := flag.String("network", "", "exact external network name; empty uses roles/router")
	project := flag.String("project-id", "", "explicit reuse/allocation owner")
	reuse := flag.Bool("reuse", true, "reuse an attached/free floating IP before allocation")
	enabled := flag.Bool("auto-ip", true, "enable conditional floating IPv4 assignment")
	execute := flag.Bool("execute", false, "assign when needed and observe actual Nova addresses")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := runServerAutomaticIp(ctx, *cloud, *server, *byID, *external, *project, *reuse, *enabled, *execute); err != nil {
		log.Fatal(err)
	}
}

func runServerAutomaticIp(ctx context.Context, cloud, nameOrID string, byID bool, external, project string, reuse, enabled, execute bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	ref := resource.Name(nameOrID)
	if byID {
		ref = resource.ID(nameOrID)
	}
	server, err := service.Servers.Find(ctx, ref)
	if err != nil {
		return err
	}
	request := compute.AutomaticFloatingIPRequest{Server: server}
	if external != "" {
		request.Network = resource.Name(external)
	}
	ipOptions := []network.EnsureFloatingIPOption{network.WithEnsureReuse(reuse)}
	if project != "" {
		ipOptions = append(ipOptions, network.WithEnsureProject(project))
	}
	options := []compute.AutomaticFloatingIPOption{
		compute.WithAutomaticIPEnabled(enabled),
		compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false)),
		compute.WithAutomaticEnsureOptions(ipOptions...),
		compute.WithAutomaticIPWait(resource.WithTimeout(3*time.Minute), resource.WithPollInterval(time.Second)),
		compute.WithAutomaticIPProgressCallback(func(current *compute.Server) error {
			fmt.Printf("observing server=%s status=%s\n", current.ID, current.Status)
			return nil
		}),
	}
	if !execute {
		decision, err := conn.PlanServerFloatingIP(ctx, request, options...)
		printDecisionServerAutomaticIp(decision)
		return err
	}

	result, err := conn.EnsureServerFloatingIP(ctx, request, options...)
	if result != nil {
		printDecisionServerAutomaticIp(result.Decision)
		fmt.Printf("observed=%t\n", result.Observed)
		if result.Server != nil {
			fmt.Printf("server=%s status=%s\n", result.Server.ID, result.Server.Status)
		}
		if result.Assignment != nil {
			assignment := result.Assignment
			fmt.Printf("reused=%t allocated=%t\n", assignment.Reused, assignment.Allocated)
			if assignment.FloatingIP != nil {
				fmt.Printf("floating4=%s ip-id=%s status=%s\n",
					assignment.FloatingIP.FloatingIP, assignment.FloatingIP.ID,
					assignment.FloatingIP.Status)
			}
		}
	}
	if err != nil {
		if errors.Is(err, resource.ErrUnsupported) {
			log.Printf("the selected backend or scope is not supported by this workflow")
		}
		var response *resource.ResponseError
		if errors.As(err, &response) {
			log.Printf("response evidence: status=%d", response.StatusCode)
		}
		return fmt.Errorf("automatic floating IPv4: %w", err)
	}
	return nil
}

func printDecisionServerAutomaticIp(decision *compute.ServerFloatingIPDecision) {
	if decision == nil {
		return
	}
	fmt.Printf("needed=%t reason=%s backend=%s\n",
		decision.Needed, decision.Reason, decision.Backend)
	if decision.Needed && decision.Backend == compute.FloatingIPNeutron {
		selection := decision.Selection
		fmt.Printf("external=%s port-network=%s port=%s fixed4=%s\n",
			selection.NetworkID, selection.PortNetworkID,
			selection.PortID, selection.FixedIPv4)
	}
}
