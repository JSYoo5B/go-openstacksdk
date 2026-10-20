# Cloud QoS Bandwidth Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/list_qos_bandwidth_limit_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListQoSBandwidthLimitRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyBandwidthLimitRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitProxyQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListBasePath`

- Complete QoS extension gate then exactly one parent policy resolution, owned untyped defaults/unknown fields/computed-null location, bound URI parent and local Body filtering with shared actual paging.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthListOwnsGateParentPagingAndRawFields`.
- Extension gate consumes every alias including later unhashable alias or failed page after an early QoS match; absence has typed unavailable extension error and no policy/callback/rule request.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`.
- Policy missing and ambiguity prevent rules/callback/filter expansion, preserve real extension inventory and typed missing/ambiguity without inventing policy None HTTP evidence.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPolicyMissingAndAmbiguityRetainActualGate`.
- Source proxy generator JMESPath literals/Unicode/object keys/nested generator/null/type errors distinguish rule HTTP from completed prerequisite HTTP; owned base_path succeeds while existing typed raw base_path rejection stays unchanged.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`.
- CPython falsey filters, truthy nonobject pre-rule errors, initialized owned callback maps, joined option error/cancellation or original captured-source mutation and real prerequisite receipts.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthFilterOwnershipCallbackGuardAndFalseyCases`.
- Actual truncated401 rule response keeps native reauthentication pointer/UnexpectedResponseCode/body/header/EOF, stops willing retry/reauth callbacks; late failed page preserves only earlier actual Inventory and completed parent/extensions.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`.

Python/Go differences:

- Concrete JSON/context/captured-client Go APIs and protected route/header policy replace dynamic Python session/Resource class/constructor/callable substitutions; arbitrary Python-only objects are not accepted.
- SourceProxy lane accepts100..399 with Source loaded local equality and computed location:null. Existing typed list/Get record status200..399 and typed raw base_path rejection remain compatibility APIs; concrete typed BasePath is additive.
- JMESPath literal object key iteration uses Go lexical order vs Python insertion order; existing strict syntax/exact JSON numeric evaluator/custom Resource function type policies remain explicit differences.
- Literal and nested generator values never acquire fake BandwidthLimitRecord or HTTP receipts. Actual extension/policy/rule proofs and observed partial inventory are retained with Go errors.
- Standalone Service computed location is Source null, including ignoring raw location in logical view. Full Connection computed location and mutable Resource/session model are tracked shared SDK-C1/R1 scope.
- Source policy and extension/user authorization are checked by the real Neutron deployment; these are read-only named helpers and no creation/update/delete or server-side policy emulation is added.

### `python:cloud/_network/NetworkCloudMixin/search_qos_bandwidth_limit_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchQoSBandwidthLimitRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyBandwidthLimitRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitProxyQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListBasePath`

- Complete QoS extension gate then exactly one parent policy resolution, owned untyped defaults/unknown fields/computed-null location, bound URI parent and local Body filtering with shared actual paging.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthListOwnsGateParentPagingAndRawFields`.
- Search delegates exactly one complete list workflow, preserves identifier/glob selection and CPython loaded bool/integer local equality, applies original ordered dictionary filter after list and retains actual Inventory/proof on unknown-key failure.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSearchKeepsRuleIDPhaseAndFilterPresence`.
- Extension gate consumes every alias including later unhashable alias or failed page after an early QoS match; absence has typed unavailable extension error and no policy/callback/rule request.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`.
- Policy missing and ambiguity prevent rules/callback/filter expansion, preserve real extension inventory and typed missing/ambiguity without inventing policy None HTTP evidence.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPolicyMissingAndAmbiguityRetainActualGate`.
- Source proxy generator JMESPath literals/Unicode/object keys/nested generator/null/type errors distinguish rule HTTP from completed prerequisite HTTP; owned base_path succeeds while existing typed raw base_path rejection stays unchanged.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`.
- CPython falsey filters, truthy nonobject pre-rule errors, initialized owned callback maps, joined option error/cancellation or original captured-source mutation and real prerequisite receipts.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthFilterOwnershipCallbackGuardAndFalseyCases`.
- Actual truncated401 rule response keeps native reauthentication pointer/UnexpectedResponseCode/body/header/EOF, stops willing retry/reauth callbacks; late failed page preserves only earlier actual Inventory and completed parent/extensions.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`.
- Source search without identifier skips falsey literal values; nested generator membership consumes actual rule records before unknown-key failure, while identifier phase rejects opaque generator without rule HTTP.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSearchSourceFalseyLiteralsAndNestedMembership`.

Python/Go differences:

