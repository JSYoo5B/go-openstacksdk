# Group Replication Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/disable_group_replication`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.DisableReplicationRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordReplicationOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordReplicationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordReplicationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordReplicationPinnedSourceWire`

- Literal escaped group action POST, no GET/Find/delete fallback, complete Cinder JSON action request and actual ACK receipt only; no fabricated Group Resource/backend completion, unparsed accepted body including arbitrary JSON/invalidUTF8 and exact HTTP100..399 admission.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationStockDefaultsLiteralRoutesAndAck`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationAllSourceAdmittedCodesIgnoreResponseJSON`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationRejectionsNeverIgnoreOrLookup`.
- Pinned Group.disable_replication enable/disable action value is null, while stock Cinder27 schema requires object. Go default uses {}, and explicit PinnedSourceWire replays null, preserving the stock-shaped actual400 rather than claiming stock acceptance.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationPinnedSourceWireKeepsSchemaRejection`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationStockDefaultsLiteralRoutesAndAck`.
- Library-owned >=3.38 static selected-version guard stops older/different-major/global-latest before mutation; absent selection uses common NegotiateRequired3.38 with class ceiling3.38 and actual advertised-bound/discovery evidence. Discovery denial/unsupported carries no POST status/body/header; original selected client remains unchanged.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationMinimumRejectsBeforeMutationAndOverrideWorks`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationDefaultDiscoveryCeilingAndUnsupportedProof`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationDiscoveryDenialIsSeparateFromAction`.
- Options/maps/pointers are owned before callbacks and private typed version overrides reconstruct selected source aliases. Fresh nil/default headers permit the first custom callback. Literal identity/context/API/nil option/reserved protocol/invalid backendUTF8 preflight preserves custom option/source/cancel causes with zeroHTTP.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationFirstCustomOptionsOwnNilHeaders`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationPreflightOptionsAndOriginalCauses`.
- Actual accepted/rejected read/Close/context/custom-cancel/source errors remain original causes and evidence even after source restoration; zero native retry hooks once faults are observed. Source/native no-ignore action400403404 and native503 retry cannot alter owned body/version/Content-Type/reserved auth/Nova/OmitHeaders, while ordinary headers and live tokens retain native retry.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationPhysicalCausesSourceRestoreAndCancelKeepActualReceipt`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationNativeRetryOwnsBodyHeadersAndPreservesLiveToken`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationRejectionsNeverIgnoreOrLookup`.
- Independent Reference/action Envelope/header/status and discovery byte/header copies retain physical observations without using response-cache state.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationActualEvidenceOwnsDiscoveryAndAction`.
- Physical Read/Close/both401 retains original actual401 receipt and physical causes with oneHTTP/zero willingRetry/Reauth callbacks; clean401 renews token once and sends two identical owned POST bodies.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationFaulty401StopsCallbacksAndClean401RenewsToken`.

Python/Go differences:

- Pinned Group.disable_replication enable/disable action value is null, while stock Cinder27 schema requires object. Go default uses {}, and explicit PinnedSourceWire replays null, preserving the stock-shaped actual400 rather than claiming stock acceptance.
- The exact Source proxy accepts a Group object or ID and no Header/Microversion/PinnedSourceWire extensions. Go takes an owned literal nonblank UTF-8 ID, uses typed known failover parameters, snapshots options, and rejects route/session/protocol/header/JSON-domain invalid input instead of supporting mutable subclasses/Resource/Munch arbitrary dynamic values.
- Go enforces action minimum3.38 before mutation; Source Group._action normally only selects class/session max3.38 and lets an explicitly selected older version reach the server. Empty Go override requests default required negotiation. Global latest/different-major selected versions do not establish the finite-major minimum in this profile.
- Selected finite >=3.38 is retained without extra discovery, and therefore does not prove advertised server/driver support. Default discovery uses shared finite readers and actual bounds; Group class ceiling is3.38, distinct from GroupSnapshot3.29. No mutable shared client default or fabricated successful Resource is changed/returned.
- Go extends Python None return with a Reference and actual ACK/error/discovery receipt. HTTP202 does not prove completion/current durable replication state; preparation discovery remains status0 in action carrier. Body decode/UTF8 is not added to accepted Source action success.
- Shared source guard/REST policy preserves actual physical/native/callback/cancellation causes and owned payload/version/reserved headers, while ordinary native retry headers/live token refresh remain available. Persistent mutable/cache/dirty/session customization and actual backend acceptance remain shared SDK boundaries.
- Native *ErrUnableToReauthenticate keeps non-Unwrap ErrOriginal/ErrReauth fields; actual401 receipt copies only ErrOriginal, preserving pointer wrapper and physical ErrReauth causes. Test validates fields directly rather than demanding a nonexistent native Unwrap.

### `python:block_storage/v3/enable_group_replication`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.EnableReplicationRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordReplicationOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordReplicationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordReplicationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordReplicationPinnedSourceWire`

- Literal escaped group action POST, no GET/Find/delete fallback, complete Cinder JSON action request and actual ACK receipt only; no fabricated Group Resource/backend completion, unparsed accepted body including arbitrary JSON/invalidUTF8 and exact HTTP100..399 admission.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationStockDefaultsLiteralRoutesAndAck`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationAllSourceAdmittedCodesIgnoreResponseJSON`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationRejectionsNeverIgnoreOrLookup`.
- Pinned Group.enable_replication enable/disable action value is null, while stock Cinder27 schema requires object. Go default uses {}, and explicit PinnedSourceWire replays null, preserving the stock-shaped actual400 rather than claiming stock acceptance.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationPinnedSourceWireKeepsSchemaRejection`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationStockDefaultsLiteralRoutesAndAck`.
- Library-owned >=3.38 static selected-version guard stops older/different-major/global-latest before mutation; absent selection uses common NegotiateRequired3.38 with class ceiling3.38 and actual advertised-bound/discovery evidence. Discovery denial/unsupported carries no POST status/body/header; original selected client remains unchanged.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationMinimumRejectsBeforeMutationAndOverrideWorks`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationDefaultDiscoveryCeilingAndUnsupportedProof`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationDiscoveryDenialIsSeparateFromAction`.
- Options/maps/pointers are owned before callbacks and private typed version overrides reconstruct selected source aliases. Fresh nil/default headers permit the first custom callback. Literal identity/context/API/nil option/reserved protocol/invalid backendUTF8 preflight preserves custom option/source/cancel causes with zeroHTTP.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationFirstCustomOptionsOwnNilHeaders`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationPreflightOptionsAndOriginalCauses`.
- Actual accepted/rejected read/Close/context/custom-cancel/source errors remain original causes and evidence even after source restoration; zero native retry hooks once faults are observed. Source/native no-ignore action400403404 and native503 retry cannot alter owned body/version/Content-Type/reserved auth/Nova/OmitHeaders, while ordinary headers and live tokens retain native retry.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationPhysicalCausesSourceRestoreAndCancelKeepActualReceipt`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationNativeRetryOwnsBodyHeadersAndPreservesLiveToken`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationRejectionsNeverIgnoreOrLookup`.
- Independent Reference/action Envelope/header/status and discovery byte/header copies retain physical observations without using response-cache state.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationActualEvidenceOwnsDiscoveryAndAction`.
- Physical Read/Close/both401 retains original actual401 receipt and physical causes with oneHTTP/zero willingRetry/Reauth callbacks; clean401 renews token once and sends two identical owned POST bodies.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationFaulty401StopsCallbacksAndClean401RenewsToken`.

Python/Go differences:

- Pinned Group.enable_replication enable/disable action value is null, while stock Cinder27 schema requires object. Go default uses {}, and explicit PinnedSourceWire replays null, preserving the stock-shaped actual400 rather than claiming stock acceptance.
- The exact Source proxy accepts a Group object or ID and no Header/Microversion/PinnedSourceWire extensions. Go takes an owned literal nonblank UTF-8 ID, uses typed known failover parameters, snapshots options, and rejects route/session/protocol/header/JSON-domain invalid input instead of supporting mutable subclasses/Resource/Munch arbitrary dynamic values.
- Go enforces action minimum3.38 before mutation; Source Group._action normally only selects class/session max3.38 and lets an explicitly selected older version reach the server. Empty Go override requests default required negotiation. Global latest/different-major selected versions do not establish the finite-major minimum in this profile.
- Selected finite >=3.38 is retained without extra discovery, and therefore does not prove advertised server/driver support. Default discovery uses shared finite readers and actual bounds; Group class ceiling is3.38, distinct from GroupSnapshot3.29. No mutable shared client default or fabricated successful Resource is changed/returned.
- Go extends Python None return with a Reference and actual ACK/error/discovery receipt. HTTP202 does not prove completion/current durable replication state; preparation discovery remains status0 in action carrier. Body decode/UTF8 is not added to accepted Source action success.
- Shared source guard/REST policy preserves actual physical/native/callback/cancellation causes and owned payload/version/reserved headers, while ordinary native retry headers/live token refresh remain available. Persistent mutable/cache/dirty/session customization and actual backend acceptance remain shared SDK boundaries.
- Native *ErrUnableToReauthenticate keeps non-Unwrap ErrOriginal/ErrReauth fields; actual401 receipt copies only ErrOriginal, preserving pointer wrapper and physical ErrReauth causes. Test validates fields directly rather than demanding a nonexistent native Unwrap.

### `python:block_storage/v3/failover_group_replication`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.FailoverReplicationRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFailoverOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFailoverAllowedAttachedVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFailoverSecondaryBackendID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFailoverMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFailoverHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFailoverPinnedSourceWire`

