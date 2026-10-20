# Cloud Object Search Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_object_store/ObjectStoreCloudMixin/search_objects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.InContainer`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchObjects`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.SearchObjects`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectSearchName`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectSearchFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectSearchExpression`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectSearchLocation`

- SDK collects the full literal-container Objects.ListRecords sequence with format=json and account-independent actual page receipts before cloud name/id exact-or-fnmatch and ordered/nested mapping selection; filters never become server query/prefix and duplicate rows remain.
  Tests: `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchCollectsEveryPageBeforeIdentifierAndMapping`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchNameUsesFnmatchAndDeclaredAliases`.
- Source falsey filters and loaded numeric equality, mapping short-circuit missing fields and header descriptor/Body alias getters are applied to declared Resource views; URI container and plain metadata remain separate and cannot become invented cloud-filter fields.
  Tests: `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchFalseyAndLoadedMappingFilters`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchPreservesLazyMissingAndNestedMappingFailures`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchLoadedHeaderBoolViewKeepsRawWireAndNull`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchUsesDefaultNullAndKeepsURIAndMetadataOutsideCloudView`.
- Full JMESPath returns null/scalar/object/list as independent Value with Expression=true and no invented Records correlation; malformed expressions on empty name selection preserve errors.
  Tests: `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchExpressionsReturnArbitraryJSONWithoutRowAssociation`.
- Direct Source Object location remains null while Connection automatically supplies configured CurrentLocation before selection; explicit overrides and whole DTO reset own all pointers/bytes and cannot accept forged remote location.
  Tests: `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchLocationIsOwnedAndRemoteWireCannotReplaceIt`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchUsesDefaultNullAndKeepsURIAndMetadataOutsideCloudView`, `connection_swift_object_search_test.go:TestConnectionSwiftObjectSearchSharesFacadeAndAutomaticLocationBeforeSelection`, `connection_swift_object_search_test.go:TestConnectionSwiftObjectSearchPerCallLocationRemainsOwnedAndCallbackRunsOnce`.
- Late page or physical handling/source/cancellation failures retain partial AllRecords and actual LastPage without a successful Value/Records. Real HTTP Content-Length faults at 200/203/401/404/429/503 preserve native pointer401 ErrOriginal/ErrReauth and block fault hooks. Source is captured before callbacks and preflight errors perform no discovery or fabricated response.
  Tests: `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchMalformedLatePagePreventsSuccessfulPartialResult`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchSourceCapturePrecedesOptionsAndKeepsCancellationCause`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchOwnsRawFiltersAndHeaderNilReset`, `objectstorage/cloud_object_search_test.go:TestSwiftCloudObjectSearchPhysicalLatePageKeepsNative401AndStopsHooks`, `connection_swift_object_search_test.go:TestConnectionSwiftObjectSearchPreflightDoesNotDiscoverOrFabricateResponse`.

Python/Go differences:

- Go returns ObjectSearchResult.Value and correlated Records only for ordinary selection, plus independent AllRecords/LastPage receipts; expression output is arbitrary JSON and never fabricates HTTP records.
- Direct Source-style clients preserve location null; Connection injects its owned CurrentLocation snapshot. Explicit raw location bytes are validated without discovery, and forged remote location remains only in Wire.
- Go exposes typed name/container strings and raw JSON-domain filters, owns per-instance metadata instead of Source class-level shared mutable state, and omits JMESPath deprecation warnings.
- Existing object listing descriptor/query/pagination contracts and documented RFC Link repair, same-origin/container/filter ownership, page bound and stronger cycle checks remain in force. SDK-R1/C1/S1 mutable Resource/dirty/session/cache and authenticated real-cloud acceptance remain tracked separately.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
objects = conn.search_objects("assets", name="logs-*", filters={"content_length": 5})
print([obj.name for obj in objects])
print(conn.search_objects("assets", filters="[].name"))
print(conn.search_objects("assets", filters="length(@)"))
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
    matched, err := conn.SearchObjects(ctx, "assets",
        objectstorage.WithObjectSearchName("logs-*"),
        objectstorage.WithObjectSearchFiltersJSON(json.RawMessage(`{"content_length":5}`)))
    if err != nil { log.Fatal(err) }
    for _, row := range matched.Records {
        fmt.Printf("name=%s bytes=%s page_status=%d\n",
            row.Resource.Body["name"], row.Resource.Body["content_length"], row.StatusCode)
    }
    names, err := conn.SearchObjects(ctx, "assets", objectstorage.WithObjectSearchExpression("[].name"))
    if err != nil { log.Fatal(err) }
    total, err := conn.SearchObjects(ctx, "assets", objectstorage.WithObjectSearchExpression("length(@)"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("names=%s total=%s\n", names.Value, total.Value)
}
```

## References

- [Cloud Object Search Test](cloud_object_search_test.go)
- [Connection Swift Object Search Test](../connection_swift_object_search_test.go)
