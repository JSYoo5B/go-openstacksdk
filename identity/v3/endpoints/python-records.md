# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="system-reader")
    parser.add_argument("--service-id", default="")
    parser.add_argument("--region-id", default="")
    args = parser.parse_args()
    query = {"is_enabled": True, "max_items": 100}
    if args.service_id:
        query["service_id"] = args.service_id
    if args.region_id:
        query["region_id"] = args.region_id
    conn = openstack.connect(cloud=args.cloud)
    for row in conn.identity.endpoints(**query):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

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
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
	cloud := flag.String("cloud", "system-reader", "clouds.yaml cloud")
	service := flag.String("service-id", "", "optional service UUID")
	region := flag.String("region-id", "", "optional region ID")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	identity, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []endpoints.RecordListOption{
		endpoints.WithRecordListQuery("is_enabled", true),
		request.MaxItems(100),
	}
	if *service != "" {
		options = append(options, endpoints.WithRecordListQuery("service_id", *service))
	}
	if *region != "" {
		options = append(options, endpoints.WithRecordListQuery("region_id", *region))
	}
	for row, err := range identity.Endpoints.ListRecords(ctx, options...) {
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(row.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	}
}
```

## References

- [`Proxy.endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py)
- [`Endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/endpoint.py)
- [Endpoint](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/endpoint.py)
- [Base](https://github.com/openstack/keystone/blob/27.0.0/keystone/common/policies/base.py)
