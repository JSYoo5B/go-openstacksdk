# Cloud QoS Rule Types Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/list_qos_rule_types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListQoSRuleTypeValues`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleTypeListFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleTypeListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithProxyListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordListBasePath`

- Every call reads the complete fresh qos alias inventory before kwargs/callbacks; missing, late HTTP failure or unhashable aliases stop the rule list and preserve actual observed Extension records. Clean HTTP403 preserves the native ErrUnexpectedResponseCode value with actual Method/URL/Body/ResponseHeader; accepted200 alias-projection faults retain ResponseError.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesFreshCompleteGateAndMissingShortCircuit`, `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesGateReadsPastFirstMatchAndKeepsActualFailure`.
- Source falsey JSON and binary64 underflow normalize to empty kwargs; truthy nonobjects fail after the extension gate. Original ordered JSON kwargs, header/version/path formals and one conflicting-attrs pass are library-owned.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesFalseyFiltersAndTruthyShapeAfterGate`, `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesOriginalKwargsOwnedHeadersVersionPathAndConflictMerge`, `network/v2/extensions/qos/ruletypes/python_proxy_list_test.go:TestQoSRuleTypeProxyOriginalExpressionWinsAfterOneConflictPass`.
- Generator JMESPath precedes list() conversion; literal arrays/objects/strings and nested generators remain polymorphic without fabricated RuleType or HTTP metadata; unused transport formals remain lazy.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesGeneratorJMESPathAndPolymorphicListConversion`, `network/v2/extensions/qos/ruletypes/python_proxy_list_test.go:TestQoSRuleTypeProxySourceFormalsStayLazyBehindConstantExpressions`, `network/v2/extensions/qos/ruletypes/python_proxy_list_test.go:TestQoSRuleTypeProxyGuardsObserveEveryOptionAndOpaqueIdentity`.
- Successful empty lists are nonnil; nullable aliases, drivers descriptors and unknown extensions retain separate owned Resource/Wire and actual page proof; late errors retain Inventory without success Values/RuleTypes/JSON. Late clean403 remains a native value error with the actual rejected page status/header/body/URL.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesEmptyDefaultsAliasesAndWireOwnership`, `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesLatePageFailureRetainsInventoryWithoutSuccessValues`.
- Every callback sees a clone and a before/after source check; cancellation, source changes, native401 physical Read/Close faults retain actual native original proof and stop faulty retry/reauth hooks.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesOptionGuardsStopRestoreAndCancellationCallbacks`, `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesPhysical401PreservesActualNativeFieldsAndNoHooks`, `network/v2/extensions/qos/ruletypes/python_proxy_list_test.go:TestQoSRuleTypeProxyUnknownValuesAndCallbackMapsAreOwned`.
- Existing typed raw reserved-key rejection and 200..399 admission stay unchanged; only the explicit typed BasePath and separate Proxy lanes broaden options.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesOldStatusAndTypedRawRejectionRemainUnchanged`, `network/v2/extensions/qos/ruletypes/python_proxy_list_test.go:TestQoSRuleTypeProxyOwnsRawFormalsAndTypedBasePathKeepsOldRawRejection`.

Python/Go differences:

- Go sorts object keys lexically when list() consumes a JMESPath object, while Python preserves insertion order; arrays and Unicode character ordering are preserved.
- Existing rule type leaf keeps 200..399 versus Source synthetic100..199, nonnegative Go-integer max_items, exact-decimal id/name local equality versus Python loaded float rounding/underflow, and explicit safe origin/path/header/UTF8/continuation policy. These are documented Go policies; they are not full runtime parity.
- Concrete owned JSON/client/context/options do not provide arbitrary Python class/session/operator replacement or mutable Resource/cloud configuration/cache lifecycle. Whole-project SDK-R1 remains separate; no live deployment support or whole SDK supported promotion is claimed.

### `python:cloud/_network/NetworkCloudMixin/get_qos_rule_type_details`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.API.GetRecord`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.

Python/Go differences:

- Stock Neutron27 policy get_rule_type is project reader, but QoSPlugin.get_rule_type adds an is_admin backend check. This cloud detail helper is queued for the core-admin phase, not implemented or counted as a user-cloud read.
- Pinned details ignores filters and performs two separate fresh full extension GET checks for qos then qos-rule-type-details; the existing low-level GetRecord is not the named cloud helper.

Unresolved boundaries:

- Administrator-phase cloud helper with two fresh ordered full extension checks and filters ignored is deliberately deferred; no user-cycle admin implementation.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import types
import openstack


def print_value(value):
    if hasattr(value, "to_dict"):
        value = value.to_dict()
    print(json.dumps(value, ensure_ascii=False, indent=2))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--filters", default="null")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    for value in conn.list_qos_rule_types(filters=json.loads(args.filters)):
        if isinstance(value, types.GeneratorType):
            for record in value:
                print_value(record)
        else:
            print_value(value)


if __name__ == "__main__":
    main()
```

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
)

func printJSON(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}

func main() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	filters := flag.String("filters", "null", "original Source cloud filters JSON")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, err := service.ListQoSRuleTypeValues(ctx, network.WithQoSRuleTypeListFilters(json.RawMessage(*filters)))
	if err != nil {
		log.Fatal(err)
	}
	if result.JSON != nil {
		var decoded any
		decoder := json.NewDecoder(bytes.NewReader(result.JSON))
		decoder.UseNumber()
		if err := decoder.Decode(&decoded); err != nil {
			log.Fatal(err)
		}
		printJSON(decoded)
		return
	}
	for _, value := range result.Values {
		if generator, ok := value.(*ruletypes.ProxyListGenerator); ok {
			for record, err := range generator.Records {
				if err != nil {
					log.Fatal(err)
				}
				printJSON(record.Resource)
			}
		} else {
			printJSON(value)
		}
	}
}
```

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [QoS Plugin](https://github.com/openstack/neutron/blob/27.0.0/neutron/services/qos/qos_plugin.py)
- [RuleTypes record reader](v2/qos-rule-types-python.md)
- [Install](../docs/install.md)
- [ Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L281-L332)
- [network proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5270)
- [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
