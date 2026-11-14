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
)

func Example_floatingIpDelete1() {
	if err := runFloatingIpDelete1(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runFloatingIpDelete1() error {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	id := flag.String("id", "", "floating IP ID to delete")
	retries := flag.Int("retry", 1, "additional deletes; zero skips verification")
	source := flag.String("source", "", "optional neutron, nova, or none override")
	direct := flag.Bool("direct", false, "use direct GET for a UUID verification ID")
	flag.Parse()
	if *id == "" {
		return errors.New("-id is required")
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
	opts := []compute.FloatingIPDeleteOption{
		func(c *compute.FloatingIPDeleteOpts) error { c.Retries = *retries; return nil },
		func(c *compute.FloatingIPDeleteOpts) error { c.DirectGet = *direct; return nil },
		compute.WithFloatingIPDeleteTimeout(time.Minute),
	}
	if *source != "" {
		opts = append(opts, func(c *compute.FloatingIPDeleteOpts) error {
			v := compute.FloatingIPSource(*source)
			c.Source = &v
			return nil
		})
	}
	result, deleteErr := conn.DeleteFloatingIP(ctx, compute.DeleteFloatingIPRequest{ID: *id}, opts...)
	var summary map[string]any
	if result != nil {
		accepted := 0
		for _, attempt := range result.Attempts {
			if attempt.Accepted {
				accepted++
			}
		}
		summary = map[string]any{
			"id": result.ID, "backend": result.Backend,
			"deleted": result.Deleted, "absent": result.Absent, "down": result.Down,
			"attempts": len(result.Attempts), "accepted_attempts": accepted,
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return errors.Join(deleteErr, encoder.Encode(summary))
}
