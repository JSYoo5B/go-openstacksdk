# Cloud Firewall Group Update Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/update_firewall_group`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateFirewallGroupRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallGroupUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallGroupUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallGroupUpdateFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallGroupUpdatePorts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallGroupUpdateOpts`

- SDK-owned exact eight cloud kwargs validate before strict target group Find, rejecting direct egress/ingress_firewall_policy_id despite the Source docstring claim. Empty attrs still look up; raw equal values skip PUT and dirty attrs preserve null/false/empty via a private SDK builder with separate lookup/native mutation output.
  Tests: `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateLookupOnlyAndDirtySelection`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateNameFallbackAndMissing`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateOwnedOptionsAndPreflight`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateConflictKeepsLookupAndSingleAttempt`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateFiltersStrictBindingAndFalseyDefaults`.
- After target Find, nonnull policy aliases resolve strict egress then ingress regardless input key order, null aliases set wire ID null without policy lookup, then ports resolve in input order with duplicates and without target-filter propagation. [] clears with no port lookup; equal resolved references still resolve before no-PUT. Midphase failures and null ports retain prior target/policy/port evidence and prevent further lookup/PUT.
  Tests: `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateTargetPolicyPortsOrderAndEmptyClear`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateNullPoliciesAndEmptyPorts`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateReferenceFailuresPreservePhaseOrder`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateResolvedEqualReferencesStillLookUp`.

Python/Go differences:

- UpdateFirewallGroupRecord owns exact decorator kwargs, target-first filters and ordered policy/port resolution. Lookup/PolicyLookups/PortLookups are separate read evidence and Group is native mutation output. The Source docstring direct-ID suggestion conflicts with its decorator; this helper follows actual rejection.

Unresolved boundaries:

- Policy references are nonempty JSON strings or null; ports are JSON arrays of nonempty strings. Arbitrary Python iterable/object/nonstring/empty response-ID/runtime-exception behavior and full same-instance Resource lifecycle remain outside this bounded profile.
- Native HTTP200-only mutation acceptance and typed/unknown/partial/empty/malformed response handling expose no original PUT receipt. Raw Find header/microversion/fetch constructor controls, full Requests encoding/nonfinite JSON, arbitrary object/numeric equality and SDK-R1/C1/S1 Resource/adapter/session/cache contracts remain separate. Referenced ports retain the existing ports.FindRecord status/encoding/Resource profile; firewall-engine coverage does not close those port-profile gaps.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--attrs", default="{}")
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
attrs = json.loads(args.attrs)
if not isinstance(attrs, dict):
    parser.error("attrs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.update_firewall_group(args.identity, filters=json.loads(args.filters), **attrs)
    print(json.dumps(None if result is None else result.to_dict(), indent=2))
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
    identity := flag.String("identity", "", "firewall group ID or exact name")
    attrs := flag.String("attrs", "{}", "cloud update kwargs as JSON")
    filters := flag.String("filters", "{}", "JSON Source Find kwargs")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.UpdateFirewallGroupRecord(ctx, *identity,
        network.WithFirewallGroupUpdateJSON(json.RawMessage(*attrs)),
        network.WithFirewallGroupUpdateFilters(json.RawMessage(*filters)))
    if err != nil { log.Fatal(err) }
    var value any = result.Group
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
