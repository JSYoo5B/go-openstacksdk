package compute_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_serverNovaFloatingIp() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	id := flag.String("server-id", "", "existing Nova server ID")
	mode := flag.String("mode", "dispatch", "dispatch or list")
	pool := flag.String("pool", "", "literal Nova pool")
	ips := flag.String("ips", "", "comma-separated floating IPv4 addresses")
	fixed := flag.String("fixed", "", "fixed IPv4 for addFloatingIp")
	reuse := flag.Bool("reuse", true, "reuse a free IP for pool/auto")
	wait := flag.Bool("wait", false, "observe each address in raw Nova")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := runServerNovaFloatingIp(ctx, *cloud, *id, *mode, *pool, *ips, *fixed, *reuse, *wait); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServerNovaFloatingIp(ctx context.Context, cloud, id, mode, pool, ips, fixed string, reuse, wait bool) error {
	if id == "" || (mode != "dispatch" && mode != "list") {
		return fmt.Errorf("provide -server-id and -mode dispatch or list")
	}
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud(cloud),
		sdk.WithMicroversion(sdk.Compute, "2.35"),
		sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNova)),
	)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	server, err := service.Servers.Get(ctx, id)
	if err != nil {
		return err
	}
	if server == nil || server.ID != id {
		return fmt.Errorf("server GET did not return %q", id)
	}
	addresses := []string{}
	if ips != "" {
		for _, address := range strings.Split(ips, ",") {
			addresses = append(addresses, strings.TrimSpace(address))
		}
	}
	destination := []network.EnsureFloatingIPOption{network.WithEnsureReuse(reuse)}
	if fixed != "" {
		destination = append(destination, network.WithEnsureFixedAddress(fixed))
	}
	automatic := []compute.AutomaticFloatingIPOption{
		compute.WithAutomaticEnsureOptions(destination...),
	}
	if pool != "" {
		automatic = append(automatic, compute.WithFloatingIPPool(resource.Name(pool)))
	}
	if len(addresses) != 0 {
		automatic = append(automatic, compute.WithFloatingIPAddresses(addresses...))
	}
	options := []compute.ServerIPOption{
		compute.WithServerIPAutomaticOptions(automatic...),
		compute.WithServerIPWait(wait),
	}
	var result *compute.AutomaticServerIPResult
	if mode == "list" {
		result, err = conn.AddIPList(ctx, server, addresses, options...)
	} else {
		result, err = conn.AddIPsToServer(ctx, compute.AutomaticFloatingIPRequest{Server: server}, options...)
	}
	if result != nil {
		fmt.Printf("mode=%s observed=%t\n", result.Mode, result.Observed)
		printAssignmentServerNovaFloatingIp("latest", result.NovaAssignment)
		for _, attempt := range result.Attempts {
			fmt.Printf("attempt=%d requested=%s completed=%t observed=%t error=%v\n",
				attempt.Index, attempt.RequestedAddress, attempt.Completed, attempt.Observed, attempt.Error)
			printAssignmentServerNovaFloatingIp("attempt", attempt.NovaAssignment)
		}
	}
	return err
}

func printAssignmentServerNovaFloatingIp(label string, assignment *compute.NovaFloatingIPAssignment) {
	if assignment == nil {
		return
	}
	fmt.Printf("%s reused=%t allocated=%t already_attached=%t action_accepted=%t\n",
		label, assignment.Reused, assignment.Allocated, assignment.AlreadyAttached, assignment.ActionAccepted)
	if ip := assignment.FloatingIP; ip != nil {
		fmt.Printf("ip=%s address=%s pool=%s fixed=%s instance=%s\n",
			ip.ID, ip.Address, ip.Pool, optionalServerNovaFloatingIp(ip.FixedAddress), optionalServerNovaFloatingIp(ip.InstanceID))
	}
	if response := assignment.ActionResponse; response != nil {
		fmt.Printf("action_status=%d\n", response.StatusCode)
	}
}

func optionalServerNovaFloatingIp(value *string) string {
	if value == nil {
		return "<null>"
	}
	return *value
}
