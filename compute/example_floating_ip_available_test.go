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
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_floatingIpAvailable() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	source := flag.String("source", "", "cloud default, neutron, nova or none")
	pool := flag.String("network", "public", "network name or literal Nova pool")
	server := flag.String("server-id", "", "optional server for new Neutron allocation")
	project := flag.String("project", "", "optional Neutron reuse filter only")
	strict := flag.Bool("strict", false, "omit Nova view compatibility aliases")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := runFloatingIpAvailable(ctx, *cloud, *source, *pool, *server, *project, *strict); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runFloatingIpAvailable(ctx context.Context, cloud, source, pool, serverID, project string, strict bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud),
		sdk.WithMicroversion(sdk.Compute, "2.35"))
	if err != nil {
		return err
	}
	input := compute.AvailableFloatingIPRequest{}
	if pool != "" {
		input.Networks = []resource.Ref{resource.Name(pool)}
	}
	if serverID != "" {
		input.Server = resource.ID(serverID)
	}
	options := []compute.AvailableFloatingIPOption{
		compute.WithAvailableIPTimeout(time.Minute),
		compute.WithAvailableIPStrict(strict),
	}
	if source != "" {
		options = append(options, compute.WithAvailableIPSource(compute.FloatingIPSource(source)))
	}
	if project != "" {
		options = append(options, compute.WithAvailableIPNetworkOptions(
			network.WithAvailableProject(project)))
	}
	result, err := conn.AvailableFloatingIP(ctx, input, options...)
	if result != nil {
		summary := map[string]any{
			"backend": result.Backend, "id": result.ID, "address": result.Address,
			"reused": result.Reused, "allocated": result.Allocated,
		}
		if record := result.FloatingIP; record != nil {
			summary["normalization_source"] = record.NormalizationSource
			summary["normalized"] = record.Normalized
			if record.Resource != nil {
				summary["resource"] = record.Resource.Body
			}
			if record.Wire != nil {
				summary["wire"] = record.Wire.Body
			}
		}
		if result.FallbackError != nil {
			summary["fallback_error"] = result.FallbackError.Error()
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(summary); encodeErr != nil {
			return errors.Join(err, encodeErr)
		}
	}
	return err
}
