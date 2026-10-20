# Cloud QoS Rule Creates Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/create_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateQoSBandwidthLimitRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateCloudBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateDirection`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateMaxBurstKBPS`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Required max_kbps/dscp_mark/min_kbps is positional JSON-domain data and retained by presence: zero/null/false/string/large integer are not omitted or coerced. Concrete optional kwargs/options are owned snapshots; invalid/duplicate required or bound policy/unknown keys fail before extension HTTP, with no application builder/interface responsibility.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateValidKwargsOwnedOptionsAndRequiredRawPresence`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Each call fully consumes a fresh uncached qos extension discovery before policy/rule stages. Initial alias discovery does not hide late malformed alias or later-page HTTP failures. Completed extension/policy evidence remains separate from any actual rule mutation receipt; failed gate causes no policy/rule HTTP.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCompleteGatesAndOptionalFailuresKeepStages`.
- Source-only policy lookup opts existing FindSourceRecord into Source status/equality/link/fault policy without modifying legacy state.Policy or native APIs. Exact member GET follows clean400/403/404 all-page list fallback, raises missing/late ambiguity before POST, and uses the returned policy ID rather than caller identity.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`.
- Translated cloud attrs alone enter collection POST. Source Body seed/defaults merge partial response with explicit-null override; missing response required fields remain only in Resource seed while Wire/Envelope retain actual response. Bound URI parent does not enter request body; foreign response qos_policy_id remains actual Wire while Resource reads the original raw JSON URI parent. Null/false/zero URI formatting uses None/False/0 and arbitrary integer precision survives.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Source selected create microversion and owned null location override same-named response members only in Resource view; actual Wire is preserved. Source100..399 handles lookup/POST199 separately from legacy baseline. Concrete input/protocol/API/source/header snapshots are captured before caller callbacks and guard every dependent stage.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Actual rejected400/403/404/409/503 POST preserves native value ErrUnexpectedResponseCode and actual status/body/header rather than ignored missing/success. Accepted/rejected200/401/404/503 read+Close/cancel/source faults retain actual receipt/causes with zero retry/reauth hooks and explicit bounded unexpected hook fences. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal value response proof and ErrReauth even without native Unwrap.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePhysicalFailuresPreserveActualProofWithoutHooks`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Clean503 retry and401 live token renewal remain supported. Changed selected route aborts before a resend with the original503 receipt, while callback MoreHeaders mutation does not affect the captured initial/gate/policy/POST headers. Fault/cancel/source failures are not normalized to missing or successful creation.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`.
- BW accepts optional max_burst_kbps/direction only and resolves policy with ignore_missing=True before the direction gate. A nonnull direction (false/zero/empty string/array/object included) triggers a second fresh complete qos-bw-limit-direction discovery. Unavailable extension removes only direction and marks DirectionDropped; null retains direction:null with no extra gate, and HTTP/physical gate failure is never treated as unsupported direction.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateBandwidthDirectionNonnullGateAfterPolicy`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCompleteGatesAndOptionalFailuresKeepStages`.

Python/Go differences:

- The concrete Go carrier exposes Rule/Wire/Envelope/Header/StatusCode, independent Policy/Extensions and DirectionDropped instead of only Python rule Resource. Policy is the independently owned actual lookup view/receipt, not Python mutable borrowed object side effects. Request seed/no-op/class state is not fabricated as server response evidence.
- Required values accept complete UTF8 JSON/Go primitives with library-owned concrete With options; custom Python runtime objects, mutable Resource/class/adapter/cache/session/current_location and dynamic class attribute dispatch remain shared SDK-R1/SDK-C1/SDK-S1 foundation differences. Their supported increment remains 0. These three cloud mappings do not promote general Proxy.create raw attrs, update/delete/details or full SDK parity.
- Go guards selected source/header/protocol and service-relative literal parent segments: slash/percent are escaped as one segment; blank/control/traversal identities fail preflight. Source URI %s formatting can introduce arbitrary path structure. Null/false/0 parents are explicitly preserved as None/False/0 paths with original raw URI view rather than incorrectly applying falsey policy member DELETE joining.
- JSON object wire key order is not guaranteed. Raw request values are forwarded without adding native constructor requirements or type/default coercions; actual server value/range/driver validation may reject null/false/string/rates even though the pinned Source forwards them. Generic JSON float serialization/runtime-domain differences remain explicit shared foundation policy.
- Fixed Neutron27 child create policies require project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Ability to read a shared policy does not confer child creation permission. Source/runtime/mock/race/main checks do not assert authenticated backend, role, driver or deployment policy acceptance.
- Actual pinned Source cloud decorator/methods + Neutron Proxy.find/create + Resource.find/new/create were imported from ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe and executed under Python3.14.8 in a bounded62-case reference. Only Adapter HTTP methods and extension generator were substituted. An additional 45-case comparison covers response, URI, and version behavior. This is not live OpenStack acceptance.

### `python:cloud/_network/NetworkCloudMixin/create_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateQoSDSCPMarkingRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateCloudDSCPMarkingRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateJSON`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Required max_kbps/dscp_mark/min_kbps is positional JSON-domain data and retained by presence: zero/null/false/string/large integer are not omitted or coerced. Concrete optional kwargs/options are owned snapshots; invalid/duplicate required or bound policy/unknown keys fail before extension HTTP, with no application builder/interface responsibility.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateValidKwargsOwnedOptionsAndRequiredRawPresence`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Each call fully consumes a fresh uncached qos extension discovery before policy/rule stages. Initial alias discovery does not hide late malformed alias or later-page HTTP failures. Completed extension/policy evidence remains separate from any actual rule mutation receipt; failed gate causes no policy/rule HTTP.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCompleteGatesAndOptionalFailuresKeepStages`.
- Source-only policy lookup opts existing FindSourceRecord into Source status/equality/link/fault policy without modifying legacy state.Policy or native APIs. Exact member GET follows clean400/403/404 all-page list fallback, raises missing/late ambiguity before POST, and uses the returned policy ID rather than caller identity.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`.
- Translated cloud attrs alone enter collection POST. Source Body seed/defaults merge partial response with explicit-null override; missing response required fields remain only in Resource seed while Wire/Envelope retain actual response. Bound URI parent does not enter request body; foreign response qos_policy_id remains actual Wire while Resource reads the original raw JSON URI parent. Null/false/zero URI formatting uses None/False/0 and arbitrary integer precision survives.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Source selected create microversion and owned null location override same-named response members only in Resource view; actual Wire is preserved. Source100..399 handles lookup/POST199 separately from legacy baseline. Concrete input/protocol/API/source/header snapshots are captured before caller callbacks and guard every dependent stage.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Actual rejected400/403/404/409/503 POST preserves native value ErrUnexpectedResponseCode and actual status/body/header rather than ignored missing/success. Accepted/rejected200/401/404/503 read+Close/cancel/source faults retain actual receipt/causes with zero retry/reauth hooks and explicit bounded unexpected hook fences. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal value response proof and ErrReauth even without native Unwrap.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePhysicalFailuresPreserveActualProofWithoutHooks`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Clean503 retry and401 live token renewal remain supported. Changed selected route aborts before a resend with the original503 receipt, while callback MoreHeaders mutation does not affect the captured initial/gate/policy/POST headers. Fault/cancel/source failures are not normalized to missing or successful creation.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`.
- DSCP cloud method has no optional kwargs in pinned Python; concrete Go variadic options provide consistent SDK call syntax but any actual optional field is rejected before HTTP. Policy lookup defaults ignore_missing=True and absence becomes a cloud NotFound error before rule POST. No direction gate/default attrs are introduced.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateValidKwargsOwnedOptionsAndRequiredRawPresence`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback`.

