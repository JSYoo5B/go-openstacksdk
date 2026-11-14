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

func Example_floatingIpQueries() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	source := flag.String("source", "", "cloud default, neutron, nova or none")
	id := flag.String("id", "", "floating IP identifier; empty lists IPs")
	filters := flag.String("filters", "", "JSON query or local filter")
	direct := flag.Bool("direct", false, "use direct GET for a UUID-shaped ID")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := runFloatingIpQueries(ctx, *cloud, *source, *id, *filters, *direct); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runFloatingIpQueries(ctx context.Context, cloud, source, id, filters string, direct bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud),
		sdk.WithMicroversion(sdk.Compute, "2.35"))
	if err != nil {
		return err
	}
	options := []compute.FloatingIPQueryOption{
		compute.WithFloatingIPQueryTimeout(time.Minute),
		func(c *compute.FloatingIPQueryOpts) error { c.DirectGet = direct; return nil },
	}
	if source != "" {
		options = append(options,
			func(c *compute.FloatingIPQueryOpts) error {
				v := compute.FloatingIPSource(source)
				c.Source = &v
				return nil
			})
	}
	if filters != "" {
		options = append(options,
			compute.WithFloatingIPQueryFilters(json.RawMessage(filters)))
	}
	var value json.RawMessage
	var pages int
	if id == "" {
		result, queryErr := conn.ListFloatingIPs(ctx, options...)
		err = queryErr
		if result != nil {
			value, pages = result.Value, len(result.Pages)
		}
	} else {
		result, queryErr := conn.GetFloatingIP(ctx,
			compute.GetFloatingIPRequest{ID: id}, options...)
		err = queryErr
		if result != nil {
			value, pages = result.Value, len(result.Pages)
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(map[string]any{"value": value, "pages": pages})
	return errors.Join(err, encodeErr)
}
