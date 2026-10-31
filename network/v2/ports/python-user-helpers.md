# Python User Helpers

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/get_subnet_ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.GetSubnetPorts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.SubnetPortID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.SubnetPortValue`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.SubnetPortRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.SubnetPortNative`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.SubnetPortFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSubnetPortOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSubnetPortHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSubnetPortMicroversion`

- All sends one unfiltered GET ports and the caller keeps the ports whose FixedIPs contain the subnet ID, matching Python's local selection.
  Tests: `network/v2/ports/python_parity_core_test.go:TestPythonParityCorePortSubnetPortsSelection`.
- Source Subnet._get_id accepts literal or owned/native Resource ID; SDK snapshots selection and scans ports without subnet lookup or server filter.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsLiteralAndOwnedTargetsScanWithoutSubnetLookup`.
- Unfiltered shared pagination, repeated fixed-IP matches append the identical record pointer without deduplication, and every response is actual remote evidence.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsDuplicateMatchesKeepIdenticalPointerAndPagination`.
- CPython loaded JSON bool/int/float and nested equality apply to arbitrary explicit JSON identifiers including null.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsRawIdentifierUsesSourceLoadedEquality`.
- Concrete target/option factories own maps/raw ID/header/version values; nil/invalid targets and nil options fail before HTTP.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsTargetsAndOptionsAreOwnedAndGuarded`.
- Malformed null fixed_ips, missing subnet_id and wrong fixed-IP shapes preserve actual page receipt plus earlier selected/observed records.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsMalformedFixedIPsRetainActualPageAndObservedRows`.
- Physical 401 mixed read/close preserves actual native pointer and causes, one close, no reauthentication/retry selection callbacks.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsPhysical401NeverRetriesSelection`.
- Simultaneous callback failure and cancellation or captured source mutation are joined before any HTTP.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsCallbackCancellationAndSourceMutationPreserveCauses`.

Python/Go differences:

- Go has no single helper; the caller applies the fixed_ips subnet_id test to the rows from All.
- Python raises KeyError for a fixed IP without subnet_id; Go treats it as an empty ID.
- The native ports.Port model lacks extension fields that Python Port Resources expose.
- Concrete JSON/context/captured-client API; arbitrary Python Resource/session/class/callable substitutions are not accepted. Shared cloud configuration disabled-service semantics and mutable Resource R1 contracts remain separate tracked scope.
- Public Gophercloud testhelper/nativefixture reuse and actual response errors preserve physical read/close causes; opaque or literal local JSON never receives invented resource/HTTP evidence.
- Captured route/header/version protection, Go error returns and retained observed inventory differ intentionally from dynamic Python exception-only return conventions.
- Go SubnetPortTarget is a concrete selector; SubnetPortValue covers non-Resource JSON input and SubnetPortFields deliberately reads only an owned input ID without inventing a response.
- Go returns partial observed Inventory/Ports with an error; Source throws without returning its partial result. No deduplication or subnet-name inference is introduced.
- Source renamed subnet_id keyword compatibility is represented by the explicit SubnetPortID constructor rather than accepting dynamic keyword aliases.

### `python:network/v2/create_ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.CreatePorts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.BulkPort`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.BulkPortJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.BulkPortFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithBulkPortAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithBulkPortOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithBulkPortHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithBulkPortMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.BulkPortResult.All`