Python/Go differences:

- The concrete Go carrier exposes Rule/Wire/Envelope/Header/StatusCode, independent Policy/Extensions and DirectionDropped instead of only Python rule Resource. Policy is the independently owned actual lookup view/receipt, not Python mutable borrowed object side effects. Request seed/no-op/class state is not fabricated as server response evidence.
- Required values accept complete UTF8 JSON/Go primitives with library-owned concrete With options; custom Python runtime objects, mutable Resource/class/adapter/cache/session/current_location and dynamic class attribute dispatch remain shared SDK-R1/SDK-C1/SDK-S1 foundation differences. Their supported increment remains 0. These three cloud mappings do not promote general Proxy.create raw attrs, update/delete/details or full SDK parity.
- Go guards selected source/header/protocol and service-relative literal parent segments: slash/percent are escaped as one segment; blank/control/traversal identities fail preflight. Source URI %s formatting can introduce arbitrary path structure. Null/false/0 parents are explicitly preserved as None/False/0 paths with original raw URI view rather than incorrectly applying falsey policy member DELETE joining.
- JSON object wire key order is not guaranteed. Raw request values are forwarded without adding native constructor requirements or type/default coercions; actual server value/range/driver validation may reject null/false/string/rates even though the pinned Source forwards them. Generic JSON float serialization/runtime-domain differences remain explicit shared foundation policy.
- Fixed Neutron27 child create policies require project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Ability to read a shared policy does not confer child creation permission. Source/runtime/mock/race/main checks do not assert authenticated backend, role, driver or deployment policy acceptance.
- Actual pinned Source cloud decorator/methods + Neutron Proxy.find/create + Resource.find/new/create were imported from ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe and executed under Python3.14.8 in a bounded62-case reference. Only Adapter HTTP methods and extension generator were substituted. An additional 45-case comparison covers response, URI, and version behavior. This is not live OpenStack acceptance.

