# Cloud Subnet Create Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--network", required=True)
parser.add_argument("--kwargs", default='{"cidr":"10.61.0.0/24"}')
args = parser.parse_args()
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    subnet = conn.create_subnet(args.network, **kwargs)
    print(json.dumps(subnet.to_dict(), indent=2))
```

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func run(ctx context.Context, cloud, networkName, keywords string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	result, err := conn.CreateSubnetRecord(ctx, networkName,
		network.WithSubnetCreateJSON(json.RawMessage(keywords)))
	if result != nil {
		if result.Network != nil {
			fmt.Fprintf(os.Stderr, "network lookup status=%d\n", result.Network.StatusCode)
		}
		if result.SubnetPool != nil {
			fmt.Fprintf(os.Stderr, "subnetpool lookup status=%d\n", result.SubnetPool.StatusCode)
		}
		if result.StatusCode != 0 {
			fmt.Fprintf(os.Stderr, "subnet POST status=%d request-id=%q\n",
				result.StatusCode, result.Header.Get("X-Request-Id"))
		}
	}
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(result.Resource, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	networkName := flag.String("network", "", "literal network ID or exact name")
	keywords := flag.String("kwargs", `{"cidr":"10.61.0.0/24"}`, "Source cloud keyword JSON object")
	flag.Parse()
	if *networkName == "" {
		fmt.Fprintln(os.Stderr, "network is required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx, *cloud, *networkName, *keywords); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

## References

- [Install](../docs/install.md)
- [Subnet](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/subnet.py#L60-L118)
- [cloud create_subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2401-L2569)
- [Subnet descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet.py)
- [Proxy._create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L771-L800)
