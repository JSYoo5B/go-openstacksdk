# Python User Helpers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--mode", choices=("values", "subnet", "bulk"), default="values")
    parser.add_argument("--filters", default="{}")
    parser.add_argument("--subnet", default="")
    parser.add_argument("--data", default="[]")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    if args.mode == "values":
        values = conn.list_ports(filters=json.loads(args.filters))
        for value in values:
            print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value, ensure_ascii=False, indent=2))
    elif args.mode == "subnet":
        if not args.subnet:
            parser.error("--subnet required")
        for value in conn.network.get_subnet_ports(args.subnet):
            print(json.dumps(value.to_dict(), ensure_ascii=False, indent=2))
    else:
        for value in conn.network.create_ports(json.loads(args.data)):
            print(json.dumps(value.to_dict(), ensure_ascii=False, indent=2))

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
	"os"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
)

func printJSON(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}

func main() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	mode := flag.String("mode", "values", "values, subnet or bulk")
	filters := flag.String("filters", "{}", "Source cloud filters JSON")
	subnetID := flag.String("subnet", "", "literal subnet ID")
	data := flag.String("data", "[]", "nonempty JSON array of port attribute objects")
	flag.Parse()
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	if *mode == "values" {
		result, err := conn.ListPortValues(ctx, ports.WithSourceCloudListFilters(json.RawMessage(*filters)))
		if err != nil {
			log.Fatal(err)
		}
		if result.JSON == nil {
			log.Fatal("opaque result; consume RecordListGenerator.Records explicitly")
		}
		printJSON(result.JSON)
		return
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	switch *mode {
	case "subnet":
		if *subnetID == "" {
			log.Fatal("--subnet required")
		}
		result, err := network.Ports.GetSubnetPorts(ctx, ports.SubnetPortID(*subnetID))
		if err != nil {
			log.Fatal(err)
		}
		for _, record := range result.Ports {
			printJSON(record.Resource.Body)
		}
	case "bulk":
		var rows []json.RawMessage
		if err := json.Unmarshal([]byte(*data), &rows); err != nil {
			log.Fatal(err)
		}
		inputs := make([]ports.BulkPortInput, len(rows))
		for i, row := range rows {
			inputs[i] = ports.BulkPortJSON(row)
		}
		result, err := network.Ports.CreatePorts(ctx, inputs)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("POST completed: %d\n", result.StatusCode)
		for record, err := range result.Records {
			if err != nil {
				log.Fatal(err)
			}
			printJSON(record.Resource.Body)
		}
	default:
		log.Fatal("--mode must be values, subnet or bulk")
	}
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/tree/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe)
- [Python List Records](python-list-records.md)
- [Source cloud `list_ports`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L202)
- [Source loop](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L4047)
- [Source create_ports](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3894)
- [V2](https://docs.openstack.org/api-ref/network/v2/#bulk-create-ports)
- [Neutron 27 Pecan BodyValidation](https://github.com/openstack/neutron/blob/27.0.0/neutron/pecan_wsgi/hooks/body_validation.py)
- [Policy hooks](https://github.com/openstack/neutron/blob/27.0.0/neutron/pecan_wsgi/hooks/policy_enforcement.py)
- [Plugin](https://github.com/openstack/neutron/blob/27.0.0/neutron/plugins/ml2/plugin.py)
- [Port](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py)
