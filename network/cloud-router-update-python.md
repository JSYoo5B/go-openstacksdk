# Cloud Router Update Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/update_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateRouterRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterUpdateOpts`

- SDK-owned six cloud formals distinguish truthy name/gateway network/fixed IPs from nonnull admin state/SNAT/routes, reuse library gateway assembly without creation admin defaults or merging existing gateway state, skip all HTTP when no effective args exist, and commit only raw dirty fields after member/name lookup. No-op and native mutation result remain distinct from lookup evidence.
  Tests: `network/cloud_router_update_test.go:TestCloudRouterUpdateEffectiveValuesAndDirtySelection`, `network/cloud_router_update_test.go:TestCloudRouterUpdateIneffectiveBeforeServiceLookup`, `network/cloud_router_update_test.go:TestCloudRouterUpdateNameFallbackAndMissing`.
- Nonnull routes including [] and false perform fresh complete extraroute discovery before lookup; missing extension drops routes only and can return lookup-free no-op, while other attrs continue. HTTP/late malformed alias/continuation failures never become unsupported-route fallback. Owned options/preflight and canceled-context guards prevent unwanted HTTP; native conflict preserves lookup and no success model.
  Tests: `network/cloud_router_update_test.go:TestCloudRouterUpdateRoutesGateAndDrop`, `network/cloud_router_update_test.go:TestCloudRouterUpdateExtensionFailureCannotBecomeDroppedRoutes`, `network/cloud_router_update_test.go:TestCloudRouterUpdateOwnedOptionsAndPreflight`, `network/cloud_router_update_test.go:TestCloudRouterUpdateOptionCancellationCannotBecomeNoOp`, `network/cloud_router_update_test.go:TestCloudRouterUpdateConflictKeepsLookupAndSingleAttempt`.
- An accepted PUT (any status 200-399, as Python raise_from_response) returns Response, an owned UpdateReceipt with the actual body bytes, headers and status. Body follows Python Resource._translate_response: the resource-key member when present, otherwise the whole top-level object, without self and with unknown members preserved. Merged is the lookup wire row overlaid with every effective attribute and then Body, so a partial body keeps lookup fields; an empty, 204, truncated or otherwise unparsable body sets BodyIgnored and keeps that pre-commit state without error. Non-object JSON or a non-object resource-key member fails after remote success with the receipt and no Merged/native model. A rejected PUT has no receipt, and nothing is replayed. The native model is the projection of Merged.
  Tests: `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptMergesPythonTranslateResponse`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptRejectedPutHasNoReceipt`.
- Lookup.SourceBody is the pinned Python Router body-attribute state: an attribute-named member such as is_admin_state_up is stored under admin_state_up, a wire/canonical collision keeps CPython dict order (first position, last value) and Router's own revision_number declaration maps a Neutron revision_number member to the revision wire key. Supplied values compare against that state, so a canonical-only equal value is a no-op without PUT, and the accepted response is merged through the same mapping while Body keeps the actual member names.
  Tests: `network/cloud_update_source_state_test.go:TestCloudUpdateSourceStateCanonicalLookupDirtyComparison`, `network/cloud_update_source_state_test.go:TestCloudUpdateSourceStateRouterRevisionAndSubnetAlias`, `internal/neutronfind/source_state_test.go:TestSourceStateConsumeAttrsMapping`.

Python/Go differences:

- UpdateRouterRecord owns formal defaults and optional extraroute handling. RoutesDropped and Extensions expose omission/discovery without adding Source logger behavior. Lookup is an owned record, Response is the owned PUT receipt and Router is the native projection of its merged view; none is the same mutable Python Resource. Client capture follows effective-option selection and retains configured native provider hooks.
- Lookup dirty comparison and receipt merge use the pinned Python body-attribute state (SourceBody/TranslateSourceBody): Resource._consume_attrs over CPython dict order maps attribute-named members to their wire keys, with translate rules for member GET and PUT responses and constructor rules for fallback LIST rows. The native model omits Merged members its schema cannot hold and reports them in UpdateReceipt.ProjectionError without failing the accepted update; Merged keeps the raw values Python would keep. NaN/Infinity bodies are invalid JSON to Go and take the BodyIgnored branch where Python would merge them (shared nonfinite JSON boundary in docs/sdk-support-ledger.md).
- Inputs are typed Go values; arbitrary Python objects, iterables and numeric types accepted by dynamic kwargs have no equivalent; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Text is decoded only from UTF-8, a byte-order mark or a supported declared charset (UTF-16/UTF-32 with a BOM or explicit endianness, ISO-8859-1); undeclared non-UTF-8 bytes and other codecs return ErrUnsupported with the original bytes instead of Requests' statistical detection or the full CPython codec registry; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Only standard finite JSON numbers are accepted: NaN, Infinity and overflowing literals such as 1e400 return ErrUnsupported with the original response, where Python's json loads them as floats, and a latest microversion must be written as "latest"; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Results are immutable owned snapshots rather than Python's mutable Resource with adapter, session, cache and dirty state (SDK-R1/C1/S1); see Deliberate runtime differences in docs/sdk-support-ledger.md.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

`result.Lookup.SourceBody()` returns the wire-keyed Python body attributes the update compared against, so a lookup row that names an attribute only by its Python attribute name (for example `is_admin_state_up`) is visible under its wire key (`admin_state_up`).

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--attrs", default="{}")
args = parser.parse_args()
attrs = json.loads(args.attrs)
if not isinstance(attrs, dict):
    parser.error("attrs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.update_router(args.identity, **attrs)
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
    identity := flag.String("identity", "", "router ID or exact name")
    attrs := flag.String("attrs", "{}", "cloud update kwargs as JSON")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.UpdateRouterRecord(ctx, *identity,
        network.WithRouterUpdateJSON(json.RawMessage(*attrs)))
    if err != nil { log.Fatal(err) }
    var value any = result.Router
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
