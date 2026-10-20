# Cloud QoS Rule Updates Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/update_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateQoSBandwidthLimitRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetCloudBandwidthLimitUpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateCloudBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.QoSRuleUpdateRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateDirection`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateMaxKBPS`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateMaxBurstKBPS`

- All named cloud kwargs are library-validated before HTTP, with concrete cloned JSON/map/primitive With options. Unknown/bound identity keys/nil or invalid JSON fail before qos gate. No caller builder/interface is needed; explicit null/zero remain present and later individual options replace prior bulk keys.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateValidKwargsBeforeHTTPAndOwnedOptions`.
- Each valid call fully consumes fresh uncached qos extension discovery and Source-only policy Find. Policy missing raises even for empty kwargs. BW/DSCP default ignore_missing=True then cloud NotFound, Minimum ignore_missing=False. Empty kwargs only after policy/optional gate returns None/no current GET/no PUT and does not validate unused child identity.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`.
- Nonempty attrs trigger exactly one strict current-rule GET with no child find/list fallback/ignored404. Dirty PUT uses actual returned rule identity and separate current/policy/rule evidence. Rejected400/403/404/409/503 retain native value status/body/header and are not normalized to successful update or None.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateStrictChildGETAndFailedPUTKeepIndependentReceipts`.
- A private borrowed seed owns original raw Body presence, URI parent, actual receipt, family/API/client/source guard. Clone deep-copies raw state and private proof; exported Resource/Wire/QoSPolicyID/Envelope/Header mutations do not create false dirtiness, forge identity or change no-PUT proof. Foreign API/family or changed selected source rejects before mutation.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`.
- Raw Source dirty equality uses presence and CPython-loaded values rather than projected defaults/getters: missing→null is dirty, true/1 and false/0 are equal, float precision/underflow equality matches Source. PUT contains changed keys only; partial response seeds untouched/new fields only in Resource view while Wire remains actual. No-dirty returns cloned private current receipt without PUT evidence.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`.
- Equal null-ID updates return without ID validation or PUT; dirty null/missing raises pre-PUT. Dirty falsey nonnull0/false/emptystring/0.0/underflow/emptyarray/object follows actual Source collection trailing-slash PUT through explicit new copied SourceCollectionPUT profile. Root shared seam requires SourceFindPolicy+SourceCollectionPUT+PUT empty identity and preserves legacy false/default/other-method behavior; selected fixed profile.Path plus slash is the only collection target.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Private nil raw parent is normalized to JSON null and path formatting follows Python %s without losing raw URI getter state. Foreign response qos_policy_id/microversion/location stay in actual Wire while Resource reads original raw URI/selected version/owned null location. Source _update resets version to null on no-dirty; a dirty commit restores selected version.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Actual strict GET/PUT accepted/rejected200/401/404/503 read+Close/cancel/source faults preserve stage receipt and exact causes with zero retry/reauth hooks and explicit unexpected bounded sentinels. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal native value response and ErrReauth despite no native Unwrap. Faults never hide as missing/None/no-op.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdatePhysicalGETPUTFaultsKeepNative401AndNoHooks`.
- New Source profiles opt100..399 in while existing typed/native profiles stay unchanged. Source PUT199, clean503 retry/401 live token renewal and sticky selected route guards retain actual proof. Source/header capture precedes option callbacks, so callback MoreHeaders mutation cannot leak into ongoing requests; changed route before retry aborts with original503.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateCleanRetryRenewal199AndHeaderCapture`.
- BW update allows max_kbps/max_burst_kbps/direction. After policy lookup, every nonnull direction (false/0/emptystr/array/object included) triggers an independent fresh qos-bw-limit-direction inventory; unavailable drops direction only, potentially making kwargs empty before child GET. Null keeps its field with no optional gate; optional HTTP/physical failure is not treated as unsupported direction.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`.

Python/Go differences:

