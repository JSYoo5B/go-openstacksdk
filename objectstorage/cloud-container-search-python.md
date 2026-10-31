# Cloud Container Search Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_object_store/ObjectStoreCloudMixin/search_containers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.SearchContainers`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerSearchName`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerSearchFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerSearchExpression`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithContainerSearchLocation`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchContainers`

- ListRecords sends GET on the account root with Accept application/json, adds prefix only when WithRecordListPrefix is given, and follows the X-Account-Container-Count marker continuation with the prefix kept until an empty page.
  Tests: `objectstorage/v1/containers/python_parity_test.go:TestPythonContainerCloudListContainersPrefixAndCountPaging`.
- SDK collects the complete account ListRecords sequence before applying Source name/id exact-or-fnmatch identification and cloud ordered/nested mapping filters. Filters never become server prefix/query controls; duplicate rows and Source aliases remain visible.
  Tests: `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchCollectsEveryPageBeforeIdentifierAndMapping`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchNameUsesFnmatchAndDeclaredAliases`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchFalseyAndLoadedMappingFilters`.
- JMESPath returns arbitrary JSON null/scalar/object/list without inventing HTTP row association, while falsey filters, loaded float truthiness/equality, short-circuit field errors and invalid expression on empty identification follow shared SelectLoaded. Header bool getters use Source-loaded truthiness from raw SeedHeaders without changing old ListRecords policy or actual Wire, preserving null and nonempty string behavior.
  Tests: `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchExpressionsReturnArbitraryJSONWithoutRowAssociation`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchPreservesLazyMissingAndNestedMappingFailures`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchLoadedHeaderBoolViewKeepsRawWireAndNull`.
- Source is captured before callbacks; recorded/configured location is owned before HTTP and remote Wire location cannot replace it. AllRecords, selected Records, Envelope and Value are independent; late paging and guarded cancellation/source failures leave receipts without successful partial selection. Actual HTTP Content-Length faults at the second 200/203/401/404/429/503 page preserve physical body/header/status, pointer ErrOriginal/ErrReauth and zero fault retry/backoff/reauth callbacks.
  Tests: `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchLocationIsOwnedAndRemoteWireCannotReplaceIt`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchMalformedLatePagePreventsSuccessfulPartialResult`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchSourceCapturePrecedesOptionsAndKeepsCancellationCause`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchOwnsRawFiltersAndHeaderNilReset`, `objectstorage/cloud_container_search_test.go:TestSwiftCloudContainerSearchPhysicalLatePageKeepsNative401AndStopsHooks`.
- Connection shares the generated Swift Cloud facade and cached service client, supplies configured location before local selection, retains explicit owned per-call location, and handles DTO reset and callbacks once. Invalid/canceled connection fails without discovery or invented response.
  Tests: `connectiontest/connection_swift_container_records_test.go:TestConnectionSwiftContainerSearchSharesFacadeAndAutomaticLocationBeforeSelection`, `connectiontest/connection_swift_container_records_test.go:TestConnectionSwiftContainerSearchPerCallLocationRemainsOwnedAndCallbackRunsOnce`, `connectiontest/connection_swift_container_records_test.go:TestConnectionSwiftContainerSearchPreflightDoesNotDiscoverOrFabricateResponse`.

Python/Go differences:

- Go returns ContainerSearchResult.Value plus independent correlated Records for ordinary selection and AllRecords/LastPage evidence; expressions have Expression=true and Records=nil for arbitrary JSON.
- Direct clients use recorded native token project scope; Connection supplies configured cloud/region/project names with the owned location option. Unknown facts stay null and no authentication discovery is made by the helper.
- Go exposes typed string names and raw JSON-domain filters and omits Python deprecation warnings. Mutable Resource/session/dirty/cache SDK-R1/C1/S1 and real authenticated cloud acceptance remain separately tracked.
- Go retains existing ListRecords origin/path/filter ownership, page bound and stronger cycle detection; actual pointer401 LastPage receipt is copied from native ErrOriginal without replay.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
print([c.name for c in conn.search_containers(name="logs-*", filters={"count": 1})])
print(conn.search_containers(filters="[].name"))
print(conn.search_containers(filters="length(@)"))
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    matched, err := conn.SearchContainers(ctx,
        objectstorage.WithContainerSearchName("logs-*"),
        objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(`{"count":1}`)))
    if err != nil { log.Fatal(err) }
    for _, row := range matched.Records {
        fmt.Printf("name=%s page_status=%d\n", row.Resource.Body["name"], row.StatusCode)
    }
    names, err := conn.SearchContainers(ctx, objectstorage.WithContainerSearchExpression("[].name"))
    if err != nil { log.Fatal(err) }
    total, err := conn.SearchContainers(ctx, objectstorage.WithContainerSearchExpression("length(@)"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("names=%s total=%s\n", names.Value, total.Value)
}
```

## References

- [Cloud Container Search Test](cloud_container_search_test.go)
