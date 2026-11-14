# QoS Bandwidth Limit Records Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--policy", required=True)
parser.add_argument("--rule", default="")
parser.add_argument("--mode", choices=("find", "get", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        rule = conn.network.find_qos_bandwidth_limit_rule(
            args.rule, args.policy, ignore_missing=False
        )
        print(json.dumps(rule.to_dict(), indent=2))
    elif args.mode == "get":
        rule = conn.network.get_qos_bandwidth_limit_rule(args.rule, args.policy)
        print(json.dumps(rule.to_dict(), indent=2))
    else:
        for rule in conn.network.qos_bandwidth_limit_rules(
            args.policy, direction="egress", max_items=100
        ):
            print(json.dumps(rule.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, mode, parent, identity string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "find":
        value, err := network.QoSRules.FindBandwidthLimitRecord(
            ctx, parent, identity, request.IgnoreMissing(false),
        )
        if err != nil || value == nil { return nil, err }
        return value.Resource, nil
    case "get":
        value, err := network.QoSRules.GetBandwidthLimitRecord(ctx, parent, identity)
        if err != nil { return nil, err }
        return value.Resource, nil
    case "list":
        values, err := network.QoSRules.AllBandwidthLimitRecords(
            ctx, parent, request.Query("direction", "egress"),
            request.MaxItems(100),
        )
        if err != nil { return nil, err }
        rows := make([]*resource.RawResource, 0, len(values))
        for _, value := range values { rows = append(rows, value.Resource) }
        return rows, nil
    default:
        return nil, fmt.Errorf("unknown mode %q", mode)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    parent := flag.String("policy", "", "literal QoS policy ID")
    identity := flag.String("rule", "", "literal rule ID or Find identity")
    mode := flag.String("mode", "list", "find, get or list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *parent, *identity)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L205)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L65)
