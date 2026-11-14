# Cloud Server Group Reads Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--operation", choices=("search", "get"), default="search")
parser.add_argument("--name")
parser.add_argument("--filters", help="JSON dictionary, string expression or other Source value")
args = parser.parse_args()
kwargs = {}
if args.filters is not None:
    kwargs["filters"] = json.loads(args.filters)

def view(value):
    if hasattr(value, "to_dict"):
        return value.to_dict()
    if isinstance(value, list):
        return [view(item) for item in value]
    return value

with openstack.connect(cloud=args.cloud) as conn:
    if args.operation == "search":
        result = conn.search_server_groups(args.name, **kwargs)
    else:
        result = conn.get_server_group(args.name, **kwargs)
    print(json.dumps(view(result), indent=2))
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
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	operation := flag.String("operation", "search", "search or get")
	name := flag.String("name", "", "name, literal ID or search glob")
	filters := flag.String("filters", "", "complete JSON filter; omitted means None")
	flag.Parse()
	if *operation != "search" && *operation != "get" {
		log.Fatal("unknown operation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	options := []compute.ServerGroupCloudReadOption{request.Custom(func(c *compute.ServerGroupCloudReadOpts) error { loc := location.Clone(); c.CurrentLocation = &loc; return nil })}
	if *filters != "" {
		options = append(options, compute.WithServerGroupCloudFilters(json.RawMessage(*filters)))
	}
	var value json.RawMessage
	if *operation == "search" {
		result, cause := service.SearchServerGroups(ctx, *name, options...)
		err = cause
		if result != nil {
			value = result.Value
		}
	} else {
		result, cause := service.GetServerGroup(ctx, *name, options...)
		err = cause
		if result != nil {
			value = result.Value
		}
	}
	if err != nil {
		log.Fatal(err)
	}
	if value == nil {
		value = json.RawMessage("null")
	}
	data, err := json.MarshalIndent(json.RawMessage(value), "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [Install](../docs/install.md)
- [Server Groups](https://github.com/openstack/nova/blob/32.0.0/nova/policies/server_groups.py#L49-L85)
- [search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L216-L242)
- [get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L728-L772)
- [filter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)
- [ServerGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_group.py)
- [Resource.find/equality](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
