package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_floatingIpCreate1() {
	if err := runFloatingIpCreate1(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runFloatingIpCreate1() error {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	networkName := flag.String("network", "", "network name/ID or literal Nova pool")
	serverID := flag.String("server", "", "optional server ID")
	portID := flag.String("port", "", "optional explicit Neutron port ID")
	fixed := flag.String("fixed", "", "optional fixed IP literal")
	nat := flag.String("nat", "", "optional NAT destination name/ID")
	source := flag.String("source", "", "optional neutron, nova, or none override")
	wait := flag.Bool("wait", false, "wait for selected Neutron port allocation")
	waitTimeout := flag.Duration("wait-timeout", time.Minute, "post-allocation wait timeout")
	timeout := flag.Duration("timeout", 0, "optional whole SDK operation cap")
	flag.Parse()

	input := compute.CreateFloatingIPRequest{}
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "network" {
			input.Network = networkName
		}
	})
	opts := []compute.FloatingIPCreateOption{
		func(c *compute.FloatingIPCreateOpts) error { c.PortID = *portID; return nil },
		func(c *compute.FloatingIPCreateOpts) error { c.FixedAddress = *fixed; return nil },
		func(c *compute.FloatingIPCreateOpts) error { c.NATDestination = *nat; return nil },
		compute.WithFloatingIPCreateWait(*wait),
		compute.WithFloatingIPCreateWaitTimeout(*waitTimeout),
	}
	if *serverID != "" {
		opts = append(opts, func(c *compute.FloatingIPCreateOpts) error { c.Server = resource.ID(*serverID); return nil })
	}
	if *source != "" {
		opts = append(opts, func(c *compute.FloatingIPCreateOpts) error {
			v := compute.FloatingIPSource(*source)
			c.Source = &v
			return nil
		})
	}
	if *timeout != 0 {
		opts = append(opts, compute.WithFloatingIPCreateTimeout(*timeout))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud(*cloud),
		sdk.WithMicroversion(sdk.Compute, "2.35"),
	)
	if err != nil {
		return err
	}
	result, createErr := conn.CreateFloatingIP(ctx, input, opts...)
	var summary map[string]any
	if result != nil {
		summary = map[string]any{
			"backend": result.Backend, "allocated": result.Allocated,
			"waited": result.Waited, "observations": len(result.Observations),
			"compatibility_attempted": result.Compatibility != nil,
		}
		if result.AllocationResponse != nil {
			summary["allocation_status"] = result.AllocationResponse.StatusCode
		}
		if result.Cleanup != nil {
			summary["cleanup_deleted"] = result.Cleanup.Deleted
			summary["cleanup_absent"] = result.Cleanup.Absent
			summary["cleanup_down"] = result.Cleanup.Down
			summary["cleanup_attempts"] = len(result.Cleanup.Attempts)
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return errors.Join(createErr, encoder.Encode(summary))
}
