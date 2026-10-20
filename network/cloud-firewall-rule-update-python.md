# Cloud Firewall Rule Update Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/update_firewall_rule`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateFirewallRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallRuleUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallRuleUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallRuleUpdateFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallRuleUpdateOpts`

- SDK-owned fourteen-key cloud attrs are validated before strict member/name Find; empty attrs still look up, equal raw kwargs skip PUT, and only dirty fields reach an SDK-private upstream builder. Explicit null/false/empty and raw ANY protocol are preserved without requiring application builders; native mutation output remains separate from lookup receipt and is nil on failure.
  Tests: `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateLookupOnlyAndDirtySelection`, `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateNameFallbackAndMissing`, `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateConflictKeepsLookupAndSingleAttempt`.
- Owned JSON filters normalize falsey JSON to default strict Find kwargs and push ordinary filter fields through member/fallback lookup. Truthy nonobjects and bound name_or_id/ignore_missing collisions fail without HTTP; currently unsupported raw headers/microversion namespaces are rejected. Option snapshots, unsupported cloud keys, cancellation and selected-source mutation are checked before lookup.
  Tests: `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateFiltersStrictBindingAndFalseyDefaults`, `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateOwnedOptionsAndPreflight`.

Python/Go differences:

- UpdateFirewallRuleRecord owns cloud options and filters, strict lookup and raw dirty selection. A private SDK builder preserves fourteen cloud kwargs without exposing builder-interface responsibility to callers. Lookup is a separate record; Rule uses upstream typed result extraction and provider hooks.

Unresolved boundaries:

- Native mutation retains upstream HTTP200-only acceptance, strict typed projection, no unknown/partial/empty/malformed response merge or same mutable Resource identity, and no original mutation receipt.
- Raw Find headers/microversion member-query versus fallback-formal behavior and base_path/skip_cache/resource_response_key/fetch constructor controls are not fully implemented; currently unsupported reserved filters fail before HTTP. Full Requests encoding/nonfinite JSON, arbitrary Python object/numeric types and SDK-R1/C1/S1 Resource/adapter/session/cache lifecycle remain separate.

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
    result = conn.update_firewall_rule(args.identity, filters=json.loads(args.filters), **attrs)
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
    identity := flag.String("identity", "", "firewall rule ID or exact name")
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
    result, err := service.UpdateFirewallRuleRecord(ctx, *identity,
        network.WithFirewallRuleUpdateJSON(json.RawMessage(*attrs)),
        network.WithFirewallRuleUpdateFilters(json.RawMessage(*filters)))
    if err != nil { log.Fatal(err) }
    var value any = result.Rule
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