### `python:cloud/_network/NetworkCloudMixin/create_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateQoSMinimumBandwidthRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateCloudMinimumBandwidthRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateDirection`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Required max_kbps/dscp_mark/min_kbps is positional JSON-domain data and retained by presence: zero/null/false/string/large integer are not omitted or coerced. Concrete optional kwargs/options are owned snapshots; invalid/duplicate required or bound policy/unknown keys fail before extension HTTP, with no application builder/interface responsibility.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateValidKwargsOwnedOptionsAndRequiredRawPresence`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Each call fully consumes a fresh uncached qos extension discovery before policy/rule stages. Initial alias discovery does not hide late malformed alias or later-page HTTP failures. Completed extension/policy evidence remains separate from any actual rule mutation receipt; failed gate causes no policy/rule HTTP.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCompleteGatesAndOptionalFailuresKeepStages`.
- Source-only policy lookup opts existing FindSourceRecord into Source status/equality/link/fault policy without modifying legacy state.Policy or native APIs. Exact member GET follows clean400/403/404 all-page list fallback, raises missing/late ambiguity before POST, and uses the returned policy ID rather than caller identity.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`.
- Translated cloud attrs alone enter collection POST. Source Body seed/defaults merge partial response with explicit-null override; missing response required fields remain only in Resource seed while Wire/Envelope retain actual response. Bound URI parent does not enter request body; foreign response qos_policy_id remains actual Wire while Resource reads the original raw JSON URI parent. Null/false/zero URI formatting uses None/False/0 and arbitrary integer precision survives.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Source selected create microversion and owned null location override same-named response members only in Resource view; actual Wire is preserved. Source100..399 handles lookup/POST199 separately from legacy baseline. Concrete input/protocol/API/source/header snapshots are captured before caller callbacks and guard every dependent stage.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Actual rejected400/403/404/409/503 POST preserves native value ErrUnexpectedResponseCode and actual status/body/header rather than ignored missing/success. Accepted/rejected200/401/404/503 read+Close/cancel/source faults retain actual receipt/causes with zero retry/reauth hooks and explicit bounded unexpected hook fences. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal value response proof and ErrReauth even without native Unwrap.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePhysicalFailuresPreserveActualProofWithoutHooks`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Clean503 retry and401 live token renewal remain supported. Changed selected route aborts before a resend with the original503 receipt, while callback MoreHeaders mutation does not affect the captured initial/gate/policy/POST headers. Fault/cancel/source failures are not normalized to missing or successful creation.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`.
- Minimum accepts optional direction only and resolves policy with ignore_missing=False. Missing policy raises strict find error before POST. Unlike BW, direction never triggers qos-bw-limit-direction discovery or gets dropped; raw required min_kbps and raw direction are forwarded as supplied.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateBandwidthDirectionNonnullGateAfterPolicy`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback`.

Python/Go differences:

- The concrete Go carrier exposes Rule/Wire/Envelope/Header/StatusCode, independent Policy/Extensions and DirectionDropped instead of only Python rule Resource. Policy is the independently owned actual lookup view/receipt, not Python mutable borrowed object side effects. Request seed/no-op/class state is not fabricated as server response evidence.
- Required values accept complete UTF8 JSON/Go primitives with library-owned concrete With options; custom Python runtime objects, mutable Resource/class/adapter/cache/session/current_location and dynamic class attribute dispatch remain shared SDK-R1/SDK-C1/SDK-S1 foundation differences. Their supported increment remains 0. These three cloud mappings do not promote general Proxy.create raw attrs, update/delete/details or full SDK parity.
- Go guards selected source/header/protocol and service-relative literal parent segments: slash/percent are escaped as one segment; blank/control/traversal identities fail preflight. Source URI %s formatting can introduce arbitrary path structure. Null/false/0 parents are explicitly preserved as None/False/0 paths with original raw URI view rather than incorrectly applying falsey policy member DELETE joining.
- JSON object wire key order is not guaranteed. Raw request values are forwarded without adding native constructor requirements or type/default coercions; actual server value/range/driver validation may reject null/false/string/rates even though the pinned Source forwards them. Generic JSON float serialization/runtime-domain differences remain explicit shared foundation policy.
- Fixed Neutron27 child create policies require project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Ability to read a shared policy does not confer child creation permission. Source/runtime/mock/race/main checks do not assert authenticated backend, role, driver or deployment policy acceptance.
- Actual pinned Source cloud decorator/methods + Neutron Proxy.find/create + Resource.find/new/create were imported from ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe and executed under Python3.14.8 in a bounded62-case reference. Only Adapter HTTP methods and extension generator were substituted. An additional 45-case comparison covers response, URI, and version behavior. This is not live OpenStack acceptance.

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
parser.add_argument("--value", default="0")
parser.add_argument("--kwargs", default="{}")
args = parser.parse_args()
value = json.loads(args.value)
kwargs = json.loads(args.kwargs)
if not isinstance(kwargs, dict):
    parser.error("kwargs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    if args.kind == "bandwidth":
        result = conn.create_qos_bandwidth_limit_rule(args.policy, value, **kwargs)
    elif args.kind == "dscp":
        result = conn.create_qos_dscp_marking_rule(args.policy, value, **kwargs)
    else:
        result = conn.create_qos_minimum_bandwidth_rule(args.policy, value, **kwargs)
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
	kind := flag.String("kind", "bandwidth", "bandwidth, dscp or minimum")
	policy := flag.String("policy", "", "literal policy ID or exact name")
	value := flag.String("value", "0", "JSON required max_kbps/dscp_mark/min_kbps")
	kwargs := flag.String("kwargs", "{}", "JSON optional cloud kwargs")
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
	required := json.RawMessage(*value)
	option := network.WithQoSRuleCreateJSON(json.RawMessage(*kwargs))
	var result *network.QoSRuleCreateResult
	switch *kind {
	case "bandwidth":
		result, err = service.CreateQoSBandwidthLimitRuleRecord(ctx, *policy, required, option)
	case "dscp":
		result, err = service.CreateQoSDSCPMarkingRuleRecord(ctx, *policy, required, option)
	case "minimum":
		result, err = service.CreateQoSMinimumBandwidthRuleRecord(ctx, *policy, required, option)
	}
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(result.Rule, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [Install](../docs/install.md)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L226-L245)
- [DSCP](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L361-L380)
- [Minimum](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L438-L457)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L56-L68)
- [BW cloud create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1620-L1668)
- [DSCP cloud create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1844-L1879)
- [Minimum cloud create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2040-L2077)
- [Source Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
