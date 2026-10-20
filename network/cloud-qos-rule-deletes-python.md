# Cloud QoS Rule Deletes Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/delete_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteQoSBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteMicroversion`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Each cloud kind checks all QoS extension evidence, resolves the parent once and sends one bodyless literal rule DELETE; valid Source accepted ACK returns Completed/Deleted true, only clean rule404 returns Completed=true/Deleted=false, other400/403/408/409/500 preserve error/actual receipt.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteSourceTrueFalseAndOtherStatuses`.
- Absent extension, policy missing/ambiguity and minimum strict parent failure are errors without DELETE or false completion; actual Extensions and parent proof are retained without fabricating missing policy response.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteGateAndPolicyErrorsNeverBecomeFalse`.
- Early QoS match does not stop extension iteration; a later failed extension page keeps observed Extensions and original rejection, with no policy/DELETE/false result.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteConsumesEntireExtensionGateBeforeMutation`.
- Owned safe header/version options and initialized callback maps apply only to strict DELETE; blank/control/dot IDs fail after prerequisites with no DELETE receipt or missing bool.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteOwnedHeadersVersionAndSafeIDs`.
- Before-DELETE callback failure joins cancellation or original selected-client mutation while actual extension/policy evidence survives; already-cancelled call sends no HTTP.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteCallbackSourceAndCancellationFailuresKeepParents`.
- Actual truncated404/401 DELETE bytes/header/status/UnexpectedEOF retain physical evidence, zero willing retry/reauth callbacks and cannot become false; real parent/extension evidence survives.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteFaulty404And401KeepRealReceiptNotFalse`.
- A native hook expanding accepted404 or returning a callback404 after actual503 leaves operation incomplete and keeps actual DELETE status; neither can establish rule absence.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteExpanded404AndCallback404NeverCompleteFalse`.
- New Source-only deletion profile opts into199/204/302/399 ACK acceptance while preserving bodyless fixed family route and opaque actual bytes/header/status with no Resource/Wire.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSourceOpaqueACKAndStatusOptIn`.
- Direct leaf defaults ignore_missing true only for clean404; explicit false remains strict. Native400/403/404/409/429/500 returns owned actual DELETE receipt and operation context without decode.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteDefaultMissingAndStrictRejectionsKeepActualReceipt`.
- Source delete IDs use safe UTF8 literal path segments including slash/space/literal percent2F; blank/control/dot parent/rule and nil option fail before HTTP with no invented receipt.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSafeLiteralIdentityAndPreflightNoReceipt`.
- Concrete header/version option and callback results own snapshots, nil maps are initialized; simultaneous option error/cancellation or captured client mutation preserves all causes before HTTP.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteOwnedHeadersVersionCallbacksAndGuardCauses`.
- Accepted204 and rejected404/401/500 physical read/close/mixed faults keep actual receipt/native401.ErrOriginal/value/status/causes, close once and stop willing callbacks; default missing policy cannot suppress physical404.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeletePhysicalFaultsCannotBecomeMissing`.
- Accepted-code404 and callback404 after real503 remain terminal, preserve real receipt and do not set IgnoredMissing even under default direct-leaf missing policy.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteAccepted404AndCallback404NeverEstablishMissing`.
- Clean503 retry and401 renewal keep one logical bodyless DELETE route/conditional-header/selected-version policy; renewed tokens work and actual final ACK proof is retained.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteCleanRetryAndAuthenticationKeepFixedBodylessPolicy`.

Python/Go differences:

- Public Go returns an owned partial QoSRuleDeleteResult with Completed/Deleted plus actual Extensions/Policy/Receipt instead of Python bool or exception only. Deleted=false is Source missing only when Completed=true and err=nil.
- New receipt-only Source leaf methods default ignore_missing true like Proxy; each cloud helper always requests strict false, then uses shared resource.IgnoreMissing terminal policy only on clean child404. Error paths never fabricate a policy None or DELETE Resource/Wire.
- Bandwidth/DSCP Source policy lookup ignore_missing=True plus explicit None NotFound differs from minimum-bandwidth Source ignore_missing=False. SDK reuses shared Policy vs PolicyStrict and preserves that difference rather than making all parents default tolerant.
- Source mutation status100..399 opt-in applies to these new profiles through root Engine.Mutate e.codes; old false/generated/typed policies remain unchanged. Opaque ACK content is not forced into JSON.
- Source dynamic Resource/class/session/adapter/custom Python IDs and mutable state remain SDK-R1/C1/S1 foundations. Go accepts safe concrete UTF8 strings/options/context, rejects blank/control/dot IDs and dangerous route/header changes; whitespace-only Source identifiers can reach HTTP while Go rejects them explicitly. No empty ID silently becomes missing false.
- Safe Header/Microversion options and callback ownership are Go transport conveniences; original cloud helper has no such kwargs. Library owns prerequisite/gate/strict child/missing behavior without requiring caller builders/interfaces.
- Fixed Neutron27 project-scoped delete_policy_bandwidth_limit_rule/delete_policy_dscp_marking_rule/delete_policy_minimum_bandwidth_rule use ADMIN_OR_PARENT_OWNER_MANAGER; manager must own parent policy, admin alternative, deprecated ADMIN_ONLY distinguished. Shared visibility/read does not imply delete rights, and deployment policy overrides/QoS driver/real deletion were not tested.

### `python:cloud/_network/NetworkCloudMixin/delete_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteQoSDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteDSCPMarkingRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteMicroversion`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Each cloud kind checks all QoS extension evidence, resolves the parent once and sends one bodyless literal rule DELETE; valid Source accepted ACK returns Completed/Deleted true, only clean rule404 returns Completed=true/Deleted=false, other400/403/408/409/500 preserve error/actual receipt.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteSourceTrueFalseAndOtherStatuses`.
- Absent extension, policy missing/ambiguity and minimum strict parent failure are errors without DELETE or false completion; actual Extensions and parent proof are retained without fabricating missing policy response.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteGateAndPolicyErrorsNeverBecomeFalse`.
- Early QoS match does not stop extension iteration; a later failed extension page keeps observed Extensions and original rejection, with no policy/DELETE/false result.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteConsumesEntireExtensionGateBeforeMutation`.
- Owned safe header/version options and initialized callback maps apply only to strict DELETE; blank/control/dot IDs fail after prerequisites with no DELETE receipt or missing bool.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteOwnedHeadersVersionAndSafeIDs`.
- Before-DELETE callback failure joins cancellation or original selected-client mutation while actual extension/policy evidence survives; already-cancelled call sends no HTTP.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteCallbackSourceAndCancellationFailuresKeepParents`.
- Actual truncated404/401 DELETE bytes/header/status/UnexpectedEOF retain physical evidence, zero willing retry/reauth callbacks and cannot become false; real parent/extension evidence survives.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteFaulty404And401KeepRealReceiptNotFalse`.
- A native hook expanding accepted404 or returning a callback404 after actual503 leaves operation incomplete and keeps actual DELETE status; neither can establish rule absence.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteExpanded404AndCallback404NeverCompleteFalse`.
- New Source-only deletion profile opts into199/204/302/399 ACK acceptance while preserving bodyless fixed family route and opaque actual bytes/header/status with no Resource/Wire.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSourceOpaqueACKAndStatusOptIn`.
- Direct leaf defaults ignore_missing true only for clean404; explicit false remains strict. Native400/403/404/409/429/500 returns owned actual DELETE receipt and operation context without decode.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteDefaultMissingAndStrictRejectionsKeepActualReceipt`.
- Source delete IDs use safe UTF8 literal path segments including slash/space/literal percent2F; blank/control/dot parent/rule and nil option fail before HTTP with no invented receipt.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSafeLiteralIdentityAndPreflightNoReceipt`.
- Concrete header/version option and callback results own snapshots, nil maps are initialized; simultaneous option error/cancellation or captured client mutation preserves all causes before HTTP.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteOwnedHeadersVersionCallbacksAndGuardCauses`.
- Accepted204 and rejected404/401/500 physical read/close/mixed faults keep actual receipt/native401.ErrOriginal/value/status/causes, close once and stop willing callbacks; default missing policy cannot suppress physical404.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeletePhysicalFaultsCannotBecomeMissing`.
- Accepted-code404 and callback404 after real503 remain terminal, preserve real receipt and do not set IgnoredMissing even under default direct-leaf missing policy.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteAccepted404AndCallback404NeverEstablishMissing`.
- Clean503 retry and401 renewal keep one logical bodyless DELETE route/conditional-header/selected-version policy; renewed tokens work and actual final ACK proof is retained.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteCleanRetryAndAuthenticationKeepFixedBodylessPolicy`.

Python/Go differences:

- Public Go returns an owned partial QoSRuleDeleteResult with Completed/Deleted plus actual Extensions/Policy/Receipt instead of Python bool or exception only. Deleted=false is Source missing only when Completed=true and err=nil.
- New receipt-only Source leaf methods default ignore_missing true like Proxy; each cloud helper always requests strict false, then uses shared resource.IgnoreMissing terminal policy only on clean child404. Error paths never fabricate a policy None or DELETE Resource/Wire.
- Bandwidth/DSCP Source policy lookup ignore_missing=True plus explicit None NotFound differs from minimum-bandwidth Source ignore_missing=False. SDK reuses shared Policy vs PolicyStrict and preserves that difference rather than making all parents default tolerant.
- Source mutation status100..399 opt-in applies to these new profiles through root Engine.Mutate e.codes; old false/generated/typed policies remain unchanged. Opaque ACK content is not forced into JSON.
- Source dynamic Resource/class/session/adapter/custom Python IDs and mutable state remain SDK-R1/C1/S1 foundations. Go accepts safe concrete UTF8 strings/options/context, rejects blank/control/dot IDs and dangerous route/header changes; whitespace-only Source identifiers can reach HTTP while Go rejects them explicitly. No empty ID silently becomes missing false.
- Safe Header/Microversion options and callback ownership are Go transport conveniences; original cloud helper has no such kwargs. Library owns prerequisite/gate/strict child/missing behavior without requiring caller builders/interfaces.
- Fixed Neutron27 project-scoped delete_policy_bandwidth_limit_rule/delete_policy_dscp_marking_rule/delete_policy_minimum_bandwidth_rule use ADMIN_OR_PARENT_OWNER_MANAGER; manager must own parent policy, admin alternative, deprecated ADMIN_ONLY distinguished. Shared visibility/read does not imply delete rights, and deployment policy overrides/QoS driver/real deletion were not tested.

### `python:cloud/_network/NetworkCloudMixin/delete_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteQoSMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteMinimumBandwidthRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteMicroversion`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Each cloud kind checks all QoS extension evidence, resolves the parent once and sends one bodyless literal rule DELETE; valid Source accepted ACK returns Completed/Deleted true, only clean rule404 returns Completed=true/Deleted=false, other400/403/408/409/500 preserve error/actual receipt.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteSourceTrueFalseAndOtherStatuses`.
- Absent extension, policy missing/ambiguity and minimum strict parent failure are errors without DELETE or false completion; actual Extensions and parent proof are retained without fabricating missing policy response.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteGateAndPolicyErrorsNeverBecomeFalse`.
- Early QoS match does not stop extension iteration; a later failed extension page keeps observed Extensions and original rejection, with no policy/DELETE/false result.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteConsumesEntireExtensionGateBeforeMutation`.
- Owned safe header/version options and initialized callback maps apply only to strict DELETE; blank/control/dot IDs fail after prerequisites with no DELETE receipt or missing bool.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteOwnedHeadersVersionAndSafeIDs`.
- Before-DELETE callback failure joins cancellation or original selected-client mutation while actual extension/policy evidence survives; already-cancelled call sends no HTTP.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteCallbackSourceAndCancellationFailuresKeepParents`.
- Actual truncated404/401 DELETE bytes/header/status/UnexpectedEOF retain physical evidence, zero willing retry/reauth callbacks and cannot become false; real parent/extension evidence survives.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteFaulty404And401KeepRealReceiptNotFalse`.
- A native hook expanding accepted404 or returning a callback404 after actual503 leaves operation incomplete and keeps actual DELETE status; neither can establish rule absence.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteExpanded404AndCallback404NeverCompleteFalse`.
- New Source-only deletion profile opts into199/204/302/399 ACK acceptance while preserving bodyless fixed family route and opaque actual bytes/header/status with no Resource/Wire.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSourceOpaqueACKAndStatusOptIn`.
- Direct leaf defaults ignore_missing true only for clean404; explicit false remains strict. Native400/403/404/409/429/500 returns owned actual DELETE receipt and operation context without decode.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteDefaultMissingAndStrictRejectionsKeepActualReceipt`.
- Source delete IDs use safe UTF8 literal path segments including slash/space/literal percent2F; blank/control/dot parent/rule and nil option fail before HTTP with no invented receipt.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSafeLiteralIdentityAndPreflightNoReceipt`.
- Concrete header/version option and callback results own snapshots, nil maps are initialized; simultaneous option error/cancellation or captured client mutation preserves all causes before HTTP.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteOwnedHeadersVersionCallbacksAndGuardCauses`.
- Accepted204 and rejected404/401/500 physical read/close/mixed faults keep actual receipt/native401.ErrOriginal/value/status/causes, close once and stop willing callbacks; default missing policy cannot suppress physical404.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeletePhysicalFaultsCannotBecomeMissing`.
- Accepted-code404 and callback404 after real503 remain terminal, preserve real receipt and do not set IgnoredMissing even under default direct-leaf missing policy.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteAccepted404AndCallback404NeverEstablishMissing`.
- Clean503 retry and401 renewal keep one logical bodyless DELETE route/conditional-header/selected-version policy; renewed tokens work and actual final ACK proof is retained.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteCleanRetryAndAuthenticationKeepFixedBodylessPolicy`.

Python/Go differences:

- Public Go returns an owned partial QoSRuleDeleteResult with Completed/Deleted plus actual Extensions/Policy/Receipt instead of Python bool or exception only. Deleted=false is Source missing only when Completed=true and err=nil.
- New receipt-only Source leaf methods default ignore_missing true like Proxy; each cloud helper always requests strict false, then uses shared resource.IgnoreMissing terminal policy only on clean child404. Error paths never fabricate a policy None or DELETE Resource/Wire.
- Bandwidth/DSCP Source policy lookup ignore_missing=True plus explicit None NotFound differs from minimum-bandwidth Source ignore_missing=False. SDK reuses shared Policy vs PolicyStrict and preserves that difference rather than making all parents default tolerant.
- Source mutation status100..399 opt-in applies to these new profiles through root Engine.Mutate e.codes; old false/generated/typed policies remain unchanged. Opaque ACK content is not forced into JSON.
- Source dynamic Resource/class/session/adapter/custom Python IDs and mutable state remain SDK-R1/C1/S1 foundations. Go accepts safe concrete UTF8 strings/options/context, rejects blank/control/dot IDs and dangerous route/header changes; whitespace-only Source identifiers can reach HTTP while Go rejects them explicitly. No empty ID silently becomes missing false.
- Safe Header/Microversion options and callback ownership are Go transport conveniences; original cloud helper has no such kwargs. Library owns prerequisite/gate/strict child/missing behavior without requiring caller builders/interfaces.
- Fixed Neutron27 project-scoped delete_policy_bandwidth_limit_rule/delete_policy_dscp_marking_rule/delete_policy_minimum_bandwidth_rule use ADMIN_OR_PARENT_OWNER_MANAGER; manager must own parent policy, admin alternative, deprecated ADMIN_ONLY distinguished. Shared visibility/read does not imply delete rights, and deployment policy overrides/QoS driver/real deletion were not tested.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import os
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--kind", choices=("bandwidth", "dscp", "minimum"), required=True)
    parser.add_argument("--policy", required=True)
    parser.add_argument("--rule", required=True)
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    methods = {
        "bandwidth": conn.delete_qos_bandwidth_limit_rule,
        "dscp": conn.delete_qos_dscp_marking_rule,
        "minimum": conn.delete_qos_minimum_bandwidth_rule,
    }
    print(methods[args.kind](args.policy, args.rule))


if __name__ == "__main__":
    main()
```

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	kind := flag.String("kind", "", "bandwidth, dscp or minimum")
	policy := flag.String("policy", "", "QoS policy ID or name")
	rule := flag.String("rule", "", "literal QoS rule ID")
	flag.Parse()
	if *policy == "" || *rule == "" || *kind == "" {
		log.Fatal("--kind, --policy and --rule required")
	}
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var result *network.QoSRuleDeleteResult
	switch *kind {
	case "bandwidth":
		result, err = service.DeleteQoSBandwidthLimitRule(ctx, *policy, *rule)
	case "dscp":
		result, err = service.DeleteQoSDSCPMarkingRule(ctx, *policy, *rule)
	case "minimum":
		result, err = service.DeleteQoSMinimumBandwidthRule(ctx, *policy, *rule)
	default:
		log.Fatal("--kind must be bandwidth, dscp or minimum")
	}
	if err != nil {
		if result != nil && result.Receipt != nil {
			log.Printf("actual DELETE response: %d %s", result.Receipt.StatusCode, string(result.Receipt.Envelope))
		}
		log.Fatal(err)
	}
	output := map[string]any{"completed": result.Completed, "deleted": result.Deleted}
	if result.Receipt != nil {
		output["status"] = result.Receipt.StatusCode
		output["body"] = string(result.Receipt.Envelope)
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1723)
- [minimum-bandwidth Source](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2115)
- [Neutron27 QoS policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [base policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py)
