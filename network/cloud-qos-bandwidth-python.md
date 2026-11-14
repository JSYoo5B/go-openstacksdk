# Cloud QoS Bandwidth Python

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
    parser.add_argument("--policy", required=True)
    parser.add_argument("--rule", default="")
    parser.add_argument("--mode", choices=("list", "search", "get"), default="list")
    parser.add_argument("--filters", default="null")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    filters = json.loads(args.filters)
    if args.mode == "list":
        values = conn.list_qos_bandwidth_limit_rules(args.policy, filters)
    elif args.mode == "search":
        values = conn.search_qos_bandwidth_limit_rules(args.policy, args.rule or None, filters)
    else:
        if not args.rule:
            parser.error("--rule required for get")
        values = [conn.get_qos_bandwidth_limit_rule(args.policy, args.rule)]
    for value in values:
        print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value, ensure_ascii=False, indent=2))

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
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	policy := flag.String("policy", "", "QoS policy ID or name")
	rule := flag.String("rule", "", "rule ID or search pattern")
	mode := flag.String("mode", "list", "list, search or get")
	filters := flag.String("filters", "null", "Source cloud filters JSON")
	flag.Parse()
	if *policy == "" {
		log.Fatal("--policy required")
	}
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var result *network.QoSBandwidthLimitResult
	switch *mode {
	case "list":
		result, err = service.ListQoSBandwidthLimitRules(ctx, *policy, network.WithQoSBandwidthLimitFilters(json.RawMessage(*filters)))
	case "search":
		result, err = service.SearchQoSBandwidthLimitRules(ctx, *policy, *rule, network.WithQoSBandwidthLimitFilters(json.RawMessage(*filters)))
	case "get":
		if *rule == "" {
			log.Fatal("--rule required for get")
		}
		result, err = service.GetQoSBandwidthLimitRule(ctx, *policy, *rule)
	default:
		log.Fatal("--mode must be list, search or get")
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.Value == nil {
		log.Fatal("opaque result: consume QoSBandwidthLimitGenerator.Records explicitly")
	}
	output, err := json.MarshalIndent(result.Value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(output))
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe` cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1535)
- [bandwidth-limit record API](v2/qos-bandwidth-limit-records-python.md)
