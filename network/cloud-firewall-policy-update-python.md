# Cloud Firewall Policy Update Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/update_firewall_policy`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateFirewallPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyUpdateFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyUpdateRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyUpdateOpts`

- SDK-owned six cloud kwargs validate before strict target policy Find, including empty attrs; raw equal attrs skip PUT and dirty attrs use a private upstream builder preserving explicit null/false/empty JSON. Lookup evidence and native Policy mutation result are distinct, with nil mutation result on failure.
  Tests: `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateLookupOnlyAndDirtySelection`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateNameFallbackAndMissing`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateOwnedOptionsAndPreflight`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateConflictKeepsLookupAndSingleAttempt`.
- A present firewall_rules key is translated only after target lookup: ordered strict rule lookup repeats duplicate references without forwarding policy filters, [] clears with zero rule lookup, and equal resolved IDs skip PUT only after lookups. A later missing reference retains target/prior rule evidence and stops further work; unsupported JSON reference shapes fail after target lookup. Falsey filters/defaults and bound argument collisions preserve the bounded strict Find contract.
  Tests: `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateOrderedRulesTargetFirstAndEmptyClear`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateRuleFailureKeepsPriorEvidence`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateUnsupportedRuleShapesAfterTargetLookup`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateFiltersStrictBindingAndFalseyDefaults`.
- An accepted PUT (any status 200-399, as Python raise_from_response) returns Response, an owned UpdateReceipt with the actual body bytes, headers and status. Body follows Python Resource._translate_response: the resource-key member when present, otherwise the whole top-level object, without self and with unknown members preserved. Merged is the lookup wire row overlaid with every effective attribute and then Body, so a partial body keeps lookup fields; an empty, 204, truncated or otherwise unparsable body sets BodyIgnored and keeps that pre-commit state without error. Non-object JSON or a non-object resource-key member fails after remote success with the receipt and no Merged/native model. A rejected PUT has no receipt, and nothing is replayed. The native model is the projection of Merged.
  Tests: `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptMergesPythonTranslateResponse`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptRejectedPutHasNoReceipt`.

Python/Go differences:

- UpdateFirewallPolicyRecord owns kwargs/filters, target-first strict lookup and ordered string-array rule translation. Lookup and RuleLookups retain separate read evidence; Response is the owned PUT receipt and Policy the native projection of its merged view. Empty and equal-update branches return a carrier rather than the same Python mutable Resource.
- Receipt merge applies response members by literal wire key; a response key equal to a Python attribute alias is not remapped (see the remaining canonical/wire collision item). The native model omits Merged members its schema cannot hold and reports them in UpdateReceipt.ProjectionError without failing the accepted update; Merged keeps the raw values Python would keep. NaN/Infinity bodies are invalid JSON to Go and take the BodyIgnored branch where Python would merge them (shared nonfinite JSON boundary in docs/sdk-support-ledger.md).

Unresolved boundaries:

- Rule references use JSON arrays of nonempty strings; arbitrary Python iterables, object/nonstring IDs, Source runtime exception classes and full same-instance Resource lifecycle are outside this bounded profile.
- Full raw Find header/microversion/fetch controls, Requests encoding/nonfinite JSON and SDK-R1/C1/S1 Resource/adapter/session/cache contracts remain separate.
- Response members named by a Python attribute alias rather than the wire key (canonical/wire collision) are not remapped by the receipt merge.

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
    result = conn.update_firewall_policy(args.identity, filters=json.loads(args.filters), **attrs)
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
    identity := flag.String("identity", "", "firewall policy ID or exact name")
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
    result, err := service.UpdateFirewallPolicyRecord(ctx, *identity,
        network.WithFirewallPolicyUpdateJSON(json.RawMessage(*attrs)),
        network.WithFirewallPolicyUpdateFilters(json.RawMessage(*filters)))
    if err != nil { log.Fatal(err) }
    var value any = result.Policy
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
