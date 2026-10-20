# Cloud Firewall Policy Remove Rule Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/remove_rule_from_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.RemoveRuleFromPolicyRecord`

- Cloud policy removal performs strict policy Find with owned filters before default ignore-missing rule Find without policy filters; missing rule returns the looked-up policy without inspecting null/missing membership or sending an action.
  Tests: `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRuleMissingTargetAndRule`, `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRuleOwnedFiltersAndFallback`.
- Source List descriptor projection and Python loaded JSON membership decide absent-rule no-op versus SDK-owned native PUT remove_rule with resolved IDs; lookup receipts remain distinct from successful native action Policy.
  Tests: `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRuleMembershipAndAction`, `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRuleFailureKeepsLookupProof`.
- Owned options isolate filters from callers; falsey filters default to empty kwargs, duplicate Find bindings/nil options/service replacement/cancellation fail without later HTTP.
  Tests: `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRulePreflightAndFalseyFilters`, `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRuleOwnedFiltersAndFallback`.

Python/Go differences:

- The cloud helper now owns policy-first and optional-rule lookups and association no-op; its action deliberately uses the existing native Gophercloud RemoveRule path, preserving a separate typed Policy rather than claiming Source Resource merge parity.
- Action HTTP status policy remains native 200-only versus Source _put_request resp.ok (<400); 202 is explicitly regression-tested as a documented Go profile difference.
- Action return remains a flat native typed Policy rather than Source applying the whole response JSON mapping, including unknown/nested fields, to a new ID-seeded Resource (the cloud action passes the lookup policy ID to Proxy; only the no-op returns the lookup Resource); action raw receipt/error metadata is not exposed.
- Action Requests response charset guessing/codecs and permissive JSON parsing remain outside the native decoder domain.
- Find/action IDs are nonempty strings; arbitrary nonstring/empty Source JSON IDs and Python path conversion/equality are not reproduced. Source location projection beyond the reused Find profile is not added by this helper.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--policy", required=True)
parser.add_argument("--rule", required=True)
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.remove_rule_from_policy(args.policy, args.rule, filters=json.loads(args.filters))
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
    policy := flag.String("policy", "", "policy exact name or ID")
    rule := flag.String("rule", "", "rule exact name or ID")
    filters := flag.String("filters", "{}", "policy Find JSON filters")
    flag.Parse()
    if *policy == "" || *rule == "" { log.Fatal("policy and rule are required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.RemoveRuleFromPolicyRecord(ctx, *policy, *rule,
        network.WithFirewallPolicyRemoveRuleFilters(json.RawMessage(*filters)))
    if err != nil { log.Fatal(err) }
    var value any = result.Policy
    if result.NoOp && result.Lookup != nil { value = result.Lookup.Resource }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Install](../docs/install.md)
