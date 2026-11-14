# Server Addresses

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack
from openstack.cloud import meta

conn = openstack.connect(cloud="dev")
server = conn.compute.find_server("web-01", ignore_missing=False)
print(conn.get_server_public_ip(server))
print(conn.get_server_private_ip(server))

expanded = meta.add_server_interfaces(conn, server)
print(expanded["public_v4"], expanded["public_v6"])
print(expanded["private_v4"], expanded["interface_ip"])
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
	server := flag.String("server", "web-01", "exact server name, or ID with -server-id")
	serverID := flag.Bool("server-id", false, "treat -server as an ID")
	private := flag.Bool("private", false, "prefer internal IPv4 for server access")
	forceIPv4 := flag.Bool("force-ipv4", false, "omit IPv6 from calculated access fields")
	probe := flag.Bool("probe", false, "try TCP reachability among multiple candidates")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx, *cloud, *server, *serverID, *private, *forceIPv4, *probe); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, serverNameOrID string, byID, private, forceIPv4, probe bool) error {
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud(cloud),
		sdk.WithServerAddressPolicy(
			compute.WithPrivateCloud(private),
			compute.WithForceIPv4(forceIPv4),
			compute.WithFloatingIPSource(compute.FloatingIPNeutron),
			compute.WithAddressReachability(probe),
		),
	)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	ref := resource.Name(serverNameOrID)
	if byID {
		ref = resource.ID(serverNameOrID)
	}
	server, err := service.Servers.Find(ctx, ref)
	if err != nil {
		return err
	}
	view, expandErr := service.ExpandServerInterfaces(ctx, server)
	if view != nil {
		fmt.Printf("server=%s public4=%q public6=%q private4=%q interface=%q access4=%q access6=%q\n",
			view.ServerID, view.PublicIPv4, view.PublicIPv6, view.PrivateIPv4,
			view.InterfaceIP, view.AccessIPv4, view.AccessIPv6)
		if view.SupplementalError != nil {
			log.Printf("supplemental address lookup: %v", view.SupplementalError)
		}
	}
	if expandErr != nil {
		return fmt.Errorf("expand server addresses: %w", expandErr)
	}

	publicIP, err := service.GetServerPublicIP(ctx, server)
	if err != nil {
		return fmt.Errorf("public address: %w", err)
	}
	privateIP, err := service.GetServerPrivateIP(ctx, server)
	if err != nil {
		return fmt.Errorf("private address: %w", err)
	}
	fmt.Printf("snapshot public4=%q private4=%q\n", publicIP, privateIP)
	return nil
}
```

## References

- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L859-L873)
- [add_server_interfaces](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L528-L563)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L215-L269)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L147-L190)
- [configured default interface](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L348-L405)
- [Ipaddress](https://docs.python.org/3.13/library/ipaddress.html#ipaddress.IPv4Address.is_private)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L272-L327)
- [Network Roles](../network/network-roles.md)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L474-L525)
- [CreateWithFloatingIP](create-with-floating-ip.md)
- [server_address_selection_test.go](server_address_selection_test.go)
- [Server Address Expansion Test](server_address_expansion_test.go)
- [Connection Server Addresses Test](../connectiontest/connection_server_addresses_test.go)
- [Cloud Config Test](../internal/cloudconfig/cloudconfig_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Server Automatic Ip](server-automatic-ip.md)
