package compute_test

import (
	"context"
	"encoding/json"
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

func Example_serverIpDispatch() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	serverID := flag.String("server-id", "", "existing Nova server ID")
	pool := flag.String("pool", "", "exact external network name")
	ips := flag.String("ips", "", "comma-separated existing IPv4 addresses")
	execute := flag.Bool("execute", false, "execute assignment and observation")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := runServerIpDispatch(ctx, *cloud, *serverID, *pool, *ips, *execute); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServerIpDispatch(ctx context.Context, cloud, id, pool, ips string, execute bool) error {
	if id == "" {
		return fmt.Errorf("-server-id is required")
	}
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
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
	opts := []compute.AutomaticFloatingIPOption{
		compute.WithAutomaticIPWait(resource.WithTimeout(2 * time.Minute)),
		compute.WithAutomaticEnsureOptions(network.WithEnsureWait(
			resource.WithPollInterval(time.Second))),
	}
	if ips != "" {
		addresses := strings.Split(ips, ",")
		for i := range addresses {
			addresses[i] = strings.TrimSpace(addresses[i])
		}
		opts = append(opts, compute.WithFloatingIPAddresses(addresses...))
	}
	if pool != "" {
		opts = append(opts, compute.WithFloatingIPPool(resource.Name(pool)))
	}
	input := compute.AutomaticFloatingIPRequest{Server: server}
	if !execute {
		decision, err := conn.PlanServerFloatingIP(ctx, input, opts...)
		if decision != nil {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if printErr := encoder.Encode(decision); printErr != nil {
				return printErr
			}
		}
		return err
	}
	result, err := conn.EnsureServerFloatingIP(ctx, input, opts...)
	if result != nil {
		fmt.Printf("mode=%s observed=%t\n", result.Mode, result.Observed)
		if result.Server != nil {
			fmt.Printf("server=%s status=%s\n", result.Server.ID, result.Server.Status)
		}
		for _, attempt := range result.Attempts {
			fmt.Printf("step=%d requested=%s completed=%t observed=%t error=%v\n",
				attempt.Index, attempt.RequestedAddress, attempt.Completed,
				attempt.Observed, attempt.Error)
			if attempt.Assignment != nil && attempt.Assignment.FloatingIP != nil {
				ip := attempt.Assignment.FloatingIP
				fmt.Printf("ip=%s address=%s port=%s\n", ip.ID, ip.FloatingIP, ip.PortID)
			}
		}
	}
	return err
}