- Exactly one eager flat POST {ports:[attrs1,attrs2]} owns canonical/wire alias order, raw request descriptor values, unknown Body names and consumes request microversion; collection iterator never resends.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsSingleEagerFlatPostAndRawAliases`.
- All row headers validate, only final row header applies, Source If-Match scalar/list rendering and selected request version win without mutating native client.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsLastRowHeaderAndSelectedVersion`.
- Every malformed/empty input, request constructor collision and earlier invalid row header fails before POST; [empty object] is accepted.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsPreflightRejectsEveryInputBeforePost`.
- Owned input/options/header snapshots and callback maps isolate mutations; public result receipt mutation does not alter retained lazy projection proof.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsInputsAndOptionCallbacksAreOwned`.
- Returned response rows are remote-only with Source defaults/coercions/unknown fields and raw Wire; row count need not match input and HTTP If-Match does not seed logical record.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsRemoteViewsHaveDefaultsAndNoHttpHeaderSeed`.
- 204/non-JSON/missing/null/noniterable ports fail eagerly with actual completed POST receipt; Source empty arrays/objects/strings remain successful empty iterables.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsEagerEnvelopeAndIterableFailuresKeepActualPost`.
- Malformed response rows and cls/self/connection/_synchronized/microversion constructor collisions fail lazily after earlier valid rows, retaining actual POST receipt; nonempty object/string ports also fail lazily.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsResponseRowsFailLazilyAfterEarlierActualRows`.
- Before-POST callback errors join cancellation/source mutation; after-POST captured source mutation rejects lazy consumption with actual completed POST proof.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsCancellationAndSourceMutationRetainCauses`.
- Actual success/rejection physical read/close/mixed failures retain body/headers/status/native401 pointer and causes, one close and zero willing retry/reauth callbacks.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsPhysicalFailuresPreserveNativeEvidenceAndStopCallbacks`.
- Clean 401 renewal and clean 503 retry preserve identical bulk body, route, selected version and conditional header; callbacks cannot expand accepted statuses; no per-row POST.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsCleanReauthenticationAndRetryKeepOneBulkBodyPolicy`.

Python/Go differences:

- Concrete JSON/context/captured-client API; arbitrary Python Resource/session/class/callable substitutions are not accepted. Shared cloud configuration disabled-service semantics and mutable Resource R1 contracts remain separate tracked scope.
- Public Gophercloud testhelper/nativefixture reuse and actual response errors preserve physical read/close causes; opaque or literal local JSON never receives invented resource/HTTP evidence.
- Captured route/header/version protection, Go error returns and retained observed inventory differ intentionally from dynamic Python exception-only return conventions.
- Factory/raw JSON keep explicit Source attribute order; public Go Attributes maps use lexical distinct-key order. Alias collision callers can use ordered factories or raw object.
- Go iterator can repeatedly project retained completed response without sending another request; Source response generator is single-use. All discards partial slice on projection error; manual iteration can retain earlier rows.
- Bulk explicitly opts into SDK-owned SourceStatusPolicy accepting valid HTTP100..399, matching Source raise_from_response below400; reviewed older list/find profiles retain200..399. Physical >=400 never becomes logical success; callback OkCodes expansion cannot alter the selected policy.
- Bulk atomicy and per-row/attribute user/admin policy remain server/plugin responsibilities; official Neutron27 Pecan validation/policy and ML2 native bulk transaction source were reviewed, no live deployment/rollback was tested.

### `python:cloud/_network/NetworkCloudMixin/list_ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ListPorts`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ListPortValues`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ListSourceValues`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListFilters`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListLocation`

