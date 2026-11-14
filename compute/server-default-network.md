# Server Default Network

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev", compute_api_version="2.37")
server = conn.create_server(
    name="web-01",
    image="ubuntu",
    flavor="small",
    auto_ip=False,
    wait=False,
)
print(server.id)
```

```go
package main

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

func main() {
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
	if err := run(ctx, *cloud, *image, *flavor, *defaultNetwork, *noDefaultNetwork, *network, *mode); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, image, flavor, defaultNetwork string, noDefaultNetwork bool, network, mode string) error {
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
```

## References

- [Server Network Interfaces](server-network-interfaces.md)
- [Network Roles](../network/network-roles.md)
- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1084-L1110)
- [Cloud Region](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/config/cloud_region.py#L1475-L1481)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L252-L264)
- [shared network discovery](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L110-L365)
- [Network Mutations](../network/network-mutations.md)
- [Connection Network Role Consumers Test](../connectiontest/connection_network_role_consumers_test.go)
- [Default Network Test](default_network_test.go)
- [Connection Default Network Test](../connectiontest/connection_default_network_test.go)
- [Cloud Config Test](../internal/cloudconfig/cloudconfig_test.go)
- [Connection Cloud Network Test](../connectiontest/connection_cloud_network_test.go)
