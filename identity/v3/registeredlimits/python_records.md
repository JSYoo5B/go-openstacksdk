# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/registered_limits`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits.WithRecordListPaginated`

- All with resource.WithQuery forwards service_id, resource_name and marker to GET registered_limits as Python registered_limits(**query) does for mapped parameters and follows links.next.
  Tests: `identity/v3/registeredlimits/python_parity_test.go:TestPythonRegisteredLimitListQueryAndPaging`.
- Pinned Proxy list performs lazy GET on fixed Keystone v3 collection with no implicit limit/discovery; only declared service/region/resource-name QueryParameters reach server. Unknown kwargs discard before marshaler. Source declarations are all untyped including documented numeric default_limit; missing null, arbitrary JSON primitive and literal Wire/page/Header/status evidence are independent.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsLazyQueryAndUntypedView`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsSourceSuccessStatusAndHeaderNoDiscovery`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`.
- Source ignores root Keystone links:{next:url} dictionaries, honors root links key precedence over plural_links, follows short nonempty pages with explicit positive limit using last raw wire ID even when locally filtered, and stops at empty page. Default paginated true/limit omission/max_items zero, single-page and raw prefilter max-items limit hint retain Source behavior; unsafe next URL yields actual last page proof.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsDictionaryLinksIgnoredWithSourcePrecedence`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsFilteredMarkerAndRawMaxItems`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsIteratorKeepsYieldedRowsAndAllDiscards`.
- Only declared unmapped Body kwargs filter row views locally, never server query. Ordered recursive dictionary subset filters preserve falsey/missing/empty-dictionary/scalar errors and short circuit; scalar/nested comparisons use CPython-loaded JSON float and bool equality without changing raw evidence.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsLoadedLocalFilterAndShortCircuit`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsFilteredMarkerAndRawMaxItems`.
- Copied functional opts preserve query raw bytes, maps and pointers; Source headers/microversion/base_path formals and one-pass __conflicting_attrs restoration normalize through shared ABI. Unknown Source control-like names are discarded without encoding, bound arg collisions and restored duplicate paginated/base_path fail before HTTP. Explicit version controls identity header on all pages with no invented discovery.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsOptionsSnapshotFormalRestorationAndHeaders`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsGuardsOptionCancellationAndContextCauses`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsSourceSuccessStatusAndHeaderNoDiscovery`.
- ProxyListRecords evaluates truthy string JMESPath against an opaque unconsumed Source generator before HTTP: projection/index/type return null, constants return JSON, length/to_string/sort_by fail, and @/to_array/not_null preserve generator identity. Omitted/falsey/nonstring filters keep lazy generator. JSON remains absent for generators; typed ListRecords explicitly directs truthy strings to polymorphic method.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitProxyListOpaqueProjectionAndRetainedGenerators`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`.
- Standalone no-Connection location defaults null and server location stays only in Wire. Immutable explicit CloudLocation uses existing ForResource; unknown project_id cannot override configured current scope. Invalid raw location fails before HTTP; there is no auth/Connection metadata inference.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsLocationIsExplicitImmutableAndDeclared`.
- Singleton plural object accepted; missing/null/scalar envelopes and cls/connection/microversion/_synchronized row collisions retain actual accepted status/body/header evidence. AllRecords discards partial rows; iterators retain earlier yielded rows. Nil callbacks/context, selected source mutation and cancellation stop before subsequent option/HTTP; Close cancellation preserves caller cause and actual physical receipt.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsSingletonMalformedAndConstructorEvidence`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsIteratorKeepsYieldedRowsAndAllDiscards`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsGuardsOptionCancellationAndContextCauses`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsCloseCancellationKeepsReceiptAndCause`.
- Actual 200/203/401/404/429/503 Read/Close/mixed physical faults retain every cause plus accepted receipt/native rejection body/header/status, with one Close/HTTP and zero Retry/Reauth callbacks. Native401 pointer preserves ErrOriginal/ErrReauth. Clean401 renewal and503 retry preserve fixed query/Accept/condition/version and original mutable service configuration.
  Tests: `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks`, `identity/v3/registeredlimits/python_records_test.go:TestPythonRegisteredLimitListRecordsCleanReauthenticationAndRetryKeepPolicy`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Source returns mutable Resource instances and polymorphic generator results. Go preserves typed native List/All separately and exposes owned ListRecords/AllRecords + ProxyListRecords with complete Source JSON view and original evidence; truthy JMESPath uses the explicit polymorphic method. Go iter.Seq2 can be iterated again to repeat the captured operation; Python generator consumption is one-shot.
- A standalone native client has no Python Connection: default computed location is null, and WithRecordListLocation supplies immutable recorded facts as an explicit Go extension. No automatic cloud/project name inference, token lookup, auth scope change or SDK-R1/C1 closure is claimed.
- Go validates functional options and JSON transport values before HTTP, requires safe fixed-origin service-relative paths, finite representable identifiers and nonnegative integer pagination amounts. Unsafe URLs/constructor collisions/malformed JSON become explicit receipt-bearing Go errors rather than Python exception classes or odd negative/noninteger pagination behavior.
- This class has no maximum microversion/discovery profile. Go honors selected native default and typed per-call microversion/header options while guarding selected source and SDK-owned auth/route/protocol headers. Dynamic Connection/session/Resource dirty/cache/warning contracts remain tracked at whole SDK level.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack


def main():
    conn = openstack.connect()
    for row in conn.identity.registered_limits(
        service_id="compute-service-id",
        resource_name="cores",
        description="team quota",
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits"
)

func main() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	identity, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// service_id/resource_name use Keystone query parameters.
	// description is a declared Body filter, evaluated locally.
	rows, err := identity.RegisteredLimits.AllRecords(ctx,
		registeredlimits.WithRecordListQuery("service_id", "compute-service-id"),
		registeredlimits.WithRecordListQuery("resource_name", "cores"),
		registeredlimits.WithRecordListQuery("description", "team quota"),
		registeredlimits.WithRecordListMaxItems(100),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		encoded, err := json.MarshalIndent(row.Resource.Body, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", encoded)
	}
}
```

```go
result, err := identity.RegisteredLimits.ProxyListRecords(ctx,
    registeredlimits.WithRecordListJMESPath("[*].id"),
)
if err != nil { log.Fatal(err) }
fmt.Println(string(result.JSON))

result, err = identity.RegisteredLimits.ProxyListRecords(ctx)
if err != nil { log.Fatal(err) }
generator := result.Value.(*registeredlimits.RecordListGenerator)
for row, err := range generator.Records {
    if err != nil { log.Fatal(err) }
    fmt.Println(string(row.Resource.Body["id"]))
}
```

## References

- [Policy](https://docs.openstack.org/keystone/latest/configuration/policy)
