# Cloud Network Lists Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("list", "search"), default="list")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
filters = json.loads(args.filters)
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "search":
        result = conn.search_networks(args.name, filters=filters)
    else:
        result = conn.list_networks(filters=filters)
    values = [value.to_dict() if hasattr(value, "to_dict") else value for value in result]
    print(json.dumps(values, indent=2))
```

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	mode := flag.String("mode", "list", "list or search")
	name := flag.String("name", "", "literal server-side network name for search")
	filters := flag.String("filters", "{}", "JSON filters; search also accepts key/value pairs")
	flag.Parse()
	if *mode != "list" && *mode != "search" {
		log.Fatal("mode must be list or search")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	options := []network.NetworkCloudListOption{
		network.WithNetworkCloudFilters(json.RawMessage(*filters)),
		request.Custom(func(c *network.NetworkCloudListOpts) error { loc := location.Clone(); c.CurrentLocation = &loc; return nil }),
	}
	var result *network.NetworkCloudRecords
	if *mode == "search" {
		result, err = service.SearchNetworkRecords(ctx, *name, options...)
	} else {
		result, err = service.ListNetworkRecords(ctx, options...)
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.JSON == nil {
		log.Fatal("result contains an opaque lazy generator; inspect result.Values instead")
	}
	var formatted json.RawMessage = result.JSON
	data, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [Neutron 27 network policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/network.py)
