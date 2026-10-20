# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/server_actions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions.API.InServer`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPaginated`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions.ActionScope.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions.ActionScope.ProxyListRecords`

- The fixed-server action history retains requestID, server identity, summary/event extensions and raw response evidence while controlling linked-page consumption.
  Tests: `api/native_scope_list_controls_test.go:TestNativeScopeListControlsFiveBindingsRawCapBeforeNameAndHostFilters`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsSinglePageSkipsForeignAndCyclicContinuation`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsCrossPageQueryFixedParentAndIteratorReuse`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsZeroDefaultsFollowNativeContinuation`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsKeepWholePageDecodeErrors`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsParentCancellationAndConsumerBreak`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsPreflightNegativeAndTroveQueryPolicy`.
- Concrete Source-owned compute-role projection, library-owned callback snapshots and selected/versionless policy retain original client and actual response evidence. Source-aware lookup/list parsing shares guarded REST, descriptor MRO, Python-loaded equality and paging rather than new transport wrappers.
  Tests: `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsSourceMROEventsProjectionAndActualWire`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsEventsNullScalarAndOrderedAliases`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsQueryAliasesLocalFiltersRawCapAndParentScope`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsDiscoveryIsLazyAndCeilingOwnsEveryPage`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsOpaqueJMESUsesActualGeneratorWithZeroHTTP`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsCancelAfterYieldDoesNotReadNextPage`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsOptionsCannotReplaceParentAndKeepNilMapOwnership`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsPhysicalFailuresPreserveActualPageAndStopHooks`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsClean401And503RetriesKeepScopeQueryAndVersion`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsEventResourceIdentityDoesNotEqualPlainJSONDictionary`.
- Server URI is explicit and independent of passive instance_uuid/request_id; Source nine Body fields and nested list_type Resource event defaults/hostId aliases are preserved. Missing/null/scalar/nonobject distinctions, Source Resource-vs-plainJSONdict array equality and raw row cap/local filters are audited. Deferred max2.84 discovery/selected-literal version owns every page; opaque Proxy JMESPath uses generator identity and performs zero HTTP until consumption.
  Tests: `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsSourceMROEventsProjectionAndActualWire`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsEventsNullScalarAndOrderedAliases`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsQueryAliasesLocalFiltersRawCapAndParentScope`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsDiscoveryIsLazyAndCeilingOwnsEveryPage`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsOpaqueJMESUsesActualGeneratorWithZeroHTTP`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsCancelAfterYieldDoesNotReadNextPage`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsOptionsCannotReplaceParentAndKeepNilMapOwnership`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsPhysicalFailuresPreserveActualPageAndStopHooks`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsClean401And503RetriesKeepScopeQueryAndVersion`, `compute/v2/instanceactions/python_records_contracts_test.go:TestActionRecordsEventResourceIdentityDoesNotEqualPlainJSONDictionary`.

Python/Go differences:

- Go exposes these local controls through the common Collection or fixed-parent scope; existing native typed List option signatures remain unchanged. Zero means unlimited and negative values fail lazily before HTTP.
- The native path does not infer a wire limit from max_items and stops exactly at the raw row cap. Native whole-page extraction or IsEmpty may still reject unused trailing rows; continuation retains the native pager and SDK cycle policy.
- Go concrete Source records separate Resource default/projection from complete actual Wire/page/header/status; SDK-owned location is null and optional HTTP cache/Connection/dirty-state remain generic SDK-R1/C1. Actual cloud permissions/acceptance are not established by HTTP fixtures.
- JSON filters cannot construct Python Resource operands; original object event items are Source Resource objects and do not equal plain JSON dicts, while nonobject items become plain{} and retain loaded equality. Typed Source Python object lifecycle remains SDK-R1.
- Opaque generator Value and serializable JSON are separate Go result carriers, never eager collected list masquerading as Source Proxy._list. Go scoped ListRecords and InServer parent resolver are additional conveniences; fixed route/cursor/source/auth guard strengthens Source arbitrary session/origin behavior.
- Official Nova31.0.0 list policy is project reader-or-admin against instance project. Base list summary does not contain events; admin traceback/host/details visibility is detail controller policy and SDK performs no detail/admin fetch. Changes-since/limit/marker begin2.58, changes-before2.66; server remains the version/schema authority.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--server", required=True)
parser.add_argument("--changes-since", default=None)
args = parser.parse_args()
query = {} if args.changes_since is None else {"changes_since": args.changes_since}
with openstack.connect(cloud=args.cloud) as conn:
    for action in conn.compute.server_actions(args.server, **query):
        print(json.dumps(action.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions"
)

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    server := flag.String("server", "", "literal server UUID")
    since := flag.String("changes-since", "", "optional ISO timestamp")
    flag.Parse()
    if *server == "" { log.Fatal("-server is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    compute, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    var options []instanceactions.RecordListOption
    if *since != "" {
        options = append(options, instanceactions.WithRecordListQuery("changes_since", *since))
    }
    for value, err := range compute.InstanceActions.ListRecords(ctx, *server, options...) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```

## References

- [server_actions](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L3221)
- [ServerAction·ServerActionEvent](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_action.py)
- [Install](../../../docs/install.md)
- [instance action policy](https://github.com/openstack/nova/blob/31.0.0/nova/policies/instance_actions.py)
- [controller](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/instance_actions.py)
- [schema](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/schemas/instance_actions.py)
