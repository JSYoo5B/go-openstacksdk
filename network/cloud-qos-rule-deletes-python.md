# Cloud QoS Rule Deletes Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import os
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--kind", choices=("bandwidth", "dscp", "minimum"), required=True)
    parser.add_argument("--policy", required=True)
    parser.add_argument("--rule", required=True)
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    methods = {
        "bandwidth": conn.delete_qos_bandwidth_limit_rule,
        "dscp": conn.delete_qos_dscp_marking_rule,
        "minimum": conn.delete_qos_minimum_bandwidth_rule,
    }
    print(methods[args.kind](args.policy, args.rule))

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
	kind := flag.String("kind", "", "bandwidth, dscp or minimum")
	policy := flag.String("policy", "", "QoS policy ID or name")
	rule := flag.String("rule", "", "literal QoS rule ID")
	flag.Parse()
	if *policy == "" || *rule == "" || *kind == "" {
		log.Fatal("--kind, --policy and --rule required")
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
	var result *network.QoSRuleDeleteResult
	switch *kind {
	case "bandwidth":
		result, err = service.DeleteQoSBandwidthLimitRule(ctx, *policy, *rule)
	case "dscp":
		result, err = service.DeleteQoSDSCPMarkingRule(ctx, *policy, *rule)
	case "minimum":
		result, err = service.DeleteQoSMinimumBandwidthRule(ctx, *policy, *rule)
	default:
		log.Fatal("--kind must be bandwidth, dscp or minimum")
	}
	if err != nil {
		if result != nil && result.Receipt != nil {
			log.Printf("actual DELETE response: %d %s", result.Receipt.StatusCode, string(result.Receipt.Envelope))
		}
		log.Fatal(err)
	}
	output := map[string]any{"completed": result.Completed, "deleted": result.Deleted}
	if result.Receipt != nil {
		output["status"] = result.Receipt.StatusCode
		output["body"] = string(result.Receipt.Envelope)
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1723)
- [minimum-bandwidth Source](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2115)
- [Neutron27 QoS policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [base policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py)