- Go carrier separates Rule/Current/Policy/Extensions and NoOp/DirectionDropped rather than Python Resource/None only. Actual GET and PUT receipt are independent owned snapshots; no-dirty returns prior receipt without fabricated PUT proof. Mutable Python borrowed object side effects are not exposed as server observations.
- Complete UTF8 JSON/concrete With defaults and owned snapshot boundaries remain explicit Go policy. Custom runtime objects, mutable Resource/class/adapter/session/cache/current_location and dynamic class dispatch remain SDK-R1/SDK-C1/SDK-S1 shared foundation remaining with supported increment0. These three cloud mappings do not promote general Proxy raw update kwargs or full SDK parity.
- Go literal service-relative URI segments are escaped and blank/control/traversal identities rejected. Source slash/percent URI formatting may create arbitrary path structure. Actual falsey nonnull collection PUT is implemented through Source-only opt-in rather than dismissed as an API convenience difference; null dirty ID still errors and equal null ID does not require a route.
- JSON object wire order is not guaranteed; raw keyword data is not coerced/clamped to native rate/direction types/defaults. Server schema/range/driver validation remains authoritative. Actual Source74-case execution under CPython3.14.8 imported pinned cloud decorator/Proxy find-get-update/Resource find-fetch-_update-commit, with HTTP Adapter and extension generator fixture only.
- Stock Neutron27 BW/DSCP/minimum update policy is project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Readable shared policy does not confer update permission. Runtime/reference/mock/race/main checks do not prove authenticated backend/driver/deployment acceptance.

### `python:cloud/_network/NetworkCloudMixin/update_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateQoSDSCPMarkingRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetCloudDSCPMarkingUpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateCloudDSCPMarkingRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.QoSRuleUpdateRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateDSCPMark`

- All named cloud kwargs are library-validated before HTTP, with concrete cloned JSON/map/primitive With options. Unknown/bound identity keys/nil or invalid JSON fail before qos gate. No caller builder/interface is needed; explicit null/zero remain present and later individual options replace prior bulk keys.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateValidKwargsBeforeHTTPAndOwnedOptions`.
- Each valid call fully consumes fresh uncached qos extension discovery and Source-only policy Find. Policy missing raises even for empty kwargs. BW/DSCP default ignore_missing=True then cloud NotFound, Minimum ignore_missing=False. Empty kwargs only after policy/optional gate returns None/no current GET/no PUT and does not validate unused child identity.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`.
- Nonempty attrs trigger exactly one strict current-rule GET with no child find/list fallback/ignored404. Dirty PUT uses actual returned rule identity and separate current/policy/rule evidence. Rejected400/403/404/409/503 retain native value status/body/header and are not normalized to successful update or None.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateStrictChildGETAndFailedPUTKeepIndependentReceipts`.
- A private borrowed seed owns original raw Body presence, URI parent, actual receipt, family/API/client/source guard. Clone deep-copies raw state and private proof; exported Resource/Wire/QoSPolicyID/Envelope/Header mutations do not create false dirtiness, forge identity or change no-PUT proof. Foreign API/family or changed selected source rejects before mutation.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`.
- Raw Source dirty equality uses presence and CPython-loaded values rather than projected defaults/getters: missing→null is dirty, true/1 and false/0 are equal, float precision/underflow equality matches Source. PUT contains changed keys only; partial response seeds untouched/new fields only in Resource view while Wire remains actual. No-dirty returns cloned private current receipt without PUT evidence.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`.
- Equal null-ID updates return without ID validation or PUT; dirty null/missing raises pre-PUT. Dirty falsey nonnull0/false/emptystring/0.0/underflow/emptyarray/object follows actual Source collection trailing-slash PUT through explicit new copied SourceCollectionPUT profile. Root shared seam requires SourceFindPolicy+SourceCollectionPUT+PUT empty identity and preserves legacy false/default/other-method behavior; selected fixed profile.Path plus slash is the only collection target.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Private nil raw parent is normalized to JSON null and path formatting follows Python %s without losing raw URI getter state. Foreign response qos_policy_id/microversion/location stay in actual Wire while Resource reads original raw URI/selected version/owned null location. Source _update resets version to null on no-dirty; a dirty commit restores selected version.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Actual strict GET/PUT accepted/rejected200/401/404/503 read+Close/cancel/source faults preserve stage receipt and exact causes with zero retry/reauth hooks and explicit unexpected bounded sentinels. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal native value response and ErrReauth despite no native Unwrap. Faults never hide as missing/None/no-op.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdatePhysicalGETPUTFaultsKeepNative401AndNoHooks`.
- New Source profiles opt100..399 in while existing typed/native profiles stay unchanged. Source PUT199, clean503 retry/401 live token renewal and sticky selected route guards retain actual proof. Source/header capture precedes option callbacks, so callback MoreHeaders mutation cannot leak into ongoing requests; changed route before retry aborts with original503.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateCleanRetryRenewal199AndHeaderCapture`.
- DSCP accepts dscp_mark only and default ignore_missing=True policy lookup with cloud NotFound on nil. Empty attrs return None after qos/policy but before current GET; nonempty attrs use strict current GET and raw-only delta commit.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`, `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse`.