- Default bodyless /ports GET and Accept header, Source binding/DNS/security/QoS/unknown fields, missing defaults and actual page evidence.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsDefaultRequestAndExtensionFields`.
- Canonical and wire server aliases, repeated Requests values, unknown kwargs discarded before JSON encoding, nested local declared Body filters.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsQueryAliasesAndLocalBodyFilters`.
- Plural/HTTP link continuation, short-page limit/marker fallback, paginated false and raw row cap.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPagingLinksAndLimitMarker`.
- Accepted representations and rejected statuses retain ports operation context; nil option performs no HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsStatusesAndOperationContext`.
- Cloud search uses dict insertion-order pushdown, exact/glob ID-name matching, nested filters and arbitrary full JMESPath; unknown filter fails after real inventory.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSelectionAndPushdown`.
- Explicit version and headers apply to every actual list page.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsHeadersAndMicroversionOnEveryPage`.
- Partial field queries do not seed responses; Source constructor alias order/null/boolean/int/list/dict projection and independent actual Wire/Envelope/Header ownership.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPartialFieldsCoercionAndEvidenceIsolation`.
- Owned runtime controls and malformed UTF-8/JSON fail before HTTP; unknown kwargs discarded without running a JSON Marshaler.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsControlsDiscardAndValidationBeforeEncoding`.
- Option factories/callback maps own input snapshots and nil Headers are initialized for direct custom assignments.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsOptionsOwnMapsAndNilHeaders`.
- Option failure retains simultaneous selected-client mutation or cancellation cause, and no HTTP escapes the boundary.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListAndSearchRecordOptionsGuardCancellationAndSource`.
- Raw cap counts filtered rows, lazy consumer break and single-page stop avoid next page requests.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsCapCountsFilteredRowsAndLazyConsumer`.
- CPython float underflow next is falsey; malformed envelope/row/constructor controls and invalid next retain actual accepted page receipt.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsContinuationTruthinessAndTerminalEvidence`.
- 18 accepted/rejected status plus Read/Close/mixed faults close once, invoke no retry/reauth hook, preserve actual receipt and native401 pointer ErrOriginal/ErrReauth fields.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks`.
- Clean401 token renewal and clean503 retry retain selected headers/version/bodyless route/query and leave original client configuration intact.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsCleanReauthenticationAndRetryKeepPolicy`.
- Eager explicit expression supports scalar/array/object/null values and preserves all original Inventory on expression failure.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordValuesArbitraryJSONAndInventory`.
- Cloud falsey filters including underflow act as absent; constructor headers snapshot input and unknown dict keys retain actual inventory on local failure.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsFalseyFiltersHeaderSnapshotAndUnknownKeys`.
- Original generator projections/index/type return null, literals/identity equality return values, typed functions/serialization/syntax fail with zero HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortProxyListRecordsSourceGeneratorExpressionsDoNoHTTP`.
- Source missing/falsey/nonstring controls retain opaque generator; @/not_null/to_array preserve same lazy generator, HTTP starts only on iteration.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortProxyListRecordsPreserveReturnedAndNestedGenerators`.
- Source formal header/version/base_path normalization and ordered single-pass conflicting attrs restoration; restored formal duplicates fail before HTTP and restored JMES cannot replace the original expression.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsNormalizeSourceFormalsAndConflictingAttrs`.
- Source unknown non-Body control names and Header kwargs are discarded before JSON marshaling without introducing new query or transport behavior.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsSourceUnknownControlNamesDiscardBeforeMarshaler`.
- Typed/raw/snapshot base_path selects service-relative collection path for every page; unsafe foreign host/query/fragment/traversal fails before HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsBasePathOwnsRouteAndLocksContinuation`.
- Source-profile local filters preserve CPython float rounding, arbitrary integer precision, nested boolean/number equality, underflow and overflow comparison.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsLocalEqualityUsesSourceLoadedFloats`.
- Cloud dict filters preserve list_ports polymorphic Proxy result: null/type errors perform no HTTP, @ retains actual inventory and empty literal containers return no fabricated receipt.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsHonorsProxyJMESBeforeCloudLocalSelection`.
- Raw Source formals and dedicated concrete factories use last option consistently for selected path, headers and microversion.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsFormalFactoriesAndTypedOptionsUseLastValue`.
- Actual Source canonical cloud dict lookup succeeds, wire Body keys fail after real inventory, raw Connection location is snapshotted and applied before selection without modifying Wire.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSourceWireFiltersAndConnectionLocationBeforeSelection`.
- Cloud second-phase SelectLoaded preserves Source numeric matches from first-phase list filtering rather than reversing rounded float/integer or nested underflow comparisons.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSecondFilterKeepsSourceLoadedNumericMatch`.
- Configured CloudLocation is cloned and computed per actual resource before filtering; foreign project clears scope names, keeps original project/zone, and never fabricates Wire location.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsCloudLocationOwnsForeignProjectAndZone`.
- Connection uses cached Neutron service; owned per-row CurrentLocation is applied before local selection without changing actual Wire, and callbacks execute once.
  Tests: `connectiontest/connection_network_record_lists_test.go:TestConnectionPortRecordsCachedServiceAndAutomaticPerRowLocation`.
- Clean absent catalog returns nonnil empty list before list options; search still evaluates loaded local filters and expressions on [], including scalar/null and invalid expressions, with zero HTTP. Mixed catalog errors and cancellation remain failures.
  Tests: `connectiontest/connection_network_record_lists_test.go:TestConnectionPortRecordsCatalogAbsencePrecedesListOptionsButSearchStillEvaluatesEmpty`, `connectiontest/connection_network_record_lists_test.go:TestConnectionPortRecordsMixedCatalogFailureAndCancellationDoNotClaimEmpty`.
- Invalid Connection/context fails before discovery; late-page failure preserves actual response error evidence and prevents a fabricated successful selection.
  Tests: `connectiontest/connection_network_record_lists_test.go:TestConnectionPortRecordsPreflightAndLatePageFailureRetainActualEvidence`.
- Source list() on literal arrays, objects, Unicode strings and empty iterables preserves JSON values without fake PortRecord or receipt; JSON object key order is explicitly Go lexical.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListLiteralIterablesDoNotFabricateResources`.
- Null/scalar noniterable and opaque-generator JMES function/syntax failures return before HTTP; result JSON is never fabricated.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListNonIterablesAndGeneratorErrorsDoNoHTTP`.
- Direct returned Source generator is eagerly collected into actual records with unknown/default fields, query/header pushdown, remote-only seed, actual page proof and computed per-record cloud location.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListReturnedGeneratorUsesActualRowsAndNoSeed`.
- Nested generator pointer remains opaque and lazy; serialization fails, explicit later Records iteration performs actual transport.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListNestedGeneratorRemainsOpaqueAndLazy`.
- Missing and CPython falsey JSON filters become empty kwargs; truthy nonobject filters fail before HTTP instead of being treated as cloud search expressions.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListFiltersFalseyAndTruthyNonMappings`.
- Raw filters/dedicated headers/typed callback snapshots own input state and nil callback maps; dedicated headers last-win without empty default overriding Source raw headers.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListHeadersInputAndCallbackSnapshots`.
- A later page physical fault preserves only observed actual Inventory; successful cloud Values/Ports/JSON are absent on failure.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListLateFaultRetainsOnlyActualInventory`.
- Clean absent network service returns nonnil [] before filter expansion or option callbacks; endpoint errors and context remain errors.
  Tests: `connectiontest/connection_port_values_test.go:TestConnectionListPortValuesAbsentServiceSkipsFilterExpansion`.
