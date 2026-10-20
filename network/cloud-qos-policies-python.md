# Cloud QoS Policies Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/list_qos_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListQoSPolicyRecords`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithOperationGuard`

- Every call collects the complete uncached Source extension alias inventory before policy reads, including later pages after an early qos match. Missing qos returns ErrUnavailableExtension with completed extension evidence; late gate permission/malformed failures remain terminal rather than cached/early-success.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyListAndSearchGateEveryCallWithoutEarlyExit`.
- Cloud filters remain owned JSON snapshots and all loaded-Python falsey values become an empty kwargs mapping after the extension gate. Truthy list nonmapping filters fail after the gate before policy HTTP. No _filter_list, glob, local ID/name search or post-server name predicate is invented.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyFiltersFalseyAndMappingErrorsHappenAfterGate`, `network/cloud_qos_policies_test.go:TestCloudQoSPolicySearchLiteralNameAndOrderedUpdateAreNotLocalSearch`.
- Existing QoSPolicy declared server aliases and tenant_id/rules local filters are owned by the library. Source Proxy lane consumes raw headers/microversion/service-relative base_path, ordered conflicting attrs and original JMES separately; raw cap precedes local filtering and malformed/bound controls retain pre-policy failures.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyRawFormalsAliasesAndLocalJSONAreOwned`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyListRawFormalsGeneratorAndOpaqueJSON`.
- Actual Source cloud list(Proxy) consumes an opaque generator or iterates arbitrary array/string/dictionary result, with null/number/type errors occurring after extension discovery without policy HTTP. Nested generators preserve identity/zero policy HTTP and JSON remains absent. Dictionary keys have the explicitly reviewed Go lexical order; [z,a] Source is [a,z] Go, never claimed to match Source insertion order.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyProxyIterableAndOpaqueResultsPreserveHTTPPhases`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyListRawFormalsGeneratorAndOpaqueJSON`.
- New Source Proxy profile opts into root-link precedence, CPython-loaded equality and physical fault stop without changing typed List/Get baseline. BasePath pointers/source/options are cloned and cloud operation guard remains active through leaf callbacks/retry/physical handling. Concurrent reusable options, late failures and physical read/Close/native401 faults preserve completed extension/actual policy evidence and discard partial Policies/Value/JSON; clean503 retries remain allowed and changed selected route retains original rejected503.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyReadOptionsSnapshotsCallbacksAndLateFailures`, `network/cloud_qos_policies_test.go:TestCloudQoSPolicyPhysicalFaultsAndCleanRetryKeepActualReceipt`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxySourceLinksLoadedEqualityAndGuardAreOptIn`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyOptsSnapshotsFormalErrorsAndTypedBasePath`.

Python/Go differences:

- The concrete Service helper returns a QoSPolicyRecords carrier: ordinary PolicyRecord Resource views plus independent actual Wire/Envelope/Header/status, completed Extensions evidence and arbitrary list Values/JSON. A failed operation keeps completed extension evidence and no partial policy values instead of returning a Python exception with no result.
- Go uses a concrete string search identity and JSON-domain filters/options. Mutable Python Resource/class/session/adapter/cache/Connection-derived location and custom objects remain shared SDK foundations; the current owned location is null. Options are library-owned snapshots, and no caller filter builder is needed.
- Source cloud list(JMES dictionary) iterates insertion-order keys; the existing shared Go JMES map result is iterated in deterministic lexical order. The explicit z,a literal yields z,a in Source and a,z in Go. Arrays, Unicode strings, opaque generators and noniterable/type failures retain their actual phases.
- Safe string Header maps, same-origin service-relative fixed paths/query/cycle guards, nonnegative Go raw caps and strict complete UTF8/JSON are explicit Go policies. Some lazy Source descriptor/Resource.list formal errors are validated eagerly in owned options. Unknown actual body and large integers remain raw.
- Only the new Proxy/cloud lane uses Source link precedence/loaded equality/physical-fault policy. Existing typed Policies List/Get query reservations, dictionary/HTTP Link interoperability and numeric equality are preserved. Source Proxy receives the unconsumed generator, never an invented materialized resource array.
- Fixed Neutron27 get_policy permits project reader for owned policies or shared policy (and admin) in project scope. Source/fake HTTP/race/main checks do not assert authenticated deployment success, enabled QoS drivers or elevated write/admin permissions.

### `python:cloud/_network/NetworkCloudMixin/search_qos_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchQoSPolicyRecords`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithOperationGuard`