Python/Go differences:

- Go carrier separates Rule/Current/Policy/Extensions and NoOp/DirectionDropped rather than Python Resource/None only. Actual GET and PUT receipt are independent owned snapshots; no-dirty returns prior receipt without fabricated PUT proof. Mutable Python borrowed object side effects are not exposed as server observations.
- Complete UTF8 JSON/concrete With defaults and owned snapshot boundaries remain explicit Go policy. Custom runtime objects, mutable Resource/class/adapter/session/cache/current_location and dynamic class dispatch remain SDK-R1/SDK-C1/SDK-S1 shared foundation remaining with supported increment0. These three cloud mappings do not promote general Proxy raw update kwargs or full SDK parity.
- Go literal service-relative URI segments are escaped and blank/control/traversal identities rejected. Source slash/percent URI formatting may create arbitrary path structure. Actual falsey nonnull collection PUT is implemented through Source-only opt-in rather than dismissed as an API convenience difference; null dirty ID still errors and equal null ID does not require a route.
- JSON object wire order is not guaranteed; raw keyword data is not coerced/clamped to native rate/direction types/defaults. Server schema/range/driver validation remains authoritative. Actual Source74-case execution under CPython3.14.8 imported pinned cloud decorator/Proxy find-get-update/Resource find-fetch-_update-commit, with HTTP Adapter and extension generator fixture only.
- Stock Neutron27 BW/DSCP/minimum update policy is project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Readable shared policy does not confer update permission. Runtime/reference/mock/race/main checks do not prove authenticated backend/driver/deployment acceptance.

### `python:cloud/_network/NetworkCloudMixin/update_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateQoSMinimumBandwidthRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetCloudMinimumBandwidthUpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateCloudMinimumBandwidthRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.QoSRuleUpdateRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateDirection`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateMinKBPS`

- All named cloud kwargs are library-validated before HTTP, with concrete cloned JSON/map/primitive With options. Unknown/bound identity keys/nil or invalid JSON fail before qos gate. No caller builder/interface is needed; explicit null/zero remain present and later individual options replace prior bulk keys.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateValidKwargsBeforeHTTPAndOwnedOptions`.
- Each valid call fully consumes fresh uncached qos extension discovery and Source-only policy Find. Policy missing raises even for empty kwargs. BW/DSCP default ignore_missing=True then cloud NotFound, Minimum ignore_missing=False. Empty kwargs only after policy/optional gate returns None/no current GET/no PUT and does not validate unused child identity.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`.
- Nonempty attrs trigger exactly one strict current-rule GET with no child find/list fallback/ignored404. Dirty PUT uses actual returned rule identity and separate current/policy/rule evidence. Rejected400/403/404/409/503 retain native value status/body/header and are not normalized to successful update or None.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateStrictChildGETAndFailedPUTKeepIndependentReceipts`.
- A private borrowed seed owns original raw Body presence, URI parent, actual receipt, family/API/client/source guard. Clone deep-copies raw state and private proof; exported Resource/Wire/QoSPolicyID/Envelope/Header mutations do not create false dirtiness, forge identity or change no-PUT proof. Foreign API/family or changed selected source rejects before mutation.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`.
- Raw Source dirty equality uses presence and CPython-loaded values rather than projected defaults/getters: missing→null is dirty, true/1 and false/0 are equal, float precision/underflow equality matches Source. PUT contains changed keys only; partial response seeds untouched/new fields only in Resource view while Wire remains actual. No-dirty returns cloned private current receipt without PUT evidence.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`.
- Equal null-ID updates return without ID validation or PUT; dirty null/missing raises pre-PUT. Dirty falsey nonnull0/false/emptystring/0.0/underflow/emptyarray/object follows actual Source collection trailing-slash PUT through explicit new copied SourceCollectionPUT profile. Root shared seam requires SourceFindPolicy+SourceCollectionPUT+PUT empty identity and preserves legacy false/default/other-method behavior; selected fixed profile.Path plus slash is the only collection target.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Private nil raw parent is normalized to JSON null and path formatting follows Python %s without losing raw URI getter state. Foreign response qos_policy_id/microversion/location stay in actual Wire while Resource reads original raw URI/selected version/owned null location. Source _update resets version to null on no-dirty; a dirty commit restores selected version.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Actual strict GET/PUT accepted/rejected200/401/404/503 read+Close/cancel/source faults preserve stage receipt and exact causes with zero retry/reauth hooks and explicit unexpected bounded sentinels. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal native value response and ErrReauth despite no native Unwrap. Faults never hide as missing/None/no-op.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdatePhysicalGETPUTFaultsKeepNative401AndNoHooks`.
- New Source profiles opt100..399 in while existing typed/native profiles stay unchanged. Source PUT199, clean503 retry/401 live token renewal and sticky selected route guards retain actual proof. Source/header capture precedes option callbacks, so callback MoreHeaders mutation cannot leak into ongoing requests; changed route before retry aborts with original503.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateCleanRetryRenewal199AndHeaderCapture`.
- Minimum accepts min_kbps/direction and uses strict ignore_missing=False policy Find; direction never triggers BW optional gate. Empty attrs still require policy lookup, and nonempty attrs strict current GET precedes dirty-only commit.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`, `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse`.

