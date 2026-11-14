package compute_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_serverDefaultNetwork() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	image := flag.String("image", "ubuntu", "exact image name")
	flavor := flag.String("flavor", "small", "exact flavor name")
	defaultNetwork := flag.String("default-network", "", "exact Connection default network name")
	noDefaultNetwork := flag.Bool("no-default-network", false, "disable Connection and YAML default network")
	network := flag.String("network", "", "exact network name for this server")
	mode := flag.String("network-mode", "", "Nova auto or none mode for this server")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runServerDefaultNetwork(ctx, *cloud, *image, *flavor, *defaultNetwork, *noDefaultNetwork, *network, *mode); err != nil {
		log.Fatal(err)
	}
}

func runServerDefaultNetwork(ctx context.Context, cloud, image, flavor, defaultNetwork string, noDefaultNetwork bool, network, mode string) error {
	opts := []sdk.ConnectionOption{
		sdk.WithCloud(cloud),
		sdk.WithMicroversion(sdk.Compute, "2.37"),
	}
	if defaultNetwork != "" {
		opts = append(opts, sdk.WithDefaultNetwork(resource.Name(defaultNetwork)))
	}
	if noDefaultNetwork {
		opts = append(opts, sdk.WithoutDefaultNetwork())
	}
	conn, err := sdk.Connect(ctx, opts...)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	var createOpts []compute.CreateServerOption
	if network != "" {
		createOpts = append(createOpts, compute.WithNetworks(resource.Name(network)))
	}
	if mode != "" {
		createOpts = append(createOpts, compute.WithNetworkMode(mode))
	}
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
		Name: "web-01", Image: resource.Name(image), Flavor: resource.Name(flavor),
	}, createOpts...)
	if err != nil {
		return err
	}
	fmt.Println(server.ID)
	return nil
}