- Every call collects the complete uncached Source extension alias inventory before policy reads, including later pages after an early qos match. Missing qos returns ErrUnavailableExtension with completed extension evidence; late gate permission/malformed failures remain terminal rather than cached/early-success.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyListAndSearchGateEveryCallWithoutEarlyExit`.
- Cloud filters remain owned JSON snapshots and all loaded-Python falsey values become an empty kwargs mapping after the extension gate. Truthy list nonmapping filters fail after the gate before policy HTTP. No _filter_list, glob, local ID/name search or post-server name predicate is invented.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyFiltersFalseyAndMappingErrorsHappenAfterGate`, `network/cloud_qos_policies_test.go:TestCloudQoSPolicySearchLiteralNameAndOrderedUpdateAreNotLocalSearch`.
- Search seeds a truthy literal identity as server name then applies Source dict.update in insertion order. Mapping values replace the original slot, including name null/override; JSON iterable pairs include Unicode string and dictionary-key pairs, final kwargs keys must be strings and invalid pair shapes fail after completed gate before policy HTTP.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicySearchLiteralNameAndOrderedUpdateAreNotLocalSearch`.
- Existing QoSPolicy declared server aliases and tenant_id/rules local filters are owned by the library. Source Proxy lane consumes raw headers/microversion/service-relative base_path, ordered conflicting attrs and original JMES separately; raw cap precedes local filtering and malformed/bound controls retain pre-policy failures.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyRawFormalsAliasesAndLocalJSONAreOwned`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyListRawFormalsGeneratorAndOpaqueJSON`.
- Actual Source cloud list(Proxy) consumes an opaque generator or iterates arbitrary array/string/dictionary result, with null/number/type errors occurring after extension discovery without policy HTTP. Nested generators preserve identity/zero policy HTTP and JSON remains absent. Dictionary keys have the explicitly reviewed Go lexical order; [z,a] Source is [a,z] Go, never claimed to match Source insertion order.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyProxyIterableAndOpaqueResultsPreserveHTTPPhases`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyListRawFormalsGeneratorAndOpaqueJSON`.
- New Source Proxy profile opts into root-link precedence, CPython-loaded equality and physical fault stop without changing typed List/Get baseline. BasePath pointers/source/options are cloned and cloud operation guard remains active through leaf callbacks/retry/physical handling. Concurrent reusable options, late failures and physical read/Close/native401 faults preserve completed extension/actual policy evidence and discard partial Policies/Value/JSON; clean503 retries remain allowed and changed selected route retains original rejected503.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyReadOptionsSnapshotsCallbacksAndLateFailures`, `network/cloud_qos_policies_test.go:TestCloudQoSPolicyPhysicalFaultsAndCleanRetryKeepActualReceipt`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxySourceLinksLoadedEqualityAndGuardAreOptIn`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyOptsSnapshotsFormalErrorsAndTypedBasePath`.

Python/Go differences:

- The concrete Service helper returns a QoSPolicyRecords carrier: ordinary PolicyRecord Resource views plus independent actual Wire/Envelope/Header/status, completed Extensions evidence and arbitrary list Values/JSON. A failed operation keeps completed extension evidence and no partial policy values instead of returning a Python exception with no result.
- Go uses a concrete string search identity and JSON-domain filters/options. Mutable Python Resource/class/session/adapter/cache/Connection-derived location and custom objects remain shared SDK foundations; the current owned location is null. Options are library-owned snapshots, and no caller filter builder is needed.
- Source cloud list(JMES dictionary) iterates insertion-order keys; the existing shared Go JMES map result is iterated in deterministic lexical order. The explicit z,a literal yields z,a in Source and a,z in Go. Arrays, Unicode strings, opaque generators and noniterable/type failures retain their actual phases.
- Safe string Header maps, same-origin service-relative fixed paths/query/cycle guards, nonnegative Go raw caps and strict complete UTF8/JSON are explicit Go policies. Some lazy Source descriptor/Resource.list formal errors are validated eagerly in owned options. Unknown actual body and large integers remain raw.
- Only the new Proxy/cloud lane uses Source link precedence/loaded equality/physical-fault policy. Existing typed Policies List/Get query reservations, dictionary/HTTP Link interoperability and numeric equality are preserved. Source Proxy receives the unconsumed generator, never an invented materialized resource array.
- Fixed Neutron27 get_policy permits project reader for owned policies or shared policy (and admin) in project scope. Source/fake HTTP/race/main checks do not assert authenticated deployment success, enabled QoS drivers or elevated write/admin permissions.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("list", "search"), default="list")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="null")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    filters = json.loads(args.filters)
    if args.mode == "search":
        values = conn.search_qos_policies(name_or_id=args.name, filters=filters)
    else:
        values = conn.list_qos_policies(filters=filters)
    for value in values:
        if hasattr(value, "to_dict"):
            value = value.to_dict()
        print(json.dumps(value, indent=2))
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
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "list", "list or search")
    name := flag.String("name", "", "literal server name query")
    filters := flag.String("filters", "null", "JSON filters")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    options := []network.QoSPolicyReadOption{
        network.WithQoSPolicyFilters(json.RawMessage(*filters)),
    }
    var result *network.QoSPolicyRecords
    switch *mode {
    case "list":
        result, err = service.ListQoSPolicyRecords(ctx, options...)
    case "search":
        result, err = service.SearchQoSPolicyRecords(ctx, *name, options...)
    default:
        log.Fatal("mode must be list or search")
    }
    if err != nil {
        if result != nil {
            log.Printf("completed extension records: %d", len(result.Extensions))
        }
        log.Fatal(err)
    }
    if result.JSON == nil {
        fmt.Printf("opaque iterable result: %T\n", result.Value)
        return
    }
    var readable any
    decoder := json.NewDecoder(bytes.NewReader(result.JSON))
    decoder.UseNumber()
    if err := decoder.Decode(&readable); err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(readable, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L42-L71)
- [Install](../docs/install.md)
- [Cloud list/search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L253-L348)
- [extension gate](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L43-L50)
- [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866-L916)
- [QoSPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py)