- Concrete JSON/context/captured-client Go APIs and protected route/header policy replace dynamic Python session/Resource class/constructor/callable substitutions; arbitrary Python-only objects are not accepted.
- SourceProxy lane accepts100..399 with Source loaded local equality and computed location:null. Existing typed list/Get record status200..399 and typed raw base_path rejection remain compatibility APIs; concrete typed BasePath is additive.
- JMESPath literal object key iteration uses Go lexical order vs Python insertion order; existing strict syntax/exact JSON numeric evaluator/custom Resource function type policies remain explicit differences.
- Literal and nested generator values never acquire fake BandwidthLimitRecord or HTTP receipts. Actual extension/policy/rule proofs and observed partial inventory are retained with Go errors.
- Standalone Service computed location is Source null, including ignoring raw location in logical view. Full Connection computed location and mutable Resource/session model are tracked shared SDK-C1/R1 scope.
- Source policy and extension/user authorization are checked by the real Neutron deployment; these are read-only named helpers and no creation/update/delete or server-side policy emulation is added.

### `python:cloud/_network/NetworkCloudMixin/get_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetQoSBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitMicroversion`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Literal rule getter gates and resolves policy once, returns raw untyped record/default location on success, and preserves native404 without rule Find fallback or ignored missing.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGetResolvesPolicyOnceAndKeeps404`.
- Extension gate consumes every alias including later unhashable alias or failed page after an early QoS match; absence has typed unavailable extension error and no policy/callback/rule request.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`.
- Policy missing and ambiguity prevent rules/callback/filter expansion, preserve real extension inventory and typed missing/ambiguity without inventing policy None HTTP evidence.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPolicyMissingAndAmbiguityRetainActualGate`.
- CPython falsey filters, truthy nonobject pre-rule errors, initialized owned callback maps, joined option error/cancellation or original captured-source mutation and real prerequisite receipts.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthFilterOwnershipCallbackGuardAndFalseyCases`.
- Actual truncated401 rule response keeps native reauthentication pointer/UnexpectedResponseCode/body/header/EOF, stops willing retry/reauth callbacks; late failed page preserves only earlier actual Inventory and completed parent/extensions.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`.

Python/Go differences:

- Concrete JSON/context/captured-client Go APIs and protected route/header policy replace dynamic Python session/Resource class/constructor/callable substitutions; arbitrary Python-only objects are not accepted.
- SourceProxy lane accepts100..399 with Source loaded local equality and computed location:null. Existing typed list/Get record status200..399 and typed raw base_path rejection remain compatibility APIs; concrete typed BasePath is additive.
- JMESPath literal object key iteration uses Go lexical order vs Python insertion order; existing strict syntax/exact JSON numeric evaluator/custom Resource function type policies remain explicit differences.
- Literal and nested generator values never acquire fake BandwidthLimitRecord or HTTP receipts. Actual extension/policy/rule proofs and observed partial inventory are retained with Go errors.
- Standalone Service computed location is Source null, including ignoring raw location in logical view. Full Connection computed location and mutable Resource/session model are tracked shared SDK-C1/R1 scope.
- Source policy and extension/user authorization are checked by the real Neutron deployment; these are read-only named helpers and no creation/update/delete or server-side policy emulation is added.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--policy", required=True)
    parser.add_argument("--rule", default="")
    parser.add_argument("--mode", choices=("list", "search", "get"), default="list")
    parser.add_argument("--filters", default="null")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    filters = json.loads(args.filters)
    if args.mode == "list":
        values = conn.list_qos_bandwidth_limit_rules(args.policy, filters)
    elif args.mode == "search":
        values = conn.search_qos_bandwidth_limit_rules(args.policy, args.rule or None, filters)
    else:
        if not args.rule:
            parser.error("--rule required for get")
        values = [conn.get_qos_bandwidth_limit_rule(args.policy, args.rule)]
    for value in values:
        print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value, ensure_ascii=False, indent=2))


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
	policy := flag.String("policy", "", "QoS policy ID or name")
	rule := flag.String("rule", "", "rule ID or search pattern")
	mode := flag.String("mode", "list", "list, search or get")
	filters := flag.String("filters", "null", "Source cloud filters JSON")
	flag.Parse()
	if *policy == "" {
		log.Fatal("--policy required")
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
	var result *network.QoSBandwidthLimitResult
	switch *mode {
	case "list":
		result, err = service.ListQoSBandwidthLimitRules(ctx, *policy, network.WithQoSBandwidthLimitFilters(json.RawMessage(*filters)))
	case "search":
		result, err = service.SearchQoSBandwidthLimitRules(ctx, *policy, *rule, network.WithQoSBandwidthLimitFilters(json.RawMessage(*filters)))
	case "get":
		if *rule == "" {
			log.Fatal("--rule required for get")
		}
		result, err = service.GetQoSBandwidthLimitRule(ctx, *policy, *rule)
	default:
		log.Fatal("--mode must be list, search or get")
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.Value == nil {
		log.Fatal("opaque result: consume QoSBandwidthLimitGenerator.Records explicitly")
	}
	output, err := json.MarshalIndent(result.Value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(output))
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe` cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1535)
- [bandwidth-limit record API](v2/qos-bandwidth-limit-records-python.md)
