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
- Owned JSON filters normalize falsey JSON to default strict Find kwargs and push ordinary filter fields through member/fallback lookup. Truthy nonobjects and bound name_or_id/ignore_missing collisions fail without HTTP; raw headers/microversion and fetch controls follow the Source Find phases instead of being rejected. Option snapshots, unsupported cloud keys, cancellation and selected-source mutation are checked before lookup.
  Tests: `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateFiltersStrictBindingAndFalseyDefaults`, `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateOwnedOptionsAndPreflight`.
- An accepted PUT (any status 200-399, as Python raise_from_response) returns Response, an owned UpdateReceipt with the actual body bytes, headers and status. Body follows Python Resource._translate_response: the resource-key member when present, otherwise the whole top-level object, without self and with unknown members preserved. Merged is the lookup wire row overlaid with every effective attribute and then Body, so a partial body keeps lookup fields; an empty, 204, truncated or otherwise unparsable body sets BodyIgnored and keeps that pre-commit state without error. Non-object JSON or a non-object resource-key member fails after remote success with the receipt and no Merged/native model. A rejected PUT has no receipt, and nothing is replayed. The native model is the projection of Merged.
  Tests: `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptMergesPythonTranslateResponse`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptRejectedPutHasNoReceipt`.
- Cloud filters reach Resource.find through the shared Source fetch-control profile: base_path routes the initial GET (and fallback LIST), resource_response_key selects its response member, skip_cache has no Go cache to bypass, and these kwargs seed the found Resource as unknown constructor attributes while the PUT keeps the class route and dirty body; list_base_path changes only the fallback LIST route.
  Tests: `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallUpdateFindFetchControls`.
- Firewall lookup state is the actual member set without self (no attribute aliases are declared), so only differing values are sent and the accepted response, including case-variant unknown members, is merged by literal key.
  Tests: `network/cloud_update_source_state_test.go:TestCloudUpdateSourceStateFirewallIdentityMapping`.

Python/Go differences:

- UpdateFirewallRuleRecord owns cloud options and filters, strict lookup and raw dirty selection. The dirty fourteen-key subset is sent as one owned PUT without exposing builder-interface responsibility to callers. Lookup is a separate record; Response is the owned PUT receipt and Rule the native projection of its merged view.
- The firewall Body declarations have no attribute aliases (every attribute name equals its wire key), so lookup dirty comparison and receipt merge use the actual members without self through the same Resource._consume_attrs state; a case-variant member stays an unknown member. The native model omits Merged members its schema cannot hold and reports them in UpdateReceipt.ProjectionError without failing the accepted update; Merged keeps the raw values Python would keep. NaN/Infinity bodies are invalid JSON to Go and take the BodyIgnored branch where Python would merge them (shared nonfinite JSON boundary in docs/sdk-support-ledger.md).

Unresolved boundaries:

- Full Requests encoding/nonfinite JSON, arbitrary Python object/numeric types and SDK-R1/C1/S1 Resource/adapter/session/cache lifecycle remain separate runtime boundaries.

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
    if result.Response != nil && result.Response.Merged != nil {
        value = result.Response.Merged // the Python post-commit resource state
    }
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
