# Cloud Endpoints Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def jsonable(value):
    if hasattr(value, "to_dict"):
        return value.to_dict()
    if isinstance(value, list):
        return [jsonable(item) for item in value]
    if isinstance(value, dict):
        return {key: jsonable(item) for key, item in value.items()}
    return value

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="system-reader")
    parser.add_argument("--id", default="")
    parser.add_argument("--filters", default="")
    parser.add_argument("--get", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    filters = json.loads(args.filters) if args.filters else None
    if args.get:
        value = conn.get_endpoint(args.id, filters=filters)
    else:
        value = conn.search_endpoints(id=args.id, filters=filters)
    print(json.dumps(jsonable(value), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
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
	"github.com/JSYoo5B/go-openstacksdk/identity"
)

func main() {
	cloud := flag.String("cloud", "system-reader", "clouds.yaml cloud")
	id := flag.String("id", "", "literal ID or search glob")
	filters := flag.String("filters", "", "JSON dict or JSON encoded JMESPath string")
	get := flag.Bool("get", false, "get one endpoint")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	api, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	reader := identity.NewEndpointReadCloud(api.Endpoints)
	var options []identity.EndpointReadOption
	if *filters != "" {
		options = append(options, identity.WithEndpointReadFilters(json.RawMessage(*filters)))
	}
	var value json.RawMessage
	if *get {
		result, err := reader.GetEndpoint(ctx, *id, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	} else {
		result, err := reader.SearchEndpoints(ctx, *id, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [`IdentityCloudMixin.search_endpoints/get_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py)
- [`_utils._filter_list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py)
- [Python Records](v3/endpoints/python-records.md)
- [Endpoint](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/endpoint.py)
- [Base](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/base.py)
