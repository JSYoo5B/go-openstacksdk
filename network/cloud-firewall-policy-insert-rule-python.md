# Cloud Firewall Policy Insert Rule Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/insert_rule_into_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.InsertFirewallRuleIntoPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyInsertOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyInsertFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyInsertAfter`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyInsertBefore`

- SDK-owned cloud insertion resolves strict policy with filters then strict rule without filters, uses the policy list descriptor view for membership and returns lookup-only no-op when already present before parsing positions. Missing/null membership fails only after both lookups. Position parsing then resolves after before before and keeps prior receipts on later failure.
  Tests: `network/cloud_firewall_policy_insert_rule_test.go:TestCloudFirewallPolicyInsertAlreadyPresentSkipsInvalidPositions`, `network/cloud_firewall_policy_insert_rule_test.go:TestCloudFirewallPolicyInsertMissingMembershipFailsAfterRuleLookup`, `network/cloud_firewall_policy_insert_rule_test.go:TestCloudFirewallPolicyInsertLaterPositionFailureKeepsReceipts`.
- Both or neither positions are accepted, omitted positions send explicit JSON null, and the existing InsertRuleRecord action preserves unwrapped object fields with separate header/status and native policy output. Options snapshot JSON and strict bound-filter/pre-canceled context errors prevent HTTP.
  Tests: `network/cloud_firewall_policy_insert_rule_test.go:TestCloudFirewallPolicyInsertOrderBothPositionsAndDefaultNulls`, `network/cloud_firewall_policy_insert_rule_test.go:TestCloudFirewallPolicyInsertOwnedOptionsAndPreflight`.

Python/Go differences:

- The SDK owns strict lookup, membership and delayed concrete position options; Lookup/Rule/Positions are independent read receipts and Action is the existing leaf record, not the same Python Resource instance.
- Explicit Go profile boundary: Existing action accepts HTTP200..399, not all Source100..399; strict typed response, full Requests encoding/nonfinite JSON and original action Envelope remain outside this profile.
- Explicit Go profile boundary: References and response IDs use nonempty strings with existing action path validation; arbitrary Python objects/nonstring/empty IDs/dynamic membership/runtime exception types, full raw Find control namespace and SDK-R1/C1/S1 Resource/adapter/session/cache lifecycle remain separate.

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
