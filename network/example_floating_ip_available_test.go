package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_floatingIpAvailable() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	candidates := flag.String("networks", "public", "ordered network names")
	server := flag.String("server-id", "", "optional existing server ID")
	project := flag.String("project", "", "optional reuse filter only")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := runFloatingIpAvailable(ctx, *cloud, *candidates, *server, *project); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runFloatingIpAvailable(ctx context.Context, cloud, candidates, serverID, project string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	input := network.AvailableFloatingIPRequest{}
	if candidates != "" {
		for _, name := range strings.Split(candidates, ",") {
			input.Networks = append(input.Networks, resource.Name(strings.TrimSpace(name)))
		}
	}
	if serverID != "" {
		input.Server = resource.ID(serverID)
	}
	options := []network.AvailableFloatingIPOption{
		network.WithAvailableTimeout(time.Minute),
	}
	if project != "" {
		options = append(options, network.WithAvailableProject(project))
	}
	result, err := service.FloatingIPs.Available(ctx, input, options...)
	if result != nil {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(result); encodeErr != nil {
			return errors.Join(err, encodeErr)
		}
	}
	return err
}