- Literal escaped group action POST, no GET/Find/delete fallback, complete Cinder JSON action request and actual ACK receipt only; no fabricated Group Resource/backend completion, unparsed accepted body including arbitrary JSON/invalidUTF8 and exact HTTP100..399 admission.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationStockDefaultsLiteralRoutesAndAck`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationAllSourceAdmittedCodesIgnoreResponseJSON`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationRejectionsNeverIgnoreOrLookup`.
- Pinned Group.failover_replication literally sends modify_body_for_action, while stock Cinder27 requires failover_replication. Go default corrects the server action and explicit PinnedSourceWire replays the fixed SDK key, preserving actual400 rather than claiming stock acceptance.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationPinnedSourceWireKeepsSchemaRejection`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationStockDefaultsLiteralRoutesAndAck`.
- Library-owned >=3.38 static selected-version guard stops older/different-major/global-latest before mutation; absent selection uses common NegotiateRequired3.38 with class ceiling3.38 and actual advertised-bound/discovery evidence. Discovery denial/unsupported carries no POST status/body/header; original selected client remains unchanged.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationMinimumRejectsBeforeMutationAndOverrideWorks`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationDefaultDiscoveryCeilingAndUnsupportedProof`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationDiscoveryDenialIsSeparateFromAction`.
- Options/maps/pointers are owned before callbacks and private typed version overrides reconstruct selected source aliases. Fresh nil/default headers permit the first custom callback. Literal identity/context/API/nil option/reserved protocol/invalid backendUTF8 preflight preserves custom option/source/cancel causes with zeroHTTP.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationFirstCustomOptionsOwnNilHeaders`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationPreflightOptionsAndOriginalCauses`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationFailoverFalseNullEmptyAndOptsReset`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationOwnedOptionsRetainedMutationAndSelectedAliases`.
- Actual accepted/rejected read/Close/context/custom-cancel/source errors remain original causes and evidence even after source restoration; zero native retry hooks once faults are observed. Source/native no-ignore action400403404 and native503 retry cannot alter owned body/version/Content-Type/reserved auth/Nova/OmitHeaders, while ordinary headers and live tokens retain native retry.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationPhysicalCausesSourceRestoreAndCancelKeepActualReceipt`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationNativeRetryOwnsBodyHeadersAndPreservesLiveToken`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationRejectionsNeverIgnoreOrLookup`.
- Independent Reference/action Envelope/header/status and discovery byte/header copies retain physical observations without using response-cache state.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationActualEvidenceOwnsDiscoveryAndAction`.
- Failover defaults always emit allow_attached_volume=false and secondary_backend_id=null; explicit false/empty backend, true/string, bulk reset-to-null and factory/retained option mutation preserve typed JSON values.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationFailoverFalseNullEmptyAndOptsReset`, `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationOwnedOptionsRetainedMutationAndSelectedAliases`.
- Physical Read/Close/both401 retains original actual401 receipt and physical causes with oneHTTP/zero willingRetry/Reauth callbacks; clean401 renews token once and sends two identical owned POST bodies.
  Tests: `blockstorage/v3/groups/python_replication_contracts_test.go:TestPythonCinderGroupReplicationFaulty401StopsCallbacksAndClean401RenewsToken`.

Python/Go differences:

- Pinned Group.failover_replication literally sends modify_body_for_action, while stock Cinder27 requires failover_replication. Go default corrects the server action and explicit PinnedSourceWire replays the fixed SDK key, preserving actual400 rather than claiming stock acceptance.
- The exact Source proxy accepts a Group object or ID and no Header/Microversion/PinnedSourceWire extensions. Go takes an owned literal nonblank UTF-8 ID, uses typed known failover parameters, snapshots options, and rejects route/session/protocol/header/JSON-domain invalid input instead of supporting mutable subclasses/Resource/Munch arbitrary dynamic values.
- Go enforces action minimum3.38 before mutation; Source Group._action normally only selects class/session max3.38 and lets an explicitly selected older version reach the server. Empty Go override requests default required negotiation. Global latest/different-major selected versions do not establish the finite-major minimum in this profile.
- Selected finite >=3.38 is retained without extra discovery, and therefore does not prove advertised server/driver support. Default discovery uses shared finite readers and actual bounds; Group class ceiling is3.38, distinct from GroupSnapshot3.29. No mutable shared client default or fabricated successful Resource is changed/returned.
- Go extends Python None return with a Reference and actual ACK/error/discovery receipt. HTTP202 does not prove completion/current durable replication state; preparation discovery remains status0 in action carrier. Body decode/UTF8 is not added to accepted Source action success.
- Shared source guard/REST policy preserves actual physical/native/callback/cancellation causes and owned payload/version/reserved headers, while ordinary native retry headers/live token refresh remain available. Persistent mutable/cache/dirty/session customization and actual backend acceptance remain shared SDK boundaries.
- Native *ErrUnableToReauthenticate keeps non-Unwrap ErrOriginal/ErrReauth fields; actual401 receipt copies only ErrOriginal, preserving pointer wrapper and physical ErrReauth causes. Test validates fields directly rather than demanding a nonexistent native Unwrap.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("group", help="literal group ID")
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--action", choices=["enable", "disable", "failover"],
                        default="enable")
    parser.add_argument("--allow-attached-volume", action="store_true")
    parser.add_argument("--secondary-backend-id", default=None)
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    if args.action == "enable":
        conn.block_storage.enable_group_replication(args.group)
    elif args.action == "disable":
        conn.block_storage.disable_group_replication(args.group)
    else:
        conn.block_storage.failover_group_replication(
            args.group,
            allowed_attached_volume=args.allow_attached_volume,
            secondary_backend_id=args.secondary_backend_id,
        )
    print("action call completed without exception")


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("group", "", "literal group ID")
    action := flag.String("action", "enable", "enable/disable/failover")
    version := flag.String("microversion", "3.38", "selected Cinder version")
    allow := flag.Bool("allow-attached-volume", false, "allow attached volumes")
    backend := flag.String("secondary-backend-id", "", "optional backend ID")
    sourceWire := flag.Bool("pinned-source-wire", false, "replay the fixed Source body")
    flag.Parse()
    backendPresent := false
    flag.Visit(func(f *flag.Flag) { if f.Name == "secondary-backend-id" { backendPresent = true } })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    var value *groups.ReplicationRecordResult
    switch *action {
    case "enable":
        value, err = service.Groups.EnableReplicationRecord(ctx, *id,
            groups.WithRecordReplicationPinnedSourceWire(*sourceWire))
    case "disable":
        value, err = service.Groups.DisableReplicationRecord(ctx, *id,
            groups.WithRecordReplicationPinnedSourceWire(*sourceWire))
    case "failover":
        options := []groups.RecordFailoverOption{
            groups.WithRecordFailoverAllowedAttachedVolume(*allow),
            groups.WithRecordFailoverPinnedSourceWire(*sourceWire),
        }
        if backendPresent { options = append(options, groups.WithRecordFailoverSecondaryBackendID(*backend)) }
        value, err = service.Groups.FailoverReplicationRecord(ctx, *id, options...)
    default:
        log.Fatal("unknown action")
    }
    if value != nil {
        fmt.Printf("reference=%q actual_http=%d actual_body=%q discovery=%d\n",
            value.Reference, value.StatusCode, value.Envelope, len(value.Discovery))
    }
    if err != nil { log.Fatal(err) }
}
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2275-L2319)
- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L101-L140)
- [Group leaf](groups-python.md)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#group-replication-groups-action)
- [Group Actions](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py#L81-L116)
- [Policy](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html)
- [Groups](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/schemas/groups.py#L132-L173)
- [controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/groups.py#L339-L404)
- [Install](../../docs/install.md)