Python/Go differences:

- Go carrier separates Rule/Current/Policy/Extensions and NoOp/DirectionDropped rather than Python Resource/None only. Actual GET and PUT receipt are independent owned snapshots; no-dirty returns prior receipt without fabricated PUT proof. Mutable Python borrowed object side effects are not exposed as server observations.
- Complete UTF8 JSON/concrete With defaults and owned snapshot boundaries remain explicit Go policy. Custom runtime objects, mutable Resource/class/adapter/session/cache/current_location and dynamic class dispatch remain SDK-R1/SDK-C1/SDK-S1 shared foundation remaining with supported increment0. These three cloud mappings do not promote general Proxy raw update kwargs or full SDK parity.
- Go literal service-relative URI segments are escaped and blank/control/traversal identities rejected. Source slash/percent URI formatting may create arbitrary path structure. Actual falsey nonnull collection PUT is implemented through Source-only opt-in rather than dismissed as an API convenience difference; null dirty ID still errors and equal null ID does not require a route.
- JSON object wire order is not guaranteed; raw keyword data is not coerced/clamped to native rate/direction types/defaults. Server schema/range/driver validation remains authoritative. Actual Source74-case execution under CPython3.14.8 imported pinned cloud decorator/Proxy find-get-update/Resource find-fetch-_update-commit, with HTTP Adapter and extension generator fixture only.
- Stock Neutron27 BW/DSCP/minimum update policy is project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Readable shared policy does not confer update permission. Runtime/reference/mock/race/main checks do not prove authenticated backend/driver/deployment acceptance.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=("bandwidth", "dscp", "minimum"), default="bandwidth")
parser.add_argument("--policy", required=True)
parser.add_argument("--rule", default="")
parser.add_argument("--kwargs", default="{}")
args = parser.parse_args()
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    if args.kind == "bandwidth":
        result = conn.update_qos_bandwidth_limit_rule(args.policy, args.rule, **kwargs)
    elif args.kind == "dscp":
        result = conn.update_qos_dscp_marking_rule(args.policy, args.rule, **kwargs)
    else:
        result = conn.update_qos_minimum_bandwidth_rule(args.policy, args.rule, **kwargs)
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
	kind := flag.String("kind", "bandwidth", "bandwidth, dscp or minimum")
	policy := flag.String("policy", "", "literal policy ID or exact name")
	rule := flag.String("rule", "", "literal rule ID")
	kwargs := flag.String("kwargs", "{}", "JSON cloud update kwargs")
	flag.Parse()
	if *policy == "" {
		log.Fatal("policy is required")
	}
	if *kind != "bandwidth" && *kind != "dscp" && *kind != "minimum" {
		log.Fatal("unknown kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	option := network.WithQoSRuleUpdateJSON(json.RawMessage(*kwargs))
	var result *network.QoSRuleUpdateResult
	switch *kind {
	case "bandwidth":
		result, err = service.UpdateQoSBandwidthLimitRuleRecord(ctx, *policy, *rule, option)
	case "dscp":
		result, err = service.UpdateQoSDSCPMarkingRuleRecord(ctx, *policy, *rule, option)
	case "minimum":
		result, err = service.UpdateQoSMinimumBandwidthRuleRecord(ctx, *policy, *rule, option)
	}
	if err != nil {
		log.Fatal(err)
	}
	var value any
	if result.Rule != nil {
		value = result.Rule.Resource
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [Install](../docs/install.md)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L244-L260)
- [DSCP](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L376-L392)
- [Minimum](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L452-L468)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L56-L68)
- [BW update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1666-L1731)
- [DSCP update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1877-L1930)
- [Minimum update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2070-L2129)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