- Whole Connection shares cached network client and injects current location for actual records, preserves literal values without HTTP/fake records, and retains existing typed ListPorts compatibility.
  Tests: `connectiontest/connection_port_values_test.go:TestConnectionListPortValuesCachedClientLocationAndTypedListCompatibility`.

Python/Go differences:

- Concrete Go records expose owned declared Resource view and independent actual Wire/Envelope/Header/StatusCode. Full mutable Python Resource/cache/dirty-state/runtime subclass handling remains tracked shared SDK-R1/C1/S1, not counted as generated transport parity.
- ProxyListRecords reproduces Source generator-based JMES zero-HTTP values/null/type errors/opaque identity and nested generators. ListRecordValues is explicitly eager Go expression functionality, not claimed identical to Source generator input.
- ListRecords keeps a resource-only iterator and directs polymorphic truthy expressions to ProxyListRecords; falsey/nonstring Proxy controls preserve lazy records. Standard Go iter.Seq2 iteration semantics remain the owned SDK abstraction.
- List routes remain within captured service endpoint; continuation preserves selected origin/collection/filter and rejects unsafe route cycles. Go dictionary/RFC link compatibility and strict validation are explicit Source differences.
- Both list local filters and cloud dictionary selection opt into CPython JSON-loaded equality. Full JMESPath retains documented exact-decimal/lexical object traversal/strict grammar boundaries, and JSON-map object functions can differ from Python Port exact-typename function checks.
- Connection models clean service-catalog absence and explicit endpoint overrides; the broad Python has_service configuration-disabled service policy remains a shared SDK-C1 contract. Typed Connection.ListPorts intentionally retains record-only results; the named list_ports row remains unresolved for polymorphic iterable conversion.
- Concrete JSON/context/captured-client API; arbitrary Python Resource/session/class/callable substitutions are not accepted. Shared cloud configuration disabled-service semantics and mutable Resource R1 contracts remain separate tracked scope.
- Public Gophercloud testhelper/nativefixture reuse and actual response errors preserve physical read/close causes; opaque or literal local JSON never receives invented resource/HTTP evidence.
- Captured route/header/version protection, Go error returns and retained observed inventory differ intentionally from dynamic Python exception-only return conventions.
- Python literal object list() uses insertion order; Go JMES maps use lexical key order. Existing strict JMES syntax/exact numeric evaluator/custom Resource function type differences remain documented.
- Existing typed Connection.ListPorts remains resource-only compatibility API; the complete Source polymorphic list path is explicitly Connection.ListPortValues/API.ListSourceValues.
- Computed current cloud location is injected by Connection and optional for direct leaf; shared Source service configuration disable controls are not inferred from absent endpoint tests.

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
    parser.add_argument("--mode", choices=("values", "subnet", "bulk"), default="values")
    parser.add_argument("--filters", default="{}")
    parser.add_argument("--subnet", default="")
    parser.add_argument("--data", default="[]")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    if args.mode == "values":
        values = conn.list_ports(filters=json.loads(args.filters))
        for value in values:
            print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value, ensure_ascii=False, indent=2))
    elif args.mode == "subnet":
        if not args.subnet:
            parser.error("--subnet required")
        for value in conn.network.get_subnet_ports(args.subnet):
            print(json.dumps(value.to_dict(), ensure_ascii=False, indent=2))
    else:
        for value in conn.network.create_ports(json.loads(args.data)):
            print(json.dumps(value.to_dict(), ensure_ascii=False, indent=2))


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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
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
	mode := flag.String("mode", "values", "values, subnet or bulk")
	filters := flag.String("filters", "{}", "Source cloud filters JSON")
	subnetID := flag.String("subnet", "", "literal subnet ID")
	data := flag.String("data", "[]", "nonempty JSON array of port attribute objects")
	flag.Parse()
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	if *mode == "values" {
		result, err := conn.ListPortValues(ctx, ports.WithSourceCloudListFilters(json.RawMessage(*filters)))
		if err != nil {
			log.Fatal(err)
		}
		if result.JSON == nil {
			log.Fatal("opaque result; consume RecordListGenerator.Records explicitly")
		}
		printJSON(result.JSON)
		return
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	switch *mode {
	case "subnet":
		if *subnetID == "" {
			log.Fatal("--subnet required")
		}
		result, err := network.Ports.GetSubnetPorts(ctx, ports.SubnetPortID(*subnetID))
		if err != nil {
			log.Fatal(err)
		}
		for _, record := range result.Ports {
			printJSON(record.Resource.Body)
		}
	case "bulk":
		var rows []json.RawMessage
		if err := json.Unmarshal([]byte(*data), &rows); err != nil {
			log.Fatal(err)
		}
		inputs := make([]ports.BulkPortInput, len(rows))
		for i, row := range rows {
			inputs[i] = ports.BulkPortJSON(row)
		}
		result, err := network.Ports.CreatePorts(ctx, inputs)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("POST completed: %d\n", result.StatusCode)
		for record, err := range result.Records {
			if err != nil {
				log.Fatal(err)
			}
			printJSON(record.Resource.Body)
		}
	default:
		log.Fatal("--mode must be values, subnet or bulk")
	}
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/tree/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe)
- [Python List Records](python-list-records.md)
- [Source cloud `list_ports`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L202)
- [Source loop](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L4047)
- [Source create_ports](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3894)
- [V2](https://docs.openstack.org/api-ref/network/v2/#bulk-create-ports)
- [Neutron 27 Pecan BodyValidation](https://github.com/openstack/neutron/blob/27.0.0/neutron/pecan_wsgi/hooks/body_validation.py)
- [Policy hooks](https://github.com/openstack/neutron/blob/27.0.0/neutron/pecan_wsgi/hooks/policy_enforcement.py)
- [Plugin](https://github.com/openstack/neutron/blob/27.0.0/neutron/plugins/ml2/plugin.py)
- [Port](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py)
