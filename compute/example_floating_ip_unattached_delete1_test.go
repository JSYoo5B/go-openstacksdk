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

func Example_floatingIpUnattachedDelete1() {
	if err := runFloatingIpUnattachedDelete1(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runFloatingIpUnattachedDelete1() error {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	retry := flag.Int("retry", 1, "additional deletes per IP; zero skips verification")
	source := flag.String("source", "", "optional neutron, nova, or none override")
	strict := flag.Bool("strict", false, "strict Nova fallback view")
	direct := flag.Bool("direct", false, "direct UUID verification GET")
	timeout := flag.Duration("timeout", time.Minute, "whole cleanup cap; zero removes SDK cap")
	flag.Parse()

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
		func(c *compute.FloatingIPDeleteOpts) error { c.Retries = *retry; return nil },
		func(c *compute.FloatingIPDeleteOpts) error { c.Strict = *strict; return nil },
		func(c *compute.FloatingIPDeleteOpts) error { c.DirectGet = *direct; return nil },
	}
	if *source != "" {
		opts = append(opts, func(c *compute.FloatingIPDeleteOpts) error {
			v := compute.FloatingIPSource(*source)
			c.Source = &v
			return nil
		})
	}
	if *timeout == 0 {
		opts = append(opts, compute.WithUnlimitedFloatingIPDeleteTimeout())
	} else {
		opts = append(opts, compute.WithFloatingIPDeleteTimeout(*timeout))
	}
	result, cleanupErr := conn.DeleteUnattachedFloatingIPs(ctx, opts...)
	var summary map[string]any
	if result != nil {
		summary = map[string]any{
			"eligible": result.Eligible, "count": result.Count,
			"all_deleted": result.AllDeleted, "items": len(result.Items),
		}
		if result.Inventory != nil {
			summary["inventory_backend"] = result.Inventory.Backend
			summary["inventory_pages"] = len(result.Inventory.Pages)
			summary["inventory_rows"] = len(result.Inventory.FloatingIPs)
		}
		items := make([]map[string]any, 0, len(result.Items))
		for _, item := range result.Items {
			entry := map[string]any{"id": item.ID}
			if item.Error != nil {
				entry["error"] = item.Error.Error()
			}
			if deletion := item.Deletion; deletion != nil {
				entry["backend"] = deletion.Backend
				entry["deleted"] = deletion.Deleted
				entry["absent"] = deletion.Absent
				entry["down"] = deletion.Down
				entry["attempts"] = len(deletion.Attempts)
			}
			items = append(items, entry)
		}
		summary["item_results"] = items
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return errors.Join(cleanupErr, encoder.Encode(summary))
}
