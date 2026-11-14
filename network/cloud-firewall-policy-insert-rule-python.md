# Cloud Firewall Policy Insert Rule Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--rule", required=True)
parser.add_argument("--after")
parser.add_argument("--before")
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.insert_rule_into_policy(
        args.identity, args.rule, insert_after=args.after,
        insert_before=args.before, filters=json.loads(args.filters))
    print(json.dumps(result.to_dict(), indent=2))
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
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("identity", "", "firewall policy ID or exact name")
    rule := flag.String("rule", "", "rule ID or exact name")
    after := flag.String("after", "", "optional preceding rule")
    before := flag.String("before", "", "optional following rule")
    filters := flag.String("filters", "{}", "JSON Source Find kwargs")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    options := []network.FirewallPolicyInsertOption{
        network.WithFirewallPolicyInsertFilters(json.RawMessage(*filters)),
    }
    if *after != "" { options = append(options, network.WithFirewallPolicyInsertAfter(*after)) }
    if *before != "" { options = append(options, network.WithFirewallPolicyInsertBefore(*before)) }
    result, err := service.InsertFirewallRuleIntoPolicyRecord(ctx, *identity, *rule, options...)
    if err != nil { log.Fatal(err) }
    var value any = result.Action
    if result.NoOp {
        value = nil
        if result.Lookup != nil { value = result.Lookup.Resource }
    }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Install](../docs/install.md)
